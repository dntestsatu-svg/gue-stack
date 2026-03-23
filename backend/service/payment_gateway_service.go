package service

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/gue/backend/cache"
	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/money"
	"github.com/example/gue/backend/pkg/paymentgateway"
	"github.com/example/gue/backend/queue"
	"github.com/example/gue/backend/repository"
	"github.com/go-playground/validator/v10"
)

const (
	qrisStatusCacheTTL      = 24 * time.Hour
	qrisCallbackDeliveryTTL = 24 * time.Hour
	transferInquiryCacheTTL = 5 * time.Minute
	defaultCallbackTimeout  = 5 * time.Second
	defaultExpiryBatchSize  = 250
)

type PaymentGatewayUseCase interface {
	Generate(ctx context.Context, tokoID uint64, input GeneratePaymentInput) (*GeneratePaymentResult, error)
	CheckStatusV2(ctx context.Context, tokoID uint64, trxID string, input CheckPaymentStatusInput) (*CheckPaymentStatusResult, error)
	InquiryTransfer(ctx context.Context, tokoID uint64, input InquiryTransferInput) (*InquiryTransferResult, error)
	TransferFund(ctx context.Context, tokoID uint64, input TransferFundInput) (*TransferFundResult, error)
	CheckTransferStatus(ctx context.Context, tokoID uint64, partnerRefNo string, input CheckTransferStatusInput) (*CheckTransferStatusResult, error)
	GetBalance(ctx context.Context, input GetBalanceInput) (*GetBalanceResult, error)
	EnqueueQrisCallback(ctx context.Context, payload queue.QrisCallbackTaskPayload) error
	EnqueueTransferCallback(ctx context.Context, payload queue.TransferCallbackTaskPayload) error
	ValidateCallbackSecret(secret string) error
}

type paymentCallbackHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type PaymentGatewayService struct {
	gatewayClient      paymentgateway.Client
	tokoRepo           repository.TokoRepository
	transactionRepo    repository.TransactionRepository
	queueProducer      queue.Producer
	validate           *validator.Validate
	defaultClient      string
	defaultKey         string
	merchantUUID       string
	webhookSecret      string
	platformFeePercent uint64
	logger             *slog.Logger
	cache              cache.Cache
	cacheEnabled       bool
	callbackClient     paymentCallbackHTTPClient
	statusCacheTTL     time.Duration
	callbackTTL        time.Duration
}

type GeneratePaymentInput struct {
	Username  string `json:"username" validate:"required,max=255"`
	Amount    uint64 `json:"amount" validate:"required,gte=10000,lte=10000000"`
	Expire    *int   `json:"expire,omitempty" validate:"omitempty,gte=30,lte=86400"`
	CustomRef string `json:"custom_ref,omitempty" validate:"omitempty,max=36,alphanum"`
}

type GeneratePaymentResult struct {
	Data      string `json:"data"`
	TrxID     string `json:"trx_id"`
	ExpiredAt *int64 `json:"expired_at,omitempty"`
}

type CheckPaymentStatusInput struct {
	Client string `json:"client" validate:"omitempty,max=255"`
}

type CheckPaymentStatusResult struct {
	Amount     uint64 `json:"amount"`
	MerchantID string `json:"merchant_id"`
	TrxID      string `json:"trx_id"`
	RRN        string `json:"rrn,omitempty"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at,omitempty"`
	FinishAt   string `json:"finish_at,omitempty"`
}

type InquiryTransferInput struct {
	Client        string  `json:"client" validate:"omitempty,max=255"`
	Amount        uint64  `json:"amount" validate:"required,gt=0"`
	BankCode      string  `json:"bank_code" validate:"required,max=32"`
	AccountNumber string  `json:"account_number" validate:"required,max=64"`
	Type          int     `json:"type" validate:"required,oneof=1 2"`
	Note          *string `json:"note,omitempty" validate:"omitempty,max=64,alphanum"`
	ClientRefID   *string `json:"client_ref_id,omitempty" validate:"omitempty,max=64,alphanum"`
}

