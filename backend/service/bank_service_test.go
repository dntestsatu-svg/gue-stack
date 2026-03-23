package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/paymentgateway"
	"github.com/example/gue/backend/repository"
	"github.com/stretchr/testify/require"
	"log/slog"
)

type fakeBankRepo struct {
	items      map[uint64][]model.Bank
	nextID     uint64
	listCalls  int
	countCalls int
}

func (f *fakeBankRepo) ListByUser(_ context.Context, userID uint64, filter repository.BankListFilter) ([]model.Bank, error) {
	f.listCalls++
	items := append([]model.Bank(nil), f.items[userID]...)
	search := strings.ToLower(strings.TrimSpace(filter.SearchTerm))
	filtered := make([]model.Bank, 0, len(items))
	for _, item := range items {
		if search == "" || strings.Contains(strings.ToLower(item.BankName), search) || strings.Contains(strings.ToLower(item.AccountName), search) || strings.Contains(strings.ToLower(item.AccountNumber), search) {
			filtered = append(filtered, item)
		}
	}

	start := filter.Offset
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + filter.Limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[start:end], nil
}

func (f *fakeBankRepo) CountByUser(_ context.Context, userID uint64, filter repository.BankListFilter) (uint64, error) {
	f.countCalls++
	search := strings.ToLower(strings.TrimSpace(filter.SearchTerm))
	var total uint64
	for _, item := range f.items[userID] {
		if search == "" || strings.Contains(strings.ToLower(item.BankName), search) || strings.Contains(strings.ToLower(item.AccountName), search) || strings.Contains(strings.ToLower(item.AccountNumber), search) {
			total++
		}
	}
	return total, nil
}

func (f *fakeBankRepo) Create(_ context.Context, bank *model.Bank) error {
	for _, existing := range f.items[bank.UserID] {
		if existing.PaymentID == bank.PaymentID && existing.AccountNumber == bank.AccountNumber {
			return fmt.Errorf("duplicate entry")
		}
	}

	f.nextID++
	bank.ID = f.nextID
	if bank.CreatedAt.IsZero() {
		bank.CreatedAt = time.Now().UTC()
	}
	if bank.UpdatedAt.IsZero() {
		bank.UpdatedAt = bank.CreatedAt
	}
	f.items[bank.UserID] = append(f.items[bank.UserID], *bank)
	return nil
}

func (f *fakeBankRepo) DeleteByUser(_ context.Context, userID uint64, bankID uint64) error {
	items := f.items[userID]
	for idx, item := range items {
		if item.ID == bankID {
			f.items[userID] = append(items[:idx], items[idx+1:]...)
			return nil
		}
	}
	return repository.ErrNotFound
}

type fakePaymentRepo struct {
	items       map[uint64]model.Payment
	searchCalls int
}

type fakeBankInquiryGateway struct {
	inquiryFn func(ctx context.Context, req paymentgateway.InquiryTransferRequest) (*paymentgateway.InquiryTransferResponse, error)
}

