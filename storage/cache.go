package storage

import (
	"context"
	"sync"
	"time"

	"github.com/kod2ulz/gostart/collections"
)

var (
	CacheKeyTTL = 15 * time.Second
)

type Cache[K comparable, T any, E error] interface {
	// Get attempts to retrieve an object from the cache. will return nil if it's not found
	Get(key K) *T
	// Fetches objects matching keys from the backend storage into the cache, if they are not in the cache and not expired
	Fetch(ctx context.Context, keys ...K) (T, E)
	// Loads objects matching the keys into the cache. It will load all if keys are not specfied
	Load(ctx context.Context, keys ...K) E
}

type cacheInitialiser[K comparable, T any, E error] func(Cache[K, T, E])

type fetchByKeysFunc[K comparable, T any, E error] func(ctx context.Context, keys []K) ([]T, E)

func WithGetAllFunc[K comparable, T any, E error](getAllFn func(ctx context.Context, keys []K) ([]T, E)) cacheInitialiser[K, Cache[K, T, E], E] {
	return nil
	// return func(c Cache[K, T, E]) {


	// 	// switch c.(type) {
	// 	// case *memoryCache[K, cacheObject[T], E]:
	// 	// 	// c.(*memoryCache[K, T, E]).getAll = getAllFn
	// 	// default:
	// 	// 	panic("Invalid cache type")
	// 	// }
	// }
}

type cacheObject[T any] struct {
	// refreshed time.Time
	expiry time.Time
	data   *T
}

func (o *cacheObject[T]) valid() bool { return o.expiry.After(time.Now()) }

func NewMemoryCache[K comparable, T cacheObject[T], E error](opts...cacheInitialiser[K,T,E]) (out *memoryCache[K, T, E]) {
	out = &memoryCache[K, T, E]{
		data: collections.NewConcurrentMap[K, T](),
		timeouts: ,
	}
	return
}

type memoryCache[K comparable, T cacheObject[T], E error] struct {
	mx       sync.RWMutex
	data     *collections.ConcurrentMap[K, T]
	timeouts *collections.ConcurrentMap[string, time.Time]
}

func (c *memoryCache[K, T, E]) Get(key K) *T { return nil }

func (c *memoryCache[K, T, E]) Fetch(ctx context.Context, keys ...K) (out T, er E) { return }

func (c *memoryCache[K, T, E]) Load(ctx context.Context, keys ...K) (er E) { return  }
