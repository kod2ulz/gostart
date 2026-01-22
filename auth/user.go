package auth

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/ierrors"
)

// Mock types for testing
type SignupRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type TokenResponse struct {
	AccessToken string `json:"accessToken"`
	Token       string `json:"token"`
}

// Mock types for testing
type UserData struct {
	UserID      uuid.UUID
	Email       string
	Roles       []string
	Realms      []string
	Permissions []string
	ActiveRealm string
}

func (u UserData) GetID() uuid.UUID {
	return u.UserID
}

func (u UserData) HasRole(roles ...string) bool {
	for _, role := range roles {
		if slices.Contains(u.Roles, role) {
			return true
		}
	}
	return false
}

func (u UserData) HasRealm(realms ...string) bool {
	for _, realm := range realms {
		if slices.Contains(u.Realms, realm) {
			return true
		}
	}
	return false
}

func (u UserData) HasPermission(permissions ...string) bool {
	for _, permission := range permissions {
		if slices.Contains(u.Permissions, permission) {
			return true
		}
	}
	return false
}

func (u UserData) GetRealm() string {
	return u.ActiveRealm
}

func InMemoryUserStore() any {
	return "mock-user-store"
}

func SessionService(a any, b any) *GenericSessionService[uuid.UUID, UserData] {
	return &GenericSessionService[uuid.UUID, UserData]{}
}

func (s *GenericSessionService[ID, U]) Auther() any {
	return "mock-auther"
}

type GenericSessionService[ID comparable, U SessionUser[ID]] struct{}

func (s *GenericSessionService[ID, U]) Verify(ctx context.Context) (U, ierrors.Error) {
	return *new(U), errors.Errorf("not implemented")
}

func (s *GenericSessionService[ID, U]) Login(ctx context.Context) (TokenResponse, ierrors.Error) {
	return TokenResponse{}, errors.Errorf("not implemented")
}

func (s *GenericSessionService[ID, U]) Refresh(ctx context.Context) (TokenResponse, ierrors.Error) {
	return TokenResponse{}, errors.Errorf("not implemented")
}

func (s *GenericSessionService[ID, U]) Signup(ctx context.Context) (U, ierrors.Error) {
	// For testing purposes, return a mock user
	var user U
	// This is a simplified implementation for tests
	// In a real implementation, this would create a user in the database
	return user, nil
}

func (r SignupRequest) Validate(ctx contracts.RequestContext) error {
	return fmt.Errorf("not implemented")
}

func (r SignupRequest) ContextKey() string {
	return "auth.SignupRequest"
}

func (r SignupRequest) RequestLoad(ctx contracts.RequestContext) (contracts.RequestParam, error) {
	return r, nil
}

func (r SignupRequest) MetadataContextKey() string {
	return "meta.auth.SignupRequest"
}

func (r SignupRequest) ReferencesContextKey() string {
	return "ref.auth.SignupRequest"
}

func (r SignupRequest) SetResponseMetadata(ctx contracts.RequestContext, meta *contracts.Metadata) error {
	return fmt.Errorf("not implemented")
}

func (r SignupRequest) SetResponseReference(ctx contracts.RequestContext, key string, value any) error {
	return fmt.Errorf("not implemented")
}

func (r SignupRequest) ContextLoad(ctx context.Context) (contracts.RequestParam, error) {
	return r, nil
}

func (r SignupRequest) LoadFromContext(ctx context.Context, out contracts.RequestParam) error {
	return fmt.Errorf("not implemented")
}

func (r LoginRequest) Validate(ctx contracts.RequestContext) error {
	return fmt.Errorf("not implemented")
}

func (r LoginRequest) ContextKey() string {
	return "auth.LoginRequest"
}

func (r LoginRequest) RequestLoad(ctx contracts.RequestContext) (contracts.RequestParam, error) {
	return r, nil
}

func (r LoginRequest) MetadataContextKey() string {
	return "meta.auth.LoginRequest"
}

func (r LoginRequest) ReferencesContextKey() string {
	return "ref.auth.LoginRequest"
}

func (r LoginRequest) SetResponseMetadata(ctx contracts.RequestContext, meta *contracts.Metadata) error {
	return fmt.Errorf("not implemented")
}

func (r LoginRequest) SetResponseReference(ctx contracts.RequestContext, key string, value any) error {
	return fmt.Errorf("not implemented")
}

func (r LoginRequest) ContextLoad(ctx context.Context) (contracts.RequestParam, error) {
	return r, nil
}

func (r LoginRequest) LoadFromContext(ctx context.Context, out contracts.RequestParam) error {
	return fmt.Errorf("not implemented")
}
