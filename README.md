# GoStart: The Opinionated Go Project Bootstrap

[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](https://github.com/kod2ulz/gostart)

**GoStart** is an opinionated Go framework bootstrap library that provides a standardized structure for building backend services. It's designed for developers who want to build production-ready applications quickly without getting lost in configuration decisions, while still maintaining the flexibility to customize when needed.

---

## Philosophy & Vision

GoStart is built on the principle that **convention over configuration** accelerates development, but **flexibility over rigidity** ensures long-term maintainability. We believe that:

- **Structure matters**: A consistent project structure reduces cognitive load and makes codebases easier to navigate
- **Standards save time**: Enforcing best practices for security, error handling, and logging prevents common pitfalls
- **Business logic is king**: Framework and infrastructure concerns should fade into the background
- **Choice is important**: While opinionated, GoStart never locks you into a single ecosystem or approach

The framework provides sensible defaults and patterns that work for 80% of use cases, while allowing deep customization for the remaining 20%.

---

## Overview

GoStart provides a comprehensive foundation for backend services by integrating:

- **Framework-agnostic API layer** with unified interfaces
- **Pluggable authentication** and session management
- **Structured logging** and observability
- **Database abstraction** with intelligent query building
- **Message queue integration** for async processing
- **Configuration management** across environments
- **Thread-safe collections** and utilities

The architecture is designed around **clean separation of concerns**, enabling you to swap components (web frameworks, databases, etc.) without rewriting business logic.

## Features

- **Rapid Setup:** Get a production-ready server running in minutes with a single `app.Init()` call.
- **Unified Response Envelope:** Consistent API response format with support for single objects, lists, pagination, and reference data.
- **Framework-Agnostic Design:** Clean separation between business logic and web framework through unified interfaces.
- **Generic Handlers:** Type-safe handler functions with automatic request parameter loading and validation.
- **Pluggable Authentication:** Comes with helpers for JWT and PASETO tokens, with integrations for providers like AWS Cognito.
- **Multiple Framework Support:** Built-in support for Gin with extensible architecture for Echo, net/http, and other frameworks.
- **Automatic Request Processing:** Declarative request parameter definitions with automatic binding and validation.
- **Structured Logging:** Centralized and configurable logging with automatic request logging middleware.
- **Database & Cache Ready:** Includes helpers and interfaces for PostgreSQL (via pgx v5) and Redis.
- **Message Queuing:** Integrated RabbitMQ publisher and worker system for background jobs.
- **Rich Utilities:** A large collection of helpers for configuration, error handling, concurrent data structures, and more.

## Getting Started

Here's how to bootstrap a new application using GoStart.

```go
package main

import (
	"context"
	"net/http"

	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/api/frameworks/gin"
	"github.com/kod2ulz/gostart/app"
)

// Define a simple request parameter
type HelloRequest struct {
	api.RequestModal[HelloRequest]
	Name string `query:"name" validate:"required"`
}

// Define a simple response type
type HelloResponse struct {
	Message string `json:"message"`
}

func main() {
	// 1. Initialize Gin framework (optional, for framework-specific setup)
	gin.Setup()

	// 2. Initialize the application
	// This sets up the logger, router, and graceful shutdown handling.
	a := app.Init()
	log := a.Log()

	log.Info("Application starting up...")

	// 3. Define a route using the unified router interface
	a.R().GET("/hello", api.Handler[HelloRequest, HelloResponse](func(ctx context.Context, req HelloRequest) (HelloResponse, error) {
		return HelloResponse{
			Message: "Hello, " + req.Name + "!",
		}, nil
	}))

	// 4. Run the application
	// This starts the HTTP server and blocks until shutdown.
	a.Run()
}
```

### Installation

To use GoStart in your project:

```bash
go get github.com/kod2ulz/gostart
```

## Usage Examples

For more detailed, real-world examples of how to use specific packages, please see the following guides:

- **Logging:** [Integrating `logr` with `pgx/v5`](./logr/pgxv5_example.md)
- *(More examples will be added as we refactor each package)*

## Package Architecture

GoStart is organized into focused packages, each with a clear responsibility:

### Core Foundation
- [`/app`](./app/README.md) - Application bootstrap, lifecycle management, and graceful shutdown
- [`/contracts`](./contracts/README.md) - Framework-agnostic interfaces and type definitions
- [`/config`](./config/README.md) - Hierarchical configuration management (Vault → Database → YAML → Env → Defaults)

### API & Web Framework
- [`/api`](./api/README.md) - Framework-agnostic API layer with generic handlers and unified responses
- [`/api/frameworks`](./api/frameworks/gin/) - Framework-specific implementations (currently Gin, extensible)
- [`/router`](./router/README.md) - Unified router interface with CORS and middleware support
- [`/errors`](./errors/README.md) - Enhanced error handling with HTTP status mapping and field validation
- [`/ierrors`](./ierrors/README.md) - Base error interface and chainable error construction

### Data & Persistence
- [`/storage`](./storage/README.md) - Multi-technology storage abstraction layer (PostgreSQL, Redis, extensible)
- [`/query`](./query/README.md) - Intelligent query builders with criteria patterns and field resolution
- [`/sqlc`](./sqlc/README.md) - Database interface definitions and pgx integration utilities

### Application Services
- [`/auth`](./auth/README.md) - Authentication interfaces and JWT/PASETO utilities
- [`/mq`](./mq/README.md) - Message queue integration with publisher-subscriber patterns
- [`/http`](./http/README.md) - Type-safe HTTP client with authentication and logging
- [`/logr`](./logr/README.md) - Structured logging with pluggable handlers and audit support

### Utilities & Extensions
- [`/collections`](./collections/README.md) - Thread-safe data structures and concurrent collections
- [`/object`](./object/README.md) - Enhanced primitive types with fluent APIs
- [`/utils`](./utils/README.md) - Common utilities for environment, validation, and more

### Documentation
- [`/docs`](./docs/README.md) - Comprehensive guides and conceptual documentation
- [Introduction to GoStart](./docs/01-introduction.md) - Getting started guide
- [Configuration Guide](./docs/02-configuration.md) - Configuration patterns and best practices
- [API Development](./docs/03-api-development.md) - Building APIs with GoStart
- [Authentication](./docs/04-authentication.md) - Authentication patterns and implementation

## Roadmap

We have an exciting vision for the future of GoStart:

- [x] **Framework-Agnostic Design:** Unified interfaces allowing developers to choose their favorite framework (Gin, Echo, net/http, etc.).
- [ ] **Advanced Configuration:** Support for YAML, JSON, and HashiCorp Vault, with a built-in caching layer.
- [ ] **Advanced Logging:** Add configurable output drivers for sending logs to files, Logstash, Loki, or Sentry.
- [ ] **Automatic OpenAPI Generation:** Automatically generate API documentation from your route definitions.
- [ ] **Enhanced Metrics:** Integrate with Prometheus for more detailed application monitoring.
- [ ] **Additional Framework Implementations:** Add built-in support for Echo, Fiber, and other popular frameworks.

## Documentation

For more high-level guides and conceptual documentation, please visit the [**docs directory**](./docs).

- [Introduction to GoStart](./docs/01-introduction.md)
- [Configuration](./docs/02-configuration.md)
- [API Development](./docs/03-api-development.md)
- [Authentication](./docs/04-authentication.md)

## Contributing

Contributions are welcome! Please feel free to submit a pull request or open an issue.