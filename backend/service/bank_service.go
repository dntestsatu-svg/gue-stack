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
)

const (
	bankInquiryCacheTTL     = 5 * time.Minute
	bankInquiryProbeAmount  = uint64(10000)
	bankInquiryTransferType = 2
)

type BankUseCase interface {
	List(ctx context.Context, userID uint64, actorRole model.UserRole, query BankListQuery) (*BankListPage, error)
	Inquiry(ctx context.Context, userID uint64, actorRole model.UserRole, input BankInquiryInput) (*BankInquiryResult, error)
	Create(ctx context.Context, userID uint64, actorRole model.UserRole, input CreateBankInput) (*BankDTO, error)
	Delete(ctx context.Context, userID uint64, actorRole model.UserRole, bankID uint64) error
	PaymentOptions(ctx context.Context, actorRole model.UserRole, query PaymentOptionQuery) ([]PaymentOptionDTO, error)
}

type bankInquiryGateway interface {
	InquiryTransfer(ctx context.Context, req paymentgateway.InquiryTransferRequest) (*paymentgateway.InquiryTransferResponse, error)
}

type BankService struct {
	bankRepo      repository.BankRepository
	paymentRepo   repository.PaymentRepository
	gatewayClient bankInquiryGateway
	cache         cache.Cache
	cacheEnabled  bool
	listCacheTTL  time.Duration
	defaultClient string
	defaultKey    string
	merchantUUID  string
	logger        *slog.Logger
	validate      *validator.Validate
}

type BankListQuery struct {
	Limit      int
	Offset     int
	SearchTerm string
}

type PaymentOptionQuery struct {
	Limit      int
	SearchTerm string
}

type BankInquiryInput struct {
	PaymentID     uint64 `json:"payment_id" validate:"required,gt=0"`
	AccountNumber string `json:"account_number" validate:"required,min=5,max=64"`
}

type BankInquiryResult struct {
	PaymentID     uint64 `json:"payment_id"`
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
	BankCode      string `json:"bank_code"`
	BankName      string `json:"bank_name"`
	PartnerRefNo  string `json:"partner_ref_no"`
	VendorRefNo   string `json:"vendor_ref_no"`
	Amount        uint64 `json:"amount"`
	Fee           uint64 `json:"fee"`
	InquiryID     uint64 `json:"inquiry_id"`
}

type CreateBankInput struct {
	PaymentID     uint64 `json:"payment_id" validate:"required,gt=0"`
	AccountName   string `json:"account_name" validate:"required,min=2,max=255"`
	AccountNumber string `json:"account_number" validate:"required,min=5,max=64"`
	InquiryID     uint64 `json:"inquiry_id" validate:"required,gt=0"`
}

type BankDTO struct {
	ID            uint64 `json:"id"`
	PaymentID     uint64 `json:"payment_id"`
	BankName      string `json:"bank_name"`
	AccountName   string `json:"account_name"`
	AccountNumber string `json:"account_number"`
	CreatedAt     string `json:"created_at"`
}

type BankListPage struct {
	Items   []BankDTO `json:"items"`
	Total   uint64    `json:"total"`
	Limit   int       `json:"limit"`
	Offset  int       `json:"offset"`
	HasMore bool      `json:"has_more"`
}

type PaymentOptionDTO struct {
	ID       uint64 `json:"id"`
	BankName string `json:"bank_name"`
}

