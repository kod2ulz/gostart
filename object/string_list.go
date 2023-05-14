package object

type StringList List[string]

func (l StringList) Set() (out Set[string]) {
	if len(l) == 0 {
		return
	}
	return Set[string](ListToMap(List[string](l), func(t string) (string, struct{}) {
		return t, struct{}{}
	}))
}

func ListOf[T any](list ...T) List[T] {
	if len(list) == 0{
		return List[T]{}
	}
	return List[T](list)
}