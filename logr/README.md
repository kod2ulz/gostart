# Logger (`logr`)

## 1. Vision & Goals

This package provides a high-performance, structured, and extensible logging framework for the `gostart` ecosystem. It is designed not just for displaying logs, but for enabling powerful, automated **log analysis**.

The core goal is to produce consistent, machine-readable logs (primarily JSON) that can be shipped to platforms like Grafana, Loki, or the ELK Stack (Elasticsearch, Logstash, Kibana). This enables:

- **Performance Monitoring:** Tracking request durations, identifying bottlenecks, and analyzing slow database queries.
- **Security & Auditing:** Detecting attacks, service misuse, and providing a clear audit trail.
- **Analytics:** Understanding user behavior and geographic usage patterns.
- **Debugging:** Tracing requests across multiple services via a `request_id` and quickly diagnosing panics and failures with exact file locations.

## 2. Core Architecture

To achieve these goals, the `logr` package is built on the following principles:

- **`slog`-native API:** The public API is a wrapper around the standard library's `slog.Logger`, ensuring modern, idiomatic Go usage.
- **Pluggable Handlers (Sinks/Writers):** The output destination is determined by one or more `slog.Handler` implementations. This allows for extreme flexibility. The library provides pre-built handlers for common use cases, and developers can easily write and plug in their own.

## 3. Configuration & Usage

Initialization is designed to be simple and declarative. The main `logr.Config(...slog.Handler)` function accepts one or more handlers that determine where logs are sent. If no handler is provided, it defaults to a JSON handler writing to the console.

### Handler: Console Output

`logr.NewConsoleHandler(pretty bool) slog.Handler`

- **pretty `false` (Production):** Outputs compact, single-line JSON.
- **pretty `true` (Development):** Outputs colorized, human-readable logs.

**Example: Pretty-printed console logger for development**
```go
// main.go
func main() {
    handler := logr.NewConsoleHandler(true)
    if err := logr.Config(handler); err != nil {
        panic(err)
    }
    // ...
}
```

### Handler: File Output

`logr.NewFileHandler(path string, rotation *logr.RotationConfig) slog.Handler`

This handler writes logs to a file with automatic rotation. The rotation behavior can be configured with the `RotationConfig` struct. If `nil` is passed, settings are read from environment variables, with sensible defaults.

**RotationConfig Struct:**
```go
type RotationConfig struct {
    MaxSize    int  // Max size in megabytes before rotation
    MaxAge     int  // Max number of days to retain old log files
    MaxBackups int  // Max number of old log files to retain
    Compress   bool // Whether to compress/gzip old log files
}
```

**Environment Variables for Default Rotation:**
- `LOG_ROTATE_MAX_SIZE`: (Default: `100` MB)
- `LOG_ROTATE_MAX_AGE`: (Default: `28` days)
- `LOG_ROTATE_MAX_BACKUPS`: (Default: `5`)
- `LOG_ROTATE_COMPRESS`: (Default: `true`)

**Example: Production file logger with explicit rotation**
```go
// main.go
func main() {
    handler := logr.NewFileHandler("/var/log/app.log", &logr.RotationConfig{
        MaxSize:    100,
        MaxBackups: 3,
        MaxAge:     28,
        Compress:   true,
    })
    if err := logr.Config(handler); err != nil {
        panic(err)
    }
    // ...
}
```

### Combining Multiple Handlers

You can easily log to multiple destinations by passing more than one handler to `Config`.

**Example: Log to both console and a file**
```go
// main.go
func main() {
    consoleHandler := logr.NewConsoleHandler(false) // Minified JSON for production
    fileHandler := logr.NewFileHandler("/var/log/app.log", nil) // Use default rotation from ENV

    if err := logr.Config(consoleHandler, fileHandler); err != nil {
        panic(err)
    }
    // ...
}
```

## 4. Log Structure

The default JSON handler produces a consistent, top-level structure. While top-level keys are standardized, attribute values can be complex nested objects or arrays.

**Example: Anonymized HTTP Request Log**
```json
{
  "time": "2025-09-13T14:20:04Z",
  "level": "INFO",
  "msg": "",
  "source": {
    "file": "/app/gostart/api/log.go:59",
    "function": "github.com/user/project/app.(*ap).initAPI.JSONLogMiddleware.func3"
  },
  "application": "my-app-name",
  "host": "app-host-1",
  "client_ip": "41.210.141.216",
  "duration": 9,
  "method": "GET",
  "path": "/api/public/settings",
  "referrer": "https://example.com/admin",
  "request_id": "b6ce62e1-fdbe-4ce4-bd91-ddc3b7435915",
  "size": 677,
  "status": 200,
  "user_id": "fcc80928-9ca7-4e9a-b7df-748d352705ad"
}
```

**Example: Anonymized Database Query Log (with nested object)**
```json
{
  "time": "2025-09-13T14:20:04Z",
  "level": "INFO",
  "msg": "Query",
  "source": {
    "file": "/app/vendor/github.com/jackc/pgx/log/adapter.go:34",
    "function": "github.com/jackc/pgx/log.Logger.Log"
  },
  "application": "my-app-name",
  "host": "app-host-1",
  "pid": 522747,
  "sql": "SELECT id, name, email FROM users WHERE id = $1 LIMIT $2",
  "args": ["user-uuid-123", 1],
  "rowCount": 1
}
```

## 5. Audit Logging

Audit logging is a separate, critical concern. The audit logger is configured independently from the main application logger to ensure audit trails are never dropped and are routed to a secure, permanent destination.

**Example: Configuring a Database Audit Writer**
```go
// main.go

dbConn := getDbConnection() // Your database connection

auditWriter := func(entry map[string]interface{}) error {
    // Custom logic to insert the audit entry into a database table
    return dbConn.Create(&models.AuditLog{Details: entry}).Error
}

logr.SetAuditWriter(auditWriter)

// Later, in your application code...
func someImportantAction(user User) {
    logr.Audit().Log(map[string]interface{}{
        "action": "user_deleted",
        "user_id": user.ID,
        "timestamp": time.Now(),
    })
}
```

## 6. Implementation Roadmap

- [x] **Phase 1: Core Refactor**
    - [x] Migrate core API from `logrus` to `slog`.
    - [x] Implement non-blocking, channel-based handler (Re-evaluated and removed for simplicity, can be re-added if needed).
    - [x] Fix log caller source location reporting.
- [x] **Phase 2: Handlers & Formatting**
    - [x] Implement `NewConsoleHandler` (supporting pretty/minified JSON).
    - [x] Implement `NewFileHandler` (with log rotation).
    - [ ] Implement a customizable text-based handler.
- [ ] **Phase 3: Advanced Routing & Audit**
    - [ ] Implement Level-based Routing (e.g., errors to a separate file/handler).
    - [ ] Implement `SetAuditWriter` and `logr.Audit()` API.
    - [ ] Provide a pre-built `NewFileAuditWriter`.
- [ ] **Phase 4: Advanced Handlers**
    - [ ] Implement `NewLogstashHandler`.
    - [ ] Implement `NewLokiHandler`.
