package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/utils"
	"github.com/pkg/errors"
)

type Claims struct {
	Username string `json:"username" validate:"required"`
	Client   string `json:"client"   validate:"required"`
	jwt.RegisteredClaims
}

func (c Claims) Validate(host, client string) (err error) {
	if c.ExpiresAt.Before(time.Now()) {
		return errors.Errorf("token expired")
	} else if c.NotBefore.After(time.Now()) {
		return errors.Errorf("issued before nbf")
	} else if c.IssuedAt.Before(c.NotBefore.Time) {
		return errors.Errorf("iat before nbf")
	} else if c.IssuedAt.After(time.Now()) {
		return errors.Errorf("iat invalid")
	} else if c.Issuer != host {
		return errors.Errorf("iss invalid")
	} else if c.Client != client {
		return errors.Errorf("client invalid")
	} else if _, err = uuid.Parse(c.Subject); err != nil {
		return errors.Errorf("sub invalid")
	}
	return utils.Validate.Struct(c)
}

func (c Claims) User() *UserData {
	return &UserData{
		UID:   uuid.MustParse(c.Subject),
		Email: c.Username,
	}
}

func (c Claims) ID() uuid.UUID {
	return uuid.MustParse(c.Subject)
}

func (c *Claims) AuthUser() *User {
	return &User{
		UserData: &UserData{
			UID:   c.ID(),
			Email: c.Username,
		},
		Claims: c,
	}
}
