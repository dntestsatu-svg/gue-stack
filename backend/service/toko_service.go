package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/repository"
	"github.com/go-playground/validator/v10"
)

type TokoUseCase interface {
	ListByUser(ctx context.Context, userID uint64) ([]TokoDTO, error)
	CreateForUser(ctx context.Context, userID uint64, input CreateTokoInput) (*TokoDTO, error)
	ListBalancesByUser(ctx context.Context, userID uint64) ([]TokoBalanceDTO, error)
	ManualSettlement(ctx context.Context, actorRole model.UserRole, tokoID uint64, input ManualSettlementInput) (*TokoBalanceDTO, error)
}

type TokoService struct {
	tokoRepo      repository.TokoRepository
	balanceRepo   repository.BalanceRepository
	validate      *validator.Validate
	maxTokos      int
	defaultCharge int
}

type CreateTokoInput struct {
	Name        string  `json:"name" validate:"required,min=2,max=255"`
	CallbackURL *string `json:"callback_url,omitempty" validate:"omitempty,url,max=255"`
}

type ManualSettlementInput struct {
	SettlementBalance float64 `json:"settlement_balance" validate:"required,gte=0,lte=999999999999.99"`
	AvailableBalance  float64 `json:"available_balance" validate:"required,gte=0,lte=999999999999.99"`
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

func NewTokoService(
	tokoRepo repository.TokoRepository,
	balanceRepo repository.BalanceRepository,
	maxTokos int,
	defaultCharge int,
) *TokoService {
	if maxTokos <= 0 {
		maxTokos = 3
	}
	if defaultCharge <= 0 {
		defaultCharge = 3
	}
	return &TokoService{
		tokoRepo:      tokoRepo,
		balanceRepo:   balanceRepo,
		validate:      validator.New(validator.WithRequiredStructEnabled()),
		maxTokos:      maxTokos,
		defaultCharge: defaultCharge,
	}
}

func (s *TokoService) ListByUser(ctx context.Context, userID uint64) ([]TokoDTO, error) {
	items, err := s.tokoRepo.ListByUser(ctx, userID)
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

func (s *TokoService) CreateForUser(ctx context.Context, userID uint64, input CreateTokoInput) (*TokoDTO, error) {
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

	return &TokoDTO{
		ID:          toko.ID,
		Name:        toko.Name,
		Token:       toko.Token,
		Charge:      toko.Charge,
		CallbackURL: toko.CallbackURL,
	}, nil
}

func (s *TokoService) ListBalancesByUser(ctx context.Context, userID uint64) ([]TokoBalanceDTO, error) {
	items, err := s.balanceRepo.ListByUser(ctx, userID)
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
	if actorRole != model.UserRoleDev && actorRole != model.UserRoleSuperAdmin {
		return nil, apperror.New(http.StatusForbidden, "insufficient role permission", "manual settlement only allowed for dev or superadmin")
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

	if err := s.balanceRepo.UpsertByTokoID(ctx, tokoID, input.SettlementBalance, input.AvailableBalance); err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to apply settlement", err.Error())
	}

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

var _ TokoUseCase = (*TokoService)(nil)
