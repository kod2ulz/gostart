package api

import (
	"context"

	"github.com/kod2ulz/gostart/ierrors"
)

type SessionService[IdentityResponse, TokenResponse any] interface {
	Verify(context.Context) (IdentityResponse, ierrors.Error)
	Login(context.Context) (TokenResponse, ierrors.Error)
	Refresh(context.Context) (TokenResponse, ierrors.Error)
}

type RegistrationService[IdentityResponse any] interface {
	Signup(context.Context) (IdentityResponse, ierrors.Error)
}