package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/gue/backend/cache"
	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/paymentgateway"
	"github.com/example/gue/backend/repository"
	"github.com/go-playground/validator/v10"
	"github.com/shopspring/decimal"
)

const (
	withdrawInquiryCacheTTL    = 5 * time.Minute
	withdrawTransferType       = 2
	withdrawWorkspaceNamespace = "tokos:workspace:namespace"
	withdrawBankOptionsLimit   = 100
)

type WithdrawUseCase interface {
	Options(ctx context.Context, userID uint64, actorRole model.UserRole) (*WithdrawOptionsResult, error)
	Inquiry(ctx context.Context, userID uint64, actorRole model.UserRole, input WithdrawInquiryInput) (*WithdrawInquiryResult, error)
	Transfer(ctx context.Context, userID uint64, actorRole model.UserRole, input WithdrawTransferInput) (*WithdrawTransferResult, error)
}

type WithdrawService struct {
	tokoRepo        repository.TokoRepository
	balanceRepo     repository.BalanceRepository
	bankRepo        repository.BankRepository
	transactionRepo repository.TransactionRepository
	gatewayClient   paymentgateway.Client
	cache           cache.Cache
	cacheEnabled    bool
	defaultClient   string
	defaultKey      string
	merchantUUID    string
	logger          *slog.Logger
	validate        *validator.Validate
}

type WithdrawOptionsResult struct {
	Tokos []WithdrawTokoOption `json:"tokos"`
	Banks []WithdrawBankOption `json:"banks"`
}

type WithdrawTokoOption struct {
	ID                uint64  `json:"id"`
	Name              string  `json:"name"`
	SettlementBalance float64 `json:"settlement_balance"`
	AvailableBalance  float64 `json:"available_balance"`
}

type WithdrawBankOption struct {
	ID            uint64 `json:"id"`
	BankName      string `json:"bank_name"`
	AccountName   string `json:"account_name"`
	AccountNumber string `json:"account_number"`
}

type WithdrawInquiryInput struct {
	TokoID uint64 `json:"toko_id" validate:"required,gt=0"`
	BankID uint64 `json:"bank_id" validate:"required,gt=0"`
	Amount uint64 `json:"amount" validate:"required,gte=25000,lte=100000000"`
}

type WithdrawTransferInput struct {
	TokoID    uint64 `json:"toko_id" validate:"required,gt=0"`
	BankID    uint64 `json:"bank_id" validate:"required,gt=0"`
	Amount    uint64 `json:"amount" validate:"required,gte=25000,lte=100000000"`
	InquiryID uint64 `json:"inquiry_id" validate:"required,gt=0"`
}

type WithdrawInquiryResult struct {
	TokoID              uint64 `json:"toko_id"`
	TokoName            string `json:"toko_name"`
	BankID              uint64 `json:"bank_id"`
	BankName            string `json:"bank_name"`
	AccountName         string `json:"account_name"`
	AccountNumber       string `json:"account_number"`
	Amount              uint64 `json:"amount"`
	Fee                 uint64 `json:"fee"`
	InquiryID           uint64 `json:"inquiry_id"`
	PartnerRefNo        string `json:"partner_ref_no"`
	SettlementBalance   uint64 `json:"settlement_balance"`
	RemainingSettlement uint64 `json:"remaining_settlement_balance"`
}

type WithdrawTransferResult struct {
	Status              bool   `json:"status"`
	Message             string `json:"message"`
	TokoID              uint64 `json:"toko_id"`
	TokoName            string `json:"toko_name"`
	BankID              uint64 `json:"bank_id"`
	BankName            string `json:"bank_name"`
	AccountName         string `json:"account_name"`
	AccountNumber       string `json:"account_number"`
	Amount              uint64 `json:"amount"`
	RemainingSettlement uint64 `json:"remaining_settlement_balance"`
}

