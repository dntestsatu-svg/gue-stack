package mysql

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/repository"
	"github.com/stretchr/testify/require"
)

func setupDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return db, mock
}

func TestUserRepository_GetByEmailFound(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "name", "email", "password_hash", "role", "is_active", "created_by", "created_at", "updated_at"}).
		AddRow(1, "Jane", "jane@example.com", "hash", model.UserRoleUser, true, nil, now, now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, email, password_hash, role, is_active, created_by, created_at, updated_at FROM users WHERE email = ? LIMIT 1")).
		WithArgs("jane@example.com").
		WillReturnRows(rows)

	user, err := repo.GetByEmail(context.Background(), "jane@example.com")
	require.NoError(t, err)
	require.Equal(t, uint64(1), user.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByEmailNotFound(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, email, password_hash, role, is_active, created_by, created_at, updated_at FROM users WHERE email = ? LIMIT 1")).
		WithArgs("none@example.com").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetByEmail(context.Background(), "none@example.com")
	require.Error(t, err)
	require.ErrorIs(t, err, repository.ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Create(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	creatorID := uint64(99)
	user := &model.User{Name: "John", Email: "john@example.com", PasswordHash: "hash", Role: model.UserRoleAdmin, IsActive: true, CreatedBy: &creatorID}

	expect := mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (name, email, password_hash, role, is_active, created_by) VALUES (?, ?, ?, ?, ?, ?)"))
	expect.WithArgs("John", "john@example.com", "hash", model.UserRoleAdmin, true, creatorID).WillReturnResult(sqlmock.NewResult(5, 1))

	err := repo.Create(context.Background(), user)
	require.NoError(t, err)
	require.Equal(t, uint64(5), user.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_UpdateRole(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET role = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?")).
		WithArgs(model.UserRoleSuperAdmin, uint64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateRole(context.Background(), 10, model.UserRoleSuperAdmin)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_ListByScope(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "name", "email", "password_hash", "role", "is_active", "created_by", "created_at", "updated_at"}).
		AddRow(1, "Dev", "dev@example.com", "hash", model.UserRoleDev, true, nil, now, now).
		AddRow(2, "Admin", "admin@example.com", "hash", model.UserRoleAdmin, true, 1, now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`
WITH RECURSIVE hierarchy AS (
  SELECT id
  FROM users
  WHERE id = ?
  UNION ALL
  SELECT u.id
  FROM users u
  INNER JOIN hierarchy h ON u.created_by = h.id
)
SELECT u.id, u.name, u.email, u.password_hash, u.role, u.is_active, u.created_by, u.created_at, u.updated_at
FROM users u
INNER JOIN hierarchy h ON h.id = u.id
ORDER BY u.created_at DESC
LIMIT ?`)).
		WithArgs(uint64(1), 50).
		WillReturnRows(rows)

	items, err := repo.ListByScope(context.Background(), 1, 50)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Nil(t, items[0].CreatedBy)
	require.NotNil(t, items[1].CreatedBy)
	require.Equal(t, uint64(1), *items[1].CreatedBy)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_IsInScope(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
WITH RECURSIVE hierarchy AS (
  SELECT id
  FROM users
  WHERE id = ?
  UNION ALL
  SELECT u.id
  FROM users u
  INNER JOIN hierarchy h ON u.created_by = h.id
)
SELECT COUNT(1)
FROM hierarchy
WHERE id = ?`)).
		WithArgs(uint64(10), uint64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	ok, err := repo.IsInScope(context.Background(), 10, 12)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, mock.ExpectationsWereMet())
}
