# Auth Package 🔐

The `auth` package provides **internal authentication interfaces and utilities** for GoStart applications. This is a foundational package that defines the core contracts and types needed for authentication, rather than a complete authentication service.

## Philosophy

The auth package is built around **flexible interfaces and extensibility**:

- **Interface-First**: Defines contracts that can be implemented by any authentication provider
- **Framework Integration**: Provides utilities for integrating with web frameworks
- **Generic Patterns**: Uses Go generics to support various user types and authentication strategies
- **Minimal Foundation**: Provides just the essentials needed to build custom authentication systems

## Quick Start

The auth package provides interfaces and types that you can use to build custom authentication systems:

```go
import "github.com/kod2ulz/gostart/auth"

// Define your user type implementing the User interface
type AppUser struct {
    ID    uuid.UUID
    Email string
    Name  string
}

func (u *AppUser) ID() uuid.UUID {
    return u.ID
}

// Use the generic session service
type UserService struct {
    // Your custom authentication logic
}

func (s *UserService) Verify(ctx context.Context) (AppUser, ierrors.Error) {
    // Implement your verification logic
    return AppUser{}, nil
}

// Create a session service
userService := auth.SessionService("user-store", "auth-service")
auther := userService.Auther()
```

## Implementation Status

### ⚠️ **Current State: Interface Definitions Only**

**IMPORTANT:** This package provides only interface definitions and basic structures. The actual authentication logic is NOT implemented.

### ✅ **Implemented Interfaces and Types**
- **User Interface** - Generic user contract with ID() method
- **SessionService Interface** - Define session management contracts
- **Auther Interface** - Authentication verification contract
- **Request/Response Types** - Authentication request and response structures
- **Middleware Framework** - Basic middleware structure for authentication
- **Gin Context Adapter** - Framework-specific context adapter

### ❌ **NOT Implemented (Stubs Only)**
- **JWT/PASETO Token Validation** - Interface exists, returns "not implemented"
- **Token Refresh Logic** - Interface exists, returns "not implemented"
- **Session Management** - Basic structure exists, no actual session logic
- **User Store Implementation** - Interface exists, no backing implementation
- **Password Verification** - Interface exists, returns "not implemented"
- **Authentication Middleware** - Structure exists, minimal functionality

### 🚧 **What You Need to Implement**
To use this package for authentication, you must implement:
- User store and retrieval logic
- Token generation and validation (JWT/PASETO/custom)
- Session management and persistence
- Password hashing and verification
- Authentication business logic

## Core Features

### **User Interface**

The package provides a minimal `User` interface that must be implemented by your user types:

```go
type User interface {
    ID() uuid.UUID
}

// Example implementation
type AppUser struct {
    UserID uuid.UUID
    Email  string
    Name   string
}

func (u *AppUser) ID() uuid.UUID {
    return u.UserID
}
```

### **Generic Session Service**

The package includes a generic session service interface that you can implement for your authentication needs:

```go
// Generic service interface
type GenericSessionService[ID comparable, U SessionUser[ID]] struct{}

// Methods to implement
func (s *GenericSessionService[ID, U]) Verify(ctx context.Context) (U, ierrors.Error)
func (s *GenericSessionService[ID, U]) Login(ctx context.Context) (TokenResponse, ierrors.Error)
func (s *GenericSessionService[ID, U]) Refresh(ctx context.Context) (TokenResponse, ierrors.Error)
func (s *GenericSessionService[ID, U]) Signup(ctx context.Context) (U, ierrors.Error)
```

### **Request Parameter Types**

The package provides request parameter types for common authentication operations:

```go
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
```

### **Configuration Support**

The package supports configuration for different authentication providers:

```go
type Config struct {
    Driver             string        // Currently supports "cognito"
    UserPool           string        // Cognito user pool ID
    ClientID           string        // Client ID
    ClientSecret       string        // Client secret
    AuthIssuerURL      string        // Issuer URL
    JwkRefreshInterval config.Value   // JWKS refresh interval
    PublicKeyURL       string        // Public key URL for token validation
}

// Initialize configuration from environment variables
config := auth.InitConfig("AUTH_")
```

