package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mockAuthService struct {
	registerResp *service.AuthResult
	registerErr  error
}

func (m *mockAuthService) Register(_ context.Context, _ service.RegisterInput) (*service.AuthResult, error) {
	return m.registerResp, m.registerErr
}
func (m *mockAuthService) Login(_ context.Context, _ service.LoginInput) (*service.AuthResult, error) {
	return nil, nil
}
func (m *mockAuthService) Refresh(_ context.Context, _ string) (*service.AuthResult, error) {
	return nil, nil
}
func (m *mockAuthService) Logout(_ context.Context, _ string) error { return nil }

func TestAuthHandler_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := &mockAuthService{
		registerResp: &service.AuthResult{
			User:         service.UserDTO{ID: 1, Name: "Jane", Email: "jane@example.com"},
			AccessToken:  "access",
			RefreshToken: "refresh",
			ExpiresIn:    900,
		},
	}
	h := NewAuthHandler(mockSvc)

	r := gin.New()
	r.POST("/register", h.Register)

	body := map[string]any{
		"name":     "Jane",
		"email":    "jane@example.com",
		"password": "secret123",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Contains(t, w.Body.String(), "success")
	require.Contains(t, w.Body.String(), "access")
}
