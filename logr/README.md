# Logr Package

This package provides a structured, context-aware logging wrapper around the popular `logrus` library.

## Overview

The goal of `logr` is to enrich log messages with useful, consistent context, such as a `trace_id` for tracking a request across services and a `process_id` for identifying a specific task or worker.

Logs can be output to the console (stdout) and optionally to a file.

## Initialization

Before using the logger, it must be initialized once during application startup. This is typically done in your `main` function by calling `logr.Config()`.

```go
import (
    "github.com/kod2ulz/gostart/logr"
)

func main() {
    if err := logr.Config(); err != nil {
        panic(err)
    }
    
    // ... rest of your application
}
```

### File Logging

To enable logging to a file in addition to the console, set the following environment variable:

- `LOG_FILE_PATH`: The full path to the log file (e.g., `/var/log/myapp.log`).

The logger will create the file if it doesn't exist and append to it if it does.

## Usage

Once initialized, you can get a logger instance and add context to it.

The logger is designed to be used with a chaining syntax. Each call to add context returns a new logger instance, preventing context from one log statement from leaking into another.

### Basic Logging

```go
import "github.com/kod2ulz/gostart/logr"

// Get a logger instance
logr.Log().Info("This is a standard log message.")
logr.Log().Warn("This is a warning.")
```

### Logging with Context

This is the primary strength of the `logr` package.

```go
// Log with a Trace ID for a specific request
logr.Log().TID().Info("User login attempt.")
// Output will include a field like: "trace_id":"<some-uuid>"

// Use a specific Trace ID if you have one from an incoming request
logr.Log().WithTID("existing-trace-id").Error("Failed to process payment.")

// Add a Process ID for a background job
logr.Log().PID().WithField("job_name", "invoice-processor").Info("Starting job.")
// Output will include: "process_id":"<some-uuid>"

// Combine multiple context fields
logr.Log().TID().WithField("user_id", 123).Infof("User %d updated their profile", 123)
```

### Key Context Methods

- `TID()`: Generates and adds a new UUID as the `trace_id`.
- `WithTID(string)`: Adds the given string as the `trace_id`.
- `PID()`: Generates and adds a new UUID as the `process_id`.
- `WithField(key, value)`: The standard logrus method for adding a custom field.
- `WithFields(logrus.Fields)`: The standard logrus method for adding multiple custom fields.

## Roadmap

- **Multiple Outputs:** Enhance the logger to support writing to multiple outputs simultaneously (e.g., console and a file).
- **External Service Integration:** Add native support for shipping logs to external services like Logstash, Loki, or Sentry.