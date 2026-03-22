package service

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/queue"
	"github.com/example/gue/backend/repository"
	"github.com/stretchr/testify/require"
)

type fakeTestingTokoRepo struct {
	toko *model.Toko
	err  error
}

func (f *fakeTestingTokoRepo) Create(_ context.Context, _ *model.Toko) error { return nil }
func (f *fakeTestingTokoRepo) CreateForUserWithQuota(_ context.Context, _ uint64, _ *model.Toko, _ int) error {
	return nil
}
func (f *fakeTestingTokoRepo) AttachUser(_ context.Context, _, _ uint64) error      { return nil }
func (f *fakeTestingTokoRepo) CountByUser(_ context.Context, _ uint64) (int, error) { return 0, nil }
func (f *fakeTestingTokoRepo) ListByUser(_ context.Context, _ uint64, _ model.UserRole) ([]model.Toko, error) {
	return nil, nil
}
func (f *fakeTestingTokoRepo) ListWorkspaceByUser(_ context.Context, _ uint64, _ model.UserRole, _ repository.TokoWorkspaceFilter) ([]repository.TokoWorkspaceRecord, error) {
	return nil, nil
}
func (f *fakeTestingTokoRepo) SummarizeWorkspaceByUser(_ context.Context, _ uint64, _ model.UserRole, _ repository.TokoWorkspaceFilter) (*repository.TokoWorkspaceSummary, error) {
	return nil, nil
}
func (f *fakeTestingTokoRepo) GetByID(_ context.Context, _ uint64) (*model.Toko, error) {
	return f.toko, f.err
}
func (f *fakeTestingTokoRepo) GetAccessibleByID(_ context.Context, _ uint64, _ model.UserRole, _ uint64) (*model.Toko, error) {
	return f.toko, f.err
}
func (f *fakeTestingTokoRepo) GetByToken(_ context.Context, _ string) (*model.Toko, error) {
	return f.toko, f.err
}
func (f *fakeTestingTokoRepo) UpdateProfile(_ context.Context, _ uint64, _ string, _ *string) error {
	return nil
}
func (f *fakeTestingTokoRepo) UpdateToken(_ context.Context, _ uint64, _ string) error { return nil }

type fakeTestingPaymentGateway struct {
	result *GeneratePaymentResult
	err    error
	tokoID uint64
	input  GeneratePaymentInput
}

func (f *fakeTestingPaymentGateway) Generate(_ context.Context, tokoID uint64, input GeneratePaymentInput) (*GeneratePaymentResult, error) {
	f.tokoID = tokoID
	f.input = input
	return f.result, f.err
}
func (f *fakeTestingPaymentGateway) CheckStatusV2(_ context.Context, _ uint64, _ string, _ CheckPaymentStatusInput) (*CheckPaymentStatusResult, error) {
	return nil, nil
}
func (f *fakeTestingPaymentGateway) InquiryTransfer(_ context.Context, _ uint64, _ InquiryTransferInput) (*InquiryTransferResult, error) {
	return nil, nil
}
func (f *fakeTestingPaymentGateway) TransferFund(_ context.Context, _ uint64, _ TransferFundInput) (*TransferFundResult, error) {
	return nil, nil
}
func (f *fakeTestingPaymentGateway) CheckTransferStatus(_ context.Context, _ uint64, _ string, _ CheckTransferStatusInput) (*CheckTransferStatusResult, error) {
	return nil, nil
}
func (f *fakeTestingPaymentGateway) GetBalance(_ context.Context, _ GetBalanceInput) (*GetBalanceResult, error) {
	return nil, nil
}
func (f *fakeTestingPaymentGateway) EnqueueQrisCallback(_ context.Context, _ queue.QrisCallbackTaskPayload) error {
	return nil
}
func (f *fakeTestingPaymentGateway) EnqueueTransferCallback(_ context.Context, _ queue.TransferCallbackTaskPayload) error {
	return nil
}
func (f *fakeTestingPaymentGateway) ValidateCallbackSecret(_ string) error { return nil }

