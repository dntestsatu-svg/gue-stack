package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/example/gue/backend/cache"
)

const listCacheNamespaceTTL = 24 * time.Hour

func getCachedJSON[T any](ctx context.Context, store cache.Cache, enabled bool, key string, logger *slog.Logger) (*T, bool) {
	if !enabled || store == nil {
		return nil, false
	}

	payload, err := store.Get(ctx, key)
	if err != nil {
		if !errors.Is(err, cache.ErrCacheMiss) && logger != nil {
			logger.Error("cache get failed", "key", key, "error", err.Error())
		}
		return nil, false
	}

	var result T
	if err := json.Unmarshal(payload, &result); err != nil {
		if logger != nil {
			logger.Error("cache unmarshal failed", "key", key, "error", err.Error())
		}
		return nil, false
	}

	return &result, true
}

func setCachedJSON(ctx context.Context, store cache.Cache, enabled bool, key string, value any, ttl time.Duration, logger *slog.Logger) {
	if !enabled || store == nil || value == nil {
		return
	}
	if err := store.Set(ctx, key, value, ttl); err != nil && logger != nil {
		logger.Error("cache set failed", "key", key, "error", err.Error())
	}
}

func getCacheNamespaceToken(ctx context.Context, store cache.Cache, enabled bool, namespaceKey string, logger *slog.Logger) string {
	if !enabled || store == nil {
		return "disabled"
	}

	payload, err := store.Get(ctx, namespaceKey)
	if err == nil {
		var token string
		unmarshalErr := json.Unmarshal(payload, &token)
		if unmarshalErr == nil && strings.TrimSpace(token) != "" {
			return token
		}
		if unmarshalErr != nil && logger != nil {
			logger.Error("cache namespace unmarshal failed", "key", namespaceKey, "error", unmarshalErr.Error())
		}
	} else if !errors.Is(err, cache.ErrCacheMiss) && logger != nil {
		logger.Error("cache namespace get failed", "key", namespaceKey, "error", err.Error())
	}

	token := "v1"
	setCachedJSON(ctx, store, enabled, namespaceKey, token, listCacheNamespaceTTL, logger)
	return token
}

func bumpCacheNamespace(ctx context.Context, store cache.Cache, enabled bool, namespaceKey string, logger *slog.Logger) {
	if !enabled || store == nil {
		return
	}
	token := strconv.FormatInt(time.Now().UTC().UnixNano(), 36)
	setCachedJSON(ctx, store, enabled, namespaceKey, token, listCacheNamespaceTTL, logger)
}

func buildHashedCacheKey(prefix string, parts ...string) string {
	digest := sha1.Sum([]byte(strings.Join(parts, "|")))
	return prefix + ":" + hex.EncodeToString(digest[:])
}
