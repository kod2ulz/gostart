# API Development

GoStart provides a structured and efficient way to build HTTP APIs. It uses Gin as the underlying framework but adds its own layer of conventions and helpers to standardize development.

## Defining a Route

Routes are defined on the router, which is available on the `app` object. You can define routes for all standard HTTP methods (`GET`, `POST`, `PUT`, `DELETE`, etc.).

```go
app.Router.GET("/users/:id", GetUserHandler)
```

## The API Handler

An API handler is a function that accepts a `*api.Context`.

```go
import "github.com/kod2ulz/gostart/api"

func GetUserHandler(c *api.Context) {
    // Handler logic here
}
```

### The `api.Context`

The `api.Context` is a wrapper around Gin's context. It provides several important helper methods:

- **`c.Success(data interface{})`**: Responds with a `200 OK` and a standardized JSON body containing the data.
- **`c.Failure(err error)`**: Responds with an appropriate HTTP status code and a standardized JSON error body. It automatically handles different error types.
- **`c.Bind(obj interface{}) error`**: A helper for binding and validating request payloads.
- **`c.Param(key string) string`**: Retrieves a URL parameter.
- **`c.User()`**: Retrieves the authenticated user for the current request.

### Example Handler

Here is an example of a handler that retrieves a user ID from the URL, fetches the user from a service, and returns the result.

```go
func GetUserHandler(c *api.Context) {
    userID := c.Param("id")

    // Assume we have a user service
    userService := services.NewUserService(c.DB())
    user, err := userService.FindByID(userID)
    if err != nil {
        // c.Failure will handle turning this error
        // into the correct HTTP response.
        c.Failure(err)
        return
    }

    // c.Success will serialize the user to JSON
    // and wrap it in a standard response structure.
    c.Success(user)
}
```

## Middleware

Middleware can be applied to individual routes or groups of routes. GoStart comes with several built-in middleware for common tasks:

- **Logging:** Logs every incoming request.
- **Recovery:** Recovers from panics and returns a `500` error.
- **Authentication:** Protects routes and attaches the authenticated user to the context.

Here's how you might apply authentication middleware to a group of routes:

```go
// Create a group that requires authentication
private := app.Router.Group("/private")
private.Use(api.AuthMiddleware()) // Attach middleware

private.GET("/profile", GetUserProfileHandler)
private.POST("/settings", UpdateSettingsHandler)
```
