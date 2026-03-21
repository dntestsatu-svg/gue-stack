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
	listFn             func(ctx context.Context, userID uint64) ([]service.TokoDTO, error)
	createFn           func(ctx context.Context, userID uint64, input service.CreateTokoInput) (*service.TokoDTO, error)
	listBalancesFn     func(ctx context.Context, userID uint64) ([]service.TokoBalanceDTO, error)
	manualSettlementFn func(ctx context.Context, actorRole model.UserRole, tokoID uint64, input service.ManualSettlementInput) (*service.TokoBalanceDTO, error)
}

func (m *mockTokoUseCase) ListByUser(ctx context.Context, userID uint64) ([]service.TokoDTO, error) {
	if m.listFn == nil {
		return nil, nil
	}
	return m.listFn(ctx, userID)
}

func (m *mockTokoUseCase) CreateForUser(ctx context.Context, userID uint64, input service.CreateTokoInput) (*service.TokoDTO, error) {
	if m.createFn == nil {
		return nil, nil
	}
	return m.createFn(ctx, userID, input)
}

func (m *mockTokoUseCase) ListBalancesByUser(ctx context.Context, userID uint64) ([]service.TokoBalanceDTO, error) {
	if m.listBalancesFn == nil {
		return nil, nil
	}
	return m.listBalancesFn(ctx, userID)
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
		listFn: func(_ context.Context, userID uint64) ([]service.TokoDTO, error) {
			require.Equal(t, uint64(5), userID)
			return []service.TokoDTO{{ID: 1, Name: "Toko A", Token: "abc", Charge: 3}}, nil
		},
		createFn: func(_ context.Context, userID uint64, input service.CreateTokoInput) (*service.TokoDTO, error) {
			require.Equal(t, uint64(5), userID)
			require.Equal(t, "Toko Baru", input.Name)
			return &service.TokoDTO{ID: 2, Name: "Toko Baru", Token: "new-token", Charge: 3}, nil
		},
		listBalancesFn: func(_ context.Context, userID uint64) ([]service.TokoBalanceDTO, error) {
			require.Equal(t, uint64(5), userID)
			return []service.TokoBalanceDTO{{TokoID: 1, TokoName: "Toko A", SettlementBalance: 1000, AvailableBalance: 1200}}, nil
		},
		manualSettlementFn: func(_ context.Context, actorRole model.UserRole, tokoID uint64, input service.ManualSettlementInput) (*service.TokoBalanceDTO, error) {
			require.Equal(t, model.UserRoleDev, actorRole)
			require.Equal(t, uint64(1), tokoID)
			require.Equal(t, 100.0, input.SettlementBalance)
			require.Equal(t, 90.0, input.AvailableBalance)
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
	r.GET("/tokos", withTokoAuthContext(5, model.UserRoleAdmin), h.List)
	r.POST("/tokos", withTokoAuthContext(5, model.UserRoleAdmin), h.Create)
	r.GET("/tokos/balances", withTokoAuthContext(5, model.UserRoleAdmin), h.ListBalances)
	r.PATCH("/tokos/:id/settlement", withTokoAuthContext(5, model.UserRoleDev), h.ManualSettlement)

	tests := []struct {
		name           string
		method         string
		path           string
		body           map[string]any
		expectedStatus int
	}{
		{name: "list", method: http.MethodGet, path: "/tokos", expectedStatus: http.StatusOK},
		{name: "create", method: http.MethodPost, path: "/tokos", body: map[string]any{"name": "Toko Baru"}, expectedStatus: http.StatusCreated},
		{name: "list balances", method: http.MethodGet, path: "/tokos/balances", expectedStatus: http.StatusOK},
		{name: "manual settlement", method: http.MethodPatch, path: "/tokos/1/settlement", body: map[string]any{"settlement_balance": 100, "available_balance": 90}, expectedStatus: http.StatusOK},
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

	req := httptest.NewRequest(http.MethodPatch, "/tokos/not-a-number/settlement", bytes.NewBufferString(`{"settlement_balance":100,"available_balance":90}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}
