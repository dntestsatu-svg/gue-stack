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
  COALESCE(b.pending, 0.00) AS settlement_balance,
  COALESCE(b.available, 0.00) AS available_balance,
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
  COALESCE(b.pending, 0.00) AS settlement_balance,
  COALESCE(b.available, 0.00) AS available_balance,
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
		item.SettlementBalance = settlement.InexactFloat64()
		item.AvailableBalance = available.InexactFloat64()
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
  COALESCE(b.pending, 0.00) AS settlement_balance,
  COALESCE(b.available, 0.00) AS available_balance,
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
	item.SettlementBalance = settlement.InexactFloat64()
	item.AvailableBalance = available.InexactFloat64()
	return item, nil
}

func (r *BalanceRepository) UpsertByTokoID(ctx context.Context, tokoID uint64, settlementBalance float64, availableBalance float64) error {
	query := `
INSERT INTO balances (toko_id, pending, available)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE
  pending = VALUES(pending),
  available = VALUES(available),
  updated_at = CURRENT_TIMESTAMP
`
	settlement := decimal.NewFromFloat(settlementBalance).StringFixed(2)
	available := decimal.NewFromFloat(availableBalance).StringFixed(2)
	if _, err := r.db.ExecContext(ctx, query, tokoID, settlement, available); err != nil {
		return fmt.Errorf("upsert toko balance: %w", err)
	}
	return nil
}

var _ repository.BalanceRepository = (*BalanceRepository)(nil)
