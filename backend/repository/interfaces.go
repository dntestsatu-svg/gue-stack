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

type RefreshTokenStore interface {
	Store(ctx context.Context, tokenID string, userID uint64, ttl time.Duration) error
	GetUserID(ctx context.Context, tokenID string) (uint64, error)
	Delete(ctx context.Context, tokenID string) error
}
