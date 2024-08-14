package collections

import (
	"sync"
)

func NewConcurrentMap[K comparable, T any]() *ConcurrentMap[K, T] {
	return &ConcurrentMap[K, T]{data: make(Map[K, T], 0)}
}

type ConcurrentMap[K comparable, T any] struct {
	data Map[K, T]
	mx   sync.RWMutex
}

func (m *ConcurrentMap[K, T]) Size() int {
	m.mx.RLock()
	defer m.mx.RUnlock()
	return m.data.Size()
}

func (m *ConcurrentMap[K, T]) Get(key K) *T {
	m.mx.RLock()
	defer m.mx.RUnlock()
	return m.data.Get(key)
}

func (m *ConcurrentMap[K, T]) Empty() bool {
	m.mx.RLock()
	defer m.mx.RUnlock()
	return m.data.Empty()
}

func (m *ConcurrentMap[K, T]) Values() (out List[T]) {
	m.mx.RLock()
	defer m.mx.RUnlock()
	return m.data.Values()
}

func (m *ConcurrentMap[K, T]) AnyOfKey(keys ...K) (out T) {
	m.mx.RLock()
	defer m.mx.RUnlock()
	return m.data.AnyOfKey(keys...)
}

func (m *ConcurrentMap[K, T]) Keys() (out List[K]) {
	m.mx.RLock()
	defer m.mx.RUnlock()
	return m.data.Keys()
}

func (m *ConcurrentMap[K, T]) Add(key K, value T) (isNew bool) {
	m.mx.Lock()
	defer m.mx.Unlock()
	isNew = !m.data.Has(key)
	m.data.Add(key, value)
	return
}

func (m *ConcurrentMap[K, T]) AddMany(kvs ...KeyValue[K, T]) {
	m.mx.Lock()
	defer m.mx.Unlock()
	if len(kvs) == 0 {
		return
	}
	for _, kv := range kvs {
		m.Add(kv.Key, kv.Value)
	}
}

func (m *ConcurrentMap[K, T]) Map(fn func(K, T) (K, any, bool)) (out Map[K, any]) {
	m.mx.RLock()
	defer m.mx.RUnlock()
	return m.data.Map(fn)
}

func (m *ConcurrentMap[K, T]) HasKey(k K) (found bool) {
	m.mx.RLock()
	defer m.mx.RUnlock()
	return m.data.HasKey(k)
}

func (m *ConcurrentMap[K, T]) Merge(in map[K]T) Map[K, T] {
	m.mx.Lock()
	defer m.mx.Unlock()
	return m.data.Merge(in)
}

func (m *ConcurrentMap[K, T]) Clear() {
	m.mx.Lock()
	defer m.mx.Unlock()
	m.data.Clear()
}

type KeyValue[K comparable, T any] struct {
	Key   K
	Value T
}
