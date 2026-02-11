# Logr Package

The `logr` package provides a structured logging framework built on Go's standard `slog` package. It enables powerful log analysis and monitoring for modern applications with pluggable handlers and audit logging capabilities.

## Philosophy

The logr package is built around **structured logging for observability**:

- **Analysis-First**: Logs are designed for machine processing, not just human reading
- **Performance Aware**: Minimal overhead with non-blocking audit logging
- **Security Focused**: Separate audit trails for compliance and security monitoring
- **Standards Compliant**: Built on Go's standard `slog` package for maximum compatibility

## Quick Start

```go
import "github.com/kod2ulz/gostart/logr"

// Initialize with console output
func main() {
    // Development: Pretty console output
    handler := logr.NewConsoleHandler(true)
    if err := logr.Config(handler); err != nil {
        panic(err)
    }

    // Start logging
    logr.Info("Application started",
        "version", "1.0.0",
        "environment", "development",
    )
}
```

## Key Features

### Multiple Output Handlers

```go
// Production: JSON to console and rotating files
func setupProductionLogging() error {
    consoleHandler := logr.NewConsoleHandler(false) // Minified JSON
    fileHandler := logr.NewFileHandler("/var/log/app.log", &logr.RotationConfig{
        MaxSize:    100,
        MaxBackups: 10,
        MaxAge:     30,
        Compress:   true,
    })

    return logr.Config(consoleHandler, fileHandler)
}
```

### Context-Aware Logging

```go
// Add context to all logs within a request scope
func UserHandler(ctx contracts.RequestContext) {
    logger := logr.With(
        "request_id", ctx.Header("X-Request-ID"),
        "user_id", ctx.Header("X-User-ID"),
        "method", ctx.Method(),
        "path", ctx.Path(),
    )

    logger.Info("Processing user request")

    // All subsequent logs include the context
    logger.Debug("Fetching user from database")
    logger.Info("Request completed", "duration_ms", 45)
}
```

### Audit Logging

```go
// Separate audit logging for security events
func setupAuditLogging() {
    auditWriter := logr.NewFileAuditWriter("/var/log/audit.log")
    logr.SetAuditWriter(auditWriter)
}

func LogSecurityEvent(eventType string, details map[string]interface{}) {
    logr.Audit().Log(map[string]interface{}{
        "event_type": eventType,
        "timestamp": time.Now(),
        "details": details,
    })
}
```

### Performance Monitoring

```go
// Automatic performance tracking
func TimeOperation[T any](operation string, fn func() (T, error)) (T, error) {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        logr.Info("Operation completed",
            "operation", operation,
            "duration_ms", duration.Milliseconds(),
        )
    }()
    return fn()
}
```

## Integration with GoStart

The logr package integrates seamlessly with other GoStart components:

- **[`api`](../api/README.md)**: HTTP request logging and middleware integration
- **[`storage`](../storage/README.md)**: Database query logging with pgx integration
- **[`config`](../config/README.md)**: Logging configuration from environment or YAML
- **[`errors`](../errors/README.md)**: Structured error logging with context

### Database Integration

```go
// Enhanced query logging with pgx
func setupDatabaseLogging(dbPool *pgxpool.Pool) {
    dbPool.Config().ConnConfig.Logger = logr.NewPgxLogger()
}

// Automatic query logging with:
// - SQL queries and parameters
// - Execution duration
// - Row counts
// - Connection information
```

## Learning Resources

### Getting Started
- [Logging Quick Start](./docs/quick-start.md) - Basic setup and configuration
- [Handler Guide](./docs/handlers.md) - Console, file, and custom handlers

### Advanced Topics
- [Audit Logging](./docs/audit.md) - Security and compliance logging
- [Performance Monitoring](./docs/performance.md) - Performance tracking and optimization
- [Database Logging](./docs/database.md) - Query logging and pgx integration

### Reference
- [API Documentation](./docs/api.md) - Complete method reference
- [Configuration Examples](./examples/) - Real-world logging configurations
- [Best Practices](./docs/best-practices.md) - Production logging patterns

## When to Use Logr

### Perfect For:
- **Production Applications**: Structured logging for observability platforms
- **Microservices**: Request tracing across service boundaries
- **Security-Conscious Applications**: Audit trails and compliance requirements
- **Performance Monitoring**: Detailed performance metrics and bottlenecks

### Consider Alternatives For:
- **Simple CLI Tools**: Standard library `log` package may suffice
- **Development Debugging**: Pretty console logging without structured needs
- **Legacy Systems**: Applications that don't need observability

## Roadmap

Future enhancements for the logr package:

- **Additional Handlers**: Logstash, Loki, and cloud platform integrations
- **Log Sampling**: Automatic log sampling for high-volume applications
- **Structured Field Validation**: Enforce consistent field names and types
- **Log Aggregation**: Built-in log aggregation and forwarding
- **Metrics Integration**: Automatic metrics extraction from log data

The logr package provides the foundation for production-ready, observability-focused logging in GoStart applications.

### Implementation Status

- [x] **Phase 1: Core Refactor** - Migrated to `slog` package
- [x] **Phase 2: Handlers & Formatting** - Console and file handlers
- [x] **Phase 3: Advanced Routing & Audit** - Audit logging capabilities
- [ ] **Phase 4: Advanced Handlers** - Logstash, Loki, and cloud integrations