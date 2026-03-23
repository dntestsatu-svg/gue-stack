package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/example/gue/backend/cache"
	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/paymentgateway"
	"github.com/example/gue/backend/repository"
	"github.com/stretchr/testify/require"
)

type fakeDashboardTransactionRepo struct {
	metrics    repository.DashboardMetrics
	series     []repository.DashboardStatusSeriesPoint
	history    []repository.TransactionHistoryRecord
	lastFilter repository.TransactionHistoryFilter
	metricsHit int
	seriesHit  int
	successHit int
}

func (f *fakeDashboardTransactionRepo) Create(_ context.Context, _ *model.Transaction) error {
	return nil
}

func (f *fakeDashboardTransactionRepo) GetByReference(_ context.Context, _ string) (*model.Transaction, error) {
	return nil, repository.ErrNotFound
}

func (f *fakeDashboardTransactionRepo) GetByReferenceAndToko(_ context.Context, _ string, _ uint64) (*model.Transaction, error) {
	return nil, repository.ErrNotFound
}

func (f *fakeDashboardTransactionRepo) UpdateStatusByReference(_ context.Context, _ string, _ model.TransactionStatus) error {
	return nil
}

func (f *fakeDashboardTransactionRepo) UpdateStatusByReferenceAndToko(_ context.Context, _ string, _ uint64, _ model.TransactionStatus) error {
	return nil
}

func (f *fakeDashboardTransactionRepo) UpdateSettlementByID(_ context.Context, _ uint64, _ model.TransactionStatus, _ uint64, _ uint64) error {
	return nil
}

func (f *fakeDashboardTransactionRepo) UpdateSettlementIfPending(_ context.Context, _ uint64, _ model.TransactionStatus, _ uint64, _ uint64) (bool, error) {
	return false, nil
}

func (f *fakeDashboardTransactionRepo) FinalizeDepositSuccessByID(_ context.Context, _ uint64, _ uint64, _ uint64, _ uint64) (bool, error) {
	return false, nil
}

func (f *fakeDashboardTransactionRepo) CreatePendingWithdrawAndReserveSettlement(_ context.Context, _ *model.Transaction) error {
	return nil
}

func (f *fakeDashboardTransactionRepo) FinalizeWithdrawIfPending(_ context.Context, _ uint64, _ model.TransactionStatus) (bool, error) {
	return false, nil
}

func (f *fakeDashboardTransactionRepo) ListPendingExpiryCandidates(_ context.Context, _ time.Time, _ int) ([]repository.PendingExpiryCandidate, error) {
	return nil, nil
}

func (f *fakeDashboardTransactionRepo) GetDashboardMetricsByUser(_ context.Context, _ uint64, _ time.Time) (*repository.DashboardMetrics, error) {
	f.metricsHit++
	value := f.metrics
	return &value, nil
}

func (f *fakeDashboardTransactionRepo) GetHourlyStatusCountsByUser(_ context.Context, _ uint64, _ time.Time) ([]repository.DashboardStatusSeriesPoint, error) {
	f.seriesHit++
	return f.series, nil
}

func (f *fakeDashboardTransactionRepo) ListRecentByUser(_ context.Context, _ uint64, filter repository.TransactionHistoryFilter) ([]repository.TransactionHistoryRecord, error) {
	f.lastFilter = filter
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	if filter.Offset >= len(f.history) {
		return []repository.TransactionHistoryRecord{}, nil
	}
	end := filter.Offset + filter.Limit
	if end > len(f.history) {
		end = len(f.history)
	}
	return f.history[filter.Offset:end], nil
}

