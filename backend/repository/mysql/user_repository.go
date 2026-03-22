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
	return r.ListPageByScope(ctx, actorUserID, repository.UserListFilter{
		Limit:  limit,
		Offset: 0,
	})
}

func (r *UserRepository) ListPageByScope(ctx context.Context, actorUserID uint64, filter repository.UserListFilter) ([]model.User, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	query, args := buildUserScopeListQuery(actorUserID, filter)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query list users: %w", err)
	}
	defer rows.Close()

	users := make([]model.User, 0, filter.Limit)
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

func (r *UserRepository) CountByScope(ctx context.Context, actorUserID uint64, filter repository.UserListFilter) (uint64, error) {
	query, args := buildUserScopeCountQuery(actorUserID, filter)
	var total uint64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count users by scope: %w", err)
	}
	return total, nil
}

func buildUserScopeListQuery(actorUserID uint64, filter repository.UserListFilter) (string, []any) {
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
WHERE 1 = 1`
	args := []any{actorUserID}

	if filter.Role != "" {
		query += "\nAND u.role = ?"
		args = append(args, filter.Role)
	}

	if search := strings.ToLower(strings.TrimSpace(filter.SearchTerm)); search != "" {
		pattern := "%" + search + "%"
		query += "\nAND (LOWER(u.name) LIKE ? OR LOWER(u.email) LIKE ?)"
		args = append(args, pattern, pattern)
	}

	query += "\nORDER BY u.created_at DESC, u.id DESC\nLIMIT ? OFFSET ?"
	args = append(args, filter.Limit, filter.Offset)
	return query, args
}

func buildUserScopeCountQuery(actorUserID uint64, filter repository.UserListFilter) (string, []any) {
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
SELECT COUNT(*)
FROM users u
INNER JOIN hierarchy h ON h.id = u.id
WHERE 1 = 1`
	args := []any{actorUserID}

	if filter.Role != "" {
		query += "\nAND u.role = ?"
		args = append(args, filter.Role)
	}

	if search := strings.ToLower(strings.TrimSpace(filter.SearchTerm)); search != "" {
		pattern := "%" + search + "%"
		query += "\nAND (LOWER(u.name) LIKE ? OR LOWER(u.email) LIKE ?)"
		args = append(args, pattern, pattern)
	}

	return query, args
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

func (r *UserRepository) UpdateActive(ctx context.Context, id uint64, isActive bool) error {
	query := `UPDATE users SET is_active = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, isActive, id)
	if err != nil {
		return fmt.Errorf("update user active status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected for update user active status: %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id uint64, passwordHash string) error {
	query := `UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, passwordHash, id)
	if err != nil {
		return fmt.Errorf("update user password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected for update user password: %w", err)
	}
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uint64) error {
	query := `DELETE FROM users WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected for delete user: %w", err)
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
