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
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mockBankUseCase struct {
	listFn           func(ctx context.Context, userID uint64, actorRole model.UserRole, query service.BankListQuery) (*service.BankListPage, error)
	inquiryFn        func(ctx context.Context, userID uint64, actorRole model.UserRole, input service.BankInquiryInput) (*service.BankInquiryResult, error)
	createFn         func(ctx context.Context, userID uint64, actorRole model.UserRole, input service.CreateBankInput) (*service.BankDTO, error)
	deleteFn         func(ctx context.Context, userID uint64, actorRole model.UserRole, bankID uint64) error
	paymentOptionsFn func(ctx context.Context, actorRole model.UserRole, query service.PaymentOptionQuery) ([]service.PaymentOptionDTO, error)
}

func (m *mockBankUseCase) List(ctx context.Context, userID uint64, actorRole model.UserRole, query service.BankListQuery) (*service.BankListPage, error) {
	if m.listFn == nil {
		return nil, nil
	}
	return m.listFn(ctx, userID, actorRole, query)
}

func (m *mockBankUseCase) Inquiry(ctx context.Context, userID uint64, actorRole model.UserRole, input service.BankInquiryInput) (*service.BankInquiryResult, error) {
	if m.inquiryFn == nil {
		return nil, nil
	}
	return m.inquiryFn(ctx, userID, actorRole, input)
}

func (m *mockBankUseCase) Create(ctx context.Context, userID uint64, actorRole model.UserRole, input service.CreateBankInput) (*service.BankDTO, error) {
	if m.createFn == nil {
		return nil, nil
	}
	return m.createFn(ctx, userID, actorRole, input)
}

func (m *mockBankUseCase) Delete(ctx context.Context, userID uint64, actorRole model.UserRole, bankID uint64) error {
	if m.deleteFn == nil {
		return nil
	}
	return m.deleteFn(ctx, userID, actorRole, bankID)
}

func (m *mockBankUseCase) PaymentOptions(ctx context.Context, actorRole model.UserRole, query service.PaymentOptionQuery) ([]service.PaymentOptionDTO, error) {
	if m.paymentOptionsFn == nil {
		return nil, nil
	}
	return m.paymentOptionsFn(ctx, actorRole, query)
}

func withBankAuthContext(userID uint64, role model.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, userID)
		c.Set(middleware.ContextKeyUserRole, role)
		c.Next()
	}
}

func TestBankHandlerListAndPaymentOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := &mockBankUseCase{
		listFn: func(_ context.Context, userID uint64, actorRole model.UserRole, query service.BankListQuery) (*service.BankListPage, error) {
			require.Equal(t, uint64(77), userID)
			require.Equal(t, model.UserRoleAdmin, actorRole)
			require.Equal(t, 15, query.Limit)
			require.Equal(t, 5, query.Offset)
			require.Equal(t, "bca", query.SearchTerm)
			return &service.BankListPage{
				Items: []service.BankDTO{
					{ID: 1, PaymentID: 8, BankName: "PT. BANK CENTRAL ASIA, TBK.", AccountName: "PT GUE", AccountNumber: "123"},
				},
				Total:   1,
				Limit:   15,
				Offset:  5,
				HasMore: false,
			}, nil
		},
		paymentOptionsFn: func(_ context.Context, actorRole model.UserRole, query service.PaymentOptionQuery) ([]service.PaymentOptionDTO, error) {
			require.Equal(t, model.UserRoleAdmin, actorRole)
			require.Equal(t, 12, query.Limit)
			require.Equal(t, "mandiri", query.SearchTerm)
			return []service.PaymentOptionDTO{
				{ID: 1, BankName: "PT. BANK MANDIRI (PERSERO), TBK."},
			}, nil
		},
	}

	h := NewBankHandler(mockUC)
	r := gin.New()
	r.GET("/banks", withBankAuthContext(77, model.UserRoleAdmin), h.List)
	r.GET("/banks/payment-options", withBankAuthContext(77, model.UserRoleAdmin), h.PaymentOptions)

	listReq := httptest.NewRequest(http.MethodGet, "/banks?limit=15&offset=5&q=bca", nil)
	listRes := httptest.NewRecorder()
	r.ServeHTTP(listRes, listReq)
	require.Equal(t, http.StatusOK, listRes.Code)
	require.Contains(t, listRes.Body.String(), "PT. BANK CENTRAL ASIA, TBK.")

	optionsReq := httptest.NewRequest(http.MethodGet, "/banks/payment-options?limit=12&q=mandiri", nil)
	optionsRes := httptest.NewRecorder()
	r.ServeHTTP(optionsRes, optionsReq)
	require.Equal(t, http.StatusOK, optionsRes.Code)
	require.Contains(t, optionsRes.Body.String(), "MANDIRI")
}

func TestBankHandlerCreateAndDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUC := &mockBankUseCase{
		inquiryFn: func(_ context.Context, userID uint64, actorRole model.UserRole, input service.BankInquiryInput) (*service.BankInquiryResult, error) {
			require.Equal(t, uint64(9), userID)
			require.Equal(t, model.UserRoleSuperAdmin, actorRole)
			require.Equal(t, uint64(8), input.PaymentID)
			require.Equal(t, "1234567890", input.AccountNumber)
			return &service.BankInquiryResult{
				PaymentID:     8,
				AccountNumber: "1234567890",
				AccountName:   "PT GUE CONTROL",
				BankCode:      "014",
				BankName:      "PT. BANK CENTRAL ASIA, TBK.",
				InquiryID:     88,
			}, nil
		},
		createFn: func(_ context.Context, userID uint64, actorRole model.UserRole, input service.CreateBankInput) (*service.BankDTO, error) {
			require.Equal(t, uint64(9), userID)
			require.Equal(t, model.UserRoleSuperAdmin, actorRole)
			require.Equal(t, uint64(8), input.PaymentID)
			require.Equal(t, "PT GUE CONTROL", input.AccountName)
			require.Equal(t, "1234567890", input.AccountNumber)
			require.Equal(t, uint64(88), input.InquiryID)
			return &service.BankDTO{
				ID:            5,
				PaymentID:     8,
				BankName:      "PT. BANK CENTRAL ASIA, TBK.",
				AccountName:   input.AccountName,
				AccountNumber: input.AccountNumber,
			}, nil
		},
		deleteFn: func(_ context.Context, userID uint64, actorRole model.UserRole, bankID uint64) error {
			require.Equal(t, uint64(9), userID)
			require.Equal(t, model.UserRoleSuperAdmin, actorRole)
			require.Equal(t, uint64(5), bankID)
			return nil
		},
	}

	h := NewBankHandler(mockUC)
	r := gin.New()
	r.POST("/banks/inquiry", withBankAuthContext(9, model.UserRoleSuperAdmin), h.Inquiry)
	r.POST("/banks", withBankAuthContext(9, model.UserRoleSuperAdmin), h.Create)
	r.DELETE("/banks/:id", withBankAuthContext(9, model.UserRoleSuperAdmin), h.Delete)

	inquiryPayload := map[string]any{
		"payment_id":     8,
		"account_number": "1234567890",
	}
	inquiryBody, _ := json.Marshal(inquiryPayload)
	inquiryReq := httptest.NewRequest(http.MethodPost, "/banks/inquiry", bytes.NewReader(inquiryBody))
	inquiryReq.Header.Set("Content-Type", "application/json")
	inquiryRes := httptest.NewRecorder()
	r.ServeHTTP(inquiryRes, inquiryReq)
	require.Equal(t, http.StatusOK, inquiryRes.Code)

	payload := map[string]any{
		"payment_id":     8,
		"account_name":   "PT GUE CONTROL",
		"account_number": "1234567890",
		"inquiry_id":     88,
	}
	body, _ := json.Marshal(payload)
	createReq := httptest.NewRequest(http.MethodPost, "/banks", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	r.ServeHTTP(createRes, createReq)
	require.Equal(t, http.StatusCreated, createRes.Code)

	deleteReq := httptest.NewRequest(http.MethodDelete, "/banks/5", nil)
	deleteRes := httptest.NewRecorder()
	r.ServeHTTP(deleteRes, deleteReq)
	require.Equal(t, http.StatusOK, deleteRes.Code)
	require.Contains(t, deleteRes.Body.String(), "bank deleted successfully")
}

func TestBankHandlerCreateValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewBankHandler(&mockBankUseCase{
		createFn: func(_ context.Context, _ uint64, _ model.UserRole, _ service.CreateBankInput) (*service.BankDTO, error) {
			return nil, apperror.New(http.StatusBadRequest, "invalid request payload", "payment_id is required")
		},
	})
	r := gin.New()
	r.POST("/banks", withBankAuthContext(9, model.UserRoleAdmin), h.Create)

	req := httptest.NewRequest(http.MethodPost, "/banks", bytes.NewReader([]byte(`{"payment_id":0}`)))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	require.Equal(t, http.StatusBadRequest, res.Code)
}