func (f *fakeDashboardTransactionRepo) ListRecentSuccessByUser(_ context.Context, _ uint64, limit int) ([]repository.TransactionHistoryRecord, error) {
	f.successHit++
	result := make([]repository.TransactionHistoryRecord, 0, limit)
	for _, item := range f.history {
		if item.Status != model.TransactionStatusSuccess {
			continue
		}
		result = append(result, item)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (f *fakeDashboardTransactionRepo) CountHistoryByUser(_ context.Context, _ uint64, _ repository.TransactionHistoryFilter) (uint64, error) {
	return uint64(len(f.history)), nil
}

type fakeDashboardGatewayClient struct {
	balance *paymentgateway.GetBalanceResponse
	err     error
	calls   int
}

func (f *fakeDashboardGatewayClient) Generate(_ context.Context, _ paymentgateway.GenerateRequest) (*paymentgateway.GenerateResponse, error) {
	return nil, nil
}

func (f *fakeDashboardGatewayClient) CheckStatusV2(_ context.Context, _ string, _ paymentgateway.CheckStatusRequest) (*paymentgateway.CheckStatusResponse, error) {
	return nil, nil
}

func (f *fakeDashboardGatewayClient) InquiryTransfer(_ context.Context, _ paymentgateway.InquiryTransferRequest) (*paymentgateway.InquiryTransferResponse, error) {
	return nil, nil
}

func (f *fakeDashboardGatewayClient) TransferFund(_ context.Context, _ paymentgateway.TransferFundRequest) (*paymentgateway.TransferFundResponse, error) {
	return nil, nil
}

func (f *fakeDashboardGatewayClient) CheckTransferStatus(_ context.Context, _ string, _ paymentgateway.CheckTransferStatusRequest) (*paymentgateway.CheckTransferStatusResponse, error) {
	return nil, nil
}

func (f *fakeDashboardGatewayClient) GetBalance(_ context.Context, _ string, _ paymentgateway.GetBalanceRequest) (*paymentgateway.GetBalanceResponse, error) {
	f.calls++
	return f.balance, f.err
}

type fakeDashboardCache struct {
	items map[string][]byte
	ttls  map[string]time.Duration
	fail  bool
}

func newFakeDashboardCache() *fakeDashboardCache {
	return &fakeDashboardCache{
		items: make(map[string][]byte),
		ttls:  make(map[string]time.Duration),
	}
}

func (f *fakeDashboardCache) Get(_ context.Context, key string) ([]byte, error) {
	if f.fail {
		return nil, assertiveCacheErr("cache unavailable")
	}
	value, ok := f.items[key]
	if !ok {
		return nil, cache.ErrCacheMiss
	}
	return value, nil
}

func (f *fakeDashboardCache) Set(_ context.Context, key string, value any, ttl time.Duration) error {
	if f.fail {
		return assertiveCacheErr("cache unavailable")
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f.items[key] = payload
	f.ttls[key] = ttl
	return nil
}

func (f *fakeDashboardCache) Delete(_ context.Context, key string) error {
	if f.fail {
		return assertiveCacheErr("cache unavailable")
	}
	delete(f.items, key)
	delete(f.ttls, key)
	return nil
}

type assertiveCacheErr string

func (e assertiveCacheErr) Error() string {
	return string(e)
}

func TestDashboardServiceOverviewBuildsSeriesAndMetrics(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Hour)
	player := "player-1"
	reference := "trx-1"
	repo := &fakeDashboardTransactionRepo{
		metrics: repository.DashboardMetrics{
			TotalCount:            10,
			SuccessCount:          7,
			PendingCount:          2,
			FailedCount:           1,
			SuccessDepositAmount:  150000,
			SuccessWithdrawAmount: 30000,
			TotalPlatformFee:      4500,
		},
		series: []repository.DashboardStatusSeriesPoint{
			{Bucket: now.Add(-1 * time.Hour), SuccessCount: 3, FailedCount: 1},
		},
		history: []repository.TransactionHistoryRecord{
			{
				ID:        1,
				TokoID:    10,
				TokoName:  "Toko A",
				Player:    &player,
				Type:      model.TransactionTypeDeposit,
				Status:    model.TransactionStatusSuccess,
				Reference: &reference,
				Amount:    100000,
				Netto:     97000,
				CreatedAt: now,
			},
		},
	}
	gateway := &fakeDashboardGatewayClient{
		balance: &paymentgateway.GetBalanceResponse{
			Status:         "true",
			PendingBalance: 10000,
			SettleBalance:  25000,
		},
	}
	svc := NewDashboardService(repo, gateway, cache.NewNoopCache(), "merchant-uuid", "dewifork", 5*time.Minute)

	result, err := svc.Overview(context.Background(), 1, model.UserRoleDev)
	require.NoError(t, err)
	require.Len(t, result.StatusSeries, 12)
	require.InDelta(t, 70, result.Metrics.SuccessRate, 0.001)
	require.Equal(t, int64(120000), result.Metrics.NetFlow)
	require.Equal(t, uint64(4500), result.Metrics.ProjectProfit)
	require.True(t, result.CanViewProjectProfit)
	require.True(t, result.CanViewExternalBalance)
	require.Len(t, result.LatestSuccessOrders, 1)
	require.Equal(t, uint64(10000), result.ExternalBalance.PendingBalance)
	require.Equal(t, uint64(25000), result.ExternalBalance.AvailableBalance)
}

func TestDashboardServiceTransactionHistoryMapsRecords(t *testing.T) {
	player := "player-a"
	reference := "trx-ref-1"
	repo := &fakeDashboardTransactionRepo{
		history: []repository.TransactionHistoryRecord{
			{
				ID:        1,
				TokoID:    10,
				TokoName:  "Toko A",
				Player:    &player,
				Type:      model.TransactionTypeDeposit,
				Status:    model.TransactionStatusSuccess,
				Reference: &reference,
				Amount:    100000,
				Netto:     100000,
				CreatedAt: time.Now().UTC(),
			},
		},
	}
	svc := NewDashboardService(repo, nil, cache.NewNoopCache(), "", "", 5*time.Minute)

	result, err := svc.TransactionHistory(context.Background(), 1, TransactionHistoryQuery{Limit: 10})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, uint64(1), result.Total)
	require.Equal(t, "Toko A", result.Items[0].TokoName)
	require.Equal(t, "deposit", result.Items[0].Type)
}

func TestDashboardServiceTransactionHistoryPropagatesSearchAndDateFilters(t *testing.T) {
	from := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 21, 23, 59, 59, 0, time.UTC)
	repo := &fakeDashboardTransactionRepo{
		history: []repository.TransactionHistoryRecord{},
	}
	svc := NewDashboardService(repo, nil, cache.NewNoopCache(), "", "", 5*time.Minute)

	_, err := svc.TransactionHistory(context.Background(), 8, TransactionHistoryQuery{
		Limit:      50,
		Offset:     10,
		From:       &from,
		To:         &to,
		SearchTerm: "trx-001",
	})
	require.NoError(t, err)
	require.Equal(t, 50, repo.lastFilter.Limit)
	require.Equal(t, 10, repo.lastFilter.Offset)
	require.Equal(t, "trx-001", repo.lastFilter.SearchTerm)
	require.NotNil(t, repo.lastFilter.From)
	require.NotNil(t, repo.lastFilter.To)
	require.True(t, repo.lastFilter.From.UTC().Equal(from))
	require.True(t, repo.lastFilter.To.UTC().Equal(to))
}

