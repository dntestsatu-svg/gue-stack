package repository

import (
	"context"
	"time"

	"github.com/example/gue/backend/model"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uint64) (*model.User, error)
}

type TokoRepository interface {
	GetByID(ctx context.Context, id uint64) (*model.Toko, error)
	GetByToken(ctx context.Context, token string) (*model.Toko, error)
}

type TransactionRepository interface {
	Create(ctx context.Context, trx *model.Transaction) error
	GetByReference(ctx context.Context, reference string) (*model.Transaction, error)
	UpdateStatusByReference(ctx context.Context, reference string, status model.TransactionStatus) error
	UpdateStatusByReferenceAndToko(ctx context.Context, reference string, tokoID uint64, status model.TransactionStatus) error
}

type RefreshTokenStore interface {
	Store(ctx context.Context, tokenID string, userID uint64, ttl time.Duration) error
	GetUserID(ctx context.Context, tokenID string) (uint64, error)
	Delete(ctx context.Context, tokenID string) error
}
