# Configuration Sources

The config package supports multiple configuration sources, each designed for specific use cases and deployment scenarios.

## Available Sources

### 1. Environment Variables

Environment variables are the simplest configuration source, perfect for containerized deployments and development environments.

```go
// Basic usage
port := config.Get("server.port", 8080).Int()
debug := config.Get("debug.enabled", false).Bool()

// Environment variables are automatically checked
// DATABASE_HOST, SERVER_PORT, DEBUG_ENABLED
```

**Features:**
- Always available (no configuration required)
- Automatic name conversion (snake_case from dot.notation)
- Type conversion with validation
- Perfect for Docker/Kubernetes deployments

**Best Practices:**
```go
// Use environment variables for deployment-specific settings
config.Get("database.host", "localhost").String()      // DATABASE_HOST
config.Get("database.port", 5432).Int()                 // DATABASE_PORT
config.Get("redis.url", "").String()                    // REDIS_URL
config.Get("jwt.secret", "").String()                    // JWT_SECRET
```

### 2. YAML Files

YAML files provide structured, environment-specific configuration that's easy to read and maintain.

```go
// Load environment-specific YAML
err := config.Yaml.Load("config.production.yaml")
if err != nil {
    log.Fatal(err)
}

// Access YAML values
dbHost := config.Get("database.host", "localhost").String()
```

**Example YAML file:**
```yaml
# config.production.yaml
database:
  host: "db.production.example.com"
  port: 5432
  name: "production_db"
  ssl_mode: "require"

server:
  port: 8080
  debug: false
  timeout: "30s"

logging:
  level: "info"
  format: "json"

features:
  new_ui: true
  beta_features: false
```

**Advanced YAML Features:**
```yaml
# Using environment-specific files
# config.{environment}.yaml

# Nested structures
database:
  primary:
    host: "db1.example.com"
    port: 5432
  replica:
    host: "db2.example.com"
    port: 5432

# Arrays and lists
allowed_origins:
  - "https://example.com"
  - "https://app.example.com"

# Environment-specific overrides
database:
  host: ${DB_HOST:-localhost}  # Environment variable fallback
  port: ${DB_PORT:-5432}
```

### 3. Database Configuration

Database configuration allows runtime updates without application restarts, perfect for feature flags and dynamic settings.

```go
// Configure database settings
config.DB.From(dbPool, "app_settings").WithCache(5 * time.Minute)

// Access database values
featureFlag := config.Get("features.new_ui", false).Bool()
maxConnections := config.Get("database.max_connections", 10).Int()
```

**Database Schema:**
```sql
CREATE TABLE app_settings (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) UNIQUE NOT NULL,
    value TEXT,
    value_type VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Example settings
INSERT INTO app_settings (key, value, value_type) VALUES
('features.new_ui', 'true', 'boolean'),
('database.max_connections', '20', 'integer'),
('api.rate_limit', '100', 'integer'),
('maintenance.mode', 'false', 'boolean');
```

**Advanced Usage:**
```go
// Direct database access (bypasses hierarchy)
dbValue := config.DB.Get("features.new_ui", false).Bool()

// Batch operations
settings := config.DB.GetAll("features.")
// Returns: map[string]interface{}{"new_ui": true, "beta": false}

// Watch for changes
config.DB.Watch(func(key string, value interface{}) {
    log.Printf("Configuration changed: %s = %v", key, value)
})
```

### 4. HashiCorp Vault

Vault integration provides secure storage for sensitive data like API keys, passwords, and secrets.

```go
// Configure Vault for production
vaultClient, err := vault.NewClient(vault.DefaultConfig())
if err != nil {
    log.Fatal(err)
}

config.Vault.From(vaultClient).WithCache(15 * time.Minute)

// Access secrets securely
apiKey := config.Get("secrets.api_key", "").String()
dbPassword := config.Get("secrets.database_password", "").String()
```

**Vault Secrets Structure:**
```
vault kv put secret/myapp/database \
    host="db.example.com" \
    password="securepassword" \
    port=5432

vault kv put secret/myapp/api \
    key="secret-api-key" \
    secret="webhook-secret"
```

**Advanced Vault Features:**
```go
// Direct Vault access
secrets := config.Vault.GetAll("secrets/")

// Specific secret path
dbSecrets := config.Vault.GetPath("secret/myapp/database")

// Custom TTL for caching
config.Vault.From(vaultClient).WithCache(30 * time.Minute)

// Multiple Vault mounts
config.Vault.AddMount("kv", "secret/")
config.Vault.AddMount("transit", "transit/")
```

### 5. Consul Integration

Consul provides service discovery and distributed configuration management.

```go
// Configure Consul
consulClient, err := api.NewClient(api.DefaultConfig())
if err != nil {
    log.Fatal(err)
}

config.Consul.From(consulClient)

// Service discovery
serviceURL, err := config.Consul.GetServiceURL("user-service")
if err != nil {
    log.Fatal(err)
}

// Distributed configuration
replicaCount := config.Get("scaling.replicas", 3).Int()
```

**Consul Key Structure:**
```
consul kv put myapp/config/database/host "db.example.com"
consul kv put myapp/config/database/port "5432"
consul kv put myapp/config/features/new_ui "true"
```

