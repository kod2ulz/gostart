# Configuration Debug Endpoints

## Overview

HTTP endpoints for inspecting the configuration of a **running application**. This shows you:
- What's currently cached in memory
- Cache expiry times
- All configuration values from all sources

## Security

**IMPORTANT**: Only enable these in development/staging! Never in production!

Two security layers protect the endpoints:
1. **Local Network Only** - Only accessible from localhost or local network (10.x.x.x, 172.16-31.x.x, 192.168.x.x)
2. **Bearer Token** - Requires a token shown at application startup

## Setup

Add to your `main.go`:

```go
package main

import (
    "log"
    "github.com/kod2ulz/gostart/app"
    "github.com/kod2ulz/gostart/api"
    "github.com/kod2ulz/gostart/api/frameworks/gin"
    "github.com/kod2ulz/gostart/config"
    _ "github.com/kod2ulz/gostart/config/yaml" // Load YAML config
)

func main() {
    // Generate debug token
    debugToken := api.MustGenerateDebugToken()

    // Show token at startup
    log.Println("=================================================")
    log.Println("DEBUG CONFIGURATION ENDPOINTS ENABLED")
    log.Println("=================================================")
    log.Printf("Debug Token: %s\n", debugToken)
    log.Println("")
    log.Println("Endpoints:")
    log.Println("  GET /debug/config?mode=effective&format=yaml")
    log.Println("  GET /debug/config/stats")
    log.Println("")
    log.Println("Example Usage:")
    log.Printf("  curl -H 'Authorization: Bearer %s' http://localhost:8080/debug/config\n", debugToken)
    log.Printf("  curl -H 'Authorization: Bearer %s' http://localhost:8080/debug/config?mode=cached\n", debugToken)
    log.Printf("  curl -H 'Authorization: Bearer %s' http://localhost:8080/debug/config?format=json\n", debugToken)
    log.Println("=================================================")
    log.Println("")

    // Load configuration
    if err := config.Yaml.Load("config.yaml"); err != nil {
        log.Printf("Warning: could not load config.yaml: %v", err)
    }

    // Setup router
    gin.Setup()

    router := app.R()

    // Register SECURE debug endpoints (with local network + token protection)
    api.RegisterSecureConfigDebugEndpoints(router, debugToken)

    // Register your other routes...
    // router.GET("/users", usersHandler)

    // Run application
    app.Run()
}
```

## Usage

### 1. Get Cache Statistics

```bash
curl -H 'Authorization: Bearer YOUR_DEBUG_TOKEN' http://localhost:8080/debug/config/stats
```

**Response:**
```json
{
  "success": true,
  "type": "...",
  "data": [
    {"source": "vault", "size": 15},
    {"source": "database", "size": 8}
  ]
}
```

### 2. View Cached Configuration (with expiry)

```bash
curl -H 'Authorization: Bearer YOUR_DEBUG_TOKEN' \
  'http://localhost:8080/debug/config?mode=cached'
```

**Response (YAML format):**
```yaml
entries:
- key: secret/data/app/api_key
  value: "sk-1234567890"
  source: vault
  type: string
  expires_at: 2026-01-11T15:30:00Z
  time_to_live: 14m30s
  is_stale: false
- key: database.max_connections
  value: "100"
  source: database
  type: integer
  expires_at: 2026-01-11T15:25:00Z
  time_to_live: 4m12s
  is_stale: false
```

### 3. View Full Effective Configuration

```bash
curl -H 'Authorization: Bearer YOUR_DEBUG_TOKEN' \
  'http://localhost:8080/debug/config?mode=effective&format=yaml'
```

Shows:
- YAML config values
- Environment variables
- Cached DB values (with expiry)
- Cached Vault secrets (with expiry)

### 4. Get JSON Format

```bash
curl -H 'Authorization: Bearer YOUR_DEBUG_TOKEN' \
  'http://localhost:8080/debug/config?mode=effective&format=json'
```

### 5. Get ENV Format (.env file format)

