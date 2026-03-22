package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/example/gue/backend/cache"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/paymentgateway"
	"github.com/example/gue/backend/repository"
)

type DashboardUseCase interface {
	Overview(ctx context.Context, userID uint64) (*DashboardOverviewResult, error)
	TransactionHistory(ctx context.Context, userID uint64, query TransactionHistoryQuery) (*TransactionHistoryPage, error)
	ExportTransactionHistory(ctx context.Context, userID uint64, query TransactionHistoryQuery, format string) (*TransactionHistoryExport, error)
}

type DashboardService struct {
	transactionRepo repository.TransactionRepository
	gatewayClient   paymentgateway.Client
	cache           cache.Cache
	merchantUUID    string
	defaultClient   string
	balanceTTL      time.Duration
}

func NewDashboardService(
	transactionRepo repository.TransactionRepository,
	gatewayClient paymentgateway.Client,
	cacheStore cache.Cache,
	merchantUUID string,
	defaultClient string,
	balanceTTL time.Duration,
) *DashboardService {
	if balanceTTL <= 0 {
		balanceTTL = 5 * time.Minute
	}
	return &DashboardService{
		transactionRepo: transactionRepo,
		gatewayClient:   gatewayClient,
		cache:           cacheStore,
		merchantUUID:    strings.TrimSpace(merchantUUID),
		defaultClient:   strings.TrimSpace(defaultClient),
		balanceTTL:      balanceTTL,
	}
}

type DashboardOverviewResult struct {
	WindowHours          int64                    `json:"window_hours"`
	CanViewProjectProfit bool                     `json:"can_view_project_profit"`
	Metrics              DashboardMetricsDTO      `json:"metrics"`
	StatusSeries         []DashboardStatusSeries  `json:"status_series"`
	LatestSuccessOrders  []TransactionHistoryItem `json:"latest_success_orders"`
	ExternalBalance      DashboardExternalBalance `json:"external_balance"`
	ExternalBalanceError string                   `json:"external_balance_error,omitempty"`
	UpdatedAt            string                   `json:"updated_at"`
}

type DashboardMetricsDTO struct {
	TotalTransactions   uint64  `json:"total_transactions"`
	SuccessTransactions uint64  `json:"success_transactions"`
	PendingTransactions uint64  `json:"pending_transactions"`
	FailedTransactions  uint64  `json:"failed_transactions"`
	SuccessRate         float64 `json:"success_rate"`
	SuccessDeposit      uint64  `json:"success_deposit"`
	SuccessWithdraw     uint64  `json:"success_withdraw"`
	NetFlow             int64   `json:"net_flow"`
	ProjectProfit       uint64  `json:"project_profit"`
}

type DashboardStatusSeries struct {
	Bucket             string `json:"bucket"`
	SuccessCount       uint64 `json:"success_count"`
	FailedExpiredCount uint64 `json:"failed_expired_count"`
}

type DashboardExternalBalance struct {
	PendingBalance   uint64 `json:"pending_balance"`
	AvailableBalance uint64 `json:"available_balance"`
}

