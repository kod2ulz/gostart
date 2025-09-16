package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/api/ginadapter"
	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/ierrors"
)

const ContextAuthUserKey = "auth.User"

// loadParamFromRequest loads and validates request parameters from gin context
func loadParamFromRequest[P contracts.RequestParam](ctx *gin.Context) (param P, err ierrors.Error) {
	var e error
	var p contracts.RequestParam
	if p, e = (*new(P)).RequestLoad(ginadapter.NewRequestContext(ctx)); e != nil {
		return param, errors.RequestLoadError[P](errors.Wrapf(e, "failed to load %T from request", param))
	}
	ctx.Set(p.ContextKey(), p)
	if e = p.Validate(ginadapter.NewRequestContext(ctx)); e != nil {
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
		ctx := ginadapter.NewRequestContext(c)
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
