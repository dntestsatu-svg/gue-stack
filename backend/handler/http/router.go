package http

import (
	"log/slog"
	"net/http"

	"github.com/example/gue/backend/middleware"
	jwtpkg "github.com/example/gue/backend/pkg/jwt"
	"github.com/gin-gonic/gin"
)

func NewRouter(
	logger *slog.Logger,
	tokenManager *jwtpkg.Manager,
	authHandler *AuthHandler,
	userHandler *UserHandler,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.CORS(), middleware.Logger(logger), middleware.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.StaticFile("/openapi.yaml", "docs/openapi.yaml")

	v1 := r.Group("/api/v1")
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
	}

	user := v1.Group("/user")
	user.Use(middleware.AuthRequired(tokenManager))
	{
		user.GET("/me", userHandler.Me)
	}

	return r
}