type TransactionHistoryItem struct {
	ID        uint64 `json:"id"`
	TokoID    uint64 `json:"toko_id"`
	TokoName  string `json:"toko_name"`
	Player    string `json:"player,omitempty"`
	Code      string `json:"code,omitempty"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	Reference string `json:"reference,omitempty"`
	Amount    uint64 `json:"amount"`
	Netto     uint64 `json:"netto"`
	CreatedAt string `json:"created_at"`
}

type TransactionHistoryQuery struct {
	Limit      int
	Offset     int
	From       *time.Time
	To         *time.Time
	SearchTerm string
}

type TransactionHistoryPage struct {
	Items   []TransactionHistoryItem `json:"items"`
	Total   uint64                   `json:"total"`
	Limit   int                      `json:"limit"`
	Offset  int                      `json:"offset"`
	HasMore bool                     `json:"has_more"`
}

type TransactionHistoryExport struct {
	Content     []byte
	ContentType string
	FileName    string
}

func (s *DashboardService) Overview(ctx context.Context, userID uint64) (*DashboardOverviewResult, error) {
	const windowHours = int64(12)
	from := time.Now().UTC().Add(-time.Duration(windowHours-1) * time.Hour).Truncate(time.Hour)

	metrics, err := s.transactionRepo.GetDashboardMetricsByUser(ctx, userID, from)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to fetch dashboard metrics", err.Error())
	}

	statusPoints, err := s.transactionRepo.GetHourlyStatusCountsByUser(ctx, userID, from)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to fetch dashboard status series", err.Error())
	}

	statusPointByBucket := make(map[time.Time]repository.DashboardStatusSeriesPoint, len(statusPoints))
	for _, point := range statusPoints {
		statusPointByBucket[point.Bucket.UTC().Truncate(time.Hour)] = point
	}

	series := make([]DashboardStatusSeries, 0, windowHours)
	current := from
	for i := int64(0); i < windowHours; i++ {
		point := statusPointByBucket[current]
		series = append(series, DashboardStatusSeries{
			Bucket:             current.Format(time.RFC3339),
			SuccessCount:       point.SuccessCount,
			FailedExpiredCount: point.FailedCount,
		})
		current = current.Add(time.Hour)
	}

	successOrders, err := s.latestSuccessOrders(ctx, userID, 8)
	if err != nil {
		return nil, err
	}

	successRate := 0.0
	if metrics.TotalCount > 0 {
		successRate = (float64(metrics.SuccessCount) / float64(metrics.TotalCount)) * 100
	}

	netFlow := int64(metrics.SuccessDepositAmount) - int64(metrics.SuccessWithdrawAmount)
	externalBalance, externalErr := s.fetchExternalBalance(ctx)

	return &DashboardOverviewResult{
		WindowHours: windowHours,
		Metrics: DashboardMetricsDTO{
			TotalTransactions:   metrics.TotalCount,
			SuccessTransactions: metrics.SuccessCount,
			PendingTransactions: metrics.PendingCount,
			FailedTransactions:  metrics.FailedCount,
			SuccessRate:         successRate,
			SuccessDeposit:      metrics.SuccessDepositAmount,
			SuccessWithdraw:     metrics.SuccessWithdrawAmount,
			NetFlow:             netFlow,
			ProjectProfit:       metrics.TotalPlatformFee,
		},
		StatusSeries:         series,
		LatestSuccessOrders:  successOrders,
		ExternalBalance:      externalBalance,
		ExternalBalanceError: externalErr,
		UpdatedAt:            time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *DashboardService) TransactionHistory(ctx context.Context, userID uint64, query TransactionHistoryQuery) (*TransactionHistoryPage, error) {
	filter := sanitizeTransactionHistoryFilter(query)

	total, err := s.transactionRepo.CountHistoryByUser(ctx, userID, filter)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to count transaction history", err.Error())
	}

	records, err := s.transactionRepo.ListRecentByUser(ctx, userID, filter)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to fetch transaction history", err.Error())
	}

	items := mapTransactionHistory(records)
	return &TransactionHistoryPage{
		Items:   items,
		Total:   total,
		Limit:   filter.Limit,
		Offset:  filter.Offset,
		HasMore: uint64(filter.Offset+len(items)) < total,
	}, nil
}

func (s *DashboardService) ExportTransactionHistory(ctx context.Context, userID uint64, query TransactionHistoryQuery, format string) (*TransactionHistoryExport, error) {
	filter := sanitizeTransactionHistoryExportFilter(query)

	records, err := s.transactionRepo.ListRecentByUser(ctx, userID, filter)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to fetch transaction history for export", err.Error())
	}
	items := mapTransactionHistory(records)

	switch strings.ToLower(strings.TrimSpace(format)) {
	case "csv":
		content, exportErr := BuildTransactionHistoryCSV(items)
		if exportErr != nil {
			return nil, apperror.New(http.StatusInternalServerError, "failed to export CSV", exportErr.Error())
		}
		return &TransactionHistoryExport{
			Content:     content,
			ContentType: "text/csv; charset=utf-8",
			FileName:    "transaction-history.csv",
		}, nil
	case "docx":
		content, exportErr := BuildTransactionHistoryDOCX(items)
		if exportErr != nil {
			return nil, apperror.New(http.StatusInternalServerError, "failed to export DOCX", exportErr.Error())
		}
		return &TransactionHistoryExport{
			Content:     content,
			ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			FileName:    "transaction-history.docx",
		}, nil
	default:
		return nil, apperror.New(http.StatusBadRequest, "invalid export format", "supported formats: csv, docx")
	}
}

func (s *DashboardService) latestSuccessOrders(ctx context.Context, userID uint64, limit int) ([]TransactionHistoryItem, error) {
	records, err := s.transactionRepo.ListRecentSuccessByUser(ctx, userID, limit)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to fetch latest success orders", err.Error())
	}
	return mapTransactionHistory(records), nil
}

func mapTransactionHistory(records []repository.TransactionHistoryRecord) []TransactionHistoryItem {
	result := make([]TransactionHistoryItem, 0, len(records))
	for _, record := range records {
		item := TransactionHistoryItem{
			ID:        record.ID,
			TokoID:    record.TokoID,
			TokoName:  record.TokoName,
			Type:      string(record.Type),
			Status:    string(record.Status),
			Amount:    record.Amount,
			Netto:     record.Netto,
			CreatedAt: record.CreatedAt.UTC().Format(time.RFC3339),
		}
		if record.Player != nil {
			item.Player = *record.Player
		}
		if record.Code != nil {
			item.Code = *record.Code
		}
		if record.Reference != nil {
			item.Reference = *record.Reference
		}
		result = append(result, item)
	}
	return result
}

func sanitizeTransactionHistoryFilter(query TransactionHistoryQuery) repository.TransactionHistoryFilter {
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	offset := query.Offset
	if offset < 0 {
		offset = 0
	}

	return repository.TransactionHistoryFilter{
		Limit:      limit,
		Offset:     offset,
		From:       query.From,
		To:         query.To,
		SearchTerm: query.SearchTerm,
	}
}

func sanitizeTransactionHistoryExportFilter(query TransactionHistoryQuery) repository.TransactionHistoryFilter {
	limit := query.Limit
	if limit <= 0 {
		limit = 10000
	}
	if limit > 10000 {
		limit = 10000
	}

	return repository.TransactionHistoryFilter{
		Limit:      limit,
		Offset:     0,
		From:       query.From,
		To:         query.To,
		SearchTerm: query.SearchTerm,
	}
}

func (s *DashboardService) fetchExternalBalance(ctx context.Context) (DashboardExternalBalance, string) {
	if s.gatewayClient == nil || s.merchantUUID == "" || s.defaultClient == "" {
		return DashboardExternalBalance{}, "gateway configuration is incomplete"
	}

	cacheKey := "dashboard:external-balance:" + s.merchantUUID + ":" + s.defaultClient
	if s.cache != nil {
		cachedBytes, err := s.cache.Get(ctx, cacheKey)
		if err == nil {
			var cached DashboardExternalBalance
			if unmarshalErr := json.Unmarshal(cachedBytes, &cached); unmarshalErr == nil {
				return cached, ""
			}
		} else if !errors.Is(err, cache.ErrCacheMiss) {
			return DashboardExternalBalance{}, err.Error()
		}
	}

	resp, err := s.gatewayClient.GetBalance(ctx, s.merchantUUID, paymentgateway.GetBalanceRequest{
		Client: s.defaultClient,
	})
	if err != nil {
		return DashboardExternalBalance{}, err.Error()
	}

	balance := DashboardExternalBalance{
		PendingBalance:   resp.PendingBalance,
		AvailableBalance: resp.SettleBalance,
	}
	if s.cache != nil {
		if err := s.cache.Set(ctx, cacheKey, balance, s.balanceTTL); err != nil {
			return balance, err.Error()
		}
	}
	return balance, ""
}
