package redisstore

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/example/gue/backend/repository"
	"github.com/redis/go-redis/v9"
)

type RefreshTokenStore struct {
	client *redis.Client
}

func NewRefreshTokenStore(client *redis.Client) *RefreshTokenStore {
	return &RefreshTokenStore{client: client}
}

func (s *RefreshTokenStore) Store(ctx context.Context, tokenID string, userID uint64, ttl time.Duration) error {
	return s.client.Set(ctx, key(tokenID), strconv.FormatUint(userID, 10), ttl).Err()
}

func (s *RefreshTokenStore) GetUserID(ctx context.Context, tokenID string) (uint64, error) {
	val, err := s.client.Get(ctx, key(tokenID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, repository.ErrNotFound
		}
		return 0, err
	}
	userID, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func (s *RefreshTokenStore) Delete(ctx context.Context, tokenID string) error {
	if err := s.client.Del(ctx, key(tokenID)).Err(); err != nil {
		return err
	}
	return nil
}

func key(tokenID string) string {
	return "auth:refresh:" + tokenID
}
