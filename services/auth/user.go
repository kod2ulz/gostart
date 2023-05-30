package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/collections"
	"github.com/pkg/errors"
)

type UserData struct {
	UID        uuid.UUID  `json:"id"`
	Email      string     `json:"email"`
	Password   string     `json:"-"`
	InvitedBy  *uuid.UUID `json:"invitedBy,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	DisabledAt *time.Time `json:"disabledAt,omitempty"`
}

// ID implements auth.User
func (u UserData) ID() uuid.UUID {
	return u.UID
}

// GetID implements SessionUser
func (u UserData) GetID() uuid.UUID {
	return u.UID
}

// GetPassword implements SessionUser
func (u UserData) GetPassword() string {
	return u.Password
}

// GetUsername implements SessionUser
func (u UserData) GetUsername() string {
	return u.Email
}

// IsDisabled implements SessionUser
func (u UserData) IsDisabled() bool {
	return u.DisabledAt != nil && !u.DisabledAt.IsZero()
}

var _ SessionUser[uuid.UUID] = (*UserData)(nil)

type _userStore struct {
	data          collections.Map[uuid.UUID, UserData]
	usernameIndex collections.Map[string, uuid.UUID]
}

func InMemoryUserStore() SessionStore[uuid.UUID, UserData] {
	return &_userStore{data: map[uuid.UUID]UserData{}, usernameIndex: map[string]uuid.UUID{}}
}

// CreateUser implements SessionStore
func (s *_userStore) CreateUser(ctx context.Context, req SignupRequest) (out UserData, err error) {
	if _, ok := s.usernameIndex[req.Username]; ok {
		return out, ErrUsernameTaken
	}
	id := uuid.New()
	s.usernameIndex[req.Username] = id
	s.data[id] = UserData{UID: id, Email: req.Username, Password: req.Password, CreatedAt: time.Now()}
	return s.data[id], nil
}

// GetUserWithID implements SessionStore
func (s *_userStore) GetUserWithID(ctx context.Context, id interface{}) (out UserData, err error) {
	var ok bool
	var idx uuid.UUID
	if val, ok := id.(string); ok {
		idx = uuid.MustParse(val)
	} else if idx, ok = id.(uuid.UUID); !ok {
		return out, errors.Wrapf(ErrInvalidID,  "expected (string) or (%T)", idx)
	}

	if s.data == nil {
		return out, ErrStoreNotInitialized
	} else if out, ok = s.data[idx]; !ok {
		return out, ErrUserNotFound
	}
	return
}

// GetUserWithUsername implements SessionStore
func (s *_userStore) GetUserWithUsername(ctx context.Context, username string) (out UserData, err error) {
	if id, ok := s.usernameIndex[username]; ok {
		return s.GetUserWithID(ctx, id)
	}
	return out, ErrUserNotFound
}

func (s *_userStore) Clear() {
	for k := range s.data {
		delete(s.usernameIndex, s.data[k].Email)
	}
	for k := range s.usernameIndex {
		delete(s.usernameIndex, k)
	}
}

var _ SessionStore[uuid.UUID, UserData] = &_userStore{}

type User struct {
	*UserData
	Claims *Claims
}
