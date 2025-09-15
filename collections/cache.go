package collections

import (
	"context"
	"fmt"
	"time"
)

var (
	// DefaultCacheKeyTTL is the default time-to-live for a cache item.
	DefaultCacheKeyTTL = 15 * time.Minute
	// DefaultEvictionInterval is the default interval for the background eviction process.
	DefaultEvictionInterval = 5 * time.Minute
)

// Cache is the interface for a generic, thread-safe in-memory cache.
type Cache[K comparable, T any, E error] interface {
	Get(ctx context.Context, key K) (*T, E)
	Fetch(ctx context.Context, keys ...K) (List[T], E)
	Load(ctx context.Context, keys ...K) E
	Clear(keys ...K)
	ClearAll()
	GetTTL(key K) time.Duration
	SetTTL(key K, ttl time.Duration)
	Set(key K, value T)
	Stop()
}

// cacheInitialiser is a function type for configuring the memoryCache.
type cacheInitialiser[K comparable, T CacheModel[K, T], E error] func(c *memoryCache[K, T, E])

// fetchCacheKeysFromStorageFunc defines the signature for the function that loads data from the persistent store.
type fetchCacheKeysFromStorageFunc[K comparable, T CacheModel[K, T], E error] func(ctx context.Context, keys []K) ([]T, E)

// WithStalePermission configures whether the cache can return stale (expired) data if a fetch fails.
func WithStalePermission[K comparable, T CacheModel[K, T], E error](allowStale bool) cacheInitialiser[K, T, E] {
	return func(c *memoryCache[K, T, E]) {
		c.allowStale = allowStale
	}
}

// WithDefaultTTL sets the default time-to-live for cache items.
func WithDefaultTTL[K comparable, T CacheModel[K, T], E error](ttl time.Duration) cacheInitialiser[K, T, E] {
	return func(c *memoryCache[K, T, E]) {
		c.ttl = ttl
	}
}

// WithEvictionInterval sets the interval for the background eviction process.
func WithEvictionInterval[K comparable, T CacheModel[K, T], E error](interval time.Duration) cacheInitialiser[K, T, E] {
	return func(c *memoryCache[K, T, E]) {
		c.evictionInterval = interval
	}
}

// WithTTLDelimiter sets the delimiter for TTL prefix matching.
func WithTTLDelimiter[K comparable, T CacheModel[K, T], E error](delimiter string) cacheInitialiser[K, T, E] {
	return func(c *memoryCache[K, T, E]) {
		c.delimiter = delimiter
	}
}

// WithFetcherFunc sets the function used to fetch data from the persistent store.
func WithFetcherFunc[K comparable, T CacheModel[K, T], E error](fetchFunc fetchCacheKeysFromStorageFunc[K, T, E]) cacheInitialiser[K, T, E] {
	return func(c *memoryCache[K, T, E]) {
		c.fetcher = fetchFunc
	}
}

// CacheModel is an interface that cached items must implement to provide their own key.
type CacheModel[K comparable, T any] interface {
	Key() K
}

// cacheObject wraps the cached data with its expiration time.
type cacheObject[K comparable, T CacheModel[K, T]] struct {
	expiry time.Time
	data   *T
}

// valid checks if the cache object has expired.
func (o *cacheObject[K, T]) valid() bool {
	return o.expiry.After(time.Now())
}

// NewMemoryCache creates a new thread-safe, in-memory cache.
func NewMemoryCache[K comparable, T CacheModel[K, T], E error](opts ...cacheInitialiser[K, T, E]) (out *memoryCache[K, T, E]) {
	out = &memoryCache[K, T, E]{
		data:             NewConcurrentMap[K, cacheObject[K, T]](),
		ttl:              DefaultCacheKeyTTL,
		evictionInterval: DefaultEvictionInterval,
		allowStale:       true,
		stopChan:         make(chan bool),
	}
	for _, opt := range opts {
		opt(out)
	}
	if out.fetcher == nil {
		var t = new(T)
		panic(fmt.Sprintf("cache is unusable without a way to load %T objects into cache. Please initialise with option NewMemoryCache(WithFetcherFunc(<func>))", t))
	}
	if out.delimiter == "" {
		out.delimiter = "."
	}
	out.ttls = NewPrefixMapper(out.delimiter, out.ttl)
	out.start() // Start the background eviction process
	return
}

