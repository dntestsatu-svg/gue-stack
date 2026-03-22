package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/example/gue/backend/cache"
	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/repository"
	"github.com/go-playground/validator/v10"
	"github.com/shopspring/decimal"
)

type TokoUseCase interface {
	ListByUser(ctx context.Context, userID uint64, actorRole model.UserRole) ([]TokoDTO, error)
	Workspace(ctx context.Context, userID uint64, actorRole model.UserRole, query TokoWorkspaceQuery) (*TokoWorkspacePage, error)
	CreateForUser(ctx context.Context, userID uint64, actorRole model.UserRole, input CreateTokoInput) (*TokoDTO, error)
	Update(ctx context.Context, userID uint64, actorRole model.UserRole, tokoID uint64, input UpdateTokoInput) (*TokoDTO, error)
	RegenerateToken(ctx context.Context, userID uint64, actorRole model.UserRole, tokoID uint64) (*TokoDTO, error)
	ListBalancesByUser(ctx context.Context, userID uint64, actorRole model.UserRole) ([]TokoBalanceDTO, error)
	ManualSettlement(ctx context.Context, actorRole model.UserRole, tokoID uint64, input ManualSettlementInput) (*TokoBalanceDTO, error)
}

type TokoService struct {
	tokoRepo      repository.TokoRepository
	balanceRepo   repository.BalanceRepository
	cache         cache.Cache
	cacheEnabled  bool
	listCacheTTL  time.Duration
	logger        *slog.Logger
	validate      *validator.Validate
	maxTokos      int
	defaultCharge int
	settlementFee decimal.Decimal
}

type CreateTokoInput struct {
	Name        string  `json:"name" validate:"required,min=2,max=255"`
	CallbackURL *string `json:"callback_url,omitempty" validate:"omitempty,url,max=255"`
}

type UpdateTokoInput struct {
	Name        string  `json:"name" validate:"required,min=2,max=255"`
	CallbackURL *string `json:"callback_url,omitempty" validate:"omitempty,url,max=255"`
}

type ManualSettlementInput struct {
	SettlementBalance float64 `json:"settlement_balance" validate:"required,gt=0,lte=999999999999.99"`
}

type TokoWorkspaceQuery struct {
	Limit      int
	Offset     int
	SearchTerm string
}

type TokoDTO struct {
	ID          uint64  `json:"id"`
	Name        string  `json:"name"`
	Token       string  `json:"token"`
	Charge      int     `json:"charge"`
	CallbackURL *string `json:"callback_url,omitempty"`
}

type TokoBalanceDTO struct {
	TokoID            uint64  `json:"toko_id"`
	TokoName          string  `json:"toko_name"`
	SettlementBalance float64 `json:"settlement_balance"`
	AvailableBalance  float64 `json:"available_balance"`
	UpdatedAt         string  `json:"updated_at"`
}

type TokoWorkspaceItemDTO struct {
	ID                uint64  `json:"id"`
	Name              string  `json:"name"`
	Token             string  `json:"token"`
	Charge            int     `json:"charge"`
	CallbackURL       *string `json:"callback_url,omitempty"`
	SettlementBalance float64 `json:"settlement_balance"`
	AvailableBalance  float64 `json:"available_balance"`
	UpdatedAt         string  `json:"updated_at"`
}

type TokoWorkspaceSummaryDTO struct {
	TotalTokos            uint64  `json:"total_tokos"`
	TotalSettlementAmount float64 `json:"total_settlement_balance"`
	TotalAvailableAmount  float64 `json:"total_available_balance"`
}

type TokoWorkspacePage struct {
	Items   []TokoWorkspaceItemDTO  `json:"items"`
	Summary TokoWorkspaceSummaryDTO `json:"summary"`
	Total   uint64                  `json:"total"`
	Limit   int                     `json:"limit"`
	Offset  int                     `json:"offset"`
	HasMore bool                    `json:"has_more"`
}

