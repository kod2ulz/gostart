# Gostart - Go Framework Documentation

## Overview

Gostart is a comprehensive Go framework designed to accelerate the development of web applications and services. It provides a rich set of utilities, patterns, and components for building robust, scalable applications with features such as authentication, HTTP handling, message queuing, logging, and more.

## Key Features

1. **API Framework** - Built on Gin with structured request/response handling
2. **Authentication System** - JWT-based authentication with session management
3. **HTTP Client** - Feature-rich HTTP client with logging and error handling
4. **Message Queue** - RabbitMQ integration for distributed messaging
5. **Collections** - Enhanced data structures (Map, List, Set, Tree)
6. **Logging** - Structured logging with Logrus integration
7. **Configuration** - Environment-based configuration management
8. **Testing Utilities** - Test helpers for API testing

## Project Structure

```
gostart/
├── api/                 # Core API framework components
├── app/                 # Application initialization and configuration
├── auth/                # Authentication utilities
├── collections/         # Enhanced data structures
├── http/                # HTTP client implementation
├── logr/                # Logging framework
├── mq/                  # Message queue (RabbitMQ) integration
├── services/            # Service layer implementations
│   └── auth/            # Authentication service
├── utils/               # Utility functions and helpers
└── ...
```

## Core Components

### 1. API Framework (`api/`)

The API package provides a structured approach to building REST APIs with:

- Request parameter handling and validation
- Response formatting (data, list, error responses)
- Middleware for logging and authentication
- Generic handlers for common patterns

Key files:
- `request_handlers.go` - Generic request handlers for different use cases
- `response.go` - Standardized response structures
- `middleware.go` - Authentication and logging middleware
- `params.go` - Request parameter parsing and validation

### 2. Application Bootstrap (`app/`)

The app package handles application initialization:

- Configuration loading from environment variables
- HTTP server setup with Gin
- Consul service registration
- Graceful shutdown handling

### 3. Authentication (`services/auth/`)

Full-featured JWT-based authentication system:

- User session management
- Login, signup, token verification, and refresh
- In-memory and extensible user storage
- Password hashing and validation

### 4. Collections (`collections/`)

Enhanced generic data structures:

- `Map[K, T]` - Enhanced map with utility methods
- `List[T]` - Slice with functional programming methods
- `Set[T]` - Set implementation with standard operations
- `Tree` - Tree data structure

### 5. HTTP Client (`http/`)

Feature-rich HTTP client with:

- Request building with headers, params, and body
- Response parsing and error handling
- Logging and tracing
- Session and authorization support

### 6. Message Queue (`mq/`)

RabbitMQ integration for message-based communication:

- Connection management
- Exchange and queue declaration
- Publishing and consuming messages
- Worker pattern implementation

### 7. Logging (`logr/`)

Structured logging built on Logrus:

- Context-aware logging
- Trace ID and process ID support
- JSON and text formatters
- Field enrichment

### 8. Utilities (`utils/`)

Various helper functions:

- Environment variable handling
- JSON processing
- String and numeric utilities
- Testing helpers
- Validation utilities

## Usage Patterns

### Application Initialization

```go
// Initialize application
app := app.Init(
    app.WithStrictEnv(),
    app.WithHeartbeatHandlers(),
)

// Register services and routes
// ...

// Start the application
app.Run()
```

### Authentication Service Setup

```go
// Create in-memory user store
userStore := auth.InMemoryUserStore()

// Initialize session service
sessionService := auth.SessionService(log, userStore)

// Register API routes
sessionService.API(router.Group("/auth"))
```

### HTTP Request Handling

```go
// Using generic handlers
router.POST("/users", api.BasicHandler[User](CreateUser))

// With parameter validation
router.POST("/users/:id", api.ParamHandlerWithResponse[UserRequest, User](GetUser))
```

### HTTP Client Usage

```go
// Create client
client := http.Client[User](log)

// Make request
response := client.
    BaseUrl("https://api.example.com").
    Header("Authorization", "Bearer token").
    Get(context.Background(), "/users/123")

// Handle response
if user, err := response.Data(); err == nil {
    // Process user data
}
```

### Message Queue Integration

```go
// Initialize RabbitMQ
rmq := mq.RabbitMQ(ctx, log, config)

// Declare exchange and queue
exchange := rmq.TopicExchange("user.events")
queue := rmq.Queue("user.notifications")

// Publish message
exchange.Publish("user.created", userData)

// Consume messages
queue.Consume(handlerFunc)
```

## Configuration

The framework uses environment variables for configuration with the following prefixes:

- `APP_*` - Application configuration
- `CONSUL_*` - Consul integration settings
- `HTTP_SERVER_*` - HTTP server settings
- `SESSION_SERVICE_TOKEN_*` - Authentication token settings

## Testing

The framework includes comprehensive testing utilities:

- Ginkgo/Gomega for BDD-style testing
- HTTP request/response testing helpers
- JSON encoding/decoding utilities for test data
- Mock implementations for dependencies

## Dependencies

Key dependencies include:

- Gin (HTTP framework)
- RabbitMQ client (message queue)
- Logrus (logging)
- Ginkgo/Gomega (testing)
- JWT (authentication)
- UUID (identifiers)
- Consul API (service discovery)

## Extensibility

The framework is designed to be extensible:

1. **Custom Authentication** - Implement the `SessionUser` and `SessionStore` interfaces
2. **Custom Collections** - Extend the provided data structures
3. **Custom HTTP Handlers** - Use the generic handler patterns
4. **Custom Message Queue Handlers** - Implement custom consumers and publishers