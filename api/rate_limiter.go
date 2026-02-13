package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kod2ulz/gostart/contracts"
)

// RateLimiter interface for rate limiting implementations
type RateLimiter interface {
	Allow(key string) (bool, error)
	Reset(key string) error
}

// MemoryRateLimiter implements in-memory rate limiting using token bucket algorithm
type MemoryRateLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*tokenBucket
	rate    int           // requests per window
	window  time.Duration // time window
}

type tokenBucket struct {
	tokens    int
	lastReset time.Time
}

// NewMemoryRateLimiter creates a new in-memory rate limiter
func NewMemoryRateLimiter(rate int, window time.Duration) *MemoryRateLimiter {
	limiter := &MemoryRateLimiter{
		buckets: make(map[string]*tokenBucket),
		rate:    rate,
		window:  window,
	}

	// Start cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// Allow checks if a request is allowed
func (m *MemoryRateLimiter) Allow(key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	bucket, exists := m.buckets[key]
	now := time.Now()

	if !exists || now.Sub(bucket.lastReset) >= m.window {
		m.buckets[key] = &tokenBucket{
			tokens:    m.rate - 1,
			lastReset: now,
		}
		return true, nil
	}

	if bucket.tokens > 0 {
		bucket.tokens--
		return true, nil
	}

	return false, nil
}

// Reset resets the rate limit for a key
func (m *MemoryRateLimiter) Reset(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.buckets, key)
	return nil
}

// cleanup removes expired buckets
func (m *MemoryRateLimiter) cleanup() {
	ticker := time.NewTicker(m.window)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for key, bucket := range m.buckets {
			if now.Sub(bucket.lastReset) >= m.window*2 {
				delete(m.buckets, key)
			}
		}
		m.mu.Unlock()
	}
}

// RedisRateLimiter implements distributed rate limiting using Redis
type RedisRateLimiter struct {
	client *redis.Client
	rate   int
	window time.Duration
	prefix string
}

// NewRedisRateLimiter creates a new Redis-based rate limiter
func NewRedisRateLimiter(client *redis.Client, rate int, window time.Duration, prefix string) *RedisRateLimiter {
	if prefix == "" {
		prefix = "ratelimit:"
	}
	return &RedisRateLimiter{
		client: client,
		rate:   rate,
		window: window,
		prefix: prefix,
	}
}

// Allow checks if a request is allowed using Redis
func (r *RedisRateLimiter) Allow(key string) (bool, error) {
	ctx := context.Background()
	redisKey := r.prefix + key

	// Increment counter
	count, err := r.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to increment counter: %w", err)
	}

	// Set expiration on first request
	if count == 1 {
		r.client.Expire(ctx, redisKey, r.window)
	}

	return count <= int64(r.rate), nil
}

// Reset resets the rate limit for a key
func (r *RedisRateLimiter) Reset(key string) error {
	ctx := context.Background()
	redisKey := r.prefix + key
	return r.client.Del(ctx, redisKey).Err()
}

// RateLimitMiddleware returns a middleware that rate limits requests
func RateLimitMiddleware(limiter RateLimiter, keyFunc func(contracts.RequestContext) string) func(contracts.RequestContext) {
	return func(ctx contracts.RequestContext) {
		key := keyFunc(ctx)

		allowed, err := limiter.Allow(key)
		if err != nil {
			ctx.JSON(500, map[string]interface{}{
				"error": "Internal server error",
			})
			ctx.Abort()
			return
		}

		if !allowed {
			ctx.JSON(429, map[string]interface{}{
				"error": "Rate limit exceeded",
			})
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}

// DefaultKeyFunc returns the client IP as the rate limit key
func DefaultKeyFunc(ctx contracts.RequestContext) string {
	return ctx.ClientIP()
}

// UserKeyFunc returns the user ID as the rate limit key
func UserKeyFunc(ctx contracts.RequestContext) string {
	userID, exists := ctx.Get("user_id")
	if exists {
		return fmt.Sprintf("user:%v", userID)
	}
	return ctx.ClientIP()
}

// PathKeyFunc returns the request path as the rate limit key
func PathKeyFunc(ctx contracts.RequestContext) string {
	return fmt.Sprintf("%s:%s", ctx.ClientIP(), ctx.Path())
}