func TestDashboardServiceTransactionHistorySanitizesPaginationBounds(t *testing.T) {
	repo := &fakeDashboardTransactionRepo{
		history: []repository.TransactionHistoryRecord{},
	}
	svc := NewDashboardService(repo, nil, cache.NewNoopCache(), "", "", 5*time.Minute)

	page, err := svc.TransactionHistory(context.Background(), 8, TransactionHistoryQuery{
		Limit:  999,
		Offset: -7,
	})
	require.NoError(t, err)
	require.NotNil(t, page)
	require.Equal(t, 200, page.Limit)
	require.Equal(t, 0, page.Offset)
	require.Equal(t, 200, repo.lastFilter.Limit)
	require.Equal(t, 0, repo.lastFilter.Offset)
}

func TestDashboardServiceExportTransactionHistoryCSV(t *testing.T) {
	player := "player-a"
	reference := "trx-ref-1"
	repo := &fakeDashboardTransactionRepo{
		history: []repository.TransactionHistoryRecord{
			{
				ID:        1,
				TokoID:    10,
				TokoName:  "Toko A",
				Player:    &player,
				Type:      model.TransactionTypeDeposit,
				Status:    model.TransactionStatusSuccess,
				Reference: &reference,
				Amount:    100000,
				Netto:     97000,
				CreatedAt: time.Now().UTC(),
			},
		},
	}
	svc := NewDashboardService(repo, nil, cache.NewNoopCache(), "", "", 5*time.Minute)

	exported, err := svc.ExportTransactionHistory(context.Background(), 1, TransactionHistoryQuery{Limit: 100}, "csv")
	require.NoError(t, err)
	require.Equal(t, "text/csv; charset=utf-8", exported.ContentType)
	require.Equal(t, "transaction-history.csv", exported.FileName)
	require.Contains(t, string(exported.Content), "toko_name")
	require.Contains(t, string(exported.Content), "Toko A")
	require.Equal(t, 100, repo.lastFilter.Limit)
	require.Equal(t, 0, repo.lastFilter.Offset)
}

