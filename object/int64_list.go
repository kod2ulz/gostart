package object

import "strconv"

type Int64List List[int64]

func (l Int64List) Size() int {
	return List[int64](l).Size()
}

func (l Int64List) Empty() bool {
	ls := List[int64](l)
	return ls.Empty()
}

func (l Int64List) Map() (out Map[int64, int64]) {
	if l.Empty() {
		return
	}
	return ListToMap(List[int64](l), func(t int64) (int64, int64) {
		return t, t
	})
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

func (l Int64List) Set() (out Set[int64]) {
	if l.Empty() {
		return
	}
	return Set[int64](ListToMap(List[int64](l), func(t int64) (int64, struct{}) {
		return t, struct{}{}
	}))
}

func (l Int64List) StringList() (out List[string]) {
	if l.Empty() {
		return List[string]{}
	}
	out = make(List[string], l.Size())
	for i := range l {
		out[i] = strconv.Itoa(int(l[i]))
	}
	return
}

func (l Int64List) Without(omit []int64) (out Int64List) {
	if l.Empty() || len(omit) == 0 {
		return l
	}
	idx := Int64List(omit).Map()
	copy(out, l)
	var j int
	for i := range l {
		if _, ok := idx[l[i]]; !ok {
			j++
			continue
		}
		out = append(out[:j], out[j+1:]...)
	}
	return
}