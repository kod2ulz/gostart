package http

import (
	"context"
	"encoding/base64"
	"net/http"

	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/collections"
)

type Headers map[string]collections.Set[string]

func (p *Headers) Values() (out map[string][]string) {
	return collections.ConvertMap(*p, func(k string, v collections.Set[string]) (string, []string) {
		return k, v.Values()
	})
}

func (p *Headers) MergeHeaders(in Headers) *Headers {
	if len(in) == 0 {
		return p
	}
	for k, v := range in {
		p.Add(k, v.Values()...)
	}
	return p
}

func (p *Headers) Merge(in map[string]string) *Headers {
	if len(in) == 0 {
		return p
	}
	for k, v := range in {
		p.Add(k, v)
	}
	return p
}

func (p *Headers) Add(key string, value...string) *Headers {
	if len(value) == 0 {
		return p
	}
	set, ok := (*p)[key]
	if !ok {
		set = make(collections.Set[string])
	} 
	for i := range value {
		set.Add(value[i])
	}
	(*p)[key] = set
	return p
}

func (p *Headers) WithRequestID(ctx context.Context) *Headers {
	if _, ok := (*p)[api.RequestID]; ok {
		return p
	} else if val := ctx.Value(api.RequestID); val != nil {
		return p.Add(api.RequestID, val.(string))
	}
	return p.Add(api.RequestID, uuid.New().String())
}

func (p *Headers) WithAuthorization(session Session) *Headers {
	if session == nil {
		return p
	} else if auth := session.Authorization(); auth != "" {
		p.Add("Authorization", auth)
	}
	return p
}

func (p *Headers) Set(request *http.Request) {
	if request == nil || len(*p) == 0 {
		return
	}
	request.Header = p.Values()
}

func (p *Headers) WithBearerToken(token string) *Headers {
	if token != "" {
		p.Add("Authorization", "Bearer " + token)
	}
	return p
}

func (p *Headers) WithBasicAuth(username, password string) *Headers {
	if username != "" {
		creds := []byte(username + ":" + password)
		p.Add("Authorization", "Basic " + base64.StdEncoding.EncodeToString(creds))
	}
	return p
}

func (p *Headers) Empty() bool {
	return len(*p) == 0
}

func (p *Headers) HasKey(key string) bool {
	return  p != nil && len(*p) > 0 && p.HasKey(key)
}
