package http

import (
	"net/http"
	"strconv"

	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/response"
	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
)

type BankHandler struct {
	bank service.BankUseCase
}

func NewBankHandler(bank service.BankUseCase) *BankHandler {
	return &BankHandler{bank: bank}
}

func (h *BankHandler) List(c *gin.Context) {
	userID, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	query, err := parseBankListQuery(c)
	if err != nil {
		handleError(c, err)
		return
	}

	page, err := h.bank.List(c.Request.Context(), userID, actorRole, query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   page,
	})
}

func (h *BankHandler) PaymentOptions(c *gin.Context) {
	_, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	query, err := parsePaymentOptionQuery(c)
	if err != nil {
		handleError(c, err)
		return
	}

	items, err := h.bank.PaymentOptions(c.Request.Context(), actorRole, query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   items,
	})
}

func (h *BankHandler) Inquiry(c *gin.Context) {
	userID, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	var req service.BankInquiryInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	result, err := h.bank.Inquiry(c.Request.Context(), userID, actorRole, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

func (h *BankHandler) Create(c *gin.Context) {
	userID, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	var req service.CreateBankInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	item, err := h.bank.Create(c.Request.Context(), userID, actorRole, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"data":   item,
	})
}

func (h *BankHandler) Delete(c *gin.Context) {
	userID, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	bankID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || bankID == 0 {
		handleError(c, apperror.New(http.StatusBadRequest, "invalid bank id", nil))
		return
	}

	if err := h.bank.Delete(c.Request.Context(), userID, actorRole, bankID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "bank deleted successfully",
	})
}
