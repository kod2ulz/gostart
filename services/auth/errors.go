package auth

import (
	"github.com/kod2ulz/gostart/api"
	"github.com/pkg/errors"
)

var (
	ErrLoginInvalid        = errors.New("invalid username or password")
	ErrLoginDisabled       = errors.New("this user has been disabled")
	ErrUsernameTaken       = errors.New("this username is not available")
	ErrTokenValidation     = errors.New("token validation failed")
	ErrUserNotFound        = errors.New("user not found")
	ErrStoreNotInitialized = errors.New("store not initialised")
	ErrInvalidID           = errors.New("invalid format for id")
)

var (
	StatusErrorCreation = "CreationError"
)

var (
	ServiceStatusUnauthorized   = "InvalidCredentials"
	ServiceErrorGeneratingToken = func(err error) api.Error {
		return api.ServerError(err).WithErrorCode("LoginFailed")
	}
)
