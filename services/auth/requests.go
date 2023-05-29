package auth

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/object"
	"github.com/pkg/errors"
)

type passwordHashFn func(string) string

type LoginUser interface {
	GetUsername() string
	GetPassword() string
}

type SignupRequest struct {
	Username string `json:"username" validate:"required,email"`
	Password string `json:"password" validate:"required,password"`
	api.RequestModal[SignupRequest]
}

func (r SignupRequest) WithHash(hasher passwordHashFn) SignupRequest {
	return SignupRequest{
		Username: r.Username, Password: hasher(r.Password),
	}
}

type LoginRequest struct {
	Username string `json:"username" validate:"required,email"`
	Password string `json:"password" validate:"required,password"`
	api.RequestModal[LoginRequest]
}

func (r LoginRequest) verify(user LoginUser, hasher passwordHashFn) bool {
	return user.GetUsername() == r.Username && user.GetPassword() == hasher(r.Password)
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
	api.RequestModal[RefreshRequest]
}

type VerifyTokenRequest struct {
	Token string `json:"token" validate:"required"`
	api.RequestModal[VerifyTokenRequest]
}

func (r VerifyTokenRequest) RequestLoad(ctx context.Context) (param api.RequestParam, err error) {
	var out VerifyTokenRequest 
	authHeader := ctx.(*gin.Context).Request.Header.Get("Authorization")
	if authHeader != "" {
		switch authType := object.String(authHeader).Split(" ").First(); authType {
		case TokenTypeBearer:
			out.Token = strings.TrimPrefix(authHeader, TokenTypeBearer+" ")
			ctx.(*gin.Context).Set(out.ContextKey(), out)
			return out, nil
			// todo: process other auth token types
		}
	} else if out.Token = r.Query(ctx, "token").String(); out.Token != "" {
		ctx.(*gin.Context).Set(out.ContextKey(), out)
		return out, nil
	} else if err = out.LoadFromJsonBody(ctx, &out); err == nil && out.Token != "" {
		ctx.(*gin.Context).Set(out.ContextKey(), out)
		return out, err
	}
	return nil, errors.New("token missing in request")
}
