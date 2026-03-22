package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/repository"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `INSERT INTO users (name, email, password_hash, role, is_active, created_by) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query, user.Name, user.Email, user.PasswordHash, user.Role, user.IsActive, nullableUserCreatedBy(user.CreatedBy))
	if err != nil {
		return fmt.Errorf("exec create user: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get inserted user id: %w", err)
	}
	user.ID = uint64(id)
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `SELECT id, name, email, password_hash, role, is_active, created_by, created_at, updated_at FROM users WHERE email = ? LIMIT 1`
	user := &model.User{}
	err := scanUserRow(r.db.QueryRowContext(ctx, query, email), user)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("query user by email: %w", err)
	}
	return user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uint64) (*model.User, error) {
	query := `SELECT id, name, email, password_hash, role, is_active, created_by, created_at, updated_at FROM users WHERE id = ? LIMIT 1`
	user := &model.User{}
	err := scanUserRow(r.db.QueryRowContext(ctx, query, id), user)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("query user by id: %w", err)
	}
	return user, nil
}

func (r *UserRepository) ListByScope(ctx context.Context, actorUserID uint64, limit int) ([]model.User, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	query := `
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
LIMIT ?`
	rows, err := r.db.QueryContext(ctx, query, actorUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("query list users: %w", err)
	}
	defer rows.Close()

	users := make([]model.User, 0, limit)
	for rows.Next() {
		var user model.User
		if err := scanUserRows(rows, &user); err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user rows: %w", err)
	}
	return users, nil
}

func (r *UserRepository) IsInScope(ctx context.Context, actorUserID uint64, targetUserID uint64) (bool, error) {
	query := `
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
WHERE id = ?`
	var count int
	if err := r.db.QueryRowContext(ctx, query, actorUserID, targetUserID).Scan(&count); err != nil {
		return false, fmt.Errorf("query user scope relation: %w", err)
	}
	return count > 0, nil
}

func (r *UserRepository) UpdateRole(ctx context.Context, id uint64, role model.UserRole) error {
	query := `UPDATE users SET role = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, role, id)
	if err != nil {
		return fmt.Errorf("update user role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected for update user role: %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func scanUserRow(scanner interface {
	Scan(dest ...any) error
}, user *model.User) error {
	var createdBy sql.NullInt64
	if err := scanner.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&createdBy,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return err
	}
	if createdBy.Valid {
		value := uint64(createdBy.Int64)
		user.CreatedBy = &value
	} else {
		user.CreatedBy = nil
	}
	return nil
}

func scanUserRows(rows *sql.Rows, user *model.User) error {
	return scanUserRow(rows, user)
}

func nullableUserCreatedBy(value *uint64) any {
	if value == nil {
		return nil
	}
	return *value
}
