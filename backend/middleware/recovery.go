package middleware

import (
	"net/http"

	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, rec any) {
		response.Error(c, &apperror.AppError{
			StatusCode: http.StatusInternalServerError,
			Message:    "internal server error",
			Details:    rec,
		})
	})
}
