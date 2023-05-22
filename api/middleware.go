package api

import (
	"github.com/gin-gonic/gin"
)

type Middleware interface {
	WithUser() gin.HandlerFunc
}

func authAbort[T any](c *gin.Context, err error) {
	e := ServiceErrorUnauthorised(err).(*ErrorModel[T])
	c.AbortWithStatusJSON(e.Http, ErrorResponse[T](e))
}

func WithUser[TokenRequest RequestParam, UserResponse, TokenResponse any](svc SessionService[UserResponse, TokenResponse]) gin.HandlerFunc {
	return func(c *gin.Context) {
		var loadError Error
		var req TokenRequest
		if req, loadError = loadParamFromRequest[TokenRequest](c); loadError != nil {
			c.JSON(loadError.http(), ErrorResponse[TokenRequest](loadError))
		} else if validationError := req.Validate(c); loadError != nil {
			authAbort[User](c, validationError)
		} else if user, err := svc.Verify(c); err != nil {
			c.AbortWithStatusJSON(err.(*ErrorModel[User]).Http, ErrorResponse[User](err))
		} else {
			c.Set(ContextAuthUserKey, user)
			c.Next()
		}
	}
}
