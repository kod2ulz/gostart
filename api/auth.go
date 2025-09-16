package api

import (
	"context"

	"github.com/kod2ulz/gostart/auth"
	"github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/ierrors"
)

// GetUser is a convenience wrapper around auth.GetUser for backward compatibility
// Note: This function is not implemented yet - implement based on your authentication strategy
func GetUser(ctx context.Context) (auth.User, ierrors.Error) {
	return nil, errors.Errorf("GetUser not implemented")
}