type cachedWithdrawInquiry struct {
	TokoID        uint64 `json:"toko_id"`
	TokoName      string `json:"toko_name"`
	BankID        uint64 `json:"bank_id"`
	BankName      string `json:"bank_name"`
	AccountName   string `json:"account_name"`
	AccountNumber string `json:"account_number"`
	BankCode      string `json:"bank_code"`
	Amount        uint64 `json:"amount"`
	Fee           uint64 `json:"fee"`
	InquiryID     uint64 `json:"inquiry_id"`
	PartnerRefNo  string `json:"partner_ref_no"`
}

func NewWithdrawService(
	tokoRepo repository.TokoRepository,
	balanceRepo repository.BalanceRepository,
	bankRepo repository.BankRepository,
	transactionRepo repository.TransactionRepository,
	gatewayClient paymentgateway.Client,
	cacheStore cache.Cache,
	cacheEnabled bool,
	defaultClient string,
	defaultKey string,
	merchantUUID string,
	logger *slog.Logger,
) *WithdrawService {
	if logger == nil {
		logger = slog.Default()
	}
	return &WithdrawService{
		tokoRepo:        tokoRepo,
		balanceRepo:     balanceRepo,
		bankRepo:        bankRepo,
		transactionRepo: transactionRepo,
		gatewayClient:   gatewayClient,
		cache:           cacheStore,
		cacheEnabled:    cacheEnabled,
		defaultClient:   strings.TrimSpace(defaultClient),
		defaultKey:      strings.TrimSpace(defaultKey),
		merchantUUID:    strings.TrimSpace(merchantUUID),
		logger:          logger,
		validate:        validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (s *WithdrawService) Options(ctx context.Context, userID uint64, actorRole model.UserRole) (*WithdrawOptionsResult, error) {
	if !canRequestWithdraw(actorRole) {
		return nil, apperror.New(http.StatusForbidden, "insufficient role permission", "withdraw hanya tersedia untuk dev, superadmin, atau admin")
	}

	balances, err := s.balanceRepo.ListByUser(ctx, userID, actorRole)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to load withdraw toko options", err.Error())
	}
	banks, err := s.bankRepo.ListByUser(ctx, userID, repository.BankListFilter{
		Limit:  withdrawBankOptionsLimit,
		Offset: 0,
	})
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to load withdraw bank options", err.Error())
	}

	result := &WithdrawOptionsResult{
		Tokos: make([]WithdrawTokoOption, 0, len(balances)),
		Banks: make([]WithdrawBankOption, 0, len(banks)),
	}
	for _, item := range balances {
		result.Tokos = append(result.Tokos, WithdrawTokoOption{
			ID:                item.TokoID,
			Name:              item.TokoName,
			SettlementBalance: item.SettlementBalance,
			AvailableBalance:  item.AvailableBalance,
		})
	}
	for _, item := range banks {
		result.Banks = append(result.Banks, WithdrawBankOption{
			ID:            item.ID,
			BankName:      item.BankName,
			AccountName:   item.AccountName,
			AccountNumber: item.AccountNumber,
		})
	}
	return result, nil
}

