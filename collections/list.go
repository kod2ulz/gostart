package collections

import (
	"sort"
)

type List[T any] []T

func (l *List[T]) Empty() bool {
	return l.Size() == 0
}

func (l List[T]) Size() int {
	return len(l)
}

func (l List[T]) First() (t T) {
	if !l.Empty() {
		return l[0]
	}
	return
}

func (l List[T]) Last() (t T) {
	if !l.Empty() {
		return l[l.Size()-1]
	}
	return
}

func (l *List[T]) Clear() bool {
	if l.Empty() {
		return false
	}
	(*l) = make(List[T], 0)
	return true
}

func (l *List[T]) Add(args ...T) *List[T] {
	return l.Append(args...)
}

func (l *List[T]) Of(args ...T) *List[T] {
	return l.Append(args...)
}

func (l *List[T]) Append(args ...T) *List[T] {
	if len(args) == 0 {
		return l
	}
	(*l) = append((*l), args...)
	return l
}

func (l *List[T]) Internal() (out []T) {
	if !l.Empty() {
		return (*l)
	}
	return
}

func (l List[T]) Sort(lessFn func(t1, t2 T) bool) (out []T) {
	if l.Empty() {
		return l
	}
	out = make([]T, l.Size())
	copy(out, l)
	sort.Slice(out, func(i, j int) bool {
		return lessFn(out[i], out[j])
	})
	return
}

func (l List[T]) Iterate(fn func(int, T) error) (err error) {
	if len(l) == 0 {
		return
	}
	for i := range l {
		if err = fn(i, l[i]); err != nil {
			return
		}
	}
	return
}

func (l List[T]) Filter(filterFn func(i int, val T) bool) (out []T) {
	if l.Empty() {
		return l
	}
	out = make([]T, 0)
	for i := range l {
		if filterFn(i, l[i]) {
			out = append(out, l[i])
		}
	}
	return
}

func (l List[T]) ForEach(fn func(i int, val T) T) (out []T) {
	if l.Empty() {
		return l
	}
	out = make([]T, len(l))
	for i := range l {
		out[i] = fn(i, l[i])
	}
	return
}

func ListReduce[T any, K any](l List[T], mapFn func(i int, val T) (K, bool)) (out []K) {
	if l.Empty() {
		return []K{}
	}
	out = make([]K, 0)
	for i := range l {
		if k, ok := mapFn(i, l[i]); ok {
			out = append(out, k)
		}
	}
	return
}

func ListMap[T any, K any](l List[T], mapFn func(i int, val T) K) (out []K) {
	if l.Empty() {
		return []K{}
	}
	out = make([]K, 0)
	for i := range l {
		out = append(out, mapFn(i, l[i]))
	}
	return
}

type ListMapFunc[K any, T any] func(T) (K, T)

func (l *List[T]) MapString(fn ListMapFunc[string, T]) (out Map[string, T]) {
	if l.Empty() {
		return
	}
	out = make(Map[string, T], 0)
	for _, t := range *l {
		k, v := fn(t)
		out[k] = v
	}
	return
}

func ListToMap[K comparable, T any, U any](list List[T], kvFn func(T) (K, U)) (out Map[K, U]) {
	if list.Empty() {
		return
	}
	out = make(Map[K, U], 0)
	for _, t := range list {
		k, v := kvFn(t)
		out[k] = v
	}
	return
}

func InList[T comparable](needle T, haystach ...T) bool {
	if len(haystach) == 0 {
		return false
	}
	for i := range haystach {
		if needle == haystach[i] {
			return true
		}
	}
	return false
}

func ListOf[T any](list ...T) List[T] {
	if len(list) == 0{
		return List[T]{}
	}
	return List[T](list)
}