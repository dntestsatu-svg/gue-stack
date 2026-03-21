package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/example/gue/backend/pkg/password"
)

type DevUserInput struct {
	Name     string
	Email    string
	Password string
}

func EnsureSingleDevUser(ctx context.Context, db *sql.DB, input DevUserInput) error {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = "Developer"
	}
	if email == "" {
		return fmt.Errorf("bootstrap dev email is required")
	}
	if strings.TrimSpace(input.Password) == "" {
		return fmt.Errorf("bootstrap dev password is required")
	}

	passwordHash, err := password.Hash(strings.TrimSpace(input.Password))
	if err != nil {
		return fmt.Errorf("hash bootstrap dev password: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin bootstrap transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE users SET role = 'superadmin', updated_at = CURRENT_TIMESTAMP WHERE role = 'dev' AND email <> ?`,
		email,
	); err != nil {
		return fmt.Errorf("demote additional dev users: %w", err)
	}

	var userID uint64
	err = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE email = ? LIMIT 1`, email).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			result, insertErr := tx.ExecContext(
				ctx,
				`INSERT INTO users (name, email, password_hash, role, is_active) VALUES (?, ?, ?, 'dev', 1)`,
				name,
				email,
				passwordHash,
			)
			if insertErr != nil {
				return fmt.Errorf("insert bootstrap dev user: %w", insertErr)
			}
			lastID, idErr := result.LastInsertId()
			if idErr != nil {
				return fmt.Errorf("read bootstrap dev user id: %w", idErr)
			}
			userID = uint64(lastID)
		} else {
			return fmt.Errorf("query bootstrap dev user: %w", err)
		}
	} else {
		if _, updateErr := tx.ExecContext(
			ctx,
			`UPDATE users SET name = ?, password_hash = ?, role = 'dev', is_active = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
			name,
			passwordHash,
			userID,
		); updateErr != nil {
			return fmt.Errorf("update bootstrap dev user: %w", updateErr)
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("commit bootstrap transaction: %w", commitErr)
	}
	return nil
}
