package mysql

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/repository"
	"github.com/stretchr/testify/require"
)

func TestTokoRepository_Create_AlsoCreatesInitialBalance(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	repo := NewTokoRepository(db)
	toko := &model.Toko{
		Name:   "Toko A",
		Token:  "token-a",
		Charge: 3,
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO tokos (name, token, charge, callback_url) VALUES (?, ?, ?, ?)`)).
		WithArgs("Toko A", "token-a", 3, nil).
		WillReturnResult(sqlmock.NewResult(10, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO balances (toko_id, pending, available) VALUES (?, ?, ?)`)).
		WithArgs(int64(10), 0, 0).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.Create(context.Background(), toko)
	require.NoError(t, err)
	require.Equal(t, uint64(10), toko.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTokoRepository_CreateForUserWithQuota_AlsoCreatesInitialBalance(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	repo := NewTokoRepository(db)
	toko := &model.Toko{
		Name:   "Toko Quota",
		Token:  "token-quota",
		Charge: 3,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM users WHERE id = ? FOR UPDATE`)).
		WithArgs(uint64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM toko_users WHERE user_id = ?`)).
		WithArgs(uint64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO tokos (name, token, charge, callback_url) VALUES (?, ?, ?, ?)`)).
		WithArgs("Toko Quota", "token-quota", 3, nil).
		WillReturnResult(sqlmock.NewResult(22, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO toko_users (user_id, toko_id) VALUES (?, ?)`)).
		WithArgs(uint64(7), int64(22)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO balances (toko_id, pending, available) VALUES (?, ?, ?)`)).
		WithArgs(int64(22), 0, 0).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.CreateForUserWithQuota(context.Background(), 7, toko, 3)
	require.NoError(t, err)
	require.Equal(t, uint64(22), toko.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTokoRepository_CreateForUserWithQuota_QuotaExceededRollsBack(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	repo := NewTokoRepository(db)
	toko := &model.Toko{
		Name:   "Toko Blocked",
		Token:  "token-blocked",
		Charge: 3,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM users WHERE id = ? FOR UPDATE`)).
		WithArgs(uint64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM toko_users WHERE user_id = ?`)).
		WithArgs(uint64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectRollback()

	err := repo.CreateForUserWithQuota(context.Background(), 9, toko, 3)
	require.Error(t, err)
	require.ErrorIs(t, err, repository.ErrQuotaExceeded)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTokoRepository_Create_RollsBackWhenInitialBalanceFails(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	repo := NewTokoRepository(db)
	toko := &model.Toko{
		Name:   "Toko A",
		Token:  "token-a",
		Charge: 3,
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO tokos (name, token, charge, callback_url) VALUES (?, ?, ?, ?)`)).
		WithArgs("Toko A", "token-a", 3, nil).
		WillReturnResult(sqlmock.NewResult(10, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO balances (toko_id, pending, available) VALUES (?, ?, ?)`)).
		WithArgs(int64(10), 0, 0).
		WillReturnError(errors.New("write balance failed"))
	mock.ExpectRollback()

	err := repo.Create(context.Background(), toko)
	require.Error(t, err)
	require.Contains(t, err.Error(), "create initial toko balance in transaction")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTokoRepository_CreateForUserWithQuota_RollsBackWhenInitialBalanceFails(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	repo := NewTokoRepository(db)
	toko := &model.Toko{
		Name:   "Toko Quota",
		Token:  "token-quota",
		Charge: 3,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM users WHERE id = ? FOR UPDATE`)).
		WithArgs(uint64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM toko_users WHERE user_id = ?`)).
		WithArgs(uint64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO tokos (name, token, charge, callback_url) VALUES (?, ?, ?, ?)`)).
		WithArgs("Toko Quota", "token-quota", 3, nil).
		WillReturnResult(sqlmock.NewResult(22, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO toko_users (user_id, toko_id) VALUES (?, ?)`)).
		WithArgs(uint64(7), int64(22)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO balances (toko_id, pending, available) VALUES (?, ?, ?)`)).
		WithArgs(int64(22), 0, 0).
		WillReturnError(errors.New("write balance failed"))
	mock.ExpectRollback()

	err := repo.CreateForUserWithQuota(context.Background(), 7, toko, 3)
	require.Error(t, err)
	require.Contains(t, err.Error(), "create initial toko balance in transaction")
	require.NoError(t, mock.ExpectationsWereMet())
}