type InquiryTransferResult struct {
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

type TransferFundInput struct {
	Client        string `json:"client" validate:"omitempty,max=255"`
	Amount        uint64 `json:"amount" validate:"required,gt=0"`
	BankCode      string `json:"bank_code" validate:"required,max=32"`
	AccountNumber string `json:"account_number" validate:"required,max=64"`
	Type          int    `json:"type" validate:"required,oneof=1 2"`
	InquiryID     uint64 `json:"inquiry_id" validate:"required,gt=0"`
}

type TransferFundResult struct {
	Status bool `json:"status"`
}

type CheckTransferStatusInput struct {
	Client string `json:"client" validate:"omitempty,max=255"`
}

type CheckTransferStatusResult struct {
	Amount       uint64 `json:"amount"`
	Fee          uint64 `json:"fee"`
	PartnerRefNo string `json:"partner_ref_no"`
	MerchantUUID string `json:"merchant_uuid"`
	Status       string `json:"status"`
}

type GetBalanceInput struct {
	Client string `json:"client" validate:"omitempty,max=255"`
}

type GetBalanceResult struct {
	Status         string `json:"status"`
	PendingBalance uint64 `json:"pending_balance"`
	SettleBalance  uint64 `json:"settle_balance"`
}

type qrisStatusCacheEntry struct {
	Amount     uint64 `json:"amount"`
	TerminalID string `json:"terminal_id,omitempty"`
	MerchantID string `json:"merchant_id"`
	TrxID      string `json:"trx_id"`
	RRN        string `json:"rrn,omitempty"`
	CustomRef  string `json:"custom_ref,omitempty"`
	Vendor     string `json:"vendor,omitempty"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at,omitempty"`
	FinishAt   string `json:"finish_at,omitempty"`
}

type merchantQrisCallbackPayload struct {
	Amount     uint64 `json:"amount"`
	TerminalID string `json:"terminal_id"`
	MerchantID string `json:"merchant_id"`
	TrxID      string `json:"trx_id"`
	RRN        string `json:"rrn"`
	CustomRef  string `json:"custom_ref,omitempty"`
	Vendor     string `json:"vendor,omitempty"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at,omitempty"`
	FinishAt   string `json:"finish_at,omitempty"`
}

type cachedTransferInquiry struct {
	TokoID        uint64 `json:"toko_id"`
	BankCode      string `json:"bank_code"`
	BankName      string `json:"bank_name"`
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
	Amount        uint64 `json:"amount"`
	Fee           uint64 `json:"fee"`
	InquiryID     uint64 `json:"inquiry_id"`
	PartnerRefNo  string `json:"partner_ref_no"`
}

func NewPaymentGatewayService(
	gatewayClient paymentgateway.Client,
	tokoRepo repository.TokoRepository,
	transactionRepo repository.TransactionRepository,
	queueProducer queue.Producer,
	defaultClient string,
	defaultKey string,
	merchantUUID string,
	webhookSecret string,
	platformFeePercent int,
	logger *slog.Logger,
	cacheStore cache.Cache,
	cacheEnabled bool,
	callbackClient paymentCallbackHTTPClient,
) *PaymentGatewayService {
	if platformFeePercent < 0 {
		platformFeePercent = 0
	}
	if logger == nil {
		logger = slog.Default()
	}
	if callbackClient == nil {
		callbackClient = &http.Client{Timeout: defaultCallbackTimeout}
	}
	return &PaymentGatewayService{
		gatewayClient:      gatewayClient,
		tokoRepo:           tokoRepo,
		transactionRepo:    transactionRepo,
		queueProducer:      queueProducer,
		validate:           validator.New(validator.WithRequiredStructEnabled()),
		defaultClient:      strings.TrimSpace(defaultClient),
		defaultKey:         strings.TrimSpace(defaultKey),
		merchantUUID:       strings.TrimSpace(merchantUUID),
		webhookSecret:      strings.TrimSpace(webhookSecret),
		platformFeePercent: uint64(platformFeePercent),
		logger:             logger,
		cache:              cacheStore,
		cacheEnabled:       cacheEnabled,
		callbackClient:     callbackClient,
		statusCacheTTL:     qrisStatusCacheTTL,
		callbackTTL:        qrisCallbackDeliveryTTL,
	}
}

func (s *PaymentGatewayService) Generate(ctx context.Context, tokoID uint64, input GeneratePaymentInput) (*GeneratePaymentResult, error) {
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}
	if _, err := s.tokoRepo.GetByID(ctx, tokoID); err != nil {
		return nil, s.mapRepositoryError("toko not found", err)
	}

	resp, err := s.gatewayClient.Generate(ctx, paymentgateway.GenerateRequest{
		Username:  input.Username,
		Amount:    input.Amount,
		UUID:      s.merchantUUID,
		Expire:    input.Expire,
		CustomRef: input.CustomRef,
	})
	if err != nil {
		return nil, s.mapGatewayError("failed to generate payment", err)
	}

	player := input.Username
	reference := resp.TrxID
	code := strings.TrimSpace(input.CustomRef)
	barcode := strings.TrimSpace(resp.Data)
	trx := &model.Transaction{
		TokoID:      tokoID,
		Player:      &player,
		Type:        model.TransactionTypeDeposit,
		Status:      model.TransactionStatusPending,
		Reference:   &reference,
		Amount:      input.Amount,
		PlatformFee: 0,
		Netto:       input.Amount,
	}
	if code != "" {
		trx.Code = &code
	}
	if barcode != "" {
		trx.Barcode = &barcode
	}
	if err := s.transactionRepo.Create(ctx, trx); err != nil {
		return nil, apperror.New(http.StatusInternalServerError, "failed to persist payment transaction", err.Error())
	}

	return &GeneratePaymentResult{
		Data:      resp.Data,
		TrxID:     resp.TrxID,
		ExpiredAt: resp.ExpiredAt,
	}, nil
}

