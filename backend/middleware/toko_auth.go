package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/response"
	"github.com/example/gue/backend/repository"
	"github.com/gin-gonic/gin"
)

func TokoTokenRequired(tokoRepo repository.TokoRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			response.Error(c, apperror.New(http.StatusUnauthorized, "missing authorization header", nil))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Error(c, apperror.New(http.StatusUnauthorized, "invalid authorization header", nil))
			return
		}

		token := strings.TrimSpace(parts[1])
		if token == "" {
			response.Error(c, apperror.New(http.StatusUnauthorized, "invalid toko token", nil))
			return
		}

		toko, err := tokoRepo.GetByToken(c.Request.Context(), token)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				response.Error(c, apperror.New(http.StatusUnauthorized, "invalid toko token", nil))
				return
			}
			response.Error(c, apperror.New(http.StatusInternalServerError, "failed to fetch toko", err.Error()))
			return
		}

		c.Set(ContextKeyTokoID, toko.ID)
		c.Next()
	}
}
