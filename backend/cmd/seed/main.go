package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/example/gue/backend/config"
	"github.com/example/gue/backend/internal/seeder"
	"github.com/example/gue/backend/pkg/db"
	"github.com/example/gue/backend/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err.Error())
		os.Exit(1)
	}

	log := logger.New()
	gormDB, err := db.NewGormMySQL(cfg.Database)
	if err != nil {
		log.Error("failed to initialize gorm mysql", "error", err.Error())
		os.Exit(1)
	}

	if err := seeder.SeedPayments(context.Background(), gormDB); err != nil {
		log.Error("failed to seed payments", "error", err.Error())
		os.Exit(1)
	}

	log.Info("payments seeded successfully")
}
