package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/gue/backend/pkg/apperror"
	"github.com/example/gue/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RateLimitKeyFunc func(c *gin.Context) (string, error)

func SlidingWindowRateLimiter(
	client *redis.Client,
	keyPrefix string,
	limit int64,
	window time.Duration,
	keyFn RateLimitKeyFunc,
) gin.HandlerFunc {
	script := redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]

redis.call("ZREMRANGEBYSCORE", key, 0, now - window)
redis.call("ZADD", key, now, member)
local count = redis.call("ZCARD", key)
redis.call("PEXPIRE", key, window)

return count
`)

	return func(c *gin.Context) {
		if client == nil || limit <= 0 || window <= 0 || keyFn == nil {
			c.Next()
			return
		}

		keySuffix, err := keyFn(c)
		if err != nil {
			response.Error(c, apperror.New(http.StatusBadRequest, "invalid rate limit key", err.Error()))
			return
		}
		key := fmt.Sprintf("ratelimit:%s:%s", strings.TrimSpace(keyPrefix), keySuffix)

		nowMillis := time.Now().UnixMilli()
		member := fmt.Sprintf("%d-%s", nowMillis, uuid.NewString())
		result, evalErr := script.Run(
			c.Request.Context(),
			client,
			[]string{key},
			nowMillis,
			window.Milliseconds(),
			limit,
			member,
		).Result()
		if evalErr != nil {
			response.Error(c, apperror.New(http.StatusInternalServerError, "failed to evaluate rate limit", evalErr.Error()))
			return
		}

		count, convErr := toInt64(result)
		if convErr != nil {
			response.Error(c, apperror.New(http.StatusInternalServerError, "invalid rate limit response", convErr.Error()))
			return
		}
		if count > limit {
			c.Header("Retry-After", strconv.Itoa(int(window.Seconds())))
			response.Error(c, apperror.New(http.StatusTooManyRequests, "too many requests", nil))
			return
		}

		c.Next()
	}
}

func RateLimitKeyByIP(c *gin.Context) (string, error) {
	ip := strings.TrimSpace(c.ClientIP())
	if ip == "" {
		return "", fmt.Errorf("request ip is empty")
	}
	return ip, nil
}

func RateLimitKeyByTokoIDOrIP(c *gin.Context) (string, error) {
	if raw, ok := c.Get(ContextKeyTokoID); ok {
		if tokoID, ok := raw.(uint64); ok && tokoID > 0 {
			return fmt.Sprintf("toko:%d", tokoID), nil
		}
	}
	return RateLimitKeyByIP(c)
}

func toInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("unexpected redis result type %T", v)
	}
}
