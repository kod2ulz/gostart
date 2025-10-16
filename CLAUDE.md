# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Testing
- **Run all tests**: `go test ./...`
- **Run test suites with Ginkgo**: Uses Ginkgo/Gomega for BDD-style testing
- **Run specific test suite**: `go test ./api` or `go test ./services/auth`
- **Run with verbose output**: `go test -v ./...`

### Building
- **Build the project**: `go build ./...`
- **Build specific package**: `go build ./api`

### Code Quality
- **Format code**: `go fmt ./...`
- **Tidy dependencies**: `go mod tidy`
- **Vendor dependencies**: `go mod vendor`

## High-Level Architecture

GoStart is an opinionated Go web framework bootstrap library that provides a standardized structure for building backend services. The architecture follows a layered approach with clear separation of concerns.

### Core Components

**Application Bootstrap (`app/`)**
- Central application lifecycle management via `app.Init()`
- Global singleton pattern for configuration, logging, and routing
- Graceful shutdown handling with SIGTERM/SIGINT support
- Automatic Consul service registration when configured
- Built-in health check endpoints (`/ok`, `/stats`)

**API Layer (`api/`)**
- Wrapper around Gin framework with standardized request/response handling
- Custom `Context` type providing helper methods like `Success()`, `Failure()`, `Bind()`
- Framework-agnostic `RequestContext` interface for potential future framework swapping
- Middleware for logging, recovery, authentication, and CORS
- Standardized error response models and validation

**Configuration System (`config/`)**
- Hierarchical configuration loading: Vault → Database → YAML → Environment → Default
- Environment variable loading with `.env` file support
- Consul integration for service discovery and configuration
- HashiCorp Vault integration for secrets management

**Authentication (`auth/`)**
- Generic session management with pluggable providers
- Top-level auth package for user interface and types

**Framework Agnostic Router (`router/`)**
- Framework-agnostic router interface for HTTP routing
- RouterFactory pattern for creating routers with different frameworks
- Configurable CORS, middleware, and static file serving
- Support for multiple HTTP frameworks through adapters

**Framework Adapters (`frameworks/`)**
- Gin router implementation in `frameworks/gin`
- Registry pattern for framework registration
- Framework-specific implementations separated from core application logic
- Easy to add new framework support

**Error Handling (`errors/`, `ierrors/`)**
- Comprehensive error handling with `ErrorModel[T any]` types
- Chainable error methods for error context
- HTTP status code integration
- Replacement for `github.com/pkg/errors` throughout the project

**Request/Response Contracts (`contracts/`)**
- Framework-agnostic interfaces for request handling
- `RequestParam` interface for request loading and validation
- `RequestContext` interface for HTTP context operations
- Standardized response types and helper functions
- JWT/PASETO token support
- Interfaces designed for multiple authentication backends
- User session management with claims and token refresh

## Recent Architectural Refactoring

The project has undergone a major architectural refactoring to achieve true framework agnosticism and proper package separation:

### Completed Refactoring Phases

**Phase 1: Error Handling Decoupling**
- Created `ierrors/` package with base Error interface
- Created `errors/` package with generic `ErrorModel[T any]`
- Removed all `github.com/pkg/errors` dependencies
- Replaced with gostart error functions throughout the project

**Phase 2: Router Abstraction**
- Created `router/` package with framework-agnostic router interface
- Moved router types from `app/` to separate `router/` package
- Created `frameworks/` package for framework-specific implementations
- Moved Gin router implementation to `frameworks/gin/`
- Eliminated import cycles between app and frameworks packages

**Phase 3: Tag-Driven Request Loading**
- Implemented reflection-based request parameter loading
- Created `contracts/` package with request/response interfaces
- Automatic loading from JSON, query params, URL params, and headers
- Struct tag support for request binding (`json:`, `query:`, `param:`, `header:`)

**Phase 4: Automated Documentation**
- Created `docs/` package with OpenAPI 3.0 documentation generation
- Zero-annotation documentation from code structure and tags
- Automatic discovery of routes, request/response types, and field metadata

### Key Architecture Benefits

- **Framework Independence**: Core application logic is completely decoupled from Gin
- **Clean Separation**: Framework-specific code isolated in `frameworks/` directory
- **Easy Extension**: Simple to add support for other web frameworks (Echo, Fiber, etc.)
- **Type Safety**: Strong typing throughout with proper interface contracts
- **Zero Dependencies**: Core packages have minimal external dependencies

