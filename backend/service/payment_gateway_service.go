package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/paymentgateway"
	"github.com/example/gue/backend/repository"
	"github.com/go-playground/validator/v10"
)

type PaymentGatewayUseCase interface {
	Generate(ctx context.Context, input GeneratePaymentInput) (*GeneratePaymentResult, error)
	CheckStatusV2(ctx context.Context, trxID string, input CheckPaymentStatusInput) (*CheckPaymentStatusResult, error)
	InquiryTransfer(ctx context.Context, input InquiryTransferInput) (*InquiryTransferResult, error)
	TransferFund(ctx context.Context, input TransferFundInput) (*TransferFundResult, error)
	CheckTransferStatus(ctx context.Context, partnerRefNo string, input CheckTransferStatusInput) (*CheckTransferStatusResult, error)
	GetBalance(ctx context.Context, merchantUUID string, input GetBalanceInput) (*GetBalanceResult, error)
	HandleQrisCallback(ctx context.Context, payload QrisCallbackPayload) error
	HandleTransferCallback(ctx context.Context, payload TransferCallbackPayload) error
	ValidateCallbackSecret(secret string) error
}

type PaymentGatewayService struct {
	gatewayClient   paymentgateway.Client
	tokoRepo        repository.TokoRepository
	transactionRepo repository.TransactionRepository
	validate        *validator.Validate
	defaultClient   string
	defaultKey      string
	callbackSecret  string
	logger          *slog.Logger
}

type GeneratePaymentInput struct {
	Username  string `json:"username" validate:"required,max=255"`
	Amount    uint64 `json:"amount" validate:"required,gte=10000,lte=10000000"`
	UUID      string `json:"uuid" validate:"required,max=255"`
	Expire    *int   `json:"expire,omitempty" validate:"omitempty,gte=30,lte=86400"`
	CustomRef string `json:"custom_ref,omitempty" validate:"omitempty,max=36,alphanum"`
}

type GeneratePaymentResult struct {
	Data      string `json:"data"`
	TrxID     string `json:"trx_id"`
	ExpiredAt *int64 `json:"expired_at,omitempty"`
}

