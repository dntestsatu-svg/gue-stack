package cache

import (
	"context"
	"time"
)

type NoopCache struct{}

func NewNoopCache() *NoopCache {
	return &NoopCache{}
}

func (c *NoopCache) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, ErrCacheMiss
}

func (c *NoopCache) Set(_ context.Context, _ string, _ any, _ time.Duration) error {
	return nil
}

func (c *NoopCache) Delete(_ context.Context, _ string) error {
	return nil
}
