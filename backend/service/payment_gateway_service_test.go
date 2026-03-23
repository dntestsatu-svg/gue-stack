package service

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/example/gue/backend/cache"
	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/money"
	"github.com/example/gue/backend/pkg/paymentgateway"
	"github.com/example/gue/backend/queue"
	"github.com/example/gue/backend/repository"
	"github.com/stretchr/testify/require"
	"log/slog"
)

type fakePaymentGatewayClient struct {
	generateResp *paymentgateway.GenerateResponse
	generateErr  error
	generateReq  paymentgateway.GenerateRequest
	checkResp    *paymentgateway.CheckStatusResponse
	checkErr     error
	checkCalls   int
	inquiryResp  *paymentgateway.InquiryTransferResponse
	inquiryErr   error
	inquiryReq   paymentgateway.InquiryTransferRequest
	transferResp *paymentgateway.TransferFundResponse
	transferErr  error
	transferReq  paymentgateway.TransferFundRequest
}

func (f *fakePaymentGatewayClient) Generate(_ context.Context, req paymentgateway.GenerateRequest) (*paymentgateway.GenerateResponse, error) {
	f.generateReq = req
	return f.generateResp, f.generateErr
}

func (f *fakePaymentGatewayClient) CheckStatusV2(_ context.Context, _ string, _ paymentgateway.CheckStatusRequest) (*paymentgateway.CheckStatusResponse, error) {
	f.checkCalls++
	return f.checkResp, f.checkErr
}

func (f *fakePaymentGatewayClient) InquiryTransfer(_ context.Context, req paymentgateway.InquiryTransferRequest) (*paymentgateway.InquiryTransferResponse, error) {
	f.inquiryReq = req
	return f.inquiryResp, f.inquiryErr
}

func (f *fakePaymentGatewayClient) TransferFund(_ context.Context, req paymentgateway.TransferFundRequest) (*paymentgateway.TransferFundResponse, error) {
	f.transferReq = req
	return f.transferResp, f.transferErr
}

func (f *fakePaymentGatewayClient) CheckTransferStatus(_ context.Context, _ string, _ paymentgateway.CheckTransferStatusRequest) (*paymentgateway.CheckTransferStatusResponse, error) {
	return nil, nil
}

func (f *fakePaymentGatewayClient) GetBalance(_ context.Context, _ string, _ paymentgateway.GetBalanceRequest) (*paymentgateway.GetBalanceResponse, error) {
	return nil, nil
}

type fakeTokoRepo struct {
	byID    map[uint64]*model.Toko
	byToken map[string]*model.Toko
}

func (f *fakeTokoRepo) Create(_ context.Context, toko *model.Toko) error {
	if f.byID == nil {
		f.byID = map[uint64]*model.Toko{}
	}
	toko.ID = uint64(len(f.byID) + 1)
	f.byID[toko.ID] = toko
	return nil
}

func (f *fakeTokoRepo) CreateForUserWithQuota(ctx context.Context, _ uint64, toko *model.Toko, _ int) error {
	return f.Create(ctx, toko)
}

func (f *fakeTokoRepo) AttachUser(_ context.Context, _, _ uint64) error {
	return nil
}

func (f *fakeTokoRepo) CountByUser(_ context.Context, _ uint64) (int, error) {
	return 0, nil
}

func (f *fakeTokoRepo) ListByUser(_ context.Context, _ uint64, _ model.UserRole) ([]model.Toko, error) {
	return nil, nil
}

func (f *fakeTokoRepo) ListWorkspaceByUser(_ context.Context, _ uint64, _ model.UserRole, _ repository.TokoWorkspaceFilter) ([]repository.TokoWorkspaceRecord, error) {
	return nil, nil
}

func (f *fakeTokoRepo) SummarizeWorkspaceByUser(_ context.Context, _ uint64, _ model.UserRole, _ repository.TokoWorkspaceFilter) (*repository.TokoWorkspaceSummary, error) {
	return &repository.TokoWorkspaceSummary{}, nil
}

