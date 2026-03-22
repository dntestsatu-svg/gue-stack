package http

import (
	"net/http"
	"strconv"

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
	userID, _, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
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

func (h *UserHandler) List(c *gin.Context) {
	actorUserID, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	limit, err := parseIntQuery(c, "limit", 50)
	if err != nil {
		handleError(c, err)
		return
	}

	users, err := h.user.List(c.Request.Context(), actorUserID, actorRole, limit)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   users,
	})
}

func (h *UserHandler) Create(c *gin.Context) {
	actorUserID, actorRole, err := readUserContext(c)
	if err != nil {
		handleError(c, err)
		return
	}

	var req service.CreateUserInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.New(http.StatusBadRequest, "invalid request payload", err.Error()))
		return
	}

	created, err := h.user.Create(c.Request.Context(), actorUserID, actorRole, req)
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
	actorUserID, actorRole, err := readUserContext(c)
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

	updated, err := h.user.UpdateRole(c.Request.Context(), actorUserID, actorRole, targetID, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   updated,
	})
}
