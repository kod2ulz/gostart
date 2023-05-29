package object

import (
	"strconv"

	"golang.org/x/exp/constraints"
)

func Num[T constraints.Integer](str string) (out T) {
	var v T
	switch any(v).(type) {
	default:
		i, _ := strconv.Atoi(str)
		out = T(i)
	case int64, uint64:
		i, _ := strconv.ParseInt(str, 10, 64)
		out = T(i)
	}
	return
}
