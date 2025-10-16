# HTTP Package

The `http` package provides a powerful, type-safe HTTP client for GoStart applications. It offers a fluent API for making HTTP requests with built-in support for authentication, logging, error handling, and response parsing.

## Overview

This package simplifies HTTP client interactions by providing:

- **Type-safe responses**: Generic type parameters for automatic JSON parsing
- **Fluent API**: Chain methods for readable request construction
- **Built-in authentication**: Support for various authentication methods
- **Comprehensive logging**: Request/response logging with performance metrics
- **Error handling**: Enhanced error messages and HTTP status code mapping
- **Configuration management**: Centralized client configuration

## Basic Usage

### Simple HTTP Requests

```go
import "github.com/kod2ulz/gostart/http"

// Create a typed client for User responses
userClient := http.Client[User](logger)

// GET request
user, err := userClient.
    BaseUrl("https://api.example.com").
    Get("/users/123")

// POST request
newUser := User{Name: "John Doe", Email: "john@example.com"}
createdUser, err := userClient.
    BaseUrl("https://api.example.com").
    Post("/users", newUser)

// PUT request
updatedUser, err := userClient.
    BaseUrl("https://api.example.com").
    Put("/users/123", updatedUserData)

// DELETE request
err := userClient.
    BaseUrl("https://api.example.com").
    Delete("/users/123")
```

### Advanced Request Configuration

```go
// Configure client with all options
client := http.Client[ApiResponse](logger)

response, err := client.
    BaseUrl("https://api.example.com").
    Timeout(30 * time.Second).                    // Custom timeout
    Header("Authorization", "Bearer token123").     // Custom headers
    Header("X-Custom-Header", "custom-value").
    Param("page", "1").                             // Query parameters
    Param("limit", "10").
    Session(authSession).                           // Authentication session
    Post("/users", newUser)
```

## Authentication

### Basic Authentication

```go
// Basic auth session
type BasicAuthSession struct {
    Username string
    Password string
}

func (s *BasicAuthSession) Authorization() string {
    credentials := base64.StdEncoding.EncodeToString(
        []byte(s.Username + ":" + s.Password),
    )
    return "Basic " + credentials
}

// Use with client
client := http.Client[User](logger)
response, err := client.
    Session(&BasicAuthSession{Username: "user", Password: "pass"}).
    Get("/users/me")
```

### Bearer Token Authentication

```go
// Bearer token session
type BearerTokenSession struct {
    Token string
}

func (s *BearerTokenSession) Authorization() string {
    return "Bearer " + s.Token
}

// Use with client
client := http.Client[User](logger)
response, err := client.
    Session(&BearerTokenSession{Token: "your-token-here"}).
    Get("/users/me")
```

### Custom Authentication

```go
// Custom authentication session
type CustomAuthSession struct {
    APIKey    string
    SecretKey string
}

func (s *CustomAuthSession) Authorization() string {
    timestamp := time.Now().Unix()
    signature := generateSignature(s.APIKey, s.SecretKey, timestamp)
    return fmt.Sprintf("HMAC %s:%d:%s", s.APIKey, timestamp, signature)
}

// Use with client
client := http.Client[ApiResponse](logger)
response, err := client.
    Session(&CustomAuthSession{
        APIKey:    "your-api-key",
        SecretKey: "your-secret-key",
    }).
    Get("/secure-data")
```

## Request Building

### URL and Path Management

```go
client := http.Client[User](logger)

// Base URL with path parameters
user, err := client.
    BaseUrl("https://api.example.com").
    Get("/users/{id}", "123")                       // Path parameters

// Dynamic URL construction
client := http.Client[User](logger)
user, err := client.
    BaseUrl("https://api.example.com/v1").
    Get("/users/{id}/profile", userId)
```

### Query Parameters

```go
client := http.Client[UserList](logger)

// Simple query parameters
users, err := client.
    BaseUrl("https://api.example.com").
    Param("page", "1").
    Param("limit", "20").
    Param("sort", "name").
    Get("/users")

// Array query parameters
users, err := client.
    BaseUrl("https://api.example.com").
    Param("status", "active").
    Param("status", "pending").                    // Multiple values for same key
    Param("roles", "admin").
    Param("roles", "user").
    Get("/users")

// Complex query parameters
users, err := client.
    BaseUrl("https://api.example.com").
    Param("filter", map[string]interface{}{
        "age_min": 18,
        "age_max": 65,
        "active":  true,
    }).
    Get("/users")
```

### Headers Management

```go
client := http.Client[ApiResponse](logger)

// Add individual headers
response, err := client.
    BaseUrl("https://api.example.com").
    Header("Content-Type", "application/json").
    Header("Accept", "application/json").
    Header("X-Request-ID", utils.UUID.New().String()).
    Post("/data", payload)

// Add multiple headers at once
headers := map[string]interface{}{
    "Content-Type":     "application/json",
    "Accept":          "application/json",
    "X-Request-ID":    utils.UUID.New().String(),
    "X-Custom-Header": "custom-value",
}

response, err := client.
    BaseUrl("https://api.example.com").
    Headers(headers).                           // Add multiple headers
    Post("/data", payload)

// Conditional headers
client := http.Client[ApiResponse](logger)
request := client.BaseUrl("https://api.example.com")

if useCompression {
    request = request.Header("Accept-Encoding", "gzip")
}

if useTracing {
    request = request.Header("X-Trace-ID", traceID)
}

response, err := request.Get("/data")
```

## Error Handling

### Enhanced Error Responses