### Package Dependency Hierarchy

```
ierrors/ (zero external deps)
    ↑
errors/ (depends on ierrors)
    ↑
contracts/ (depends on errors)
    ↑
api/ (depends on contracts)
    ↑
router/ (depends on contracts)
    ↑
frameworks/gin/ (depends on router, api/ginadapter)
    ↑
app/ (depends on router, frameworks)
```

**Message Queue (`mq/`)**
- RabbitMQ integration with connection management
- Generic worker patterns for background job processing
- Publisher-subscriber pattern support
- Error handling and retry mechanisms
- Context-aware message processing

**Database & Query (`query/`, `storage/`)**
- PostgreSQL integration via pgx v5
- Query builders and criteria patterns
- Redis caching abstraction
- Database configuration and connection management
- Support for complex query conditions and pagination

**Logging (`logr/`)**
- Structured logging using Logrus
- Configurable output formats and levels
- Context-aware logging with field injection
- Integration with pgx for database logging
- Audit logging capabilities

### Key Design Patterns

**Convention over Configuration**: Sensible defaults are provided throughout the framework, reducing boilerplate while maintaining flexibility.

**Dependency Injection**: Services and components are designed to be easily injectable and testable, with context-based dependency passing.

**Error Handling**: Standardized error models with proper HTTP status code mapping and internationalization support.

**Middleware Architecture**: Cross-cutting concerns like authentication, logging, and CORS are handled through middleware.

**Generic Types**: Extensive use of Go generics to provide type-safe interfaces while maintaining flexibility.

### Package Structure

```
├── app/           # Application bootstrap and lifecycle
├── api/           # HTTP API layer and request handling
├── auth/          # Authentication interfaces and JWT utilities
├── collections/   # Thread-safe data structures
├── config/        # Configuration management
├── errors/        # Error handling and models
├── http/          # HTTP client utilities
├── logr/          # Structured logging
├── mq/            # Message queue integration
├── query/         # Database query builders
├── services/      # Business logic services (auth, etc.)
├── storage/       # Caching and persistence
└── utils/         # Common utilities
```

### Development Workflow

1. **Initialize Application**: Start with `app.Init()` to set up global state
2. **Define Routes**: Use the Gin router via `app.R()`
3. **Create Handlers**: Implement handlers using `api.Context` for standardized responses
4. **Add Business Logic**: Place business logic in `services/` package
5. **Configure**: Use hierarchical configuration system
6. **Test**: Write Ginkgo/Gomega tests following existing patterns

### Testing Strategy

- **BDD-style tests** using Ginkgo/Gomega framework
- **Test suites** organized by package with `_suite_test.go` files
- **Integration tests** using testcontainers for PostgreSQL and Vault
- **Mock-friendly** interfaces throughout the codebase
- **Test utilities** for common test scenarios (user registration, authentication)

### Security Considerations

- JWT/PASETO token validation and refresh patterns
- CORS configuration with environment variable support
- Secure configuration loading with Vault integration
- Input validation and sanitization through request models
- Audit logging capabilities throughout the framework

## New Architecture Usage Examples

### Basic Application Setup

```go
package main

import (
    "github.com/kod2ulz/gostart/app"
    "github.com/kod2ulz/gostart/frameworks/gin"
)

func main() {
    // 1. Initialize the router framework
    gin.Setup()

    // 2. Initialize the application
    application := app.Init(
        app.WithHeartbeatHandlers(),
    )

    // 3. Run the application
    application.Run()
}
```

### Custom Router Configuration

```go
gin.SetupWithOptions(func(config *router.RouterConfig) {
    config.AllowOrigins = []string{"*"}
    config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE"}
    config.EnableRecovery = true
    config.EnableLogging = true
    config.CustomMiddleware = []router.MiddlewareFunc{
        // Add custom middleware here
    }
})
```

### Framework-Agnostic Handler

```go
func MyHandler(ctx contracts.RequestContext) {
    // Handler works with any framework
    ctx.JSON(http.StatusOK, map[string]interface{}{
        "message": "Hello from framework-agnostic handler",
    })
}
```

### Using the New Router

```go
// In your application setup
router := app.R()

// Add routes using framework-agnostic interface
router.GET("/users", func(ctx contracts.RequestContext) {
    // Your handler logic here
})

router.POST("/users", func(ctx contracts.RequestContext) {
    // Your handler logic here
})
```