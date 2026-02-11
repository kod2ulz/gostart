# Authentication and Authorization Guide

This guide explains how to implement authentication and authorization in your gostart-based application.

## Overview

gostart provides **minimal building blocks** for authentication:
- `User` interface - a minimal user representation with `GetID()` method
- `SessionUser[ID]` interface - extends User for type-safe user retrieval
- `GetUser()` and `GetSessionUser[ID, U]()` - helpers to retrieve users from context
- `ContextAuthUserKey` constant - standard key for storing users in context

**The actual authentication logic, user models, and authorization guards are implemented in your project**, not in gostart.

## Why This Design?

1. **Not every project needs auth** - Some services are public APIs
2. **Different auth strategies** - Tokens, session cookies, API keys, OAuth, etc.
3. **Different user models** - Some need roles, others need permissions, some need both
4. **Framework-agnostic** - Works with any web framework (Gin, Echo, Fiber, etc.)

## Quick Start

### 1. Define Your User Type

In your project, create a user type that implements `auth.User`:

```go
package myapp

import (
    "github.com/google/uuid"
    "github.com/kod2ulz/gostart/auth"
)

// AuthUser represents your authenticated user
type AuthUser interface {
    auth.User // Embed for GetID() method

    // Add your application-specific methods
    GetEmail() string
    GetRoles() []string
    GetPermissions() []string
    HasRole(role string) bool
    HasPermission(perm string) bool
}

// User implements AuthUser
type User struct {
    ID       uuid.UUID
    Email    string
    Roles    []string
    Permissions []string
}

func (u User) GetID() uuid.UUID {
    return u.ID
}

func (u User) GetEmail() string {
    return u.Email
}

func (u User) GetRoles() []string {
    return u.Roles
}

func (u User) GetPermissions() []string {
    return u.Permissions
}

func (u User) HasRole(role string) bool {
    for _, r := range u.Roles {
        if r == role {
            return true
        }
    }
    return false
}

func (u User) HasPermission(perm string) bool {
    for _, p := range u.Permissions {
        if p == perm {
            return true
        }
    }
    return false
}
```

**Important**: Use `GetID()` instead of `ID()` to avoid conflicts with struct fields named `ID`.

### 2. Create a Token Verifier

Implement a verifier that validates tokens and returns users:

```go
package myapp

import (
    "context"
    "github.com/kod2ulz/gostart/contracts"
    "github.com/kod2ulz/gostart/ierrors"
)

type TokenVerifier struct {
    tokenService *TokenService
}

func (v *TokenVerifier) VerifyToken(ctx context.Context, realm, token string) (AuthUser, ierrors.Error) {
    // 1. Parse and validate the token (JWT, PASETO, etc.)
    claims, err := v.tokenService.ParseToken(token)
    if err != nil {
        return nil, ierrors.ServiceUnauthorised(err)
    }

    // 2. Load user from database
    user, err := v.tokenService.GetUserByID(ctx, claims.UserID)
    if err != nil {
        return nil, ierrors.ServiceUnauthorised(err)
    }

    // 3. Return the user
    return user, nil
}
```

### 3. Create Authentication Middleware

```go
package myapp

import (
    "github.com/kod2ulz/gostart/api"
    "github.com/kod2ulz/gostart/contracts"
    gstartAuth "github.com/kod2ulz/gostart/auth"
    "github.com/kod2ulz/gostart/errors"
    "github.com/kod2ulz/gostart/ierrors"
)

func WithUser(verifier TokenVerifier) api.MiddlewareFunc {
    return func(ctx contracts.RequestContext) (bool, error) {
        // 1. Load token from request (Authorization header, query param, or JSON body)
        var req TokenRequest
        loaded, err := req.RequestLoad(ctx)
        if err != nil {
            return false, errors.RequestLoadFailed[TokenRequest](err)
        }

        // 2. Type assert to concrete type
        var ok bool
        if req, ok = loaded.(TokenRequest); !ok {
            return false, errors.RequestLoadFailed[TokenRequest](
                errors.Errorf("failed to cast loaded token request to %T", req),
            )
        }

        // 3. Validate the request
        if validateErr := req.Validate(ctx); validateErr != nil {
            return false, errors.ValidatorError[TokenRequest](validateErr)
        }

        // 4. Verify the token and get the user
        user, verifyErr := verifier.VerifyToken(ctx.Context(), req.Realm, req.Token)
        if verifyErr != nil {
            return false, verifyErr
        }

        // 5. Store the user in context
        if ctxSetter, ok := ctx.(interface{ Set(string, interface{}) }); ok {
            ctxSetter.Set(gstartAuth.ContextAuthUserKey, user)
        }

        return true, nil
    }
}
```

### 4. Create Authorization Guards

```go
// Role-based authorization
func WithRoles(roles ...string) api.MiddlewareFunc {
    return func(ctx contracts.RequestContext) (bool, error) {
        user, err := gstartAuth.GetUser(ctx.Context())
        if err != nil {
            return false, errors.ServiceUnauthorised(err)
        }

        // Type assert to your AuthUser type
        if authUser, ok := user.(interface{ HasRole(string) bool }); ok {
            if !authUser.HasRole(roles...) {
                return false, errors.ServiceForbidden(
                    errors.Errorf("insufficient roles: required one of %v", roles),
                )
            }
        }

        return true, nil
    }
}

// Permission-based authorization
func WithPermissions(permissions ...string) api.MiddlewareFunc {
    return func(ctx contracts.RequestContext) (bool, error) {
        user, err := gstartAuth.GetUser(ctx.Context())
        if err != nil {
            return false, errors.ServiceUnauthorised(err)
        }

        // Type assert to your AuthUser type
        if authUser, ok := user.(interface{ HasPermission(string) bool }); ok {
            if !authUser.HasPermission(permissions...) {
                return false, errors.ServiceForbidden(
                    errors.Errorf("insufficient permissions: required one of %v", permissions),
                )
            }
        }

        return true, nil
    }
}
```

