package main

import (
	"context"
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
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err.Error())
		os.Exit(1)
	}

	log := logger.New()
	worker := app.NewWorker(cfg, log)
	if err := worker.Start(); err != nil {
		log.Error("failed to start worker", "error", err.Error())
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Info("shutdown signal received", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	worker.Shutdown(ctx)
	log.Info("worker shutdown completed")
}
