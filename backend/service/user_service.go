package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/example/gue/backend/cache"
	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/password"
	"github.com/example/gue/backend/queue"
	"github.com/example/gue/backend/repository"
	"github.com/go-playground/validator/v10"
)

type UserUseCase interface {
	Me(ctx context.Context, userID uint64) (*UserDTO, error)
	List(ctx context.Context, actorUserID uint64, actorRole model.UserRole, limit int) ([]UserDTO, error)
	Create(ctx context.Context, actorUserID uint64, actorRole model.UserRole, input CreateUserInput) (*UserDTO, error)
	UpdateRole(ctx context.Context, actorUserID uint64, actorRole model.UserRole, targetUserID uint64, input UpdateUserRoleInput) (*UserDTO, error)
}

type UserService struct {
	userRepo      repository.UserRepository
	cache         cache.Cache
	cacheEnabled  bool
	userCacheTTL  time.Duration
	queueProducer queue.Producer
	logger        *slog.Logger
	validate      *validator.Validate
}

func NewUserService(
	userRepo repository.UserRepository,
	cache cache.Cache,
	cacheEnabled bool,
	userCacheTTL time.Duration,
	queueProducer queue.Producer,
	logger *slog.Logger,
) *UserService {
	return &UserService{
		userRepo:      userRepo,
		cache:         cache,
		cacheEnabled:  cacheEnabled,
		userCacheTTL:  userCacheTTL,
		queueProducer: queueProducer,
		logger:        logger,
		validate:      validator.New(validator.WithRequiredStructEnabled()),
	}
}

type CreateUserInput struct {
	Name     string  `json:"name" validate:"required,min=2,max=100"`
	Email    string  `json:"email" validate:"required,email,max=255"`
	Password string  `json:"password" validate:"required,min=8,max=72"`
	Role     *string `json:"role,omitempty" validate:"omitempty,oneof=dev superadmin admin user"`
	IsActive *bool   `json:"is_active,omitempty"`
}

type UpdateUserRoleInput struct {
	Role string `json:"role" validate:"required,oneof=dev superadmin admin user"`
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

	dto := &UserDTO{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role, IsActive: user.IsActive}
	if s.cacheEnabled && s.cache != nil {
		if err := s.cache.Set(ctx, cacheKey, dto, s.userCacheTTL); err != nil {
			s.logger.Error("cache set failed", "key", cacheKey, "error", err.Error())
		}
	}

	return dto, nil
}

func (s *UserService) List(ctx context.Context, actorUserID uint64, actorRole model.UserRole, limit int) ([]UserDTO, error) {
	if !canCreateUser(actorRole) {
		return nil, apperror.New(http.StatusForbidden, "insufficient role to list users", nil)
	}

	users, err := s.userRepo.ListByScope(ctx, actorUserID, limit)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to list users", err.Error())
	}

	items := make([]UserDTO, 0, len(users))
	for _, user := range users {
		items = append(items, UserDTO{
			ID:       user.ID,
			Name:     user.Name,
			Email:    user.Email,
			Role:     user.Role,
			IsActive: user.IsActive,
		})
	}
	return items, nil
}

