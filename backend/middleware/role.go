package middleware

import (
	"net/http"

	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

func RoleRequired(roles ...model.UserRole) gin.HandlerFunc {
	allowed := make(map[model.UserRole]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c *gin.Context) {
		rawRole, ok := c.Get(ContextKeyUserRole)
		if !ok {
			response.Error(c, apperror.New(http.StatusUnauthorized, "missing user role in context", nil))
			return
		}

		role, ok := rawRole.(model.UserRole)
		if !ok {
			response.Error(c, apperror.New(http.StatusUnauthorized, "invalid user role in context", nil))
			return
		}

		if _, exists := allowed[role]; !exists {
			response.Error(c, apperror.New(http.StatusForbidden, "insufficient role permission", nil))
			return
		}

		c.Next()
	}
}
