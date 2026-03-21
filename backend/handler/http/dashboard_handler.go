package http

import (
	"net/http"
	"strconv"

	"github.com/example/gue/backend/middleware"
	"github.com/example/gue/backend/pkg/apperror"
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
	userID, ok := c.Get(middleware.ContextKeyUserID)
	if !ok {
		handleError(c, apperror.New(http.StatusUnauthorized, "unauthorized", nil))
		return
	}
	uid, ok := userID.(uint64)
	if !ok {
		handleError(c, apperror.New(http.StatusUnauthorized, "invalid user id in context", nil))
		return
	}

	data, err := h.dashboard.Overview(c.Request.Context(), uid)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   data,
	})
}

func (h *DashboardHandler) TransactionHistory(c *gin.Context) {
	userID, ok := c.Get(middleware.ContextKeyUserID)
	if !ok {
		handleError(c, apperror.New(http.StatusUnauthorized, "unauthorized", nil))
		return
	}
	uid, ok := userID.(uint64)
	if !ok {
		handleError(c, apperror.New(http.StatusUnauthorized, "invalid user id in context", nil))
		return
	}

	limit := 20
	if rawLimit := c.Query("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			handleError(c, apperror.New(http.StatusBadRequest, "invalid limit query parameter", nil))
			return
		}
		limit = parsed
	}

	data, err := h.dashboard.TransactionHistory(c.Request.Context(), uid, limit)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   data,
	})
}
