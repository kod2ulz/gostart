package api

import (
	"github.com/gin-gonic/gin"
	"github.com/kod2ulz/gostart/api/ginadapter"
	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/ierrors"
)

const ContextAuthUserKey = "auth.User"

type Middleware interface {
	WithUser() gin.HandlerFunc
}

func WithUser[TokenRequest contracts.RequestParam, UserResponse, TokenResponse any](svc SessionService[UserResponse, TokenResponse]) gin.HandlerFunc {
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
