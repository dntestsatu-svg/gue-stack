package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
)

func parseIntQuery(c *gin.Context, key string, fallback int) (int, error) {
	rawValue := strings.TrimSpace(c.Query(key))
	if rawValue == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(rawValue)
	if err != nil {
		return 0, apperror.New(http.StatusBadRequest, "invalid "+key+" query parameter", nil)
	}

	return value, nil
}

func parseTransactionHistoryQuery(c *gin.Context) (service.TransactionHistoryQuery, error) {
	limit, err := parseIntQuery(c, "limit", 20)
	if err != nil {
		return service.TransactionHistoryQuery{}, err
	}

	offset, err := parseIntQuery(c, "offset", 0)
	if err != nil {
		return service.TransactionHistoryQuery{}, err
	}

	query := service.TransactionHistoryQuery{
		Limit:      limit,
		Offset:     offset,
		SearchTerm: strings.TrimSpace(c.Query("q")),
	}

	if rawFrom := strings.TrimSpace(c.Query("from")); rawFrom != "" {
		parsed, parseErr := parseQueryDate(rawFrom, false)
		if parseErr != nil {
			return service.TransactionHistoryQuery{}, apperror.New(http.StatusBadRequest, "invalid from query parameter", nil)
		}
		query.From = &parsed
	}

	if rawTo := strings.TrimSpace(c.Query("to")); rawTo != "" {
		parsed, parseErr := parseQueryDate(rawTo, true)
		if parseErr != nil {
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

	return time.Time{}, apperror.New(
		http.StatusBadRequest,
		"invalid date format",
		"supported formats: RFC3339 or YYYY-MM-DD",
	)
}
