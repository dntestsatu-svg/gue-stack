package mysql

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/example/gue/backend/model"
	"github.com/stretchr/testify/require"
)

func TestBalanceRepositoryApplySettlementByTokoIDWritesLedger(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	repo := NewBalanceRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE balances
SET pending = pending - ?, available = available + ?, updated_at = CURRENT_TIMESTAMP
WHERE toko_id = ? AND pending >= ?
`)).
		WithArgs("250000.00", "250000.00", uint64(12), "250000.00").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO financial_ledger_entries (toko_id, transaction_id, entry_type, amount, reference)
VALUES (?, ?, ?, ?, ?)`)).
		WithArgs(uint64(12), nil, model.FinancialLedgerEntryManualSettlementPendingDeb, uint64(250000), nil).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO financial_ledger_entries (toko_id, transaction_id, entry_type, amount, reference)
VALUES (?, ?, ?, ?, ?)`)).
		WithArgs(uint64(12), nil, model.FinancialLedgerEntryManualSettlementSettleCred, uint64(250000), nil).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()

	err := repo.ApplySettlementByTokoID(context.Background(), 12, 250000)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
