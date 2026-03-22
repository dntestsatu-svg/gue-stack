package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/example/gue/backend/cache"
	"github.com/example/gue/backend/config"
	httpHandler "github.com/example/gue/backend/handler/http"
	"github.com/example/gue/backend/pkg/db"
	jwtpkg "github.com/example/gue/backend/pkg/jwt"
	"github.com/example/gue/backend/pkg/paymentgateway"
	securitypkg "github.com/example/gue/backend/pkg/security"
	"github.com/example/gue/backend/queue"
	"github.com/example/gue/backend/repository/mysql"
	"github.com/example/gue/backend/repository/redisstore"
	"github.com/example/gue/backend/service"
	"github.com/redis/go-redis/v9"
)

type HTTPApp struct {
	Server        *http.Server
	DB            *sql.DB
	RedisClient   *redis.Client
	QueueProducer *queue.AsynqProducer
}

func NewHTTPApp(cfg config.Config, logger *slog.Logger) (*HTTPApp, error) {
	database, err := db.NewMySQL(cfg.Database)
	if err != nil {
		return nil, err
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	var queryCache cache.Cache
	switch cfg.Cache.Driver {
	case "none":
		queryCache = cache.NewNoopCache()
	default:
		queryCache = cache.NewMemcachedCache(cfg.Memcached.Addr)
	}

	userRepo := mysql.NewUserRepository(database)
	tokoRepo := mysql.NewTokoRepository(database)
	balanceRepo := mysql.NewBalanceRepository(database)
	bankRepo := mysql.NewBankRepository(database)
	paymentRepo := mysql.NewPaymentRepository(database)
	transactionRepo := mysql.NewTransactionRepository(database)
	refreshStore := redisstore.NewRefreshTokenStore(redisClient)
	tokenManager := jwtpkg.NewManager(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessTokenTTL,
		cfg.JWT.RefreshTokenTTL,
		cfg.JWT.Issuer,
		cfg.JWT.Audience,
	)

	producer := queue.NewAsynqProducer(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, logger)
	gatewayClient := paymentgateway.NewClient(cfg.PaymentGateway.BaseURL, cfg.PaymentGateway.Timeout)
	cookieManager := securitypkg.NewCookieManager(cfg.Security.Cookie, cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL)
	authSvc := service.NewAuthService(userRepo, refreshStore, tokenManager, producer, logger)
	userSvc := service.NewUserService(userRepo, queryCache, cfg.Cache.QueryCacheOn, cfg.Cache.UserMeTTL, cfg.Cache.DefaultTTL, producer, logger)
	tokoSvc := service.NewTokoService(tokoRepo, balanceRepo, queryCache, cfg.Cache.QueryCacheOn, cfg.Cache.DefaultTTL, 3, 3, logger)
	bankSvc := service.NewBankService(bankRepo, paymentRepo, queryCache, cfg.Cache.QueryCacheOn, cfg.Cache.DefaultTTL, logger)
	dashboardSvc := service.NewDashboardService(
		transactionRepo,
		gatewayClient,
		queryCache,
		cfg.PaymentGateway.MerchantUUID,
		cfg.PaymentGateway.DefaultClient,
		5*time.Minute,
	)
	paymentGatewaySvc := service.NewPaymentGatewayService(
		gatewayClient,
		tokoRepo,
		transactionRepo,
		producer,
		cfg.PaymentGateway.DefaultClient,
		cfg.PaymentGateway.DefaultKey,
		cfg.PaymentGateway.MerchantUUID,
		cfg.PaymentGateway.WebhookSecret,
		cfg.PaymentGateway.PlatformFeePercent,
		logger,
	)
	testingSvc := service.NewTestingService(
		tokoRepo,
		paymentGatewaySvc,
		nil,
		logger,
	)

	authHandler := httpHandler.NewAuthHandler(authSvc, cookieManager)
	userHandler := httpHandler.NewUserHandler(userSvc)
	tokoHandler := httpHandler.NewTokoHandler(tokoSvc)
	bankHandler := httpHandler.NewBankHandler(bankSvc)
	dashboardHandler := httpHandler.NewDashboardHandler(dashboardSvc)
	testingHandler := httpHandler.NewTestingHandler(testingSvc)
	paymentGatewayHandler := httpHandler.NewPaymentGatewayHandler(paymentGatewaySvc)
	router := httpHandler.NewRouter(
		logger,
		tokenManager,
		userRepo,
		tokoRepo,
		redisClient,
		cfg.Security,
		cookieManager,
		authHandler,
		userHandler,
		tokoHandler,
		bankHandler,
		dashboardHandler,
		testingHandler,
		paymentGatewayHandler,
	)

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	return &HTTPApp{
		Server:        server,
		DB:            database,
		RedisClient:   redisClient,
		QueueProducer: producer,
	}, nil
}

func (a *HTTPApp) Close() error {
	var errs []error
	if a.QueueProducer != nil {
		if err := a.QueueProducer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if a.RedisClient != nil {
		if err := a.RedisClient.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if a.DB != nil {
		if err := a.DB.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close resources: %v", errs)
	}
	return nil
}

func NewWorker(cfg config.Config, logger *slog.Logger) (*queue.Worker, *sql.DB, error) {
	database, err := db.NewMySQL(cfg.Database)
	if err != nil {
		return nil, nil, err
	}

	tokoRepo := mysql.NewTokoRepository(database)
	transactionRepo := mysql.NewTransactionRepository(database)
	gatewayClient := paymentgateway.NewClient(cfg.PaymentGateway.BaseURL, cfg.PaymentGateway.Timeout)
	callbackProcessor := service.NewPaymentGatewayService(
		gatewayClient,
		tokoRepo,
		transactionRepo,
		nil,
		cfg.PaymentGateway.DefaultClient,
		cfg.PaymentGateway.DefaultKey,
		cfg.PaymentGateway.MerchantUUID,
		cfg.PaymentGateway.WebhookSecret,
		cfg.PaymentGateway.PlatformFeePercent,
		logger,
	)

	worker := queue.NewWorker(
		cfg.Redis.Addr,
		cfg.Redis.Password,
		cfg.Redis.DB,
		cfg.Asynq.Concurrency,
		callbackProcessor,
		logger,
	)
	return worker, database, nil
}
