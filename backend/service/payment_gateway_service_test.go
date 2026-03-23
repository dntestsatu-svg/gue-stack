package service

import (
	"context"
	"testing"
	"time"

	"github.com/example/gue/backend/model"
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
}

func (f *fakePaymentGatewayClient) Generate(_ context.Context, req paymentgateway.GenerateRequest) (*paymentgateway.GenerateResponse, error) {
	f.generateReq = req
	return f.generateResp, f.generateErr
}

func (f *fakePaymentGatewayClient) CheckStatusV2(_ context.Context, _ string, _ paymentgateway.CheckStatusRequest) (*paymentgateway.CheckStatusResponse, error) {
	return nil, nil
}

func (f *fakePaymentGatewayClient) InquiryTransfer(_ context.Context, _ paymentgateway.InquiryTransferRequest) (*paymentgateway.InquiryTransferResponse, error) {
	return nil, nil
}

func (f *fakePaymentGatewayClient) TransferFund(_ context.Context, _ paymentgateway.TransferFundRequest) (*paymentgateway.TransferFundResponse, error) {
	return nil, nil
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
	history           []repository.TransactionHistoryRecord
	historyCount      uint64
	lastHistoryFilter repository.TransactionHistoryFilter
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

func TestPaymentGatewayServiceGenerateCreatesPendingTransaction(t *testing.T) {
	tokoRepo := &fakeTokoRepo{byID: map[uint64]*model.Toko{
		55: {ID: 55, Token: "toko-token-1", Name: "Toko A"},
	}}
	trxRepo := &fakeTransactionRepo{}
	gateway := &fakePaymentGatewayClient{generateResp: &paymentgateway.GenerateResponse{Data: "qr-data", TrxID: "trx-123"}}

	svc := NewPaymentGatewayService(gateway, tokoRepo, trxRepo, nil, "dewifork", "key", "merchant-uuid", "", 3, slog.Default())
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
		&fakeTokoRepo{},
		trxRepo,
		nil,
		"dewifork",
		"key",
		"merchant-uuid",
		"",
		3,
		slog.Default(),
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
	svc := NewPaymentGatewayService(&fakePaymentGatewayClient{}, &fakeTokoRepo{}, &fakeTransactionRepo{}, nil, "", "", "merchant-uuid", "secret", 3, slog.Default())

	err := svc.ValidateCallbackSecret("wrong")
	require.Error(t, err)

	err = svc.ValidateCallbackSecret("secret")
	require.NoError(t, err)
}
