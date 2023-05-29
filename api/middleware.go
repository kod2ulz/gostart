package api

import (
	"github.com/gin-gonic/gin"
)

type Middleware interface {
	WithUser() gin.HandlerFunc
}

func WithUser[TokenRequest RequestParam, UserResponse, TokenResponse any](svc SessionService[UserResponse, TokenResponse]) gin.HandlerFunc {
	return func(c *gin.Context) {
		var loadError Error
		var req TokenRequest
		if req, loadError = loadParamFromRequest[TokenRequest](c); loadError != nil {
			c.AbortWithStatusJSON(loadError.http(), ErrorResponse[TokenRequest](loadError))
		} else if validationError := req.Validate(c); loadError != nil {
			e := ServiceErrorUnauthorised(validationError).(*ErrorModel[User])
			c.AbortWithStatusJSON(e.Http, ErrorResponse[User](e))
		} else if user, err := svc.Verify(c); err != nil {
			c.AbortWithStatusJSON(err.(*ErrorModel[User]).Http, ErrorResponse[User](err))
		} else {
			c.Set(ContextAuthUserKey, user)
			c.Next()
		}
	}
}
