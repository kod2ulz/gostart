package auth

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kod2ulz/gostart/api"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const (
	ContextAuthUserKey         = "auth.User"
	AuthTokenClaimCountryIdKey = "countryId"
)

type Middleware interface {
	WithUser() gin.HandlerFunc
}

type SessionService interface {
	VerifyToken(context.Context) (jwt.Token, error)
}

type VerifyTokenRequest struct {
	Token string `json:"token,omitempty" validate:"required"`
	api.RequestModal[VerifyTokenRequest]
}

func WithUser(svc SessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if req, err := (VerifyTokenRequest{}).RequestLoad(c); err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, c.Error(err))
		} else if err = req.Validate(c); err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, c.Error(err))
		} else {
			c.Set(req.ContextKey(), req.(VerifyTokenRequest))
			if token, err := svc.VerifyToken(c); err != nil {
				c.AbortWithStatusJSON(http.StatusForbidden, c.Error(err))
			} else {
				c.Set(ContextAuthUserKey, tokenUser(token))
				c.Next()
			}
		}
	}
}
