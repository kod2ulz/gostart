package api

type Meta struct {
	Total   int64 `json:"total,omitempty"`
	Current int64 `json:"current,omitempty"`
	Limit   int64 `json:"limit,omitempty"`
	Offset  int64 `json:"offset,omitempty"`
	Page    int64 `json:"page,omitempty"`
}

type Response[T any] struct {
	Data       T                      `json:"data,omitempty"`
	References map[string]interface{} `json:"references,omitempty"`
	Meta       *Meta                  `json:"meta,omitempty"`
	Error      error                  `json:"error,omitempty"`
}

func (r Response[T]) HasError() bool {
	return r.Error != nil
}

func DataResponse[T any](data T, meta Meta) Response[T] {
	return Response[T]{
		Data: data,
		Meta: &meta,
	}
}

type ListResponse[T any] struct {
	Data []T
	Meta Meta
}
