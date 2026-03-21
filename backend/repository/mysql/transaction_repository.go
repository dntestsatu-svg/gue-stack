package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/repository"
)

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(ctx context.Context, trx *model.Transaction) error {
	query := "INSERT INTO transactions (" +
		"toko_id, player, code, `type`, status, barcode, reference, amount, fee_withdrawal, netto" +
		") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"

	result, err := r.db.ExecContext(
		ctx,
		query,
		trx.TokoID,
		nullableString(trx.Player),
		nullableString(trx.Code),
		trx.Type,
		trx.Status,
		nullableString(trx.Barcode),
		nullableString(trx.Reference),
		trx.Amount,
		nullableUint64(trx.FeeWithdrawal),
		trx.Netto,
	)
	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get transaction id: %w", err)
	}
	trx.ID = uint64(id)
	return nil
}

func (r *TransactionRepository) GetByReference(ctx context.Context, reference string) (*model.Transaction, error) {
	query := "SELECT id, toko_id, player, code, `type`, status, barcode, reference, amount, fee_withdrawal, netto, created_at, updated_at " +
		"FROM transactions WHERE reference = ? LIMIT 1"

	trx := &model.Transaction{}
	var player, code, barcode, ref sql.NullString
	var fee sql.NullInt64
	if err := r.db.QueryRowContext(ctx, query, reference).Scan(
		&trx.ID,
		&trx.TokoID,
		&player,
		&code,
		&trx.Type,
		&trx.Status,
		&barcode,
		&ref,
		&trx.Amount,
		&fee,
		&trx.Netto,
		&trx.CreatedAt,
		&trx.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("query transaction by reference: %w", err)
	}

	if player.Valid {
		trx.Player = &player.String
	}
	if code.Valid {
		trx.Code = &code.String
	}
	if barcode.Valid {
		trx.Barcode = &barcode.String
	}
	if ref.Valid {
		trx.Reference = &ref.String
	}
	if fee.Valid {
		v := uint64(fee.Int64)
		trx.FeeWithdrawal = &v
	}

	return trx, nil
}

func (r *TransactionRepository) UpdateStatusByReference(ctx context.Context, reference string, status model.TransactionStatus) error {
	query := `UPDATE transactions SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE reference = ?`
	result, err := r.db.ExecContext(ctx, query, status, reference)
	if err != nil {
		return fmt.Errorf("update transaction status by reference: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *TransactionRepository) UpdateStatusByReferenceAndToko(ctx context.Context, reference string, tokoID uint64, status model.TransactionStatus) error {
	query := `UPDATE transactions SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE reference = ? AND toko_id = ?`
	result, err := r.db.ExecContext(ctx, query, status, reference, tokoID)
	if err != nil {
		return fmt.Errorf("update transaction status by reference and toko: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableUint64(value *uint64) any {
	if value == nil {
		return nil
	}
	return *value
}
