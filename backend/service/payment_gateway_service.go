package service

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/paymentgateway"
	"github.com/example/gue/backend/queue"
	"github.com/example/gue/backend/repository"
	"github.com/go-playground/validator/v10"
)

const percentBase = 100

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
) *PaymentGatewayService {
	if platformFeePercent < 0 {
		platformFeePercent = 0
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

	client := fallback(input.Client, s.defaultClient)
	resp, err := s.gatewayClient.CheckStatusV2(ctx, trxID, paymentgateway.CheckStatusRequest{
		UUID:   s.merchantUUID,
		Client: client,
	})
	if err != nil {
		return nil, s.mapGatewayError("failed to check payment status", err)
	}

	status := normalizeTransactionStatus(resp.Status)
	if status != "" {
		if err := s.applySettlementUpdate(ctx, localTrx, status); err != nil {
			return nil, err
		}
	}

	return &CheckPaymentStatusResult{
		Amount:     resp.Amount,
		MerchantID: resp.MerchantID,
		TrxID:      resp.TrxID,
		RRN:        resp.RRN,
		Status:     resp.Status,
		CreatedAt:  resp.CreatedAt,
		FinishAt:   resp.FinishAt,
	}, nil
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

	reference := resp.PartnerRefNo
	_, getErr := s.transactionRepo.GetByReferenceAndToko(ctx, reference, tokoID)
	if getErr != nil {
		if !errors.Is(getErr, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusInternalServerError, "failed to query existing inquiry transaction", getErr.Error())
		}
		fee := resp.Fee
		trx := &model.Transaction{
			TokoID:        tokoID,
			Type:          model.TransactionTypeWithdraw,
			Status:        model.TransactionStatusPending,
			Reference:     &reference,
			Amount:        resp.Amount,
			FeeWithdrawal: &fee,
			PlatformFee:   0,
			Netto:         computePendingNetto(resp.Amount, &fee),
		}
		if err := s.transactionRepo.Create(ctx, trx); err != nil {
			return nil, apperror.New(http.StatusInternalServerError, "failed to persist inquiry transaction", err.Error())
		}
	}

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

	resp, err := s.gatewayClient.TransferFund(ctx, paymentgateway.TransferFundRequest{
		Client:        client,
		ClientKey:     s.defaultKey,
		UUID:          s.merchantUUID,
		Amount:        input.Amount,
		BankCode:      input.BankCode,
		AccountNumber: input.AccountNumber,
		Type:          input.Type,
		InquiryID:     input.InquiryID,
	})
	if err != nil {
		return nil, s.mapGatewayError("failed to transfer fund", err)
	}

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
	if status != "" {
		if err := s.applySettlementUpdate(ctx, localTrx, status); err != nil {
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

	status := normalizeTransactionStatus(payload.Status)
	if status == "" {
		return apperror.New(http.StatusBadRequest, "invalid callback status", payload.Status)
	}

	trx, err := s.transactionRepo.GetByReference(ctx, payload.TrxID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.Warn("qris callback transaction not found", "trx_id", payload.TrxID)
			return nil
		}
		return apperror.New(http.StatusInternalServerError, "failed to load callback transaction", err.Error())
	}

	return s.applySettlementUpdate(ctx, trx, status)
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

	return s.applySettlementUpdate(ctx, trx, status)
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

func (s *PaymentGatewayService) applySettlementUpdate(ctx context.Context, trx *model.Transaction, status model.TransactionStatus) error {
	platformFee, netto := s.calculateSettlement(trx, status)
	if err := s.transactionRepo.UpdateSettlementByID(ctx, trx.ID, status, platformFee, netto); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(http.StatusNotFound, "transaction not found", nil)
		}
		return apperror.New(http.StatusInternalServerError, "failed to update local transaction settlement", err.Error())
	}
	return nil
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
	case model.TransactionStatusFailed:
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
	if s.platformFeePercent == 0 {
		return 0
	}
	return (amount * s.platformFeePercent) / percentBase
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
	default:
		return ""
	}
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