```bash
curl -H 'Authorization: Bearer YOUR_DEBUG_TOKEN' \
  'http://localhost:8080/debug/config?format=env&mode=static'
```

**Output (.env file format):**
```bash
# Config key: staff.admin.seeder.run
STAFF_ADMIN_SEEDER_RUN=true

# Config key: database.host (from yaml)
DATABASE_HOST=localhost

# Config key: http.server.port (from environment)
HTTP_SERVER_PORT=8080
```

This format is useful for:
- Creating `.env` files from your current config
- Seeing what environment variables map to what config keys
- Copying values to your `.env` file

### 6. Get YAML Format (nested structure)

```bash
curl -H 'Authorization: Bearer YOUR_DEBUG_TOKEN' \
  'http://localhost:8080/debug/config?format=yaml&mode=static'
```

**Output (nested YAML like a config file):**
```yaml
staff:
  admin:
    seeder:
      run: true
database:
  host: localhost
  port: "5432"
http:
  server:
    port: "8080"
```

This format is useful for:
- Visualizing the config structure
- Spotting typos or wrong nesting levels
- Creating YAML config files from environment variables

## Query Parameters

### `mode` (optional, default: `effective`)

- `cached` - Only cached values (Vault, DB) with expiry metadata
- `static` - Only static values (YAML, ENV) without caching
- `effective` - All configuration from all sources

### `format` (optional, default: `yaml`)

