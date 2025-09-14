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

For cases where you need to bypass the hierarchy and read from a specific source, the package will provide explicit accessor objects.

```go
// Explicitly get a value from Environment Variables
logLevel := config.Env.GetString("LOG_LEVEL", "info")

// Explicitly get a value from a YAML file
debugMode := config.Yaml.GetBool("debug", false)

// Explicitly get a value from the Database
featureFlag := config.DB.GetBool("feature.new_dashboard.enabled", false)

// Explicitly get a secret from Vault
apiKey := config.Vault.GetString("external_api.key")
```

### Configuring Sources

Each source will require some initial setup, which will be handled during application startup.

```go
// Example of how sources might be configured (conceptual)
func main() {
    // The Env and Default sources work out of the box.

    // Configure YAML source
    config.Yaml.Load("config.yaml")

    // Configure DB source (simple method)
    // For seeding to work, the 'key' column must have a UNIQUE or PRIMARY KEY constraint.
    dbPool, _ := connectToPostgres()
    config.DB.From(dbPool, "app_settings").WithCache(5 * time.Minute).SeedMissing(true)

    // Configure DB source (advanced method with custom lookup and seeder)
    customLookup := func(ctx context.Context, db config.Dbtx, key string) (string, error) {
        // ... your custom query logic ...
        var value string
        err := db.QueryRow(ctx, "SELECT config_value FROM my_special_table WHERE config_key = $1", key).Scan(&value)
        return value, err
    }
    customSeeder := func(ctx context.Context, db config.Dbtx, key, value string) error {
        // ... your custom upsert logic ...
        _, err := db.Exec(ctx, "INSERT INTO my_special_table (config_key, config_value) VALUES ($1, $2) ON CONFLICT DO NOTHING", key, value)
        return err
    }
    config.DB.From(dbPool, "").WithCache(5 * time.Minute).WithLookup(customLookup).WithSeeder(customSeeder).SeedMissing(true)

    // Configure Vault source
    config.Vault.Endpoint("https://vault.example.com:8200", "VAULT_TOKEN_ENV_VAR")

    // ... start application
}
```

---

## Development Roadmap

The `config` package will be developed in the following phases:

*   **Phase 1: Centralize Environment Configuration**
    *   **Goal:** Establish the `config` package and make it the central point for all environment variable access.
    *   **Steps:**
        1. Create the `config` package.
        2. Move the core logic from `utils.Env` into `config/env.go`.
        3. Refactor all existing modules (starting with `app/config.go`) to use `config.Env.Get...` instead of `utils.Env`.

*   **Phase 2: Introduce File-based Configuration (YAML)**
    *   **Goal:** Add support for loading configuration from YAML files.
    *   **Steps:**
        1. Implement `config.Yaml.Load(path)` to parse a YAML file.
        2. Integrate the YAML source into the unified `config.Get()` function, with YAML values overriding Environment Variables.

*   **Phase 3: Add Database-backed Dynamic Configuration**
    *   **Goal:** Allow configuration to be managed dynamically from a database table.
    *   **Steps:**
        1. Implement `config.DB.From(...)` to configure the database source.
        2. Implement a caching mechanism with a configurable TTL to optimize performance.
        3. Integrate the DB source into `config.Get()`, with DB values overriding YAML and Env.

*   **Phase 4: Integrate Vault for Secrets Management**
    *   **Goal:** Add Vault as the highest-priority source for secrets.
    *   **Steps:**
        1. Implement `config.Vault.Endpoint(...)` to configure the Vault client.
        2. Integrate the Vault source into `config.Get()`, with Vault values overriding all other sources.
