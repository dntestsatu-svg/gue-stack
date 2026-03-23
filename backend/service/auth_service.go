package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/apperror"
	jwtpkg "github.com/example/gue/backend/pkg/jwt"
	"github.com/example/gue/backend/pkg/password"
	"github.com/example/gue/backend/queue"
	"github.com/example/gue/backend/repository"
	"github.com/go-playground/validator/v10"
)

type AuthUseCase interface {
	Register(ctx context.Context, input RegisterInput) (*AuthResult, error)
	Login(ctx context.Context, input LoginInput) (*AuthResult, error)
	Refresh(ctx context.Context, refreshToken string) (*AuthResult, error)
	SessionStatus(ctx context.Context, refreshToken string) (*SessionStatusResult, error)
	Logout(ctx context.Context, refreshToken string) error
}

type AuthService struct {
	userRepo      repository.UserRepository
	refreshStore  repository.RefreshTokenStore
	tokenManager  *jwtpkg.Manager
	queueProducer queue.Producer
	validate      *validator.Validate
	logger        *slog.Logger
}

type RegisterInput struct {
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type LoginInput struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type AuthResult struct {
	User         UserDTO `json:"user"`
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	ExpiresIn    int64   `json:"expires_in"`
}

type SessionStatusResult struct {
	Authenticated bool     `json:"authenticated"`
	User          *UserDTO `json:"user,omitempty"`
}

type UserDTO struct {
	ID       uint64         `json:"id"`
	Name     string         `json:"name"`
	Email    string         `json:"email"`
	Role     model.UserRole `json:"role"`
	IsActive bool           `json:"is_active"`
}

func NewAuthService(
	userRepo repository.UserRepository,
	refreshStore repository.RefreshTokenStore,
	tokenManager *jwtpkg.Manager,
	queueProducer queue.Producer,
	logger *slog.Logger,
) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		refreshStore:  refreshStore,
		tokenManager:  tokenManager,
		queueProducer: queueProducer,
		validate:      validator.New(validator.WithRequiredStructEnabled()),
		logger:        logger,
	}
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	existing, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err == nil && existing != nil {
		return nil, apperror.New(http.StatusConflict, "email is already registered", nil)
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.New(http.StatusInternalServerError, "failed to check existing account", err.Error())
	}

	hash, err := password.Hash(input.Password)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to hash password", err.Error())
	}

	user := &model.User{
		Name:         strings.TrimSpace(input.Name),
		Email:        input.Email,
		PasswordHash: hash,
		Role:         model.UserRoleAdmin,
		IsActive:     true,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to create user", err.Error())
	}

	tokenPair, err := s.tokenManager.GenerateTokenPair(user.ID, user.Email, time.Now().UTC())
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to generate tokens", err.Error())
	}

	if err := s.refreshStore.Store(ctx, tokenPair.RefreshID, user.ID, s.tokenManager.RefreshTTL()); err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to persist refresh token", err.Error())
	}

	if s.queueProducer != nil {
		if err := s.queueProducer.EnqueueWelcomeEmail(ctx, user.Email, user.Name); err != nil {
			s.logger.Error("failed to enqueue welcome email", "error", err.Error(), "user_id", user.ID)
		}
	}

	return &AuthResult{
		User:         UserDTO{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role, IsActive: user.IsActive},
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}

	user, err := s.userRepo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(input.Email)))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusUnauthorized, "invalid credentials", nil)
		}
		return nil, apperror.New(http.StatusInternalServerError, "failed to fetch user", err.Error())
	}

	if err := password.Compare(user.PasswordHash, input.Password); err != nil {
		return nil, apperror.New(http.StatusUnauthorized, "invalid credentials", nil)
	}
	if !user.IsActive {
		return nil, apperror.New(http.StatusForbidden, "user account is inactive", nil)
	}

	tokenPair, err := s.tokenManager.GenerateTokenPair(user.ID, user.Email, time.Now().UTC())
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to generate tokens", err.Error())
	}

	if err := s.refreshStore.Store(ctx, tokenPair.RefreshID, user.ID, s.tokenManager.RefreshTTL()); err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to persist refresh token", err.Error())
	}

	return &AuthResult{
		User:         UserDTO{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role, IsActive: user.IsActive},
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	claims, err := s.tokenManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, apperror.New(http.StatusUnauthorized, "invalid refresh token", nil)
	}

	storedUserID, err := s.refreshStore.GetUserID(ctx, claims.TokenID)
	if err != nil {
		return nil, apperror.New(http.StatusUnauthorized, "refresh token revoked or expired", nil)
	}
	if storedUserID != claims.UserID {
		return nil, apperror.New(http.StatusUnauthorized, "refresh token does not match user", nil)
	}

	if err := s.refreshStore.Delete(ctx, claims.TokenID); err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to rotate refresh token", err.Error())
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, apperror.New(http.StatusUnauthorized, "user not found", nil)
	}
	if !user.IsActive {
		return nil, apperror.New(http.StatusForbidden, "user account is inactive", nil)
	}

	newTokens, err := s.tokenManager.GenerateTokenPair(user.ID, user.Email, time.Now().UTC())
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to generate tokens", err.Error())
	}

	if err := s.refreshStore.Store(ctx, newTokens.RefreshID, user.ID, s.tokenManager.RefreshTTL()); err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to persist refresh token", err.Error())
	}

	return &AuthResult{
		User:         UserDTO{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role, IsActive: user.IsActive},
		AccessToken:  newTokens.AccessToken,
		RefreshToken: newTokens.RefreshToken,
		ExpiresIn:    newTokens.ExpiresIn,
	}, nil
}

func (s *AuthService) SessionStatus(ctx context.Context, refreshToken string) (*SessionStatusResult, error) {
	claims, err := s.tokenManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return &SessionStatusResult{Authenticated: false}, nil
	}

	storedUserID, err := s.refreshStore.GetUserID(ctx, claims.TokenID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return &SessionStatusResult{Authenticated: false}, nil
		}
		return nil, apperror.New(http.StatusInternalServerError, "failed to inspect session token", err.Error())
	}
	if storedUserID != claims.UserID {
		return &SessionStatusResult{Authenticated: false}, nil
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return &SessionStatusResult{Authenticated: false}, nil
		}
		return nil, apperror.New(http.StatusInternalServerError, "failed to fetch session user", err.Error())
	}
	if !user.IsActive {
		return &SessionStatusResult{Authenticated: false}, nil
	}

	userDTO := &UserDTO{
		ID:       user.ID,
		Name:     user.Name,
		Email:    user.Email,
		Role:     user.Role,
		IsActive: user.IsActive,
	}

	return &SessionStatusResult{
		Authenticated: true,
		User:          userDTO,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.tokenManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return apperror.New(http.StatusUnauthorized, "invalid refresh token", nil)
	}

	if err := s.refreshStore.Delete(ctx, claims.TokenID); err != nil {
		return apperror.New(http.StatusInternalServerError, "failed to revoke refresh token", err.Error())
	}

	return nil
}