- `yaml` - **Nested YAML format** (looks like a config file with proper structure)
- `json` - Flat JSON array with all entries and metadata
- `env` - **.env file format** (shows environment variables like you'd use in a .env file)

## Examples

### See what Vault secrets are cached and when they expire:

```bash
curl -H 'Authorization: Bearer TOKEN' \
  'http://localhost:8080/debug/config?mode=cached&format=json' | jq '.entries[] | select(.source == "vault")'
```

### See all config as pretty YAML:

```bash
curl -H 'Authorization: Bearer TOKEN' \
  'http://localhost:8080/debug/config?mode=effective&format=yaml'
```

### Check cache size:

```bash
curl -H 'Authorization: Bearer TOKEN' \
  'http://localhost:8080/debug/config/stats' | jq '.data'
```

## Security Notes

1. **Local Network Restriction** - Blocks external requests
2. **Bearer Token** - Token generated at startup (64-character hex string)
3. **Development Only** - Never expose these endpoints in production
4. **Sensitive Data** - Tokens, passwords, API keys will be visible

## What Gets Cached

- **Vault secrets** - Cached for 15 minutes (configurable via `config.Vault.WithCacheTTL()`)
- **Database config** - Cached for 5 minutes (configurable via `config.DB.WithCacheTTL()`)

Once cached, values remain in memory until they expire, even if the source changes.

## Troubleshooting

### "FORBIDDEN: Access allowed from local network only"
- Make sure you're accessing from localhost or a local network IP
- If using Docker, ensure proper port forwarding

### "UNAUTHORIZED: Missing Authorization header"
- Add `-H 'Authorization: Bearer YOUR_DEBUG_TOKEN'` to your curl command
- Check the startup logs for the correct token

### Empty response
- Check if Vault/DB sources are configured (`config.Vault.Endpoint()`, `config.DB.From()`)
- No values may have been cached yet (config is lazy-loaded)
- Try accessing an endpoint that uses Vault/DB config first
- For environment variables, make sure they're set before starting the application

### Environment variable not showing up
1. Check if the variable is actually set: `echo $YOUR_VAR`
2. Verify the naming convention (SCREAMING_SNAKE_CASE → dot.notation)
3. Check the `original_key` field in the debug output to see what was found
4. Make sure a higher-priority source (Vault/DB/YAML) isn't overriding it
5. Check the system environment variable filter - some system vars are filtered out

## Output Fields

For cached entries:

| Field | Description |
|-------|-------------|
| `key` | Configuration key (in dot notation) |
| `value` | Configuration value |
| `source` | Where it came from (vault/database/yaml/environment) |
| `type` | Data type (string/integer/boolean/array) |
| `original_key` | Original environment variable name (for environment source only) |
| `expires_at` | When the cache entry expires (ISO 8601) |
| `time_to_live` | How long until expiry (duration) |
| `is_stale` | Whether the entry is expired but still usable |

## Environment Variable Naming

Environment variables are automatically converted from `SCREAMING_SNAKE_CASE` to `dot.notation`:

| Environment Variable | Config Key |
|---------------------|------------|
| `STAFF_ADMIN_SEEDER_RUN` | `staff.admin.seeder.run` |
| `DATABASE_HOST` | `database.host` |
| `HTTP_SERVER_PORT` | `http.server.port` |
| `APP_DEBUG_MODE` | `app.debug.mode` |

**Important:** The prefix matters! If you set `CONFIG_STAFF_ADMIN_SEEDER_RUN=true`, it becomes `config.staff.admin.seeder.run`, not `staff.admin.seeder.run`. The correct environment variable name for `config.Get("staff.admin.seeder.run")` is `STAFF_ADMIN_SEEDER_RUN=true`.

## Troubleshooting Environment Variables

If your environment variable isn't being loaded:

1. **Use the debug endpoint** to see what's actually loaded:
   ```bash
   curl -H 'Authorization: Bearer TOKEN' \
     'http://localhost:8080/debug/config?mode=static&format=json' | jq '.entries[] | select(.source == "environment")'
   ```

2. **Check the `original_key` field** to see the exact environment variable name that was found

3. **Verify your environment variable name:**
   - Should be in `SCREAMING_SNAKE_CASE`
   - Should NOT have extra prefixes (like `CONFIG_`) unless you want them in the config key
   - Examples:
     - ❌ `CONFIG_STAFF_ADMIN_SEEDER_RUN` → becomes `config.staff.admin.seeder.run`
     - ✅ `STAFF_ADMIN_SEEDER_RUN` → becomes `staff.admin.seeder.run`

4. **Check config priority** - values are loaded in this order:
   - Vault (highest priority)
   - Database
   - YAML
   - Environment
   - Default (lowest priority)

   If Vault, DB, or YAML has the value set, it will override your environment variable!

## Best Practices

### 1. Use a Consistent Prefix

To avoid system environment variables polluting your config, use a consistent prefix like `APP_`:

```bash
# Good: Clear prefix
APP_STAFF_ADMIN_SEEDER_RUN=true
APP_DATABASE_HOST=localhost

# Bad: No prefix (may conflict with system vars)
STAFF_ADMIN_SEEDER_RUN=true
DATABASE_HOST=localhost
```

### 2. Follow the Naming Convention

| Config Key | Environment Variable |
|------------|---------------------|
| `staff.admin.seeder.run` | `STAFF_ADMIN_SEEDER_RUN` |
| `database.host` | `DATABASE_HOST` |
| `http.server.port` | `HTTP_SERVER_PORT` |
| `app.debug.mode` | `APP_DEBUG_MODE` |

**Rule:** `SCREAMING_SNAKE_CASE` (with underscores) ↔ `dot.notation` (with dots)

### 3. Use the Key() Helper Pattern

Like in your `staff/config.go`, use a `Key()` helper to avoid typos:

```go
func (c Config) Key(param ...string) string {
    if len(c.prefix) > 0 {
        param = append([]string{c.prefix}, param...)
    }
    return strings.Join(param, ".")
}

// Usage
c.Key("seeder.run")  // Returns: "staff.admin.seeder.run"
```

Then log the key when it's not set:

```go
s.log.Info(fmt.Sprintf("set config: %s=true to run seeder", s.config.Key("seeder.run")))
// Output: "set config: staff.admin.seeder.run=true to run seeder"
```

### 4. Check the Debug Output

When a config value isn't what you expect:

```bash
# See all environment variables in .env format
curl -H 'Authorization: Bearer TOKEN' \
  'http://localhost:8080/debug/config?mode=static&format=env'

# See nested YAML structure to spot nesting issues
curl -H 'Authorization: Bearer TOKEN' \
  'http://localhost:8080/debug/config?mode=static&format=yaml'
```
