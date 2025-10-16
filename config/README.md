# Config Package

This package provides a robust, hierarchical configuration system for GoStart applications. It is designed to be the single entry point for all configuration access, abstracting away the underlying sources of configuration properties.

## Overview

The config package solves the common challenge of managing configuration across multiple environments and sources:

- **Hierarchical Access**: Single `Get()` call that searches through multiple sources in priority order
- **Type Safety**: Automatic type conversion with sensible defaults
- **Graceful Fallbacks**: Missing sources don't break your application
- **Caching**: Intelligent caching for improved performance
- **Multiple Sources**: Environment variables, YAML files, database, Vault, and more

## Core Architecture

The package follows a hierarchical configuration model where each source is consulted in order:

```
1. HashiCorp Vault (production secrets)
2. Database (runtime settings)
3. YAML Files (environment-specific config)
4. Environment Variables (deployment overrides)
5. Default Values (application fallbacks)
```

## Basic Usage

The package is designed to be flexible, allowing for both simple, unified access and explicit access to specific configuration sources.

### Unified Access (Recommended)

The primary way to access configuration is through the unified `Get` function. This function automatically searches through the hierarchy and returns a `Value` type that can be fluently converted to the desired Go type.

```go
func setupDatabase() *sql.DB {
    // Automatic hierarchy search with fallbacks
    host := config.Get("database.host", "localhost").String()
    port := config.Get("database.port", 5432).Int()
    user := config.Get("database.user", "app").String()
    password := config.Get("database.password", "").String()
    name := config.Get("database.name", "myapp").String()

    dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
        host, port, user, password, name)

    db, err := sql.Open("postgres", dsn)
    if err != nil {
        log.Fatal(err)
    }
    return db
}
```

### Type-Safe Conversions

The `Value` type provides fluent conversion methods:

```go
// All these work with automatic type conversion
port := config.Get("server.port", 8080).Int()           // int
timeout := config.Get("server.timeout", "30s").Duration()  // time.Duration
debug := config.Get("server.debug", false).Bool()          // bool
enabled := config.Get("features.new_ui", true).Bool()      // bool with default
```

## Configuration Sources

### Environment Variables
Environment variables work automatically with SCREAMING_SNAKE_CASE conversion:

```go
// These all access the same environment variable
config.Get("database.host", "localhost").String()
config.Get("database_host", "localhost").String()       // Also works

// Environment variable: DATABASE_HOST=localhost
```

### YAML Configuration
Load YAML files for environment-specific configuration:

```go
// Load YAML configuration manually
err := config.Yaml.Load("config.production.yaml")

// Access nested values with dot notation
dbHost := config.Get("database.host", "localhost").String()
maxConns := config.Get("database.max_connections", 10).Int()
```

### Database Configuration
Store configuration in your database for runtime updates:

```go
// Setup database configuration source
config.DB.From(dbPool, "app_settings").WithCache(5 * time.Minute)

// Configuration is stored as key-value pairs
// Table structure: key, value, updated_at
```

### HashiCorp Vault
Secure secret management with automatic caching:

```go
// Setup Vault integration
config.Vault.From(vaultClient).WithCache(15 * time.Minute)

// Access secrets using path.key notation
apiKey := config.Get("secrets.api_key", "").String()
```

### Service Discovery
Consul integration for service discovery (note: Consul is not a configuration source):

```go
// Discover service URLs
serviceURL, err := config.Consul.GetServiceURL("user-service")
```

## How the Hierarchy Works

The configuration system searches sources in this order:

1. **Vault** (if configured) - Production secrets
2. **Database** (if configured) - Runtime settings
3. **YAML** (if loaded) - Environment-specific config
4. **Environment Variables** - Deployment overrides
5. **Default Values** - Application fallbacks

```go
// This single call searches all sources:
host := config.Get("database.host", "localhost").String()

// Missing sources are automatically skipped
// If Vault isn't configured, it tries Database
// If Database isn't configured, it tries YAML
// And so on...
```

## Implementation Status

### ✅ **Fully Implemented Features**
- **Complete Hierarchical Configuration**: Sources searched in priority order (Vault → DB → YAML → Environment → Default)
- **Comprehensive Type-Safe Conversions**: Int, String, Bool, Duration, Time, UUID, Float64, and custom types
- **Environment Variable Integration**: Automatic SCREAMING_SNAKE_CASE conversion with `.env` file support
- **YAML Configuration Loading**: Manual file loading with nested dot-notation access (`database.host`)
- **Database Configuration Source**: Runtime settings with intelligent caching (5+ minute TTL support)
- **HashiCorp Vault Integration**: Secure secret management with configurable caching (15+ minute TTL)
- **Service Discovery**: Consul integration for service URL discovery and health checking
- **Graceful Fallback System**: Missing sources automatically skipped, no application crashes
- **Configuration Value Interface**: Fluent conversion methods with proper error handling

