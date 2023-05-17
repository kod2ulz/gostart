package object
import . "github.com/kod2ulz/gostart/collections"

type StringList List[string]

func (l StringList) Set() (out Set[string]) {
	if len(l) == 0 {
		return
	}
	return Set[string](ListToMap(List[string](l), func(t string) (string, struct{}) {
		return t, struct{}{}
	}))
}