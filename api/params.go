package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Params map[string]string

func (p *Params) Add(key, value string) *Params {
	(*p)[key] = value
	return p
}

func (p *Params) AddIfMissing(key, value string) *Params {
	if _, ok := (*p)[key]; !ok {
		(*p)[key] = value
	}
	return p
}

func (p *Params) SetQueryParams(_url *url.URL) {
	if len(*p) == 0 {
		return
	}
	query := _url.Query()
	for k, v := range *p {
		query.Add(k, v)
	}
	_url.RawQuery = query.Encode()
}

func (p *Params) Request(method, path string, body interface{}) (request *http.Request, err error) {
	switch method {
	case http.MethodGet:
		_url, err := url.Parse(path)
		if err != nil {
			return nil, err
		}
		p.SetQueryParams(_url)
		return http.NewRequest(method, _url.String(), nil)
	case http.MethodPost:
		var payload interface{}
		if body != nil {
			payload = body
			// payload = utils.JSON.ToMap(body).Merge(utils.ConvertMap(*p, p.remapperFunc))
		} else {
			payload = p
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		return http.NewRequest(method, path, bytes.NewReader(data))
	default:
		panic("method " + method + " unsupported")
	}
}

func (p *Params) Merge(in map[string]interface{}) *Params {
	if len(in) == 0 {
		return p
	}
	for k, v := range in {
		(*p)[k] = fmt.Sprintf("%v", v)
	}
	return p
}

func (p *Params) remapperFunc(k1 string, v1 string) (k2 string, v2 interface{}) {
	k2, v2 = k1, v1
	return
}