func NewBankService(
	bankRepo repository.BankRepository,
	paymentRepo repository.PaymentRepository,
	gatewayClient bankInquiryGateway,
	cacheStore cache.Cache,
	cacheEnabled bool,
	listCacheTTL time.Duration,
	defaultClient string,
	defaultKey string,
	merchantUUID string,
	logger *slog.Logger,
) *BankService {
	if listCacheTTL <= 0 {
		listCacheTTL = 5 * time.Minute
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &BankService{
		bankRepo:      bankRepo,
		paymentRepo:   paymentRepo,
		gatewayClient: gatewayClient,
		cache:         cacheStore,
		cacheEnabled:  cacheEnabled,
		listCacheTTL:  listCacheTTL,
		defaultClient: strings.TrimSpace(defaultClient),
		defaultKey:    strings.TrimSpace(defaultKey),
		merchantUUID:  strings.TrimSpace(merchantUUID),
		logger:        logger,
		validate:      validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (s *BankService) List(ctx context.Context, userID uint64, actorRole model.UserRole, query BankListQuery) (*BankListPage, error) {
	if !canManageBanks(actorRole) {
		return nil, apperror.New(http.StatusForbidden, "insufficient role permission", "role user cannot access bank management")
	}

	filter := normalizeBankListQuery(query)
	cacheKey := s.listCacheKey(ctx, userID, filter)
	if cached, ok := getCachedJSON[BankListPage](ctx, s.cache, s.cacheEnabled, cacheKey, s.logger); ok {
		return cached, nil
	}

	total, err := s.bankRepo.CountByUser(ctx, userID, repository.BankListFilter{
		SearchTerm: filter.SearchTerm,
	})
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to count banks", err.Error())
	}

	items, err := s.bankRepo.ListByUser(ctx, userID, repository.BankListFilter{
		Limit:      filter.Limit,
		Offset:     filter.Offset,
		SearchTerm: filter.SearchTerm,
	})
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to list banks", err.Error())
	}

	result := make([]BankDTO, 0, len(items))
	for _, item := range items {
		result = append(result, mapBankDTO(item))
	}

	page := &BankListPage{
		Items:   result,
		Total:   total,
		Limit:   filter.Limit,
		Offset:  filter.Offset,
		HasMore: uint64(filter.Offset+len(result)) < total,
	}
	setCachedJSON(ctx, s.cache, s.cacheEnabled, cacheKey, page, s.listCacheTTL, s.logger)
	return page, nil
}

func (s *BankService) Inquiry(ctx context.Context, userID uint64, actorRole model.UserRole, input BankInquiryInput) (*BankInquiryResult, error) {
	if !canManageBanks(actorRole) {
		return nil, apperror.New(http.StatusForbidden, "insufficient role permission", "role user cannot access bank management")
	}
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}
	if s.gatewayClient == nil {
		return nil, apperror.New(http.StatusInternalServerError, "bank inquiry gateway is not configured", nil)
	}
	if s.defaultClient == "" || s.defaultKey == "" || s.merchantUUID == "" {
		return nil, apperror.New(http.StatusInternalServerError, "payment gateway integration is not configured", nil)
	}

	payment, err := s.paymentRepo.GetByID(ctx, input.PaymentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusBadRequest, "invalid bank selection", "selected bank_name is not available in payments catalog")
		}
		return nil, apperror.New(http.StatusInternalServerError, "failed to validate bank option", err.Error())
	}

	resp, err := s.gatewayClient.InquiryTransfer(ctx, paymentgateway.InquiryTransferRequest{
		Client:        s.defaultClient,
		ClientKey:     s.defaultKey,
		UUID:          s.merchantUUID,
		Amount:        bankInquiryProbeAmount,
		BankCode:      strings.TrimSpace(payment.BankCode),
		AccountNumber: strings.TrimSpace(input.AccountNumber),
		Type:          bankInquiryTransferType,
	})
	if err != nil {
		return nil, s.mapGatewayError("failed to inquiry bank account", err)
	}

	if !sameBankCode(payment.BankCode, resp.BankCode) {
		return nil, apperror.New(http.StatusBadRequest, "bank inquiry mismatch", "bank_code from inquiry does not match selected bank")
	}

	result := &BankInquiryResult{
		PaymentID:     payment.ID,
		AccountNumber: resp.AccountNumber,
		AccountName:   resp.AccountName,
		BankCode:      resp.BankCode,
		BankName:      resp.BankName,
		PartnerRefNo:  resp.PartnerRefNo,
		VendorRefNo:   resp.VendorRefNo,
		Amount:        resp.Amount,
		Fee:           resp.Fee,
		InquiryID:     resp.InquiryID,
	}

	setCachedJSON(ctx, s.cache, s.cacheEnabled, s.inquiryCacheKey(userID, payment.ID, result.AccountNumber), result, bankInquiryCacheTTL, s.logger)
	return result, nil
}

