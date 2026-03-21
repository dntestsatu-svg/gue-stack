package http

import (
	"net/http"

	"github.com/example/gue/backend/middleware"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/response"
	"github.com/example/gue/backend/queue"
	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
)

type PaymentGatewayHandler struct {
	gateway service.PaymentGatewayUseCase
}

func NewPaymentGatewayHandler(gateway service.PaymentGatewayUseCase) *PaymentGatewayHandler {
	return &PaymentGatewayHandler{gateway: gateway}
}

func (h *PaymentGatewayHandler) Generate(c *gin.Context) {
	tokoID, err := readTokoIDFromContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	var req service.GeneratePaymentInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	result, err := h.gateway.Generate(c.Request.Context(), tokoID, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": result})
}

func (h *PaymentGatewayHandler) CheckStatusV2(c *gin.Context) {
	tokoID, err := readTokoIDFromContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	var req service.CheckPaymentStatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	trxID := c.Param("trx_id")
	result, err := h.gateway.CheckStatusV2(c.Request.Context(), tokoID, trxID, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": result})
}

func (h *PaymentGatewayHandler) InquiryTransfer(c *gin.Context) {
	tokoID, err := readTokoIDFromContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	var req service.InquiryTransferInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	result, err := h.gateway.InquiryTransfer(c.Request.Context(), tokoID, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": result})
}

func (h *PaymentGatewayHandler) TransferFund(c *gin.Context) {
	tokoID, err := readTokoIDFromContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	var req service.TransferFundInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	result, err := h.gateway.TransferFund(c.Request.Context(), tokoID, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": result})
}

func (h *PaymentGatewayHandler) CheckTransferStatus(c *gin.Context) {
	tokoID, err := readTokoIDFromContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	var req service.CheckTransferStatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	partnerRefNo := c.Param("partner_ref_no")
	result, err := h.gateway.CheckTransferStatus(c.Request.Context(), tokoID, partnerRefNo, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": result})
}

func (h *PaymentGatewayHandler) GetBalance(c *gin.Context) {
	var req service.GetBalanceInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	result, err := h.gateway.GetBalance(c.Request.Context(), req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": result})
}

func (h *PaymentGatewayHandler) QrisCallback(c *gin.Context) {
	if err := h.gateway.ValidateCallbackSecret(c.GetHeader("X-Callback-Secret")); err != nil {
		handleError(c, err)
		return
	}

	var req queue.QrisCallbackTaskPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid callback payload", err.Error()))
		return
	}

	if err := h.gateway.EnqueueQrisCallback(c.Request.Context(), req); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "success", "message": "callback accepted"})
}

func (h *PaymentGatewayHandler) TransferCallback(c *gin.Context) {
	if err := h.gateway.ValidateCallbackSecret(c.GetHeader("X-Callback-Secret")); err != nil {
		handleError(c, err)
		return
	}

	var req queue.TransferCallbackTaskPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid callback payload", err.Error()))
		return
	}

	if err := h.gateway.EnqueueTransferCallback(c.Request.Context(), req); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "success", "message": "callback accepted"})
}

func readTokoIDFromContext(c *gin.Context) (uint64, error) {
	rawTokoID, ok := c.Get(middleware.ContextKeyTokoID)
	if !ok {
		return 0, apperror.New(http.StatusUnauthorized, "invalid toko token", nil)
	}
	tokoID, ok := rawTokoID.(uint64)
	if !ok || tokoID == 0 {
		return 0, apperror.New(http.StatusUnauthorized, "invalid toko token", nil)
	}
	return tokoID, nil
}
