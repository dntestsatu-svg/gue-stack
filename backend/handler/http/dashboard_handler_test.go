package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/gue/backend/middleware"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mockDashboardUseCase struct {
	overviewFn func(ctx context.Context, userID uint64) (*service.DashboardOverviewResult, error)
	historyFn  func(ctx context.Context, userID uint64, limit int) ([]service.TransactionHistoryItem, error)
}

func (m *mockDashboardUseCase) Overview(ctx context.Context, userID uint64) (*service.DashboardOverviewResult, error) {
	if m.overviewFn == nil {
		return nil, nil
	}
	return m.overviewFn(ctx, userID)
}

func (m *mockDashboardUseCase) TransactionHistory(ctx context.Context, userID uint64, limit int) ([]service.TransactionHistoryItem, error) {
	if m.historyFn == nil {
		return nil, nil
	}
	return m.historyFn(ctx, userID, limit)
}

func withDashboardUserID(userID uint64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, userID)
		c.Next()
	}
}

func TestDashboardHandlerOverviewAndHistory_TableDriven(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := &mockDashboardUseCase{
		overviewFn: func(_ context.Context, userID uint64) (*service.DashboardOverviewResult, error) {
			require.Equal(t, uint64(77), userID)
			return &service.DashboardOverviewResult{WindowHours: 12}, nil
		},
		historyFn: func(_ context.Context, userID uint64, limit int) ([]service.TransactionHistoryItem, error) {
			require.Equal(t, uint64(77), userID)
			require.Equal(t, 50, limit)
			return []service.TransactionHistoryItem{{ID: 1, TokoID: 1, TokoName: "A"}}, nil
		},
	}

	h := NewDashboardHandler(mockUC)
	r := gin.New()
	r.GET("/dashboard/overview", withDashboardUserID(77), h.Overview)
	r.GET("/transactions/history", withDashboardUserID(77), h.TransactionHistory)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{name: "overview success", path: "/dashboard/overview", expectedStatus: http.StatusOK},
		{name: "history success", path: "/transactions/history?limit=50", expectedStatus: http.StatusOK},
		{name: "history invalid limit", path: "/transactions/history?limit=invalid", expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestDashboardHandlerOverviewUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := &mockDashboardUseCase{
		overviewFn: func(_ context.Context, _ uint64) (*service.DashboardOverviewResult, error) {
			return nil, apperror.New(http.StatusInternalServerError, "unexpected", nil)
		},
	}
	h := NewDashboardHandler(mockUC)
	r := gin.New()
	r.GET("/dashboard/overview", h.Overview)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/overview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}
