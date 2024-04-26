package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kod2ulz/gostart/utils"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

func ErrorResponse[T any](err Error) Response[T] {
	return Response[T]{
		Timestamp: time.Now().Unix(), Error: err,
		Type: strings.TrimPrefix(fmt.Sprintf("%T", new(T)), "*")}
}

func DataResponse[T any](data T) Response[T] {
	return Response[T]{
		Timestamp: time.Now().Unix(), Data: data, Success: true,
		Type: strings.TrimPrefix(fmt.Sprintf("%T", new(T)), "*")}
}

func ListResponse[T any](data []T, meta Metadata) Response[[]T] {
	return Response[[]T]{
		Timestamp: time.Now().Unix(), Data: data, Meta: &meta, Success: true,
		Type: strings.TrimPrefix(fmt.Sprintf("%T", new(T)), "*") + "[]"}
}

func EmptyResponse[T any]() (out Response[T]) {
	return ErrorResponse[T](GeneralError[T](nil))
}

type Response[T any] struct {
	code       int            `json:"-"`
	headers    http.Header    `json:"-"`
	cookies    []*http.Cookie `json:"-"`
	Success    bool           `json:"success"`
	Type       string         `json:"type,omitempty"`
	Error      Error          `json:"error,omitempty"`
	Data       interface{}    `json:"data,omitempty"`
	References map[string]any `json:"references,omitempty"`
	Meta       *Metadata      `json:"meta,omitempty"`
	Timestamp  int64          `json:"time,omitempty"`
}

func (r Response[T]) HasError() bool {
	return r.Error != nil
}

func (r Response[T]) ParseDataTo(out *T) error {
	if out == nil {
		return errors.New("out is nil")
	} else if r.Data == nil {
		return errors.New("Response.Data is nil")
	} else if data, ok := r.Data.(T); ok {
		*out = data
	} else if data, ok := r.Data.(map[string]interface{}); ok {
		if err := mapstructure.Decode(data, out); err == nil {
			return nil
		}
	}
	return errors.Wrapf(utils.StructCopy(r.Data, out), "failed to parse %T to %T", r.Data, *out)
}

func (r Response[T]) Failed() bool {
	return r.HasError()
}

func (r Response[T]) WithReferences(refs map[string]any) Response[T] {
	if len(refs) > 0 {
		r.References = refs
	}
	return r
}

func (r Response[T]) WithHeaders(headers http.Header) Response[T] {
	if len(headers) > 0 {
		r.headers = headers
	}
	return r
}

func (r Response[T]) WithCookies(cookies []*http.Cookie) Response[T] {
	if len(cookies) > 0 {
		r.cookies = cookies
	}
	return r
}

func (r Response[T]) WithCode(code int) Response[T] {
	r.code = code
	return r
}

func (r Response[T]) Cookies() []*http.Cookie {
	return r.cookies
}

func (r Response[T]) Headers() http.Header {
	return r.headers
}

func (r Response[T]) Code() int {
	return r.code
}

type Metadata struct {
	Total   int64 `json:"total,omitempty"`
	Current int64 `json:"current,omitempty"`
	Limit   int64 `json:"limit,omitempty"`
	Offset  int64 `json:"offset,omitempty"`
	Page    int64 `json:"page,omitempty"`
}

func (m *Metadata) WithTotal(total int64) *Metadata {
	m.Total = total
	return m
}

func (m *Metadata) WithLimit(limit int64) *Metadata {
	m.Limit = limit
	return m
}

func (m *Metadata) WithOffset(offset int64) *Metadata {
	m.Offset = offset
	return m
}

func (m *Metadata) WithPage(page int64) *Metadata {
	m.Page = page
	return m
}

func (m *Metadata) WithCurrent(current int64) *Metadata {
	m.Current = current
	return m
}