func (s *PaymentGatewayService) CheckStatusV2(ctx context.Context, tokoID uint64, trxID string, input CheckPaymentStatusInput) (*CheckPaymentStatusResult, error) {
	trxID = strings.TrimSpace(trxID)
	if trxID == "" {
		return nil, apperror.New(http.StatusBadRequest, "trx_id is required", nil)
	}
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}

	localTrx, err := s.transactionRepo.GetByReferenceAndToko(ctx, trxID, tokoID)
	if err != nil {
		return nil, s.mapRepositoryError("transaction not found", err)
	}

	if cached, ok := s.getCachedQrisStatus(ctx, trxID); ok {
		if status := normalizeTransactionStatus(cached.Status); isFinalTransactionStatus(status) {
			if _, err := s.applySettlementUpdate(ctx, localTrx, status); err != nil {
				return nil, err
			}
		}
		return s.mapQrisStatusCacheEntryToResult(*cached), nil
	}

	client := fallback(input.Client, s.defaultClient)
	resp, err := s.gatewayClient.CheckStatusV2(ctx, trxID, paymentgateway.CheckStatusRequest{
		UUID:   s.merchantUUID,
		Client: client,
	})
	if err != nil {
		return nil, s.mapGatewayError("failed to check payment status", err)
	}

	status := normalizeTransactionStatus(resp.Status)
	cacheEntry := s.buildCheckStatusCacheEntry(localTrx, resp)
	s.cacheQrisStatus(ctx, trxID, cacheEntry)
	if isFinalTransactionStatus(status) {
		if _, err := s.applySettlementUpdate(ctx, localTrx, status); err != nil {
			return nil, err
		}
	}

	return s.mapQrisStatusCacheEntryToResult(cacheEntry), nil
}

func (s *PaymentGatewayService) InquiryTransfer(ctx context.Context, tokoID uint64, input InquiryTransferInput) (*InquiryTransferResult, error) {
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}
	if _, err := s.tokoRepo.GetByID(ctx, tokoID); err != nil {
		return nil, s.mapRepositoryError("toko not found", err)
	}

	client := fallback(input.Client, s.defaultClient)
	if client == "" || s.defaultKey == "" {
		return nil, apperror.New(http.StatusBadRequest, "client and client_key are required", nil)
	}

	resp, err := s.gatewayClient.InquiryTransfer(ctx, paymentgateway.InquiryTransferRequest{
		Client:        client,
		ClientKey:     s.defaultKey,
		UUID:          s.merchantUUID,
		Amount:        input.Amount,
		BankCode:      input.BankCode,
		AccountNumber: input.AccountNumber,
		Type:          input.Type,
		Note:          input.Note,
		ClientRefID:   input.ClientRefID,
	})
	if err != nil {
		return nil, s.mapGatewayError("failed to inquiry transfer", err)
	}
	s.cacheTransferInquiry(ctx, tokoID, resp)

	return &InquiryTransferResult{
		AccountNumber: resp.AccountNumber,
		AccountName:   resp.AccountName,
		BankCode:      resp.BankCode,
		BankName:      resp.BankName,
		PartnerRefNo:  resp.PartnerRefNo,
		VendorRefNo:   resp.VendorRefNo,
		Amount:        resp.Amount,
		Fee:           resp.Fee,
		InquiryID:     resp.InquiryID,
	}, nil
}

