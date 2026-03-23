package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/repository"
	"github.com/shopspring/decimal"
)

type BalanceRepository struct {
	db *sql.DB
}

func NewBalanceRepository(db *sql.DB) *BalanceRepository {
	return &BalanceRepository{db: db}
}

func (r *BalanceRepository) ListByUser(ctx context.Context, userID uint64, actorRole model.UserRole) ([]repository.TokoBalanceRecord, error) {
	query := `
SELECT
  t.id,
  t.name,
  COALESCE(b.pending, 0.00) AS pending_balance,
  COALESCE(b.available, 0.00) AS settle_balance,
  COALESCE(b.updated_at, t.updated_at) AS last_settlement_time
FROM tokos t
LEFT JOIN balances b ON b.toko_id = t.id
ORDER BY t.created_at DESC
`
	args := []any{}
	if !canViewAllTokos(actorRole) {
		query = tokoVisibilityCTE() + `
SELECT
  t.id,
  t.name,
  COALESCE(b.pending, 0.00) AS pending_balance,
  COALESCE(b.available, 0.00) AS settle_balance,
  COALESCE(b.updated_at, t.updated_at) AS last_settlement_time
FROM tokos t
INNER JOIN accessible_tokos at ON at.toko_id = t.id
LEFT JOIN balances b ON b.toko_id = t.id
ORDER BY t.created_at DESC
`
		args = append(args, userID, userID)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query toko balances by user: %w", err)
	}
	defer rows.Close()

	result := make([]repository.TokoBalanceRecord, 0)
	for rows.Next() {
		var item repository.TokoBalanceRecord
		var settlementRaw string
		var availableRaw string
		if err := rows.Scan(
			&item.TokoID,
			&item.TokoName,
			&settlementRaw,
			&availableRaw,
			&item.LastSettlementTime,
		); err != nil {
			return nil, fmt.Errorf("scan toko balance row: %w", err)
		}

		settlement, err := decimal.NewFromString(settlementRaw)
		if err != nil {
			return nil, fmt.Errorf("parse settlement balance: %w", err)
		}
		available, err := decimal.NewFromString(availableRaw)
		if err != nil {
			return nil, fmt.Errorf("parse available balance: %w", err)
		}
		item.PendingBalance = settlement.InexactFloat64()
		item.SettleBalance = available.InexactFloat64()
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate toko balance rows: %w", err)
	}
	return result, nil
}

func (r *BalanceRepository) GetByTokoID(ctx context.Context, tokoID uint64) (*repository.TokoBalanceRecord, error) {
	query := `
SELECT
  t.id,
  t.name,
  COALESCE(b.pending, 0.00) AS pending_balance,
  COALESCE(b.available, 0.00) AS settle_balance,
  COALESCE(b.updated_at, t.updated_at) AS last_settlement_time
FROM tokos t
LEFT JOIN balances b ON b.toko_id = t.id
WHERE t.id = ?
LIMIT 1
`

	item := &repository.TokoBalanceRecord{}
	var settlementRaw string
	var availableRaw string
	if err := r.db.QueryRowContext(ctx, query, tokoID).Scan(
		&item.TokoID,
		&item.TokoName,
		&settlementRaw,
		&availableRaw,
		&item.LastSettlementTime,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("query toko balance by toko id: %w", err)
	}

	settlement, err := decimal.NewFromString(settlementRaw)
	if err != nil {
		return nil, fmt.Errorf("parse settlement balance: %w", err)
	}
	available, err := decimal.NewFromString(availableRaw)
	if err != nil {
		return nil, fmt.Errorf("parse available balance: %w", err)
	}
	item.PendingBalance = settlement.InexactFloat64()
	item.SettleBalance = available.InexactFloat64()
	return item, nil
}