### 5. Use in Your Routes

```go
package main

import (
    "github.com/kod2ulz/gostart/app"
    "github.com/yourproject/api/auth"
    myapp "github.com/yourproject/auth"
)

func main() {
    application := app.Init()
    router := application.R()

    // Create token verifier
    tokenVerifier := myapp.NewTokenVerifier(tokenService)

    // Apply authentication middleware
    router.Group("/api", auth.WithUser(tokenVerifier)).
        GET("/users", auth.WithPermissions("user.list"), listUsersHandler).
        POST("/users", auth.WithPermissions("user.create"), createUserHandler).
        GET("/admin", auth.WithRoles("admin"), adminHandler)

    application.Run()
}
```

## Alternative: Session Cookie Authentication

To support session cookies instead of tokens:

```go
type CookieVerifier struct {
    sessionService *SessionService
}

func (v *CookieVerifier) VerifyToken(ctx context.Context, realm, token string) (AuthUser, error) {
    // Token parameter is ignored for cookie-based auth

    // 1. Get session cookie from request
    sessionID, err := getSessionCookie(ctx)
    if err != nil {
        return nil, err
    }

    // 2. Load user from session store
    user, err := v.sessionService.Get(ctx, sessionID)
    if err != nil {
        return nil, err
    }

    return user, nil
}

// Usage is exactly the same!
router.Group("/api", auth.WithUser(&CookieVerifier{sessions: sessionService}))
```

## Advanced: Type-Safe User Retrieval

For typed user retrieval in your handlers:

```go
func (s *Service) UpdateUser(ctx contracts.RequestContext, req UpdateUserRequest) (User, error) {
    // Type-safe user retrieval
    user, err := gstartAuth.GetSessionUser[uuid.UUID, *User](ctx.Context())
    if err != nil {
        return User{}, err
    }

    // Now you have full type safety
    fmt.Printf("User email: %s", user.Email)

    // ... business logic
    return s.db.UpdateUser(ctx, req), nil
}
```

## Best Practices

### 1. Method Naming

- **Use `GetID()` instead of `ID()`** - Avoids conflicts with struct fields
- Example: `user.GetID()` instead of `user.ID()`

### 2. Interface Composition

Embed `auth.User` in your interface to get the `GetID()` method:

```go
type AuthUser interface {
    auth.User // Provides GetID()
    GetEmail() string
    HasRole(role string) bool
}
```

### 3. Type Assertions

When retrieving users, always type assert to your concrete type before calling your methods:

```go
user, err := auth.GetUser(ctx)
if err != nil {
    return err
}

// Type assert to access your methods
if authUser, ok := user.(interface{ HasRole(string) bool }); ok {
    if !authUser.HasRole("admin") {
        return errors.ServiceForbidden("missing role: admin")
    }
}
```

### 4. Error Handling

The middleware automatically handles these errors:

- **401 Unauthorized** - Token missing, invalid, or verification failed
- **403 Forbidden** - User authenticated but lacks required role/permission
- **400 Bad Request** - Token request validation failed

All errors follow the gostart error response format.

## Migration from ID() to GetID()

If you have existing code using `ID()`, update it:

```go
// Before
func (u User) ID() uuid.UUID {
    return u.ID
}

// After
func (u User) GetID() uuid.UUID {
    return u.ID
}
```

And update call sites:

```go
// Before
userID := user.ID()

// After
userID := user.GetID()
```

This change prevents conflicts when your struct has a field named `ID`.

## Accessing the Authenticated User

### Using ctx.GetUser() (Simple)

The `RequestContext` interface now has a convenient `GetUser()` method:

```go
func (s *Service) SomeHandler(ctx contracts.RequestContext) (Response, error) {
    // Get user from context - returns nil if not authenticated
    if user := ctx.GetUser(); user != nil {
        // Type assert to access auth.User methods
        if u, ok := user.(auth.User); ok {
            userID := u.GetID()
            // Use userID...
        }
    }

    // Your handler logic...
    return Response{}, nil
}
```

**Important**: `ctx.GetUser()` returns `any` type. You must type assert it to access methods:
- `auth.User` - for `GetID()` method
- Your concrete user type - for application-specific methods

### Using GetSessionUser (Type-Safe)

For type-safe user retrieval in your handlers:

```go
func (s *Service) UpdateUser(ctx contracts.RequestContext, req UpdateUserRequest) (User, error) {
    // Type-safe user retrieval
    user, err := auth.GetSessionUser[uuid.UUID, *User](ctx.Context())
    if err != nil {
        return User{}, err
    }

    // Now you have full type safety
    fmt.Printf("User email: %s", user.Email)

    // ... business logic
    return s.db.UpdateUser(ctx, req), nil
}
```

### When is User Nil?

`ctx.GetUser()` returns `nil` when:
1. **No auth middleware ran** - Route doesn't require authentication
2. **Auth middleware failed** - Token invalid, expired, or verification failed
3. **User not authenticated** - No valid token provided

Always check for nil before using the user!
