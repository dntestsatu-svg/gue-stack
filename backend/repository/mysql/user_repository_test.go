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

	rows := sqlmock.NewRows([]string{"id", "name", "email", "password_hash", "created_at", "updated_at"}).
		AddRow(1, "Jane", "jane@example.com", "hash", now, now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, email, password_hash, created_at, updated_at FROM users WHERE email = ? LIMIT 1")).
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

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, email, password_hash, created_at, updated_at FROM users WHERE email = ? LIMIT 1")).
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
	user := &model.User{Name: "John", Email: "john@example.com", PasswordHash: "hash"}

	expect := mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (name, email, password_hash) VALUES (?, ?, ?)"))
	expect.WithArgs("John", "john@example.com", "hash").WillReturnResult(sqlmock.NewResult(5, 1))

	err := repo.Create(context.Background(), user)
	require.NoError(t, err)
	require.Equal(t, uint64(5), user.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}
