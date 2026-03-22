package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/repository"
)

type PaymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) GetByID(ctx context.Context, id uint64) (*model.Payment, error) {
	query := `SELECT id, bank_code, bank_name, bank_swift_code, created_at, updated_at FROM payments WHERE id = ? LIMIT 1`
	var payment model.Payment
	if err := scanPaymentRow(r.db.QueryRowContext(ctx, query, id), &payment); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("query payment by id: %w", err)
	}
	return &payment, nil
}

func (r *PaymentRepository) SearchOptions(ctx context.Context, filter repository.PaymentOptionFilter) ([]model.Payment, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 50 {
		filter.Limit = 50
	}

	query := `SELECT id, bank_code, bank_name, bank_swift_code, created_at, updated_at FROM payments WHERE 1 = 1`
	args := make([]any, 0, 3)
	if search := strings.ToLower(strings.TrimSpace(filter.SearchTerm)); search != "" {
		pattern := "%" + search + "%"
		query += ` AND (LOWER(bank_name) LIKE ? OR LOWER(bank_code) LIKE ?)`
		args = append(args, pattern, pattern)
	}
	query += ` ORDER BY bank_name ASC, id ASC LIMIT ?`
	args = append(args, filter.Limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query payment options: %w", err)
	}
	defer rows.Close()

	result := make([]model.Payment, 0, filter.Limit)
	for rows.Next() {
		var payment model.Payment
		if err := scanPaymentRow(rows, &payment); err != nil {
			return nil, fmt.Errorf("scan payment option row: %w", err)
		}
		result = append(result, payment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate payment option rows: %w", err)
	}
	return result, nil
}

func scanPaymentRow(scanner interface {
	Scan(dest ...any) error
}, payment *model.Payment) error {
	var swiftCode sql.NullString
	if err := scanner.Scan(
		&payment.ID,
		&payment.BankCode,
		&payment.BankName,
		&swiftCode,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	); err != nil {
		return err
	}
	if swiftCode.Valid {
		payment.BankSwiftCode = &swiftCode.String
	} else {
		payment.BankSwiftCode = nil
	}
	return nil
}

var _ repository.PaymentRepository = (*PaymentRepository)(nil)
