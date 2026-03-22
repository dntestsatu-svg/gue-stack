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

type mockTokoUseCase struct {
	workspaceFn        func(ctx context.Context, userID uint64, actorRole model.UserRole, query service.TokoWorkspaceQuery) (*service.TokoWorkspacePage, error)
	listFn             func(ctx context.Context, userID uint64, actorRole model.UserRole) ([]service.TokoDTO, error)
	createFn           func(ctx context.Context, userID uint64, actorRole model.UserRole, input service.CreateTokoInput) (*service.TokoDTO, error)
	updateFn           func(ctx context.Context, userID uint64, actorRole model.UserRole, tokoID uint64, input service.UpdateTokoInput) (*service.TokoDTO, error)
	regenerateTokenFn  func(ctx context.Context, userID uint64, actorRole model.UserRole, tokoID uint64) (*service.TokoDTO, error)
	listBalancesFn     func(ctx context.Context, userID uint64, actorRole model.UserRole) ([]service.TokoBalanceDTO, error)
	manualSettlementFn func(ctx context.Context, actorRole model.UserRole, tokoID uint64, input service.ManualSettlementInput) (*service.TokoBalanceDTO, error)
}

func (m *mockTokoUseCase) Workspace(ctx context.Context, userID uint64, actorRole model.UserRole, query service.TokoWorkspaceQuery) (*service.TokoWorkspacePage, error) {
	if m.workspaceFn == nil {
		return nil, nil
	}
	return m.workspaceFn(ctx, userID, actorRole, query)
}

func (m *mockTokoUseCase) ListByUser(ctx context.Context, userID uint64, actorRole model.UserRole) ([]service.TokoDTO, error) {
	if m.listFn == nil {
		return nil, nil
	}
	return m.listFn(ctx, userID, actorRole)
}

func (m *mockTokoUseCase) CreateForUser(ctx context.Context, userID uint64, actorRole model.UserRole, input service.CreateTokoInput) (*service.TokoDTO, error) {
	if m.createFn == nil {
		return nil, nil
	}
	return m.createFn(ctx, userID, actorRole, input)
}

func (m *mockTokoUseCase) Update(ctx context.Context, userID uint64, actorRole model.UserRole, tokoID uint64, input service.UpdateTokoInput) (*service.TokoDTO, error) {
	if m.updateFn == nil {
		return nil, nil
	}
	return m.updateFn(ctx, userID, actorRole, tokoID, input)
}

func (m *mockTokoUseCase) RegenerateToken(ctx context.Context, userID uint64, actorRole model.UserRole, tokoID uint64) (*service.TokoDTO, error) {
	if m.regenerateTokenFn == nil {
		return nil, nil
	}
	return m.regenerateTokenFn(ctx, userID, actorRole, tokoID)
}

func (m *mockTokoUseCase) ListBalancesByUser(ctx context.Context, userID uint64, actorRole model.UserRole) ([]service.TokoBalanceDTO, error) {
	if m.listBalancesFn == nil {
		return nil, nil
	}
	return m.listBalancesFn(ctx, userID, actorRole)
}

func (m *mockTokoUseCase) ManualSettlement(ctx context.Context, actorRole model.UserRole, tokoID uint64, input service.ManualSettlementInput) (*service.TokoBalanceDTO, error) {
	if m.manualSettlementFn == nil {
		return nil, nil
	}
	return m.manualSettlementFn(ctx, actorRole, tokoID, input)
}

func withTokoAuthContext(userID uint64, role model.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, userID)
		c.Set(middleware.ContextKeyUserRole, role)
		c.Next()
	}
}

