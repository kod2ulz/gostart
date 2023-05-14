package api

import (
	"fmt"
	"strings"
	"time"
)

func ErrorResponse[T any](err Error) Response[T] {
	return Response[T]{
		Timestamp: time.Now().Unix(), Error: err,
		Type: strings.TrimPrefix(fmt.Sprintf("%T", new(T)), "*")}
}

func DataResponse[T any](data T) Response[T] {
	return Response[T]{
		Timestamp: time.Now().Unix(), Data: data,
		Type: strings.TrimPrefix(fmt.Sprintf("%T", new(T)), "*")}
}

func ListResponse[T any](data []T, meta Metadata) Response[[]T] {
	return Response[[]T]{
		Timestamp: time.Now().Unix(), Data: data, Meta: &meta,
		Type: strings.TrimPrefix(fmt.Sprintf("%T", new(T)), "*")}
}

type Response[T any] struct {
	Type       string                 `json:"type,omitempty"`
	Error      Error                  `json:"error,omitempty"`
	Data       interface{}            `json:"data,omitempty"`
	References map[string]interface{} `json:"references,omitempty"`
	Meta       *Metadata              `json:"meta,omitempty"`
	Timestamp  int64                  `json:"time,omitempty"`
}

func (r Response[T]) HasError() bool {
	return r.Error != nil
}

func (r Response[T]) Failed() bool {
	return r.HasError()
}

type Metadata struct {
	Total   int64 `json:"total,omitempty"`
	Current int64 `json:"current,omitempty"`
	Limit   int64 `json:"limit,omitempty"`
	Offset  int64 `json:"offset,omitempty"`
	Page    int64 `json:"page,omitempty"`
}
