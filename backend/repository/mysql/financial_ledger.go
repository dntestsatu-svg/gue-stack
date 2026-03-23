package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/example/gue/backend/model"
)

func insertFinancialLedgerEntryTx(
	ctx context.Context,
	tx *sql.Tx,
	tokoID *uint64,
	transactionID *uint64,
	entryType model.FinancialLedgerEntryType,
	amount uint64,
	reference *string,
) error {
	if tx == nil || amount == 0 {
		return nil
	}

	query := `INSERT INTO financial_ledger_entries (toko_id, transaction_id, entry_type, amount, reference)
VALUES (?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(
		ctx,
		query,
		nullableUint64(tokoID),
		nullableUint64(transactionID),
		entryType,
		amount,
		nullableString(reference),
	); err != nil {
		return fmt.Errorf("insert financial ledger entry %s: %w", entryType, err)
	}
	return nil
}
