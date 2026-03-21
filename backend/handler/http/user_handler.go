package http

import (
	"net/http"

	"github.com/example/gue/backend/middleware"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/service"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	user service.UserUseCase
}

func NewUserHandler(user service.UserUseCase) *UserHandler {
	return &UserHandler{user: user}
}

func (h *UserHandler) Me(c *gin.Context) {
	value, ok := c.Get(middleware.ContextKeyUserID)
	if !ok {
		handleError(c, apperror.New(http.StatusUnauthorized, "unauthorized", nil))
		return
	}
	userID, ok := value.(uint64)
	if !ok {
		handleError(c, apperror.New(http.StatusUnauthorized, "invalid token claims", nil))
		return
	}

	me, err := h.user.Me(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   me,
	})
}