func NewTokoService(
	tokoRepo repository.TokoRepository,
	balanceRepo repository.BalanceRepository,
	cacheStore cache.Cache,
	cacheEnabled bool,
	listCacheTTL time.Duration,
	maxTokos int,
	defaultCharge int,
	logger *slog.Logger,
) *TokoService {
	if maxTokos <= 0 {
		maxTokos = 3
	}
	if defaultCharge <= 0 {
		defaultCharge = 3
	}
	if listCacheTTL <= 0 {
		listCacheTTL = 5 * time.Minute
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &TokoService{
		tokoRepo:      tokoRepo,
		balanceRepo:   balanceRepo,
		cache:         cacheStore,
		cacheEnabled:  cacheEnabled,
		listCacheTTL:  listCacheTTL,
		logger:        logger,
		validate:      validator.New(validator.WithRequiredStructEnabled()),
		maxTokos:      maxTokos,
		defaultCharge: defaultCharge,
		settlementFee: decimal.NewFromInt(3000),
	}
}

func (s *TokoService) ListByUser(ctx context.Context, userID uint64, actorRole model.UserRole) ([]TokoDTO, error) {
	items, err := s.tokoRepo.ListByUser(ctx, userID, actorRole)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to list tokos", err.Error())
	}

	result := make([]TokoDTO, 0, len(items))
	for _, item := range items {
		result = append(result, TokoDTO{
			ID:          item.ID,
			Name:        item.Name,
			Token:       item.Token,
			Charge:      item.Charge,
			CallbackURL: item.CallbackURL,
		})
	}
	return result, nil
}

func (s *TokoService) Workspace(ctx context.Context, userID uint64, actorRole model.UserRole, query TokoWorkspaceQuery) (*TokoWorkspacePage, error) {
	filter := normalizeTokoWorkspaceQuery(query)
	cacheKey := s.workspaceCacheKey(ctx, userID, actorRole, filter)
	if cached, ok := getCachedJSON[TokoWorkspacePage](ctx, s.cache, s.cacheEnabled, cacheKey, s.logger); ok {
		return cached, nil
	}

	summary, err := s.tokoRepo.SummarizeWorkspaceByUser(ctx, userID, actorRole, repository.TokoWorkspaceFilter{
		SearchTerm: filter.SearchTerm,
	})
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to summarize toko workspace", err.Error())
	}

	items, err := s.tokoRepo.ListWorkspaceByUser(ctx, userID, actorRole, repository.TokoWorkspaceFilter{
		Limit:      filter.Limit,
		Offset:     filter.Offset,
		SearchTerm: filter.SearchTerm,
	})
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to list toko workspace", err.Error())
	}

	result := make([]TokoWorkspaceItemDTO, 0, len(items))
	for _, item := range items {
		result = append(result, TokoWorkspaceItemDTO{
			ID:                item.ID,
			Name:              item.Name,
			Token:             item.Token,
			Charge:            item.Charge,
			CallbackURL:       item.CallbackURL,
			SettlementBalance: item.SettlementBalance,
			AvailableBalance:  item.AvailableBalance,
			UpdatedAt:         item.LastSettlementTime.UTC().Format(time.RFC3339),
		})
	}

	page := &TokoWorkspacePage{
		Items: result,
		Summary: TokoWorkspaceSummaryDTO{
			TotalTokos:            summary.TotalTokos,
			TotalSettlementAmount: summary.TotalSettlementAmount,
			TotalAvailableAmount:  summary.TotalAvailableAmount,
		},
		Total:   summary.TotalTokos,
		Limit:   filter.Limit,
		Offset:  filter.Offset,
		HasMore: uint64(filter.Offset+len(result)) < summary.TotalTokos,
	}
	setCachedJSON(ctx, s.cache, s.cacheEnabled, cacheKey, page, s.listCacheTTL, s.logger)
	return page, nil
}

