# Advanced Configuration Patterns

This guide covers advanced configuration patterns and techniques for building robust, maintainable applications.

## 1. Environment-Specific Configuration

### Multi-Environment Setup

```go
func setupConfig() error {
    // Get environment from configuration or default to development
    env := config.Get("environment", "development").String()

    // Load environment-specific configuration
    configFile := fmt.Sprintf("config.%s.yaml", env)
    if err := config.Yaml.Load(configFile); err != nil {
        return fmt.Errorf("failed to load %s: %w", configFile, err)
    }

    // Environment-specific setup
    switch env {
    case "production":
        return setupProductionConfig()
    case "staging":
        return setupStagingConfig()
    case "development":
        return setupDevelopmentConfig()
    default:
        return fmt.Errorf("unknown environment: %s", env)
    }
}

func setupProductionConfig() error {
    // Production: Full security and monitoring
    vaultClient, err := vault.NewClient(vault.DefaultConfig())
    if err != nil {
        return fmt.Errorf("failed to connect to Vault: %w", err)
    }
    config.Vault.From(vaultClient).WithCache(15 * time.Minute)

    // Database configuration with monitoring
    config.DB.From(dbPool, "app_settings").
        WithCache(5 * time.Minute).
        WithErrorHandler(func(err error) {
            monitoring.Alert("Database configuration error", err)
        })

    return nil
}

func setupDevelopmentConfig() error {
    // Development: Local settings and debugging
    config.DB.From(dbPool, "dev_settings").WithCache(1 * time.Minute)

    // Enable debug logging
    config.SetDebug(true)

    return nil
}
```

### Configuration Validation

```go
type DatabaseConfig struct {
    Host     string `json:"host" validate:"required"`
    Port     int    `json:"port" validate:"min=1,max=65535"`
    User     string `json:"user" validate:"required"`
    Password string `json:"password"`
    Name     string `json:"name" validate:"required"`
    SSLMode  string `json:"ssl_mode" validate:"oneof=disable require verify-full"`
}

type ServerConfig struct {
    Port     int           `json:"port" validate:"min=1,max=65535"`
    Timeout  time.Duration `json:"timeout"`
    Debug    bool          `json:"debug"`
    CORS     CORSConfig    `json:"cors"`
}

type AppConfig struct {
    Database DatabaseConfig `json:"database" validate:"required"`
    Server   ServerConfig   `json:"server" validate:"required"`
    Logging  LoggingConfig  `json:"logging"`
}

func validateConfiguration() error {
    var appConfig AppConfig

    // Load configuration into struct with validation
    if err := config.Get("database").Struct(&appConfig.Database); err != nil {
        return fmt.Errorf("database configuration validation failed: %w", err)
    }

    if err := config.Get("server").Struct(&appConfig.Server); err != nil {
        return fmt.Errorf("server configuration validation failed: %w", err)
    }

    // Environment-specific validation
    env := config.Get("environment", "development").String()
    if env == "production" {
        if appConfig.Server.Debug {
            return errors.New("debug mode cannot be enabled in production")
        }

        if appConfig.Database.SSLMode == "disable" {
            return errors.New("SSL cannot be disabled in production")
        }
    }

    return nil
}
```

## 2. Dynamic Configuration Updates

### Real-time Configuration Updates

```go
func setupConfigurationWatcher() {
    // Watch for configuration changes
    config.Watch(func(change config.Change) {
        log.Printf("Configuration changed: %s = %v", change.Key, change.NewValue)

        // Handle specific changes
        switch change.Key {
        case "server.debug":
            updateLogLevel(change.NewValue.Bool())

        case "database.max_connections":
            updateConnectionPool(change.NewValue.Int())

        case "features.new_ui":
            toggleFeatureFlag("new_ui", change.NewValue.Bool())

        case "maintenance.mode":
            handleMaintenanceMode(change.NewValue.Bool())
        }
    })
}

func updateConnectionPool(maxConnections int) {
    // Update database connection pool
    if pool, ok := dbPool.(*pgxpool.Pool); ok {
        pool.Config().MaxConns = int32(maxConnections)
        log.Printf("Updated connection pool max connections to %d", maxConnections)
    }
}

func toggleFeatureFlag(feature string, enabled bool) {
    // Update feature flags
    featureFlags.Store(feature, enabled)
    log.Printf("Feature %s %s", feature, map[bool]string{true: "enabled", false: "disabled"}[enabled])
}
```

### Configuration Reloading