### ⚠️ **Needs Verification/Enhancement**
- **Configuration Validation**: Basic validation exists, may need enhancement for complex scenarios
- **Caching Performance**: Current caching works, but performance tuning may be needed
- **Error Handling**: Error messages could be more descriptive for configuration issues

### 🚧 **Planned Enhancements**
- **Configuration Monitoring**: Real-time change detection and automatic reloading
- **Automatic YAML Loading**: Environment-based automatic file discovery and loading
- **Advanced Configuration Validation**: Struct validation with custom validation rules
- **Consul as Configuration Source**: Currently only service discovery, planned for config storage
- **Additional Configuration Providers**: AWS Parameter Store, Azure Key Vault, Google Secret Manager
- **Configuration Management UI**: Web interface for managing configuration across environments

## Advanced Usage

### Database Configuration Setup
```go
// Setup database configuration with caching
config.DB.From(dbPool, "app_settings").WithCache(5 * time.Minute)

// Database table structure expected:
// CREATE TABLE app_settings (
//     key VARCHAR(255) PRIMARY KEY,
//     value TEXT,
//     updated_at TIMESTAMP
// );
```

### Vault Integration
```go
// Setup Vault for secret management
config.Vault.From(vaultClient).WithCache(15 * time.Minute)

// Access secrets using path.key notation
apiKey := config.Get("secrets.api_key", "").String()
dbPassword := config.Get("secrets.database.password", "").String()
```

### Direct Source Access
```go
// Access specific sources directly (bypassing hierarchy)
dbValue := config.DB.Get("database.max_connections", 10).Int()
vaultValue := config.Vault.Get("secrets.api_key", "").String()
```

## Environment Setup

### Development Environment
```go
func setupDevelopmentConfig() {
    // Load development YAML
    config.Yaml.Load("config.development.yaml")

    // Environment variables override YAML
    // DATABASE_HOST=localhost (from env) overrides YAML database.host
}
```

### Production Environment
```go
func setupProductionConfig() error {
    // Load production YAML
    if err := config.Yaml.Load("config.production.yaml"); err != nil {
        return err
    }

    // Setup Vault for secrets
    config.Vault.From(vaultClient).WithCache(15 * time.Minute)

    // Setup database for runtime configuration
    config.DB.From(dbPool, "app_settings").WithCache(5 * time.Minute)

    return nil
}
```

## Best Practices

### Configuration Organization
```go
// Group related configuration
type DatabaseConfig struct {
    Host     string
    Port     int
    User     string
    Password string
    Name     string
}

func loadDatabaseConfig() DatabaseConfig {
    return DatabaseConfig{
        Host:     config.Get("database.host", "localhost").String(),
        Port:     config.Get("database.port", 5432).Int(),
        User:     config.Get("database.user", "app").String(),
        Password: config.Get("database.password", "").String(),
        Name:     config.Get("database.name", "myapp").String(),
    }
}
```

### Error Handling
```go
// Always check for configuration errors
timeout, err := config.Get("server.timeout", "30s").Duration()
if err != nil {
    log.Printf("Invalid timeout configuration: %v", err)
    timeout = 30 * time.Second
}
```

### Environment-Specific Configuration
```go
// Use environment to determine configuration source
env := config.Get("environment", "development").String()

switch env {
case "production":
    setupProductionConfig()
case "staging":
    setupStagingConfig()
default:
    setupDevelopmentConfig()
}
```

## Integration with Other Packages

The config package works seamlessly with other GoStart packages:

```go
import (
    "github.com/kod2ulz/gostart/logr"
    "github.com/kod2ulz/gostart/storage"
)

func setupApplication() {
    // Configure logging
    logLevel := config.Get("logging.level", "info").String()
    log := logr.New(logLevel)

    // Configure storage
    storage.Init(storage.Config{
        ConnectionString: config.Get("database.url", "").String(),
        MaxConnections: config.Get("database.max_connections", 10).Int(),
    })
}
```

## Learn More

📚 **Detailed guides:**
- [Configuration Hierarchy](./docs/hierarchy.md) - How the fallback system works
- [Configuration Sources](./docs/sources.md) - Setting up different sources
- [Advanced Patterns](./docs/advanced-patterns.md) - Direct access and caching strategies

---

This package provides a solid foundation for configuration management in Go applications with an emphasis on simplicity, type safety, and flexibility.