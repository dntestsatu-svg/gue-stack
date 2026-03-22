package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/repository"
	"github.com/go-playground/validator/v10"
)

type TestingUseCase interface {
	GenerateQris(ctx context.Context, userID uint64, actorRole model.UserRole, input TestingGenerateQrisInput) (*TestingGenerateQrisResult, error)
	CheckCallbackReadiness(ctx context.Context, userID uint64, actorRole model.UserRole, input TestingCallbackReadinessInput) (*TestingCallbackReadinessResult, error)
}

type testingHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type TestingService struct {
	tokoRepo       repository.TokoRepository
	paymentGateway PaymentGatewayUseCase
	callbackClient testingHTTPClient
	validate       *validator.Validate
	logger         *slog.Logger
}

type TestingGenerateQrisInput struct {
	TokoID    uint64 `json:"toko_id" validate:"required,gt=0"`
	Username  string `json:"username" validate:"required,max=255"`
	Amount    uint64 `json:"amount" validate:"required,gte=10000,lte=10000000"`
	Expire    *int   `json:"expire,omitempty" validate:"omitempty,gte=30,lte=86400"`
	CustomRef string `json:"custom_ref,omitempty" validate:"omitempty,max=36,alphanum"`
}

type TestingGenerateQrisResult struct {
	TokoID             uint64 `json:"toko_id"`
	TokoName           string `json:"toko_name"`
	Data               string `json:"data"`
	TrxID              string `json:"trx_id"`
	ExpiredAt          *int64 `json:"expired_at,omitempty"`
	ServerProcessingMS int64  `json:"server_processing_ms"`
}

type TestingCallbackReadinessInput struct {
	TokoID uint64 `json:"toko_id" validate:"required,gt=0"`
}

type TestingCallbackReadinessResult struct {
	TokoID             uint64 `json:"toko_id"`
	TokoName           string `json:"toko_name"`
	CallbackURL        string `json:"callback_url"`
	Ready              bool   `json:"ready"`
	Message            string `json:"message"`
	Detail             string `json:"detail,omitempty"`
	StatusCode         int    `json:"status_code"`
	ReceivedSuccess    bool   `json:"received_success"`
	ResponseExcerpt    string `json:"response_excerpt,omitempty"`
	CallbackLatencyMS  int64  `json:"callback_latency_ms"`
	ServerProcessingMS int64  `json:"server_processing_ms"`
}

type testingCallbackProbePayload struct {
	Type      string `json:"type"`
	Source    string `json:"source"`
	TokoID    uint64 `json:"toko_id"`
	TokoName  string `json:"toko_name"`
	Timestamp string `json:"timestamp"`
}

type testingCallbackProbeResponse struct {
	Success bool `json:"success"`
}

func NewTestingService(
	tokoRepo repository.TokoRepository,
	paymentGateway PaymentGatewayUseCase,
	callbackClient testingHTTPClient,
	logger *slog.Logger,
) *TestingService {
	if callbackClient == nil {
		callbackClient = &http.Client{Timeout: 5 * time.Second}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &TestingService{
		tokoRepo:       tokoRepo,
		paymentGateway: paymentGateway,
		callbackClient: callbackClient,
		validate:       validator.New(validator.WithRequiredStructEnabled()),
		logger:         logger,
	}
}

func (s *TestingService) GenerateQris(ctx context.Context, userID uint64, actorRole model.UserRole, input TestingGenerateQrisInput) (*TestingGenerateQrisResult, error) {
	startedAt := time.Now()
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}
	if s.paymentGateway == nil {
		return nil, apperror.New(http.StatusInternalServerError, "payment gateway service is not configured", nil)
	}

	toko, err := s.tokoRepo.GetAccessibleByID(ctx, userID, actorRole, input.TokoID)
	if err != nil {
		return nil, mapTestingTokoAccessError(err)
	}

	result, err := s.paymentGateway.Generate(ctx, toko.ID, GeneratePaymentInput{
		Username:  strings.TrimSpace(input.Username),
		Amount:    input.Amount,
		Expire:    input.Expire,
		CustomRef: strings.TrimSpace(input.CustomRef),
	})
	if err != nil {
		return nil, err
	}

	return &TestingGenerateQrisResult{
		TokoID:             toko.ID,
		TokoName:           toko.Name,
		Data:               result.Data,
		TrxID:              result.TrxID,
		ExpiredAt:          result.ExpiredAt,
		ServerProcessingMS: time.Since(startedAt).Milliseconds(),
	}, nil
}