func (s *PaymentGatewayService) TransferFund(ctx context.Context, tokoID uint64, input TransferFundInput) (*TransferFundResult, error) {
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}
	if _, err := s.tokoRepo.GetByID(ctx, tokoID); err != nil {
		return nil, s.mapRepositoryError("toko not found", err)
	}

	client := fallback(input.Client, s.defaultClient)
	if client == "" || s.defaultKey == "" {
		return nil, apperror.New(http.StatusBadRequest, "client and client_key are required", nil)
	}

	cachedInquiry, ok := s.getCachedTransferInquiry(ctx, tokoID, input.BankCode, input.AccountNumber, input.Amount, input.InquiryID)
	if !ok || cachedInquiry == nil {
		return nil, apperror.New(http.StatusBadRequest, "transfer inquiry expired", "silakan lakukan inquiry ulang sebelum transfer")
	}
	if cachedInquiry.PartnerRefNo == "" {
		return nil, apperror.New(http.StatusBadRequest, "invalid transfer inquiry state", "partner_ref_no dari inquiry belum tersedia")
	}

	if _, err := s.transactionRepo.GetByReferenceAndToko(ctx, cachedInquiry.PartnerRefNo, tokoID); err == nil {
		return nil, apperror.New(http.StatusConflict, "transfer already requested", "permintaan transfer untuk inquiry ini sudah pernah dibuat")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.New(http.StatusInternalServerError, "failed to verify existing transfer transaction", err.Error())
	}

	fee := cachedInquiry.Fee
	reference := cachedInquiry.PartnerRefNo
	trx := &model.Transaction{
		TokoID:        tokoID,
		Type:          model.TransactionTypeWithdraw,
		Status:        model.TransactionStatusPending,
		Reference:     &reference,
		Amount:        cachedInquiry.Amount,
		FeeWithdrawal: &fee,
		PlatformFee:   0,
		Netto:         cachedInquiry.Amount,
	}
	if err := s.transactionRepo.CreatePendingWithdrawAndReserveSettlement(ctx, trx); err != nil {
		if errors.Is(err, repository.ErrInsufficientBalance) {
			return nil, apperror.New(http.StatusBadRequest, "insufficient settlement balance", "saldo settlement toko tidak mencukupi untuk transfer ini")
		}
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusNotFound, "toko balance not found", nil)
		}
		return nil, apperror.New(http.StatusInternalServerError, "failed to reserve settlement balance", err.Error())
	}

	resp, err := s.gatewayClient.TransferFund(ctx, paymentgateway.TransferFundRequest{
		Client:        client,
		ClientKey:     s.defaultKey,
		UUID:          s.merchantUUID,
		Amount:        cachedInquiry.Amount,
		BankCode:      cachedInquiry.BankCode,
		AccountNumber: cachedInquiry.AccountNumber,
		Type:          input.Type,
		InquiryID:     input.InquiryID,
	})
	if err != nil {
		_, _ = s.transactionRepo.FinalizeWithdrawIfPending(ctx, trx.ID, model.TransactionStatusFailed)
		return nil, s.mapGatewayError("failed to transfer fund", err)
	}

	if !resp.Status {
		_, _ = s.transactionRepo.FinalizeWithdrawIfPending(ctx, trx.ID, model.TransactionStatusFailed)
		return nil, apperror.New(http.StatusBadGateway, "transfer rejected", "error")
	}

	s.clearTransferInquiryCache(ctx, tokoID, input.BankCode, input.AccountNumber, input.Amount, input.InquiryID)

	return &TransferFundResult{Status: resp.Status}, nil
}

func (s *PaymentGatewayService) CheckTransferStatus(ctx context.Context, tokoID uint64, partnerRefNo string, input CheckTransferStatusInput) (*CheckTransferStatusResult, error) {
	partnerRefNo = strings.TrimSpace(partnerRefNo)
	if partnerRefNo == "" {
		return nil, apperror.New(http.StatusBadRequest, "partner_ref_no is required", nil)
	}
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}

	localTrx, err := s.transactionRepo.GetByReferenceAndToko(ctx, partnerRefNo, tokoID)
	if err != nil {
		return nil, s.mapRepositoryError("transaction not found", err)
	}

	client := fallback(input.Client, s.defaultClient)
	resp, err := s.gatewayClient.CheckTransferStatus(ctx, partnerRefNo, paymentgateway.CheckTransferStatusRequest{
		Client: client,
		UUID:   s.merchantUUID,
	})
	if err != nil {
		return nil, s.mapGatewayError("failed to check transfer status", err)
	}

	status := normalizeTransactionStatus(resp.Status)
	if isFinalTransactionStatus(status) {
		if _, err := s.finalizeWithdrawStatus(ctx, localTrx, status); err != nil {
			return nil, err
		}
	}

	return &CheckTransferStatusResult{
		Amount:       resp.Amount,
		Fee:          resp.Fee,
		PartnerRefNo: resp.PartnerRefNo,
		MerchantUUID: resp.MerchantUUID,
		Status:       resp.Status,
	}, nil
}

func (s *PaymentGatewayService) GetBalance(ctx context.Context, input GetBalanceInput) (*GetBalanceResult, error) {
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}

	client := fallback(input.Client, s.defaultClient)
	if client == "" {
		return nil, apperror.New(http.StatusBadRequest, "client is required", nil)
	}

	resp, err := s.gatewayClient.GetBalance(ctx, s.merchantUUID, paymentgateway.GetBalanceRequest{Client: client})
	if err != nil {
		return nil, s.mapGatewayError("failed to fetch merchant balance", err)
	}

	return &GetBalanceResult{
		Status:         resp.Status,
		PendingBalance: resp.PendingBalance,
		SettleBalance:  resp.SettleBalance,
	}, nil
}

func (s *PaymentGatewayService) EnqueueQrisCallback(ctx context.Context, payload queue.QrisCallbackTaskPayload) error {
	if err := s.validate.Struct(payload); err != nil {
		return apperror.New(http.StatusBadRequest, "invalid callback payload", err.Error())
	}
	if err := s.ensureMerchantID(payload.MerchantID); err != nil {
		return err
	}
	if s.queueProducer == nil {
		return apperror.New(http.StatusInternalServerError, "callback queue producer is not configured", nil)
	}
	if err := s.queueProducer.EnqueueQrisCallback(ctx, payload); err != nil {
		return apperror.New(http.StatusInternalServerError, "failed to enqueue qris callback", err.Error())
	}
	return nil
}

