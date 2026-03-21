package service

import (
	"context"
	"testing"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/paymentgateway"
	"github.com/example/gue/backend/repository"
	"github.com/stretchr/testify/require"
	"log/slog"
)

type fakePaymentGatewayClient struct {
	generateResp *paymentgateway.GenerateResponse
	generateErr  error
}

func (f *fakePaymentGatewayClient) Generate(_ context.Context, _ paymentgateway.GenerateRequest) (*paymentgateway.GenerateResponse, error) {
	return f.generateResp, f.generateErr
}

func (f *fakePaymentGatewayClient) CheckStatusV2(_ context.Context, _ string, _ paymentgateway.CheckStatusRequest) (*paymentgateway.CheckStatusResponse, error) {
	return nil, nil
}

func (f *fakePaymentGatewayClient) InquiryTransfer(_ context.Context, _ paymentgateway.InquiryTransferRequest) (*paymentgateway.InquiryTransferResponse, error) {
	return nil, nil
}

func (f *fakePaymentGatewayClient) TransferFund(_ context.Context, _ paymentgateway.TransferFundRequest) (*paymentgateway.TransferFundResponse, error) {
	return nil, nil
}

func (f *fakePaymentGatewayClient) CheckTransferStatus(_ context.Context, _ string, _ paymentgateway.CheckTransferStatusRequest) (*paymentgateway.CheckTransferStatusResponse, error) {
	return nil, nil
}

func (f *fakePaymentGatewayClient) GetBalance(_ context.Context, _ string, _ paymentgateway.GetBalanceRequest) (*paymentgateway.GetBalanceResponse, error) {
	return nil, nil
}

type fakeTokoRepo struct {
	byToken map[string]*model.Toko
}

func (f *fakeTokoRepo) GetByID(_ context.Context, _ uint64) (*model.Toko, error) {
	return nil, repository.ErrNotFound
}

func (f *fakeTokoRepo) GetByToken(_ context.Context, token string) (*model.Toko, error) {
	t, ok := f.byToken[token]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return t, nil
}

type fakeTransactionRepo struct {
	created []*model.Transaction
}

func (f *fakeTransactionRepo) Create(_ context.Context, trx *model.Transaction) error {
	f.created = append(f.created, trx)
	return nil
}

func (f *fakeTransactionRepo) GetByReference(_ context.Context, _ string) (*model.Transaction, error) {
	return nil, repository.ErrNotFound
}

func (f *fakeTransactionRepo) UpdateStatusByReference(_ context.Context, _ string, _ model.TransactionStatus) error {
	return nil
}

func (f *fakeTransactionRepo) UpdateStatusByReferenceAndToko(_ context.Context, _ string, _ uint64, _ model.TransactionStatus) error {
	return nil
}

func TestPaymentGatewayServiceGenerateCreatesPendingTransaction(t *testing.T) {
	tokoRepo := &fakeTokoRepo{byToken: map[string]*model.Toko{
		"uuid-123": {ID: 55, Token: "uuid-123", Name: "Toko A"},
	}}
	trxRepo := &fakeTransactionRepo{}
	gateway := &fakePaymentGatewayClient{generateResp: &paymentgateway.GenerateResponse{Data: "qr-data", TrxID: "trx-123"}}

	svc := NewPaymentGatewayService(gateway, tokoRepo, trxRepo, "", "", "", slog.Default())
	result, err := svc.Generate(context.Background(), GeneratePaymentInput{
		Username: "player-1",
		Amount:   10000,
		UUID:     "uuid-123",
	})

	require.NoError(t, err)
	require.Equal(t, "trx-123", result.TrxID)
	require.Len(t, trxRepo.created, 1)
	require.Equal(t, model.TransactionStatusPending, trxRepo.created[0].Status)
	require.Equal(t, model.TransactionTypeDeposit, trxRepo.created[0].Type)
	require.Equal(t, uint64(55), trxRepo.created[0].TokoID)
}

func TestPaymentGatewayServiceValidateCallbackSecret(t *testing.T) {
	svc := NewPaymentGatewayService(&fakePaymentGatewayClient{}, &fakeTokoRepo{}, &fakeTransactionRepo{}, "", "", "secret", slog.Default())

	err := svc.ValidateCallbackSecret("wrong")
	require.Error(t, err)

	err = svc.ValidateCallbackSecret("secret")
	require.NoError(t, err)
}
