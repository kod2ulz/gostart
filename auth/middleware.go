package auth

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/ierrors"
)

const ContextAuthUserKey = "auth.User"

// ginContextAdapter provides a minimal RequestContext implementation for gin.Context
type ginContextAdapter struct {
	ctx *gin.Context
}

func (g *ginContextAdapter) Query(key string, defaultValue ...string) contracts.Value {
	if val := g.ctx.Query(key); val != "" {
		return contracts.Value(val)
	} else if len(defaultValue) > 0 {
		return contracts.Value(defaultValue[0])
	}
	return ""
}

func (g *ginContextAdapter) Param(key string, defaultValue ...string) contracts.Value {
	if val := g.ctx.Param(key); val != "" {
		return contracts.Value(val)
	} else if len(defaultValue) > 0 {
		return contracts.Value(defaultValue[0])
	}
	return ""
}

func (g *ginContextAdapter) Header(key string) string {
	return g.ctx.Request.Header.Get(key)
}

func (g *ginContextAdapter) ShouldBindJSON(obj interface{}) error {
	return g.ctx.ShouldBindJSON(obj)
}

func (g *ginContextAdapter) Context() context.Context {
	return g.ctx
}

func (g *ginContextAdapter) Value(key interface{}) interface{} {
	return g.ctx.Value(key)
}

func (g *ginContextAdapter) Set(key string, value interface{}) {
	g.ctx.Set(key, value)
}

// loadParamFromRequest loads and validates request parameters from gin context
func loadParamFromRequest[P contracts.RequestParam](ctx *gin.Context) (param P, err ierrors.Error) {
	var e error
	var p contracts.RequestParam
	adapterCtx := &ginContextAdapter{ctx}
	if p, e = (*new(P)).RequestLoad(adapterCtx); e != nil {
		return param, errors.RequestLoadError[P](errors.Wrapf(e, "failed to load %T from request", param))
	}
	ctx.Set(p.ContextKey(), p)
	if e = p.Validate(adapterCtx); e != nil {
		return param, errors.ValidatorError[P](errors.Wrapf(e, "validation failed for %T", param))
	}
	param = p.(P)
	return
}

type Middleware interface {
	WithUser() gin.HandlerFunc
}

func WithUser[TokenRequest contracts.RequestParam, UserResponse SessionUser[uuid.UUID], TokenResponse any](svc *GenericSessionService[uuid.UUID, UserResponse]) gin.HandlerFunc {
	return func(c *gin.Context) {
		var loadError ierrors.Error
		var req TokenRequest
		ctx := &ginContextAdapter{c}
		if req, loadError = loadParamFromRequest[TokenRequest](c); loadError != nil {
			c.AbortWithStatusJSON(loadError.HttpCode(), contracts.ErrorResponse[TokenRequest](loadError))
		} else if validationError := req.Validate(ctx); validationError != nil {
			e := errors.ServiceErrorUnauthorised(validationError)
			c.AbortWithStatusJSON(e.HttpCode(), contracts.ErrorResponse[UserResponse](e))
		} else if user, err := svc.Verify(c); err != nil {
			c.AbortWithStatusJSON(err.HttpCode(), contracts.ErrorResponse[UserResponse](err))
		} else {
			c.Set(ContextAuthUserKey, user)
			c.Next()
		}
	}
}
