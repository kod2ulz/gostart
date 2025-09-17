package auth

import (
	"context"

	"github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/ierrors"
)

// GetUser retrieves the authenticated user from context
// Note: This function is not implemented yet - implement based on your authentication strategy
func GetUser(ctx context.Context) (User, ierrors.Error) {
	return nil, errors.Errorf("GetUser not implemented")
}
