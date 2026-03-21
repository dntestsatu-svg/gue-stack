package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

type MemcachedCache struct {
	client *memcache.Client
}

func NewMemcachedCache(addr string) *MemcachedCache {
	return &MemcachedCache{client: memcache.New(addr)}
}

func (c *MemcachedCache) Get(_ context.Context, key string) ([]byte, error) {
	item, err := c.client.Get(key)
	if err != nil {
		if errors.Is(err, memcache.ErrCacheMiss) {
			return nil, ErrCacheMiss
		}
		return nil, err
	}
	return item.Value, nil
}

func (c *MemcachedCache) Set(_ context.Context, key string, value any, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	item := &memcache.Item{
		Key:        key,
		Value:      b,
		Expiration: int32(ttl.Seconds()),
	}
	return c.client.Set(item)
}

func (c *MemcachedCache) Delete(_ context.Context, key string) error {
	if err := c.client.Delete(key); err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
		return err
	}
	return nil
}
