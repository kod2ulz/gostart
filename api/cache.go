package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kod2ulz/gostart/contracts"
)

// CacheStore interface for caching implementations
type CacheStore interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte, ttl time.Duration) error
	Delete(key string) error
	Clear() error
}

// MemoryCacheStore implements in-memory caching
type MemoryCacheStore struct {
	cache map[string]*cacheEntry
	mu    *sync.RWMutex
}

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// NewMemoryCacheStore creates a new in-memory cache store
func NewMemoryCacheStore() *MemoryCacheStore {
	store := &MemoryCacheStore{
		cache: make(map[string]*cacheEntry),
		mu:    &sync.RWMutex{},
	}

	// Start cleanup goroutine
	go store.cleanup()

	return store
}

// Get retrieves a value from the cache
func (m *MemoryCacheStore) Get(key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.cache[key]
	if !exists {
		return nil, fmt.Errorf("cache miss")
	}

	if time.Now().After(entry.expiresAt) {
		delete(m.cache, key)
		return nil, fmt.Errorf("cache expired")
	}

	return entry.value, nil
}

// Set stores a value in the cache
func (m *MemoryCacheStore) Set(key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cache[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Delete removes a value from the cache
func (m *MemoryCacheStore) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cache, key)
	return nil
}

// Clear removes all values from the cache
func (m *MemoryCacheStore) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache = make(map[string]*cacheEntry)
	return nil
}

// cleanup removes expired entries
func (m *MemoryCacheStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for key, entry := range m.cache {
			if now.After(entry.expiresAt) {
				delete(m.cache, key)
			}
		}
		m.mu.Unlock()
	}
}

// RedisCacheStore implements Redis-based caching
type RedisCacheStore struct {
	client *redis.Client
	prefix string
}

// NewRedisCacheStore creates a new Redis cache store
func NewRedisCacheStore(client *redis.Client, prefix string) *RedisCacheStore {
	if prefix == "" {
		prefix = "cache:"
	}
	return &RedisCacheStore{
		client: client,
		prefix: prefix,
	}
}

// Get retrieves a value from Redis
func (r *RedisCacheStore) Get(key string) ([]byte, error) {
	ctx := context.Background()
	redisKey := r.prefix + key

	data, err := r.client.Get(ctx, redisKey).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("cache miss")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get from cache: %w", err)
	}

	return data, nil
}

// Set stores a value in Redis
func (r *RedisCacheStore) Set(key string, value []byte, ttl time.Duration) error {
	ctx := context.Background()
	redisKey := r.prefix + key

	err := r.client.Set(ctx, redisKey, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set in cache: %w", err)
	}

	return nil
}

// Delete removes a value from Redis
func (r *RedisCacheStore) Delete(key string) error {
	ctx := context.Background()
	redisKey := r.prefix + key
	return r.client.Del(ctx, redisKey).Err()
}

// Clear removes all cached values
func (r *RedisCacheStore) Clear() error {
	ctx := context.Background()
	iter := r.client.Scan(ctx, 0, r.prefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		r.client.Del(ctx, iter.Val())
	}
	return iter.Err()
}

// CacheConfig configures response caching
type CacheConfig struct {
	Store      CacheStore
	TTL        time.Duration
	KeyFunc    func(contracts.RequestContext) string
	ShouldCache func(contracts.RequestContext) bool
}

// CacheMiddleware returns a middleware that caches responses
func CacheMiddleware(config CacheConfig) func(contracts.RequestContext) {
	if config.TTL == 0 {
		config.TTL = 5 * time.Minute
	}

	if config.KeyFunc == nil {
		config.KeyFunc = DefaultCacheKeyFunc
	}

	if config.ShouldCache == nil {
		config.ShouldCache = func(ctx contracts.RequestContext) bool {
			return ctx.Method() == http.MethodGet
		}
	}

	return func(ctx contracts.RequestContext) {
		// Only cache if configured to do so
		if !config.ShouldCache(ctx) {
			ctx.Next()
			return
		}

		key := config.KeyFunc(ctx)

		// Try to get from cache
		cached, err := config.Store.Get(key)
		if err == nil {
			// Cache hit - return cached response
			var cachedResp cachedResponse
			if err := json.Unmarshal(cached, &cachedResp); err == nil {
				ctx.Header("X-Cache", "HIT")
				ctx.Header("X-Cache-Key", key)
				ctx.Data(cachedResp.StatusCode, cachedResp.ContentType, cachedResp.Body)
				return
			}
		}

		// Cache miss - proceed with request and cache the response
		ctx.Header("X-Cache", "MISS")
		ctx.Header("X-Cache-Key", key)

		// Capture response
		writer := &responseWriter{
			ResponseWriter: ctx.Writer(),
			body:           &bytes.Buffer{},
		}
		ctx.SetWriter(writer)

		ctx.Next()

		// Cache the response if successful
		if writer.statusCode >= 200 && writer.statusCode < 300 {
			cachedResp := cachedResponse{
				StatusCode:  writer.statusCode,
				ContentType: writer.Header().Get("Content-Type"),
				Body:        writer.body.Bytes(),
			}

			if data, err := json.Marshal(cachedResp); err == nil {
				config.Store.Set(key, data, config.TTL)
			}
		}
	}
}

// cachedResponse represents a cached HTTP response
type cachedResponse struct {
	StatusCode  int    `json:"status_code"`
	ContentType string `json:"content_type"`
	Body        []byte `json:"body"`
}

// responseWriter wraps http.ResponseWriter to capture the response
type responseWriter struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (w *responseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// DefaultCacheKeyFunc generates a cache key from the request path and query
func DefaultCacheKeyFunc(ctx contracts.RequestContext) string {
	path := ctx.Path()
	query := ctx.Request().URL.RawQuery

	if query != "" {
		path = path + "?" + query
	}

	// Generate hash for the key
	hash := sha256.Sum256([]byte(path))
	return fmt.Sprintf("%x", hash)
}

// PathCacheKeyFunc generates a cache key from just the request path
func PathCacheKeyFunc(ctx contracts.RequestContext) string {
	hash := sha256.Sum256([]byte(ctx.Path()))
	return fmt.Sprintf("%x", hash)
}

// UserPathCacheKeyFunc generates a cache key from user ID and path
func UserPathCacheKeyFunc(ctx contracts.RequestContext) string {
	userID, _ := ctx.Get("user_id")
	key := fmt.Sprintf("%v:%s", userID, ctx.Path())
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash)
}

// CacheInvalidator provides methods to invalidate cached responses
type CacheInvalidator struct {
	store CacheStore
}

// NewCacheInvalidator creates a new cache invalidator
func NewCacheInvalidator(store CacheStore) *CacheInvalidator {
	return &CacheInvalidator{store: store}
}

// Invalidate removes a specific cache entry
func (c *CacheInvalidator) Invalidate(key string) error {
	return c.store.Delete(key)
}

// InvalidateAll removes all cache entries
func (c *CacheInvalidator) InvalidateAll() error {
	return c.store.Clear()
}

// InvalidatePattern removes cache entries matching a pattern (Redis only)
func (c *CacheInvalidator) InvalidatePattern(pattern string) error {
	if redisStore, ok := c.store.(*RedisCacheStore); ok {
		ctx := context.Background()
		iter := redisStore.client.Scan(ctx, 0, redisStore.prefix+pattern+"*", 0).Iterator()
		for iter.Next(ctx) {
			redisStore.client.Del(ctx, iter.Val())
		}
		return iter.Err()
	}
	return fmt.Errorf("pattern invalidation only supported for Redis")
}
