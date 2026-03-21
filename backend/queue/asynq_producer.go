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

func (p *AsynqProducer) EnqueueQrisCallback(_ context.Context, payloadValue QrisCallbackTaskPayload) error {
	payload, err := NewQrisCallbackTaskPayload(payloadValue)
	if err != nil {
		return fmt.Errorf("marshal qris callback payload: %w", err)
	}
	task := asynq.NewTask(TypeProcessQrisCallback, payload)
	if _, err := p.client.Enqueue(task, asynq.MaxRetry(10), asynq.Queue("callbacks")); err != nil {
		return fmt.Errorf("enqueue qris callback task: %w", err)
	}
	p.logger.Info("enqueued qris callback", "trx_id", payloadValue.TrxID)
	return nil
}

func (p *AsynqProducer) EnqueueTransferCallback(_ context.Context, payloadValue TransferCallbackTaskPayload) error {
	payload, err := NewTransferCallbackTaskPayload(payloadValue)
	if err != nil {
		return fmt.Errorf("marshal transfer callback payload: %w", err)
	}
	task := asynq.NewTask(TypeProcessTransferCallback, payload)
	if _, err := p.client.Enqueue(task, asynq.MaxRetry(10), asynq.Queue("callbacks")); err != nil {
		return fmt.Errorf("enqueue transfer callback task: %w", err)
	}
	p.logger.Info("enqueued transfer callback", "partner_ref_no", payloadValue.PartnerRefNo)
	return nil
}
