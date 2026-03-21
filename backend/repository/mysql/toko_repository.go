package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/repository"
)

type TokoRepository struct {
	db *sql.DB
}

func NewTokoRepository(db *sql.DB) *TokoRepository {
	return &TokoRepository{db: db}
}

func (r *TokoRepository) Create(ctx context.Context, toko *model.Toko) error {
	query := `INSERT INTO tokos (name, token, charge, callback_url) VALUES (?, ?, ?, ?)`
	result, err := r.db.ExecContext(
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

func (r *TokoRepository) ListByUser(ctx context.Context, userID uint64) ([]model.Toko, error) {
	query := `
SELECT t.id, t.name, t.token, t.charge, t.callback_url, t.created_at, t.updated_at
FROM tokos t
INNER JOIN toko_users tu ON tu.toko_id = t.id
WHERE tu.user_id = ?
ORDER BY t.created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID)
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
