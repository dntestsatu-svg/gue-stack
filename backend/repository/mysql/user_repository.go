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
	query := `INSERT INTO users (name, email, password_hash, role, is_active) VALUES (?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query, user.Name, user.Email, user.PasswordHash, user.Role, user.IsActive)
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
	query := `SELECT id, name, email, password_hash, role, is_active, created_at, updated_at FROM users WHERE email = ? LIMIT 1`
	user := &model.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("query user by email: %w", err)
	}
	return user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uint64) (*model.User, error) {
	query := `SELECT id, name, email, password_hash, role, is_active, created_at, updated_at FROM users WHERE id = ? LIMIT 1`
	user := &model.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("query user by id: %w", err)
	}
	return user, nil
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
