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
	List(ctx context.Context, actorUserID uint64, actorRole model.UserRole, query UserListQuery) (*UserListPage, error)
	Create(ctx context.Context, actorUserID uint64, actorRole model.UserRole, input CreateUserInput) (*UserDTO, error)
	UpdateRole(ctx context.Context, actorUserID uint64, actorRole model.UserRole, targetUserID uint64, input UpdateUserRoleInput) (*UserDTO, error)
	UpdateActive(ctx context.Context, actorUserID uint64, actorRole model.UserRole, targetUserID uint64, input UpdateUserActiveInput) (*UserDTO, error)
	Delete(ctx context.Context, actorUserID uint64, actorRole model.UserRole, targetUserID uint64) error
	ChangePassword(ctx context.Context, userID uint64, input ChangePasswordInput) error
}

type UserService struct {
	userRepo      repository.UserRepository
	cache         cache.Cache
	cacheEnabled  bool
	userCacheTTL  time.Duration
	listCacheTTL  time.Duration
	queueProducer queue.Producer
	logger        *slog.Logger
	validate      *validator.Validate
}

func NewUserService(
	userRepo repository.UserRepository,
	cache cache.Cache,
	cacheEnabled bool,
	userCacheTTL time.Duration,
	listCacheTTL time.Duration,
	queueProducer queue.Producer,
	logger *slog.Logger,
) *UserService {
	if logger == nil {
		logger = slog.Default()
	}
	if listCacheTTL <= 0 {
		listCacheTTL = 5 * time.Minute
	}
	return &UserService{
		userRepo:      userRepo,
		cache:         cache,
		cacheEnabled:  cacheEnabled,
		userCacheTTL:  userCacheTTL,
		listCacheTTL:  listCacheTTL,
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

type UpdateUserActiveInput struct {
	IsActive *bool `json:"is_active" validate:"required"`
}

type ChangePasswordInput struct {
	CurrentPassword string `json:"current_password" validate:"required,min=8,max=72"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=72"`
	ConfirmPassword string `json:"confirm_password" validate:"required,min=8,max=72"`
}

type UserListQuery struct {
	Limit      int
	Offset     int
	SearchTerm string
	Role       model.UserRole
}

type UserListPage struct {
	Items   []UserDTO `json:"items"`
	Total   uint64    `json:"total"`
	Limit   int       `json:"limit"`
	Offset  int       `json:"offset"`
	HasMore bool      `json:"has_more"`
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

func (s *UserService) List(ctx context.Context, actorUserID uint64, actorRole model.UserRole, query UserListQuery) (*UserListPage, error) {
	if !canCreateUser(actorRole) {
		return nil, apperror.New(http.StatusForbidden, "insufficient role to list users", nil)
	}

	normalized, err := normalizeUserListQuery(query)
	if err != nil {
		return nil, err
	}

	cacheKey := s.userListCacheKey(ctx, actorUserID, actorRole, normalized)
	if cached, ok := getCachedJSON[UserListPage](ctx, s.cache, s.cacheEnabled, cacheKey, s.logger); ok {
		return cached, nil
	}

	total, err := s.userRepo.CountByScope(ctx, actorUserID, repository.UserListFilter{
		SearchTerm: normalized.SearchTerm,
		Role:       normalized.Role,
	})
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to count users", err.Error())
	}

	users, err := s.userRepo.ListPageByScope(ctx, actorUserID, repository.UserListFilter{
		Limit:      normalized.Limit,
		Offset:     normalized.Offset,
		SearchTerm: normalized.SearchTerm,
		Role:       normalized.Role,
	})
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
	page := &UserListPage{
		Items:   items,
		Total:   total,
		Limit:   normalized.Limit,
		Offset:  normalized.Offset,
		HasMore: uint64(normalized.Offset+len(items)) < total,
	}
	setCachedJSON(ctx, s.cache, s.cacheEnabled, cacheKey, page, s.listCacheTTL, s.logger)
	return page, nil
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
	s.invalidateUserListCache(ctx)

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
		s.invalidateUserMeCache(ctx, targetUserID)
	}
	s.invalidateUserListCache(ctx)

	return &UserDTO{
		ID:       targetUser.ID,
		Name:     targetUser.Name,
		Email:    targetUser.Email,
		Role:     targetUser.Role,
		IsActive: targetUser.IsActive,
	}, nil
}

func (s *UserService) UpdateActive(ctx context.Context, actorUserID uint64, actorRole model.UserRole, targetUserID uint64, input UpdateUserActiveInput) (*UserDTO, error) {
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}
	if !canCreateUser(actorRole) {
		return nil, apperror.New(http.StatusForbidden, "insufficient role to update user status", nil)
	}
	if targetUserID == actorUserID {
		return nil, apperror.New(http.StatusForbidden, "you cannot update your own active status", nil)
	}

	targetUser, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusNotFound, "user not found", nil)
		}
		return nil, apperror.New(http.StatusInternalServerError, "failed to fetch target user", err.Error())
	}
	if !canManageTargetRole(actorRole, targetUser.Role) {
		return nil, apperror.New(http.StatusForbidden, "insufficient role to update target user status", nil)
	}

	inScope, err := s.userRepo.IsInScope(ctx, actorUserID, targetUserID)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to verify user scope", err.Error())
	}
	if !inScope {
		return nil, apperror.New(http.StatusForbidden, "target user is outside your hierarchy scope", nil)
	}

	if targetUser.IsActive != *input.IsActive {
		if err := s.userRepo.UpdateActive(ctx, targetUserID, *input.IsActive); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, apperror.New(http.StatusNotFound, "user not found", nil)
			}
			return nil, apperror.New(http.StatusInternalServerError, "failed to update user status", err.Error())
		}
		targetUser.IsActive = *input.IsActive
	}

	s.invalidateUserMeCache(ctx, targetUserID)
	s.invalidateUserListCache(ctx)

	return &UserDTO{
		ID:       targetUser.ID,
		Name:     targetUser.Name,
		Email:    targetUser.Email,
		Role:     targetUser.Role,
		IsActive: targetUser.IsActive,
	}, nil
}