type testingCallbackClient struct {
	client *http.Client
}

func (c testingCallbackClient) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

func TestTestingServiceGenerateQrisUsesAccessibleToko(t *testing.T) {
	callbackURL := "https://merchant.example.com/callback"
	tokoRepo := &fakeTestingTokoRepo{
		toko: &model.Toko{
			ID:          8,
			Name:        "Toko Alpha",
			CallbackURL: &callbackURL,
		},
	}
	paymentGateway := &fakeTestingPaymentGateway{
		result: &GeneratePaymentResult{
			Data:  "qr-data",
			TrxID: "trx-001",
		},
	}
	service := NewTestingService(tokoRepo, paymentGateway, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := service.GenerateQris(context.Background(), 44, model.UserRoleAdmin, TestingGenerateQrisInput{
		TokoID:    8,
		Username:  "player-1",
		Amount:    25000,
		CustomRef: "ABC123",
	})
	require.NoError(t, err)
	require.Equal(t, uint64(8), paymentGateway.tokoID)
	require.Equal(t, "player-1", paymentGateway.input.Username)
	require.Equal(t, uint64(25000), paymentGateway.input.Amount)
	require.Equal(t, "Toko Alpha", result.TokoName)
	require.Equal(t, "trx-001", result.TrxID)
	require.GreaterOrEqual(t, result.ServerProcessingMS, int64(0))
}

func TestTestingServiceCheckCallbackReadinessReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	tokoRepo := &fakeTestingTokoRepo{
		toko: &model.Toko{
			ID:          9,
			Name:        "Toko Bravo",
			CallbackURL: &server.URL,
		},
	}
	service := NewTestingService(
		tokoRepo,
		&fakeTestingPaymentGateway{},
		testingCallbackClient{client: server.Client()},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	result, err := service.CheckCallbackReadiness(context.Background(), 1, model.UserRoleUser, TestingCallbackReadinessInput{TokoID: 9})
	require.NoError(t, err)
	require.True(t, result.Ready)
	require.Equal(t, "API kamu sudah ready.", result.Message)
	require.Equal(t, http.StatusOK, result.StatusCode)
	require.True(t, result.ReceivedSuccess)
	require.GreaterOrEqual(t, result.CallbackLatencyMS, int64(0))
	require.GreaterOrEqual(t, result.ServerProcessingMS, int64(0))
}

func TestTestingServiceCheckCallbackReadinessNotReadyOnUnexpectedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success": false}`))
	}))
	defer server.Close()

	tokoRepo := &fakeTestingTokoRepo{
		toko: &model.Toko{
			ID:          10,
			Name:        "Toko Charlie",
			CallbackURL: &server.URL,
		},
	}
	service := NewTestingService(
		tokoRepo,
		&fakeTestingPaymentGateway{},
		testingCallbackClient{client: server.Client()},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	result, err := service.CheckCallbackReadiness(context.Background(), 2, model.UserRoleAdmin, TestingCallbackReadinessInput{TokoID: 10})
	require.NoError(t, err)
	require.False(t, result.Ready)
	require.Equal(t, "API kamu sepertinya belum terintegrasi dengan baik.", result.Message)
	require.Equal(t, http.StatusAccepted, result.StatusCode)
	require.False(t, result.ReceivedSuccess)
	require.Contains(t, result.Detail, "HTTP 200")
}

func TestTestingServiceCheckCallbackReadinessHandlesMissingCallbackURL(t *testing.T) {
	tokoRepo := &fakeTestingTokoRepo{
		toko: &model.Toko{
			ID:   11,
			Name: "Toko Delta",
		},
	}
	service := NewTestingService(tokoRepo, &fakeTestingPaymentGateway{}, &http.Client{Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := service.CheckCallbackReadiness(context.Background(), 3, model.UserRoleAdmin, TestingCallbackReadinessInput{TokoID: 11})
	require.NoError(t, err)
	require.False(t, result.Ready)
	require.Contains(t, result.Detail, "belum dikonfigurasi")
}
