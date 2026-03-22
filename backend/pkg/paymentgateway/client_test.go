package paymentgateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTPClientGetBalanceSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/balance/merchant-123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":          true,
			"pending_balance": uint64(5000),
			"settle_balance":  uint64(12000),
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, 0)

	resp, err := client.GetBalance(context.Background(), "merchant-123", GetBalanceRequest{Client: "dewifork"})
	require.NoError(t, err)
	require.Equal(t, "true", resp.Status)
	require.Equal(t, uint64(5000), resp.PendingBalance)
	require.Equal(t, uint64(12000), resp.SettleBalance)
}

func TestHTTPClientGenerateUpstreamFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/generate", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": false,
			"error":  "Vendor relation not found",
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, 0)

	_, err := client.Generate(context.Background(), GenerateRequest{
		Username: "player-001",
		Amount:   10000,
		UUID:     "dummy-merchant-uuid",
	})
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	require.Equal(t, "Vendor relation not found", apiErr.Message)
}

func TestHTTPClientGenerateNormalizesRelativeExpiredAtToUnixSeconds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/generate", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     true,
			"data":       "qr-data",
			"trx_id":     "trx-001",
			"expired_at": 300,
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, 0)
	startedAt := time.Now().UTC().Unix()

	resp, err := client.Generate(context.Background(), GenerateRequest{
		Username: "player-001",
		Amount:   10000,
		UUID:     "dummy-merchant-uuid",
		Expire:   ptrInt(300),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.ExpiredAt)
	require.GreaterOrEqual(t, *resp.ExpiredAt, startedAt+300)
	require.LessOrEqual(t, *resp.ExpiredAt, time.Now().UTC().Unix()+300)
}

func TestHTTPClientGenerateNormalizesMillisecondsExpiredAt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":     true,
			"data":       "qr-data",
			"trx_id":     "trx-001",
			"expired_at": int64(1770000000000),
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, 0)
	resp, err := client.Generate(context.Background(), GenerateRequest{
		Username: "player-001",
		Amount:   10000,
		UUID:     "dummy-merchant-uuid",
	})
	require.NoError(t, err)
	require.NotNil(t, resp.ExpiredAt)
	require.Equal(t, int64(1770000000), *resp.ExpiredAt)
}

func TestHTTPClientNon2xxReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, 0)
	_, err := client.GetBalance(context.Background(), "merchant-123", GetBalanceRequest{Client: "dewifork"})
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	require.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	require.Contains(t, apiErr.Body, "unauthorized")
}

func ptrInt(value int) *int {
	return &value
}