func (s *WithdrawService) Inquiry(ctx context.Context, userID uint64, actorRole model.UserRole, input WithdrawInquiryInput) (*WithdrawInquiryResult, error) {
	if !canRequestWithdraw(actorRole) {
		return nil, apperror.New(http.StatusForbidden, "insufficient role permission", "withdraw hanya tersedia untuk dev, superadmin, atau admin")
	}
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}
	if s.gatewayClient == nil {
		return nil, apperror.New(http.StatusInternalServerError, "withdraw gateway is not configured", nil)
	}
	if s.defaultClient == "" || s.defaultKey == "" || s.merchantUUID == "" {
		return nil, apperror.New(http.StatusInternalServerError, "payment gateway integration is not configured", nil)
	}

	toko, bank, balance, err := s.resolveWithdrawContext(ctx, userID, actorRole, input.TokoID, input.BankID)
	if err != nil {
		return nil, err
	}
	if err := ensureSettlementSufficient(balance.SettlementBalance, input.Amount); err != nil {
		return nil, err
	}

	resp, err := s.gatewayClient.InquiryTransfer(ctx, paymentgateway.InquiryTransferRequest{
		Client:        s.defaultClient,
		ClientKey:     s.defaultKey,
		UUID:          s.merchantUUID,
		Amount:        input.Amount,
		BankCode:      strings.TrimSpace(bank.BankCode),
		AccountNumber: strings.TrimSpace(bank.AccountNumber),
		Type:          withdrawTransferType,
	})
	if err != nil {
		return nil, s.mapGatewayError("failed to inquiry withdraw destination", err)
	}
	if !strings.EqualFold(strings.TrimSpace(resp.BankCode), strings.TrimSpace(bank.BankCode)) {
		return nil, apperror.New(http.StatusBadRequest, "bank inquiry mismatch", "bank inquiry response does not match selected bank")
	}

	cached := &cachedWithdrawInquiry{
		TokoID:        toko.ID,
		TokoName:      toko.Name,
		BankID:        bank.ID,
		BankName:      bank.BankName,
		AccountName:   resp.AccountName,
		AccountNumber: resp.AccountNumber,
		BankCode:      resp.BankCode,
		Amount:        resp.Amount,
		Fee:           resp.Fee,
		InquiryID:     resp.InquiryID,
		PartnerRefNo:  resp.PartnerRefNo,
	}
	setCachedJSON(ctx, s.cache, s.cacheEnabled, s.inquiryCacheKey(userID, input.TokoID, input.BankID, input.Amount), cached, withdrawInquiryCacheTTL, s.logger)

	currentSettlement := uint64(decimal.NewFromFloat(balance.SettlementBalance).IntPart())
	remaining := currentSettlement
	if currentSettlement >= input.Amount {
		remaining = currentSettlement - input.Amount
	}

	return &WithdrawInquiryResult{
		TokoID:              toko.ID,
		TokoName:            toko.Name,
		BankID:              bank.ID,
		BankName:            bank.BankName,
		AccountName:         resp.AccountName,
		AccountNumber:       resp.AccountNumber,
		Amount:              resp.Amount,
		Fee:                 resp.Fee,
		InquiryID:           resp.InquiryID,
		PartnerRefNo:        resp.PartnerRefNo,
		SettlementBalance:   currentSettlement,
		RemainingSettlement: remaining,
	}, nil
}

