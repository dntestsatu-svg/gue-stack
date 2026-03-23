package queue

import (
	"context"
	"time"
)

type Producer interface {
	EnqueueWelcomeEmail(ctx context.Context, email, name string) error
	EnqueueQrisCallback(ctx context.Context, payload QrisCallbackTaskPayload) error
	EnqueueTransferCallback(ctx context.Context, payload TransferCallbackTaskPayload) error
}

type CallbackProcessor interface {
	ProcessQrisCallback(ctx context.Context, payload QrisCallbackTaskPayload) error
	ProcessTransferCallback(ctx context.Context, payload TransferCallbackTaskPayload) error
}

type PendingExpiryProcessor interface {
	ExpirePendingTransactions(ctx context.Context, olderThan time.Time, limit int) (int, error)
}
