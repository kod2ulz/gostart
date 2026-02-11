# Configuration Hierarchy

The configuration hierarchy is the core feature of the config package, providing intelligent fallback behavior across multiple configuration sources.

## How Hierarchy Works

The config package searches for configuration values in a specific order, stopping at the first source that has the requested value:

1. **HashiCorp Vault** - Production secrets and sensitive data
2. **Database** - Dynamic runtime settings stored in your database
3. **YAML Files** - Environment-specific static configuration
4. **Environment Variables** - Container and deployment overrides
5. **Default Values** - Your application fallback values

## Visual Representation

```
Request: config.Get("database.host", "localhost")

Search Order:
┌─────────────────────────────────────────────────────────────┐
│                    config.Get()                              │
├─────────────────────────────────────────────────────────────┤
│  1. Vault: Check secret/database/host                       │
│     ├─ Configured? Yes → Check if key exists                 │
│     └─ Found? Yes → Return value                             │
│                                                             │
│  2. Database: Check app_settings.database_host              │
│     ├─ Configured? Yes → Query database                     │
│     └─ Found? Yes → Return value                             │
│                                                             │
│  3. YAML: Check database.host from config.{env}.yaml        │
│     ├─ File exists? Yes → Parse YAML                        │
│     └─ Key exists? Yes → Return value                       │
│                                                             │
│  4. Environment: Check DATABASE_HOST                        │
│     ├─ Variable set? Yes → Use value                        │
│     └─ Not found? Continue to next                          │
│                                                             │
│  5. Default: Return "localhost"                             │
└─────────────────────────────────────────────────────────────┘
```

## Missing Source Handling

A key feature of the hierarchy is **graceful degradation** - if a source isn't configured or returns an error, the system automatically falls back to the next source:

```go
// This works even if some sources are not configured
apiKey := config.Get("secrets.api_key", "default").String()

// Possible scenarios:
// 1. Vault not configured → Check Database
// 2. Database not configured → Check YAML
// 3. YAML file not found → Check Environment
// 4. Environment variable not set → Use Default
```

## Performance Implications

### Caching Strategy
Each configuration source can have its own caching strategy:

```go
// Cache database settings for 5 minutes
config.DB.From(dbPool, "app_settings").WithCache(5 * time.Minute)

// Cache Vault secrets for 15 minutes
config.Vault.From(vaultClient).WithCache(15 * time.Minute)
```

### First Request Penalty
The first request for a configuration value will hit the actual sources, but subsequent requests for the same value will return cached results instantly.

## Configuration Source Prioritization

### Production Environment
```go
// Production: Full hierarchy with Vault
config.Get("api.key", "")
// → Vault → Database → YAML → Environment → Default
```

### Development Environment
```go
// Development: Simplified hierarchy
config.Get("api.key", "")
// → Database → YAML → Environment → Default (Vault skipped if not configured)
```

### Testing Environment
```go
// Testing: Environment variables only
config.Get("api.key", "")
// → Environment → Default (other sources skipped)
```

## Custom Hierarchy Behavior

### Bypassing Hierarchy
Sometimes you need to access specific sources directly:

```go
// Get fresh data from database, bypassing cache and hierarchy
dbValue := config.DB.Get("feature_flags.new_ui", false).Bool()

// Access Vault directly, not through hierarchy
vaultSecrets := config.Vault.GetAll("secrets/")
```

### Selective Source Configuration
You can configure which sources to include in your application:

```go
func setupConfig() error {
    env := config.Get("environment", "development").String()

    // Always load YAML configuration
    if err := config.Yaml.Load(fmt.Sprintf("config.%s.yaml", env)); err != nil {
        return err
    }

    // Production-only sources
    if env == "production" {
        // Configure Vault for secrets
        vaultClient, err := vault.NewClient(vault.DefaultConfig())
        if err != nil {
            return err
        }
        config.Vault.From(vaultClient).WithCache(15 * time.Minute)

        // Configure database settings
        config.DB.From(dbPool, "app_settings").WithCache(5 * time.Minute)
    }

    return nil
}
```

## Hierarchy Configuration Examples

### E-commerce Application
```go
// Typical hierarchy for an e-commerce platform
config.Get("payment.gateway.provider", "stripe").String()
// Search order:
// 1. Vault: secret/payment/gateway/provider (production secrets)
// 2. Database: app_settings.payment_gateway_provider (runtime config)
// 3. YAML: payment.gateway.provider (environment-specific)
// 4. Environment: PAYMENT_GATEWAY_PROVIDER (deployment override)
// 5. Default: "stripe"
```

### API Service
```go
// API service configuration
config.Get("server.port", 8080).Int()
config.Get("database.max_connections", 10).Int()
config.Get("auth.jwt_secret", "").String()
config.Get("logging.level", "info").String()
```

## Debugging Hierarchy Issues

### Enable Debug Logging
```go
// Enable debug logging to see hierarchy search process
config.SetDebug(true)

// Now you'll see logs like:
// DEBUG: Checking Vault for key "database.host"
// DEBUG: Vault not configured, falling back to Database
// DEBUG: Querying database for "app_settings.database_host"
// DEBUG: Found value "db.example.com" in database
```

### Check Source Configuration
```go
// Check which sources are configured
sources := config.GetConfiguredSources()
fmt.Printf("Configured sources: %v\n", sources)

// Output: [Database YAML Environment]
```

## Best Practices

### 1. Environment-Specific Defaults
```go
// Use different defaults based on environment
env := config.Get("environment", "development").String()
defaultTimeout := "30s"
if env == "production" {
    defaultTimeout = "10s"
}

timeout := config.Get("server.timeout", defaultTimeout).Duration()
```

### 2. Sensitive Data Handling
```go
// Always use Vault for sensitive data in production
apiKey := config.Get("secrets.api_key", "").String()
if apiKey == "" && env == "production" {
    log.Fatal("API key must be configured in production")
}
```

### 3. Runtime Configuration Updates
```go
// Use database for settings that need to change at runtime
maxConnections := config.Get("database.max_connections", 10).Int()
// This value can be updated in the database without restarting the application
```

### 4. Development Overrides
```go
// Allow developers to override any setting via environment variables
debugMode := config.Get("debug.enabled", false).Bool()
// Developers can set DEBUG_ENABLED=true to enable debugging
```

The configuration hierarchy provides a powerful, flexible system for managing application settings across different environments and deployment scenarios while maintaining security and performance.