// memoryCache is the concrete implementation of the Cache interface.
type memoryCache[K comparable, T CacheModel[K, T], E error] struct {
	allowStale       bool
	ttls             Prefixes[time.Duration]
	data             ConcurrentMap[K, cacheObject[K, T]]
	fetcher          fetchCacheKeysFromStorageFunc[K, T, E]
	ttl              time.Duration
	evictionInterval time.Duration
	delimiter        string
	stopChan         chan bool
}

// start launches the background goroutine for periodic cache eviction.
func (c *memoryCache[K, T, E]) start() {
	ticker := time.NewTicker(c.evictionInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				c.sweep()
			case <-c.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop terminates the background eviction goroutine.
func (c *memoryCache[K, T, E]) Stop() {
	c.stopChan <- true
}

// sweep iterates through the cache and removes expired items.
func (c *memoryCache[K, T, E]) sweep() {
	var expiredKeys []K
	keys := c.data.Keys()
	for _, key := range keys {
		if val := c.data.Get(key); val != nil && !val.valid() {
			expiredKeys = append(expiredKeys, key)
		}
	}
	if len(expiredKeys) > 0 {
		c.data.Clear(expiredKeys...)
	}
}

func (c *memoryCache[K, T, E]) GetTTL(key K) time.Duration {
	switch k := any(key).(type) {
	case string:
		return c.ttls.Get(k)
	case int, int8, int16, int32, int64, float32, float64:
		return c.ttls.Get(fmt.Sprint(k))
	default:
		if c.ttl != 0 {
			return c.ttl
		}
		return DefaultCacheKeyTTL
	}
}

// absent checks a list of keys and returns those that are not in the cache or have expired.
func (c *memoryCache[K, T, E]) absent(keys ...K) (out []K) {
	if len(keys) == 0 {
		return []K{}
	}
	out = make([]K, 0, len(keys))
	for _, key := range keys {
		if t := c.data.Get(key); t == nil || !t.valid() {
			out = append(out, key)
		}
	}
	return
}

func (c *memoryCache[K, T, E]) Get(ctx context.Context, key K) (*T, E) {
	if item := c.data.Get(key); item != nil && item.valid() {
		return item.data, c.zeroErr()
	}

	results, err := c.fetcher(ctx, []K{key})
	if any(err) != nil {
		if c.allowStale {
			if item := c.data.Get(key); item != nil {
				return item.data, c.zeroErr()
			}
		}
		return nil, err
	}

	if len(results) == 0 {
		return nil, c.zeroErr()
	}

	item := results[0]
	c.Set(item.Key(), item)
	return &item, c.zeroErr()
}

// Clear removes specific items from the cache.
func (c *memoryCache[K, T, E]) Clear(keys ...K) {
	if len(keys) > 0 {
		c.data.Clear(keys...)
	}
}

// ClearAll removes all items from the cache.
func (c *memoryCache[K, T, E]) ClearAll() {
	c.data.Clear()
}

// Set manually adds or overwrites an item in the cache.
func (c *memoryCache[K, T, E]) Set(key K, value T) {
	c.data.Add(key, cacheObject[K, T]{
		expiry: time.Now().Add(c.GetTTL(key)),
		data:   &value,
	})
}

func (c *memoryCache[K, T, E]) SetTTL(key K, ttl time.Duration) {
	switch k := any(key).(type) {
	case string:
		c.ttls.Set(k, ttl)
	case int, int8, int16, int32, int64, float32, float64:
		c.ttls.Set(fmt.Sprint(k), ttl)
	}
}

func (c *memoryCache[K, T, E]) Fetch(ctx context.Context, keys ...K) (List[T], E) {
	if len(keys) == 0 {
		return List[T]{}, c.zeroErr()
	}

	toFetch := c.absent(keys...)
	if len(toFetch) > 0 {
		res, err := c.fetcher(ctx, toFetch)
		if any(err) != nil {
			return nil, err
		}
		for _, item := range res {
			c.Set(item.Key(), item)
		}
	}

	out := make(List[T], 0, len(keys))
	for _, key := range keys {
		if item := c.data.Get(key); item != nil {
			out = append(out, *item.data)
		}
	}
	return out, c.zeroErr()
}

func (c *memoryCache[K, T, E]) Load(ctx context.Context, keys ...K) E {
	_, err := c.Fetch(ctx, keys...)
	return err
}

func (c *memoryCache[K, T, E]) zeroErr() E {
	var zero E
	return zero
}