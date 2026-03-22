package http

import (
	"net/http"
	"strings"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	dashboard service.DashboardUseCase
}

func NewDashboardHandler(dashboard service.DashboardUseCase) *DashboardHandler {
	return &DashboardHandler{dashboard: dashboard}
}

func (h *DashboardHandler) Overview(c *gin.Context) {
	uid, role, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	data, err := h.dashboard.Overview(c.Request.Context(), uid)
	if err != nil {
		handleError(c, err)
		return
	}
	data.CanViewProjectProfit = role == model.UserRoleDev
	if !data.CanViewProjectProfit {
		data.Metrics.ProjectProfit = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   data,
	})
}

func (h *DashboardHandler) TransactionHistory(c *gin.Context) {
	uid, _, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	query, err := parseTransactionHistoryQuery(c)
	if err != nil {
		handleError(c, err)
		return
	}

	data, err := h.dashboard.TransactionHistory(c.Request.Context(), uid, query)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   data,
	})
}

func (h *DashboardHandler) ExportTransactionHistory(c *gin.Context) {
	uid, _, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	query, err := parseTransactionHistoryQuery(c)
	if err != nil {
		handleError(c, err)
		return
	}
	format := strings.ToLower(strings.TrimSpace(c.DefaultQuery("format", "csv")))

	exported, err := h.dashboard.ExportTransactionHistory(c.Request.Context(), uid, query, format)
	if err != nil {
		handleError(c, err)
		return
	}

	c.Header("Content-Type", exported.ContentType)
	c.Header("Content-Disposition", `attachment; filename="`+exported.FileName+`"`)
	c.Data(http.StatusOK, exported.ContentType, exported.Content)
}
