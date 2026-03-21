package http

import (
	"net/http"
	"strings"

	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/response"
	"github.com/example/gue/backend/pkg/security"
	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth          service.AuthUseCase
	cookieManager *security.CookieManager
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
}

type authSessionResponse struct {
	User      service.UserDTO `json:"user"`
	ExpiresIn int64           `json:"expires_in"`
	CSRFToken string          `json:"csrf_token"`
}

func NewAuthHandler(auth service.AuthUseCase, cookieManager *security.CookieManager) *AuthHandler {
	return &AuthHandler{
		auth:          auth,
		cookieManager: cookieManager,
	}
}

func (h *AuthHandler) CSRF(c *gin.Context) {
	token, err := h.cookieManager.EnsureCSRFCookie(c)
	if err != nil {
		handleError(c, apperror.New(http.StatusInternalServerError, "failed to issue csrf token", err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"csrf_token": token,
		},
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req service.RegisterInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	result, err := h.auth.Register(c.Request.Context(), req)
	if err != nil {
		handleError(c, err)
		return
	}

	h.cookieManager.SetAuthCookies(c, result.AccessToken, result.RefreshToken)
	csrfToken, csrfErr := h.cookieManager.EnsureCSRFCookie(c)
	if csrfErr != nil {
		handleError(c, apperror.New(http.StatusInternalServerError, "failed to issue csrf token", csrfErr.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"data": authSessionResponse{
			User:      result.User,
			ExpiresIn: result.ExpiresIn,
			CSRFToken: csrfToken,
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	result, err := h.auth.Login(c.Request.Context(), req)
	if err != nil {
		handleError(c, err)
		return
	}

	h.cookieManager.SetAuthCookies(c, result.AccessToken, result.RefreshToken)
	csrfToken, csrfErr := h.cookieManager.EnsureCSRFCookie(c)
	if csrfErr != nil {
		handleError(c, apperror.New(http.StatusInternalServerError, "failed to issue csrf token", csrfErr.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": authSessionResponse{
			User:      result.User,
			ExpiresIn: result.ExpiresIn,
			CSRFToken: csrfToken,
		},
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := h.readRefreshToken(c)
	if err != nil {
		handleError(c, err)
		return
	}

	result, err := h.auth.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		handleError(c, err)
		return
	}

	h.cookieManager.SetAuthCookies(c, result.AccessToken, result.RefreshToken)
	csrfToken, csrfErr := h.cookieManager.EnsureCSRFCookie(c)
	if csrfErr != nil {
		handleError(c, apperror.New(http.StatusInternalServerError, "failed to issue csrf token", csrfErr.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": authSessionResponse{
			User:      result.User,
			ExpiresIn: result.ExpiresIn,
			CSRFToken: csrfToken,
		},
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, err := h.readRefreshToken(c)
	if err != nil {
		handleError(c, err)
		return
	}

	if err := h.auth.Logout(c.Request.Context(), refreshToken); err != nil {
		handleError(c, err)
		return
	}
	h.cookieManager.ClearAuthCookies(c)
	h.cookieManager.ClearCSRFCookie(c)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "logged out",
	})
}

func (h *AuthHandler) readRefreshToken(c *gin.Context) (string, error) {
	if token, err := h.cookieManager.ReadRefreshToken(c); err == nil && strings.TrimSpace(token) != "" {
		return token, nil
	}

	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return "", apperror.New(http.StatusBadRequest, "invalid request payload", err.Error())
	}
	token := strings.TrimSpace(req.RefreshToken)
	if token == "" {
		return "", apperror.New(http.StatusUnauthorized, "missing refresh token", nil)
	}
	return token, nil
}
