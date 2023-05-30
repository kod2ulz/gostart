package collections

type Set[T comparable] Map[T, struct{}]

func (m Set[T]) Size() int {
	return len(m)
}

func (m Set[T]) Empty() bool {
	return m.Size() == 0
}

func (m Set[T]) Values() (out List[T]) {
	if m.Empty() {
		return nil
	}
	out = List[T]{}
	for v := range m {
		out = append(out, v)
	}
	return
}

func (m *Set[T]) Add(val T) *Set[T] {
	if !m.Has(val) {
		(*m)[val] = struct{}{}
	}
	return m
}

func (m Set[T]) Has(val T)(found bool) {
	if m.Empty() {
		return false
	}
	_, found = m[val]
	return
}
