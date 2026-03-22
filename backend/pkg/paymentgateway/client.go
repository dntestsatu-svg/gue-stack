package paymentgateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client interface {
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
	CheckStatusV2(ctx context.Context, trxID string, req CheckStatusRequest) (*CheckStatusResponse, error)
	InquiryTransfer(ctx context.Context, req InquiryTransferRequest) (*InquiryTransferResponse, error)
	TransferFund(ctx context.Context, req TransferFundRequest) (*TransferFundResponse, error)
	CheckTransferStatus(ctx context.Context, partnerRefNo string, req CheckTransferStatusRequest) (*CheckTransferStatusResponse, error)
	GetBalance(ctx context.Context, merchantUUID string, req GetBalanceRequest) (*GetBalanceResponse, error)
}

type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

type APIError struct {
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
	Body       string `json:"body"`
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return "payment gateway error"
	}
	return e.Message
}

func NewClient(baseURL string, timeout time.Duration) *HTTPClient {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type GenerateRequest struct {
	Username  string `json:"username"`
	Amount    uint64 `json:"amount"`
	UUID      string `json:"uuid"`
	Expire    *int   `json:"expire,omitempty"`
	CustomRef string `json:"custom_ref,omitempty"`
}

type GenerateResponse struct {
	Data      string `json:"data"`
	TrxID     string `json:"trx_id"`
	ExpiredAt *int64 `json:"expired_at,omitempty"`
}

type CheckStatusRequest struct {
	UUID   string `json:"uuid"`
	Client string `json:"client"`
}

type CheckStatusResponse struct {
	Amount     uint64 `json:"amount"`
	MerchantID string `json:"merchant_id"`
	TrxID      string `json:"trx_id"`
	RRN        string `json:"rrn,omitempty"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at,omitempty"`
	FinishAt   string `json:"finish_at,omitempty"`
}

type InquiryTransferRequest struct {
	Client        string  `json:"client"`
	ClientKey     string  `json:"client_key"`
	UUID          string  `json:"uuid"`
	Amount        uint64  `json:"amount"`
	BankCode      string  `json:"bank_code"`
	AccountNumber string  `json:"account_number"`
	Type          int     `json:"type"`
	Note          *string `json:"note,omitempty"`
	ClientRefID   *string `json:"client_ref_id,omitempty"`
}

type InquiryTransferResponse struct {
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

type TransferFundRequest struct {
	Client        string `json:"client"`
	ClientKey     string `json:"client_key"`
	UUID          string `json:"uuid"`
	Amount        uint64 `json:"amount"`
	BankCode      string `json:"bank_code"`
	AccountNumber string `json:"account_number"`
	Type          int    `json:"type"`
	InquiryID     uint64 `json:"inquiry_id"`
}

type TransferFundResponse struct {
	Status bool `json:"status"`
}

type CheckTransferStatusRequest struct {
	Client string `json:"client"`
	UUID   string `json:"uuid"`
}

type CheckTransferStatusResponse struct {
	Amount       uint64 `json:"amount"`
	Fee          uint64 `json:"fee"`
	PartnerRefNo string `json:"partner_ref_no"`
	MerchantUUID string `json:"merchant_uuid"`
	Status       string `json:"status"`
}

type GetBalanceRequest struct {
	Client string `json:"client"`
}

type GetBalanceResponse struct {
	Status         string `json:"status"`
	PendingBalance uint64 `json:"pending_balance"`
	SettleBalance  uint64 `json:"settle_balance"`
}

func (c *HTTPClient) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	body, err := c.post(ctx, "/api/generate", req)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Status    bool   `json:"status"`
		Data      string `json:"data"`
		TrxID     string `json:"trx_id"`
		ExpiredAt *int64 `json:"expired_at,omitempty"`
		Error     string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode generate response: %w", err)
	}
	if !resp.Status {
		return nil, &APIError{Message: upstreamMessage(resp.Error, "generate failed"), StatusCode: http.StatusBadGateway, Body: string(body)}
	}
	return &GenerateResponse{
		Data:      resp.Data,
		TrxID:     resp.TrxID,
		ExpiredAt: normalizeGenerateExpiredAt(resp.ExpiredAt, req.Expire, time.Now().UTC()),
	}, nil
}

func (c *HTTPClient) CheckStatusV2(ctx context.Context, trxID string, req CheckStatusRequest) (*CheckStatusResponse, error) {
	body, err := c.post(ctx, "/api/checkstatus/v2/"+trxID, req)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Status     any    `json:"status"`
		Amount     uint64 `json:"amount"`
		MerchantID string `json:"merchant_id"`
		TrxID      string `json:"trx_id"`
		RRN        string `json:"rrn"`
		CreatedAt  string `json:"created_at"`
		FinishAt   string `json:"finish_at"`
		Error      string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode check status response: %w", err)
	}

	status, err := normalizeStatus(resp.Status)
	if err != nil {
		return nil, fmt.Errorf("normalize check status response: %w", err)
	}
	if status == "false" {
		return nil, &APIError{Message: upstreamMessage(resp.Error, "check status failed"), StatusCode: http.StatusBadGateway, Body: string(body)}
	}

	return &CheckStatusResponse{
		Amount:     resp.Amount,
		MerchantID: resp.MerchantID,
		TrxID:      resp.TrxID,
		RRN:        resp.RRN,
		Status:     status,
		CreatedAt:  resp.CreatedAt,
		FinishAt:   resp.FinishAt,
	}, nil
}

