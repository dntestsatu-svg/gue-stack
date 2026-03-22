package service

import (
	"context"
	"testing"
	"time"

	"github.com/example/gue/backend/cache"
	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/password"
	"github.com/stretchr/testify/require"
	"log/slog"
)

func TestUserService_CreateForbiddenForUserRole(t *testing.T) {
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID:    map[uint64]*model.User{},
		nextID:  0,
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

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
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

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
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

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
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

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
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

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
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

	page, err := svc.List(context.Background(), 1, model.UserRoleSuperAdmin, UserListQuery{
		Limit: 10,
	})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, model.UserRoleAdmin, page.Items[0].Role)
	require.Equal(t, uint64(1), page.Total)
}

func TestUserService_ListForbiddenForUserRole(t *testing.T) {
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID:    map[uint64]*model.User{},
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

	_, err := svc.List(context.Background(), 1, model.UserRoleUser, UserListQuery{Limit: 10})
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient role")
}

func TestUserService_ListSupportsSearchAndRoleFilter(t *testing.T) {
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID: map[uint64]*model.User{
			1: {ID: 1, Name: "Admin One", Email: "admin.one@example.com", Role: model.UserRoleAdmin, IsActive: true},
			2: {ID: 2, Name: "Admin Two", Email: "admin.two@example.com", Role: model.UserRoleAdmin, IsActive: true},
			3: {ID: 3, Name: "User Three", Email: "user.three@example.com", Role: model.UserRoleUser, IsActive: true},
		},
		nextID: 3,
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

	page, err := svc.List(context.Background(), 1, model.UserRoleSuperAdmin, UserListQuery{
		Limit:      10,
		Offset:     0,
		SearchTerm: "admin",
		Role:       model.UserRoleAdmin,
	})

	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	require.Equal(t, uint64(2), page.Total)
	require.False(t, page.HasMore)
}

func TestUserService_SuperAdminCannotCreateSuperAdmin(t *testing.T) {
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID:    map[uint64]*model.User{},
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

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
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

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
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

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

func TestUserService_ListUsesCacheForPaginatedResult(t *testing.T) {
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID: map[uint64]*model.User{
			1: {ID: 1, Name: "Admin One", Email: "admin.one@example.com", Role: model.UserRoleAdmin, IsActive: true},
			2: {ID: 2, Name: "Admin Two", Email: "admin.two@example.com", Role: model.UserRoleAdmin, IsActive: true},
		},
		nextID: 2,
	}
	cacheStore := newFakeCacheStore()
	svc := NewUserService(repo, cacheStore, true, time.Minute, time.Minute, nil, slog.Default())
	query := UserListQuery{Limit: 10, Offset: 0, SearchTerm: "admin", Role: model.UserRoleAdmin}

	firstPage, err := svc.List(context.Background(), 10, model.UserRoleSuperAdmin, query)
	require.NoError(t, err)
	require.Len(t, firstPage.Items, 2)
	require.Equal(t, 1, repo.countCalls)
	require.Equal(t, 1, repo.listPageCalls)

	secondPage, err := svc.List(context.Background(), 10, model.UserRoleSuperAdmin, query)
	require.NoError(t, err)
	require.Len(t, secondPage.Items, 2)
	require.Equal(t, 1, repo.countCalls)
	require.Equal(t, 1, repo.listPageCalls)
}

func TestUserService_CreateInvalidatesPaginatedListNamespace(t *testing.T) {
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID: map[uint64]*model.User{
			1: {ID: 1, Name: "Admin One", Email: "admin.one@example.com", Role: model.UserRoleAdmin, IsActive: true},
		},
		nextID: 1,
	}
	cacheStore := newFakeCacheStore()
	svc := NewUserService(repo, cacheStore, true, time.Minute, time.Minute, nil, slog.Default())
	query := UserListQuery{Limit: 10, Offset: 0}

	_, err := svc.List(context.Background(), 10, model.UserRoleSuperAdmin, query)
	require.NoError(t, err)
	require.Equal(t, 1, repo.countCalls)
	require.Equal(t, 1, repo.listPageCalls)

	_, err = svc.Create(context.Background(), 10, model.UserRoleSuperAdmin, CreateUserInput{
		Name:     "New User",
		Email:    "new.user@example.com",
		Password: "secret123",
	})
	require.NoError(t, err)

	page, err := svc.List(context.Background(), 10, model.UserRoleSuperAdmin, query)
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	require.Equal(t, 2, repo.countCalls)
	require.Equal(t, 2, repo.listPageCalls)
}

func TestUserService_UpdateActiveAllowedForAdminManagedUser(t *testing.T) {
	creatorID := uint64(10)
	target := &model.User{
		ID:           12,
		Name:         "Managed User",
		Email:        "managed.user@example.com",
		PasswordHash: "hash",
		Role:         model.UserRoleUser,
		IsActive:     true,
		CreatedBy:    &creatorID,
	}
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{target.Email: target},
		byID:    map[uint64]*model.User{target.ID: target},
		nextID:  target.ID,
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

	updated, err := svc.UpdateActive(context.Background(), creatorID, model.UserRoleAdmin, target.ID, UpdateUserActiveInput{
		IsActive: boolPtr(false),
	})

	require.NoError(t, err)
	require.False(t, updated.IsActive)
	require.False(t, repo.byID[target.ID].IsActive)
}

