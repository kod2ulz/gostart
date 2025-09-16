package http

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

const (
	RequestID = "X-REQUEST-ID"
)

// Session represents an authentication session
type Session interface {
	Authorization() string
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

// Additional methods for compatibility with existing client.go

// MergeHeaders merges another Headers instance
func (p *Headers) MergeHeaders(other Headers) *Headers {
	for k, v := range other {
		(*p)[k] = v
	}
	return p
}

// Empty checks if the Headers map is empty
func (p *Headers) Empty() bool {
	return len(*p) == 0
}

// HasKey checks if a specific key exists in the Headers
func (p *Headers) HasKey(key string) bool {
	_, exists := (*p)[key]
	return exists
}

// MergeStringMap merges a map[string]string into Headers
func (p *Headers) MergeStringMap(other map[string]string) *Headers {
	for k, v := range other {
		(*p)[k] = v
	}
	return p
}