```go
func setupConfigReloading() {
    // Reload YAML files when they change
    config.Yaml.WatchFiles(func(filename string) {
        log.Printf("Configuration file changed: %s", filename)

        // Validate new configuration
        if err := validateConfiguration(); err != nil {
            log.Printf("Configuration validation failed: %v", err)
            // Revert to previous configuration
            config.Yaml.Reload()
            return
        }

        // Notify other services about configuration change
        notifyConfigChange()
    })
}

func notifyConfigChange() {
    // Send configuration change notification to other services
    // This could be via message queue, HTTP webhook, etc.
    event := ConfigChangeEvent{
        Timestamp: time.Now(),
        Source:    "config-reloader",
        Version:   config.Get("config.version", "unknown").String(),
    }

    if err := eventBus.Publish("config.changed", event); err != nil {
        log.Printf("Failed to publish config change event: %v", err)
    }
}
```

## 3. Configuration Security Patterns

### Secure Secret Management

```go
func setupSecureConfiguration() error {
    env := config.Get("environment", "development").String()

    if env == "production" {
        // Production: Use Vault for all secrets
        vaultClient, err := vault.NewClient(vault.DefaultConfig())
        if err != nil {
            return fmt.Errorf("failed to connect to Vault: %w", err)
        }

        // Configure Vault with secure settings
        config.Vault.From(vaultClient).
            WithCache(15 * time.Minute).
            WithPrefix("secret/myapp/")

        // Validate that required secrets are available
        requiredSecrets := []string{
            "secrets.database_password",
            "secrets.api_key",
            "secrets.jwt_secret",
        }

        for _, secret := range requiredSecrets {
            value := config.Get(secret, "").String()
            if value == "" {
                return fmt.Errorf("required secret not found: %s", secret)
            }
        }
    }

    return nil
}

// Secret rotation handler
func handleSecretRotation() {
    // Watch for secret changes
    config.Vault.Watch(func(path string, data map[string]interface{}) {
        log.Printf("Secret rotation detected for: %s", path)

        switch path {
        case "secret/myapp/database":
            rotateDatabaseCredentials(data)
        case "secret/myapp/api":
            rotateAPIKeys(data)
        }
    })
}

func rotateDatabaseCredentials(newCreds map[string]interface{}) {
    // Safely rotate database credentials
    newPassword := newCreds["password"].(string)

    // Update database connection
    if err := updateDatabasePassword(newPassword); err != nil {
        log.Printf("Failed to rotate database password: %v", err)
        return
    }

    log.Printf("Database credentials rotated successfully")
}
```

### Configuration Encryption

```go
type EncryptedConfig struct {
    EncryptedData string `json:"encrypted_data"`
    IV           string `json:"iv"`
    Tag          string `json:"tag"`
}

func encryptConfig(data []byte, key []byte) (*EncryptedConfig, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }

    encrypted := gcm.Seal(nonce, nonce, data, nil)

    return &EncryptedConfig{
        EncryptedData: hex.EncodeToString(encrypted[gcm.NonceSize():]),
        IV:           hex.EncodeToString(nonce),
        Tag:          hex.EncodeToString(encrypted[len(encrypted)-gcm.NonceSize():]),
    }, nil
}

func decryptConfig(encrypted *EncryptedConfig, key []byte) ([]byte, error) {
    ciphertext, err := hex.DecodeString(encrypted.EncryptedData)
    if err != nil {
        return nil, err
    }

    nonce, err := hex.DecodeString(encrypted.IV)
    if err != nil {
        return nil, err
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    data, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, err
    }

    return data, nil
}
```

## 4. Configuration Migration Patterns

### Versioned Configuration

```go
type ConfigMigration struct {
    Version string
    Migrate func(map[string]interface{}) (map[string]interface{}, error)
}

var migrations = []ConfigMigration{
    {
        Version: "1.0.0",
        Migrate: func(oldConfig map[string]interface{}) (map[string]interface{}, error) {
            // Migrate from old database config structure
            if dbHost, ok := oldConfig["db_host"]; ok {
                if _, hasNew := oldConfig["database"]; !hasNew {
                    oldConfig["database"] = map[string]interface{}{
                        "host": dbHost,
                    }
                    delete(oldConfig, "db_host")
                }
            }
            return oldConfig, nil
        },
    },
    {
        Version: "2.0.0",
        Migrate: func(oldConfig map[string]interface{}) (map[string]interface{}, error) {
            // Add new logging configuration
            if _, hasLogging := oldConfig["logging"]; !hasLogging {
                oldConfig["logging"] = map[string]interface{}{
                    "level":  "info",
                    "format": "text",
                }
            }
            return oldConfig, nil
        },
    },
}

func migrateConfiguration(configData map[string]interface{}) error {
    currentVersion := config.Get("config.version", "1.0.0").String()

    for _, migration := range migrations {
        if compareVersions(currentVersion, migration.Version) < 0 {
            log.Printf("Applying migration %s", migration.Version)

            var err error
            configData, err = migration.Migrate(configData)
            if err != nil {
                return fmt.Errorf("migration %s failed: %w", migration.Version, err)
            }

            // Update version
            configData["config_version"] = migration.Version
        }
    }

    return nil
}
```