func (s *TokoService) CreateForUser(ctx context.Context, userID uint64, actorRole model.UserRole, input CreateTokoInput) (*TokoDTO, error) {
	if !canManageTokos(actorRole) {
		return nil, apperror.New(http.StatusForbidden, "insufficient role permission", "only dev, superadmin, or admin can create toko")
	}
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}

	name := strings.TrimSpace(input.Name)
	callbackURL := trimOptionalString(input.CallbackURL)

	var createErr error
	var toko *model.Toko
	for attempt := 0; attempt < 5; attempt++ {
		token, err := generateTokoToken()
		if err != nil {
			return nil, apperror.New(http.StatusInternalServerError, "failed to generate toko token", err.Error())
		}

		toko = &model.Toko{
			Name:        name,
			Token:       token,
			Charge:      s.defaultCharge,
			CallbackURL: callbackURL,
		}

		createErr = s.tokoRepo.CreateForUserWithQuota(ctx, userID, toko, s.maxTokos)
		if createErr == nil {
			break
		}
		if errors.Is(createErr, repository.ErrQuotaExceeded) {
			return nil, apperror.New(http.StatusForbidden, "maximum toko limit reached", "max toko per user is 3")
		}
		if errors.Is(createErr, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusNotFound, "user not found", nil)
		}
		if !isDuplicateKeyError(createErr) {
			return nil, apperror.New(http.StatusInternalServerError, "failed to create toko", createErr.Error())
		}
	}
	if createErr != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to create toko", createErr.Error())
	}
	s.invalidateWorkspaceCache(ctx)

	return &TokoDTO{
		ID:          toko.ID,
		Name:        toko.Name,
		Token:       toko.Token,
		Charge:      toko.Charge,
		CallbackURL: toko.CallbackURL,
	}, nil
}

func (s *TokoService) Update(ctx context.Context, userID uint64, actorRole model.UserRole, tokoID uint64, input UpdateTokoInput) (*TokoDTO, error) {
	if !canManageTokos(actorRole) {
		return nil, apperror.New(http.StatusForbidden, "insufficient role permission", "only dev, superadmin, or admin can update toko")
	}
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}

	toko, err := s.tokoRepo.GetAccessibleByID(ctx, userID, actorRole, tokoID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusNotFound, "toko not found", nil)
		}
		return nil, apperror.New(http.StatusInternalServerError, "failed to fetch toko", err.Error())
	}

	name := strings.TrimSpace(input.Name)
	callbackURL := trimOptionalString(input.CallbackURL)
	if err := s.tokoRepo.UpdateProfile(ctx, tokoID, name, callbackURL); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusNotFound, "toko not found", nil)
		}
		return nil, apperror.New(http.StatusInternalServerError, "failed to update toko", err.Error())
	}
	s.invalidateWorkspaceCache(ctx)

	toko.Name = name
	toko.CallbackURL = callbackURL
	return &TokoDTO{
		ID:          toko.ID,
		Name:        toko.Name,
		Token:       toko.Token,
		Charge:      toko.Charge,
		CallbackURL: toko.CallbackURL,
	}, nil
}

func (s *TokoService) RegenerateToken(ctx context.Context, userID uint64, actorRole model.UserRole, tokoID uint64) (*TokoDTO, error) {
	if !canManageTokos(actorRole) {
		return nil, apperror.New(http.StatusForbidden, "insufficient role permission", "only dev, superadmin, or admin can regenerate toko token")
	}

	toko, err := s.tokoRepo.GetAccessibleByID(ctx, userID, actorRole, tokoID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusNotFound, "toko not found", nil)
		}
		return nil, apperror.New(http.StatusInternalServerError, "failed to fetch toko", err.Error())
	}

	var updateErr error
	for attempt := 0; attempt < 5; attempt++ {
		token, err := generateTokoToken()
		if err != nil {
			return nil, apperror.New(http.StatusInternalServerError, "failed to generate toko token", err.Error())
		}

		updateErr = s.tokoRepo.UpdateToken(ctx, tokoID, token)
		if updateErr == nil {
			toko.Token = token
			s.invalidateWorkspaceCache(ctx)
			return &TokoDTO{
				ID:          toko.ID,
				Name:        toko.Name,
				Token:       toko.Token,
				Charge:      toko.Charge,
				CallbackURL: toko.CallbackURL,
			}, nil
		}
		if errors.Is(updateErr, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusNotFound, "toko not found", nil)
		}
		if !isDuplicateKeyError(updateErr) {
			return nil, apperror.New(http.StatusInternalServerError, "failed to regenerate toko token", updateErr.Error())
		}
	}

	return nil, apperror.New(http.StatusInternalServerError, "failed to regenerate toko token", updateErr.Error())
}

