package service

import (
	"context"
	"testing"
	"time"

	"github.com/example/gue/backend/model"
	jwtpkg "github.com/example/gue/backend/pkg/jwt"
	"github.com/example/gue/backend/repository"
	"github.com/stretchr/testify/require"
	"log/slog"
)

type fakeUserRepo struct {
	byEmail map[string]*model.User
	byID    map[uint64]*model.User
	nextID  uint64
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
