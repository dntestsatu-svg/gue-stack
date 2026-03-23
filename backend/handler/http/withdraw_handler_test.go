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
	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mockWithdrawUseCase struct {
	optionsFn  func(ctx context.Context, userID uint64, actorRole model.UserRole) (*service.WithdrawOptionsResult, error)
	historyFn  func(ctx context.Context, userID uint64, actorRole model.UserRole, query service.WithdrawHistoryQuery) (*service.WithdrawHistoryPage, error)
	inquiryFn  func(ctx context.Context, userID uint64, actorRole model.UserRole, input service.WithdrawInquiryInput) (*service.WithdrawInquiryResult, error)
	transferFn func(ctx context.Context, userID uint64, actorRole model.UserRole, input service.WithdrawTransferInput) (*service.WithdrawTransferResult, error)
}

func (m *mockWithdrawUseCase) Options(ctx context.Context, userID uint64, actorRole model.UserRole) (*service.WithdrawOptionsResult, error) {
	if m.optionsFn == nil {
		return nil, nil
	}
	return m.optionsFn(ctx, userID, actorRole)
}

func (m *mockWithdrawUseCase) Inquiry(ctx context.Context, userID uint64, actorRole model.UserRole, input service.WithdrawInquiryInput) (*service.WithdrawInquiryResult, error) {
	if m.inquiryFn == nil {
		return nil, nil
	}
	return m.inquiryFn(ctx, userID, actorRole, input)
}

func (m *mockWithdrawUseCase) History(ctx context.Context, userID uint64, actorRole model.UserRole, query service.WithdrawHistoryQuery) (*service.WithdrawHistoryPage, error) {
	if m.historyFn == nil {
		return nil, nil
	}
	return m.historyFn(ctx, userID, actorRole, query)
}

func (m *mockWithdrawUseCase) Transfer(ctx context.Context, userID uint64, actorRole model.UserRole, input service.WithdrawTransferInput) (*service.WithdrawTransferResult, error) {
	if m.transferFn == nil {
		return nil, nil
	}
	return m.transferFn(ctx, userID, actorRole, input)
}

func withWithdrawAuthContext(userID uint64, role model.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, userID)
		c.Set(middleware.ContextKeyUserRole, role)
		c.Next()
	}
}

