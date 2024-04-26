package api

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var (
	ContextAuthUserKey = "auth.User"
	Authorization      = "Authorization"
)

func GetUser(ctx context.Context) (User, error) {
	if user := ctx.Value(ContextAuthUserKey); user == nil {
		return nil, errors.Errorf("invalid context. unauthorised")
	} else {
		return user.(User), nil
	}
}

type User interface {
	ID() uuid.UUID
	// GetUsername() string
}
