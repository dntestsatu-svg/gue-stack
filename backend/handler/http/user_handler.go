package http

import (
	"net/http"
	"strconv"

	"github.com/example/gue/backend/middleware"
	"github.com/example/gue/backend/model"
	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/response"
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
	userID, role, err := readAuthContext(c)
	if err != nil {
		handleError(c, err)
		return
	}
	_ = role

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

func (h *UserHandler) Create(c *gin.Context) {
	userID, actorRole, err := readAuthContext(c)
	if err != nil {
		handleError(c, err)
		return
	}
	_ = userID

	var req service.CreateUserInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	created, err := h.user.Create(c.Request.Context(), actorRole, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"data":   created,
	})
}

func (h *UserHandler) UpdateRole(c *gin.Context) {
	_, actorRole, err := readAuthContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || targetID == 0 {
		handleError(c, apperror.New(http.StatusBadRequest, "invalid user id", nil))
		return
	}

	var req service.UpdateUserRoleInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	updated, err := h.user.UpdateRole(c.Request.Context(), actorRole, targetID, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   updated,
	})
}

func readAuthContext(c *gin.Context) (uint64, model.UserRole, error) {
	rawUserID, ok := c.Get(middleware.ContextKeyUserID)
	if !ok {
		return 0, "", apperror.New(http.StatusUnauthorized, "unauthorized", nil)
	}
	userID, ok := rawUserID.(uint64)
	if !ok {
		return 0, "", apperror.New(http.StatusUnauthorized, "invalid token claims", nil)
	}

	rawRole, ok := c.Get(middleware.ContextKeyUserRole)
	if !ok {
		return 0, "", apperror.New(http.StatusUnauthorized, "missing user role", nil)
	}
	role, ok := rawRole.(model.UserRole)
	if !ok {
		return 0, "", apperror.New(http.StatusUnauthorized, "invalid user role", nil)
	}

	return userID, role, nil
}
