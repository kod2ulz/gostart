package collections

import (
	"fmt"
	"strconv"

	"golang.org/x/exp/constraints"
)

type ComparableList[T comparable] struct {
	List[T]
}

func (l ComparableList[T]) ToMap() (out Map[T, T]) {
	if l.Empty() {
		return
	}
	return ListToMap(List[T](l.List), func(t T) (T, T) {
		return t, t
	})
}

func (l ComparableList[T]) ToSet() (out Set[T]) {
	if l.Empty() {
		return
	}
	return Set[T](ListToMap(List[T](l.List), func(t T) (T, struct{}) {
		return t, struct{}{}
	}))
}

func (l ComparableList[T]) Without(omit []T) (out ComparableList[T]) {
	if l.Empty() || len(omit) == 0 {
		return l
	}
	idx := ComparableList[T]{List: omit}.ToMap()
	out = ComparableList[T]{}
	copy(out.List, l.List)
	var j int
	for i := range l.List {
		if _, ok := idx[l.List[i]]; !ok {
			j++
			continue
		}
		out.List = append(out.List[:j], out.List[j+1:]...)
	}
	return
}

func (l ComparableList[T]) ToStringList(toStrFn ...func(T) string) (out List[string]) {
	if l.Empty() {
		return
	}
	if len(toStrFn) == 0 || toStrFn[0] == nil {
		toStrFn[0] = func(t T) string { return fmt.Sprint(t) }
	}
	return MapList(l.List, func(t T) string { return toStrFn[0](t) })
}

type Number interface {
	constraints.Integer | constraints.Float
}

type StringList = ComparableList[string] 

type NumericList[T Number] struct{ ComparableList[T] }

type Int64List struct{ NumericList[int64] }

func (l Int64List) StringList() (out List[string]) {
	return l.ToStringList(func(i int64) string {
		return strconv.Itoa(int(i))
	})
}