## Source Configuration

### Selective Source Activation

```go
func setupConfig() error {
    env := config.Get("environment", "development").String()

    // Always available sources
    // Environment variables - no configuration needed
    // YAML files
    if err := config.Yaml.Load(fmt.Sprintf("config.%s.yaml", env)); err != nil {
        return err
    }

    // Production-only sources
    if env == "production" {
        // Database configuration
        config.DB.From(dbPool, "app_settings").WithCache(5 * time.Minute)

        // Vault for secrets
        vaultClient, err := vault.NewClient(vault.DefaultConfig())
        if err != nil {
            return err
        }
        config.Vault.From(vaultClient).WithCache(15 * time.Minute)

        // Consul for service discovery
        consulClient, err := api.NewClient(api.DefaultConfig())
        if err != nil {
            return err
        }
        config.Consul.From(consulClient)
    }

    return nil
}
```

### Custom Source Configuration

```go
// Configure caching strategies
config.DB.From(dbPool, "settings").
    WithCache(10 * time.Minute).
    WithErrorHandler(func(err error) {
        log.Printf("Database config error: %v", err)
    })

config.Vault.From(vaultClient).
    WithCache(30 * time.Minute).
    WithPrefix("secret/myapp/")

config.Yaml.
    WithLoader(customYamlLoader).
    WithWatcher(true) // Watch for file changes
```

## Source Interaction Patterns

### 1. Hierarchical Access (Recommended)
```go
// Automatic search through all configured sources
apiKey := config.Get("secrets.api_key", "default").String()
```

### 2. Direct Source Access
```go
// Access specific source directly
dbValue := config.DB.Get("features.new_ui", false).Bool()
vaultValue := config.Vault.Get("secrets.api_key", "").String()
```

### 3. Bulk Operations
```go
// Get all settings from a source
allSettings := config.DB.GetAll("app.")
allSecrets := config.Vault.GetAll("secrets/")
```

### 4. Configuration Watching
```go
// Watch for configuration changes
config.Watch(func(change config.Change) {
    log.Printf("Config changed: %s = %v", change.Key, change.NewValue)

    // Handle specific changes
    switch change.Key {
    case "features.new_ui":
        updateUIFeature(change.NewValue.Bool())
    case "database.max_connections":
        updateConnectionPool(change.NewValue.Int())
    }
})
```

## Error Handling and Resilience

### Graceful Degradation
```go
// Sources that fail to initialize are automatically skipped
// The system continues with the next available source

tryConfig := func(fn func() error) {
    if err := fn(); err != nil {
        log.Printf("Configuration source failed: %v", err)
    }
}

tryConfig(func() error {
    return config.Yaml.Load("config.yaml")
})

tryConfig(func() error {
    return config.DB.From(dbPool, "settings")
})

// Configuration still works even if some sources fail
```

### Custom Error Handlers
```go
config.DB.From(dbPool, "settings").
    WithErrorHandler(func(err error) {
        // Log database configuration errors
        log.Printf("Database config error: %v", err)

        // Send alert if configuration is critical
        if strings.Contains(err.Error(), "connection") {
            monitoring.Alert("Database configuration unavailable")
        }
    })
```

## Performance Considerations

### Caching Strategies
```go
// Aggressive caching for frequently accessed values
config.DB.From(dbPool, "settings").WithCache(30 * time.Minute)

// No caching for values that need to be fresh
config.DB.From(dbPool, "feature_flags").WithCache(0)

// Custom cache TTL per source
config.Vault.From(vaultClient).WithCache(15 * time.Minute)  // Secrets change rarely
config.DB.From(dbPool, "metrics").WithCache(1 * time.Minute)  // Metrics change frequently
```

### Lazy Loading
```go
// Configuration values are loaded on demand
// No upfront loading of all configuration

// First request: hits actual source
apiKey := config.Get("secrets.api_key", "").String()

// Subsequent requests: return cached value
apiKey = config.Get("secrets.api_key", "").String()  // Instant!
```

## Migration Between Sources

### Environment to Database
```go
// Migrate configuration from environment to database
func migrateToDatabase() {
    // Old way (environment variables)
    oldPort := os.Getenv("SERVER_PORT")
    oldDebug := os.Getenv("DEBUG_ENABLED")

    // New way (database)
    config.DB.Set("server.port", oldPort)
    config.DB.Set("debug.enabled", oldDebug)

    // Configuration now works with both sources
    port := config.Get("server.port", 8080).Int()
    // Will check Database → Environment → Default
}
```

### YAML to Vault
```go
// Migrate secrets from YAML to Vault
func migrateSecretsToVault() {
    // Load secrets from YAML (old way)
    var secrets struct {
        APIKey string `yaml:"api_key"`
        DBPassword string `yaml:"db_password"`
    }

    // Store in Vault (new way)
    vaultClient.Logical().Write("secret/myapp/api", map[string]interface{}{
        "key": secrets.APIKey,
    })

    // Now secrets are securely stored
    apiKey := config.Get("secrets.api_key", "").String()
}
```

The configuration sources provide a flexible, secure, and performant way to manage application settings across different environments and deployment scenarios.