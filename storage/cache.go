package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/kod2ulz/gostart/collections"
	"github.com/kod2ulz/gostart/logr"
)

var (
	CacheKeyTTL = 15 * time.Second
)

type Cache[K comparable, T any, E error] interface {
	// Get attempts to retrieve an object from the cache. will return nil if it's not found
	Get(key K) *T
	// Fetches objects matching keys from the backend storage into the cache, if they are not in the cache and not expired
	Fetch(ctx context.Context, keys ...K) (collections.List[T], E)
	// Loads objects matching the keys into the cache. It will load all if keys are not specfied
	Load(ctx context.Context, keys ...K) E
}

// type cacheInitialiser[K comparable, T any, E error] func(*memoryCache[K, T, cacheObject[K, T], E])
type cacheInitialiser[K comparable, T any, E error] func(Cache[K, T, E])

type fetchCacheKeysFromStorageFunc[K comparable, T any, E error] func(ctx context.Context, keys []K) ([]T, E)

func WithStalePermission[K comparable, T CacheModel[K, T], E error](keepStaleData bool) cacheInitialiser[K, T, E] {
	return func(c Cache[K, T, E]) {
		switch c := c.(type) {
		case *memoryCache[K, T, E]:
			c.allowStale = keepStaleData
		default:
			panic("unsupported cache type")
		}
	}
}
func WithDefaultTTL[K comparable, T CacheModel[K, T], E error](ttl time.Duration) cacheInitialiser[K, T, E] {
	return func(c Cache[K, T, E]) {
		switch c := c.(type) {
		case *memoryCache[K, T, E]:
			c.ttl = ttl
		default:
			panic("unsupported cache type")
		}
	}
}

func WithTTLDelimiter[K comparable, T CacheModel[K, T], E error](delimeter string) cacheInitialiser[K, T, E] {
	return func(c Cache[K, T, E]) {
		switch c := c.(type) {
		case *memoryCache[K, T, E]:
			c.delimiter = delimeter
		default:
			panic("unsupported cache type")
		}
	}
}

func WithDataFunc[K comparable, T CacheModel[K, T], E error](fetchFunc fetchCacheKeysFromStorageFunc[K, T, E]) cacheInitialiser[K, T, E] {
	return func(c Cache[K, T, E]) {
		switch c := c.(type) {
		case *memoryCache[K, T, E]:
			c.fetchFn = fetchFunc
		default:
			panic("unsupported cache type")
		}
	}
}

type CacheModel[K comparable, T any] interface{ Key() K }

type cacheObject[K comparable, T CacheModel[K, T]] struct {
	// refreshed time.Time
	expiry time.Time
	data   *T
}

func (o *cacheObject[K, T]) valid() bool { return o.expiry.After(time.Now()) }

func NewMemoryCache[K comparable, T CacheModel[K, T], E error](log *logr.Logger, opts ...cacheInitialiser[K, T, E]) (out *memoryCache[K, T, E]) {
	out = &memoryCache[K, T, E]{
		data:       collections.NewConcurrentMap[K, cacheObject[K, T]](),
		ttl:        CacheKeyTTL,
		allowStale: true,
	}
	for _, opt := range opts {
		opt(out)
	}
	if out.fetchFn == nil {
		var t = new(T)
		log.Fatalf("cache is unusable without a way to load %T objects into cache. Please initialise with option NewMemoryCache(WithDataFunc(<func>))", t)
	}
	if out.delimiter == "" {
		out.delimiter = "."
	}
	out.ttls = NewPrefixMapper(out.delimiter, out.ttl)
	return
}

// type memoryCache[K comparable, T cacheObject[K, T], E error] struct {
type memoryCache[K comparable, T CacheModel[K, T], E error] struct {
	allowStale bool
	ttls       Prefixes[time.Duration]
	data       collections.ConcurrentMap[K, cacheObject[K, T]]
	fetchFn    fetchCacheKeysFromStorageFunc[K, T, E]

	ttl       time.Duration
	delimiter string
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
		return CacheKeyTTL
	}
}

func (c *memoryCache[K, T, E]) absent(keys ...K) (out []K) {
	if len(keys) == 0 {
		return []K{}
	}
	out = make([]K, 0)
	for _, key := range keys {
		if t := c.data.Get(key); t == nil || !t.valid() {
			out = append(out, key)
		}
	}
	return
}

func (c *memoryCache[K, T, E]) Get(key K) *T {
	var res []T
	if len(c.absent(key)) > 0 {
		if res, _ = c.Fetch(context.TODO(), key); len(res) > 0 {
			return &res[0]
		} else if !c.allowStale {
			return nil
		}
	}
	if t := c.data.Get(key); t != nil && (t.valid() || c.allowStale) {
		return t.data
	}
	return nil
}

func (c *memoryCache[K, T, E]) SetTTL(key K, ttl time.Duration) {
	switch k := any(key).(type) {
	case string:
		c.ttls.Set(k, ttl)
	case int, int8, int16, int32, int64, float32, float64:
		c.ttls.Set(fmt.Sprint(k), ttl)
	}
}

func (c *memoryCache[K, T, E]) Fetch(ctx context.Context, keys ...K) (out collections.List[T], err E) {
	var res = make(collections.List[T], 0)
	if len(keys) > 0 && len(c.absent(keys...)) == 0 {
		return
	} else if res, err = c.fetchFn(ctx, keys); any(err) != nil {
		return
	}
	var now = time.Now()
	var data = make([]collections.KeyValue[K, cacheObject[K, T]], len(res))
	out = make(collections.List[T], len(res))
	for i, st := range res {
		out[i], data[i] = st, collections.KeyValue[K, cacheObject[K, T]]{
			Key: st.Key(), Value: cacheObject[K, T]{
				expiry: now.Add(c.GetTTL(st.Key())), data: &st,
			},
		}
	}
	c.data.AddMany(data...)
	// todo: async job to remove stale/expired data
	return
}

func (c *memoryCache[K, T, E]) Load(ctx context.Context, keys ...K) (er E) {
	_, er = c.Fetch(ctx, keys...)
	return
}