func (f *fakePaymentRepo) GetByID(_ context.Context, id uint64) (*model.Payment, error) {
	item, ok := f.items[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copyItem := item
	return &copyItem, nil
}

func (f *fakePaymentRepo) SearchOptions(_ context.Context, filter repository.PaymentOptionFilter) ([]model.Payment, error) {
	f.searchCalls++
	search := strings.ToLower(strings.TrimSpace(filter.SearchTerm))
	result := make([]model.Payment, 0, filter.Limit)
	for _, item := range f.items {
		if search == "" || strings.Contains(strings.ToLower(item.BankName), search) || strings.Contains(strings.ToLower(item.BankCode), search) {
			result = append(result, item)
		}
		if len(result) == filter.Limit {
			break
		}
	}
	return result, nil
}

func (f *fakeBankInquiryGateway) InquiryTransfer(ctx context.Context, req paymentgateway.InquiryTransferRequest) (*paymentgateway.InquiryTransferResponse, error) {
	if f.inquiryFn == nil {
		return nil, nil
	}
	return f.inquiryFn(ctx, req)
}

func TestBankService_ListForbiddenForUserRole(t *testing.T) {
	svc := NewBankService(&fakeBankRepo{items: map[uint64][]model.Bank{}}, &fakePaymentRepo{}, nil, nil, false, time.Minute, "client", "key", "merchant", slog.Default())

	_, err := svc.List(context.Background(), 10, model.UserRoleUser, BankListQuery{Limit: 10})

	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient role")
}

func TestBankService_CreateUsesPaymentCatalogValidation(t *testing.T) {
	bankRepo := &fakeBankRepo{items: map[uint64][]model.Bank{}, nextID: 40}
	paymentRepo := &fakePaymentRepo{
		items: map[uint64]model.Payment{
			8: {
				ID:       8,
				BankCode: "014",
				BankName: "PT. BANK CENTRAL ASIA, TBK.",
			},
		},
	}
	svc := NewBankService(bankRepo, paymentRepo, nil, nil, false, time.Minute, "client", "key", "merchant", slog.Default())

	created, err := svc.Create(context.Background(), 99, model.UserRoleAdmin, CreateBankInput{
		PaymentID:     8,
		AccountName:   "PT GUE CONTROL",
		AccountNumber: "1234567890",
		InquiryID:     88,
	})

	require.NoError(t, err)
	require.Equal(t, uint64(8), created.PaymentID)
	require.Equal(t, "PT. BANK CENTRAL ASIA, TBK.", created.BankName)
	require.Equal(t, "PT GUE CONTROL", created.AccountName)
	require.Equal(t, "1234567890", created.AccountNumber)
	require.NotZero(t, created.ID)
	require.Len(t, bankRepo.items[99], 1)
	require.Equal(t, "014", bankRepo.items[99][0].BankCode)
}

func TestBankService_ListUsesPerUserCacheNamespace(t *testing.T) {
	bankRepo := &fakeBankRepo{
		items: map[uint64][]model.Bank{
			10: {
				{ID: 1, UserID: 10, PaymentID: 1, BankName: "Bank Alpha", AccountName: "Alpha", AccountNumber: "1111", CreatedAt: time.Now().UTC()},
			},
			20: {
				{ID: 2, UserID: 20, PaymentID: 2, BankName: "Bank Beta", AccountName: "Beta", AccountNumber: "2222", CreatedAt: time.Now().UTC()},
			},
		},
		nextID: 2,
	}
	paymentRepo := &fakePaymentRepo{
		items: map[uint64]model.Payment{
			3: {ID: 3, BankCode: "009", BankName: "BNI"},
		},
	}
	cacheStore := newFakeCacheStore()
	svc := NewBankService(bankRepo, paymentRepo, nil, cacheStore, true, time.Minute, "client", "key", "merchant", slog.Default())
	query := BankListQuery{Limit: 10, Offset: 0}

	_, err := svc.List(context.Background(), 10, model.UserRoleAdmin, query)
	require.NoError(t, err)
	_, err = svc.List(context.Background(), 10, model.UserRoleAdmin, query)
	require.NoError(t, err)
	require.Equal(t, 1, bankRepo.countCalls)
	require.Equal(t, 1, bankRepo.listCalls)

	_, err = svc.List(context.Background(), 20, model.UserRoleAdmin, query)
	require.NoError(t, err)
	require.Equal(t, 2, bankRepo.countCalls)
	require.Equal(t, 2, bankRepo.listCalls)

	_, err = svc.Create(context.Background(), 10, model.UserRoleAdmin, CreateBankInput{
		PaymentID:     3,
		AccountName:   "New Alpha",
		AccountNumber: "33333",
		InquiryID:     99,
	})
	require.NoError(t, err)

	_, err = svc.List(context.Background(), 10, model.UserRoleAdmin, query)
	require.NoError(t, err)
	require.Equal(t, 3, bankRepo.countCalls)
	require.Equal(t, 3, bankRepo.listCalls)

	_, err = svc.List(context.Background(), 20, model.UserRoleAdmin, query)
	require.NoError(t, err)
	require.Equal(t, 3, bankRepo.countCalls)
	require.Equal(t, 3, bankRepo.listCalls)
}

func TestBankService_PaymentOptionsUsesSearch(t *testing.T) {
	svc := NewBankService(
		&fakeBankRepo{items: map[uint64][]model.Bank{}},
		&fakePaymentRepo{
			items: map[uint64]model.Payment{
				1: {ID: 1, BankCode: "014", BankName: "PT. BANK CENTRAL ASIA, TBK."},
				2: {ID: 2, BankCode: "009", BankName: "PT. BANK NEGARA INDONESIA (PERSERO), TBK"},
			},
		},
		nil,
		nil,
		false,
		time.Minute,
		"client",
		"key",
		"merchant",
		slog.Default(),
	)

	items, err := svc.PaymentOptions(context.Background(), model.UserRoleSuperAdmin, PaymentOptionQuery{
		Limit:      10,
		SearchTerm: "central",
	})

	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "PT. BANK CENTRAL ASIA, TBK.", items[0].BankName)
}

func TestBankService_InquiryUsesPaymentCatalogAndReturnsAccountName(t *testing.T) {
	svc := NewBankService(
		&fakeBankRepo{items: map[uint64][]model.Bank{}},
		&fakePaymentRepo{
			items: map[uint64]model.Payment{
				8: {ID: 8, BankCode: "542", BankName: "PT. BANK ARTOS INDONESIA (Bank Jago)"},
			},
		},
		&fakeBankInquiryGateway{
			inquiryFn: func(_ context.Context, req paymentgateway.InquiryTransferRequest) (*paymentgateway.InquiryTransferResponse, error) {
				require.Equal(t, "gue-client", req.Client)
				require.Equal(t, "gue-key", req.ClientKey)
				require.Equal(t, "gue-merchant", req.UUID)
				require.Equal(t, uint64(10000), req.Amount)
				require.Equal(t, "542", req.BankCode)
				require.Equal(t, "100009689749", req.AccountNumber)
				require.Equal(t, bankInquiryTransferType, req.Type)
				return &paymentgateway.InquiryTransferResponse{
					AccountNumber: "100009689749",
					AccountName:   "SISKA DAMAYANTI",
					BankCode:      "542",
					BankName:      "PT. BANK ARTOS INDONESIA (Bank Jago)",
					PartnerRefNo:  "partner-ref",
					VendorRefNo:   "",
					Amount:        700000,
					Fee:           1800,
					InquiryID:     2949850,
				}, nil
			},
		},
		newFakeCacheStore(),
		true,
		time.Minute,
		"gue-client",
		"gue-key",
		"gue-merchant",
		slog.Default(),
	)

	result, err := svc.Inquiry(context.Background(), 55, model.UserRoleAdmin, BankInquiryInput{
		PaymentID:     8,
		AccountNumber: "100009689749",
	})

	require.NoError(t, err)
	require.Equal(t, uint64(8), result.PaymentID)
	require.Equal(t, "SISKA DAMAYANTI", result.AccountName)
	require.Equal(t, uint64(2949850), result.InquiryID)
}
