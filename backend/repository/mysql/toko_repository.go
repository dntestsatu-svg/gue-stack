package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/repository"
	"github.com/shopspring/decimal"
)

type TokoRepository struct {
	db *sql.DB
}

func NewTokoRepository(db *sql.DB) *TokoRepository {
	return &TokoRepository{db: db}
}

func (r *TokoRepository) Create(ctx context.Context, toko *model.Toko) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create toko transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	query := `INSERT INTO tokos (name, token, charge, callback_url) VALUES (?, ?, ?, ?)`
	result, err := tx.ExecContext(
		ctx,
		query,
		toko.Name,
		toko.Token,
		toko.Charge,
		nullableString(toko.CallbackURL),
	)
	if err != nil {
		return fmt.Errorf("create toko: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get inserted toko id: %w", err)
	}

	if err := insertInitialBalanceTx(ctx, tx, id); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create toko transaction: %w", err)
	}
	committed = true
	toko.ID = uint64(id)
	return nil
}

func (r *TokoRepository) CreateForUserWithQuota(ctx context.Context, userID uint64, toko *model.Toko, maxTokos int) error {
	if maxTokos <= 0 {
		maxTokos = 3
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create toko transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var lockedUserID uint64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE id = ? FOR UPDATE`, userID).Scan(&lockedUserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.ErrNotFound
		}
		return fmt.Errorf("lock user row for toko quota: %w", err)
	}

	var total int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM toko_users WHERE user_id = ?`, userID).Scan(&total); err != nil {
		return fmt.Errorf("count tokos by user in transaction: %w", err)
	}
	if total >= maxTokos {
		return repository.ErrQuotaExceeded
	}

	result, err := tx.ExecContext(
		ctx,
		`INSERT INTO tokos (name, token, charge, callback_url) VALUES (?, ?, ?, ?)`,
		toko.Name,
		toko.Token,
		toko.Charge,
		nullableString(toko.CallbackURL),
	)
	if err != nil {
		return fmt.Errorf("create toko in transaction: %w", err)
	}
	insertedID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get inserted toko id in transaction: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO toko_users (user_id, toko_id) VALUES (?, ?)`, userID, insertedID); err != nil {
		return fmt.Errorf("attach user to toko in transaction: %w", err)
	}

	if err := insertInitialBalanceTx(ctx, tx, insertedID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create toko transaction: %w", err)
	}
	committed = true
	toko.ID = uint64(insertedID)
	return nil
}

func (r *TokoRepository) AttachUser(ctx context.Context, userID, tokoID uint64) error {
	query := `INSERT INTO toko_users (user_id, toko_id) VALUES (?, ?)`
	if _, err := r.db.ExecContext(ctx, query, userID, tokoID); err != nil {
		return fmt.Errorf("attach user to toko: %w", err)
	}
	return nil
}

func (r *TokoRepository) CountByUser(ctx context.Context, userID uint64) (int, error) {
	query := `SELECT COUNT(*) FROM toko_users WHERE user_id = ?`
	var total int
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(&total); err != nil {
		return 0, fmt.Errorf("count tokos by user: %w", err)
	}
	return total, nil
}

func (r *TokoRepository) ListByUser(ctx context.Context, userID uint64, actorRole model.UserRole) ([]model.Toko, error) {
	query := `SELECT t.id, t.name, t.token, t.charge, t.callback_url, t.created_at, t.updated_at
FROM tokos t
ORDER BY t.created_at DESC`
	args := []any{}
	if !canViewAllTokos(actorRole) {
		query = tokoVisibilityCTE() + `
SELECT t.id, t.name, t.token, t.charge, t.callback_url, t.created_at, t.updated_at
FROM tokos t
INNER JOIN accessible_tokos at ON at.toko_id = t.id
ORDER BY t.created_at DESC`
		args = append(args, userID, userID)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tokos by user: %w", err)
	}
	defer rows.Close()

	result := make([]model.Toko, 0)
	for rows.Next() {
		var item model.Toko
		var callbackURL sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Token,
			&item.Charge,
			&callbackURL,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan toko row: %w", err)
		}
		if callbackURL.Valid {
			item.CallbackURL = &callbackURL.String
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate toko rows: %w", err)
	}
	return result, nil
}

func (r *TokoRepository) ListWorkspaceByUser(ctx context.Context, userID uint64, actorRole model.UserRole, filter repository.TokoWorkspaceFilter) ([]repository.TokoWorkspaceRecord, error) {
	query, args := buildTokoWorkspaceListQuery(userID, actorRole, filter)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list toko workspace by user: %w", err)
	}
	defer rows.Close()

	result := make([]repository.TokoWorkspaceRecord, 0, filter.Limit)
	for rows.Next() {
		var item repository.TokoWorkspaceRecord
		var callbackURL sql.NullString
		var settlementRaw string
		var availableRaw string
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Token,
			&item.Charge,
			&callbackURL,
			&settlementRaw,
			&availableRaw,
			&item.LastSettlementTime,
		); err != nil {
			return nil, fmt.Errorf("scan toko workspace row: %w", err)
		}
		if callbackURL.Valid {
			item.CallbackURL = &callbackURL.String
		}

		settlement, err := decimal.NewFromString(settlementRaw)
		if err != nil {
			return nil, fmt.Errorf("parse settlement balance: %w", err)
		}
		available, err := decimal.NewFromString(availableRaw)
		if err != nil {
			return nil, fmt.Errorf("parse available balance: %w", err)
		}
		item.SettlementBalance = settlement.InexactFloat64()
		item.AvailableBalance = available.InexactFloat64()
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate toko workspace rows: %w", err)
	}
	return result, nil
}

func (r *TokoRepository) SummarizeWorkspaceByUser(ctx context.Context, userID uint64, actorRole model.UserRole, filter repository.TokoWorkspaceFilter) (*repository.TokoWorkspaceSummary, error) {
	query, args := buildTokoWorkspaceSummaryQuery(userID, actorRole, filter)

	var summary repository.TokoWorkspaceSummary
	var settlementRaw string
	var availableRaw string
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&summary.TotalTokos,
		&settlementRaw,
		&availableRaw,
	); err != nil {
		return nil, fmt.Errorf("summarize toko workspace by user: %w", err)
	}

	settlement, err := decimal.NewFromString(settlementRaw)
	if err != nil {
		return nil, fmt.Errorf("parse total settlement balance: %w", err)
	}
	available, err := decimal.NewFromString(availableRaw)
	if err != nil {
		return nil, fmt.Errorf("parse total available balance: %w", err)
	}

	summary.TotalSettlementAmount = settlement.InexactFloat64()
	summary.TotalAvailableAmount = available.InexactFloat64()
	return &summary, nil
}

func (r *TokoRepository) GetByID(ctx context.Context, id uint64) (*model.Toko, error) {
	query := `SELECT id, name, token, charge, callback_url, created_at, updated_at FROM tokos WHERE id = ? LIMIT 1`
	return r.getOne(ctx, query, id)
}

func (r *TokoRepository) GetByToken(ctx context.Context, token string) (*model.Toko, error) {
	query := `SELECT id, name, token, charge, callback_url, created_at, updated_at FROM tokos WHERE token = ? LIMIT 1`
	return r.getOne(ctx, query, token)
}

func (r *TokoRepository) getOne(ctx context.Context, query string, arg any) (*model.Toko, error) {
	toko := &model.Toko{}
	var callbackURL sql.NullString
	if err := r.db.QueryRowContext(ctx, query, arg).Scan(
		&toko.ID,
		&toko.Name,
		&toko.Token,
		&toko.Charge,
		&callbackURL,
		&toko.CreatedAt,
		&toko.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("query toko: %w", err)
	}
	if callbackURL.Valid {
		toko.CallbackURL = &callbackURL.String
	}
	return toko, nil
}

func insertInitialBalanceTx(ctx context.Context, tx *sql.Tx, tokoID int64) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO balances (toko_id, pending, available) VALUES (?, ?, ?)`, tokoID, 0, 0); err != nil {
		return fmt.Errorf("create initial toko balance in transaction: %w", err)
	}
	return nil
}

