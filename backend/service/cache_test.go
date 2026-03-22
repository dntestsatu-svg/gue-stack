package service

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/example/gue/backend/cache"
)

type fakeCacheStore struct {
	mu      sync.Mutex
	items   map[string][]byte
	sets    map[string]int
	deletes map[string]int
}

func newFakeCacheStore() *fakeCacheStore {
	return &fakeCacheStore{
		items:   map[string][]byte{},
		sets:    map[string]int{},
		deletes: map[string]int{},
	}
}

func (f *fakeCacheStore) Get(_ context.Context, key string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	payload, ok := f.items[key]
	if !ok {
		return nil, cache.ErrCacheMiss
	}
	copyPayload := append([]byte(nil), payload...)
	return copyPayload, nil
}

func (f *fakeCacheStore) Set(_ context.Context, key string, value any, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f.items[key] = payload
	f.sets[key]++
	return nil
}

func (f *fakeCacheStore) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.items, key)
	f.deletes[key]++
	return nil
}
