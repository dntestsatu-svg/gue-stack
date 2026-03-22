package mysql

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

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

func TestTokoRepository_GetAccessibleByID(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	repo := NewTokoRepository(db)
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "name", "token", "charge", "callback_url", "created_at", "updated_at"}).
		AddRow(3, "Toko Scope", "token-scope", 3, "https://example.com/callback", now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`
WITH RECURSIVE actor_user AS (
  SELECT id, role, created_by
  FROM users
  WHERE id = ?
),
hierarchy AS (
  SELECT id
  FROM users
  WHERE id = ?
  UNION ALL
  SELECT u.id
  FROM users u
  INNER JOIN hierarchy h ON u.created_by = h.id
),
scoped_users AS (
  SELECT id
  FROM hierarchy
  UNION
  SELECT au.created_by
  FROM actor_user au
  WHERE au.role = 'user' AND au.created_by IS NOT NULL
),
accessible_tokos AS (
  SELECT DISTINCT tu.toko_id
  FROM toko_users tu
  CROSS JOIN actor_user au
  LEFT JOIN scoped_users su ON su.id = tu.user_id
  WHERE au.role = 'dev' OR su.id IS NOT NULL
)
SELECT t.id, t.name, t.token, t.charge, t.callback_url, t.created_at, t.updated_at
FROM tokos t
INNER JOIN accessible_tokos at ON at.toko_id = t.id
WHERE t.id = ?
LIMIT 1`)).
		WithArgs(uint64(10), uint64(10), uint64(3)).
		WillReturnRows(rows)

	item, err := repo.GetAccessibleByID(context.Background(), 10, model.UserRoleAdmin, 3)
	require.NoError(t, err)
	require.Equal(t, uint64(3), item.ID)
	require.Equal(t, "Toko Scope", item.Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTokoRepository_UpdateProfile(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	repo := NewTokoRepository(db)
	callbackURL := "https://example.com/updated"

	mock.ExpectExec(regexp.QuoteMeta("UPDATE tokos SET name = ?, callback_url = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?")).
		WithArgs("Toko Updated", callbackURL, uint64(3)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateProfile(context.Background(), 3, "Toko Updated", &callbackURL)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTokoRepository_UpdateToken(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	repo := NewTokoRepository(db)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE tokos SET token = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?")).
		WithArgs("rotated-token", uint64(3)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateToken(context.Background(), 3, "rotated-token")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