func TestDashboardServiceExportTransactionHistoryDefaultsToTenThousand(t *testing.T) {
	repo := &fakeDashboardTransactionRepo{
		history: []repository.TransactionHistoryRecord{},
	}
	svc := NewDashboardService(repo, nil, cache.NewNoopCache(), "", "", 5*time.Minute)

	_, err := svc.ExportTransactionHistory(context.Background(), 1, TransactionHistoryQuery{}, "csv")
	require.NoError(t, err)
	require.Equal(t, 10000, repo.lastFilter.Limit)
	require.Equal(t, 0, repo.lastFilter.Offset)
}

func TestDashboardServiceExportTransactionHistoryLimitIsCappedToTenThousand(t *testing.T) {
	repo := &fakeDashboardTransactionRepo{
		history: []repository.TransactionHistoryRecord{},
	}
	svc := NewDashboardService(repo, nil, cache.NewNoopCache(), "", "", 5*time.Minute)

	_, err := svc.ExportTransactionHistory(context.Background(), 1, TransactionHistoryQuery{Limit: 25000}, "csv")
	require.NoError(t, err)
	require.Equal(t, 10000, repo.lastFilter.Limit)
	require.Equal(t, 0, repo.lastFilter.Offset)
}

func TestDashboardServiceExportTransactionHistoryDOCX(t *testing.T) {
	player := "player-a"
	reference := "trx-ref-1"
	repo := &fakeDashboardTransactionRepo{
		history: []repository.TransactionHistoryRecord{
			{
				ID:        1,
				TokoID:    10,
				TokoName:  "Toko A",
				Player:    &player,
				Type:      model.TransactionTypeDeposit,
				Status:    model.TransactionStatusSuccess,
				Reference: &reference,
				Amount:    100000,
				Netto:     97000,
				CreatedAt: time.Now().UTC(),
			},
		},
	}
	svc := NewDashboardService(repo, nil, cache.NewNoopCache(), "", "", 5*time.Minute)

	exported, err := svc.ExportTransactionHistory(context.Background(), 1, TransactionHistoryQuery{Limit: 100}, "docx")
	require.NoError(t, err)
	require.Equal(t, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", exported.ContentType)
	require.Equal(t, "transaction-history.docx", exported.FileName)
	require.Greater(t, len(exported.Content), 2)
	require.Equal(t, []byte("PK"), exported.Content[:2])
}

func TestDashboardServiceOverviewCachesExternalBalanceForFiveMinutes(t *testing.T) {
	repo := &fakeDashboardTransactionRepo{}
	gateway := &fakeDashboardGatewayClient{
		balance: &paymentgateway.GetBalanceResponse{
			Status:         "true",
			PendingBalance: 1200,
			SettleBalance:  3400,
		},
	}
	cacheStore := newFakeDashboardCache()
	svc := NewDashboardService(repo, gateway, cacheStore, "merchant-uuid", "dewifork", 5*time.Minute)

	first, err := svc.Overview(context.Background(), 1, model.UserRoleDev)
	require.NoError(t, err)
	require.Equal(t, uint64(1200), first.ExternalBalance.PendingBalance)
	require.Equal(t, uint64(3400), first.ExternalBalance.AvailableBalance)

	second, err := svc.Overview(context.Background(), 1, model.UserRoleDev)
	require.NoError(t, err)
	require.Equal(t, uint64(1200), second.ExternalBalance.PendingBalance)
	require.Equal(t, uint64(3400), second.ExternalBalance.AvailableBalance)

	require.Equal(t, 1, gateway.calls)
	require.Equal(t, 5*time.Minute, cacheStore.ttls["dashboard:external-balance:merchant-uuid:dewifork"])
}

func TestDashboardServiceOverviewCachesComputedOverviewForShortTTL(t *testing.T) {
	repo := &fakeDashboardTransactionRepo{
		metrics: repository.DashboardMetrics{
			TotalCount:   4,
			SuccessCount: 3,
		},
	}
	cacheStore := newFakeDashboardCache()
	svc := NewDashboardService(repo, nil, cacheStore, "", "", 5*time.Minute)

	first, err := svc.Overview(context.Background(), 44, model.UserRoleDev)
	require.NoError(t, err)
	require.NotNil(t, first)

	second, err := svc.Overview(context.Background(), 44, model.UserRoleDev)
	require.NoError(t, err)
	require.NotNil(t, second)

	require.Equal(t, 1, repo.metricsHit)
	require.Equal(t, 1, repo.seriesHit)
	require.Equal(t, 1, repo.successHit)

	overviewCacheKey := ""
	for key, ttl := range cacheStore.ttls {
		if strings.HasPrefix(key, "dashboard:overview:44:dev:") {
			overviewCacheKey = key
			require.Equal(t, 10*time.Second, ttl)
			break
		}
	}
	require.NotEmpty(t, overviewCacheKey)
}

func TestDashboardServiceOverviewStillSucceedsWhenCacheFails(t *testing.T) {
	repo := &fakeDashboardTransactionRepo{
		metrics: repository.DashboardMetrics{
			TotalCount:   2,
			SuccessCount: 1,
		},
	}
	gateway := &fakeDashboardGatewayClient{
		balance: &paymentgateway.GetBalanceResponse{
			PendingBalance: 100,
			SettleBalance:  200,
		},
	}
	cacheStore := newFakeDashboardCache()
	cacheStore.fail = true
	svc := NewDashboardService(repo, gateway, cacheStore, "merchant-uuid", "dewifork", 5*time.Minute)

	result, err := svc.Overview(context.Background(), 99, model.UserRoleDev)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.ExternalBalanceError)
	require.Equal(t, uint64(100), result.ExternalBalance.PendingBalance)
	require.Equal(t, uint64(200), result.ExternalBalance.AvailableBalance)
}

