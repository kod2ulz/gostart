# Application Bootstrap (`app`)

This package is the core of a `gostart` service. It is responsible for bootstrapping the application, managing its lifecycle, and integrating all the other library components like configuration, logging, and HTTP routing.

## Overview

The `app` package provides a single entry point, `app.Init()`, which creates a global application instance. This instance handles:

- **Configuration Loading:** Initializes the `config` package, including loading `.env` files.
- **Logger Initialization:** Sets up the global `logr` logger.
- **HTTP Server:** Configures and runs a `gin-gonic` HTTP router.
- **Service Registration:** Manages service registration and deregistration with Consul.
- **Graceful Shutdown:** Listens for interrupt signals (`SIGTERM`, `SIGINT`) to shut down the application cleanly.

## Usage

The typical usage is to call `app.Init()` at the start of your `main()` function, and then `Run()` to start the application and block until a shutdown signal is received.

```go
func main() {
    // Initialize the application
    app.Init(
        app.WithHeartbeatHandlers(), // Adds /ok and /stats endpoints
    )

    // Register a new API route
    app.R().GET("/my-route", func(c *gin.Context) {
        app.Log().Info("Request received!")
        c.JSON(200, gin.H{"message": "hello"})
    })

    // Run the application
    app.Run()
}
```

### Customization

The `Init` function accepts `AppIniter` options to customize its behavior:

- `app.WithHeartbeatHandlers()`: Automatically adds `/ok` and `/stats` health check endpoints.
- `app.WithHandlerOverride(key, handler)`: Overrides one of the default handlers (e.g., `ok`, `stats`).
- `app.WithStaticFileHandler(webPath, filePath)`: Serves a single static file.

### Service Registration (Consul)

If Consul is configured via environment variables (`CONSUL_HTTP_ADDR`), the `app.Run()` method will automatically attempt to register the service with Consul.

- **Configuration:** The registration is configured using environment variables with the `CONSUL_` prefix (e.g., `CONSUL_SERVICE_NAME`, `CONSUL_SERVICE_ID_AUTO`).
- **Lifecycle:** The service is registered on startup and gracefully deregistered on shutdown.