func (s *UserService) Create(ctx context.Context, actorUserID uint64, actorRole model.UserRole, input CreateUserInput) (*UserDTO, error) {
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}
	if !canCreateUser(actorRole) {
		return nil, apperror.New(http.StatusForbidden, "insufficient role to create user", nil)
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))
	existing, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, apperror.New(http.StatusConflict, "email is already registered", nil)
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.New(http.StatusInternalServerError, "failed to check existing account", err.Error())
	}

	role := model.UserRoleUser
	if input.Role != nil && strings.TrimSpace(*input.Role) != "" {
		role = model.UserRole(strings.ToLower(strings.TrimSpace(*input.Role)))
		if role == model.UserRoleDev {
			return nil, apperror.New(http.StatusForbidden, "role dev is reserved for bootstrap only", nil)
		}
		if !canCreateRole(actorRole, role) {
			return nil, apperror.New(http.StatusForbidden, "insufficient role to create target role", nil)
		}
	}
	if !canCreateRole(actorRole, role) {
		return nil, apperror.New(http.StatusForbidden, "insufficient role to create target role", nil)
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	hash, err := password.Hash(input.Password)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to hash password", err.Error())
	}

	user := &model.User{
		Name:         strings.TrimSpace(input.Name),
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		IsActive:     isActive,
		CreatedBy:    &actorUserID,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to create user", err.Error())
	}
	if s.queueProducer != nil {
		if err := s.queueProducer.EnqueueWelcomeEmail(ctx, user.Email, user.Name); err != nil {
			s.logger.Error("failed to enqueue welcome email", "error", err.Error(), "user_id", user.ID)
		}
	}

	return &UserDTO{
		ID:       user.ID,
		Name:     user.Name,
		Email:    user.Email,
		Role:     user.Role,
		IsActive: user.IsActive,
	}, nil
}

func (s *UserService) UpdateRole(ctx context.Context, actorUserID uint64, actorRole model.UserRole, targetUserID uint64, input UpdateUserRoleInput) (*UserDTO, error) {
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}
	if !canChangeRole(actorRole) {
		return nil, apperror.New(http.StatusForbidden, "insufficient role to update user role", nil)
	}

	targetUser, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusNotFound, "user not found", nil)
		}
		return nil, apperror.New(http.StatusInternalServerError, "failed to fetch target user", err.Error())
	}

	newRole := model.UserRole(strings.ToLower(strings.TrimSpace(input.Role)))
	if newRole == model.UserRoleDev {
		return nil, apperror.New(http.StatusForbidden, "role dev is reserved for bootstrap only", nil)
	}
	if !canCreateRole(actorRole, newRole) {
		return nil, apperror.New(http.StatusForbidden, "insufficient role to assign target role", nil)
	}
	inScope, err := s.userRepo.IsInScope(ctx, actorUserID, targetUserID)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to verify user scope", err.Error())
	}
	if !inScope {
		return nil, apperror.New(http.StatusForbidden, "target user is outside your hierarchy scope", nil)
	}
	if targetUser.Role != newRole {
		if err := s.userRepo.UpdateRole(ctx, targetUserID, newRole); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, apperror.New(http.StatusNotFound, "user not found", nil)
			}
			return nil, apperror.New(http.StatusInternalServerError, "failed to update user role", err.Error())
		}
		targetUser.Role = newRole
	}

	if s.cacheEnabled && s.cache != nil {
		cacheKey := fmt.Sprintf("user:me:%d", targetUserID)
		if err := s.cache.Delete(ctx, cacheKey); err != nil {
			s.logger.Error("cache delete failed", "key", cacheKey, "error", err.Error())
		}
	}

	return &UserDTO{
		ID:       targetUser.ID,
		Name:     targetUser.Name,
		Email:    targetUser.Email,
		Role:     targetUser.Role,
		IsActive: targetUser.IsActive,
	}, nil
}

func canChangeRole(role model.UserRole) bool {
	return role == model.UserRoleDev || role == model.UserRoleSuperAdmin
}

func canCreateUser(role model.UserRole) bool {
	return role != model.UserRoleUser && role != ""
}

func canCreateRole(actorRole model.UserRole, targetRole model.UserRole) bool {
	switch actorRole {
	case model.UserRoleDev:
		return targetRole == model.UserRoleSuperAdmin || targetRole == model.UserRoleAdmin || targetRole == model.UserRoleUser
	case model.UserRoleSuperAdmin:
		return targetRole == model.UserRoleAdmin || targetRole == model.UserRoleUser
	case model.UserRoleAdmin:
		return targetRole == model.UserRoleUser
	default:
		return false
	}
}
