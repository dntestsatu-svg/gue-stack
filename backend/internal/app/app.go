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
	case "memcached":
		if cfg.Memcached.Enabled {
			queryCache = cache.NewMemcachedCache(cfg.Memcached.Addr)
		} else {
			queryCache = cache.NewNoopCache()
		}
	case "none":
		queryCache = cache.NewNoopCache()
	default:
		queryCache = cache.NewRedisCache(redisClient)
	}

	userRepo := mysql.NewUserRepository(database)
	tokoRepo := mysql.NewTokoRepository(database)
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
	authSvc := service.NewAuthService(userRepo, refreshStore, tokenManager, producer, logger)
	userSvc := service.NewUserService(userRepo, queryCache, cfg.Cache.QueryCacheOn, cfg.Cache.UserMeTTL, logger)
	paymentGatewaySvc := service.NewPaymentGatewayService(
		gatewayClient,
		tokoRepo,
		transactionRepo,
		cfg.PaymentGateway.DefaultClient,
		cfg.PaymentGateway.DefaultKey,
		cfg.PaymentGateway.CallbackSecret,
		logger,
	)

	authHandler := httpHandler.NewAuthHandler(authSvc)
	userHandler := httpHandler.NewUserHandler(userSvc)
	paymentGatewayHandler := httpHandler.NewPaymentGatewayHandler(paymentGatewaySvc)
	router := httpHandler.NewRouter(logger, tokenManager, authHandler, userHandler, paymentGatewayHandler)

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

func NewWorker(cfg config.Config, logger *slog.Logger) *queue.Worker {
	return queue.NewWorker(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, cfg.Asynq.Concurrency, logger)
}
