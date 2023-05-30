package collections

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

func ListMapToNoPtrFunc[T any](_ int, t *T) T { return *t }

func ListMapToPtrFunc[T any](_ int, t T) *T { return &t }

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
	if len(list) == 0 {
		return List[T]{}
	}
	return List[T](list)
}

func MapList[A any, B any](list List[A], mapFn func(A) B) (out List[B]) {
	if len(list) == 0 {
		return []B{}
	}
	out = make(List[B], len(list))
	for i := range list {
		out[i] = mapFn(list[i])
	}
	return
}
