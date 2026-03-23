package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/gue/backend/config"
	"github.com/example/gue/backend/pkg/security"
	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mockAuthService struct {
	registerResp *service.AuthResult
	registerErr  error
	sessionResp  *service.SessionStatusResult
	sessionErr   error
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
func (m *mockAuthService) SessionStatus(_ context.Context, _ string) (*service.SessionStatusResult, error) {
	return m.sessionResp, m.sessionErr
}
func (m *mockAuthService) Logout(_ context.Context, _ string) error { return nil }

func testCookieManager() *security.CookieManager {
	return security.NewCookieManager(
		config.CookieConfig{
			AccessTokenName:  "access_token",
			RefreshTokenName: "refresh_token",
			CSRFCookieName:   "csrf_token",
			SessionHintName:  "session_hint",
			Domain:           "",
			Path:             "/",
			Secure:           false,
			HTTPOnly:         true,
			SameSite:         "strict",
		},
		15*time.Minute,
		7*24*time.Hour,
	)
}

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
	h := NewAuthHandler(mockSvc, testCookieManager())

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
	require.Contains(t, w.Body.String(), "csrf_token")
	setCookies := w.Result().Cookies()
	require.NotEmpty(t, setCookies)
}

func TestAuthHandler_RefreshMissingTokenReturnsUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAuthHandler(&mockAuthService{}, testCookieManager())

	r := gin.New()
	r.POST("/refresh", h.Refresh)

	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(nil))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "missing refresh token")
}

func TestAuthHandler_SessionReturnsUnauthenticatedWithoutRefreshCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAuthHandler(&mockAuthService{}, testCookieManager())

	r := gin.New()
	r.GET("/session", h.Session)

	req := httptest.NewRequest(http.MethodGet, "/session", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"authenticated":false`)
}

func TestAuthHandler_SessionReturnsAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAuthHandler(&mockAuthService{
		sessionResp: &service.SessionStatusResult{
			Authenticated: true,
			User: &service.UserDTO{
				ID:       7,
				Name:     "Admin",
				Email:    "admin@example.com",
				Role:     "admin",
				IsActive: true,
			},
		},
	}, testCookieManager())

	r := gin.New()
	r.GET("/session", h.Session)

	req := httptest.NewRequest(http.MethodGet, "/session", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "valid-refresh"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"authenticated":true`)
	require.Contains(t, w.Body.String(), `admin@example.com`)
}
