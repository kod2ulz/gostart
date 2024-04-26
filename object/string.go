package object

import (
	"fmt"
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

func (s String) Variations(formats...string) (out List[string]) {
	if s == "" || len(formats) == 0 {
		return List[string]{string(s)}
	}
	out = make(List[string], len(formats))
	for i := range formats {
		out = append(out, fmt.Sprintf(formats[i], s))
	}
	return 
}