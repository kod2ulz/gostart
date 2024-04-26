package collections

func SetOf[T comparable](items ...T) (out Set[T]) {
	if len(items) == 0 {
		return Set[T]{}
	}
	out = make(Set[T])
	for i := range items {
		out.Add(items[i])
	}
	return 
}