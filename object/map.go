package object


type Map[K comparable, T any] map[K]T

func (m Map[K, T]) Size() int {
	return len(m)
}

func (m Map[K, T]) Empty() bool {
	return m.Size() == 0
}

func (m Map[K, T]) Values() (out List[T]) {
	if m.Empty() {
		return nil
	}
	out = List[T]{}
	for _, v := range m {
		out = append(out, v)
	}
	return
}

func (m Map[K, T]) AnyOfKey(keys...K) (out T) {
	if m.Empty() {
		return out
	}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v
		} 
	}
	return
}

func (m Map[K, T]) Keys() (out List[K]) {
	if m.Empty() {
		return nil
	}
	out = List[K]{}
	for k := range m {
		out = append(out, k)
	}
	return
}

func (m *Map[K, T]) Add(key K, value T) *Map[K, T] {
	(*m)[key] = value
	return m
}

func (m Map[K, T]) Map(fn func(K, T) (K, any, bool)) (out Map[K, any]) {
	out = make(Map[K, any])
	for k, v := range m {
		ko, vo, ok := fn(k, v)
		if !ok {
			continue
		} else if _, ok := out[ko]; !ok {
			out[ko] = vo
		}
	}
	return 
}

func (m Map[K, T]) HasKey(k K) (found bool) {
	if m.Empty() {
		return false
	}
	_, found = m[k]
	return
}



func (m Map[K, T]) Merge(in map[K]T) Map[K, T] {
	if len(in) == 0 {
		return m
	} else if len(m) == 0 {
		m = Map[K, T]{}
	}
	for k, v := range in {
		m[k] = v
	}
	return m
}

func MapOf[K comparable, V any](keyVal ...interface{}) (out Map[K, V]) {
	if len(keyVal) == 0 {
		return 
	}
	max := len(keyVal)
	if max < 2{
		return
	} else 	if max == 2{
		return Map[K, V]{keyVal[0].(K): keyVal[1].(V)}
	} else if max%2 != 0 {
		max--
	}
	out = make(Map[K, V])
	for i := 0; i < max-1; i +=2 {
		out[keyVal[i].(K)] = keyVal[i+1].(V)
	}

	return
}

func ConvertMap[K1, K2 comparable, T1, T2 any](in map[K1]T1, cFn func(K1, T1) (K2, T2)) (out Map[K2, T2]) {
	if len(in) == 0 {
		return
	}
	out = make(Map[K2, T2])
	for k, v := range in {
		k2, v2 := cFn(k, v)
		out[k2] = v2
	}
	return 
}