func buildTokoWorkspaceListQuery(userID uint64, actorRole model.UserRole, filter repository.TokoWorkspaceFilter) (string, []any) {
	base, args := buildTokoWorkspaceBaseQuery(userID, actorRole, filter)
	query := base + `
SELECT
  t.id,
  t.name,
  t.token,
  t.charge,
  t.callback_url,
  COALESCE(b.pending, 0.00) AS settlement_balance,
  COALESCE(b.available, 0.00) AS available_balance,
  COALESCE(b.updated_at, t.updated_at) AS last_settlement_time
FROM tokos t
LEFT JOIN balances b ON b.toko_id = t.id
` + buildTokoWorkspaceJoinAndWhere(actorRole, filter) + `
ORDER BY t.created_at DESC, t.id DESC
LIMIT ? OFFSET ?`

	args = append(args, filter.Limit, filter.Offset)
	return query, args
}

func buildTokoWorkspaceSummaryQuery(userID uint64, actorRole model.UserRole, filter repository.TokoWorkspaceFilter) (string, []any) {
	base, args := buildTokoWorkspaceBaseQuery(userID, actorRole, filter)
	query := base + `
SELECT
  COUNT(*) AS total_tokos,
  COALESCE(CAST(SUM(COALESCE(b.pending, 0.00)) AS CHAR), '0.00') AS total_settlement_balance,
  COALESCE(CAST(SUM(COALESCE(b.available, 0.00)) AS CHAR), '0.00') AS total_available_balance
FROM tokos t
LEFT JOIN balances b ON b.toko_id = t.id
` + buildTokoWorkspaceJoinAndWhere(actorRole, filter)

	return query, args
}

func buildTokoWorkspaceBaseQuery(userID uint64, actorRole model.UserRole, filter repository.TokoWorkspaceFilter) (string, []any) {
	if canViewAllTokos(actorRole) {
		return "", buildTokoWorkspaceFilterArgs(filter)
	}

	return tokoVisibilityCTE(), append([]any{userID, userID}, buildTokoWorkspaceFilterArgs(filter)...)
}

func buildTokoWorkspaceJoinAndWhere(actorRole model.UserRole, filter repository.TokoWorkspaceFilter) string {
	parts := make([]string, 0, 2)
	if !canViewAllTokos(actorRole) {
		parts = append(parts, "INNER JOIN accessible_tokos at ON at.toko_id = t.id")
	}

	clauses := []string{"1 = 1"}
	if search := strings.TrimSpace(filter.SearchTerm); search != "" {
		clauses = append(clauses, "(LOWER(t.name) LIKE ? OR LOWER(t.token) LIKE ? OR LOWER(COALESCE(t.callback_url, '')) LIKE ?)")
	}

	parts = append(parts, "WHERE "+strings.Join(clauses, " AND "))
	return "\n" + strings.Join(parts, "\n")
}

func buildTokoWorkspaceFilterArgs(filter repository.TokoWorkspaceFilter) []any {
	search := strings.ToLower(strings.TrimSpace(filter.SearchTerm))
	if search == "" {
		return nil
	}

	pattern := "%" + search + "%"
	return []any{pattern, pattern, pattern}
}
