package http

import (
	"net/http"
	"strconv"

	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/response"
	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
)

type TokoHandler struct {
	toko service.TokoUseCase
}

func NewTokoHandler(toko service.TokoUseCase) *TokoHandler {
	return &TokoHandler{toko: toko}
}

func (h *TokoHandler) List(c *gin.Context) {
	userID, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	items, err := h.toko.ListByUser(c.Request.Context(), userID, actorRole)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   items,
	})
}

func (h *TokoHandler) Workspace(c *gin.Context) {
	userID, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	query, err := parseTokoWorkspaceQuery(c)
	if err != nil {
		handleError(c, err)
		return
	}

	page, err := h.toko.Workspace(c.Request.Context(), userID, actorRole, query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   page,
	})
}

func (h *TokoHandler) Create(c *gin.Context) {
	userID, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	var req service.CreateTokoInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	item, err := h.toko.CreateForUser(c.Request.Context(), userID, actorRole, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"data":   item,
	})
}

func (h *TokoHandler) Update(c *gin.Context) {
	userID, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	tokoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || tokoID == 0 {
		handleError(c, apperror.New(http.StatusBadRequest, "invalid toko id", nil))
		return
	}

	var req service.UpdateTokoInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	item, err := h.toko.Update(c.Request.Context(), userID, actorRole, tokoID, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   item,
	})
}

func (h *TokoHandler) RegenerateToken(c *gin.Context) {
	userID, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	tokoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || tokoID == 0 {
		handleError(c, apperror.New(http.StatusBadRequest, "invalid toko id", nil))
		return
	}

	item, err := h.toko.RegenerateToken(c.Request.Context(), userID, actorRole, tokoID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   item,
	})
}

func (h *TokoHandler) ListBalances(c *gin.Context) {
	userID, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	items, err := h.toko.ListBalancesByUser(c.Request.Context(), userID, actorRole)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   items,
	})
}

func (h *TokoHandler) ManualSettlement(c *gin.Context) {
	_, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	tokoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || tokoID == 0 {
		handleError(c, apperror.New(http.StatusBadRequest, "invalid toko id", nil))
		return
	}

	var req service.ManualSettlementInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	item, err := h.toko.ManualSettlement(c.Request.Context(), actorRole, tokoID, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   item,
	})
}
