package http

import (
	"net/http"

	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/response"
	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth service.AuthUseCase
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func NewAuthHandler(auth service.AuthUseCase) *AuthHandler {
	return &AuthHandler{auth: auth}
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

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"data":   result,
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

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	result, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	if err := h.auth.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "logged out",
	})
}
