package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/kod2ulz/gostart/utils"
	"github.com/pkg/errors"
)

type RequestParam interface {
	Validate(ctx context.Context) error
	RequestLoad(ctx context.Context) (RequestParam, error)
	ContextKey() string
	MetadataContextKey() string
	SetResponseMetadata(context.Context, *Metadata) error
	ContextLoad(context.Context) (RequestParam, error)
}

func ParamsFromContext[P RequestParam](ctx context.Context) (P, error) {
	var err error
	var reqParam RequestParam
	if reqParam, err = (*new(P)).ContextLoad(ctx); err != nil {
		var ctxParam P
		return ctxParam, errors.Wrapf(err, "Failed to load %T from context", ctxParam)
	}
	return reqParam.(P), nil
}

func QueryFromContext(ctx context.Context, name, _default string) utils.Value {
	if v := ctx.(*gin.Context).Query(name); v != "" {
		return utils.Value(v)
	}
	return utils.Value(_default)
}




