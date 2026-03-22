package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/gue/backend/middleware"
	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mockTestingUseCase struct {
	generateQrisFn           func(ctx context.Context, userID uint64, actorRole model.UserRole, input service.TestingGenerateQrisInput) (*service.TestingGenerateQrisResult, error)
	checkCallbackReadinessFn func(ctx context.Context, userID uint64, actorRole model.UserRole, input service.TestingCallbackReadinessInput) (*service.TestingCallbackReadinessResult, error)
}

func (m *mockTestingUseCase) GenerateQris(ctx context.Context, userID uint64, actorRole model.UserRole, input service.TestingGenerateQrisInput) (*service.TestingGenerateQrisResult, error) {
	if m.generateQrisFn == nil {
		return nil, nil
	}
	return m.generateQrisFn(ctx, userID, actorRole, input)
}

func (m *mockTestingUseCase) CheckCallbackReadiness(ctx context.Context, userID uint64, actorRole model.UserRole, input service.TestingCallbackReadinessInput) (*service.TestingCallbackReadinessResult, error) {
	if m.checkCallbackReadinessFn == nil {
		return nil, nil
	}
	return m.checkCallbackReadinessFn(ctx, userID, actorRole, input)
}

func withTestingAuth(userID uint64, role model.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, userID)
		c.Set(middleware.ContextKeyUserRole, role)
		c.Next()
	}
}

func TestTestingHandlerGenerateQrisAndCallbackReadiness(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUseCase := &mockTestingUseCase{
		generateQrisFn: func(_ context.Context, userID uint64, actorRole model.UserRole, input service.TestingGenerateQrisInput) (*service.TestingGenerateQrisResult, error) {
			require.Equal(t, uint64(11), userID)
			require.Equal(t, model.UserRoleAdmin, actorRole)
			require.Equal(t, uint64(3), input.TokoID)
			return &service.TestingGenerateQrisResult{
				TokoID:   3,
				TokoName: "Toko Test",
				TrxID:    "trx-001",
				Data:     "qr-data",
			}, nil
		},
		checkCallbackReadinessFn: func(_ context.Context, userID uint64, actorRole model.UserRole, input service.TestingCallbackReadinessInput) (*service.TestingCallbackReadinessResult, error) {
			require.Equal(t, uint64(11), userID)
			require.Equal(t, model.UserRoleAdmin, actorRole)
			require.Equal(t, uint64(3), input.TokoID)
			return &service.TestingCallbackReadinessResult{
				TokoID:      3,
				TokoName:    "Toko Test",
				Ready:       true,
				Message:     "API kamu sudah ready.",
				StatusCode:  http.StatusOK,
				CallbackURL: "https://merchant.example.com/callback",
			}, nil
		},
	}

	handler := NewTestingHandler(mockUseCase)
	router := gin.New()
	router.POST("/testing/generate-qris", withTestingAuth(11, model.UserRoleAdmin), handler.GenerateQris)
	router.POST("/testing/callback-readiness", withTestingAuth(11, model.UserRoleAdmin), handler.CheckCallbackReadiness)

	tests := []struct {
		name           string
		path           string
		body           map[string]any
		expectedStatus int
	}{
		{
			name:           "generate qris success",
			path:           "/testing/generate-qris",
			body:           map[string]any{"toko_id": 3, "username": "player-1", "amount": 25000},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "callback readiness success",
			path:           "/testing/callback-readiness",
			body:           map[string]any{"toko_id": 3},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			require.Equal(t, tt.expectedStatus, recorder.Code)
		})
	}
}

func TestTestingHandlerInvalidPayloadAndUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUseCase := &mockTestingUseCase{
		generateQrisFn: func(_ context.Context, _ uint64, _ model.UserRole, _ service.TestingGenerateQrisInput) (*service.TestingGenerateQrisResult, error) {
			return nil, apperror.New(http.StatusInternalServerError, "unexpected", nil)
		},
	}

	handler := NewTestingHandler(mockUseCase)
	router := gin.New()
	router.POST("/testing/generate-qris", handler.GenerateQris)

	req := httptest.NewRequest(http.MethodPost, "/testing/generate-qris", bytes.NewBufferString(`{"toko_id":0}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	router = gin.New()
	router.POST("/testing/callback-readiness", withTestingAuth(1, model.UserRoleUser), handler.CheckCallbackReadiness)
	req = httptest.NewRequest(http.MethodPost, "/testing/callback-readiness", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}
