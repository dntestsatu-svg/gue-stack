package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/gue/backend/middleware"
	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mockUserUseCase struct {
	meFn         func(ctx context.Context, userID uint64) (*service.UserDTO, error)
	listFn       func(ctx context.Context, actorUserID uint64, actorRole model.UserRole, limit int) ([]service.UserDTO, error)
	createFn     func(ctx context.Context, actorUserID uint64, actorRole model.UserRole, input service.CreateUserInput) (*service.UserDTO, error)
	updateRoleFn func(ctx context.Context, actorUserID uint64, actorRole model.UserRole, targetUserID uint64, input service.UpdateUserRoleInput) (*service.UserDTO, error)
}

func (m *mockUserUseCase) Me(ctx context.Context, userID uint64) (*service.UserDTO, error) {
	if m.meFn == nil {
		return nil, nil
	}
	return m.meFn(ctx, userID)
}

func (m *mockUserUseCase) List(ctx context.Context, actorUserID uint64, actorRole model.UserRole, limit int) ([]service.UserDTO, error) {
	if m.listFn == nil {
		return nil, nil
	}
	return m.listFn(ctx, actorUserID, actorRole, limit)
}

func (m *mockUserUseCase) Create(ctx context.Context, actorUserID uint64, actorRole model.UserRole, input service.CreateUserInput) (*service.UserDTO, error) {
	if m.createFn == nil {
		return nil, nil
	}
	return m.createFn(ctx, actorUserID, actorRole, input)
}

func (m *mockUserUseCase) UpdateRole(ctx context.Context, actorUserID uint64, actorRole model.UserRole, targetUserID uint64, input service.UpdateUserRoleInput) (*service.UserDTO, error) {
	if m.updateRoleFn == nil {
		return nil, nil
	}
	return m.updateRoleFn(ctx, actorUserID, actorRole, targetUserID, input)
}

func withAuthContext(userID uint64, role model.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, userID)
		c.Set(middleware.ContextKeyUserRole, role)
		c.Next()
	}
}

func TestUserHandlerMe_TableDriven(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		withContext    bool
		meResult       *service.UserDTO
		meErr          error
		expectedStatus int
	}{
		{
			name:        "success",
			withContext: true,
			meResult: &service.UserDTO{
				ID:       10,
				Name:     "Alex",
				Email:    "alex@example.com",
				Role:     model.UserRoleAdmin,
				IsActive: true,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing auth context",
			withContext:    false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "service app error",
			withContext:    true,
			meErr:          apperror.New(http.StatusNotFound, "user not found", nil),
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &mockUserUseCase{
				meFn: func(_ context.Context, _ uint64) (*service.UserDTO, error) {
					return tt.meResult, tt.meErr
				},
			}
			h := NewUserHandler(mockUC)
			r := gin.New()
			if tt.withContext {
				r.GET("/me", withAuthContext(10, model.UserRoleAdmin), h.Me)
			} else {
				r.GET("/me", h.Me)
			}

			req := httptest.NewRequest(http.MethodGet, "/me", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestUserHandlerCreate_AndUpdateRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := &mockUserUseCase{
		createFn: func(_ context.Context, actorUserID uint64, actorRole model.UserRole, input service.CreateUserInput) (*service.UserDTO, error) {
			require.Equal(t, uint64(1), actorUserID)
			require.Equal(t, model.UserRoleAdmin, actorRole)
			require.Equal(t, "new@example.com", input.Email)
			return &service.UserDTO{
				ID:       22,
				Name:     "New User",
				Email:    "new@example.com",
				Role:     model.UserRoleUser,
				IsActive: true,
			}, nil
		},
		updateRoleFn: func(_ context.Context, actorUserID uint64, actorRole model.UserRole, targetUserID uint64, input service.UpdateUserRoleInput) (*service.UserDTO, error) {
			require.Equal(t, uint64(1), actorUserID)
			require.Equal(t, model.UserRoleSuperAdmin, actorRole)
			require.Equal(t, uint64(22), targetUserID)
			require.Equal(t, "admin", input.Role)
			return &service.UserDTO{
				ID:       22,
				Name:     "New User",
				Email:    "new@example.com",
				Role:     model.UserRoleAdmin,
				IsActive: true,
			}, nil
		},
	}

	h := NewUserHandler(mockUC)
	r := gin.New()
	r.POST("/users", withAuthContext(1, model.UserRoleAdmin), h.Create)
	r.PATCH("/users/:id/role", withAuthContext(1, model.UserRoleSuperAdmin), h.UpdateRole)

	createPayload := map[string]any{
		"name":      "New User",
		"email":     "new@example.com",
		"password":  "secret123",
		"role":      "user",
		"is_active": true,
	}
	createBody, _ := json.Marshal(createPayload)
	createReq := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	r.ServeHTTP(createRes, createReq)
	require.Equal(t, http.StatusCreated, createRes.Code)

	updatePayload := map[string]any{"role": "admin"}
	updateBody, _ := json.Marshal(updatePayload)
	updateReq := httptest.NewRequest(http.MethodPatch, "/users/22/role", bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRes := httptest.NewRecorder()
	r.ServeHTTP(updateRes, updateReq)
	require.Equal(t, http.StatusOK, updateRes.Code)
}

func TestUserHandlerList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := &mockUserUseCase{
		listFn: func(_ context.Context, actorUserID uint64, actorRole model.UserRole, limit int) ([]service.UserDTO, error) {
			require.Equal(t, uint64(1), actorUserID)
			require.Equal(t, model.UserRoleAdmin, actorRole)
			require.Equal(t, 25, limit)
			return []service.UserDTO{
				{
					ID:       1,
					Name:     "Dev",
					Email:    "dev@gue.local",
					Role:     model.UserRoleDev,
					IsActive: true,
				},
			}, nil
		},
	}

	h := NewUserHandler(mockUC)
	r := gin.New()
	r.GET("/users", withAuthContext(1, model.UserRoleAdmin), h.List)

	req := httptest.NewRequest(http.MethodGet, "/users?limit=25", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}
