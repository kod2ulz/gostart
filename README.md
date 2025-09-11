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
- **Robust API Layer:** A clean pattern for defining HTTP handlers, middleware, and request/response models on top of Gin.
- **Pluggable Authentication:** Comes with helpers for JWT and PASETO tokens, with integrations for providers like AWS Cognito.
- **Structured Logging:** Centralized and configurable logging using Logrus, with built-in adapters for components like `pgx`.
- **Database & Cache Ready:** Includes helpers and interfaces for PostgreSQL (via pgx v5) and Redis.
- **Message Queuing:** Integrated RabbitMQ publisher and worker system for background jobs.
- **Rich Utilities:** A large collection of helpers for configuration, error handling, concurrent data structures, and more.

## Getting Started

Here's how to bootstrap a new application using GoStart.

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/app"
)

func main() {
	// 1. Initialize the application
	// This sets up the logger, router, and graceful shutdown handling.
	a := app.Init()
	ctx, log := a.Ctx(), a.Log()

	log.Info("Application starting up...")

	// 2. Define a route using the underlying Gin router
	a.R().GET("/hello", func(c *gin.Context) {
		// Use the api.Respond helper for consistent JSON responses
		api.Respond(c).Success("world")
	})

	// 3. Run the application
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

- `/app`: Core application, configuration, and router.
- `/api`: API request/response handling.
- `/auth`: Authentication and session management.
- `/storage`: Caching (Redis) and persistence helpers.
- `/mq`: Message queue integration (RabbitMQ).
- `/logr`: Structured logging configuration.
- `/query`: Database query builders and helpers.
- `/collections`: Thread-safe collections and data structures.
- `/utils`: Common utilities for errors, environment variables, etc.

## Roadmap

We have an exciting vision for the future of GoStart:

- [ ] **Pluggable Web Frameworks:** Allow developers to choose their favorite framework (Echo, Fiber, etc.) instead of being locked into Gin.
- [ ] **Advanced Configuration:** Support for YAML, JSON, and HashiCorp Vault, with a built-in caching layer.
- [ ] **Advanced Logging:** Add configurable output drivers for sending logs to files, Logstash, Loki, or Sentry.
- [ ] **Automatic OpenAPI Generation:** Automatically generate API documentation from your route definitions.
- [ ] **Enhanced Metrics:** Integrate with Prometheus for more detailed application monitoring.

## Documentation

For more high-level guides and conceptual documentation, please visit the [**docs directory**](./docs).

- [Introduction to GoStart](./docs/01-introduction.md)
- [Configuration](./docs/02-configuration.md)
- [API Development](./docs/03-api-development.md)
- [Authentication](./docs/04-authentication.md)

## Contributing

Contributions are welcome! Please feel free to submit a pull request or open an issue.