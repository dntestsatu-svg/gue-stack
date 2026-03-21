package middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	allowedOrigins := parseAllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))
	allowAnyOrigin := len(allowedOrigins) == 1 && allowedOrigins[0] == "*"

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		switch {
		case origin == "":
			// Non-browser client or same-origin request without Origin header.
		case allowAnyOrigin:
			// Cookies cannot be used with wildcard origin, so reflect Origin for credentialed requests.
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Vary", "Origin")
		case isOriginAllowed(origin, allowedOrigins):
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Vary", "Origin")
		default:
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(403)
				return
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Request-ID, X-CSRF-Token")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func parseAllowedOrigins(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		}
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin == "" {
			continue
		}
		result = append(result, origin)
	}
	if len(result) == 0 {
		return []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	}
	return result
}

func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}