func (s *PaymentGatewayService) EnqueueTransferCallback(ctx context.Context, payload queue.TransferCallbackTaskPayload) error {
	if err := s.validate.Struct(payload); err != nil {
		return apperror.New(http.StatusBadRequest, "invalid callback payload", err.Error())
	}
	if err := s.ensureMerchantID(payload.MerchantID); err != nil {
		return err
	}
	if s.queueProducer == nil {
		return apperror.New(http.StatusInternalServerError, "callback queue producer is not configured", nil)
	}
	if err := s.queueProducer.EnqueueTransferCallback(ctx, payload); err != nil {
		return apperror.New(http.StatusInternalServerError, "failed to enqueue transfer callback", err.Error())
	}
	return nil
}

func (s *PaymentGatewayService) ProcessQrisCallback(ctx context.Context, payload queue.QrisCallbackTaskPayload) error {
	if err := s.validate.Struct(payload); err != nil {
		return apperror.New(http.StatusBadRequest, "invalid callback payload", err.Error())
	}
	if err := s.ensureMerchantID(payload.MerchantID); err != nil {
		return err
	}

	s.logger.Info(
		"qris webhook received",
		"trx_id", payload.TrxID,
		"status", payload.Status,
		"amount", payload.Amount,
		"vendor", payload.Vendor,
	)

	status := normalizeTransactionStatus(payload.Status)
	if status == "" {
		return apperror.New(http.StatusBadRequest, "invalid callback status", payload.Status)
	}
	cacheEntry := s.buildWebhookCacheEntry(payload)
	s.cacheQrisStatus(ctx, payload.TrxID, cacheEntry)

	trx, err := s.transactionRepo.GetByReference(ctx, payload.TrxID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.Warn("qris callback transaction not found", "trx_id", payload.TrxID)
			return nil
		}
		return apperror.New(http.StatusInternalServerError, "failed to load callback transaction", err.Error())
	}

	if _, err := s.applySettlementUpdate(ctx, trx, status); err != nil {
		return err
	}
	if !isFinalTransactionStatus(status) {
		return nil
	}

	toko, err := s.tokoRepo.GetByID(ctx, trx.TokoID)
	if err != nil {
		return s.mapRepositoryError("toko not found", err)
	}

	return s.sendMerchantQrisCallback(ctx, toko, cacheEntry)
}

func (s *PaymentGatewayService) ProcessTransferCallback(ctx context.Context, payload queue.TransferCallbackTaskPayload) error {
	if err := s.validate.Struct(payload); err != nil {
		return apperror.New(http.StatusBadRequest, "invalid callback payload", err.Error())
	}
	if err := s.ensureMerchantID(payload.MerchantID); err != nil {
		return err
	}

	status := normalizeTransactionStatus(payload.Status)
	if status == "" {
		return apperror.New(http.StatusBadRequest, "invalid callback status", payload.Status)
	}

	trx, err := s.transactionRepo.GetByReference(ctx, payload.PartnerRefNo)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.Warn("transfer callback transaction not found", "partner_ref_no", payload.PartnerRefNo)
			return nil
		}
		return apperror.New(http.StatusInternalServerError, "failed to load callback transfer", err.Error())
	}

	_, err = s.finalizeWithdrawStatus(ctx, trx, status)
	return err
}

func (s *PaymentGatewayService) ValidateCallbackSecret(secret string) error {
	expected := strings.TrimSpace(s.webhookSecret)
	if expected == "" {
		return nil
	}
	incoming := strings.TrimSpace(secret)
	if subtle.ConstantTimeCompare([]byte(incoming), []byte(expected)) == 1 {
		return nil
	}
	return apperror.New(http.StatusUnauthorized, "invalid callback secret", nil)
}

func (s *PaymentGatewayService) applySettlementUpdate(ctx context.Context, trx *model.Transaction, status model.TransactionStatus) (bool, error) {
	if trx == nil || !isFinalTransactionStatus(status) {
		return false, nil
	}
	platformFee, netto := s.calculateSettlement(trx, status)
	if trx.Type == model.TransactionTypeDeposit && status == model.TransactionStatusSuccess {
		updated, err := s.transactionRepo.FinalizeDepositSuccessByID(ctx, trx.ID, trx.TokoID, platformFee, netto)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return false, apperror.New(http.StatusNotFound, "transaction not found", nil)
			}
			return false, apperror.New(http.StatusInternalServerError, "failed to update local transaction settlement", err.Error())
		}
		if updated {
			trx.Status = status
			trx.PlatformFee = platformFee
			trx.Netto = netto
		}
		return updated, nil
	}
	updated, err := s.transactionRepo.UpdateSettlementIfPending(ctx, trx.ID, status, platformFee, netto)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, apperror.New(http.StatusNotFound, "transaction not found", nil)
		}
		return false, apperror.New(http.StatusInternalServerError, "failed to update local transaction settlement", err.Error())
	}
	if updated {
		trx.Status = status
		trx.PlatformFee = platformFee
		trx.Netto = netto
	}
	return updated, nil
}

