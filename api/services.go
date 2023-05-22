package api

import "context"

type SessionService[IdentityResponse, TokenResponse any] interface {
	Verify(context.Context) (IdentityResponse, Error)
	Login(context.Context) (TokenResponse, Error)
	Refresh(context.Context) (TokenResponse, Error)
}

type RegistrationService[IdentityResponse any] interface {
	Signup(context.Context) (IdentityResponse, Error)
}