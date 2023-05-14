package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/utils"
	"github.com/pkg/errors"
)

type RequestParam interface {
	Validate(ctx context.Context) error
	RequestLoad(ctx context.Context) (RequestParam, error)
	ContextKey() string
	ContextLoad(context.Context) (RequestParam, error)
}

func ParamsFromContext[P RequestParam](ctx context.Context) (P, error) {
	var err error
	var reqParam RequestParam
	if reqParam, err = (*new(P)).ContextLoad(ctx); err != nil {
		var ctxParam P
		return ctxParam, errors.Wrapf(err, "Failed to load %T from context", ctxParam)
	}
	return reqParam.(P), nil
}

func QueryFromContext(ctx context.Context, name, _default string) utils.Value {
	if v := ctx.(*gin.Context).Query(name); v != "" {
		return utils.Value(v)
	}
	return utils.Value(_default)
}

type Headers map[string]string

func (p *Headers) Merge(in map[string]interface{}) *Headers {
	if len(in) == 0 {
		return p
	}
	for k, v := range in {
		(*p)[k] = fmt.Sprintf("%v", v)
	}
	return p
}

func (p *Headers) Add(key, value string) *Headers {
	(*p)[key] = value
	return p
}

func (p *Headers) WithRequestID(ctx context.Context) *Headers {
	if _, ok := (*p)[RequestID]; ok {
		return p
	}
	val := ctx.Value(RequestID)
	if val != nil {
		return p.Add(RequestID, val.(string))
	}
	return p.Add(RequestID, uuid.New().String())
}

func (p *Headers) WithAuthorization(session Session) *Headers {
	if session == nil {
		return p
	} else if auth := session.Authorization(); auth != "" {
		(*p)["Authorization"] = auth
	}
	return p
}

func (p *Headers) Set(request *http.Request) {
	if request == nil || len(*p) == 0 {
		return
	}
	for key, value := range *p {
		request.Header.Set(key, value)
	}
}

func (p *Headers) WithBearerToken(token string) *Headers {
	if token != "" {
		(*p)["Authorization"] = "Bearer " + token
	}
	return p
}

func (p *Headers) WithBasicAuth(username, password string) *Headers {
	if username != "" {
		creds := []byte(username + ":" + password)
		(*p)["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString(creds)
	}
	return p
}

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