func TestUserService_UpdateActiveRejectsSelfUpdate(t *testing.T) {
	actorID := uint64(10)
	user := &model.User{
		ID:           actorID,
		Name:         "Admin User",
		Email:        "admin.user@example.com",
		PasswordHash: "hash",
		Role:         model.UserRoleAdmin,
		IsActive:     true,
	}
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{user.Email: user},
		byID:    map[uint64]*model.User{user.ID: user},
		nextID:  user.ID,
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

	_, err := svc.UpdateActive(context.Background(), actorID, model.UserRoleAdmin, actorID, UpdateUserActiveInput{
		IsActive: boolPtr(false),
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "own active status")
}

func TestUserService_DeleteRemovesManagedUser(t *testing.T) {
	creatorID := uint64(20)
	target := &model.User{
		ID:           21,
		Name:         "Staff User",
		Email:        "staff.user@example.com",
		PasswordHash: "hash",
		Role:         model.UserRoleUser,
		IsActive:     true,
		CreatedBy:    &creatorID,
	}
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{target.Email: target},
		byID:    map[uint64]*model.User{target.ID: target},
		nextID:  target.ID,
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

	err := svc.Delete(context.Background(), creatorID, model.UserRoleAdmin, target.ID)

	require.NoError(t, err)
	_, exists := repo.byID[target.ID]
	require.False(t, exists)
}

func TestUserService_DeleteRejectsSelfDelete(t *testing.T) {
	actorID := uint64(30)
	user := &model.User{
		ID:           actorID,
		Name:         "Developer",
		Email:        "dev@example.com",
		PasswordHash: "hash",
		Role:         model.UserRoleDev,
		IsActive:     true,
	}
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{user.Email: user},
		byID:    map[uint64]*model.User{user.ID: user},
		nextID:  user.ID,
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

	err := svc.Delete(context.Background(), actorID, model.UserRoleDev, actorID)

	require.Error(t, err)
	require.Contains(t, err.Error(), "own account")
}

func TestUserService_ChangePasswordSuccess(t *testing.T) {
	hash, err := password.Hash("current123")
	require.NoError(t, err)

	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID: map[uint64]*model.User{
			10: {ID: 10, Name: "Alex", Email: "alex@example.com", PasswordHash: hash, Role: model.UserRoleAdmin, IsActive: true},
		},
		nextID: 10,
	}
	repo.byEmail["alex@example.com"] = repo.byID[10]
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

	err = svc.ChangePassword(context.Background(), 10, ChangePasswordInput{
		CurrentPassword: "current123",
		NewPassword:     "newsecret123",
		ConfirmPassword: "newsecret123",
	})
	require.NoError(t, err)
	require.NoError(t, password.Compare(repo.byID[10].PasswordHash, "newsecret123"))
}

func TestUserService_ChangePasswordRejectsWrongCurrentPassword(t *testing.T) {
	hash, err := password.Hash("current123")
	require.NoError(t, err)

	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID: map[uint64]*model.User{
			10: {ID: 10, Name: "Alex", Email: "alex@example.com", PasswordHash: hash, Role: model.UserRoleAdmin, IsActive: true},
		},
		nextID: 10,
	}
	repo.byEmail["alex@example.com"] = repo.byID[10]
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

	err = svc.ChangePassword(context.Background(), 10, ChangePasswordInput{
		CurrentPassword: "wrongpass123",
		NewPassword:     "newsecret123",
		ConfirmPassword: "newsecret123",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "current password is invalid")
}

func TestUserService_ChangePasswordRejectsMismatchedConfirmation(t *testing.T) {
	hash, err := password.Hash("current123")
	require.NoError(t, err)

	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID: map[uint64]*model.User{
			10: {ID: 10, Name: "Alex", Email: "alex@example.com", PasswordHash: hash, Role: model.UserRoleAdmin, IsActive: true},
		},
		nextID: 10,
	}
	repo.byEmail["alex@example.com"] = repo.byID[10]
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

	err = svc.ChangePassword(context.Background(), 10, ChangePasswordInput{
		CurrentPassword: "current123",
		NewPassword:     "newsecret123",
		ConfirmPassword: "different123",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "confirmation")
}

func TestUserService_ChangePasswordFailsWhenUserMissing(t *testing.T) {
	repo := &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byID:    map[uint64]*model.User{},
	}
	svc := NewUserService(repo, cache.NewNoopCache(), false, time.Minute, time.Minute, nil, slog.Default())

	err := svc.ChangePassword(context.Background(), 999, ChangePasswordInput{
		CurrentPassword: "current123",
		NewPassword:     "newsecret123",
		ConfirmPassword: "newsecret123",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "user not found")
}

func boolPtr(value bool) *bool {
	return &value
}