func (s *PaymentGatewayService) finalizeWithdrawStatus(ctx context.Context, trx *model.Transaction, status model.TransactionStatus) (bool, error) {
	if trx == nil || trx.Type != model.TransactionTypeWithdraw || !isFinalTransactionStatus(status) {
		return false, nil
	}
	updated, err := s.transactionRepo.FinalizeWithdrawIfPending(ctx, trx.ID, status)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, apperror.New(http.StatusNotFound, "transaction not found", nil)
		}
		if errors.Is(err, repository.ErrInsufficientBalance) {
			return false, apperror.New(http.StatusBadRequest, "insufficient settlement balance", "saldo settlement toko tidak mencukupi untuk finalisasi withdraw")
		}
		return false, apperror.New(http.StatusInternalServerError, "failed to finalize withdraw transaction", err.Error())
	}
	if updated {
		trx.Status = status
	}
	return updated, nil
}

func (s *PaymentGatewayService) calculateSettlement(trx *model.Transaction, status model.TransactionStatus) (uint64, uint64) {
	pendingNetto := computePendingNetto(trx.Amount, trx.FeeWithdrawal)
	switch status {
	case model.TransactionStatusSuccess:
		platformFee := s.computePlatformFee(trx.Amount)
		if pendingNetto <= platformFee {
			return platformFee, 0
		}
		return platformFee, pendingNetto - platformFee
	case model.TransactionStatusFailed, model.TransactionStatusExpired:
		return 0, 0
	default:
		return 0, pendingNetto
	}
}

func computePendingNetto(amount uint64, feeWithdrawal *uint64) uint64 {
	if feeWithdrawal == nil {
		return amount
	}
	fee := *feeWithdrawal
	if amount <= fee {
		return 0
	}
	return amount - fee
}

func (s *PaymentGatewayService) computePlatformFee(amount uint64) uint64 {
	fee, err := money.PercentageFee(amount, s.platformFeePercent)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to compute platform fee", "amount", amount, "percent", s.platformFeePercent, "error", err.Error())
		}
		return 0
	}
	return fee
}

func (s *PaymentGatewayService) ExpirePendingTransactions(ctx context.Context, olderThan time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = defaultExpiryBatchSize
	}

	candidates, err := s.transactionRepo.ListPendingExpiryCandidates(ctx, olderThan, limit)
	if err != nil {
		return 0, apperror.New(http.StatusInternalServerError, "failed to query pending transactions for expiry", err.Error())
	}

	enqueued := 0
	var firstErr error
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.TrxID) == "" {
			s.logger.Warn("skip expiring transaction without reference", "transaction_id", candidate.TransactionID)
			continue
		}

		payload := queue.QrisCallbackTaskPayload{
			Amount:     candidate.Amount,
			TerminalID: "",
			MerchantID: s.merchantUUID,
			TrxID:      candidate.TrxID,
			RRN:        candidate.RRN,
			CustomRef:  candidate.CustomRef,
			Vendor:     fallback(candidate.Vendor, "scheduler"),
			Status:     string(model.TransactionStatusExpired),
			CreatedAt:  candidate.CreatedAt.UTC().Format(time.RFC3339),
			FinishAt:   time.Now().UTC().Format(time.RFC3339),
		}

		if err := s.EnqueueQrisCallback(ctx, payload); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			s.logger.Error("failed to enqueue expired qris callback", "trx_id", candidate.TrxID, "error", err.Error())
			continue
		}
		enqueued++
	}

	return enqueued, firstErr
}

func (s *PaymentGatewayService) ensureMerchantID(merchantID string) error {
	merchantID = strings.TrimSpace(merchantID)
	if merchantID == "" {
		return apperror.New(http.StatusBadRequest, "merchant_id is required", nil)
	}
	if merchantID != s.merchantUUID {
		return apperror.New(http.StatusUnauthorized, "merchant_id mismatch", nil)
	}
	return nil
}

func (s *PaymentGatewayService) mapGatewayError(message string, err error) error {
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

func (s *PaymentGatewayService) mapRepositoryError(notFoundMsg string, err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return apperror.New(http.StatusNotFound, notFoundMsg, nil)
	}
	return apperror.New(http.StatusInternalServerError, "repository operation failed", err.Error())
}

func normalizeTransactionStatus(status string) model.TransactionStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case string(model.TransactionStatusSuccess):
		return model.TransactionStatusSuccess
	case string(model.TransactionStatusPending):
		return model.TransactionStatusPending
	case string(model.TransactionStatusFailed):
		return model.TransactionStatusFailed
	case string(model.TransactionStatusExpired):
		return model.TransactionStatusExpired
	default:
		return ""
	}
}

