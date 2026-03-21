package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/example/gue/backend/cache"
	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/paymentgateway"
	"github.com/example/gue/backend/repository"
	"github.com/stretchr/testify/require"
)

type fakeDashboardTransactionRepo struct {
	metrics repository.DashboardMetrics
	series  []repository.DashboardStatusSeriesPoint
	history []repository.TransactionHistoryRecord
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

func (f *fakeDashboardTransactionRepo) GetDashboardMetricsByUser(_ context.Context, _ uint64, _ time.Time) (*repository.DashboardMetrics, error) {
	value := f.metrics
	return &value, nil
}

func (f *fakeDashboardTransactionRepo) GetHourlyStatusCountsByUser(_ context.Context, _ uint64, _ time.Time) ([]repository.DashboardStatusSeriesPoint, error) {
	return f.series, nil
}

func (f *fakeDashboardTransactionRepo) ListRecentByUser(_ context.Context, _ uint64, _ int) ([]repository.TransactionHistoryRecord, error) {
	return f.history, nil
}

func (f *fakeDashboardTransactionRepo) ListRecentSuccessByUser(_ context.Context, _ uint64, limit int) ([]repository.TransactionHistoryRecord, error) {
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
}

func newFakeDashboardCache() *fakeDashboardCache {
	return &fakeDashboardCache{
		items: make(map[string][]byte),
		ttls:  make(map[string]time.Duration),
	}
}

func (f *fakeDashboardCache) Get(_ context.Context, key string) ([]byte, error) {
	value, ok := f.items[key]
	if !ok {
		return nil, cache.ErrCacheMiss
	}
	return value, nil
}

func (f *fakeDashboardCache) Set(_ context.Context, key string, value any, ttl time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f.items[key] = payload
	f.ttls[key] = ttl
	return nil
}

func (f *fakeDashboardCache) Delete(_ context.Context, key string) error {
	delete(f.items, key)
	delete(f.ttls, key)
	return nil
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

	result, err := svc.Overview(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, result.StatusSeries, 12)
	require.InDelta(t, 70, result.Metrics.SuccessRate, 0.001)
	require.Equal(t, int64(120000), result.Metrics.NetFlow)
	require.Equal(t, uint64(4500), result.Metrics.ProjectProfit)
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

	result, err := svc.TransactionHistory(context.Background(), 1, 10)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, "Toko A", result[0].TokoName)
	require.Equal(t, "deposit", result[0].Type)
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

	first, err := svc.Overview(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, uint64(1200), first.ExternalBalance.PendingBalance)
	require.Equal(t, uint64(3400), first.ExternalBalance.AvailableBalance)

	second, err := svc.Overview(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, uint64(1200), second.ExternalBalance.PendingBalance)
	require.Equal(t, uint64(3400), second.ExternalBalance.AvailableBalance)

	require.Equal(t, 1, gateway.calls)
	require.Equal(t, 5*time.Minute, cacheStore.ttls["dashboard:external-balance:merchant-uuid:dewifork"])
}
