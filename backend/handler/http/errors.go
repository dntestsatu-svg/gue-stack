package http

import (
	"errors"

	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

func handleError(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		response.Error(c, appErr)
		return
	}
	response.Internal(c, err)
}