### **Gin Framework Integration**

The package provides utilities for integrating with Gin framework:

```go
// Middleware for user authentication
func WithUser[TokenRequest contracts.RequestParam, UserResponse SessionUser[uuid.UUID], TokenResponse any](
    svc *GenericSessionService[uuid.UUID, UserResponse],
) gin.HandlerFunc

// Usage in Gin routes
userService := auth.SessionService("store", "service")
router.Use(auth.WithUser[LoginRequest, AppUser, TokenResponse](userService))
```

## Integration with GoStart

The auth package integrates with other GoStart components:

- **[`api`](../api/README.md)**: Request parameter handling and validation
- **[`errors`](../errors/README.md)**: Enhanced error handling for auth failures
- **[`config`](../config/README.md)**: Configuration management for auth settings
- **[`contracts`](../contracts/README.md)**: Request parameter interfaces

### Usage Example

Here's a complete example of how to use the auth package to create a simple authentication system:

```go
package main

import (
    "context"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/kod2ulz/gostart/auth"
    "github.com/kod2ulz/gostart/contracts"
    "github.com/kod2ulz/gostart/errors"
)

// Define your user type
type AppUser struct {
    UserID uuid.UUID
    Email  string
    Name   string
    Active bool
}

func (u *AppUser) ID() uuid.UUID { return u.UserID }
func (u *AppUser) GetID() uuid.UUID { return u.UserID }

// Implement the SessionUser interface
func (u AppUser) SessionUser[uuid.UUID] {}

// Create your authentication service
type AuthService struct {
    // Your authentication logic here
}

func (s *AuthService) Verify(ctx context.Context) (AppUser, ierrors.Error) {
    // Implement your verification logic
    return AppUser{}, nil
}

func (s *AuthService) Login(ctx context.Context) (auth.TokenResponse, ierrors.Error) {
    // Implement your login logic
    return auth.TokenResponse{}, nil
}

func main() {
    r := gin.Default()

    // Create authentication service
    authService := &AuthService{}
    sessionService := auth.SessionService("user-store", authService)

    // Add authentication middleware
    r.Use(auth.WithUser[auth.LoginRequest, AppUser, auth.TokenResponse](sessionService))

    // Protected routes
    r.GET("/profile", func(c *gin.Context) {
        // User will be available in context
        c.JSON(200, gin.H{"message": "authenticated"})
    })

    r.Run(":8080")
}
```

## Implementation Status

### Currently Implemented ✅
- User interface definition
- Generic session service interfaces
- Request parameter types for common auth operations
- Configuration support for Cognito
- Gin framework middleware integration
- Context utilities for user management

### Not Implemented ❌
- JWT/PASETO token providers
- Session storage implementations
- Role-based access control
- Built-in authentication providers
- Token validation and refresh
- User management functions

## When to Use This Package

### Perfect For:
- **Custom Authentication Systems**: When you need to build your own auth logic
- **Framework Integration**: When you need auth utilities for Gin or other frameworks
- **Type-Safe Authentication**: When you want Go generics for your user types
- **Minimal Foundation**: When you don't need a full auth service

### Consider Alternatives For:
- **Complete Auth Solutions**: Use a full auth service like Auth0, Firebase Auth
- **Quick Setup**: Use existing authentication libraries
- **Production Systems**: This package is still in development

## Roadmap

Future enhancements for the auth package:

- **JWT/PASETO Support**: Built-in token generation and validation
- **Session Storage**: Redis, memory, and database session stores
- **Role-Based Access**: Built-in role and permission management
- **More Providers**: Support for OAuth2, SAML, and other protocols
- **Enhanced Middleware**: More sophisticated middleware patterns

The auth package provides the foundational interfaces and utilities for building custom authentication systems in GoStart applications.