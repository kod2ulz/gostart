# Logger (`logr`)

This package provides a structured, centralized logging solution for the `gostart` ecosystem. It is built on top of the standard library's `slog` package, providing a consistent and extensible logging experience.

## Overview

The `logr` package is designed to be the single source of truth for all logging. It offers:
- A simple, global logger instance.
- Structured, key-value based logging.
- Helpers for adding common context like trace and process IDs.
- A pluggable architecture for future expansion.

## Initialization & Configuration

The logger is initialized once at application startup, typically in `main.go`. The `Config()` function reads environment variables to set up the default logging pipeline.

**Example (`main.go`):**
```go
package main

import (
	"github.com/kod2ulz/gostart/app"
	"github.com/kod2ulz/gostart/logr"
)

func main() {
	if err := logr.Config(); err != nil {
		panic(err)
	}

	app.Init()
	// ... rest of your application
}
```

### Configuration via Environment Variables

- `LOG_LEVEL`: Sets the minimum log level. Supported values are `debug`, `info`, `warn`, `error`. Defaults to `info`.

## Basic Usage

Once initialized, you can access the global logger using `logr.Log()`.

```go
import "github.com/kod2ulz/gostart/logr"

func MyFunction() {
    log := logr.Log()

    // Log a simple informational message
    log.Info("User has logged in")

    // Log with structured key-value pairs
    log.Info("User processed payment", "user_id", 123, "amount_cents", 5000)

    // Log an error
    err := someFunction()
    if err != nil {
        log.Error("Something went wrong", "error", err)
    }
}
```

## Adding Context

You can add persistent context to a logger instance for a specific scope or request.

```go
func HandleRequest(req *http.Request) {
    // Create a logger instance with request-specific fields
    log := logr.Log().TID().ExtendWithField("http_method", req.Method)

    log.Info("Handling request")

    // ...

    log.Info("Finished handling request")
}
```

- `TID()`: Adds a unique Trace ID to the logger instance.
- `PID()`: Adds a unique Process ID to the logger instance.
- `WithTID(string)`: Adds a specific, externally provided Trace ID.
- `ExtendWithField(key string, value interface{})`: Returns a new logger instance with the specified field added.

## Roadmap

- [x] Core: Migrate from `logrus` to a `slog`-based API.
- [ ] Configuration: Implement high-level, configurable output "writers" (e.g., for console, file, Loki, Logstash).
- [ ] Audit Logging: Implement a separate, independently configured audit logger accessible via `logr.Audit()`.
- [ ] Pluggable Backends: Finalize the API to allow swapping the default `slog.Handler` for other backends (e.g., a `zap`-based handler).
