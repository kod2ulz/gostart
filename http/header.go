package http

import (
	"context"
	"encoding/base64"
	"net/http"

	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/api"
)

type Headers map[string]string

func (p *Headers) Merge(in map[string]string) *Headers {
	if len(in) == 0 {
		return p
	}
	for k, v := range in {
		(*p)[k] = v
	}
	return p
}

func (p *Headers) Add(key, value string) *Headers {
	(*p)[key] = value
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
