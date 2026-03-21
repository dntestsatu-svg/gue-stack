package middleware

import (
	"net/http"
	"strings"

	"github.com/example/gue/backend/pkg/apperror"
	jwtpkg "github.com/example/gue/backend/pkg/jwt"
	"github.com/example/gue/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

func AuthRequired(tokenManager *jwtpkg.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, apperror.New(http.StatusUnauthorized, "missing authorization header", nil))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Error(c, apperror.New(http.StatusUnauthorized, "invalid authorization header", nil))
			return
		}

		claims, err := tokenManager.ParseAccessToken(parts[1])
		if err != nil {
			response.Error(c, apperror.New(http.StatusUnauthorized, "invalid access token", err.Error()))
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Next()
	}
}
