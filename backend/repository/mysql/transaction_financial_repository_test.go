package mysql

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/example/gue/backend/model"
	"github.com/stretchr/testify/require"
)

func TestTransactionRepositoryFinalizeDepositSuccessByIDWritesLedgerAndPendingBalance(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	repo := NewTransactionRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE transactions
SET status = ?, platform_fee = ?, netto = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND status = ? AND type = ?`)).
		WithArgs(model.TransactionStatusSuccess, uint64(3000), uint64(97000), uint64(9), model.TransactionStatusPending, model.TransactionTypeDeposit).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO financial_ledger_entries (toko_id, transaction_id, entry_type, amount, reference)
VALUES (?, ?, ?, ?, ?)`)).
		WithArgs(uint64(5), uint64(9), model.FinancialLedgerEntryDepositPendingCredit, uint64(97000), nil).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO financial_ledger_entries (toko_id, transaction_id, entry_type, amount, reference)
VALUES (?, ?, ?, ?, ?)`)).
		WithArgs(uint64(5), uint64(9), model.FinancialLedgerEntryProjectPlatformFeeCredit, uint64(3000), nil).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE balances
SET pending = pending + ?, updated_at = CURRENT_TIMESTAMP
WHERE toko_id = ?`)).
		WithArgs("97000.00", uint64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	updated, err := repo.FinalizeDepositSuccessByID(context.Background(), 9, 5, 3000, 97000)
	require.NoError(t, err)
	require.True(t, updated)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionRepositoryCreatePendingWithdrawAndReserveSettlement(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	repo := NewTransactionRepository(db)
	reference := "partner-ref-1"
	fee := uint64(1500)
	trx := &model.Transaction{
		TokoID:        5,
		Type:          model.TransactionTypeWithdraw,
		Status:        model.TransactionStatusPending,
		Reference:     &reference,
		Amount:        100000,
		FeeWithdrawal: &fee,
		PlatformFee:   0,
		Netto:         100000,
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE balances
SET available = available - ?, updated_at = CURRENT_TIMESTAMP
WHERE toko_id = ? AND available >= ?`)).
		WithArgs("101500.00", uint64(5), "101500.00").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO transactions (toko_id, player, code, `+"`type`, status, barcode, reference, amount, fee_withdrawal, platform_fee, netto"+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)).
		WithArgs(uint64(5), nil, nil, model.TransactionTypeWithdraw, model.TransactionStatusPending, nil, reference, uint64(100000), uint64(1500), uint64(0), uint64(100000)).
		WillReturnResult(sqlmock.NewResult(77, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO financial_ledger_entries (toko_id, transaction_id, entry_type, amount, reference)
VALUES (?, ?, ?, ?, ?)`)).
		WithArgs(uint64(5), uint64(77), model.FinancialLedgerEntryWithdrawSettleDebit, uint64(100000), reference).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO financial_ledger_entries (toko_id, transaction_id, entry_type, amount, reference)
VALUES (?, ?, ?, ?, ?)`)).
		WithArgs(uint64(5), uint64(77), model.FinancialLedgerEntryWithdrawFeeDebit, uint64(1500), reference).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()

	err := repo.CreatePendingWithdrawAndReserveSettlement(context.Background(), trx)
	require.NoError(t, err)
	require.Equal(t, uint64(77), trx.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionRepositoryFinalizeWithdrawIfPendingFailedRefundsSettlement(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	repo := NewTransactionRepository(db)
	reference := "partner-ref-2"

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT toko_id, amount, fee_withdrawal, reference, status, type
FROM transactions
WHERE id = ?
FOR UPDATE`)).
		WithArgs(uint64(88)).
		WillReturnRows(sqlmock.NewRows([]string{"toko_id", "amount", "fee_withdrawal", "reference", "status", "type"}).
			AddRow(uint64(5), uint64(100000), int64(1500), reference, model.TransactionStatusPending, model.TransactionTypeWithdraw))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE transactions
SET status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND status = ?`)).
		WithArgs(model.TransactionStatusFailed, uint64(88), model.TransactionStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE balances
SET available = available + ?, updated_at = CURRENT_TIMESTAMP
WHERE toko_id = ?`)).
		WithArgs("101500.00", uint64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO financial_ledger_entries (toko_id, transaction_id, entry_type, amount, reference)
VALUES (?, ?, ?, ?, ?)`)).
		WithArgs(uint64(5), uint64(88), model.FinancialLedgerEntryWithdrawSettleRefund, uint64(100000), reference).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO financial_ledger_entries (toko_id, transaction_id, entry_type, amount, reference)
VALUES (?, ?, ?, ?, ?)`)).
		WithArgs(uint64(5), uint64(88), model.FinancialLedgerEntryWithdrawFeeRefund, uint64(1500), reference).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()

	updated, err := repo.FinalizeWithdrawIfPending(context.Background(), 88, model.TransactionStatusFailed)
	require.NoError(t, err)
	require.True(t, updated)
	require.NoError(t, mock.ExpectationsWereMet())
}