func (f *fakeTokoRepo) GetByID(_ context.Context, id uint64) (*model.Toko, error) {
	t, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return t, nil
}

func (f *fakeTokoRepo) GetAccessibleByID(ctx context.Context, _ uint64, _ model.UserRole, tokoID uint64) (*model.Toko, error) {
	return f.GetByID(ctx, tokoID)
}

func (f *fakeTokoRepo) GetByToken(_ context.Context, token string) (*model.Toko, error) {
	t, ok := f.byToken[token]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return t, nil
}

func (f *fakeTokoRepo) UpdateProfile(_ context.Context, tokoID uint64, name string, callbackURL *string) error {
	t, ok := f.byID[tokoID]
	if !ok {
		return repository.ErrNotFound
	}
	t.Name = name
	t.CallbackURL = callbackURL
	return nil
}

func (f *fakeTokoRepo) UpdateToken(_ context.Context, tokoID uint64, token string) error {
	t, ok := f.byID[tokoID]
	if !ok {
		return repository.ErrNotFound
	}
	t.Token = token
	return nil
}

type settlementUpdate struct {
	id          uint64
	status      model.TransactionStatus
	platformFee uint64
	netto       uint64
}

type statusUpdate struct {
	reference string
	tokoID    uint64
	status    model.TransactionStatus
}

type fakeTransactionRepo struct {
	created           []*model.Transaction
	byReference       map[string]*model.Transaction
	updates           []settlementUpdate
	statuses          []statusUpdate
	withdrawFinalized []statusUpdate
	balanceRepo       *fakeBalanceRepo
	history           []repository.TransactionHistoryRecord
	historyCount      uint64
	lastHistoryFilter repository.TransactionHistoryFilter
	expiryCandidates  []repository.PendingExpiryCandidate
}

func (f *fakeTransactionRepo) Create(_ context.Context, trx *model.Transaction) error {
	if f.byReference == nil {
		f.byReference = map[string]*model.Transaction{}
	}
	if trx.ID == 0 {
		trx.ID = uint64(len(f.created) + 1)
	}
	f.created = append(f.created, trx)
	if trx.Reference != nil {
		f.byReference[*trx.Reference] = trx
	}
	return nil
}