func (s *TestingService) CheckCallbackReadiness(ctx context.Context, userID uint64, actorRole model.UserRole, input TestingCallbackReadinessInput) (*TestingCallbackReadinessResult, error) {
	startedAt := time.Now()
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}

	toko, err := s.tokoRepo.GetAccessibleByID(ctx, userID, actorRole, input.TokoID)
	if err != nil {
		return nil, mapTestingTokoAccessError(err)
	}

	callbackURL := ""
	if toko.CallbackURL != nil {
		callbackURL = strings.TrimSpace(*toko.CallbackURL)
	}

	baseResult := &TestingCallbackReadinessResult{
		TokoID:      toko.ID,
		TokoName:    toko.Name,
		CallbackURL: callbackURL,
		StatusCode:  0,
	}
	finalize := func(result *TestingCallbackReadinessResult) *TestingCallbackReadinessResult {
		result.ServerProcessingMS = time.Since(startedAt).Milliseconds()
		return result
	}

	if callbackURL == "" {
		baseResult.Ready = false
		baseResult.Message = "API kamu sepertinya belum terintegrasi dengan baik."
		baseResult.Detail = "Callback URL toko belum dikonfigurasi."
		return finalize(baseResult), nil
	}

	payloadBytes, err := json.Marshal(testingCallbackProbePayload{
		Type:      "integration_check",
		Source:    "gue",
		TokoID:    toko.ID,
		TokoName:  toko.Name,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to build callback readiness payload", err.Error())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, bytes.NewReader(payloadBytes))
	if err != nil {
		baseResult.Ready = false
		baseResult.Message = "API kamu sepertinya belum terintegrasi dengan baik."
		baseResult.Detail = "Callback URL tidak valid atau tidak dapat dipakai untuk probe."
		s.logger.Warn("testing callback probe request creation failed", "toko_id", toko.ID, "callback_url", callbackURL, "error", err)
		return finalize(baseResult), nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "gue-testing-probe/1.0")
	req.Header.Set("X-GUE-Testing", "true")

	probeStarted := time.Now()
	resp, err := s.callbackClient.Do(req)
	baseResult.CallbackLatencyMS = time.Since(probeStarted).Milliseconds()
	if err != nil {
		baseResult.Ready = false
		baseResult.Message = "API kamu sepertinya belum terintegrasi dengan baik."
		baseResult.Detail = "Server callback toko tidak dapat dijangkau atau timeout."
		s.logger.Warn("testing callback probe failed", "toko_id", toko.ID, "callback_url", callbackURL, "error", err)
		return finalize(baseResult), nil
	}
	defer resp.Body.Close()

	baseResult.StatusCode = resp.StatusCode
	bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		baseResult.Ready = false
		baseResult.Message = "API kamu sepertinya belum terintegrasi dengan baik."
		baseResult.Detail = "Response callback tidak dapat dibaca dengan benar."
		s.logger.Warn("testing callback probe response read failed", "toko_id", toko.ID, "callback_url", callbackURL, "error", readErr)
		return finalize(baseResult), nil
	}

	var probeResponse testingCallbackProbeResponse
	if err := json.Unmarshal(bodyBytes, &probeResponse); err != nil {
		baseResult.Ready = false
		baseResult.Message = "API kamu sepertinya belum terintegrasi dengan baik."
		baseResult.Detail = "Callback URL belum mengembalikan JSON {\"success\": true}."
		baseResult.ResponseExcerpt = sanitizeProbeExcerpt(bodyBytes)
		return finalize(baseResult), nil
	}

	baseResult.ReceivedSuccess = probeResponse.Success
	baseResult.ResponseExcerpt = sanitizeProbeExcerpt(bodyBytes)
	if resp.StatusCode != http.StatusOK || !probeResponse.Success {
		baseResult.Ready = false
		baseResult.Message = "API kamu sepertinya belum terintegrasi dengan baik."
		baseResult.Detail = "Callback harus merespons HTTP 200 dengan body {\"success\": true}."
		return finalize(baseResult), nil
	}

	baseResult.Ready = true
	baseResult.Message = "API kamu sudah ready."
	baseResult.Detail = "Callback URL merespons sesuai kontrak integrasi."
	return finalize(baseResult), nil
}

func mapTestingTokoAccessError(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return apperror.New(http.StatusNotFound, "toko not found", nil)
	}
	return apperror.New(http.StatusInternalServerError, "failed to access toko", err.Error())
}

func sanitizeProbeExcerpt(raw []byte) string {
	compacted := strings.Join(strings.Fields(strings.TrimSpace(string(raw))), " ")
	if len(compacted) <= 180 {
		return compacted
	}
	return compacted[:177] + "..."
}

var _ TestingUseCase = (*TestingService)(nil)
