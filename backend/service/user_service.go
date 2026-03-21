package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/example/gue/backend/cache"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/repository"
)

type UserUseCase interface {
	Me(ctx context.Context, userID uint64) (*UserDTO, error)
}

type UserService struct {
	userRepo     repository.UserRepository
	cache        cache.Cache
	cacheEnabled bool
	userCacheTTL time.Duration
	logger       *slog.Logger
}

func NewUserService(userRepo repository.UserRepository, cache cache.Cache, cacheEnabled bool, userCacheTTL time.Duration, logger *slog.Logger) *UserService {
	return &UserService{
		userRepo:     userRepo,
		cache:        cache,
		cacheEnabled: cacheEnabled,
		userCacheTTL: userCacheTTL,
		logger:       logger,
	}
}

func (s *UserService) Me(ctx context.Context, userID uint64) (*UserDTO, error) {
	cacheKey := fmt.Sprintf("user:me:%d", userID)
	if s.cacheEnabled && s.cache != nil {
		cached, err := s.cache.Get(ctx, cacheKey)
		if err == nil {
			var dto UserDTO
			if unmarshalErr := json.Unmarshal(cached, &dto); unmarshalErr == nil {
				return &dto, nil
			}
		}
		if err != nil && !errors.Is(err, cache.ErrCacheMiss) {
			s.logger.Error("cache get failed", "key", cacheKey, "error", err.Error())
		}
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusNotFound, "user not found", nil)
		}
		return nil, apperror.New(http.StatusInternalServerError, "failed to fetch user", err.Error())
	}

	dto := &UserDTO{ID: user.ID, Name: user.Name, Email: user.Email}
	if s.cacheEnabled && s.cache != nil {
		if err := s.cache.Set(ctx, cacheKey, dto, s.userCacheTTL); err != nil {
			s.logger.Error("cache set failed", "key", cacheKey, "error", err.Error())
		}
	}

	return dto, nil
}
