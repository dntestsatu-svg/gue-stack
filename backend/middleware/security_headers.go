package middleware

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := c.Writer.Header()
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		headers.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		headers.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if shouldNoIndexResponse(c.Request.Host, c.Request.URL.Path) {
			headers.Set("X-Robots-Tag", "noindex, nofollow")
		}

		if isHTTPSRequest(c) {
			headers.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		}

		c.Next()
	}
}

func shouldNoIndexResponse(host, requestPath string) bool {
	cleanHost := normalizeHost(host)
	if strings.HasPrefix(cleanHost, "api.") {
		return true
	}
	return requestPath == "/openapi.yaml" || strings.HasPrefix(requestPath, "/api/")
}

func normalizeHost(host string) string {
	trimmed := strings.TrimSpace(host)
	if trimmed == "" {
		return ""
	}
	if parsedHost, _, err := net.SplitHostPort(trimmed); err == nil {
		return strings.ToLower(parsedHost)
	}
	return strings.ToLower(trimmed)
}

func isHTTPSRequest(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")), "https")
}
