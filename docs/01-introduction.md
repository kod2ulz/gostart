# Introduction to GoStart

GoStart is designed to be a powerful and opinionated starting point for new Go backend services. The primary goal is to solve common problems and provide a standardized structure so that developers can focus on writing business logic that delivers value.

## Core Philosophy

1.  **Convention over Configuration:** Where possible, GoStart makes sensible decisions for you. This reduces boilerplate and ensures that different projects built with the library have a consistent structure.
2.  **Standards-Driven:** The library enforces best practices for API design, error handling, and security. This ensures that applications are robust and maintainable.
3.  **Extensibility:** While opinionated, GoStart is not rigid. Key components like authentication and the web framework are designed to be pluggable, allowing you to adapt the library to your specific needs.

## Project Structure Explained

- **`/app`**: This is the heart of your service. It's responsible for loading configuration, setting up the router, and managing the application lifecycle.

- **`/api`**: This package provides the tools for building your HTTP API. It contains a custom `Context` that wraps Gin's context, providing helper methods for standardized success and error responses. It also includes middleware for logging, authentication, and more.

- **`/auth`**: This package handles user authentication. The default implementation uses AWS Cognito, but the interfaces are designed to allow for other providers to be integrated.

- **`/collections`**: A set of useful, thread-safe data structures that are commonly needed in concurrent applications.

- **`/http`**: Generic HTTP client utilities for making requests to other services.

- **`/logr`**: A wrapper around Logrus that provides a standardized logging setup for the application.

- **`/mq`**: Contains everything needed to interact with RabbitMQ, including connection management, publishers for sending messages, and a worker system for processing them.

- **`/storage`**: Provides an abstraction for caching. The default implementation uses Redis.

- **`/services`**: This is where your core business logic should reside. Services can be used by your API handlers to perform actions and interact with your database or other parts of the application.
