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
	"github.com/example/gue/backend/repository"
	"github.com/go-playground/validator/v10"
)

type BankUseCase interface {
	List(ctx context.Context, userID uint64, actorRole model.UserRole, query BankListQuery) (*BankListPage, error)
	Create(ctx context.Context, userID uint64, actorRole model.UserRole, input CreateBankInput) (*BankDTO, error)
	Delete(ctx context.Context, userID uint64, actorRole model.UserRole, bankID uint64) error
	PaymentOptions(ctx context.Context, actorRole model.UserRole, query PaymentOptionQuery) ([]PaymentOptionDTO, error)
}

type BankService struct {
	bankRepo      repository.BankRepository
	paymentRepo   repository.PaymentRepository
	cache         cache.Cache
	cacheEnabled  bool
	listCacheTTL  time.Duration
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

type CreateBankInput struct {
	PaymentID     uint64 `json:"payment_id" validate:"required,gt=0"`
	AccountName   string `json:"account_name" validate:"required,min=2,max=255"`
	AccountNumber string `json:"account_number" validate:"required,min=5,max=64"`
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
	cacheStore cache.Cache,
	cacheEnabled bool,
	listCacheTTL time.Duration,
	logger *slog.Logger,
) *BankService {
	if listCacheTTL <= 0 {
		listCacheTTL = 5 * time.Minute
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &BankService{
		bankRepo:     bankRepo,
		paymentRepo:  paymentRepo,
		cache:        cacheStore,
		cacheEnabled: cacheEnabled,
		listCacheTTL: listCacheTTL,
		logger:       logger,
		validate:     validator.New(validator.WithRequiredStructEnabled()),
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
	if err := s.bankRepo.Create(ctx, bank); err != nil {
		if isBankDuplicateKeyError(err) {
			return nil, apperror.New(http.StatusConflict, "bank account already exists", "kombinasi bank dan nomor rekening ini sudah tersimpan")
		}
		return nil, apperror.New(http.StatusInternalServerError, "failed to create bank", err.Error())
	}

	s.invalidateListCache(ctx, userID)
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

var _ BankUseCase = (*BankService)(nil)
