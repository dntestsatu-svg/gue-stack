package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
)

type Worker struct {
	logger            *slog.Logger
	server            *asynq.Server
	callbackProcessor CallbackProcessor
	expiryProcessor   PendingExpiryProcessor
	expiryStopCh      chan struct{}
	producerCloser    interface{ Close() error }
}

func NewWorker(redisAddr, redisPassword string, redisDB, concurrency int, callbackProcessor CallbackProcessor, producerCloser interface{ Close() error }, logger *slog.Logger) *Worker {
	server := asynq.NewServer(asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	}, asynq.Config{
		Concurrency: concurrency,
		Queues: map[string]int{
			"default":   3,
			"callbacks": 7,
		},
	})
	worker := &Worker{
		logger:            logger,
		server:            server,
		callbackProcessor: callbackProcessor,
		producerCloser:    producerCloser,
	}
	if processor, ok := callbackProcessor.(PendingExpiryProcessor); ok {
		worker.expiryProcessor = processor
	}
	return worker
}

func (w *Worker) Start() error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeSendWelcomeEmail, w.handleSendWelcomeEmail)
	if w.callbackProcessor != nil {
		mux.HandleFunc(TypeProcessQrisCallback, w.handleQrisCallback)
		mux.HandleFunc(TypeProcessTransferCallback, w.handleTransferCallback)
	}
	if err := w.server.Start(mux); err != nil {
		return fmt.Errorf("start asynq worker: %w", err)
	}
	if w.expiryProcessor != nil {
		w.expiryStopCh = make(chan struct{})
		go w.runPendingExpiryLoop()
	}
	w.logger.Info("asynq worker started")
	return nil
}

func (w *Worker) Shutdown(ctx context.Context) {
	w.logger.Info("shutting down asynq worker")
	if w.expiryStopCh != nil {
		close(w.expiryStopCh)
		w.expiryStopCh = nil
	}
	w.server.Shutdown()
	if w.producerCloser != nil {
		if err := w.producerCloser.Close(); err != nil {
			w.logger.Error("close asynq producer failed", "error", err.Error())
		}
	}
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

func (w *Worker) handleQrisCallback(ctx context.Context, t *asynq.Task) error {
	if w.callbackProcessor == nil {
		return fmt.Errorf("callback processor is not configured")
	}

	var payload QrisCallbackTaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal qris callback payload: %w", err)
	}

	if err := w.callbackProcessor.ProcessQrisCallback(ctx, payload); err != nil {
		return fmt.Errorf("process qris callback: %w", err)
	}
	w.logger.Info("processed qris callback job", "trx_id", payload.TrxID, "status", payload.Status)
	return nil
}

func (w *Worker) handleTransferCallback(ctx context.Context, t *asynq.Task) error {
	if w.callbackProcessor == nil {
		return fmt.Errorf("callback processor is not configured")
	}

	var payload TransferCallbackTaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal transfer callback payload: %w", err)
	}

	if err := w.callbackProcessor.ProcessTransferCallback(ctx, payload); err != nil {
		return fmt.Errorf("process transfer callback: %w", err)
	}
	w.logger.Info("processed transfer callback job", "partner_ref_no", payload.PartnerRefNo, "status", payload.Status)
	return nil
}

func (w *Worker) runPendingExpiryLoop() {
	w.processPendingExpiryBatch()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.processPendingExpiryBatch()
		case <-w.expiryStopCh:
			return
		}
	}
}

func (w *Worker) processPendingExpiryBatch() {
	if w.expiryProcessor == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	expiredBefore := time.Now().UTC().Add(-20 * time.Minute)
	count, err := w.expiryProcessor.ExpirePendingTransactions(ctx, expiredBefore, 250)
	if err != nil {
		w.logger.Error("expire pending transactions failed", "error", err.Error(), "older_than", expiredBefore.Format(time.RFC3339))
		return
	}
	if count > 0 {
		w.logger.Info("enqueued expired pending transactions", "count", count, "older_than", expiredBefore.Format(time.RFC3339))
	}
}
