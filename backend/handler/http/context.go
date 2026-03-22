package http

import (
	"net/http"

	"github.com/example/gue/backend/middleware"
	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/gin-gonic/gin"
)

func readUserContext(c *gin.Context) (uint64, model.UserRole, error) {
	rawUserID, ok := c.Get(middleware.ContextKeyUserID)
	if !ok {
		return 0, "", apperror.New(http.StatusUnauthorized, "unauthorized", nil)
	}

	userID, ok := rawUserID.(uint64)
	if !ok || userID == 0 {
		return 0, "", apperror.New(http.StatusUnauthorized, "invalid user id in context", nil)
	}

	rawRole, ok := c.Get(middleware.ContextKeyUserRole)
	if !ok {
		return 0, "", apperror.New(http.StatusUnauthorized, "missing user role", nil)
	}

	role, ok := rawRole.(model.UserRole)
	if !ok || role == "" {
		return 0, "", apperror.New(http.StatusUnauthorized, "invalid user role", nil)
	}

	return userID, role, nil
}

func readTokoIDFromContext(c *gin.Context) (uint64, error) {
	rawTokoID, ok := c.Get(middleware.ContextKeyTokoID)
	if !ok {
		return 0, apperror.New(http.StatusUnauthorized, "invalid toko token", nil)
	}

	tokoID, ok := rawTokoID.(uint64)
	if !ok || tokoID == 0 {
		return 0, apperror.New(http.StatusUnauthorized, "invalid toko token", nil)
	}

	return tokoID, nil
}
