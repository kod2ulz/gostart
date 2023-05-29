package utils

import (
	json "github.com/json-iterator/go"
)

type structUtils struct{}

// StructCopy there must have a better way of doing this
func StructCopy[A , B any](from A, to B) (err error) {
	var in []byte
	if in, err = json.Marshal(from); err != nil {
		return
	}
	return json.Unmarshal(in, to)
}
