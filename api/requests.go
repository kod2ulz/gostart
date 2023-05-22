package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/utils"
	"github.com/pkg/errors"
)

type ListRequest struct {
	User   User `validate:"required"`
	Limit  int32    `validate:"required,gte=1"`
	Offset int32    `validate:"omitempty,gte=0"`
	RequestModal[ListRequest]
}

func (r ListRequest) Metadata() *Metadata {
	return &Metadata{
		Current: int64(r.Offset), Limit: int64(r.Limit), Offset: int64(r.Offset),
	}
}

func (r ListRequest) RequestLoad(ctx context.Context) (param RequestParam, err error) {
	var out ListRequest = ListRequest{}
	if out.User, err = GetUser(ctx); err != nil {
		return
	}
	out.Limit = int32(out.Query(ctx, "limit", "20").Int())
	out.Offset = int32(out.Query(ctx, "offset", "0").Int())
	ctx.(*gin.Context).Set(out.ContextKey(), &out)
	return out, err
}

type ListRequestWithID[ID string | uuid.UUID] struct {
	ID ID `validate:"required"`
	*ListRequest
	*RequestModal[ListRequestWithID[ID]]
}

func (r ListRequestWithID[ID]) RequestLoad(ctx context.Context) (param RequestParam, err error) {
	var out ListRequestWithID[ID] = ListRequestWithID[ID]{
		ListRequest: &ListRequest{},
	}
	var e error
	var p RequestParam
	if p, e = out.ListRequest.RequestLoad(ctx); e != nil {
		return param, RequestLoadError[ListRequestWithID[ID]](errors.Wrapf(e, "failed to load %T from request", r))
	}
	if e = p.Validate(out.ListRequest.InContext(ctx, *out.ListRequest)); e != nil {
		return param, ValidatorError[ListRequestWithID[ID]](errors.Wrapf(e, "validation failed for %T", r))
	}
	out.ListRequest = p.(*ListRequest)
	if id := ctx.(*gin.Context).Param("id"); id == "" {
		return param, errors.Errorf("could not load path parameter value with key:id")
	} else {
		utils.StructCopy(id, &out.ID)
	}
	ctx.(*gin.Context).Set(out.ContextKey(), &out)

	return out, err
}