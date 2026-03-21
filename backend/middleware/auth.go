package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/apperror"
	jwtpkg "github.com/example/gue/backend/pkg/jwt"
	"github.com/example/gue/backend/pkg/response"
	"github.com/example/gue/backend/pkg/security"
	"github.com/example/gue/backend/repository"
	"github.com/gin-gonic/gin"
)

func AuthRequired(tokenManager *jwtpkg.Manager, userRepo repository.UserRepository, cookieManager *security.CookieManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken := ""
		if cookieManager != nil {
			token, err := cookieManager.ReadAccessToken(c)
			if err == nil && strings.TrimSpace(token) != "" {
				accessToken = strings.TrimSpace(token)
			}
		}

		if accessToken == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
					response.Error(c, apperror.New(http.StatusUnauthorized, "invalid authorization header", nil))
					return
				}
				accessToken = strings.TrimSpace(parts[1])
			}
		}
		if accessToken == "" {
			response.Error(c, apperror.New(http.StatusUnauthorized, "missing access token", nil))
			return
		}

		claims, err := tokenManager.ParseAccessToken(accessToken)
		if err != nil {
			response.Error(c, apperror.New(http.StatusUnauthorized, "invalid access token", err.Error()))
			return
		}

		user, err := userRepo.GetByID(c.Request.Context(), claims.UserID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				response.Error(c, apperror.New(http.StatusUnauthorized, "user not found", nil))
				return
			}
			response.Error(c, apperror.New(http.StatusInternalServerError, "failed to fetch user", err.Error()))
			return
		}
		if !user.IsActive {
			response.Error(c, apperror.New(http.StatusForbidden, "user account is inactive", nil))
			return
		}
		if user.Role == "" {
			user.Role = model.UserRoleUser
		}

		c.Set(ContextKeyUserID, user.ID)
		c.Set(ContextKeyUserRole, user.Role)
		c.Next()
	}
}
