package queue

import "context"

type Producer interface {
	EnqueueWelcomeEmail(ctx context.Context, email, name string) error
}
