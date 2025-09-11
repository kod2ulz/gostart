package object

import (
	"fmt"
	"strings"

	"github.com/kod2ulz/gostart/collections"
)

type String string

func (s String) String() string {
	return string(s)
}

func (s String) Split(sep string) (out collections.List[string]) {
	if len(s) == 0 {
		return []string{}
	} else if len(sep) == 0 || !strings.Contains(string(s), sep) {
		return []string{string(s)}
	}
	return strings.Split(string(s), sep)
}

func (s String) Variations(formats...string) (out collections.List[string]) {
	if s == "" || len(formats) == 0 {
		return collections.List[string]{string(s)}
	}
	out = make(collections.List[string], len(formats))
	for i := range formats {
		out[i] = fmt.Sprintf(formats[i], s)
	}
	return 
}

func (s String) SubstringBefore(r byte) String {
	if len(s) == 0 {
		return s
	} else if !strings.ContainsRune(string(s), rune(r)) {
		return s
	}
	return String(string(s)[:strings.IndexByte(string(s), r)])
}

func (s String) Before(r byte) String {
	return s.SubstringBefore(r)
}