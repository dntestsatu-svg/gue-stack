package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/example/gue/backend/model"
	jwtpkg "github.com/example/gue/backend/pkg/jwt"
	"github.com/example/gue/backend/pkg/password"
	"github.com/example/gue/backend/queue"
	"github.com/example/gue/backend/repository"
	"github.com/stretchr/testify/require"
	"log/slog"
)

type fakeUserRepo struct {
	byEmail map[string]*model.User
	byID    map[uint64]*model.User
	nextID  uint64

	listPageCalls int
	countCalls    int
}

func (r *fakeUserRepo) Create(_ context.Context, user *model.User) error {
	r.nextID++
	user.ID = r.nextID
	if r.byEmail == nil {
		r.byEmail = map[string]*model.User{}
	}
	if r.byID == nil {
		r.byID = map[uint64]*model.User{}
	}
	r.byEmail[user.Email] = user
	r.byID[user.ID] = user
	return nil
}

func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	if user, ok := r.byEmail[email]; ok {
		return user, nil
	}
	return nil, repository.ErrNotFound
}

func (r *fakeUserRepo) GetByID(_ context.Context, id uint64) (*model.User, error) {
	if user, ok := r.byID[id]; ok {
		return user, nil
	}
	return nil, repository.ErrNotFound
}

func (r *fakeUserRepo) ListByScope(_ context.Context, _ uint64, limit int) ([]model.User, error) {
	if limit <= 0 {
		limit = len(r.byID)
	}
	items := make([]model.User, 0, limit)
	for _, user := range r.byID {
		items = append(items, *user)
		if len(items) >= limit {
			break
		}
	}
	return items, nil
}

func (r *fakeUserRepo) ListPageByScope(_ context.Context, _ uint64, filter repository.UserListFilter) ([]model.User, error) {
	r.listPageCalls++
	if filter.Limit <= 0 {
		filter.Limit = len(r.byID)
	}

	items := make([]model.User, 0, filter.Limit)
	for _, user := range r.byID {
		if filter.Role != "" && user.Role != filter.Role {
			continue
		}
		if filter.SearchTerm != "" {
			search := strings.ToLower(filter.SearchTerm)
			if !strings.Contains(strings.ToLower(user.Name), search) && !strings.Contains(strings.ToLower(user.Email), search) {
				continue
			}
		}
		items = append(items, *user)
	}

	if filter.Offset >= len(items) {
		return []model.User{}, nil
	}

	end := filter.Offset + filter.Limit
	if end > len(items) {
		end = len(items)
	}
	return items[filter.Offset:end], nil
}

func (r *fakeUserRepo) CountByScope(_ context.Context, _ uint64, filter repository.UserListFilter) (uint64, error) {
	r.countCalls++
	var count uint64
	for _, user := range r.byID {
		if filter.Role != "" && user.Role != filter.Role {
			continue
		}
		if filter.SearchTerm != "" {
			search := strings.ToLower(filter.SearchTerm)
			if !strings.Contains(strings.ToLower(user.Name), search) && !strings.Contains(strings.ToLower(user.Email), search) {
				continue
			}
		}
		count++
	}
	return count, nil
}

func (r *fakeUserRepo) IsInScope(_ context.Context, actorUserID uint64, targetUserID uint64) (bool, error) {
	if actorUserID == targetUserID {
		return true, nil
	}
	target, ok := r.byID[targetUserID]
	if !ok {
		return false, nil
	}
	return target.CreatedBy != nil && *target.CreatedBy == actorUserID, nil
}

func (r *fakeUserRepo) UpdateRole(_ context.Context, id uint64, role model.UserRole) error {
	user, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	user.Role = role
	if r.byEmail != nil {
		r.byEmail[user.Email] = user
	}
	return nil
}

func (r *fakeUserRepo) UpdateActive(_ context.Context, id uint64, isActive bool) error {
	user, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	user.IsActive = isActive
	if r.byEmail != nil {
		r.byEmail[user.Email] = user
	}
	return nil
}

func (r *fakeUserRepo) UpdatePassword(_ context.Context, id uint64, passwordHash string) error {
	user, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	user.PasswordHash = passwordHash
	if r.byEmail != nil {
		r.byEmail[user.Email] = user
	}
	return nil
}

func (r *fakeUserRepo) Delete(_ context.Context, id uint64) error {
	user, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	delete(r.byID, id)
	if r.byEmail != nil {
		delete(r.byEmail, user.Email)
	}
	return nil
}

type fakeRefreshStore struct {
	tokens map[string]uint64
}

func (s *fakeRefreshStore) Store(_ context.Context, tokenID string, userID uint64, _ time.Duration) error {
	if s.tokens == nil {
		s.tokens = map[string]uint64{}
	}
	s.tokens[tokenID] = userID
	return nil
}

func (s *fakeRefreshStore) GetUserID(_ context.Context, tokenID string) (uint64, error) {
	if userID, ok := s.tokens[tokenID]; ok {
		return userID, nil
	}
	return 0, repository.ErrNotFound
}

func (s *fakeRefreshStore) Delete(_ context.Context, tokenID string) error {
	delete(s.tokens, tokenID)
	return nil
}

type fakeProducer struct {
	count int
}

func (p *fakeProducer) EnqueueWelcomeEmail(_ context.Context, _, _ string) error {
	p.count++
	return nil
}