func (s *WithdrawService) Transfer(ctx context.Context, userID uint64, actorRole model.UserRole, input WithdrawTransferInput) (*WithdrawTransferResult, error) {
	if !canRequestWithdraw(actorRole) {
		return nil, apperror.New(http.StatusForbidden, "insufficient role permission", "withdraw hanya tersedia untuk dev, superadmin, atau admin")
	}
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}
	if s.gatewayClient == nil {
		return nil, apperror.New(http.StatusInternalServerError, "withdraw gateway is not configured", nil)
	}
	if s.defaultClient == "" || s.defaultKey == "" || s.merchantUUID == "" {
		return nil, apperror.New(http.StatusInternalServerError, "payment gateway integration is not configured", nil)
	}

	toko, bank, _, err := s.resolveWithdrawContext(ctx, userID, actorRole, input.TokoID, input.BankID)
	if err != nil {
		return nil, err
	}

	cached, ok := getCachedJSON[cachedWithdrawInquiry](ctx, s.cache, s.cacheEnabled, s.inquiryCacheKey(userID, input.TokoID, input.BankID, input.Amount), s.logger)
	if !ok || cached == nil {
		return nil, apperror.New(http.StatusBadRequest, "withdraw inquiry expired", "silakan lakukan inquiry ulang sebelum meminta withdraw")
	}
	if cached.InquiryID != input.InquiryID {
		return nil, apperror.New(http.StatusBadRequest, "invalid withdraw inquiry confirmation", "inquiry withdraw tidak cocok atau sudah kadaluarsa")
	}
	if cached.PartnerRefNo == "" {
		return nil, apperror.New(http.StatusBadRequest, "invalid withdraw inquiry state", "partner_ref_no dari inquiry belum tersedia")
	}

	if _, err := s.transactionRepo.GetByReferenceAndToko(ctx, cached.PartnerRefNo, toko.ID); err == nil {
		return nil, apperror.New(http.StatusConflict, "withdraw already requested", "permintaan withdraw untuk inquiry ini sudah pernah dibuat")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.New(http.StatusInternalServerError, "failed to verify existing withdraw transaction", err.Error())
	}

	if err := s.balanceRepo.DecreaseSettlementByTokoID(ctx, toko.ID, float64(cached.Amount)); err != nil {
		if errors.Is(err, repository.ErrInsufficientBalance) {
			return nil, apperror.New(http.StatusBadRequest, "insufficient settlement balance", "saldo settlement toko tidak mencukupi untuk withdraw ini")
		}
		return nil, apperror.New(http.StatusInternalServerError, "failed to reserve settlement balance", err.Error())
	}

	reference := cached.PartnerRefNo
	fee := cached.Fee
	trx := &model.Transaction{
		TokoID:        toko.ID,
		Type:          model.TransactionTypeWithdraw,
		Status:        model.TransactionStatusPending,
		Reference:     &reference,
		Amount:        cached.Amount,
		FeeWithdrawal: &fee,
		PlatformFee:   0,
		Netto:         computePendingNetto(cached.Amount, &fee),
	}
	if err := s.transactionRepo.Create(ctx, trx); err != nil {
		_ = s.balanceRepo.IncreaseSettlementByTokoID(ctx, toko.ID, float64(cached.Amount))
		return nil, apperror.New(http.StatusInternalServerError, "failed to persist withdraw transaction", err.Error())
	}

	resp, err := s.gatewayClient.TransferFund(ctx, paymentgateway.TransferFundRequest{
		Client:        s.defaultClient,
		ClientKey:     s.defaultKey,
		UUID:          s.merchantUUID,
		Amount:        cached.Amount,
		BankCode:      bank.BankCode,
		AccountNumber: bank.AccountNumber,
		Type:          withdrawTransferType,
		InquiryID:     cached.InquiryID,
	})
	if err != nil {
		_ = s.transactionRepo.UpdateStatusByReferenceAndToko(ctx, cached.PartnerRefNo, toko.ID, model.TransactionStatusFailed)
		_ = s.balanceRepo.IncreaseSettlementByTokoID(ctx, toko.ID, float64(cached.Amount))
		return nil, s.mapGatewayError("failed to request withdraw transfer", err)
	}

	if !resp.Status {
		_ = s.transactionRepo.UpdateStatusByReferenceAndToko(ctx, cached.PartnerRefNo, toko.ID, model.TransactionStatusFailed)
		_ = s.balanceRepo.IncreaseSettlementByTokoID(ctx, toko.ID, float64(cached.Amount))
		return nil, apperror.New(http.StatusBadGateway, "withdraw transfer rejected", "error")
	}

	s.clearInquiryCache(ctx, userID, input.TokoID, input.BankID, input.Amount)
	s.invalidateWorkspaceCache(ctx)

	remainingBalance, balanceErr := s.balanceRepo.GetByTokoID(ctx, toko.ID)
	if balanceErr != nil {
		if s.logger != nil {
			s.logger.Warn("failed to load updated settlement balance after withdraw", "toko_id", toko.ID, "error", balanceErr.Error())
		}
	}
	remaining := uint64(0)
	if remainingBalance != nil {
		remaining = uint64(decimal.NewFromFloat(remainingBalance.SettlementBalance).IntPart())
	}

	return &WithdrawTransferResult{
		Status:              true,
		Message:             "Uangnya akan segera sampai ke bank anda.",
		TokoID:              toko.ID,
		TokoName:            toko.Name,
		BankID:              bank.ID,
		BankName:            bank.BankName,
		AccountName:         cached.AccountName,
		AccountNumber:       bank.AccountNumber,
		Amount:              cached.Amount,
		RemainingSettlement: remaining,
	}, nil
}