```go
client := http.Client[User](logger)

user, err := client.
    BaseUrl("https://api.example.com").
    Get("/users/123")

if err != nil {
    // Enhanced error information
    if httpErr, ok := err.(*http.Error); ok {
        log.Printf("HTTP Error: %d - %s", httpErr.StatusCode, httpErr.Message)
        log.Printf("Request ID: %s", httpErr.RequestID)
        log.Printf("Response Body: %s", httpErr.ResponseBody)

        // Handle specific status codes
        switch httpErr.StatusCode {
        case 401:
            // Handle unauthorized
        case 404:
            // Handle not found
        case 429:
            // Handle rate limiting
        default:
            // Handle other errors
        }
    }
    return nil, err
}
```

### Retry Logic

```go
// Implement retry logic with backoff
func GetUserWithRetry(client *http.Client[User], userID string, maxRetries int) (*User, error) {
    var lastErr error

    for i := 0; i < maxRetries; i++ {
        user, err := client.
            BaseUrl("https://api.example.com").
            Get(fmt.Sprintf("/users/%s", userID))

        if err == nil {
            return user, nil
        }

        // Check if error is retryable
        if isRetryableError(err) {
            lastErr = err
            time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second)
            continue
        }

        // Non-retryable error
        return nil, err
    }

    return nil, fmt.Errorf("failed after %d retries, last error: %w", maxRetries, lastErr)
}

func isRetryableError(err error) bool {
    if httpErr, ok := err.(*http.Error); ok {
        // Retry on 5xx errors, 429 (rate limit), and network errors
        return httpErr.StatusCode >= 500 || httpErr.StatusCode == 429
    }
    return false
}
```

## Performance Monitoring

### Request Timing

```go
// Enable request timing
client := http.Client[User](logger)

response, err := client.
    BaseUrl("https://api.example.com").
    WithTiming().                                  // Enable timing metrics
    Get("/users/123")

// Timing information is automatically logged
// You can also access it programmatically if needed
```

### Request Logging

```go
// Configure logging levels
client := http.Client[User](logger)

// Enable detailed logging
response, err := client.
    BaseUrl("https://api.example.com").
    WithDetailedLogging().                        // Log request/response details
    Get("/users/123")

// Logs will include:
// - Request method, URL, headers, and body
// - Response status, headers, and body
// - Request duration
// - Request ID for tracing
```

## Response Handling

### Response Validation

```go
// Validate response before parsing
type User struct {
    ID    string `json:"id" validate:"required"`
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"required,email"`
}

client := http.Client[User](logger)

user, err := client.
    BaseUrl("https://api.example.com").
    WithValidation().                               // Enable response validation
    Get("/users/123")

if err != nil {
    // Validation errors are automatically handled
    return nil, err
}

// user is guaranteed to be valid
```

### Raw Response Access

```go
// Access raw response when needed
client := http.Client[User](logger)

user, err := client.
    BaseUrl("https://api.example.com").
    WithRawResponse().                            // Capture raw response
    Get("/users/123")

if err == nil {
    // Access raw response details
    statusCode := client.LastResponse().StatusCode
    headers := client.LastResponse().Headers
    body := client.LastResponse().Body

    log.Printf("Status: %d", statusCode)
    log.Printf("Response Headers: %v", headers)
}
```

## Configuration Management

### Global Client Configuration

```go
// Configure default settings
func setupHTTPClient() {
    // Set global defaults
    http.SetDefaultTimeout(30 * time.Second)
    http.SetDefaultRetryCount(3)
    http.SetDefaultLogLevel(logrus.InfoLevel)
}

// Create client with global defaults
client := http.Client[User](logger)
// Client will use configured defaults
```

### Environment-Specific Configuration

```go
// Configure based on environment
func createHTTPClient() *http.Client[User] {
    client := http.Client[User](logger)

    switch os.Getenv("ENVIRONMENT") {
    case "production":
        client.Timeout(10 * time.Second)
        client.Header("User-Agent", "MyApp/1.0")
        client.WithDetailedLogging()

    case "development":
        client.Timeout(30 * time.Second)
        client.WithDebugLogging()

    case "testing":
        client.Timeout(5 * time.Second)
        client.WithMockResponses()  // Use mock responses for testing
    }

    return &client
}
```

## Integration with GoStart Framework

### Service Layer Integration

```go
type UserService struct {
    client *http.Client[User]
    logger *logrus.Entry
}

func NewUserService(logger *logrus.Entry) *UserService {
    return &UserService{
        client: http.Client[User](logger),
        logger: logger,
    }
}

func (s *UserService) GetUserByID(userID string) (*User, ierrors.Error) {
    user, err := s.client.
        BaseUrl("https://api.example.com").
        Get(fmt.Sprintf("/users/%s", userID))

    if err != nil {
        return nil, errors.ServiceUnavailable(err)
    }

    return user, nil
}

func (s *UserService) CreateUser(user *User) (*User, ierrors.Error) {
    created, err := s.client.
        BaseUrl("https://api.example.com").
        Post("/users", user)

    if err != nil {
        return nil, errors.ServiceFailure(err)
    }

    return created, nil
}
```

### API Handler Integration

```go
func ExternalAPIHandler(ctx contracts.RequestContext) (Response, ierrors.Error) {
    var req ExternalAPIRequest

    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return Response{}, err
    }

    // Create HTTP client for external API
    client := http.Client[ExternalAPIResponse](logger)

    // Forward request to external API
    response, err := client.
        BaseUrl("https://external-api.example.com").
        Header("Authorization", "Bearer "+getAPIToken()).
        Timeout(10 * time.Second).
        Post("/endpoint", req.Data)

    if err != nil {
        return Response{}, errors.ServiceFailure(err)
    }

    return Response{
        Data:     response,
        Metadata: extractMetadata(response),
    }, nil
}
```

The http package provides a comprehensive, production-ready HTTP client that simplifies external API interactions while maintaining type safety, performance, and observability.