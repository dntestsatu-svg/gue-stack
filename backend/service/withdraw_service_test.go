package service

import (
	"context"
	"testing"
	"time"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/paymentgateway"
	"github.com/example/gue/backend/repository"
	"github.com/stretchr/testify/require"
	"log/slog"
)

type fakeWithdrawGatewayClient struct {
	inquiryReq  paymentgateway.InquiryTransferRequest
	inquiryResp *paymentgateway.InquiryTransferResponse
	inquiryErr  error

	transferReq  paymentgateway.TransferFundRequest
	transferResp *paymentgateway.TransferFundResponse
	transferErr  error
}

func (f *fakeWithdrawGatewayClient) Generate(_ context.Context, _ paymentgateway.GenerateRequest) (*paymentgateway.GenerateResponse, error) {
	return nil, nil
}

func (f *fakeWithdrawGatewayClient) CheckStatusV2(_ context.Context, _ string, _ paymentgateway.CheckStatusRequest) (*paymentgateway.CheckStatusResponse, error) {
	return nil, nil
}

func (f *fakeWithdrawGatewayClient) InquiryTransfer(_ context.Context, req paymentgateway.InquiryTransferRequest) (*paymentgateway.InquiryTransferResponse, error) {
	f.inquiryReq = req
	return f.inquiryResp, f.inquiryErr
}

func (f *fakeWithdrawGatewayClient) TransferFund(_ context.Context, req paymentgateway.TransferFundRequest) (*paymentgateway.TransferFundResponse, error) {
	f.transferReq = req
	return f.transferResp, f.transferErr
}

func (f *fakeWithdrawGatewayClient) CheckTransferStatus(_ context.Context, _ string, _ paymentgateway.CheckTransferStatusRequest) (*paymentgateway.CheckTransferStatusResponse, error) {
	return nil, nil
}

func (f *fakeWithdrawGatewayClient) GetBalance(_ context.Context, _ string, _ paymentgateway.GetBalanceRequest) (*paymentgateway.GetBalanceResponse, error) {
	return nil, nil
}

func TestWithdrawServiceOptionsReturnsScopedTokosAndBanks(t *testing.T) {
	svc := NewWithdrawService(
		&fakeTokoDomainRepo{},
		&fakeBalanceRepo{byTokoID: map[uint64]repository.TokoBalanceRecord{
			7: {TokoID: 7, TokoName: "Toko Alpha", SettlementBalance: 500000, AvailableBalance: 900000, LastSettlementTime: time.Now().UTC()},
		}},
		&fakeBankRepo{items: map[uint64][]model.Bank{
			11: {
				{ID: 9, UserID: 11, PaymentID: 1, BankCode: "014", BankName: "PT. BANK CENTRAL ASIA, TBK.", AccountName: "PT GUE CONTROL", AccountNumber: "1234567890"},
			},
		}},
		&fakeTransactionRepo{},
		&fakeWithdrawGatewayClient{},
		nil,
		false,
		"client",
		"key",
		"merchant",
		slog.Default(),
	)

	result, err := svc.Options(context.Background(), 11, model.UserRoleAdmin)

	require.NoError(t, err)
	require.Len(t, result.Tokos, 1)
	require.Len(t, result.Banks, 1)
	require.Equal(t, uint64(7), result.Tokos[0].ID)
	require.Equal(t, uint64(9), result.Banks[0].ID)
}

func TestWithdrawServiceInquiryReturnsAccountConfirmationData(t *testing.T) {
	tokoRepo := &fakeTokoDomainRepo{
		byID: map[uint64]*model.Toko{
			5: {ID: 5, Name: "Toko Alpha"},
		},
	}
	balanceRepo := &fakeBalanceRepo{byTokoID: map[uint64]repository.TokoBalanceRecord{
		5: {TokoID: 5, TokoName: "Toko Alpha", SettlementBalance: 500000, AvailableBalance: 1000000, LastSettlementTime: time.Now().UTC()},
	}}
	bankRepo := &fakeBankRepo{items: map[uint64][]model.Bank{
		11: {
			{ID: 9, UserID: 11, PaymentID: 1, BankCode: "014", BankName: "PT. BANK CENTRAL ASIA, TBK.", AccountName: "PT GUE CONTROL", AccountNumber: "1234567890"},
		},
	}}
	gateway := &fakeWithdrawGatewayClient{
		inquiryResp: &paymentgateway.InquiryTransferResponse{
			AccountNumber: "1234567890",
			AccountName:   "PT GUE CONTROL",
			BankCode:      "014",
			BankName:      "PT. BANK CENTRAL ASIA, TBK.",
			PartnerRefNo:  "partner-ref-1",
			VendorRefNo:   "",
			Amount:        100000,
			Fee:           1500,
			InquiryID:     77,
		},
	}
	svc := NewWithdrawService(tokoRepo, balanceRepo, bankRepo, &fakeTransactionRepo{}, gateway, newFakeCacheStore(), true, "gue-client", "gue-key", "merchant-uuid", slog.Default())

	result, err := svc.Inquiry(context.Background(), 11, model.UserRoleAdmin, WithdrawInquiryInput{
		TokoID: 5,
		BankID: 9,
		Amount: 100000,
	})

	require.NoError(t, err)
	require.Equal(t, "PT GUE CONTROL", result.AccountName)
	require.Equal(t, uint64(400000), result.RemainingSettlement)
	require.Equal(t, "gue-client", gateway.inquiryReq.Client)
	require.Equal(t, "gue-key", gateway.inquiryReq.ClientKey)
	require.Equal(t, "merchant-uuid", gateway.inquiryReq.UUID)
	require.Equal(t, "014", gateway.inquiryReq.BankCode)
}

