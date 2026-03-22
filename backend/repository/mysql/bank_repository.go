package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/repository"
)

type BankRepository struct {
	db *sql.DB
}

func NewBankRepository(db *sql.DB) *BankRepository {
	return &BankRepository{db: db}
}

func (r *BankRepository) ListByUser(ctx context.Context, userID uint64, filter repository.BankListFilter) ([]model.Bank, error) {
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Limit > 50 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	query := `
SELECT id, user_id, payment_id, bank_code, bank_name, account_name, account_number, created_at, updated_at
FROM banks
WHERE user_id = ?`
	args := []any{userID}

	if search := strings.ToLower(strings.TrimSpace(filter.SearchTerm)); search != "" {
		pattern := "%" + search + "%"
		query += ` AND (LOWER(bank_name) LIKE ? OR LOWER(account_name) LIKE ? OR LOWER(account_number) LIKE ?)`
		args = append(args, pattern, pattern, pattern)
	}

	query += ` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query list banks: %w", err)
	}
	defer rows.Close()

	result := make([]model.Bank, 0, filter.Limit)
	for rows.Next() {
		var bank model.Bank
		if err := scanBankRow(rows, &bank); err != nil {
			return nil, fmt.Errorf("scan bank row: %w", err)
		}
		result = append(result, bank)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bank rows: %w", err)
	}
	return result, nil
}

func (r *BankRepository) CountByUser(ctx context.Context, userID uint64, filter repository.BankListFilter) (uint64, error) {
	query := `SELECT COUNT(*) FROM banks WHERE user_id = ?`
	args := []any{userID}

	if search := strings.ToLower(strings.TrimSpace(filter.SearchTerm)); search != "" {
		pattern := "%" + search + "%"
		query += ` AND (LOWER(bank_name) LIKE ? OR LOWER(account_name) LIKE ? OR LOWER(account_number) LIKE ?)`
		args = append(args, pattern, pattern, pattern)
	}

	var total uint64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count banks by user: %w", err)
	}
	return total, nil
}

func (r *BankRepository) Create(ctx context.Context, bank *model.Bank) error {
	now := time.Now().UTC()
	bank.CreatedAt = now
	bank.UpdatedAt = now
	query := `
INSERT INTO banks (user_id, payment_id, bank_code, bank_name, account_name, account_number)
VALUES (?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query, bank.UserID, bank.PaymentID, bank.BankCode, bank.BankName, bank.AccountName, bank.AccountNumber)
	if err != nil {
		return fmt.Errorf("create bank: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get inserted bank id: %w", err)
	}
	bank.ID = uint64(id)
	return nil
}

func (r *BankRepository) DeleteByUser(ctx context.Context, userID uint64, bankID uint64) error {
	query := `DELETE FROM banks WHERE id = ? AND user_id = ?`
	result, err := r.db.ExecContext(ctx, query, bankID, userID)
	if err != nil {
		return fmt.Errorf("delete bank by user: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected for delete bank: %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func scanBankRow(scanner interface {
	Scan(dest ...any) error
}, bank *model.Bank) error {
	return scanner.Scan(
		&bank.ID,
		&bank.UserID,
		&bank.PaymentID,
		&bank.BankCode,
		&bank.BankName,
		&bank.AccountName,
		&bank.AccountNumber,
		&bank.CreatedAt,
		&bank.UpdatedAt,
	)
}

func isBankDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate")
}

var _ repository.BankRepository = (*BankRepository)(nil)
