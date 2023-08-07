package collections

import "strings"

var Filters = filters{
	Strings: stringFilters{},
}

type filters struct {
	Strings stringFilters
}

type stringFilters struct {}

func (stringFilters) NotBlank(_ int, str string) bool {
	return strings.Trim(str, " ") != "" 
}