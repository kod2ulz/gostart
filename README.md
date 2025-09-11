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

- **Rapid Setup:** Get a production-ready server running in minutes.
- **Robust API Layer:** A clean pattern for defining HTTP handlers, middleware, and request/response models.
- **Pluggable Authentication:** Comes with a pre-configured AWS Cognito integration, but is designed to be extensible.
- **Structured Logging:** Centralized and configurable logging using Logrus.
- **Database & Cache Ready:** Includes helpers and interfaces for PostgreSQL (via pgx) and Redis.
- **Message Queuing:** Integrated RabbitMQ publisher and worker system for background jobs.
- **Rich Utilities:** A large collection of helpers for configuration, error handling, data structures, and more.

## Getting Started

Here's how to bootstrap a new application using GoStart.

```go
package main

import (
	"github.com/kod2ulz/gostart/app"
	"github.com/kod2ulz/gostart/api"
)

func main() {
    // 1. Initialize the application
	app, err := app.New()
	if err != nil {
		panic(err)
	}

    // 2. Define a route
	app.Router.GET("/hello", func(c *api.Context) {
		c.Success("world")
	})

    // 3. Run the application
	if err := app.Run(); err != nil {
        panic(err)
    }
}
```

### Installation

To use GoStart in your project:

```bash
go get github.com/kod2ulz/gostart
```

## Project Structure

The library is organized into logical packages. For a detailed breakdown, see the [Introduction to GoStart](./docs/01-introduction.md).

- `/app`: Core application, configuration, and router.
- `/api`: API request/response handling.
- `/auth`: Authentication and session management.
- `/storage`: Caching and persistence (Redis).
- `/mq`: Message queue integration (RabbitMQ).
- `/services`: Business logic layer.
- `/query`: Database query builders and helpers.

## Roadmap

We have an exciting vision for the future of GoStart:

- [ ] **Automatic OpenAPI Generation:** Automatically generate API documentation from your route definitions.
- [ ] **Pluggable Web Frameworks:** Allow developers to choose their favorite framework (Echo, Fiber, etc.) instead of being locked into Gin.
- [ ] **Enhanced Metrics:** Integrate with Prometheus for more detailed application monitoring.

## Documentation

For more detailed guides and conceptual documentation, please visit the [**docs directory**](./docs).

- [Introduction to GoStart](./docs/01-introduction.md)
- [Configuration](./docs/02-configuration.md)
- [API Development](./docs/03-api-development.md)
- [Authentication](./docs/04-authentication.md)

## Contributing

Contributions are welcome! Please feel free to submit a pull request or open an issue.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
