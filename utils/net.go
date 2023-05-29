package utils

import (
	"io"

	"github.com/pkg/errors"
	json "github.com/json-iterator/go"
)

var Net netutils

type netutils struct {}

func (netutils) ReadJson(reader io.ReadCloser, out interface{}) (err error) {
	if reader == nil {
		return
	}
	var data []byte
	data, err = io.ReadAll(reader)
	if err != nil {
		return errors.Wrap(err, "error ready response body from API")
	}
	defer reader.Close()

	if err = json.Unmarshal(data, out); err != nil {
		return errors.Wrapf(err, "failed to unmarshall response to %T", out)
	}
	return
}