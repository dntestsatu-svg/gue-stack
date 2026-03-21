package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/gue/backend/middleware"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/queue"
	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mockPaymentGatewayUseCase struct {
	generateFn               func(ctx context.Context, tokoID uint64, input service.GeneratePaymentInput) (*service.GeneratePaymentResult, error)
	checkStatusFn            func(ctx context.Context, tokoID uint64, trxID string, input service.CheckPaymentStatusInput) (*service.CheckPaymentStatusResult, error)
	inquiryFn                func(ctx context.Context, tokoID uint64, input service.InquiryTransferInput) (*service.InquiryTransferResult, error)
	transferFn               func(ctx context.Context, tokoID uint64, input service.TransferFundInput) (*service.TransferFundResult, error)
	checkTransferStatusFn    func(ctx context.Context, tokoID uint64, partnerRefNo string, input service.CheckTransferStatusInput) (*service.CheckTransferStatusResult, error)
	balanceFn                func(ctx context.Context, input service.GetBalanceInput) (*service.GetBalanceResult, error)
	enqueueQrisFn            func(ctx context.Context, payload queue.QrisCallbackTaskPayload) error
	enqueueTransferFn        func(ctx context.Context, payload queue.TransferCallbackTaskPayload) error
	validateCallbackSecretFn func(secret string) error
}

func (m *mockPaymentGatewayUseCase) Generate(ctx context.Context, tokoID uint64, input service.GeneratePaymentInput) (*service.GeneratePaymentResult, error) {
	if m.generateFn == nil {
		return nil, nil
	}
	return m.generateFn(ctx, tokoID, input)
}

func (m *mockPaymentGatewayUseCase) CheckStatusV2(ctx context.Context, tokoID uint64, trxID string, input service.CheckPaymentStatusInput) (*service.CheckPaymentStatusResult, error) {
	if m.checkStatusFn == nil {
		return nil, nil
	}
	return m.checkStatusFn(ctx, tokoID, trxID, input)
}

func (m *mockPaymentGatewayUseCase) InquiryTransfer(ctx context.Context, tokoID uint64, input service.InquiryTransferInput) (*service.InquiryTransferResult, error) {
	if m.inquiryFn == nil {
		return nil, nil
	}
	return m.inquiryFn(ctx, tokoID, input)
}

func (m *mockPaymentGatewayUseCase) TransferFund(ctx context.Context, tokoID uint64, input service.TransferFundInput) (*service.TransferFundResult, error) {
	if m.transferFn == nil {
		return nil, nil
	}
	return m.transferFn(ctx, tokoID, input)
}

func (m *mockPaymentGatewayUseCase) CheckTransferStatus(ctx context.Context, tokoID uint64, partnerRefNo string, input service.CheckTransferStatusInput) (*service.CheckTransferStatusResult, error) {
	if m.checkTransferStatusFn == nil {
		return nil, nil
	}
	return m.checkTransferStatusFn(ctx, tokoID, partnerRefNo, input)
}

func (m *mockPaymentGatewayUseCase) GetBalance(ctx context.Context, input service.GetBalanceInput) (*service.GetBalanceResult, error) {
	if m.balanceFn == nil {
		return nil, nil
	}
	return m.balanceFn(ctx, input)
}

func (m *mockPaymentGatewayUseCase) EnqueueQrisCallback(ctx context.Context, payload queue.QrisCallbackTaskPayload) error {
	if m.enqueueQrisFn == nil {
		return nil
	}
	return m.enqueueQrisFn(ctx, payload)
}

func (m *mockPaymentGatewayUseCase) EnqueueTransferCallback(ctx context.Context, payload queue.TransferCallbackTaskPayload) error {
	if m.enqueueTransferFn == nil {
		return nil
	}
	return m.enqueueTransferFn(ctx, payload)
}

func (m *mockPaymentGatewayUseCase) ValidateCallbackSecret(secret string) error {
	if m.validateCallbackSecretFn == nil {
		return nil
	}
	return m.validateCallbackSecretFn(secret)
}

func withTokoID(tokoID uint64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.ContextKeyTokoID, tokoID)
		c.Next()
	}
}

func TestPaymentGatewayHandlerGenerate_TableDriven(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		withContext    bool
		body           map[string]any
		useCaseErr     error
		expectedStatus int
	}{
		{
			name:        "success",
			withContext: true,
			body: map[string]any{
				"username": "player-1",
				"amount":   10000,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing toko context",
			withContext:    false,
			body:           map[string]any{"username": "player-1", "amount": 10000},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid payload",
			withContext:    true,
			body:           map[string]any{"username": "", "amount": 0},
			useCaseErr:     apperror.New(http.StatusBadRequest, "invalid request payload", nil),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "service error",
			withContext:    true,
			body:           map[string]any{"username": "player-1", "amount": 10000},
			useCaseErr:     apperror.New(http.StatusBadGateway, "upstream error", nil),
			expectedStatus: http.StatusBadGateway,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &mockPaymentGatewayUseCase{
				generateFn: func(_ context.Context, tokoID uint64, _ service.GeneratePaymentInput) (*service.GeneratePaymentResult, error) {
					require.Equal(t, uint64(99), tokoID)
					if tt.useCaseErr != nil {
						return nil, tt.useCaseErr
					}
					return &service.GeneratePaymentResult{TrxID: "trx-1", Data: "qr-data"}, nil
				},
			}

			h := NewPaymentGatewayHandler(mockUC)
			r := gin.New()
			if tt.withContext {
				r.POST("/payments/generate", withTokoID(99), h.Generate)
			} else {
				r.POST("/payments/generate", h.Generate)
			}

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/payments/generate", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestPaymentGatewayHandlerQrisCallbackSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := &mockPaymentGatewayUseCase{
		validateCallbackSecretFn: func(secret string) error {
			if secret != "valid-secret" {
				return apperror.New(http.StatusUnauthorized, "invalid callback secret", nil)
			}
			return nil
		},
		enqueueQrisFn: func(_ context.Context, _ queue.QrisCallbackTaskPayload) error {
			return nil
		},
	}
	h := NewPaymentGatewayHandler(mockUC)
	r := gin.New()
	r.POST("/callbacks/qris", h.QrisCallback)

	callbackBody := []byte(`{"amount":10000,"terminal_id":"t1","merchant_id":"m1","trx_id":"trx","rrn":"rrn","vendor":"v","status":"success","created_at":"2026-03-21 10:00:00","finish_at":"2026-03-21 10:00:10"}`)

	invalidReq := httptest.NewRequest(http.MethodPost, "/callbacks/qris", bytes.NewReader(callbackBody))
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidReq.Header.Set("X-Callback-Secret", "wrong")
	invalidRes := httptest.NewRecorder()
	r.ServeHTTP(invalidRes, invalidReq)
	require.Equal(t, http.StatusUnauthorized, invalidRes.Code)

	validReq := httptest.NewRequest(http.MethodPost, "/callbacks/qris", bytes.NewReader(callbackBody))
	validReq.Header.Set("Content-Type", "application/json")
	validReq.Header.Set("X-Callback-Secret", "valid-secret")
	validRes := httptest.NewRecorder()
	r.ServeHTTP(validRes, validReq)
	require.Equal(t, http.StatusAccepted, validRes.Code)
}
