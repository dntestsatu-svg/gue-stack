package middleware

import (
	"net/http"
	"strings"

	"github.com/example/gue/backend/config"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/response"
	"github.com/example/gue/backend/pkg/security"
	"github.com/gin-gonic/gin"
)

func CSRFProtection(
	csrfCfg config.CSRFConfig,
	cookieManager *security.CookieManager,
	skipPrefixes ...string,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !csrfCfg.Enabled || isSafeMethod(c.Request.Method) || hasSkippedPrefix(c.Request.URL.Path, skipPrefixes) {
			c.Next()
			return
		}

		// CSRF is required for cookie-authenticated requests. API clients using bearer token only are exempt.
		if isBearerOnlyRequest(c, cookieManager) {
			c.Next()
			return
		}

		cookieToken, err := cookieManager.CSRFTokenFromCookie(c)
		if err != nil || strings.TrimSpace(cookieToken) == "" {
			response.Error(c, apperror.New(http.StatusForbidden, "missing csrf cookie", nil))
			return
		}

		headerToken := strings.TrimSpace(c.GetHeader(csrfCfg.HeaderName))
		if headerToken == "" {
			response.Error(c, apperror.New(http.StatusForbidden, "missing csrf header", nil))
			return
		}
		if headerToken != cookieToken {
			response.Error(c, apperror.New(http.StatusForbidden, "invalid csrf token", nil))
			return
		}

		c.Next()
	}
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

func hasSkippedPrefix(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func isBearerOnlyRequest(c *gin.Context, cookieManager *security.CookieManager) bool {
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return false
	}
	if _, err := cookieManager.ReadAccessToken(c); err == nil {
		return false
	}
	return true
}