func (c *HTTPClient) InquiryTransfer(ctx context.Context, req InquiryTransferRequest) (*InquiryTransferResponse, error) {
	body, err := c.post(ctx, "/api/inquiry", req)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Status bool `json:"status"`
		Data   struct {
			AccountNumber string `json:"account_number"`
			AccountName   string `json:"account_name"`
			BankCode      string `json:"bank_code"`
			BankName      string `json:"bank_name"`
			PartnerRefNo  string `json:"partner_ref_no"`
			VendorRefNo   string `json:"vendor_ref_no"`
			Amount        uint64 `json:"amount"`
			Fee           uint64 `json:"fee"`
			InquiryID     uint64 `json:"inquiry_id"`
		} `json:"data"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode inquiry response: %w", err)
	}
	if !resp.Status {
		return nil, &APIError{Message: upstreamMessage(resp.Error, "inquiry failed"), StatusCode: http.StatusBadGateway, Body: string(body)}
	}

	return &InquiryTransferResponse{
		AccountNumber: resp.Data.AccountNumber,
		AccountName:   resp.Data.AccountName,
		BankCode:      resp.Data.BankCode,
		BankName:      resp.Data.BankName,
		PartnerRefNo:  resp.Data.PartnerRefNo,
		VendorRefNo:   resp.Data.VendorRefNo,
		Amount:        resp.Data.Amount,
		Fee:           resp.Data.Fee,
		InquiryID:     resp.Data.InquiryID,
	}, nil
}

func (c *HTTPClient) TransferFund(ctx context.Context, req TransferFundRequest) (*TransferFundResponse, error) {
	body, err := c.post(ctx, "/api/transfer", req)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Status bool   `json:"status"`
		Error  string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode transfer response: %w", err)
	}
	if !resp.Status {
		return nil, &APIError{Message: upstreamMessage(resp.Error, "transfer failed"), StatusCode: http.StatusBadGateway, Body: string(body)}
	}
	return &TransferFundResponse{Status: resp.Status}, nil
}

func (c *HTTPClient) CheckTransferStatus(ctx context.Context, partnerRefNo string, req CheckTransferStatusRequest) (*CheckTransferStatusResponse, error) {
	body, err := c.post(ctx, "/api/disbursement/check-status/"+partnerRefNo, req)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Status       any    `json:"status"`
		Amount       uint64 `json:"amount"`
		Fee          uint64 `json:"fee"`
		PartnerRefNo string `json:"partner_ref_no"`
		MerchantUUID string `json:"merchant_uuid"`
		Error        string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode check transfer status response: %w", err)
	}

	status, err := normalizeStatus(resp.Status)
	if err != nil {
		return nil, fmt.Errorf("normalize transfer status response: %w", err)
	}
	if status == "false" {
		return nil, &APIError{Message: upstreamMessage(resp.Error, "check transfer status failed"), StatusCode: http.StatusBadGateway, Body: string(body)}
	}

	return &CheckTransferStatusResponse{
		Amount:       resp.Amount,
		Fee:          resp.Fee,
		PartnerRefNo: resp.PartnerRefNo,
		MerchantUUID: resp.MerchantUUID,
		Status:       status,
	}, nil
}

func (c *HTTPClient) GetBalance(ctx context.Context, merchantUUID string, req GetBalanceRequest) (*GetBalanceResponse, error) {
	body, err := c.post(ctx, "/api/balance/"+merchantUUID, req)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Status         any    `json:"status"`
		PendingBalance uint64 `json:"pending_balance"`
		SettleBalance  uint64 `json:"settle_balance"`
		Error          string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode balance response: %w", err)
	}

	status, err := normalizeStatus(resp.Status)
	if err != nil {
		return nil, fmt.Errorf("normalize balance status response: %w", err)
	}
	if status == "false" {
		return nil, &APIError{Message: upstreamMessage(resp.Error, "get balance failed"), StatusCode: http.StatusBadGateway, Body: string(body)}
	}

	return &GetBalanceResponse{
		Status:         status,
		PendingBalance: resp.PendingBalance,
		SettleBalance:  resp.SettleBalance,
	}, nil
}

func (c *HTTPClient) post(ctx context.Context, path string, req any) ([]byte, error) {
	url := c.baseURL + path

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{Message: "payment gateway returned non-2xx status", StatusCode: resp.StatusCode, Body: string(body)}
	}

	return body, nil
}

func normalizeStatus(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return strings.ToLower(v), nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case nil:
		return "", fmt.Errorf("status is missing")
	default:
		return "", fmt.Errorf("unsupported status type %T", value)
	}
}

func upstreamMessage(msg, fallback string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return fallback
	}
	return msg
}

func normalizeGenerateExpiredAt(raw *int64, requestedExpire *int, now time.Time) *int64 {
	if raw == nil {
		if requestedExpire == nil || *requestedExpire <= 0 {
			return nil
		}
		expiresAt := now.Add(time.Duration(*requestedExpire) * time.Second).Unix()
		return &expiresAt
	}

	value := *raw
	switch {
	case value <= 0:
		if requestedExpire == nil || *requestedExpire <= 0 {
			return nil
		}
		expiresAt := now.Add(time.Duration(*requestedExpire) * time.Second).Unix()
		return &expiresAt
	case value >= 1_000_000_000_000:
		expiresAt := value / 1000
		return &expiresAt
	case value < 946684800:
		expiresAt := now.Add(time.Duration(value) * time.Second).Unix()
		return &expiresAt
	default:
		return raw
	}
}
