package contracts

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/kod2ulz/gostart/ierrors"
)

// Response represents a standardized API response
type Response[T any] struct {
	code       int            `json:"-"`
	headers    http.Header    `json:"-"`
	cookies    []*http.Cookie `json:"-"`
	Success    bool           `json:"success"`
	Type       string         `json:"type,omitempty"`
	Error      ierrors.Error  `json:"error,omitempty"`
	Data       interface{}    `json:"data,omitempty"`
	References map[string]any `json:"references,omitempty"`
	Meta       *Metadata      `json:"meta,omitempty"`
	Timestamp  int64          `json:"time,omitempty"`
}

// ErrorResponse creates an error response
func ErrorResponse[T any](err ierrors.Error) Response[T] {
	return Response[T]{
		Timestamp: time.Now().Unix(),
		Error:     err,
	}
}

// DataResponse creates a success response with data
func DataResponse[T any](data T) Response[T] {
	return Response[T]{
		Timestamp: time.Now().Unix(),
		Data:      data,
		Success:   true,
	}
}

// ListResponse creates a success response with a list and metadata
func ListResponse[T any](data []T, meta Metadata) Response[[]T] {
	return Response[[]T]{
		Timestamp: time.Now().Unix(),
		Data:      data,
		Meta:      &meta,
		Success:   true,
	}
}

// EmptyResponse creates an empty error response
func EmptyResponse[T any]() Response[T] {
	return ErrorResponse[T](nil)
}

// HasError returns true if the response contains an error
func (r Response[T]) HasError() bool {
	return r.Error != nil
}

// Failed returns true if the response has an error
func (r Response[T]) Failed() bool {
	return r.HasError()
}

// WithReferences adds references to the response
func (r Response[T]) WithReferences(refs map[string]any) Response[T] {
	if len(refs) > 0 {
		r.References = refs
	}
	return r
}

// WithHeaders adds headers to the response
func (r Response[T]) WithHeaders(headers http.Header) Response[T] {
	if len(headers) > 0 {
		r.headers = headers
	}
	return r
}

// WithCookies adds cookies to the response
func (r Response[T]) WithCookies(cookies []*http.Cookie) Response[T] {
	if len(cookies) > 0 {
		r.cookies = cookies
	}
	return r
}

// WithCode sets the HTTP status code
func (r Response[T]) WithCode(code int) Response[T] {
	r.code = code
	return r
}

// Cookies returns the response cookies
func (r Response[T]) Cookies() []*http.Cookie {
	return r.cookies
}

// Headers returns the response headers
func (r Response[T]) Headers() http.Header {
	return r.headers
}

// Code returns the HTTP status code
func (r Response[T]) Code() int {
	return r.code
}

// Metadata represents pagination and list metadata
type Metadata struct {
	Total   int64 `json:"total,omitempty"`
	Current int64 `json:"current,omitempty"`
	Limit   int64 `json:"limit,omitempty"`
	Offset  int64 `json:"offset,omitempty"`
	Page    int64 `json:"page,omitempty"`
}

// WithTotal sets the total count
func (m *Metadata) WithTotal(total int64) *Metadata {
	m.Total = total
	return m
}

// WithLimit sets the limit
func (m *Metadata) WithLimit(limit int64) *Metadata {
	m.Limit = limit
	return m
}

// WithOffset sets the offset
func (m *Metadata) WithOffset(offset int64) *Metadata {
	m.Offset = offset
	return m
}

// WithPage sets the page
func (m *Metadata) WithPage(page int64) *Metadata {
	m.Page = page
	return m
}

// WithCurrent sets the current count
func (m *Metadata) WithCurrent(current int64) *Metadata {
	m.Current = current
	return m
}

// FileResponse represents a file download response
type FileResponse struct {
	ContentType string
	Filename    string
	Ext         string // Recommended if filename is not set
	Data        []byte
}

func (r FileResponse) GetContentType() string {
	if r.ContentType != "" {
		return r.ContentType
	}
	return "application/octet-stream"
}

func (r FileResponse) GetFilename(ctx RequestContext) string {
	if r.Filename != "" {
		return r.Filename
	}
	if r.Ext != "" {
		return "file." + r.Ext
	}
	return "file"
}

// ParseDataTo parses the response data into the provided target
func (r Response[T]) ParseDataTo(target interface{}) error {
	if r.Data == nil {
		return nil
	}
	data, err := json.Marshal(r.Data)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}