type CheckPaymentStatusInput struct {
	UUID   string `json:"uuid" validate:"required,max=255"`
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
	ClientKey     string  `json:"client_key" validate:"omitempty,max=255"`
	UUID          string  `json:"uuid" validate:"required,max=255"`
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
	ClientKey     string `json:"client_key" validate:"omitempty,max=255"`
	UUID          string `json:"uuid" validate:"required,max=255"`
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
	UUID   string `json:"uuid" validate:"required,max=255"`
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

type QrisCallbackPayload struct {
	Amount     uint64 `json:"amount" validate:"required,gt=0"`
	TerminalID string `json:"terminal_id" validate:"required,max=255"`
	MerchantID string `json:"merchant_id" validate:"required,max=255"`
	TrxID      string `json:"trx_id" validate:"required,max=255"`
	RRN        string `json:"rrn" validate:"required,max=255"`
	CustomRef  string `json:"custom_ref" validate:"omitempty,max=36"`
	Vendor     string `json:"vendor" validate:"required,max=255"`
	Status     string `json:"status" validate:"required,oneof=success failed pending"`
	CreatedAt  string `json:"created_at" validate:"required"`
	FinishAt   string `json:"finish_at" validate:"required"`
}

type TransferCallbackPayload struct {
	Amount          uint64 `json:"amount" validate:"required,gt=0"`
	PartnerRefNo    string `json:"partner_ref_no" validate:"required,max=255"`
	Status          string `json:"status" validate:"required,oneof=success failed pending"`
	TransactionDate string `json:"transaction_date" validate:"required"`
	MerchantID      string `json:"merchant_id" validate:"required,max=255"`
}

func NewPaymentGatewayService(
	gatewayClient paymentgateway.Client,
	tokoRepo repository.TokoRepository,
	transactionRepo repository.TransactionRepository,
	defaultClient string,
	defaultKey string,
	callbackSecret string,
	logger *slog.Logger,
) *PaymentGatewayService {
	return &PaymentGatewayService{
		gatewayClient:   gatewayClient,
		tokoRepo:        tokoRepo,
		transactionRepo: transactionRepo,
		validate:        validator.New(validator.WithRequiredStructEnabled()),
		defaultClient:   defaultClient,
		defaultKey:      defaultKey,
		callbackSecret:  callbackSecret,
		logger:          logger,
	}
}

func (s *PaymentGatewayService) Generate(ctx context.Context, input GeneratePaymentInput) (*GeneratePaymentResult, error) {
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}

	toko, err := s.tokoRepo.GetByToken(ctx, input.UUID)
	if err != nil {
		return nil, s.mapRepositoryError("toko not found", err)
	}

	resp, err := s.gatewayClient.Generate(ctx, paymentgateway.GenerateRequest{
		Username:  input.Username,
		Amount:    input.Amount,
		UUID:      input.UUID,
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
		TokoID:    toko.ID,
		Player:    &player,
		Type:      model.TransactionTypeDeposit,
		Status:    model.TransactionStatusPending,
		Reference: &reference,
		Amount:    input.Amount,
		Netto:     input.Amount,
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

func (s *PaymentGatewayService) CheckStatusV2(ctx context.Context, trxID string, input CheckPaymentStatusInput) (*CheckPaymentStatusResult, error) {
	trxID = strings.TrimSpace(trxID)
	if trxID == "" {
		return nil, apperror.New(http.StatusBadRequest, "trx_id is required", nil)
	}
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}

	toko, err := s.tokoRepo.GetByToken(ctx, input.UUID)
	if err != nil {
		return nil, s.mapRepositoryError("toko not found", err)
	}

	client := fallback(input.Client, s.defaultClient)
	resp, err := s.gatewayClient.CheckStatusV2(ctx, trxID, paymentgateway.CheckStatusRequest{
		UUID:   input.UUID,
		Client: client,
	})
	if err != nil {
		return nil, s.mapGatewayError("failed to check payment status", err)
	}

	status := normalizeTransactionStatus(resp.Status)
	if status != "" {
		if err := s.transactionRepo.UpdateStatusByReferenceAndToko(ctx, trxID, toko.ID, status); err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusInternalServerError, "failed to update local transaction status", err.Error())
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

func (s *PaymentGatewayService) InquiryTransfer(ctx context.Context, input InquiryTransferInput) (*InquiryTransferResult, error) {
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}

	toko, err := s.tokoRepo.GetByToken(ctx, input.UUID)
	if err != nil {
		return nil, s.mapRepositoryError("toko not found", err)
	}

	client := fallback(input.Client, s.defaultClient)
	clientKey := fallback(input.ClientKey, s.defaultKey)
	if client == "" || clientKey == "" {
		return nil, apperror.New(http.StatusBadRequest, "client and client_key are required", nil)
	}

	resp, err := s.gatewayClient.InquiryTransfer(ctx, paymentgateway.InquiryTransferRequest{
		Client:        client,
		ClientKey:     clientKey,
		UUID:          input.UUID,
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

	fee := resp.Fee
	reference := resp.PartnerRefNo
	netto := resp.Amount
	if resp.Amount > resp.Fee {
		netto = resp.Amount - resp.Fee
	} else {
		netto = 0
	}

	_, err = s.transactionRepo.GetByReference(ctx, reference)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusInternalServerError, "failed to query existing inquiry transaction", err.Error())
		}
		trx := &model.Transaction{
			TokoID:        toko.ID,
			Type:          model.TransactionTypeWithdraw,
			Status:        model.TransactionStatusPending,
			Reference:     &reference,
			Amount:        resp.Amount,
			FeeWithdrawal: &fee,
			Netto:         netto,
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

func (s *PaymentGatewayService) TransferFund(ctx context.Context, input TransferFundInput) (*TransferFundResult, error) {
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}

	if _, err := s.tokoRepo.GetByToken(ctx, input.UUID); err != nil {
		return nil, s.mapRepositoryError("toko not found", err)
	}

	client := fallback(input.Client, s.defaultClient)
	clientKey := fallback(input.ClientKey, s.defaultKey)
	if client == "" || clientKey == "" {
		return nil, apperror.New(http.StatusBadRequest, "client and client_key are required", nil)
	}

	resp, err := s.gatewayClient.TransferFund(ctx, paymentgateway.TransferFundRequest{
		Client:        client,
		ClientKey:     clientKey,
		UUID:          input.UUID,
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

func (s *PaymentGatewayService) CheckTransferStatus(ctx context.Context, partnerRefNo string, input CheckTransferStatusInput) (*CheckTransferStatusResult, error) {
	partnerRefNo = strings.TrimSpace(partnerRefNo)
	if partnerRefNo == "" {
		return nil, apperror.New(http.StatusBadRequest, "partner_ref_no is required", nil)
	}
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}

	toko, err := s.tokoRepo.GetByToken(ctx, input.UUID)
	if err != nil {
		return nil, s.mapRepositoryError("toko not found", err)
	}

	client := fallback(input.Client, s.defaultClient)
	resp, err := s.gatewayClient.CheckTransferStatus(ctx, partnerRefNo, paymentgateway.CheckTransferStatusRequest{
		Client: client,
		UUID:   input.UUID,
	})
	if err != nil {
		return nil, s.mapGatewayError("failed to check transfer status", err)
	}

	status := normalizeTransactionStatus(resp.Status)
	if status != "" {
		if err := s.transactionRepo.UpdateStatusByReferenceAndToko(ctx, partnerRefNo, toko.ID, status); err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(http.StatusInternalServerError, "failed to update local transfer status", err.Error())
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

func (s *PaymentGatewayService) GetBalance(ctx context.Context, merchantUUID string, input GetBalanceInput) (*GetBalanceResult, error) {
	merchantUUID = strings.TrimSpace(merchantUUID)
	if merchantUUID == "" {
		return nil, apperror.New(http.StatusBadRequest, "merchant UUID is required", nil)
	}
	if err := s.validate.Struct(input); err != nil {
		return nil, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}

	client := fallback(input.Client, s.defaultClient)
	if client == "" {
		return nil, apperror.New(http.StatusBadRequest, "client is required", nil)
	}

	resp, err := s.gatewayClient.GetBalance(ctx, merchantUUID, paymentgateway.GetBalanceRequest{Client: client})
	if err != nil {
		return nil, s.mapGatewayError("failed to fetch merchant balance", err)
	}

	return &GetBalanceResult{
		Status:         resp.Status,
		PendingBalance: resp.PendingBalance,
		SettleBalance:  resp.SettleBalance,
	}, nil
}

func (s *PaymentGatewayService) HandleQrisCallback(ctx context.Context, payload QrisCallbackPayload) error {
	if err := s.validate.Struct(payload); err != nil {
		return apperror.New(http.StatusBadRequest, "invalid callback payload", err.Error())
	}

	toko, err := s.tokoRepo.GetByToken(ctx, payload.MerchantID)
	if err != nil {
		return s.mapRepositoryError("toko not found", err)
	}

	status := normalizeTransactionStatus(payload.Status)
	if status == "" {
		return apperror.New(http.StatusBadRequest, "invalid callback status", payload.Status)
	}

	if err := s.transactionRepo.UpdateStatusByReferenceAndToko(ctx, payload.TrxID, toko.ID, status); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.Warn("qris callback transaction not found", "trx_id", payload.TrxID, "toko_id", toko.ID)
			return nil
		}
		return apperror.New(http.StatusInternalServerError, "failed to update callback transaction", err.Error())
	}

	return nil
}

func (s *PaymentGatewayService) HandleTransferCallback(ctx context.Context, payload TransferCallbackPayload) error {
	if err := s.validate.Struct(payload); err != nil {
		return apperror.New(http.StatusBadRequest, "invalid callback payload", err.Error())
	}

	toko, err := s.tokoRepo.GetByToken(ctx, payload.MerchantID)
	if err != nil {
		return s.mapRepositoryError("toko not found", err)
	}

	status := normalizeTransactionStatus(payload.Status)
	if status == "" {
		return apperror.New(http.StatusBadRequest, "invalid callback status", payload.Status)
	}

	if err := s.transactionRepo.UpdateStatusByReferenceAndToko(ctx, payload.PartnerRefNo, toko.ID, status); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.Warn("transfer callback transaction not found", "partner_ref_no", payload.PartnerRefNo, "toko_id", toko.ID)
			return nil
		}
		return apperror.New(http.StatusInternalServerError, "failed to update callback transfer", err.Error())
	}

	return nil
}

func (s *PaymentGatewayService) ValidateCallbackSecret(secret string) error {
	if strings.TrimSpace(s.callbackSecret) == "" {
		return nil
	}
	if strings.TrimSpace(secret) == s.callbackSecret {
		return nil
	}
	return apperror.New(http.StatusUnauthorized, "invalid callback secret", nil)
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