### Configuration Backup and Restore

```go
func backupConfiguration() error {
    // Get current configuration
    configData := make(map[string]interface{})

    // Backup from all sources
    if dbData := config.DB.GetAll(""); len(dbData) > 0 {
        configData["database"] = dbData
    }

    if vaultData := config.Vault.GetAll(""); len(vaultData) > 0 {
        configData["vault"] = vaultData
    }

    // Add metadata
    configData["backup_timestamp"] = time.Now()
    configData["version"] = config.Get("config.version", "unknown").String()

    // Save backup
    backupFile := fmt.Sprintf("config_backup_%s.json",
        time.Now().Format("20060102_150405"))

    backupData, err := json.MarshalIndent(configData, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal backup: %w", err)
    }

    if err := os.WriteFile(backupFile, backupData, 0600); err != nil {
        return fmt.Errorf("failed to write backup file: %w", err)
    }

    log.Printf("Configuration backed up to %s", backupFile)
    return nil
}

func restoreConfiguration(backupFile string) error {
    // Load backup
    backupData, err := os.ReadFile(backupFile)
    if err != nil {
        return fmt.Errorf("failed to read backup file: %w", err)
    }

    var configData map[string]interface{}
    if err := json.Unmarshal(backupData, &configData); err != nil {
        return fmt.Errorf("failed to unmarshal backup: %w", err)
    }

    // Restore configuration
    if dbData, ok := configData["database"].(map[string]interface{}); ok {
        for key, value := range dbData {
            config.DB.Set(key, value)
        }
    }

    if vaultData, ok := configData["vault"].(map[string]interface{}); ok {
        for key, value := range vaultData {
            config.Vault.Set(key, value)
        }
    }

    log.Printf("Configuration restored from %s", backupFile)
    return nil
}
```

## 5. Configuration Testing Patterns

### Test Configuration Setup

```go
func setupTestConfig() {
    // Reset configuration for tests
    config.Reset()

    // Set test-specific values
    config.Set("environment", "test")
    config.Set("database.host", "localhost")
    config.Set("database.port", 5432)
    config.Set("server.port", 8080)
    config.Set("debug.enabled", true)

    // Use in-memory database for tests
    testDB := setupTestDatabase()
    config.DB.From(testDB, "test_settings").WithCache(0)
}

func TestConfigurationHierarchy(t *testing.T) {
    setupTestConfig()

    tests := []struct {
        name     string
        key      string
        fallback interface{}
        expected interface{}
    }{
        {
            name:     "database host from test config",
            key:      "database.host",
            fallback: "default",
            expected: "localhost",
        },
        {
            name:     "server port from test config",
            key:      "server.port",
            fallback: 3000,
            expected: 8080,
        },
        {
            name:     "debug mode from test config",
            key:      "debug.enabled",
            fallback: false,
            expected: true,
        },
        {
            name:     "fallback value",
            key:      "nonexistent.key",
            fallback: "default_value",
            expected: "default_value",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            switch tt.expected.(type) {
            case string:
                result := config.Get(tt.key, tt.fallback).String()
                assert.Equal(t, tt.expected.(string), result)
            case int:
                result := config.Get(tt.key, tt.fallback).Int()
                assert.Equal(t, tt.expected.(int), result)
            case bool:
                result := config.Get(tt.key, tt.fallback).Bool()
                assert.Equal(t, tt.expected.(bool), result)
            }
        })
    }
}

func TestConfigurationValidation(t *testing.T) {
    setupTestConfig()

    // Test valid configuration
    validConfig := DatabaseConfig{
        Host:    "localhost",
        Port:    5432,
        User:    "testuser",
        Name:    "testdb",
        SSLMode: "require",
    }

    if err := config.Get("database").Struct(&validConfig); err != nil {
        t.Errorf("Valid configuration failed validation: %v", err)
    }

    // Test invalid configuration
    invalidConfig := DatabaseConfig{
        Host:    "", // Empty host should fail validation
        Port:    99999, // Invalid port should fail validation
        User:    "testuser",
        Name:    "testdb",
        SSLMode: "invalid_mode", // Invalid SSL mode should fail validation
    }

    if err := config.Get("database").Struct(&invalidConfig); err == nil {
        t.Error("Invalid configuration should have failed validation")
    }
}
```

These advanced patterns help you build robust, maintainable configuration systems that can handle complex requirements across different environments and deployment scenarios.