func isFinalTransactionStatus(status model.TransactionStatus) bool {
	switch status {
	case model.TransactionStatusSuccess, model.TransactionStatusFailed, model.TransactionStatusExpired:
		return true
	default:
		return false
	}
}

func (s *PaymentGatewayService) buildCheckStatusCacheEntry(trx *model.Transaction, resp *paymentgateway.CheckStatusResponse) qrisStatusCacheEntry {
	entry := qrisStatusCacheEntry{
		Amount:     resp.Amount,
		TerminalID: "",
		MerchantID: strings.TrimSpace(resp.MerchantID),
		TrxID:      strings.TrimSpace(resp.TrxID),
		RRN:        strings.TrimSpace(resp.RRN),
		Vendor:     "",
		Status:     strings.TrimSpace(resp.Status),
		CreatedAt:  strings.TrimSpace(resp.CreatedAt),
		FinishAt:   strings.TrimSpace(resp.FinishAt),
	}
	if trx != nil && trx.Code != nil {
		entry.CustomRef = strings.TrimSpace(*trx.Code)
	}
	return entry
}

func (s *PaymentGatewayService) buildWebhookCacheEntry(payload queue.QrisCallbackTaskPayload) qrisStatusCacheEntry {
	return qrisStatusCacheEntry{
		Amount:     payload.Amount,
		TerminalID: strings.TrimSpace(payload.TerminalID),
		MerchantID: strings.TrimSpace(payload.MerchantID),
		TrxID:      strings.TrimSpace(payload.TrxID),
		RRN:        strings.TrimSpace(payload.RRN),
		CustomRef:  strings.TrimSpace(payload.CustomRef),
		Vendor:     strings.TrimSpace(payload.Vendor),
		Status:     strings.TrimSpace(payload.Status),
		CreatedAt:  strings.TrimSpace(payload.CreatedAt),
		FinishAt:   strings.TrimSpace(payload.FinishAt),
	}
}

func (s *PaymentGatewayService) mapQrisStatusCacheEntryToResult(entry qrisStatusCacheEntry) *CheckPaymentStatusResult {
	return &CheckPaymentStatusResult{
		Amount:     entry.Amount,
		MerchantID: entry.MerchantID,
		TrxID:      entry.TrxID,
		RRN:        entry.RRN,
		Status:     entry.Status,
		CreatedAt:  entry.CreatedAt,
		FinishAt:   entry.FinishAt,
	}
}

func (s *PaymentGatewayService) getCachedQrisStatus(ctx context.Context, trxID string) (*qrisStatusCacheEntry, bool) {
	return getCachedJSON[qrisStatusCacheEntry](ctx, s.cache, s.cacheEnabled, trxID, s.logger)
}

func (s *PaymentGatewayService) cacheQrisStatus(ctx context.Context, trxID string, value qrisStatusCacheEntry) {
	setCachedJSON(ctx, s.cache, s.cacheEnabled, trxID, value, s.statusCacheTTL, s.logger)
}

func (s *PaymentGatewayService) transferInquiryCacheKey(tokoID uint64, bankCode string, accountNumber string, amount uint64, inquiryID uint64) string {
	return buildHashedCacheKey(
		"payment:transfer:inquiry",
		"toko="+strconv.FormatUint(tokoID, 10),
		"bank_code="+strings.TrimSpace(bankCode),
		"account_number="+strings.TrimSpace(accountNumber),
		"amount="+strconv.FormatUint(amount, 10),
		"inquiry_id="+strconv.FormatUint(inquiryID, 10),
	)
}

func (s *PaymentGatewayService) cacheTransferInquiry(ctx context.Context, tokoID uint64, resp *paymentgateway.InquiryTransferResponse) {
	if resp == nil {
		return
	}
	entry := cachedTransferInquiry{
		TokoID:        tokoID,
		BankCode:      strings.TrimSpace(resp.BankCode),
		BankName:      strings.TrimSpace(resp.BankName),
		AccountNumber: strings.TrimSpace(resp.AccountNumber),
		AccountName:   strings.TrimSpace(resp.AccountName),
		Amount:        resp.Amount,
		Fee:           resp.Fee,
		InquiryID:     resp.InquiryID,
		PartnerRefNo:  strings.TrimSpace(resp.PartnerRefNo),
	}
	setCachedJSON(ctx, s.cache, s.cacheEnabled, s.transferInquiryCacheKey(tokoID, entry.BankCode, entry.AccountNumber, entry.Amount, entry.InquiryID), entry, transferInquiryCacheTTL, s.logger)
}

