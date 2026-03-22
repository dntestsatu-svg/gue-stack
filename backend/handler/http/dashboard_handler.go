package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/gue/backend/middleware"
	"github.com/example/gue/backend/model"
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
	uid, role, err := readDashboardContext(c)
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
	uid, _, err := readDashboardContext(c)
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
	uid, _, err := readDashboardContext(c)
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

func readDashboardContext(c *gin.Context) (uint64, model.UserRole, error) {
	rawUserID, ok := c.Get(middleware.ContextKeyUserID)
	if !ok {
		return 0, "", apperror.New(http.StatusUnauthorized, "unauthorized", nil)
	}
	userID, ok := rawUserID.(uint64)
	if !ok {
		return 0, "", apperror.New(http.StatusUnauthorized, "invalid user id in context", nil)
	}

	rawRole, ok := c.Get(middleware.ContextKeyUserRole)
	if !ok {
		return 0, "", apperror.New(http.StatusUnauthorized, "missing user role", nil)
	}
	role, ok := rawRole.(model.UserRole)
	if !ok {
		return 0, "", apperror.New(http.StatusUnauthorized, "invalid user role", nil)
	}
	return userID, role, nil
}

func parseTransactionHistoryQuery(c *gin.Context) (service.TransactionHistoryQuery, error) {
	query := service.TransactionHistoryQuery{
		Limit:      20,
		Offset:     0,
		SearchTerm: strings.TrimSpace(c.Query("q")),
	}

	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			return service.TransactionHistoryQuery{}, apperror.New(http.StatusBadRequest, "invalid limit query parameter", nil)
		}
		query.Limit = parsed
	}
	if rawOffset := strings.TrimSpace(c.Query("offset")); rawOffset != "" {
		parsed, err := strconv.Atoi(rawOffset)
		if err != nil {
			return service.TransactionHistoryQuery{}, apperror.New(http.StatusBadRequest, "invalid offset query parameter", nil)
		}
		query.Offset = parsed
	}

	if rawFrom := strings.TrimSpace(c.Query("from")); rawFrom != "" {
		parsed, err := parseQueryDate(rawFrom, false)
		if err != nil {
			return service.TransactionHistoryQuery{}, apperror.New(http.StatusBadRequest, "invalid from query parameter", nil)
		}
		query.From = &parsed
	}
	if rawTo := strings.TrimSpace(c.Query("to")); rawTo != "" {
		parsed, err := parseQueryDate(rawTo, true)
		if err != nil {
			return service.TransactionHistoryQuery{}, apperror.New(http.StatusBadRequest, "invalid to query parameter", nil)
		}
		query.To = &parsed
	}
	return query, nil
}

func parseQueryDate(value string, endOfDay bool) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), nil
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		utc := parsed.UTC()
		if endOfDay {
			return utc.Add(23*time.Hour + 59*time.Minute + 59*time.Second), nil
		}
		return utc, nil
	}
	return time.Time{}, apperror.New(http.StatusBadRequest, "invalid date format", "supported formats: RFC3339 or YYYY-MM-DD")
}
