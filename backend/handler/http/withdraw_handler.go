package http

import (
	"net/http"

	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/response"
	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
)

type WithdrawHandler struct {
	withdraw service.WithdrawUseCase
}

func NewWithdrawHandler(withdraw service.WithdrawUseCase) *WithdrawHandler {
	return &WithdrawHandler{withdraw: withdraw}
}

func (h *WithdrawHandler) Options(c *gin.Context) {
	userID, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	result, err := h.withdraw.Options(c.Request.Context(), userID, actorRole)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

func (h *WithdrawHandler) Inquiry(c *gin.Context) {
	userID, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	var req service.WithdrawInquiryInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	result, err := h.withdraw.Inquiry(c.Request.Context(), userID, actorRole, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

func (h *WithdrawHandler) History(c *gin.Context) {
	userID, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	query, err := parseWithdrawHistoryQuery(c)
	if err != nil {
		handleError(c, err)
		return
	}

	result, err := h.withdraw.History(c.Request.Context(), userID, actorRole, query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

func (h *WithdrawHandler) Transfer(c *gin.Context) {
	userID, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	var req service.WithdrawTransferInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	result, err := h.withdraw.Transfer(c.Request.Context(), userID, actorRole, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}