func (f *fakeTransactionRepo) GetByReference(_ context.Context, reference string) (*model.Transaction, error) {
	if trx, ok := f.byReference[reference]; ok {
		return trx, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeTransactionRepo) GetByReferenceAndToko(_ context.Context, reference string, tokoID uint64) (*model.Transaction, error) {
	if trx, ok := f.byReference[reference]; ok && trx.TokoID == tokoID {
		return trx, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeTransactionRepo) UpdateStatusByReference(_ context.Context, _ string, _ model.TransactionStatus) error {
	return nil
}

func (f *fakeTransactionRepo) UpdateStatusByReferenceAndToko(_ context.Context, reference string, tokoID uint64, status model.TransactionStatus) error {
	f.statuses = append(f.statuses, statusUpdate{
		reference: reference,
		tokoID:    tokoID,
		status:    status,
	})
	if trx, ok := f.byReference[reference]; ok && trx.TokoID == tokoID {
		trx.Status = status
		return nil
	}
	return nil
}

func (f *fakeTransactionRepo) UpdateSettlementByID(_ context.Context, id uint64, status model.TransactionStatus, platformFee uint64, netto uint64) error {
	f.updates = append(f.updates, settlementUpdate{
		id:          id,
		status:      status,
		platformFee: platformFee,
		netto:       netto,
	})
	return nil
}

func (f *fakeTransactionRepo) UpdateSettlementIfPending(_ context.Context, id uint64, status model.TransactionStatus, platformFee uint64, netto uint64) (bool, error) {
	f.updates = append(f.updates, settlementUpdate{
		id:          id,
		status:      status,
		platformFee: platformFee,
		netto:       netto,
	})
	for _, trx := range f.byReference {
		if trx.ID == id {
			if trx.Status != model.TransactionStatusPending {
				return false, nil
			}
			trx.Status = status
			trx.PlatformFee = platformFee
			trx.Netto = netto
			return true, nil
		}
	}
	return true, nil
}

func (f *fakeTransactionRepo) FinalizeDepositSuccessByID(_ context.Context, id uint64, _ uint64, platformFee uint64, netto uint64) (bool, error) {
	return f.UpdateSettlementIfPending(context.Background(), id, model.TransactionStatusSuccess, platformFee, netto)
}

func (f *fakeTransactionRepo) CreatePendingWithdrawAndReserveSettlement(_ context.Context, trx *model.Transaction) error {
	if f.balanceRepo != nil {
		item, ok := f.balanceRepo.byTokoID[trx.TokoID]
		if !ok {
			return repository.ErrNotFound
		}
		fee := uint64(0)
		if trx.FeeWithdrawal != nil {
			fee = *trx.FeeWithdrawal
		}
		totalDebit, err := money.AddUint64(trx.Amount, fee)
		if err != nil {
			return err
		}
		if item.SettleBalance < float64(totalDebit) {
			return repository.ErrInsufficientBalance
		}
		item.SettleBalance -= float64(totalDebit)
		f.balanceRepo.byTokoID[trx.TokoID] = item
	}
	return f.Create(context.Background(), trx)
}

func (f *fakeTransactionRepo) FinalizeWithdrawIfPending(_ context.Context, id uint64, status model.TransactionStatus) (bool, error) {
	f.withdrawFinalized = append(f.withdrawFinalized, statusUpdate{
		tokoID: id,
		status: status,
	})
	for _, trx := range f.byReference {
		if trx.ID == id {
			if trx.Status != model.TransactionStatusPending {
				return false, nil
			}
			if f.balanceRepo != nil && (status == model.TransactionStatusFailed || status == model.TransactionStatusExpired) {
				item, ok := f.balanceRepo.byTokoID[trx.TokoID]
				if !ok {
					return false, repository.ErrNotFound
				}
				fee := uint64(0)
				if trx.FeeWithdrawal != nil {
					fee = *trx.FeeWithdrawal
				}
				totalRefund, err := money.AddUint64(trx.Amount, fee)
				if err != nil {
					return false, err
				}
				item.SettleBalance += float64(totalRefund)
				f.balanceRepo.byTokoID[trx.TokoID] = item
			}
			trx.Status = status
			return true, nil
		}
	}
	return false, repository.ErrNotFound
}

func (f *fakeTransactionRepo) ListPendingExpiryCandidates(_ context.Context, _ time.Time, _ int) ([]repository.PendingExpiryCandidate, error) {
	return append([]repository.PendingExpiryCandidate(nil), f.expiryCandidates...), nil
}

func (f *fakeTransactionRepo) GetDashboardMetricsByUser(_ context.Context, _ uint64, _ time.Time) (*repository.DashboardMetrics, error) {
	return &repository.DashboardMetrics{}, nil
}

func (f *fakeTransactionRepo) GetHourlyStatusCountsByUser(_ context.Context, _ uint64, _ time.Time) ([]repository.DashboardStatusSeriesPoint, error) {
	return nil, nil
}

func (f *fakeTransactionRepo) ListRecentByUser(_ context.Context, _ uint64, filter repository.TransactionHistoryFilter) ([]repository.TransactionHistoryRecord, error) {
	f.lastHistoryFilter = filter
	if f.history == nil {
		return nil, nil
	}
	return append([]repository.TransactionHistoryRecord(nil), f.history...), nil
}

func (f *fakeTransactionRepo) ListRecentSuccessByUser(_ context.Context, _ uint64, _ int) ([]repository.TransactionHistoryRecord, error) {
	return nil, nil
}

func (f *fakeTransactionRepo) CountHistoryByUser(_ context.Context, _ uint64, filter repository.TransactionHistoryFilter) (uint64, error) {
	f.lastHistoryFilter = filter
	return f.historyCount, nil
}

type fakeGatewayProducer struct {
	qrisCallbacks []queue.QrisCallbackTaskPayload
}

func (f *fakeGatewayProducer) EnqueueWelcomeEmail(_ context.Context, _, _ string) error {
	return nil
}

func (f *fakeGatewayProducer) EnqueueQrisCallback(_ context.Context, payload queue.QrisCallbackTaskPayload) error {
	f.qrisCallbacks = append(f.qrisCallbacks, payload)
	return nil
}

func (f *fakeGatewayProducer) EnqueueTransferCallback(_ context.Context, _ queue.TransferCallbackTaskPayload) error {
	return nil
}

type fakePaymentCache struct {
	items map[string][]byte
}

func newFakePaymentCache() *fakePaymentCache {
	return &fakePaymentCache{items: map[string][]byte{}}
}

func (f *fakePaymentCache) Get(_ context.Context, key string) ([]byte, error) {
	if value, ok := f.items[key]; ok {
		return value, nil
	}
	return nil, cache.ErrCacheMiss
}

func (f *fakePaymentCache) Set(_ context.Context, key string, value any, _ time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f.items[key] = payload
	return nil
}

func (f *fakePaymentCache) Delete(_ context.Context, key string) error {
	delete(f.items, key)
	return nil
}

type fakeCallbackHTTPClient struct {
	statusCode int
	body       string
	err        error
	calls      int
	lastURL    string
}

func (f *fakeCallbackHTTPClient) Do(req *http.Request) (*http.Response, error) {
	f.calls++
	f.lastURL = req.URL.String()
	if f.err != nil {
		return nil, f.err
	}
	if f.statusCode == 0 {
		f.statusCode = http.StatusOK
	}
	return &http.Response{
		StatusCode: f.statusCode,
		Body:       http.NoBody,
		Header:     make(http.Header),
	}, nil
}

func TestPaymentGatewayServiceGenerateCreatesPendingTransaction(t *testing.T) {
	tokoRepo := &fakeTokoRepo{byID: map[uint64]*model.Toko{
		55: {ID: 55, Token: "toko-token-1", Name: "Toko A"},
	}}
	trxRepo := &fakeTransactionRepo{}
	gateway := &fakePaymentGatewayClient{generateResp: &paymentgateway.GenerateResponse{Data: "qr-data", TrxID: "trx-123"}}

	svc := NewPaymentGatewayService(gateway, tokoRepo, trxRepo, nil, "dewifork", "key", "merchant-uuid", "", 3, slog.Default(), nil, false, nil)
	result, err := svc.Generate(context.Background(), 55, GeneratePaymentInput{
		Username: "player-1",
		Amount:   10000,
	})

	require.NoError(t, err)
	require.Equal(t, "trx-123", result.TrxID)
	require.Len(t, trxRepo.created, 1)
	require.Equal(t, model.TransactionStatusPending, trxRepo.created[0].Status)
	require.Equal(t, model.TransactionTypeDeposit, trxRepo.created[0].Type)
	require.Equal(t, uint64(55), trxRepo.created[0].TokoID)
	require.Equal(t, uint64(0), trxRepo.created[0].PlatformFee)
	require.Equal(t, "merchant-uuid", gateway.generateReq.UUID)
}

func TestPaymentGatewayServiceProcessQrisCallbackApplyPlatformFee(t *testing.T) {
	trxRepo := &fakeTransactionRepo{
		byReference: map[string]*model.Transaction{
			"trx-abc": {
				ID:        99,
				TokoID:    55,
				Type:      model.TransactionTypeDeposit,
				Status:    model.TransactionStatusPending,
				Amount:    100000,
				Netto:     100000,
				CreatedAt: time.Now().UTC(),
			},
		},
	}
	svc := NewPaymentGatewayService(
		&fakePaymentGatewayClient{},
		&fakeTokoRepo{byID: map[uint64]*model.Toko{
			55: {ID: 55, Token: "toko-token-55"},
		}},
		trxRepo,
		nil,
		"dewifork",
		"key",
		"merchant-uuid",
		"",
		3,
		slog.Default(),
		nil,
		false,
		nil,
	)

	err := svc.ProcessQrisCallback(context.Background(), queue.QrisCallbackTaskPayload{
		Amount:     100000,
		TerminalID: "T1",
		MerchantID: "merchant-uuid",
		TrxID:      "trx-abc",
		RRN:        "rrn-123",
		Vendor:     "vendor",
		Status:     "success",
		CreatedAt:  "2026-03-21 10:00:00",
		FinishAt:   "2026-03-21 10:00:05",
	})

	require.NoError(t, err)
	require.Len(t, trxRepo.updates, 1)
	require.Equal(t, uint64(99), trxRepo.updates[0].id)
	require.Equal(t, model.TransactionStatusSuccess, trxRepo.updates[0].status)
	require.Equal(t, uint64(3000), trxRepo.updates[0].platformFee)
	require.Equal(t, uint64(97000), trxRepo.updates[0].netto)
}

func TestPaymentGatewayServiceEnqueueQrisCallback(t *testing.T) {
	producer := &fakeGatewayProducer{}
	svc := NewPaymentGatewayService(
		&fakePaymentGatewayClient{},
		&fakeTokoRepo{},
		&fakeTransactionRepo{},
		producer,
		"dewifork",
		"key",
		"merchant-uuid",
		"",
		3,
		slog.Default(),
		nil,
		false,
		nil,
	)

	err := svc.EnqueueQrisCallback(context.Background(), queue.QrisCallbackTaskPayload{
		Amount:     25000,
		TerminalID: "T1",
		MerchantID: "merchant-uuid",
		TrxID:      "trx-queue",
		RRN:        "rrn-queue",
		Vendor:     "vendor",
		Status:     "pending",
		CreatedAt:  "2026-03-21 10:00:00",
		FinishAt:   "2026-03-21 10:00:05",
	})

	require.NoError(t, err)
	require.Len(t, producer.qrisCallbacks, 1)
	require.Equal(t, "trx-queue", producer.qrisCallbacks[0].TrxID)
}

func TestPaymentGatewayServiceValidateCallbackSecret(t *testing.T) {
	svc := NewPaymentGatewayService(&fakePaymentGatewayClient{}, &fakeTokoRepo{}, &fakeTransactionRepo{}, nil, "", "", "merchant-uuid", "secret", 3, slog.Default(), nil, false, nil)

	err := svc.ValidateCallbackSecret("wrong")
	require.Error(t, err)

	err = svc.ValidateCallbackSecret("secret")
	require.NoError(t, err)
}

func TestPaymentGatewayServiceCheckStatusUsesMemcachedByTrxID(t *testing.T) {
	cacheStore := newFakePaymentCache()
	require.NoError(t, cacheStore.Set(context.Background(), "trx-cache-1", qrisStatusCacheEntry{
		Amount:     25000,
		MerchantID: "merchant-uuid",
		TrxID:      "trx-cache-1",
		Status:     "success",
		CreatedAt:  "2026-03-21T10:00:00Z",
		FinishAt:   "2026-03-21T10:05:00Z",
	}, time.Minute))

	trxRepo := &fakeTransactionRepo{
		byReference: map[string]*model.Transaction{
			"trx-cache-1": {
				ID:        11,
				TokoID:    55,
				Status:    model.TransactionStatusPending,
				Type:      model.TransactionTypeDeposit,
				Amount:    25000,
				Netto:     25000,
				CreatedAt: time.Now().UTC(),
			},
		},
	}
	gateway := &fakePaymentGatewayClient{}
	svc := NewPaymentGatewayService(
		gateway,
		&fakeTokoRepo{byID: map[uint64]*model.Toko{
			55: {ID: 55, Token: "toko-token-55"},
		}},
		trxRepo,
		nil,
		"dewifork",
		"key",
		"merchant-uuid",
		"",
		3,
		slog.Default(),
		cacheStore,
		true,
		nil,
	)

	result, err := svc.CheckStatusV2(context.Background(), 55, "trx-cache-1", CheckPaymentStatusInput{})
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)
	require.Equal(t, 0, gateway.checkCalls)
	require.Len(t, trxRepo.updates, 1)
	require.Equal(t, model.TransactionStatusSuccess, trxRepo.updates[0].status)
}

func TestPaymentGatewayServiceProcessQrisCallbackCachesAndForwardsMerchantCallback(t *testing.T) {
	callbackURL := "https://merchant.example.com/callback"
	cacheStore := newFakePaymentCache()
	callbackClient := &fakeCallbackHTTPClient{}
	trxRepo := &fakeTransactionRepo{
		byReference: map[string]*model.Transaction{
			"trx-forward-1": {
				ID:        20,
				TokoID:    9,
				Status:    model.TransactionStatusPending,
				Type:      model.TransactionTypeDeposit,
				Amount:    100000,
				Netto:     100000,
				CreatedAt: time.Now().UTC(),
			},
		},
	}

	svc := NewPaymentGatewayService(
		&fakePaymentGatewayClient{},
		&fakeTokoRepo{byID: map[uint64]*model.Toko{
			9: {ID: 9, Token: "toko-token-9", CallbackURL: &callbackURL},
		}},
		trxRepo,
		nil,
		"dewifork",
		"key",
		"merchant-uuid",
		"",
		3,
		slog.Default(),
		cacheStore,
		true,
		callbackClient,
	)

	err := svc.ProcessQrisCallback(context.Background(), queue.QrisCallbackTaskPayload{
		Amount:     100000,
		TerminalID: "TERM-1",
		MerchantID: "merchant-uuid",
		TrxID:      "trx-forward-1",
		RRN:        "rrn-001",
		CustomRef:  "ORDER001",
		Vendor:     "motpay",
		Status:     "success",
		CreatedAt:  "2026-03-21T10:00:00Z",
		FinishAt:   "2026-03-21T10:00:05Z",
	})
	require.NoError(t, err)
	require.Len(t, trxRepo.updates, 1)
	require.Equal(t, 1, callbackClient.calls)
	require.Equal(t, callbackURL, callbackClient.lastURL)

	cachedBytes, err := cacheStore.Get(context.Background(), "trx-forward-1")
	require.NoError(t, err)
	require.Contains(t, string(cachedBytes), "\"status\":\"success\"")
}

func TestPaymentGatewayServiceExpirePendingTransactionsEnqueuesExpiredCallbacks(t *testing.T) {
	producer := &fakeGatewayProducer{}
	trxRepo := &fakeTransactionRepo{
		expiryCandidates: []repository.PendingExpiryCandidate{
			{
				TransactionID: 44,
				TokoID:        7,
				TokoToken:     "toko-token-7",
				Amount:        50000,
				TrxID:         "trx-exp-1",
				CreatedAt:     time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
			},
		},
	}

	svc := NewPaymentGatewayService(
		&fakePaymentGatewayClient{},
		&fakeTokoRepo{},
		trxRepo,
		producer,
		"dewifork",
		"key",
		"merchant-uuid",
		"",
		3,
		slog.Default(),
		nil,
		false,
		nil,
	)

	count, err := svc.ExpirePendingTransactions(context.Background(), time.Now().UTC(), 100)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Len(t, producer.qrisCallbacks, 1)
	require.Equal(t, "expired", producer.qrisCallbacks[0].Status)
	require.Equal(t, "trx-exp-1", producer.qrisCallbacks[0].TrxID)
}

func pointerToUint64(value uint64) *uint64 {
	return &value
}

func TestPaymentGatewayServiceInquiryTransferCachesInquirySnapshot(t *testing.T) {
	cacheStore := newFakePaymentCache()
	gateway := &fakePaymentGatewayClient{
		inquiryResp: &paymentgateway.InquiryTransferResponse{
			AccountNumber: "1234567890",
			AccountName:   "PT GUE CONTROL",
			BankCode:      "014",
			BankName:      "PT. BANK CENTRAL ASIA, TBK.",
			PartnerRefNo:  "partner-ref-cache",
			Amount:        100000,
			Fee:           1500,
			InquiryID:     77,
		},
	}

	svc := NewPaymentGatewayService(
		gateway,
		&fakeTokoRepo{byID: map[uint64]*model.Toko{5: {ID: 5, Name: "Toko Alpha"}}},
		&fakeTransactionRepo{},
		nil,
		"gue-client",
		"gue-key",
		"merchant-uuid",
		"",
		3,
		slog.Default(),
		cacheStore,
		true,
		nil,
	)

	result, err := svc.InquiryTransfer(context.Background(), 5, InquiryTransferInput{
		Amount:        100000,
		BankCode:      "014",
		AccountNumber: "1234567890",
		Type:          2,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(77), result.InquiryID)

	cached, ok := svc.getCachedTransferInquiry(context.Background(), 5, "014", "1234567890", 100000, 77)
	require.True(t, ok)
	require.NotNil(t, cached)
	require.Equal(t, "partner-ref-cache", cached.PartnerRefNo)
}

func TestPaymentGatewayServiceTransferFundCreatesPendingWithdrawFromCachedInquiry(t *testing.T) {
	cacheStore := newFakePaymentCache()
	trxRepo := &fakeTransactionRepo{byReference: map[string]*model.Transaction{}}
	gateway := &fakePaymentGatewayClient{
		transferResp: &paymentgateway.TransferFundResponse{Status: true},
	}

	svc := NewPaymentGatewayService(
		gateway,
		&fakeTokoRepo{byID: map[uint64]*model.Toko{5: {ID: 5, Name: "Toko Alpha"}}},
		trxRepo,
		nil,
		"gue-client",
		"gue-key",
		"merchant-uuid",
		"",
		3,
		slog.Default(),
		cacheStore,
		true,
		nil,
	)
	svc.cacheTransferInquiry(context.Background(), 5, &paymentgateway.InquiryTransferResponse{
		AccountNumber: "1234567890",
		AccountName:   "PT GUE CONTROL",
		BankCode:      "014",
		BankName:      "PT. BANK CENTRAL ASIA, TBK.",
		PartnerRefNo:  "partner-ref-transfer",
		Amount:        100000,
		Fee:           1500,
		InquiryID:     77,
	})

	result, err := svc.TransferFund(context.Background(), 5, TransferFundInput{
		Amount:        100000,
		BankCode:      "014",
		AccountNumber: "1234567890",
		Type:          2,
		InquiryID:     77,
	})
	require.NoError(t, err)
	require.True(t, result.Status)
	require.Len(t, trxRepo.created, 1)
	require.Equal(t, uint64(100000), trxRepo.created[0].Amount)
	require.Equal(t, uint64(100000), trxRepo.created[0].Netto)
	require.Equal(t, uint64(100000), gateway.transferReq.Amount)
	require.Equal(t, "014", gateway.transferReq.BankCode)
}

func TestPaymentGatewayServiceProcessTransferCallbackFailedReleasesPendingWithdraw(t *testing.T) {
	reference := "partner-ref-failed"
	trxRepo := &fakeTransactionRepo{
		byReference: map[string]*model.Transaction{
			reference: {
				ID:            45,
				TokoID:        5,
				Type:          model.TransactionTypeWithdraw,
				Status:        model.TransactionStatusPending,
				Amount:        100000,
				FeeWithdrawal: pointerToUint64(1500),
				Reference:     &reference,
			},
		},
	}
	svc := NewPaymentGatewayService(
		&fakePaymentGatewayClient{},
		&fakeTokoRepo{},
		trxRepo,
		nil,
		"gue-client",
		"gue-key",
		"merchant-uuid",
		"",
		3,
		slog.Default(),
		nil,
		false,
		nil,
	)

	err := svc.ProcessTransferCallback(context.Background(), queue.TransferCallbackTaskPayload{
		Amount:          100000,
		PartnerRefNo:    reference,
		Status:          "failed",
		TransactionDate: "2026-03-22T10:00:00Z",
		MerchantID:      "merchant-uuid",
	})
	require.NoError(t, err)
	require.Len(t, trxRepo.withdrawFinalized, 1)
	require.Equal(t, model.TransactionStatusFailed, trxRepo.withdrawFinalized[0].status)
}
