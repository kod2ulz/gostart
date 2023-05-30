package object

import (
	"strings"
	. "github.com/kod2ulz/gostart/collections"
)

type String string

func (s String) String() string {
	return string(s)
}

func (s String) Split(sep string) (out List[string]) {
	if sep == "" || len(s) == 0 {
		return List[string]{}
	}
	return List[string](strings.Split(string(s), sep))
}