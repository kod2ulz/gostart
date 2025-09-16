# GoStart: The Opinionated Go Project Bootstrap

[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](https://github.com/kod2ulz/gostart)

**GoStart** is a library built to bootstrap Go projects quickly, enforcing best practices for request and route management, authentication, and database interactions. It allows developers to build faster, providing extendable logging and metrics without compromising on standards.

---

## Overview

This library provides a solid foundation for building scalable and maintainable backend services in Go. It wires together common components like a web framework, authentication, data access, caching, and message queues, allowing you to focus on business logic instead of boilerplate code.

The core design philosophy is to be "opinionated" where it matters (structure, standards, security) but flexible where it counts (pluggable components).

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

## Project Structure

The library is organized into logical packages. For a detailed breakdown, see the [Introduction to GoStart](./docs/01-introduction.md).

- `/app`: Core application bootstrap and configuration.
- `/api`: Framework-agnostic API layer with unified response envelopes and generic handlers.
- `/api/frameworks`: Framework-specific implementations (Gin, Echo, etc.).
- `/contracts`: Core interfaces defining the framework-agnostic contracts.
- `/auth`: Authentication, session management, and authorization middleware.
- `/http`: HTTP client utilities for making external API calls.
- `/storage`: Caching (Redis) and persistence helpers.
- `/mq`: Message queue integration (RabbitMQ).
- `/logr`: Structured logging configuration.
- `/query`: Database query builders and helpers.
- `/collections`: Thread-safe collections and data structures.
- `/utils`: Common utilities for errors, environment variables, etc.

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