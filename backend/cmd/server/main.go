package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/gue/backend/config"
	"github.com/example/gue/backend/internal/app"
	"github.com/example/gue/backend/pkg/logger"
)

func main() {
	workerMode := flag.Bool("worker", false, "run as queue worker")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err.Error())
		os.Exit(1)
	}

	log := logger.New()
	if *workerMode {
		runWorker(cfg, log)
		return
	}
	runServer(cfg, log)
}

func runServer(cfg config.Config, log *slog.Logger) {
	httpApp, err := app.NewHTTPApp(cfg, log)
	if err != nil {
		log.Error("failed to initialize http app", "error", err.Error())
		os.Exit(1)
	}
	defer func() {
		if err := httpApp.Close(); err != nil {
			log.Error("failed to close resources", "error", err.Error())
		}
	}()

	go func() {
		log.Info("http server started", "port", cfg.Server.Port)
		if serveErr := httpApp.Server.ListenAndServe(); serveErr != nil && serveErr.Error() != "http: Server closed" {
			log.Error("http server failed", "error", serveErr.Error())
			os.Exit(1)
		}
	}()

	waitForShutdown(cfg, log, func(ctx context.Context) error {
		return httpApp.Server.Shutdown(ctx)
	})
}

func runWorker(cfg config.Config, log *slog.Logger) {
	worker := app.NewWorker(cfg, log)
	if err := worker.Start(); err != nil {
		log.Error("failed to start worker", "error", err.Error())
		os.Exit(1)
	}

	waitForShutdown(cfg, log, func(ctx context.Context) error {
		worker.Shutdown(ctx)
		return nil
	})
}

func waitForShutdown(_ config.Config, log *slog.Logger, shutdownFn func(ctx context.Context) error) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	log.Info("shutdown signal received", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := shutdownFn(ctx); err != nil {
		log.Error("graceful shutdown failed", "error", err.Error())
		os.Exit(1)
	}
	log.Info("shutdown completed")
}