func (s *BankService) Create(ctx context.Context, userID uint64, actorRole model.UserRole, input CreateBankInput) (*BankDTO, error) {
	if !canManageBanks(actorRole) {
		return nil, apperror.New(http.StatusForbidden, "insufficient role permission", "role user cannot access bank management")
	}
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}

	payment, err := s.paymentRepo.GetByID(ctx, input.PaymentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusBadRequest, "invalid bank selection", "selected bank_name is not available in payments catalog")
		}
		return nil, apperror.New(http.StatusInternalServerError, "failed to validate bank option", err.Error())
	}

	bank := &model.Bank{
		UserID:        userID,
		PaymentID:     payment.ID,
		BankCode:      payment.BankCode,
		BankName:      payment.BankName,
		AccountName:   strings.TrimSpace(input.AccountName),
		AccountNumber: strings.TrimSpace(input.AccountNumber),
	}

	if cached, ok := getCachedJSON[BankInquiryResult](ctx, s.cache, s.cacheEnabled, s.inquiryCacheKey(userID, input.PaymentID, input.AccountNumber), s.logger); ok {
		if cached.InquiryID != input.InquiryID {
			return nil, apperror.New(http.StatusBadRequest, "invalid inquiry confirmation", "inquiry confirmation has expired or does not match")
		}
		bank.AccountName = strings.TrimSpace(cached.AccountName)
		bank.AccountNumber = strings.TrimSpace(cached.AccountNumber)
	}

	if err := s.bankRepo.Create(ctx, bank); err != nil {
		if isBankDuplicateKeyError(err) {
			return nil, apperror.New(http.StatusConflict, "bank account already exists", "kombinasi bank dan nomor rekening ini sudah tersimpan")
		}
		return nil, apperror.New(http.StatusInternalServerError, "failed to create bank", err.Error())
	}

	s.invalidateListCache(ctx, userID)
	s.clearInquiryCache(ctx, userID, input.PaymentID, bank.AccountNumber)
	return &BankDTO{
		ID:            bank.ID,
		PaymentID:     bank.PaymentID,
		BankName:      bank.BankName,
		AccountName:   bank.AccountName,
		AccountNumber: bank.AccountNumber,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *BankService) Delete(ctx context.Context, userID uint64, actorRole model.UserRole, bankID uint64) error {
	if !canManageBanks(actorRole) {
		return apperror.New(http.StatusForbidden, "insufficient role permission", "role user cannot access bank management")
	}
	if bankID == 0 {
		return apperror.New(http.StatusBadRequest, "invalid bank id", nil)
	}
	if err := s.bankRepo.DeleteByUser(ctx, userID, bankID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(http.StatusNotFound, "bank not found", nil)
		}
		return apperror.New(http.StatusInternalServerError, "failed to delete bank", err.Error())
	}
	s.invalidateListCache(ctx, userID)
	return nil
}