func (s *UserService) Delete(ctx context.Context, actorUserID uint64, actorRole model.UserRole, targetUserID uint64) error {
	if !canCreateUser(actorRole) {
		return apperror.New(http.StatusForbidden, "insufficient role to delete user", nil)
	}
	if targetUserID == actorUserID {
		return apperror.New(http.StatusForbidden, "you cannot delete your own account", nil)
	}

	targetUser, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(http.StatusNotFound, "user not found", nil)
		}
		return apperror.New(http.StatusInternalServerError, "failed to fetch target user", err.Error())
	}
	if !canManageTargetRole(actorRole, targetUser.Role) {
		return apperror.New(http.StatusForbidden, "insufficient role to delete target user", nil)
	}

	inScope, err := s.userRepo.IsInScope(ctx, actorUserID, targetUserID)
	if err != nil {
		return apperror.New(http.StatusInternalServerError, "failed to verify user scope", err.Error())
	}
	if !inScope {
		return apperror.New(http.StatusForbidden, "target user is outside your hierarchy scope", nil)
	}

	if err := s.userRepo.Delete(ctx, targetUserID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(http.StatusNotFound, "user not found", nil)
		}
		return apperror.New(http.StatusInternalServerError, "failed to delete user", err.Error())
	}

	s.invalidateUserMeCache(ctx, targetUserID)
	s.invalidateUserListCache(ctx)
	return nil
}

func (s *UserService) ChangePassword(ctx context.Context, userID uint64, input ChangePasswordInput) error {
	if err := s.validate.Struct(input); err != nil {
		return apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}

	if strings.TrimSpace(input.NewPassword) != strings.TrimSpace(input.ConfirmPassword) {
		return apperror.New(http.StatusBadRequest, "password confirmation does not match", nil)
	}
	if input.CurrentPassword == input.NewPassword {
		return apperror.New(http.StatusBadRequest, "new password must be different from current password", nil)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(http.StatusNotFound, "user not found", nil)
		}
		return apperror.New(http.StatusInternalServerError, "failed to fetch user", err.Error())
	}

	if err := password.Compare(user.PasswordHash, input.CurrentPassword); err != nil {
		return apperror.New(http.StatusUnauthorized, "current password is invalid", nil)
	}

	hash, err := password.Hash(input.NewPassword)
	if err != nil {
		return apperror.New(http.StatusInternalServerError, "failed to hash password", err.Error())
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, hash); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(http.StatusNotFound, "user not found", nil)
		}
		return apperror.New(http.StatusInternalServerError, "failed to update password", err.Error())
	}

	return nil
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

func canManageTargetRole(actorRole model.UserRole, targetRole model.UserRole) bool {
	return canCreateRole(actorRole, targetRole)
}

func normalizeUserListQuery(query UserListQuery) (UserListQuery, error) {
	if query.Limit <= 0 {
		query.Limit = 10
	}
	if query.Limit > 50 {
		query.Limit = 50
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	query.SearchTerm = strings.TrimSpace(query.SearchTerm)
	if query.Role == "" {
		return query, nil
	}

	switch query.Role {
	case model.UserRoleDev, model.UserRoleSuperAdmin, model.UserRoleAdmin, model.UserRoleUser:
		return query, nil
	default:
		return UserListQuery{}, apperror.New(http.StatusBadRequest, "invalid role query parameter", nil)
	}
}

func (s *UserService) userListCacheKey(ctx context.Context, actorUserID uint64, actorRole model.UserRole, query UserListQuery) string {
	namespace := getCacheNamespaceToken(ctx, s.cache, s.cacheEnabled, s.userListNamespaceKey(), s.logger)
	return buildHashedCacheKey(
		"user:list",
		"ns="+namespace,
		fmt.Sprintf("actor=%d", actorUserID),
		"actor_role="+string(actorRole),
		fmt.Sprintf("limit=%d", query.Limit),
		fmt.Sprintf("offset=%d", query.Offset),
		"search="+strings.ToLower(strings.TrimSpace(query.SearchTerm)),
		"role="+string(query.Role),
	)
}

func (s *UserService) userListNamespaceKey() string {
	return "users:list:namespace"
}

func (s *UserService) invalidateUserMeCache(ctx context.Context, userID uint64) {
	if !s.cacheEnabled || s.cache == nil {
		return
	}

	cacheKey := fmt.Sprintf("user:me:%d", userID)
	if err := s.cache.Delete(ctx, cacheKey); err != nil {
		s.logger.Error("cache delete failed", "key", cacheKey, "error", err.Error())
	}
}

func (s *UserService) invalidateUserListCache(ctx context.Context) {
	bumpCacheNamespace(ctx, s.cache, s.cacheEnabled, s.userListNamespaceKey(), s.logger)
}
