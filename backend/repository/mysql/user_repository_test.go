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

	rows := sqlmock.NewRows([]string{"id", "name", "email", "password_hash", "role", "is_active", "created_at", "updated_at"}).
		AddRow(1, "Jane", "jane@example.com", "hash", model.UserRoleUser, true, now, now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, email, password_hash, role, is_active, created_at, updated_at FROM users WHERE email = ? LIMIT 1")).
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

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, email, password_hash, role, is_active, created_at, updated_at FROM users WHERE email = ? LIMIT 1")).
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
	user := &model.User{Name: "John", Email: "john@example.com", PasswordHash: "hash", Role: model.UserRoleAdmin, IsActive: true}

	expect := mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (name, email, password_hash, role, is_active) VALUES (?, ?, ?, ?, ?)"))
	expect.WithArgs("John", "john@example.com", "hash", model.UserRoleAdmin, true).WillReturnResult(sqlmock.NewResult(5, 1))

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
