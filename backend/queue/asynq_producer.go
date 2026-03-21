package queue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

type AsynqProducer struct {
	client *asynq.Client
	logger *slog.Logger
}

func NewAsynqProducer(redisAddr, redisPassword string, redisDB int, logger *slog.Logger) *AsynqProducer {
	return &AsynqProducer{
		client: asynq.NewClient(asynq.RedisClientOpt{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       redisDB,
		}),
		logger: logger,
	}
}

func (p *AsynqProducer) Close() error {
	return p.client.Close()
}

func (p *AsynqProducer) EnqueueWelcomeEmail(_ context.Context, email, name string) error {
	payload, err := NewWelcomeEmailPayload(email, name)
	if err != nil {
		return fmt.Errorf("marshal welcome email payload: %w", err)
	}
	task := asynq.NewTask(TypeSendWelcomeEmail, payload)
	if _, err := p.client.Enqueue(task, asynq.MaxRetry(5), asynq.Queue("default")); err != nil {
		return fmt.Errorf("enqueue welcome email task: %w", err)
	}
	p.logger.Info("enqueued welcome email", "email", email)
	return nil
}