func (p *fakeProducer) EnqueueQrisCallback(_ context.Context, _ queue.QrisCallbackTaskPayload) error {
	return nil
}

func (p *fakeProducer) EnqueueTransferCallback(_ context.Context, _ queue.TransferCallbackTaskPayload) error {
	return nil
}

func TestAuthService_RegisterSuccess(t *testing.T) {
	userRepo := &fakeUserRepo{}
	refreshStore := &fakeRefreshStore{}
	producer := &fakeProducer{}
	tm := jwtpkg.NewManager("access-secret", "refresh-secret", 15*time.Minute, 24*time.Hour, "issuer", "aud")
	svc := NewAuthService(userRepo, refreshStore, tm, producer, slog.Default())

	result, err := svc.Register(context.Background(), RegisterInput{
		Name:     "Jane Doe",
		Email:    "jane@example.com",
		Password: "secret123",
	})

	require.NoError(t, err)
	require.NotEmpty(t, result.AccessToken)
	require.NotEmpty(t, result.RefreshToken)
	require.Equal(t, uint64(1), result.User.ID)
	require.Equal(t, model.UserRoleAdmin, result.User.Role)
	require.Equal(t, model.UserRoleAdmin, userRepo.byID[result.User.ID].Role)
	require.Equal(t, 1, producer.count)
}

func TestAuthService_LoginInvalidCredentials(t *testing.T) {
	userRepo := &fakeUserRepo{byEmail: map[string]*model.User{}}
	hashUser := &model.User{
		ID:           1,
		Name:         "Jane",
		Email:        "jane@example.com",
		PasswordHash: "$2a$10$mQ1eQn8rH8j8xM3jV5x0bOCk29Qa58aeGeq/QAdzTZziEtGlUZEM6", // "secret123"
	}
	userRepo.byEmail[hashUser.Email] = hashUser
	userRepo.byID = map[uint64]*model.User{1: hashUser}

	refreshStore := &fakeRefreshStore{}
	tm := jwtpkg.NewManager("access-secret", "refresh-secret", 15*time.Minute, 24*time.Hour, "issuer", "aud")
	svc := NewAuthService(userRepo, refreshStore, tm, nil, slog.Default())

	_, err := svc.Login(context.Background(), LoginInput{
		Email:    "jane@example.com",
		Password: "wrongpassword",
	})

	require.Error(t, err)
}

func TestAuthService_LoginInactiveUserForbidden(t *testing.T) {
	hash, err := password.Hash("secret123")
	require.NoError(t, err)

	userRepo := &fakeUserRepo{byEmail: map[string]*model.User{}}
	hashUser := &model.User{
		ID:           7,
		Name:         "Inactive Jane",
		Email:        "inactive@example.com",
		PasswordHash: hash,
		Role:         model.UserRoleUser,
		IsActive:     false,
	}
	userRepo.byEmail[hashUser.Email] = hashUser
	userRepo.byID = map[uint64]*model.User{7: hashUser}

	refreshStore := &fakeRefreshStore{}
	tm := jwtpkg.NewManager("access-secret", "refresh-secret", 15*time.Minute, 24*time.Hour, "issuer", "aud")
	svc := NewAuthService(userRepo, refreshStore, tm, nil, slog.Default())

	_, err = svc.Login(context.Background(), LoginInput{
		Email:    "inactive@example.com",
		Password: "secret123",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "inactive")
}

func TestAuthService_SessionStatusAuthenticatedWithValidRefreshToken(t *testing.T) {
	refreshStore := &fakeRefreshStore{}
	userRepo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID: map[uint64]*model.User{
			42: {
				ID:       42,
				Name:     "Active Admin",
				Email:    "active@example.com",
				Role:     model.UserRoleAdmin,
				IsActive: true,
			},
		},
	}

	tm := jwtpkg.NewManager("access-secret", "refresh-secret", 15*time.Minute, 24*time.Hour, "issuer", "aud")
	tokenPair, err := tm.GenerateTokenPair(42, "active@example.com", time.Now().UTC())
	require.NoError(t, err)
	require.NoError(t, refreshStore.Store(context.Background(), tokenPair.RefreshID, 42, tm.RefreshTTL()))

	svc := NewAuthService(userRepo, refreshStore, tm, nil, slog.Default())

	status, err := svc.SessionStatus(context.Background(), tokenPair.RefreshToken)

	require.NoError(t, err)
	require.True(t, status.Authenticated)
	require.NotNil(t, status.User)
	require.Equal(t, uint64(42), status.User.ID)
	require.Equal(t, model.UserRoleAdmin, status.User.Role)
}

func TestAuthService_SessionStatusReturnsUnauthenticatedForRevokedToken(t *testing.T) {
	tm := jwtpkg.NewManager("access-secret", "refresh-secret", 15*time.Minute, 24*time.Hour, "issuer", "aud")
	tokenPair, err := tm.GenerateTokenPair(7, "stale@example.com", time.Now().UTC())
	require.NoError(t, err)

	svc := NewAuthService(&fakeUserRepo{}, &fakeRefreshStore{}, tm, nil, slog.Default())

	status, err := svc.SessionStatus(context.Background(), tokenPair.RefreshToken)

	require.NoError(t, err)
	require.False(t, status.Authenticated)
	require.Nil(t, status.User)
}
