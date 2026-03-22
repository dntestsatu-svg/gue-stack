package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/gue/backend/middleware"
	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mockDashboardUseCase struct {
	overviewFn func(ctx context.Context, userID uint64) (*service.DashboardOverviewResult, error)
	historyFn  func(ctx context.Context, userID uint64, query service.TransactionHistoryQuery) (*service.TransactionHistoryPage, error)
	exportFn   func(ctx context.Context, userID uint64, query service.TransactionHistoryQuery, format string) (*service.TransactionHistoryExport, error)
}

func (m *mockDashboardUseCase) Overview(ctx context.Context, userID uint64) (*service.DashboardOverviewResult, error) {
	if m.overviewFn == nil {
		return nil, nil
	}
	return m.overviewFn(ctx, userID)
}

func (m *mockDashboardUseCase) TransactionHistory(ctx context.Context, userID uint64, query service.TransactionHistoryQuery) (*service.TransactionHistoryPage, error) {
	if m.historyFn == nil {
		return nil, nil
	}
	return m.historyFn(ctx, userID, query)
}

func (m *mockDashboardUseCase) ExportTransactionHistory(ctx context.Context, userID uint64, query service.TransactionHistoryQuery, format string) (*service.TransactionHistoryExport, error) {
	if m.exportFn == nil {
		return nil, nil
	}
	return m.exportFn(ctx, userID, query, format)
}

func withDashboardAuth(userID uint64, role model.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, userID)
		c.Set(middleware.ContextKeyUserRole, role)
		c.Next()
	}
}

func TestDashboardHandlerOverviewAndHistory_TableDriven(t *testing.T) {
	gin.SetMode(gin.TestMode)

	from := "2026-03-20"
	to := "2026-03-21"

	mockUC := &mockDashboardUseCase{
		overviewFn: func(_ context.Context, userID uint64) (*service.DashboardOverviewResult, error) {
			require.Equal(t, uint64(77), userID)
			return &service.DashboardOverviewResult{WindowHours: 12}, nil
		},
		historyFn: func(_ context.Context, userID uint64, query service.TransactionHistoryQuery) (*service.TransactionHistoryPage, error) {
			require.Equal(t, uint64(77), userID)
			require.Equal(t, 50, query.Limit)
			require.Equal(t, 10, query.Offset)
			require.Equal(t, "trx", query.SearchTerm)
			require.NotNil(t, query.From)
			require.NotNil(t, query.To)
			require.Equal(t, from, query.From.UTC().Format("2006-01-02"))
			require.Equal(t, to, query.To.UTC().Format("2006-01-02"))
			return &service.TransactionHistoryPage{
				Items: []service.TransactionHistoryItem{{ID: 1, TokoID: 1, TokoName: "A"}},
				Total: 1,
				Limit: 50,
			}, nil
		},
	}

	h := NewDashboardHandler(mockUC)
	r := gin.New()
	r.GET("/dashboard/overview", withDashboardAuth(77, model.UserRoleDev), h.Overview)
	r.GET("/transactions/history", withDashboardAuth(77, model.UserRoleDev), h.TransactionHistory)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{name: "overview success", path: "/dashboard/overview", expectedStatus: http.StatusOK},
		{name: "history success", path: "/transactions/history?limit=50&offset=10&q=trx&from=2026-03-20&to=2026-03-21", expectedStatus: http.StatusOK},
		{name: "history invalid limit", path: "/transactions/history?limit=invalid", expectedStatus: http.StatusBadRequest},
		{name: "history invalid offset", path: "/transactions/history?offset=invalid", expectedStatus: http.StatusBadRequest},
		{name: "history invalid from date", path: "/transactions/history?from=invalid-date", expectedStatus: http.StatusBadRequest},
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

func TestDashboardHandlerExportHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := &mockDashboardUseCase{
		exportFn: func(_ context.Context, userID uint64, query service.TransactionHistoryQuery, format string) (*service.TransactionHistoryExport, error) {
			require.Equal(t, uint64(77), userID)
			require.Equal(t, "csv", format)
			require.Equal(t, 25, query.Limit)
			return &service.TransactionHistoryExport{
				Content:     []byte("id,toko\n1,Tokoku"),
				ContentType: "text/csv; charset=utf-8",
				FileName:    "transaction-history.csv",
			}, nil
		},
	}

	h := NewDashboardHandler(mockUC)
	r := gin.New()
	r.GET("/transactions/history/export", withDashboardAuth(77, model.UserRoleDev), h.ExportTransactionHistory)

	req := httptest.NewRequest(http.MethodGet, "/transactions/history/export?format=csv&limit=25", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "text/csv; charset=utf-8", w.Header().Get("Content-Type"))
	require.Contains(t, w.Header().Get("Content-Disposition"), "transaction-history.csv")
	require.Contains(t, w.Body.String(), "id,toko")
}

func TestDashboardHandlerOverviewRedactsProjectProfitForNonDev(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := &mockDashboardUseCase{
		overviewFn: func(_ context.Context, _ uint64) (*service.DashboardOverviewResult, error) {
			return &service.DashboardOverviewResult{
				Metrics: service.DashboardMetricsDTO{
					ProjectProfit: 7777,
				},
				UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			}, nil
		},
	}

	h := NewDashboardHandler(mockUC)
	r := gin.New()
	r.GET("/dashboard/overview", withDashboardAuth(88, model.UserRoleSuperAdmin), h.Overview)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/overview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"can_view_project_profit":false`)
	require.Contains(t, w.Body.String(), `"project_profit":0`)
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
