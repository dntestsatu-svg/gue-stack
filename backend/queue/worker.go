package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

type Worker struct {
	logger *slog.Logger
	server *asynq.Server
}

func NewWorker(redisAddr, redisPassword string, redisDB, concurrency int, logger *slog.Logger) *Worker {
	server := asynq.NewServer(asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	}, asynq.Config{Concurrency: concurrency})
	return &Worker{logger: logger, server: server}
}

func (w *Worker) Start() error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeSendWelcomeEmail, w.handleSendWelcomeEmail)
	if err := w.server.Start(mux); err != nil {
		return fmt.Errorf("start asynq worker: %w", err)
	}
	w.logger.Info("asynq worker started")
	return nil
}

func (w *Worker) Shutdown(ctx context.Context) {
	w.logger.Info("shutting down asynq worker")
	w.server.Shutdown()
	select {
	case <-ctx.Done():
		return
	default:
	}
}

func (w *Worker) handleSendWelcomeEmail(_ context.Context, t *asynq.Task) error {
	var payload WelcomeEmailPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}
	w.logger.Info("processed welcome email job", "email", payload.Email, "name", payload.Name)
	return nil
}
