package collections

import "sync"


func ConcurrentList[T any]() *concurrentList[T] {
	return &concurrentList[T]{data: make(List[T], 0)}
}

type concurrentList[T any] struct {
	data List[T]
	mx   sync.RWMutex
}

func (l *concurrentList[T]) Empty() bool {
	l.mx.Lock()
	defer l.mx.Unlock()
	return l.data.Empty()
}

func (l *concurrentList[T]) Size() int {
	l.mx.RLock()
	defer l.mx.RUnlock()
	return l.data.Size()
}

func (l *concurrentList[T]) Data() List[T] {
	l.mx.RLock()
	defer l.mx.RUnlock()
	return l.data
}

func (l *concurrentList[T]) First() (t T) {
	l.mx.RLock()
	defer l.mx.RUnlock()
	return l.data.First()
}

func (l *concurrentList[T]) Last() (t T) {
	l.mx.RLock()
	defer l.mx.RUnlock()
	return l.data.Last()
}

func (l *concurrentList[T]) Clear() bool {
	l.mx.Lock()
	defer l.mx.Unlock()
	return l.data.Clear()
}

func (l *concurrentList[T]) Add(args ...T) *concurrentList[T] {
	l.mx.Lock()
	defer l.mx.Unlock()
  l.data.Add(args...)
	return l
}

func (l *concurrentList[T]) Of(args ...T) *concurrentList[T] {
	l.mx.Lock()
	defer l.mx.Unlock()
  l.data.Of(args...)
	return l
}

func (l *concurrentList[T]) Append(args ...T) *concurrentList[T] {
	l.mx.Lock()
	defer l.mx.Unlock()
  l.data.Append(args...)
	return l
}

func (l concurrentList[T]) Sort(lessFn func(t1, t2 T) bool) (out []T) {
	l.mx.RLock()
	defer l.mx.RUnlock()
	return l.data.Sort(lessFn)
}

func (l *concurrentList[T]) Iterator() Iterator[T] {
	l.mx.RLock()
	defer l.mx.RUnlock()
	return l.data.Iterator()
}

func (l concurrentList[T]) Iterate(fn func(int, T) error) (err error) {
	l.mx.RLock()
	defer l.mx.RUnlock()
	return l.data.Iterate(fn)
}

func (l *concurrentList[T]) Filter(filterFn func(i int, val T) bool) (out List[T]) {
	l.mx.RLock()
	defer l.mx.RUnlock()
	return l.data.Filter(filterFn)
}

func (l concurrentList[T]) ForEach(fn func(i int, val T) T) (out []T) {
	l.mx.RLock()
	defer l.mx.RUnlock()
	return l.data.ForEach(fn)
}

func (l concurrentList[T]) Slice(from, to int) (out []T) {
	l.mx.RLock()
	defer l.mx.RUnlock()
	return l.data.Slice(from, to)
}
