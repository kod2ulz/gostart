package api

import (
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/api/ginadapter"
	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/errors"
	"golang.org/x/exp/constraints"
)

type ListRequest struct {
	Limit  int32 `validate:"required,gte=1"`
	Offset int32 `validate:"omitempty,gte=0"`
	contracts.RequestModal[ListRequest]
}

func (r ListRequest) Metadata() (out *contracts.Metadata) {
	out = &contracts.Metadata{
		// Current: int64(r.Offset),
		Limit: int64(r.Limit), Offset: int64(r.Offset),
	}
	if out.Offset > 0 && out.Limit > 0 {
		out.Page = (out.Offset / out.Limit) + 1
	}
	return
}

func (r ListRequest) DefaultMetadata(ctx contracts.RequestContext) (out *contracts.Metadata) {
	out = r.Metadata()
	r.SetResponseMetadata(ctx, out)
	return
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
	if page := query("page", 1); page > 1 && out.Offset == 0 {
		out.Offset = out.Limit * (page - 1)
	}
	if ginCtx, ok := ctx.(*ginadapter.GinRequestContext); ok {
		ginCtx.Set(out.ContextKey(), &out)
	}
	return out, err
}



type ListRequestIdType interface {
	string | uuid.UUID | constraints.Integer
}

type ListRequestWithID[ID ListRequestIdType] struct {
	ID ID `validate:"required"`
	ListRequest
	contracts.RequestModal[ListRequestWithID[ID]]
}

func (r *ListRequestWithID[ID]) setId(id interface{}) {
	r.ID = id.(ID)
}

func (r ListRequestWithID[ID]) RequestLoad(ctx contracts.RequestContext) (param contracts.RequestParam, err error) {
	var _id ID
	var pathId string
	var out ListRequestWithID[ID] = ListRequestWithID[ID]{ListRequest: ListRequest{}}
	if p, e := out.ListRequest.RequestLoad(ctx); e != nil {
		return param, errors.RequestLoadError[ListRequestWithID[ID]](errors.Wrapf(e, "failed to load %T from request", r))
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
	if ginCtx, ok := ctx.(*ginadapter.GinRequestContext); ok {
		ginCtx.Set(out.ContextKey(), out)
	}
	return out, err
}