func TestWithdrawHandlerOptionsInquiryTransfer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := &mockWithdrawUseCase{
		optionsFn: func(_ context.Context, userID uint64, actorRole model.UserRole) (*service.WithdrawOptionsResult, error) {
			require.Equal(t, uint64(99), userID)
			require.Equal(t, model.UserRoleAdmin, actorRole)
			return &service.WithdrawOptionsResult{
				Tokos: []service.WithdrawTokoOption{{ID: 7, Name: "Toko Alpha", SettlementBalance: 500000}},
				Banks: []service.WithdrawBankOption{{ID: 9, BankName: "PT. BANK CENTRAL ASIA, TBK.", AccountName: "PT GUE CONTROL", AccountNumber: "1234567890"}},
			}, nil
		},
		historyFn: func(_ context.Context, userID uint64, actorRole model.UserRole, query service.WithdrawHistoryQuery) (*service.WithdrawHistoryPage, error) {
			require.Equal(t, uint64(99), userID)
			require.Equal(t, model.UserRoleAdmin, actorRole)
			require.Equal(t, 10, query.Limit)
			require.Equal(t, 0, query.Offset)
			return &service.WithdrawHistoryPage{
				Items: []service.WithdrawHistoryItem{
					{
						ID:        51,
						TokoID:    7,
						TokoName:  "Toko Alpha",
						Status:    "pending",
						Reference: "partner-ref-1",
						Amount:    100000,
						Netto:     98500,
						CreatedAt: "2026-03-21T10:00:00Z",
					},
				},
				Total:   1,
				Limit:   10,
				Offset:  0,
				HasMore: false,
			}, nil
		},
		inquiryFn: func(_ context.Context, userID uint64, actorRole model.UserRole, input service.WithdrawInquiryInput) (*service.WithdrawInquiryResult, error) {
			require.Equal(t, uint64(99), userID)
			require.Equal(t, model.UserRoleAdmin, actorRole)
			require.Equal(t, uint64(7), input.TokoID)
			require.Equal(t, uint64(9), input.BankID)
			require.Equal(t, uint64(100000), input.Amount)
			return &service.WithdrawInquiryResult{
				TokoID:              7,
				TokoName:            "Toko Alpha",
				BankID:              9,
				BankName:            "PT. BANK CENTRAL ASIA, TBK.",
				AccountName:         "PT GUE CONTROL",
				AccountNumber:       "1234567890",
				Amount:              100000,
				Fee:                 1500,
				InquiryID:           77,
				PartnerRefNo:        "partner-ref-1",
				SettlementBalance:   500000,
				RemainingSettlement: 400000,
			}, nil
		},
		transferFn: func(_ context.Context, userID uint64, actorRole model.UserRole, input service.WithdrawTransferInput) (*service.WithdrawTransferResult, error) {
			require.Equal(t, uint64(99), userID)
			require.Equal(t, model.UserRoleAdmin, actorRole)
			require.Equal(t, uint64(77), input.InquiryID)
			return &service.WithdrawTransferResult{
				Status:              true,
				Message:             "Uangnya akan segera sampai ke bank anda.",
				TokoID:              7,
				TokoName:            "Toko Alpha",
				BankID:              9,
				BankName:            "PT. BANK CENTRAL ASIA, TBK.",
				AccountName:         "PT GUE CONTROL",
				AccountNumber:       "1234567890",
				Amount:              100000,
				RemainingSettlement: 400000,
			}, nil
		},
	}

	h := NewWithdrawHandler(mockUC)
	r := gin.New()
	r.GET("/withdraw/options", withWithdrawAuthContext(99, model.UserRoleAdmin), h.Options)
	r.GET("/withdraw/history", withWithdrawAuthContext(99, model.UserRoleAdmin), h.History)
	r.POST("/withdraw/inquiry", withWithdrawAuthContext(99, model.UserRoleAdmin), h.Inquiry)
	r.POST("/withdraw/transfer", withWithdrawAuthContext(99, model.UserRoleAdmin), h.Transfer)

	optionsReq := httptest.NewRequest(http.MethodGet, "/withdraw/options", nil)
	optionsRes := httptest.NewRecorder()
	r.ServeHTTP(optionsRes, optionsReq)
	require.Equal(t, http.StatusOK, optionsRes.Code)
	require.Contains(t, optionsRes.Body.String(), "Toko Alpha")

	historyReq := httptest.NewRequest(http.MethodGet, "/withdraw/history?limit=10&offset=0", nil)
	historyRes := httptest.NewRecorder()
	r.ServeHTTP(historyRes, historyReq)
	require.Equal(t, http.StatusOK, historyRes.Code)
	require.Contains(t, historyRes.Body.String(), "partner-ref-1")

	inquiryBody, _ := json.Marshal(map[string]any{
		"toko_id": 7,
		"bank_id": 9,
		"amount":  100000,
	})
	inquiryReq := httptest.NewRequest(http.MethodPost, "/withdraw/inquiry", bytes.NewReader(inquiryBody))
	inquiryReq.Header.Set("Content-Type", "application/json")
	inquiryRes := httptest.NewRecorder()
	r.ServeHTTP(inquiryRes, inquiryReq)
	require.Equal(t, http.StatusOK, inquiryRes.Code)
	require.Contains(t, inquiryRes.Body.String(), "PT GUE CONTROL")

	transferBody, _ := json.Marshal(map[string]any{
		"toko_id":    7,
		"bank_id":    9,
		"amount":     100000,
		"inquiry_id": 77,
	})
	transferReq := httptest.NewRequest(http.MethodPost, "/withdraw/transfer", bytes.NewReader(transferBody))
	transferReq.Header.Set("Content-Type", "application/json")
	transferRes := httptest.NewRecorder()
	r.ServeHTTP(transferRes, transferReq)
	require.Equal(t, http.StatusOK, transferRes.Code)
	require.Contains(t, transferRes.Body.String(), "Uangnya akan segera sampai ke bank anda.")
}