func (s *PaymentGatewayService) getCachedTransferInquiry(ctx context.Context, tokoID uint64, bankCode string, accountNumber string, amount uint64, inquiryID uint64) (*cachedTransferInquiry, bool) {
	return getCachedJSON[cachedTransferInquiry](ctx, s.cache, s.cacheEnabled, s.transferInquiryCacheKey(tokoID, bankCode, accountNumber, amount, inquiryID), s.logger)
}

func (s *PaymentGatewayService) clearTransferInquiryCache(ctx context.Context, tokoID uint64, bankCode string, accountNumber string, amount uint64, inquiryID uint64) {
	if !s.cacheEnabled || s.cache == nil {
		return
	}
	key := s.transferInquiryCacheKey(tokoID, bankCode, accountNumber, amount, inquiryID)
	if err := s.cache.Delete(ctx, key); err != nil && s.logger != nil {
		s.logger.Error("cache delete failed", "key", key, "error", err.Error())
	}
}

func (s *PaymentGatewayService) qrisCallbackDeliveryKey(trxID string, status model.TransactionStatus) string {
	return "qris:callback:delivered:" + strings.TrimSpace(trxID) + ":" + string(status)
}

func (s *PaymentGatewayService) isMerchantQrisCallbackDelivered(ctx context.Context, trxID string, status model.TransactionStatus) bool {
	value, ok := getCachedJSON[bool](ctx, s.cache, s.cacheEnabled, s.qrisCallbackDeliveryKey(trxID, status), s.logger)
	return ok && value != nil && *value
}

func (s *PaymentGatewayService) markMerchantQrisCallbackDelivered(ctx context.Context, trxID string, status model.TransactionStatus) {
	setCachedJSON(ctx, s.cache, s.cacheEnabled, s.qrisCallbackDeliveryKey(trxID, status), true, s.callbackTTL, s.logger)
}

func (s *PaymentGatewayService) sendMerchantQrisCallback(ctx context.Context, toko *model.Toko, entry qrisStatusCacheEntry) error {
	if toko == nil {
		return nil
	}
	status := normalizeTransactionStatus(entry.Status)
	if !isFinalTransactionStatus(status) {
		return nil
	}
	if s.isMerchantQrisCallbackDelivered(ctx, entry.TrxID, status) {
		s.logger.Info("skip duplicate merchant qris callback", "trx_id", entry.TrxID, "status", status)
		return nil
	}

	callbackURL := ""
	if toko.CallbackURL != nil {
		callbackURL = strings.TrimSpace(*toko.CallbackURL)
	}
	if callbackURL == "" {
		s.logger.Warn("merchant callback url not configured", "toko_id", toko.ID, "trx_id", entry.TrxID)
		return nil
	}

	payloadBytes, err := json.Marshal(merchantQrisCallbackPayload{
		Amount:     entry.Amount,
		TerminalID: entry.TerminalID,
		MerchantID: toko.Token,
		TrxID:      entry.TrxID,
		RRN:        entry.RRN,
		CustomRef:  entry.CustomRef,
		Vendor:     entry.Vendor,
		Status:     entry.Status,
		CreatedAt:  entry.CreatedAt,
		FinishAt:   entry.FinishAt,
	})
	if err != nil {
		return apperror.New(http.StatusInternalServerError, "failed to encode merchant callback payload", err.Error())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return apperror.New(http.StatusBadGateway, "failed to build merchant callback request", err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "gue-payment-callback/1.0")

	resp, err := s.callbackClient.Do(req)
	if err != nil {
		s.logger.Error("merchant qris callback failed", "toko_id", toko.ID, "trx_id", entry.TrxID, "callback_url", callbackURL, "error", err.Error())
		return apperror.New(http.StatusBadGateway, "failed to deliver merchant callback", err.Error())
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		excerpt := sanitizeMerchantCallbackExcerpt(bodyBytes)
		s.logger.Error(
			"merchant qris callback rejected",
			"toko_id", toko.ID,
			"trx_id", entry.TrxID,
			"status_code", resp.StatusCode,
			"callback_url", callbackURL,
			"response_excerpt", excerpt,
		)
		return apperror.New(http.StatusBadGateway, "merchant callback returned non-2xx status", excerpt)
	}

	s.markMerchantQrisCallbackDelivered(ctx, entry.TrxID, status)
	s.logger.Info("merchant qris callback delivered", "toko_id", toko.ID, "trx_id", entry.TrxID, "status", status, "callback_url", callbackURL)
	return nil
}

func sanitizeMerchantCallbackExcerpt(raw []byte) string {
	compacted := strings.Join(strings.Fields(strings.TrimSpace(string(raw))), " ")
	if len(compacted) <= 180 {
		return compacted
	}
	return compacted[:177] + "..."
}

func fallback(value, defaultValue string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return strings.TrimSpace(defaultValue)
}

var _ PaymentGatewayUseCase = (*PaymentGatewayService)(nil)
var _ queue.CallbackProcessor = (*PaymentGatewayService)(nil)