func (s *WithdrawService) resolveWithdrawContext(ctx context.Context, userID uint64, actorRole model.UserRole, tokoID uint64, bankID uint64) (*model.Toko, *model.Bank, *repository.TokoBalanceRecord, error) {
	toko, err := s.tokoRepo.GetAccessibleByID(ctx, userID, actorRole, tokoID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, nil, apperror.New(http.StatusNotFound, "toko not found", nil)
		}
		return nil, nil, nil, apperror.New(http.StatusInternalServerError, "failed to access toko", err.Error())
	}

	bank, err := s.bankRepo.GetByUser(ctx, userID, bankID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, nil, apperror.New(http.StatusNotFound, "bank not found", nil)
		}
		return nil, nil, nil, apperror.New(http.StatusInternalServerError, "failed to access user bank", err.Error())
	}

	balance, err := s.balanceRepo.GetByTokoID(ctx, toko.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, nil, apperror.New(http.StatusNotFound, "toko balance not found", nil)
		}
		return nil, nil, nil, apperror.New(http.StatusInternalServerError, "failed to read toko balance", err.Error())
	}

	return toko, bank, balance, nil
}

func (s *WithdrawService) inquiryCacheKey(userID uint64, tokoID uint64, bankID uint64, amount uint64) string {
	return buildHashedCacheKey(
		"withdraw:inquiry",
		"user="+strconv.FormatUint(userID, 10),
		"toko="+strconv.FormatUint(tokoID, 10),
		"bank="+strconv.FormatUint(bankID, 10),
		"amount="+strconv.FormatUint(amount, 10),
	)
}

func (s *WithdrawService) clearInquiryCache(ctx context.Context, userID uint64, tokoID uint64, bankID uint64, amount uint64) {
	if !s.cacheEnabled || s.cache == nil {
		return
	}
	key := s.inquiryCacheKey(userID, tokoID, bankID, amount)
	if err := s.cache.Delete(ctx, key); err != nil && s.logger != nil {
		s.logger.Error("cache delete failed", "key", key, "error", err.Error())
	}
}

func (s *WithdrawService) invalidateWorkspaceCache(ctx context.Context) {
	bumpCacheNamespace(ctx, s.cache, s.cacheEnabled, withdrawWorkspaceNamespace, s.logger)
}

func (s *WithdrawService) mapGatewayError(message string, err error) error {
	var upstream *paymentgateway.APIError
	if errors.As(err, &upstream) {
		status := http.StatusBadGateway
		if upstream.StatusCode >= 400 && upstream.StatusCode < 500 {
			status = http.StatusBadRequest
		}
		return apperror.New(status, message, upstream.Message)
	}
	return apperror.New(http.StatusBadGateway, message, err.Error())
}

func canRequestWithdraw(role model.UserRole) bool {
	return role == model.UserRoleDev || role == model.UserRoleSuperAdmin || role == model.UserRoleAdmin
}

func ensureSettlementSufficient(currentBalance float64, amount uint64) error {
	current := decimal.NewFromFloat(currentBalance)
	requested := decimal.NewFromInt(int64(amount))
	if current.LessThan(requested) {
		return apperror.New(http.StatusBadRequest, "insufficient settlement balance", "saldo settlement toko tidak mencukupi untuk withdraw ini")
	}
	return nil
}

var _ WithdrawUseCase = (*WithdrawService)(nil)