func TestWithdrawServiceTransferCreatesPendingWithdrawAndDeductsSettlement(t *testing.T) {
	tokoRepo := &fakeTokoDomainRepo{
		byID: map[uint64]*model.Toko{
			5: {ID: 5, Name: "Toko Alpha"},
		},
	}
	balanceRepo := &fakeBalanceRepo{byTokoID: map[uint64]repository.TokoBalanceRecord{
		5: {TokoID: 5, TokoName: "Toko Alpha", SettlementBalance: 500000, AvailableBalance: 1000000, LastSettlementTime: time.Now().UTC()},
	}}
	bankRepo := &fakeBankRepo{items: map[uint64][]model.Bank{
		11: {
			{ID: 9, UserID: 11, PaymentID: 1, BankCode: "014", BankName: "PT. BANK CENTRAL ASIA, TBK.", AccountName: "PT GUE CONTROL", AccountNumber: "1234567890"},
		},
	}}
	cacheStore := newFakeCacheStore()
	gateway := &fakeWithdrawGatewayClient{
		transferResp: &paymentgateway.TransferFundResponse{Status: true},
	}
	svc := NewWithdrawService(tokoRepo, balanceRepo, bankRepo, &fakeTransactionRepo{}, gateway, cacheStore, true, "gue-client", "gue-key", "merchant-uuid", slog.Default())

	setCachedJSON(context.Background(), cacheStore, true, svc.inquiryCacheKey(11, 5, 9, 100000), &cachedWithdrawInquiry{
		TokoID:        5,
		TokoName:      "Toko Alpha",
		BankID:        9,
		BankName:      "PT. BANK CENTRAL ASIA, TBK.",
		AccountName:   "PT GUE CONTROL",
		AccountNumber: "1234567890",
		BankCode:      "014",
		Amount:        100000,
		Fee:           1500,
		InquiryID:     77,
		PartnerRefNo:  "partner-ref-1",
	}, withdrawInquiryCacheTTL, slog.Default())

	result, err := svc.Transfer(context.Background(), 11, model.UserRoleAdmin, WithdrawTransferInput{
		TokoID:    5,
		BankID:    9,
		Amount:    100000,
		InquiryID: 77,
	})

	require.NoError(t, err)
	require.True(t, result.Status)
	require.Equal(t, uint64(400000), result.RemainingSettlement)
	require.Equal(t, uint64(100000), result.Amount)
	require.Equal(t, uint64(100000), gateway.transferReq.Amount)
	require.Equal(t, uint64(77), gateway.transferReq.InquiryID)
	require.Equal(t, "014", gateway.transferReq.BankCode)

	updatedBalance, err := balanceRepo.GetByTokoID(context.Background(), 5)
	require.NoError(t, err)
	require.Equal(t, 400000.0, updatedBalance.SettlementBalance)
}

func TestWithdrawServiceTransferRefundsSettlementWhenGatewayFails(t *testing.T) {
	tokoRepo := &fakeTokoDomainRepo{
		byID: map[uint64]*model.Toko{
			5: {ID: 5, Name: "Toko Alpha"},
		},
	}
	balanceRepo := &fakeBalanceRepo{byTokoID: map[uint64]repository.TokoBalanceRecord{
		5: {TokoID: 5, TokoName: "Toko Alpha", SettlementBalance: 500000, AvailableBalance: 1000000, LastSettlementTime: time.Now().UTC()},
	}}
	bankRepo := &fakeBankRepo{items: map[uint64][]model.Bank{
		11: {
			{ID: 9, UserID: 11, PaymentID: 1, BankCode: "014", BankName: "PT. BANK CENTRAL ASIA, TBK.", AccountName: "PT GUE CONTROL", AccountNumber: "1234567890"},
		},
	}}
	cacheStore := newFakeCacheStore()
	gateway := &fakeWithdrawGatewayClient{
		transferErr: &paymentgateway.APIError{Message: "Invalid client", StatusCode: 400},
	}
	trxRepo := &fakeTransactionRepo{}
	svc := NewWithdrawService(tokoRepo, balanceRepo, bankRepo, trxRepo, gateway, cacheStore, true, "gue-client", "gue-key", "merchant-uuid", slog.Default())

	setCachedJSON(context.Background(), cacheStore, true, svc.inquiryCacheKey(11, 5, 9, 100000), &cachedWithdrawInquiry{
		TokoID:        5,
		TokoName:      "Toko Alpha",
		BankID:        9,
		BankName:      "PT. BANK CENTRAL ASIA, TBK.",
		AccountName:   "PT GUE CONTROL",
		AccountNumber: "1234567890",
		BankCode:      "014",
		Amount:        100000,
		Fee:           1500,
		InquiryID:     77,
		PartnerRefNo:  "partner-ref-2",
	}, withdrawInquiryCacheTTL, slog.Default())

	_, err := svc.Transfer(context.Background(), 11, model.UserRoleAdmin, WithdrawTransferInput{
		TokoID:    5,
		BankID:    9,
		Amount:    100000,
		InquiryID: 77,
	})

	require.Error(t, err)
	updatedBalance, balanceErr := balanceRepo.GetByTokoID(context.Background(), 5)
	require.NoError(t, balanceErr)
	require.Equal(t, 500000.0, updatedBalance.SettlementBalance)
	require.Len(t, trxRepo.statuses, 1)
	require.Equal(t, model.TransactionStatusFailed, trxRepo.statuses[0].status)
}
