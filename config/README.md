# Gostart Configuration (`config`)

This package provides a robust, hierarchical configuration system for `gostart` applications. It is designed to be the single entry point for all configuration access, abstracting away the underlying sources of configuration properties.

## Overview

The core principle is to load configuration from multiple sources with a defined order of precedence. When a configuration key is requested, the system searches for it in each source, starting from the highest priority, and returns the first value it finds.

### Loading Precedence

The system loads configuration from the following sources in this exact order of priority:

1.  **HashiCorp Vault:** For secrets and the most sensitive production data.
2.  **Database (DB):** For dynamic settings that can be changed at runtime. Values are cached to reduce load.
3.  **YAML File:** For environment-specific static configuration (e.g., `config.prod.yaml`).
4.  **Environment Variables (Env):** To allow for overrides in containerized environments.
5.  **Default Values:** Hard-coded values provided at runtime as a final fallback.

---

## Usage

The package is designed to be flexible, allowing for both simple, unified access and explicit access to specific configuration sources.

### Unified Access (Recommended)

The primary way to access configuration is through the unified `Get` function. This function automatically searches through the hierarchy and returns a `Value` type that can be fluently converted to the desired Go type.

```go
import "github.com/kod2ulz/gostart/config"

// Get a string value, falling back to "localhost" if not found anywhere.
host := config.Get("database.host", "localhost").String()

// Get an integer value, falling back to 3000 if not found.
port := config.Get("server.port", 3000).Int()

// Get a time.Duration, falling back to 5 minutes if not found.
// The system can intelligently parse string values like "5m", "1h", etc.
timeout := config.Get("cache.ttl", "5m").Duration()
```

### Explicit Source Access

For cases where you need to bypass the hierarchy and read from a specific source, each source provider has its own `Get()` method.

```go
// Explicitly get a value from Environment Variables
// Note: The Env helper requires a prefix, or use an empty string for global scope.
logLevel := config.Env.Helper("").Get("LOG_LEVEL", "info").String()

// Explicitly get a value from a YAML file
debugMode := config.Yaml.Get("debug", false).Bool()

// Explicitly get a value from the Database
featureFlag := config.DB.Get("feature.new_dashboard.enabled", false).Bool()

// Explicitly get a secret from Vault
apiKey := config.Vault.Get("secret/keys.api-key").String()
```

### Configuring Sources

Each source must be configured during application startup before `config.Get()` is called.

```go
// Example of how sources might be configured (conceptual)
func main() {
    // The Env and Default sources work out of the box.

    // Configure YAML source
    if err := config.Yaml.Load("config.yaml"); err != nil {
        // handle error
    }

    // Configure DB source (simple method)
    // For seeding to work, the 'key' column must have a UNIQUE or PRIMARY KEY constraint.
    dbPool, _ := connectToPostgres()
    config.DB.From(dbPool, "app_settings").WithCache(5 * time.Minute).SeedMissing(true)

    // Configure DB source (advanced method with custom lookup and seeder)
    customLookup := func(ctx context.Context, db config.Dbtx, key string) (string, error) {
        var value string
        err := db.QueryRow(ctx, "SELECT config_value FROM my_special_table WHERE config_key = $1", key).Scan(&value)
        return value, err
    }
    customSeeder := func(ctx context.Context, db config.Dbtx, key, value string) (string, error) {
        _, err := db.Exec(ctx, "INSERT INTO my_special_table (config_key, config_value) VALUES ($1, $2) ON CONFLICT DO NOTHING", key, value)
        return value, err // Return the value that was intended to be set
    }
    config.DB.From(dbPool, "").WithCache(5 * time.Minute).WithLookup(customLookup).WithSeeder(customSeeder)

    // Configure Vault source
    if _, err := config.Vault.Endpoint("https://vault.example.com:8200", "VAULT_TOKEN_ENV_VAR"); err != nil {
        // handle error
    }

    // ... start application
}
```

---

## Implementation Status

As of the latest update, all planned features for the `config` package are implemented:

- [x] **Phase 1: Centralize Environment Configuration**
- [x] **Phase 2: Introduce File-based Configuration (YAML)**
- [x] **Phase 3: Add Database-backed Dynamic Configuration** (including caching, custom lookups, and seeding)
- [x] **Phase 4: Integrate Vault for Secrets Management** (including caching)
- [x] **Final: Unified `Get()` function** with hierarchical lookup.
