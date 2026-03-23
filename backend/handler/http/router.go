package http

import (
	"log/slog"
	"net/http"

	"github.com/example/gue/backend/config"
	"github.com/example/gue/backend/middleware"
	"github.com/example/gue/backend/model"
	jwtpkg "github.com/example/gue/backend/pkg/jwt"
	securitypkg "github.com/example/gue/backend/pkg/security"
	"github.com/example/gue/backend/repository"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func NewRouter(
	logger *slog.Logger,
	tokenManager *jwtpkg.Manager,
	userRepo repository.UserRepository,
	tokoRepo repository.TokoRepository,
	redisClient *redis.Client,
	securityCfg config.SecurityConfig,
	cookieManager *securitypkg.CookieManager,
	authHandler *AuthHandler,
	userHandler *UserHandler,
	tokoHandler *TokoHandler,
	bankHandler *BankHandler,
	withdrawHandler *WithdrawHandler,
	dashboardHandler *DashboardHandler,
	testingHandler *TestingHandler,
	paymentGatewayHandler *PaymentGatewayHandler,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(
		middleware.RequestID(),
		middleware.SecurityHeaders(),
		middleware.CORS(),
		middleware.Logger(logger),
		middleware.Recovery(),
	)

	csrfMiddleware := middleware.CSRFProtection(
		securityCfg.CSRF,
		cookieManager,
		"/api/v1/payments/gateway/callback",
	)

	authLoginLimiter := passThrough()
	authRegisterLimiter := passThrough()
	paymentLimiter := passThrough()
	if securityCfg.RateLimit.Enabled && redisClient != nil {
		authLoginLimiter = middleware.SlidingWindowRateLimiter(
			redisClient,
			"auth:login",
			securityCfg.RateLimit.AuthLogin,
			securityCfg.RateLimit.Window,
			middleware.RateLimitKeyByIP,
		)
		authRegisterLimiter = middleware.SlidingWindowRateLimiter(
			redisClient,
			"auth:register",
			securityCfg.RateLimit.AuthRegister,
			securityCfg.RateLimit.Window,
			middleware.RateLimitKeyByIP,
		)
		paymentLimiter = middleware.SlidingWindowRateLimiter(
			redisClient,
			"payments:bridge",
			securityCfg.RateLimit.PaymentBridge,
			securityCfg.RateLimit.Window,
			middleware.RateLimitKeyByTokoIDOrIP,
		)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.StaticFile("/openapi.yaml", "docs/openapi.yaml")

	v1 := r.Group("/api/v1")
	auth := v1.Group("/auth")
	{
		auth.GET("/csrf", authHandler.CSRF)
		auth.GET("/session", authHandler.Session)
		auth.POST("/register", authRegisterLimiter, csrfMiddleware, authHandler.Register)
		auth.POST("/login", authLoginLimiter, csrfMiddleware, authHandler.Login)
		auth.POST("/refresh", csrfMiddleware, authHandler.Refresh)
		auth.POST("/logout", csrfMiddleware, authHandler.Logout)
	}

	user := v1.Group("/user")
	user.Use(middleware.AuthRequired(tokenManager, userRepo, cookieManager))
	{
		user.GET("/me", userHandler.Me)
		user.PATCH("/password", csrfMiddleware, userHandler.ChangePassword)
	}

	dashboard := v1.Group("/dashboard")
	dashboard.Use(middleware.AuthRequired(tokenManager, userRepo, cookieManager))
	{
		dashboard.GET("/overview", dashboardHandler.Overview)
	}

	testing := v1.Group("/testing")
	testing.Use(middleware.AuthRequired(tokenManager, userRepo, cookieManager))
	{
		testing.POST("/generate-qris", csrfMiddleware, testingHandler.GenerateQris)
		testing.POST("/callback-readiness", csrfMiddleware, testingHandler.CheckCallbackReadiness)
	}

	transactions := v1.Group("/transactions")
	transactions.Use(middleware.AuthRequired(tokenManager, userRepo, cookieManager))
	{
		transactions.GET("/history", dashboardHandler.TransactionHistory)
		transactions.GET("/history/export", dashboardHandler.ExportTransactionHistory)
	}

	users := v1.Group("/users")
	users.Use(middleware.AuthRequired(tokenManager, userRepo, cookieManager))
	{
		users.GET("", middleware.RoleRequired(model.UserRoleDev, model.UserRoleSuperAdmin, model.UserRoleAdmin), userHandler.List)
		users.POST("", csrfMiddleware, middleware.RoleRequired(model.UserRoleDev, model.UserRoleSuperAdmin, model.UserRoleAdmin), userHandler.Create)
		users.PATCH("/:id/role", csrfMiddleware, middleware.RoleRequired(model.UserRoleDev, model.UserRoleSuperAdmin), userHandler.UpdateRole)
		users.PATCH("/:id/active", csrfMiddleware, middleware.RoleRequired(model.UserRoleDev, model.UserRoleSuperAdmin, model.UserRoleAdmin), userHandler.UpdateActive)
		users.DELETE("/:id", csrfMiddleware, middleware.RoleRequired(model.UserRoleDev, model.UserRoleSuperAdmin, model.UserRoleAdmin), userHandler.Delete)
	}

	banks := v1.Group("/banks")
	banks.Use(
		middleware.AuthRequired(tokenManager, userRepo, cookieManager),
		middleware.RoleRequired(model.UserRoleDev, model.UserRoleSuperAdmin, model.UserRoleAdmin),
	)
	{
		banks.GET("", bankHandler.List)
		banks.GET("/payment-options", bankHandler.PaymentOptions)
		banks.POST("/inquiry", csrfMiddleware, bankHandler.Inquiry)
		banks.POST("", csrfMiddleware, bankHandler.Create)
		banks.DELETE("/:id", csrfMiddleware, bankHandler.Delete)
	}

	withdraw := v1.Group("/withdraw")
	withdraw.Use(
		middleware.AuthRequired(tokenManager, userRepo, cookieManager),
		middleware.RoleRequired(model.UserRoleDev, model.UserRoleSuperAdmin, model.UserRoleAdmin),
	)
	{
		withdraw.GET("/options", withdrawHandler.Options)
		withdraw.GET("/history", withdrawHandler.History)
		withdraw.POST("/inquiry", csrfMiddleware, withdrawHandler.Inquiry)
		withdraw.POST("/transfer", csrfMiddleware, withdrawHandler.Transfer)
	}

	tokos := v1.Group("/tokos")
	tokos.Use(middleware.AuthRequired(tokenManager, userRepo, cookieManager))
	{
		tokos.GET("/workspace", tokoHandler.Workspace)
		tokos.GET("", tokoHandler.List)
		tokos.POST("", csrfMiddleware, middleware.RoleRequired(model.UserRoleDev, model.UserRoleSuperAdmin, model.UserRoleAdmin), tokoHandler.Create)
		tokos.PATCH("/:id", csrfMiddleware, middleware.RoleRequired(model.UserRoleDev, model.UserRoleSuperAdmin, model.UserRoleAdmin), tokoHandler.Update)
		tokos.POST("/:id/regenerate-token", csrfMiddleware, middleware.RoleRequired(model.UserRoleDev, model.UserRoleSuperAdmin, model.UserRoleAdmin), tokoHandler.RegenerateToken)
		tokos.GET("/balances", tokoHandler.ListBalances)
		tokos.PATCH("/:id/settlement", csrfMiddleware, middleware.RoleRequired(model.UserRoleDev), tokoHandler.ManualSettlement)
	}

	paymentGateway := v1.Group("/payments/gateway")
	paymentGateway.Use(middleware.TokoTokenRequired(tokoRepo), paymentLimiter, csrfMiddleware)
	{
		paymentGateway.POST("/generate", paymentGatewayHandler.Generate)
		paymentGateway.POST("/check-status/:trx_id", paymentGatewayHandler.CheckStatusV2)
		paymentGateway.POST("/inquiry", paymentGatewayHandler.InquiryTransfer)
		paymentGateway.POST("/transfer", paymentGatewayHandler.TransferFund)
		paymentGateway.POST("/transfer/check-status/:partner_ref_no", paymentGatewayHandler.CheckTransferStatus)
		paymentGateway.POST("/balance", paymentGatewayHandler.GetBalance)
	}

	callbacks := v1.Group("/payments/gateway/callback")
	{
		callbacks.POST("/qris", paymentGatewayHandler.QrisCallback)
		callbacks.POST("/transfer", paymentGatewayHandler.TransferCallback)
	}

	return r
}

func passThrough() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
