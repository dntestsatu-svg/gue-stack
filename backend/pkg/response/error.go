package response

import (
	"net/http"

	"github.com/example/gue/backend/pkg/apperror"
	"github.com/gin-gonic/gin"
)

type ErrorBody struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func Error(c *gin.Context, appErr *apperror.AppError) {
	c.AbortWithStatusJSON(appErr.StatusCode, ErrorBody{
		Status:  "error",
		Message: appErr.Message,
		Details: appErr.Details,
	})
}

func Internal(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorBody{
		Status:  "error",
		Message: "internal server error",
		Details: err.Error(),
	})
}