func TestTokoHandlerEndpoints_TableDriven(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := &mockTokoUseCase{
		workspaceFn: func(_ context.Context, userID uint64, actorRole model.UserRole, query service.TokoWorkspaceQuery) (*service.TokoWorkspacePage, error) {
			require.Equal(t, uint64(5), userID)
			require.Equal(t, model.UserRoleAdmin, actorRole)
			require.Equal(t, 10, query.Limit)
			require.Equal(t, 20, query.Offset)
			require.Equal(t, "alpha", query.SearchTerm)
			return &service.TokoWorkspacePage{
				Items: []service.TokoWorkspaceItemDTO{{ID: 1, Name: "Toko A", Token: "abc", Charge: 3, SettlementBalance: 1000, AvailableBalance: 2000}},
				Summary: service.TokoWorkspaceSummaryDTO{
					TotalTokos:            1,
					TotalSettlementAmount: 1000,
					TotalAvailableAmount:  2000,
				},
				Total:   1,
				Limit:   10,
				Offset:  20,
				HasMore: false,
			}, nil
		},
		listFn: func(_ context.Context, userID uint64, actorRole model.UserRole) ([]service.TokoDTO, error) {
			require.Equal(t, uint64(5), userID)
			require.Equal(t, model.UserRoleAdmin, actorRole)
			return []service.TokoDTO{{ID: 1, Name: "Toko A", Token: "abc", Charge: 3}}, nil
		},
		createFn: func(_ context.Context, userID uint64, actorRole model.UserRole, input service.CreateTokoInput) (*service.TokoDTO, error) {
			require.Equal(t, uint64(5), userID)
			require.Equal(t, model.UserRoleAdmin, actorRole)
			require.Equal(t, "Toko Baru", input.Name)
			return &service.TokoDTO{ID: 2, Name: "Toko Baru", Token: "new-token", Charge: 3}, nil
		},
		updateFn: func(_ context.Context, userID uint64, actorRole model.UserRole, tokoID uint64, input service.UpdateTokoInput) (*service.TokoDTO, error) {
			require.Equal(t, uint64(5), userID)
			require.Equal(t, model.UserRoleAdmin, actorRole)
			require.Equal(t, uint64(2), tokoID)
			require.Equal(t, "Toko Update", input.Name)
			return &service.TokoDTO{ID: 2, Name: "Toko Update", Token: "new-token", Charge: 3, CallbackURL: input.CallbackURL}, nil
		},
		regenerateTokenFn: func(_ context.Context, userID uint64, actorRole model.UserRole, tokoID uint64) (*service.TokoDTO, error) {
			require.Equal(t, uint64(5), userID)
			require.Equal(t, model.UserRoleAdmin, actorRole)
			require.Equal(t, uint64(2), tokoID)
			return &service.TokoDTO{ID: 2, Name: "Toko Update", Token: "rotated-token", Charge: 3}, nil
		},
		listBalancesFn: func(_ context.Context, userID uint64, actorRole model.UserRole) ([]service.TokoBalanceDTO, error) {
			require.Equal(t, uint64(5), userID)
			require.Equal(t, model.UserRoleAdmin, actorRole)
			return []service.TokoBalanceDTO{{TokoID: 1, TokoName: "Toko A", SettlementBalance: 1000, AvailableBalance: 1200}}, nil
		},
		manualSettlementFn: func(_ context.Context, actorRole model.UserRole, tokoID uint64, input service.ManualSettlementInput) (*service.TokoBalanceDTO, error) {
			require.Equal(t, model.UserRoleDev, actorRole)
			require.Equal(t, uint64(1), tokoID)
			require.Equal(t, 100.0, input.SettlementBalance)
			return &service.TokoBalanceDTO{
				TokoID:            1,
				TokoName:          "Toko A",
				SettlementBalance: 100,
				AvailableBalance:  90,
			}, nil
		},
	}
	h := NewTokoHandler(mockUC)

	r := gin.New()
	r.GET("/tokos/workspace", withTokoAuthContext(5, model.UserRoleAdmin), h.Workspace)
	r.GET("/tokos", withTokoAuthContext(5, model.UserRoleAdmin), h.List)
	r.POST("/tokos", withTokoAuthContext(5, model.UserRoleAdmin), h.Create)
	r.PATCH("/tokos/:id", withTokoAuthContext(5, model.UserRoleAdmin), h.Update)
	r.POST("/tokos/:id/regenerate-token", withTokoAuthContext(5, model.UserRoleAdmin), h.RegenerateToken)
	r.GET("/tokos/balances", withTokoAuthContext(5, model.UserRoleAdmin), h.ListBalances)
	r.PATCH("/tokos/:id/settlement", withTokoAuthContext(5, model.UserRoleDev), h.ManualSettlement)

	tests := []struct {
		name           string
		method         string
		path           string
		body           map[string]any
		expectedStatus int
	}{
		{name: "workspace", method: http.MethodGet, path: "/tokos/workspace?limit=10&offset=20&q=alpha", expectedStatus: http.StatusOK},
		{name: "list", method: http.MethodGet, path: "/tokos", expectedStatus: http.StatusOK},
		{name: "create", method: http.MethodPost, path: "/tokos", body: map[string]any{"name": "Toko Baru"}, expectedStatus: http.StatusCreated},
		{name: "update", method: http.MethodPatch, path: "/tokos/2", body: map[string]any{"name": "Toko Update", "callback_url": "https://example.com/updated"}, expectedStatus: http.StatusOK},
		{name: "regenerate token", method: http.MethodPost, path: "/tokos/2/regenerate-token", expectedStatus: http.StatusOK},
		{name: "list balances", method: http.MethodGet, path: "/tokos/balances", expectedStatus: http.StatusOK},
		{name: "manual settlement", method: http.MethodPatch, path: "/tokos/1/settlement", body: map[string]any{"settlement_balance": 100}, expectedStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if tt.body != nil {
				bodyBytes, _ = json.Marshal(tt.body)
			}
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader(bodyBytes))
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestTokoHandlerManualSettlementInvalidTokoID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := &mockTokoUseCase{
		manualSettlementFn: func(_ context.Context, _ model.UserRole, _ uint64, _ service.ManualSettlementInput) (*service.TokoBalanceDTO, error) {
			return nil, apperror.New(http.StatusForbidden, "forbidden", nil)
		},
	}
	h := NewTokoHandler(mockUC)
	r := gin.New()
	r.PATCH("/tokos/:id/settlement", withTokoAuthContext(5, model.UserRoleDev), h.ManualSettlement)

	req := httptest.NewRequest(http.MethodPatch, "/tokos/not-a-number/settlement", bytes.NewBufferString(`{"settlement_balance":100}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}
