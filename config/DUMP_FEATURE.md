# Configuration Dump Feature

## Overview

The configuration dump feature allows you to output the current application configuration in multiple formats for debugging and auditing purposes.

## Three Dump Modes

### 1. Static Configuration Dump (`-dump-config`)

Dumps all static configuration (YAML file, environment variables) that's loaded at startup.

**What's included:**
- YAML config values (from config.yaml)
- Environment variables (APP_*, HTTP_SERVER_*, DATABASE_*, etc.)

**Use when:** You want to see the static configuration that's always available.

```bash
./your-app -dump-config=yaml
./your-app -dump-config=json
./your-app -dump-config=env
```

### 2. Cached Configuration Dump (`-dump-cached`)

Dumps **only** the cached values from dynamic sources (Vault, Database) with their expiry metadata.

**What's included:**
- Vault cached secrets (with time-to-live)
- Database cached values (with time-to-live)
- Expiration timestamps
- Stale entry indicators

**Use when:** You want to see what's currently in the runtime cache and when it will expire.

```bash
./your-app -dump-cached
./your-app -dump-cached -format=json
```

**Example output:**
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

### 3. Effective Configuration Dump (`-dump-effective`)

Shows the **full picture** - all configuration from all sources, both static and cached.

**What's included:**
- YAML config values (static)
- Environment variables (static)
- Database cached values (with expiry)
- Vault cached secrets (with expiry)

**Use when:** You want to see the complete runtime configuration state.

```bash
./your-app -dump-effective
./your-app -dump-effective -format=json
```

## Command Line Flags

Add these flags to your application:

```bash
# Static config (YAML + ENV only)
./your-app -dump-config=yaml

# Cached config (Vault + DB only)
./your-app -dump-cached

# Full effective config (all sources)
./your-app -dump-effective

# Change format of any dump
./your-app -dump-effective -format=json
./your-app -dump-cached -format=env

# Shorthand for YAML dump
./your-app -show-config
```

## Cache Metadata

Cached entries include additional metadata:

```yaml
entries:
- key: database.password
  value: "********"
  source: vault              # Where it came from
  expires_at: 2026-01-11T15:30:00Z  # When it expires
  time_to_live: 14m30s       # How long until expiry
  is_stale: false            # Whether it's expired but still usable
```

## Programmatic Usage

```go
import "github.com/kod2ulz/gostart/config"

// Dump static config
config.DumpConfig(config.FormatYAML)

// Dump cached config with metadata
config.DumpCachedConfig(config.FormatJSON)

// Dump full effective config
dump, err := config.CollectEffectiveConfig()
if err != nil {
    log.Fatal(err)
}
dump.PrintYAML()

// Inspect specific source caches
vaultEntries := config.Vault.InspectCache()
dbEntries := config.DB.InspectCache()

fmt.Printf("Vault cache: %d items\n", config.Vault.CacheSize())
fmt.Printf("DB cache: %d items\n", config.DB.CacheSize())
```

## Integration Example

```go
func main() {
    // Define CLI flags
    configFormat := flag.String("dump-config", "", "Dump config (json|yaml|env)")
    dumpCached := flag.Bool("dump-cached", false, "Dump cached config (Vault, DB)")
    dumpEffective := flag.Bool("dump-effective", false, "Dump full effective config")
    flag.Parse()

    // Load configuration
    config.Yaml.Load("config.yaml")

    // Check which dump mode user wants
    if *dumpEffective {
        format := *configFormat
        if format == "" {
            format = "yaml"
        }

        fmt.Printf("Effective Configuration:\n")
        fmt.Printf("Vault cache: %d items\n", config.Vault.CacheSize())
        fmt.Printf("DB cache: %d items\n", config.DB.CacheSize())

        config.DumpCachedConfig(config.ConfigFormat(format))
        os.Exit(0)
    }

    // ... continue with normal startup
}
```

## Use Cases

1. **Debugging Cache Issues** - Use `-dump-cached` to see what's cached and when it expires
2. **Configuration Audit** - Use `-dump-effective` to see the full runtime state
3. **Troubleshooting** - See which config values are actually loaded vs. what you expect
4. **Cache Monitoring** - Check cache sizes and expiration times
5. **Development** - Quick way to verify environment variables are being read correctly

## Notes

- Cached configs show the **effective value** (including defaults)
- `time_to_live` is how long until the entry expires
- `is_stale` indicates if the entry is expired but still being used (stale-while-revalidate)
- Sensitive values (passwords, API keys) will be visible - use with caution
- The cache size shown is the number of items currently cached
