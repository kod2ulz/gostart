package api

import (
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/logr"
	"golang.org/x/exp/constraints"
)

var _ contracts.RequestParam = ListRequest{}

type ListRequest struct {
	Limit  int32 `validate:"required,gte=1"`
	Offset int32 `validate:"omitempty,gte=0"`
	RequestModal[ListRequest]
}

func (r ListRequest) Metadata() (out *Meta) {
	limit := int(r.Limit)
	offset := int(r.Offset)
	out = &Meta{
		Limit:  &limit,
		Offset: &offset,
	}
	return
}

func (r ListRequest) DefaultMetadata(ctx contracts.RequestContext) (out *Meta) {
	out = r.Metadata()
	SetResponseMetadata(ctx, out)
	return
}

func (r ListRequest) GetLimit() int {
	return int(r.Limit)
}

func (r ListRequest) GetOffset() int {
	return int(r.Offset)
}

func (r ListRequest) RequestLoad(ctx contracts.RequestContext) (param contracts.RequestParam, err error) {
	var out ListRequest = ListRequest{}
	var query = func(param string, _default interface{}) int32 {
		return int32(ctx.Query(param, fmt.Sprint(_default)).Int())
	}
	// Note: User loading logic removed since auth.GetUser doesn't exist
	// This should be implemented based on your authentication strategy
	out.Limit = query("limit", 20)
	out.Offset = query("offset", 0)
	page := query("page", 1)
	if page > 1 && out.Offset == 0 {
		out.Offset = out.Limit * (page - 1)
	}

	// Debug logging
	logr.Log().Debug("ListRequest loaded",
		"limit", out.Limit,
		"offset", out.Offset,
		"page", page,
		"calculated_offset", out.Offset)

	if ctxSetter, ok := ctx.(interface{ Set(string, interface{}) }); ok {
		ctxSetter.Set(out.ContextKey(), &out)
	}
	return out, err
}

type ListRequestIdType interface {
	string | uuid.UUID | constraints.Integer
}

type ListRequestWithID[ID ListRequestIdType] struct {
	ID ID `validate:"required"`
	ListRequest
	RequestModal[ListRequestWithID[ID]]
}

func (r *ListRequestWithID[ID]) setId(id interface{}) {
	r.ID = id.(ID)
}

func (r ListRequestWithID[ID]) RequestLoad(ctx contracts.RequestContext) (param contracts.RequestParam, err error) {
	var _id ID
	var pathId string
	var out ListRequestWithID[ID] = ListRequestWithID[ID]{ListRequest: ListRequest{}}
	if p, e := out.ListRequest.RequestLoad(ctx); e != nil {
		return param, errors.RequestLoadFailed[ListRequestWithID[ID]](errors.Wrapf(e, "failed to load %T from request", r))
	} else if pathId = string(ctx.Param("id", "")); pathId == "" {
		return param, errors.Errorf("could not load path parameter value with key:id")
	} else {
		out.ListRequest = p.(ListRequest)
	}
	switch any(_id).(type) {
	case string:
		out.setId(pathId)
	case uuid.UUID:
		if uid, parseErr := uuid.Parse(pathId); parseErr != nil {
			return param, errors.Wrapf(parseErr, "could not parse path parameter value with key:id to %T", _id)
		} else {
			out.setId(uid)
		}
	case int64, uint64:
		i, _ := strconv.ParseInt(pathId, 10, 64)
		out.setId(i)
	default:
		i, _ := strconv.Atoi(pathId)
		out.setId(i)
	}
	if ctxSetter, ok := ctx.(interface{ Set(string, interface{}) }); ok {
		ctxSetter.Set(out.ContextKey(), out)
	}
	return out, err
}
