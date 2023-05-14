package auth

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/pkg/errors"
)

func GetUser(ctx context.Context) (User, error) {
	if authUser := ctx.Value(ContextAuthUserKey); authUser == nil {
		return nil, errors.Errorf("invalid context. unauthorised")
	} else {
		return authUser.(User), nil
	}
}

type User interface {
	ID() uuid.UUID
	Username() string
	Scope() []string
	Token() *jwt.Token
	CountryID() uuid.UUID
}

type user struct {
	id       uuid.UUID
	username string
	scope    []string
	token    *jwt.Token
}

func (u *user) ID() uuid.UUID {
	return u.id
}

func (u *user) Username() string {
	return u.username
}

func (u *user) Scope() []string {
	return u.scope
}

func (u *user) Token() *jwt.Token {
	return u.token
}

func tokenUser(token jwt.Token) *user {
	usr := &user{
		id:    uuid.MustParse(token.Subject()),
		token: &token,
		scope: []string{},
	}
	if username, ok := token.Get("username"); ok {
		usr.username = username.(string)
	}
	if scope, ok := token.Get("scope"); ok && scope != nil && scope.(string) != "" {
		usr.scope = strings.Split(scope.(string), " ")
	}
	return usr
}
