package http

import (
	"net/http"

	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/response"
	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
)

type TestingHandler struct {
	testing service.TestingUseCase
}

func NewTestingHandler(testing service.TestingUseCase) *TestingHandler {
	return &TestingHandler{testing: testing}
}

func (h *TestingHandler) GenerateQris(c *gin.Context) {
	userID, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	var req service.TestingGenerateQrisInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	result, err := h.testing.GenerateQris(c.Request.Context(), userID, actorRole, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

func (h *TestingHandler) CheckCallbackReadiness(c *gin.Context) {
	userID, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	var req service.TestingCallbackReadinessInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	result, err := h.testing.CheckCallbackReadiness(c.Request.Context(), userID, actorRole, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}
