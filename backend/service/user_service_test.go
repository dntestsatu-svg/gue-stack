package service

import (
	"context"
	"testing"
	"time"

	"github.com/example/gue/backend/cache"
	"github.com/example/gue/backend/model"
	"github.com/stretchr/testify/require"
	"log/slog"
)

func TestUserService_CreateForbiddenForUserRole(t *testing.T) {
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID:    map[uint64]*model.User{},
		nextID:  0,
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, nil, slog.Default())

	_, err := svc.Create(context.Background(), model.UserRoleUser, CreateUserInput{
		Name:     "John User",
		Email:    "john.user@example.com",
		Password: "secret123",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient role")
}

func TestUserService_CreateByAdminDefaultsToUserRole(t *testing.T) {
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID:    map[uint64]*model.User{},
		nextID:  0,
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, nil, slog.Default())

	created, err := svc.Create(context.Background(), model.UserRoleAdmin, CreateUserInput{
		Name:     "New Member",
		Email:    "member@example.com",
		Password: "secret123",
	})

	require.NoError(t, err)
	require.Equal(t, model.UserRoleUser, created.Role)
	require.True(t, created.IsActive)
}

func TestUserService_UpdateRoleOnlyForDevOrSuperAdmin(t *testing.T) {
	target := &model.User{
		ID:           10,
		Name:         "Target User",
		Email:        "target@example.com",
		PasswordHash: "hash",
		Role:         model.UserRoleUser,
		IsActive:     true,
	}
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{target.Email: target},
		byID:    map[uint64]*model.User{target.ID: target},
		nextID:  10,
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, nil, slog.Default())

	_, err := svc.UpdateRole(context.Background(), model.UserRoleAdmin, target.ID, UpdateUserRoleInput{Role: string(model.UserRoleSuperAdmin)})
	require.Error(t, err)

	updated, err := svc.UpdateRole(context.Background(), model.UserRoleDev, target.ID, UpdateUserRoleInput{Role: string(model.UserRoleAdmin)})
	require.NoError(t, err)
	require.Equal(t, model.UserRoleAdmin, updated.Role)
}

func TestUserService_CreateRejectsDevRoleAssignment(t *testing.T) {
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID:    map[uint64]*model.User{},
		nextID:  0,
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, nil, slog.Default())

	role := string(model.UserRoleDev)
	_, err := svc.Create(context.Background(), model.UserRoleSuperAdmin, CreateUserInput{
		Name:     "Reserved Dev",
		Email:    "reserved.dev@example.com",
		Password: "secret123",
		Role:     &role,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "reserved for bootstrap")
}

func TestUserService_UpdateRoleRejectsDevRoleAssignment(t *testing.T) {
	target := &model.User{
		ID:           11,
		Name:         "Target User",
		Email:        "target2@example.com",
		PasswordHash: "hash",
		Role:         model.UserRoleUser,
		IsActive:     true,
	}
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{target.Email: target},
		byID:    map[uint64]*model.User{target.ID: target},
		nextID:  11,
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, nil, slog.Default())

	_, err := svc.UpdateRole(context.Background(), model.UserRoleSuperAdmin, target.ID, UpdateUserRoleInput{Role: string(model.UserRoleDev)})
	require.Error(t, err)
	require.Contains(t, err.Error(), "reserved for bootstrap")
}