func TestDashboardServiceOverviewRedactsPrivilegedFinancialsForNonDev(t *testing.T) {
	repo := &fakeDashboardTransactionRepo{
		metrics: repository.DashboardMetrics{
			TotalCount:       6,
			SuccessCount:     4,
			PendingCount:     1,
			FailedCount:      1,
			TotalPlatformFee: 5500,
		},
	}
	gateway := &fakeDashboardGatewayClient{
		balance: &paymentgateway.GetBalanceResponse{
			PendingBalance: 1000,
			SettleBalance:  2000,
		},
	}
	svc := NewDashboardService(repo, gateway, cache.NewNoopCache(), "merchant-uuid", "dewifork", 5*time.Minute)

	result, err := svc.Overview(context.Background(), 101, model.UserRoleAdmin)
	require.NoError(t, err)
	require.False(t, result.CanViewProjectProfit)
	require.False(t, result.CanViewExternalBalance)
	require.Equal(t, uint64(0), result.Metrics.ProjectProfit)
	require.Equal(t, uint64(0), result.ExternalBalance.PendingBalance)
	require.Equal(t, uint64(0), result.ExternalBalance.AvailableBalance)
	require.Empty(t, result.ExternalBalanceError)
	require.Equal(t, 0, gateway.calls)
}
