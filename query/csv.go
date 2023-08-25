package query

import "github.com/kod2ulz/gostart/collections"

type CSV interface {
	Set(headers ...string) CSV
	Data(data ...string) CSV
	Empty() bool
	IsTitle() bool
	Read(header string) string
}

func Csvv(titles ...string) (out CSV) {
	out = &csvHeaders{}
	return out.Set(titles...)
}

type csvHeaders struct {
	collections.Map[string, int]
	data collections.List[string]
}

func (h *csvHeaders) Set(headers ...string) CSV {
	if len(headers) == 0 {
		return h
	} else if h.Map == nil {
		h.Map = collections.Map[string, int]{}
	}
	for i, label := range headers {
		h.Map[label] = i
	}
	return h
}

func (h *csvHeaders) Data(data ...string) CSV {
	h.data = data
	return h
}

func (h *csvHeaders) Empty() bool {
	return len(h.data) == 0
}

func (h *csvHeaders) IsTitle() bool {
	if len(h.Map) == 0 || len(h.data) == 0 {
		return false
	}
	for header := range h.Map {
		if header != h.Read(header) {
			return false
		}
	}
	return true
}

func (h *csvHeaders) Read(header string) (out string) {
	var i int
	if h.Empty() {
		return
	} else if i = h.Map[header]; i > h.data.Size() {
		return
	}
	return h.data[i]
}