func (s *TokoService) ListBalancesByUser(ctx context.Context, userID uint64, actorRole model.UserRole) ([]TokoBalanceDTO, error) {
	items, err := s.balanceRepo.ListByUser(ctx, userID, actorRole)
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to list toko balances", err.Error())
	}

	result := make([]TokoBalanceDTO, 0, len(items))
	for _, item := range items {
		result = append(result, TokoBalanceDTO{
			TokoID:            item.TokoID,
			TokoName:          item.TokoName,
			SettlementBalance: item.SettlementBalance,
			AvailableBalance:  item.AvailableBalance,
			UpdatedAt:         item.LastSettlementTime.UTC().Format(time.RFC3339),
		})
	}
	return result, nil
}

func (s *TokoService) ManualSettlement(ctx context.Context, actorRole model.UserRole, tokoID uint64, input ManualSettlementInput) (*TokoBalanceDTO, error) {
	if actorRole != model.UserRoleDev {
		return nil, apperror.New(http.StatusForbidden, "insufficient role permission", "manual settlement only allowed for dev")
	}
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}

	if _, err := s.tokoRepo.GetByID(ctx, tokoID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusNotFound, "toko not found", nil)
		}
		return nil, apperror.New(http.StatusInternalServerError, "failed to validate toko", err.Error())
	}

	currentBalance, err := s.balanceRepo.GetByTokoID(ctx, tokoID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusNotFound, "toko balance not found", nil)
		}
		return nil, apperror.New(http.StatusInternalServerError, "failed to fetch current toko balance", err.Error())
	}

	settlementAmount := decimal.NewFromFloat(input.SettlementBalance)
	currentSettlement := decimal.NewFromFloat(currentBalance.SettlementBalance)
	currentAvailable := decimal.NewFromFloat(currentBalance.AvailableBalance)
	nextAvailable := currentAvailable.Sub(settlementAmount).Sub(s.settlementFee)
	if nextAvailable.IsNegative() {
		return nil, apperror.New(http.StatusBadRequest, "insufficient available balance", "available balance cannot be negative after settlement and admin fee")
	}
	nextSettlement := currentSettlement.Add(settlementAmount)

	if err := s.balanceRepo.UpsertByTokoID(ctx, tokoID, nextSettlement.InexactFloat64(), nextAvailable.InexactFloat64()); err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to apply settlement", err.Error())
	}
	s.invalidateWorkspaceCache(ctx)

	record, err := s.balanceRepo.GetByTokoID(ctx, tokoID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusNotFound, "toko balance not found", nil)
		}
		return nil, apperror.New(http.StatusInternalServerError, "failed to fetch updated toko balance", err.Error())
	}

	return &TokoBalanceDTO{
		TokoID:            record.TokoID,
		TokoName:          record.TokoName,
		SettlementBalance: record.SettlementBalance,
		AvailableBalance:  record.AvailableBalance,
		UpdatedAt:         record.LastSettlementTime.UTC().Format(time.RFC3339),
	}, nil
}

func generateTokoToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate")
}

func normalizeTokoWorkspaceQuery(query TokoWorkspaceQuery) TokoWorkspaceQuery {
	if query.Limit <= 0 {
		query.Limit = 10
	}
	if query.Limit > 25 {
		query.Limit = 25
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	query.SearchTerm = strings.TrimSpace(query.SearchTerm)
	return query
}

func canManageTokos(role model.UserRole) bool {
	return role == model.UserRoleDev || role == model.UserRoleSuperAdmin || role == model.UserRoleAdmin
}

func (s *TokoService) workspaceCacheKey(ctx context.Context, userID uint64, actorRole model.UserRole, query TokoWorkspaceQuery) string {
	namespace := getCacheNamespaceToken(ctx, s.cache, s.cacheEnabled, s.workspaceNamespaceKey(), s.logger)
	return buildHashedCacheKey(
		"toko:workspace",
		"ns="+namespace,
		fmt.Sprintf("actor=%d", userID),
		"actor_role="+string(actorRole),
		fmt.Sprintf("limit=%d", query.Limit),
		fmt.Sprintf("offset=%d", query.Offset),
		"search="+strings.ToLower(strings.TrimSpace(query.SearchTerm)),
	)
}

func (s *TokoService) workspaceNamespaceKey() string {
	return "tokos:workspace:namespace"
}

func (s *TokoService) invalidateWorkspaceCache(ctx context.Context) {
	bumpCacheNamespace(ctx, s.cache, s.cacheEnabled, s.workspaceNamespaceKey(), s.logger)
}

var _ TokoUseCase = (*TokoService)(nil)
