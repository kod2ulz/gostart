package http

import (
	"bytes"
	"io"
	"net/http"
	"net/url"

	"github.com/goccy/go-json"
	"github.com/pkg/errors"
)

func setUrlQueryParams(_url *url.URL, params map[string][]string) {
	if len(params) == 0 {
		return
	}
	query := _url.Query()
	for k, vals := range params {
		if len(vals) == 0 {
			continue
		}
		for i := range vals {
			query.Add(k, vals[i])
		}
	}
	_url.RawQuery = query.Encode()
}

func toReader(body interface{}) (out io.Reader, err error) {
	var ok bool
	var data []byte
	if body == nil {
		return nil, nil
	} else if out, ok = body.(io.Reader); ok {
		return out, nil
	} else if data, err = json.Marshal(body); err != nil {
		return nil, errors.Wrap(err, "failed to marshal request body to json")
	}
	return bytes.NewReader(data), nil
}

func newHttpRequest(_url *url.URL, method string, body interface{}) (request *http.Request, err error) {
	var payload io.Reader
	if payload, err = toReader(body); err != nil {
		return nil, errors.Wrap(err, "failed to encode body into reader")
	} 
	return http.NewRequest(method, _url.String(), payload)
}
