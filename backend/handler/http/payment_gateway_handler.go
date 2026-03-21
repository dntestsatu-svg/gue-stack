package http

import (
	"net/http"

	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/response"
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
	var req service.GeneratePaymentInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	result, err := h.gateway.Generate(c.Request.Context(), req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": result})
}

func (h *PaymentGatewayHandler) CheckStatusV2(c *gin.Context) {
	var req service.CheckPaymentStatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	trxID := c.Param("trx_id")
	result, err := h.gateway.CheckStatusV2(c.Request.Context(), trxID, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": result})
}

func (h *PaymentGatewayHandler) InquiryTransfer(c *gin.Context) {
	var req service.InquiryTransferInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	result, err := h.gateway.InquiryTransfer(c.Request.Context(), req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": result})
}

func (h *PaymentGatewayHandler) TransferFund(c *gin.Context) {
	var req service.TransferFundInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	result, err := h.gateway.TransferFund(c.Request.Context(), req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": result})
}

func (h *PaymentGatewayHandler) CheckTransferStatus(c *gin.Context) {
	var req service.CheckTransferStatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	partnerRefNo := c.Param("partner_ref_no")
	result, err := h.gateway.CheckTransferStatus(c.Request.Context(), partnerRefNo, req)
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

	merchantUUID := c.Param("merchant_uuid")
	result, err := h.gateway.GetBalance(c.Request.Context(), merchantUUID, req)
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

	var req service.QrisCallbackPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid callback payload", err.Error()))
		return
	}

	if err := h.gateway.HandleQrisCallback(c.Request.Context(), req); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "callback processed"})
}

func (h *PaymentGatewayHandler) TransferCallback(c *gin.Context) {
	if err := h.gateway.ValidateCallbackSecret(c.GetHeader("X-Callback-Secret")); err != nil {
		handleError(c, err)
		return
	}

	var req service.TransferCallbackPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid callback payload", err.Error()))
		return
	}

	if err := h.gateway.HandleTransferCallback(c.Request.Context(), req); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "callback processed"})
}
