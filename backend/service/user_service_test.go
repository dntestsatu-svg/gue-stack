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

	_, err := svc.Create(context.Background(), 10, model.UserRoleUser, CreateUserInput{
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

	created, err := svc.Create(context.Background(), 10, model.UserRoleAdmin, CreateUserInput{
		Name:     "New Member",
		Email:    "member@example.com",
		Password: "secret123",
	})

	require.NoError(t, err)
	require.Equal(t, model.UserRoleUser, created.Role)
	require.True(t, created.IsActive)
	require.NotNil(t, repo.byID[created.ID].CreatedBy)
	require.Equal(t, uint64(10), *repo.byID[created.ID].CreatedBy)
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

	_, err := svc.UpdateRole(context.Background(), 10, model.UserRoleAdmin, target.ID, UpdateUserRoleInput{Role: string(model.UserRoleSuperAdmin)})
	require.Error(t, err)

	updated, err := svc.UpdateRole(context.Background(), 10, model.UserRoleDev, target.ID, UpdateUserRoleInput{Role: string(model.UserRoleAdmin)})
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
	_, err := svc.Create(context.Background(), 10, model.UserRoleSuperAdmin, CreateUserInput{
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

	_, err := svc.UpdateRole(context.Background(), 10, model.UserRoleSuperAdmin, target.ID, UpdateUserRoleInput{Role: string(model.UserRoleDev)})
	require.Error(t, err)
	require.Contains(t, err.Error(), "reserved for bootstrap")
}

func TestUserService_ListAllowedForPrivilegedRoles(t *testing.T) {
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID: map[uint64]*model.User{
			1: {
				ID:       1,
				Name:     "Admin",
				Email:    "admin@example.com",
				Role:     model.UserRoleAdmin,
				IsActive: true,
			},
		},
		nextID: 1,
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, nil, slog.Default())

	items, err := svc.List(context.Background(), 1, model.UserRoleSuperAdmin, 10)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, model.UserRoleAdmin, items[0].Role)
}

func TestUserService_ListForbiddenForUserRole(t *testing.T) {
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID:    map[uint64]*model.User{},
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, nil, slog.Default())

	_, err := svc.List(context.Background(), 1, model.UserRoleUser, 10)
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient role")
}

func TestUserService_SuperAdminCannotCreateSuperAdmin(t *testing.T) {
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID:    map[uint64]*model.User{},
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, nil, slog.Default())

	role := string(model.UserRoleSuperAdmin)
	_, err := svc.Create(context.Background(), 11, model.UserRoleSuperAdmin, CreateUserInput{
		Name:     "Another Superadmin",
		Email:    "another.superadmin@example.com",
		Password: "secret123",
		Role:     &role,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient role")
}

func TestUserService_AdminCannotCreateAdmin(t *testing.T) {
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID:    map[uint64]*model.User{},
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, nil, slog.Default())

	role := string(model.UserRoleAdmin)
	_, err := svc.Create(context.Background(), 12, model.UserRoleAdmin, CreateUserInput{
		Name:     "Admin Candidate",
		Email:    "admin.candidate@example.com",
		Password: "secret123",
		Role:     &role,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient role")
}

func TestUserService_DevCanCreateSuperAdmin(t *testing.T) {
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID:    map[uint64]*model.User{},
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, nil, slog.Default())

	role := string(model.UserRoleSuperAdmin)
	created, err := svc.Create(context.Background(), 1, model.UserRoleDev, CreateUserInput{
		Name:     "Superadmin Team",
		Email:    "superadmin.team@example.com",
		Password: "secret123",
		Role:     &role,
	})

	require.NoError(t, err)
	require.Equal(t, model.UserRoleSuperAdmin, created.Role)
}
