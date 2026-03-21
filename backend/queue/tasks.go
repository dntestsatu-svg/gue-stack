package queue

import "encoding/json"

const (
	TypeSendWelcomeEmail        = "email:send_welcome"
	TypeProcessQrisCallback     = "payment:callback:qris"
	TypeProcessTransferCallback = "payment:callback:transfer"
)

type WelcomeEmailPayload struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type QrisCallbackTaskPayload struct {
	Amount     uint64 `json:"amount" validate:"required,gt=0"`
	TerminalID string `json:"terminal_id" validate:"required,max=255"`
	MerchantID string `json:"merchant_id" validate:"required,max=255"`
	TrxID      string `json:"trx_id" validate:"required,max=255"`
	RRN        string `json:"rrn" validate:"required,max=255"`
	CustomRef  string `json:"custom_ref,omitempty" validate:"omitempty,max=36"`
	Vendor     string `json:"vendor" validate:"required,max=255"`
	Status     string `json:"status" validate:"required,oneof=success failed pending"`
	CreatedAt  string `json:"created_at" validate:"required"`
	FinishAt   string `json:"finish_at" validate:"required"`
}

type TransferCallbackTaskPayload struct {
	Amount          uint64 `json:"amount" validate:"required,gt=0"`
	PartnerRefNo    string `json:"partner_ref_no" validate:"required,max=255"`
	Status          string `json:"status" validate:"required,oneof=success failed pending"`
	TransactionDate string `json:"transaction_date" validate:"required"`
	MerchantID      string `json:"merchant_id" validate:"required,max=255"`
}

func NewWelcomeEmailPayload(email, name string) ([]byte, error) {
	return json.Marshal(WelcomeEmailPayload{
		Email: email,
		Name:  name,
	})
}

func NewQrisCallbackTaskPayload(payload QrisCallbackTaskPayload) ([]byte, error) {
	return json.Marshal(payload)
}

func NewTransferCallbackTaskPayload(payload TransferCallbackTaskPayload) ([]byte, error) {
	return json.Marshal(payload)
}
