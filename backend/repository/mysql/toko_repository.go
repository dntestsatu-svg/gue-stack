package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/repository"
)

type TokoRepository struct {
	db *sql.DB
}

func NewTokoRepository(db *sql.DB) *TokoRepository {
	return &TokoRepository{db: db}
}

func (r *TokoRepository) GetByID(ctx context.Context, id uint64) (*model.Toko, error) {
	query := `SELECT id, name, token, charge, callback_url, created_at, updated_at FROM tokos WHERE id = ? LIMIT 1`
	return r.getOne(ctx, query, id)
}

func (r *TokoRepository) GetByToken(ctx context.Context, token string) (*model.Toko, error) {
	query := `SELECT id, name, token, charge, callback_url, created_at, updated_at FROM tokos WHERE token = ? LIMIT 1`
	return r.getOne(ctx, query, token)
}

func (r *TokoRepository) getOne(ctx context.Context, query string, arg any) (*model.Toko, error) {
	toko := &model.Toko{}
	var callbackURL sql.NullString
	if err := r.db.QueryRowContext(ctx, query, arg).Scan(
		&toko.ID,
		&toko.Name,
		&toko.Token,
		&toko.Charge,
		&callbackURL,
		&toko.CreatedAt,
		&toko.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("query toko: %w", err)
	}
	if callbackURL.Valid {
		toko.CallbackURL = &callbackURL.String
	}
	return toko, nil
}