func (r *BalanceRepository) UpsertByTokoID(ctx context.Context, tokoID uint64, pendingBalance float64, settleBalance float64) error {
	query := `
INSERT INTO balances (toko_id, pending, available)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE
  pending = VALUES(pending),
  available = VALUES(available),
  updated_at = CURRENT_TIMESTAMP
`
	pending := decimal.NewFromFloat(pendingBalance).StringFixed(2)
	settle := decimal.NewFromFloat(settleBalance).StringFixed(2)
	if _, err := r.db.ExecContext(ctx, query, tokoID, pending, settle); err != nil {
		return fmt.Errorf("upsert toko balance: %w", err)
	}
	return nil
}

func (r *BalanceRepository) ApplySettlementByTokoID(ctx context.Context, tokoID uint64, amount float64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin apply settlement transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	query := `
UPDATE balances
SET pending = pending - ?, available = available + ?, updated_at = CURRENT_TIMESTAMP
WHERE toko_id = ? AND pending >= ?
`
	delta := decimal.NewFromFloat(amount).StringFixed(2)
	result, err := tx.ExecContext(ctx, query, delta, delta, tokoID, delta)
	if err != nil {
		return fmt.Errorf("apply settlement by toko id: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows for apply settlement: %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrInsufficientBalance
	}

	settlementAmount := decimal.NewFromFloat(amount).Round(0).BigInt()
	if !settlementAmount.IsUint64() {
		return fmt.Errorf("apply settlement amount exceeds uint64 range")
	}
	amountUint := settlementAmount.Uint64()
	if err := insertFinancialLedgerEntryTx(ctx, tx, &tokoID, nil, model.FinancialLedgerEntryManualSettlementPendingDeb, amountUint, nil); err != nil {
		return err
	}
	if err := insertFinancialLedgerEntryTx(ctx, tx, &tokoID, nil, model.FinancialLedgerEntryManualSettlementSettleCred, amountUint, nil); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit apply settlement transaction: %w", err)
	}
	committed = true
	return nil
}

func (r *BalanceRepository) IncreasePendingByTokoID(ctx context.Context, tokoID uint64, amount float64) error {
	query := `
UPDATE balances
SET pending = pending + ?, updated_at = CURRENT_TIMESTAMP
WHERE toko_id = ?
`
	delta := decimal.NewFromFloat(amount).StringFixed(2)
	result, err := r.db.ExecContext(ctx, query, delta, tokoID)
	if err != nil {
		return fmt.Errorf("increase pending balance: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows for increase pending: %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *BalanceRepository) DecreasePendingByTokoID(ctx context.Context, tokoID uint64, amount float64) error {
	query := `
UPDATE balances
SET pending = pending - ?, updated_at = CURRENT_TIMESTAMP
WHERE toko_id = ? AND pending >= ?
`
	delta := decimal.NewFromFloat(amount).StringFixed(2)
	result, err := r.db.ExecContext(ctx, query, delta, tokoID, delta)
	if err != nil {
		return fmt.Errorf("decrease pending balance: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows for decrease pending: %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrInsufficientBalance
	}
	return nil
}

func (r *BalanceRepository) DecreaseSettlementByTokoID(ctx context.Context, tokoID uint64, amount float64) error {
	query := `
UPDATE balances
SET available = available - ?, updated_at = CURRENT_TIMESTAMP
WHERE toko_id = ? AND available >= ?
`
	delta := decimal.NewFromFloat(amount).StringFixed(2)
	result, err := r.db.ExecContext(ctx, query, delta, tokoID, delta)
	if err != nil {
		return fmt.Errorf("decrease settlement balance: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows for decrease settlement: %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrInsufficientBalance
	}
	return nil
}

func (r *BalanceRepository) IncreaseSettlementByTokoID(ctx context.Context, tokoID uint64, amount float64) error {
	query := `
UPDATE balances
SET available = available + ?, updated_at = CURRENT_TIMESTAMP
WHERE toko_id = ?
`
	delta := decimal.NewFromFloat(amount).StringFixed(2)
	result, err := r.db.ExecContext(ctx, query, delta, tokoID)
	if err != nil {
		return fmt.Errorf("increase settlement balance: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows for increase settlement: %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

var _ repository.BalanceRepository = (*BalanceRepository)(nil)
