package collections

type Iterator[T any] interface {
	HasNext() bool
	Next() *T
}

type iterator[T any] struct {
	index int
	data  []T
}

func (u *iterator[T]) HasNext() bool {
	return u.index < len(u.data)
}

func (u *iterator[T]) Next() *T {
	if u.HasNext() {
		i := u.data[u.index]
		u.index++
		return &i
	}
	return nil
}