func (s *BankService) PaymentOptions(ctx context.Context, actorRole model.UserRole, query PaymentOptionQuery) ([]PaymentOptionDTO, error) {
	if !canManageBanks(actorRole) {
		return nil, apperror.New(http.StatusForbidden, "insufficient role permission", "role user cannot access bank management")
	}

	filter := normalizePaymentOptionQuery(query)
	cacheKey := buildHashedCacheKey(
		"bank:payment-options",
		"limit="+strconvItoa(filter.Limit),
		"search="+strings.ToLower(strings.TrimSpace(filter.SearchTerm)),
	)
	if cached, ok := getCachedJSON[[]PaymentOptionDTO](ctx, s.cache, s.cacheEnabled, cacheKey, s.logger); ok {
		return *cached, nil
	}

	items, err := s.paymentRepo.SearchOptions(ctx, repository.PaymentOptionFilter{
		Limit:      filter.Limit,
		SearchTerm: filter.SearchTerm,
	})
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to search bank options", err.Error())
	}

	result := make([]PaymentOptionDTO, 0, len(items))
	for _, item := range items {
		result = append(result, PaymentOptionDTO{
			ID:       item.ID,
			BankName: item.BankName,
		})
	}
	setCachedJSON(ctx, s.cache, s.cacheEnabled, cacheKey, result, s.listCacheTTL, s.logger)
	return result, nil
}

func mapBankDTO(bank model.Bank) BankDTO {
	return BankDTO{
		ID:            bank.ID,
		PaymentID:     bank.PaymentID,
		BankName:      bank.BankName,
		AccountName:   bank.AccountName,
		AccountNumber: bank.AccountNumber,
		CreatedAt:     bank.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func normalizeBankListQuery(query BankListQuery) BankListQuery {
	if query.Limit <= 0 {
		query.Limit = 10
	}
	if query.Limit > 50 {
		query.Limit = 50
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	query.SearchTerm = strings.TrimSpace(query.SearchTerm)
	return query
}

func normalizePaymentOptionQuery(query PaymentOptionQuery) PaymentOptionQuery {
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 50 {
		query.Limit = 50
	}
	query.SearchTerm = strings.TrimSpace(query.SearchTerm)
	return query
}

func canManageBanks(role model.UserRole) bool {
	return role == model.UserRoleDev || role == model.UserRoleSuperAdmin || role == model.UserRoleAdmin
}

func (s *BankService) listNamespaceKey(userID uint64) string {
	return "banks:list:namespace:user:" + strconvItoaFromUint64(userID)
}

func (s *BankService) listCacheKey(ctx context.Context, userID uint64, query BankListQuery) string {
	namespace := getCacheNamespaceToken(ctx, s.cache, s.cacheEnabled, s.listNamespaceKey(userID), s.logger)
	return buildHashedCacheKey(
		"banks:list",
		"ns="+namespace,
		"user="+strconvItoaFromUint64(userID),
		"limit="+strconvItoa(query.Limit),
		"offset="+strconvItoa(query.Offset),
		"search="+strings.ToLower(strings.TrimSpace(query.SearchTerm)),
	)
}

func (s *BankService) invalidateListCache(ctx context.Context, userID uint64) {
	bumpCacheNamespace(ctx, s.cache, s.cacheEnabled, s.listNamespaceKey(userID), s.logger)
}

func (s *BankService) inquiryCacheKey(userID uint64, paymentID uint64, accountNumber string) string {
	return buildHashedCacheKey(
		"banks:inquiry",
		"user="+strconvItoaFromUint64(userID),
		"payment="+strconvItoaFromUint64(paymentID),
		"account="+normalizeAccountNumber(accountNumber),
	)
}

func (s *BankService) clearInquiryCache(ctx context.Context, userID uint64, paymentID uint64, accountNumber string) {
	if !s.cacheEnabled || s.cache == nil {
		return
	}
	if err := s.cache.Delete(ctx, s.inquiryCacheKey(userID, paymentID, accountNumber)); err != nil && s.logger != nil {
		s.logger.Error("cache delete failed", "key", s.inquiryCacheKey(userID, paymentID, accountNumber), "error", err.Error())
	}
}

func strconvItoa(value int) string {
	return strconv.Itoa(value)
}

func strconvItoaFromUint64(value uint64) string {
	return strconv.FormatUint(value, 10)
}

func isBankDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate")
}

func normalizeAccountNumber(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func sameBankCode(expected, actual string) bool {
	return strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(actual))
}

func (s *BankService) mapGatewayError(message string, err error) error {
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

var _ BankUseCase = (*BankService)(nil)
