# APP Package Migration Guide

This guide helps you migrate from the original APP package structure to the enhanced version with simplified initialization, unified router interface, and improved configuration management.

## Key Changes

### 1. **Simplified Application Structure**
The original package had complex router initialization and manual configuration loading. The new version provides a streamlined initialization process with framework-agnostic router support.

### 2. **Unified Router Interface**
Moved from framework-specific routers (Gin) to a unified router interface that supports multiple web frameworks through adapters.

### 3. **Configuration Management**
Centralized configuration through the `config` package instead of scattered environment variable handling.

### 4. **Improved Logging**
Migrated from standard log to structured logging with `log/slog` integration.

### 5. **Enhanced Consul Integration**
Simplified service registration with improved error handling and configuration.

## Migration Steps

### Step 1: Update Application Initialization

#### Old API
```go
func main() {
    // Manual configuration loading
    if err := godotenv.Load(); err != nil {
        log.Printf("error loading env files. %v", err)
    }

    // Direct service creation with manual dependencies
    userService := users.NewService(database.Connect())
    authService := auth.NewService(database.Connect(), os.Getenv("AUTH_SECRET"))

    // Manual router setup with CORS middleware
    router := gin.New()
    router.Use(gin.Recovery(), cors.New(cors.Config{
        AllowOrigins:     []string{"*"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
        AllowHeaders:     []string{"Origin", "Content-Length", "Accept-Encoding", "Authorization"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }))

    // Manual service registration with Consul
    consulClient, err := consulapi.NewClient(consulapi.DefaultConfig())
    if err != nil {
        log.Fatal("Failed to create Consul client:", err)
    }

    serviceID := fmt.Sprintf("my-service-%s-%s", os.Getenv("HOST"), "1.0.0")
    err = consulClient.Agent().ServiceRegister(&consulapi.AgentServiceRegistration{
        ID:      serviceID,
        Name:    "my-service",
        Port:    8080,
        Address: os.Getenv("HOST"),
        Check: &consulapi.AgentServiceCheck{
            HTTP:     fmt.Sprintf("http://%s:8080/ok", os.Getenv("HOST")),
            Interval: "10s",
            Timeout:  "30s",
        },
    })
    if err != nil {
        log.Fatal("Failed to register service:", err)
    }

    // Setup routes
    setupRoutes(router, userService, authService)

    // Basic HTTP server with signal handling
    server := &http.Server{
        Addr:    ":8080",
        Handler: router,
    }

    go func() {
        log.Println("Starting server on port 8080")
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal("Server failed:", err)
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    <-quit

    // Graceful shutdown
    log.Println("Shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        log.Fatal("Server forced to shutdown:", err)
    }

    // Deregister from Consul
    consulClient.Agent().ServiceDeregister(serviceID)
    log.Println("Server exited")
}
```

#### New API
```go
func main() {
    // Initialize router framework (Gin in this case)
    gin.Setup()

    // Simplified application initialization
    application := app.Init(
        app.WithHeartbeatHandlers(), // Auto-adds /ok and /stats endpoints
    )

    // Register routes using unified router interface
    application.R().GET("/users", func(ctx api.RequestContext) {
        // Handle user listing with proper error handling
        users, err := userService.GetAll(ctx.Context())
        if err != nil {
            ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        ctx.JSON(http.StatusOK, users)
    })

    application.R().POST("/users", func(ctx api.RequestContext) {
        var req users.CreateRequest
        if err := ctx.ShouldBindJSON(&req); err != nil {
            ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }

        user, err := userService.Create(ctx.Context(), req)
        if err != nil {
            ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        ctx.JSON(http.StatusCreated, user)
    })

    // Run the application - handles Consul registration and graceful shutdown automatically
    application.Run("my-service")
}
```

### Step 2: Update Configuration Management

#### Old API
```go
// Manual environment variable handling throughout the code
var host = utils.Env.GetHost()
var env = utils.Env.Helper("APP")

_config = &conf{
    Host:        host,
    Name:        env.Get("NAME", host).String(),
    Version:     env.Get("VERSION", "ver-0.0.0").String(),
    HttpPort:    env.Get("HTTP_PORT", "49080").Int(),
    HttpAddress: env.Get("HTTP_ADDRESS", "0.0.0.0").String(),
    Location:    env.Get("TIME_LOCATION", "Africa/Kampala").Location(),
    Uptime:      UptimeCheckConf(env.Prefix(), "UPTIME_CHECK"),
    Http:        HttpConf(env.Prefix(), "HTTP_SERVER"),
}

// Manual Consul configuration
var env = utils.Env.Helper("CONSUL")
if consulAddress := env.Get("HTTP_ADDR", ""); !consulAddress.Valid() {
    log.Printf("env var %s_HTTP_ADDR not set. skipping consul initialization", env.Prefix())
}

// Scattered HTTP server configuration
type httpConf struct {
    AllowOrigins     []string
    AllowMethods     []string
    AllowHeaders     []string
    ExposeHeaders    []string
    MaxAge           time.Duration
    AllowCredentials bool
}

func HttpConf(prefix ...string) (conf *httpConf) {
    env := utils.Env.Helper(prefix...).OrDefault("HTTP_SERVER")
    return &httpConf{
        AllowOrigins:     env.Get("ALLOW_ORIGINS", "*").StringList(","),
        AllowMethods:     env.Get("ALLOW_METHODS", "GET,POST,PUT,HEAD,OPTIONS").StringList(","),
        AllowHeaders:     env.Get("ALLOW_HEADERS", "Origin,Content-Length,Accept-Encoding,Authorization,Accept-Language,Content-Type").StringList(","),
        ExposeHeaders:    env.Get("EXPOSE_HEADERS", "Content-Length,Host,Content-Type,Connection").StringList(","),
        MaxAge:           env.Get("MAX_AGE", "12h").Duration(),
        AllowCredentials: env.Get("ALLOW_CREDENTIALS", "true").Bool(),
    }
}
```

#### New API
```go
// Centralized configuration through config package
var host = config.Get("host").String()

_config = &conf{
    Host:        host,
    Name:        config.Get("APP_NAME", host).String(),
    Version:     config.Get("APP_VERSION", "ver-0.0.0").String(),
    HttpPort:    config.Get("APP_HTTP_PORT", "9025").Int(),
    HttpAddress: config.Get("APP_HTTP_ADDRESS", "0.0.0.0").String(),
    Location:    config.Get("APP_TIME_LOCATION", "Africa/Kampala").Location(),
    Uptime:      UptimeCheckConf(),
}

// Simplified uptime configuration
func UptimeCheckConf() (conf *uptimeCheckConf) {
    return &uptimeCheckConf{
        Interval: config.Get("UPTIME_CHECK_INTERVAL", "10s").Duration(),
        Timeout:  config.Get("UPTIME_CHECK_TIMEOUT", "30s").Duration(),
    }
}

// Consul configuration moved to config package with auto-registration
func (a *ap) tryRegisterConsul(name ...string) {
    id, err := config.Consul.RegisterFromEnv(name...)
    if err != nil {
        if strings.Contains(err.Error(), "consul address not configured") {
            a.log.Warn("skipping consul registration", "reason", "consul address not configured")
        } else {
            a.log.Error("service registration failed", "error", err)
        }
        return
    }
    a.serviceId = id
    a.log.Info("service successfully registered with consul", "id", a.serviceId)
}

// HTTP server configuration moved to api package with router config
routerConfig := api.DefaultRouterConfig()
a.router, err = api.CreateRouter(routerConfig)
```

### Step 3: Update Router and Framework Integration

#### Old API
```go
// Framework-specific router setup with manual middleware
func (a *ap) initAPI(opts ...AppIniter) {
    a.handlers = map[string]gin.HandlerFunc{
        "ok": func(c *gin.Context) {
            c.JSON(http.StatusOK, "OK")
        },
        "stats": func(c *gin.Context) {
            c.JSON(http.StatusOK, map[string]interface{}{
                "host": a.conf.Host, "started": a.start, "app": a.conf.Name,
                "uptime": time.Since(a.start).Round(100 * time.Millisecond).String(),
            })
        },
    }

    a.router = gin.New()
    a.router.Use(api.JSONLogMiddleware(a.log), gin.Recovery(), cors.New(cors.Config{
        AllowOrigins:     a.conf.Http.AllowOrigins,
        AllowMethods:     a.conf.Http.AllowMethods,
        AllowHeaders:     a.conf.Http.ExposeHeaders,
        AllowCredentials: a.conf.Http.AllowCredentials,
        MaxAge:           a.conf.Http.MaxAge,
    }))

    for i := range opts {
        opts[i](a)
    }

    if a.heartbeatHandlers {
        a.router.GET("/", a.handlers["ok"])
        a.router.GET("/ok", a.handlers["ok"])
        a.router.GET("/stats", a.handlers["stats"])
    }
}

// Direct Gin router access
func (a *ap) Router() *gin.Engine {
    return a.router
}

func (a *ap) R() *gin.Engine {
    return a.router
}
```

#### New API
```go
// Framework-agnostic router initialization
func (a *ap) initAPI(opts ...AppIniter) {
    // Create router using unified configuration from environment
    routerConfig := api.DefaultRouterConfig()
    var err error
    a.router, err = api.CreateRouter(routerConfig)
    if err != nil {
        panic(err)
    }

    for i := range opts {
        opts[i](a)
    }

    if a.heartbeatHandlers {
        // Use new router interface with wrapped handlers
        a.router.GET("/", func(ctx contracts.RequestContext) {
            if appCtx, ok := ctx.(api.RequestContext); ok {
                appCtx.JSON(http.StatusOK, "OK")
            }
        })
        a.router.GET("/ok", func(ctx contracts.RequestContext) {
            if appCtx, ok := ctx.(api.RequestContext); ok {
                appCtx.JSON(http.StatusOK, "OK")
            }
        })
        a.router.GET("/stats", func(ctx contracts.RequestContext) {
            if appCtx, ok := ctx.(api.RequestContext); ok {
                appCtx.JSON(http.StatusOK, map[string]any{
                    "host": a.conf.Host, "started": a.start, "app": a.conf.Name,
                    "uptime": time.Since(a.start).Round(100 * time.Millisecond).String(),
                })
            }
        })
    }
}

// Unified router interface
func (a *ap) Router() api.Router {
    return a.router
}

func (a *ap) R() api.Router {
    return a.router
}

// Framework initialization (e.g., Gin setup)
func main() {
    // Initialize router framework first
    gin.Setup()

    // Then initialize app - it will use the configured router
    application := app.Init()
    application.Run()
}
```

### Step 4: Update Logging and Error Handling

#### Old API
```go
// Manual log configuration and error handling
func Init(opts ...AppIniter) *ap {
    if err := godotenv.Load(); err != nil {
        if strictEnv {
            log.Fatalf("error loading env files. %v", err)
        }
        log.Printf("error loading env files. %v", err)
    }
    if err := logr.Config(); err != nil {
        logr.Log().WithError(err).Fatal("Application log initialisation failed")
    }
    logr.Log().Println("starting app initialisation")
    // ...
}

// Manual Consul error handling
func (a *ap) Register(name ...string) (err error) {
    if a.consul != nil {
        return utils.Error.LogOK(a.log.Infof, "service already registered with consul")
    }
    if consulAddress := env.Get("HTTP_ADDR", ""); !consulAddress.Valid() {
        return utils.Error.LogOK(a.log.Warnf, "env var %s_HTTP_ADDR not set. skipping consul initialization", env.Prefix())
    }
    // Complex error handling with utils.Error.LogOK pattern
}

// Standard log output throughout
a.log.Printf("service successfully registered with consul")
a.log.Printf("shutting down")
```

#### New API
```go
// Structured logging with log/slog
func Init(opts ...AppIniter) *ap {
    config.Load() // Centralized config loading

    if err := logr.Config(); err != nil {
        slog.Error("Application log initialisation failed", "error", err)
        panic(err)
    }
    logr.Log().Info("starting app initialisation")
    // ...
}

// Improved Consul error handling with structured logging
func (a *ap) tryRegisterConsul(name ...string) {
    id, err := config.Consul.RegisterFromEnv(name...)
    if err != nil {
        // Check if the error is because the endpoint is not configured
        if strings.Contains(err.Error(), "consul address not configured") {
            a.log.Warn("skipping consul registration", "reason", "consul address not configured")
        } else {
            a.log.Error("service registration failed", "error", err)
        }
        return
    }
    a.serviceId = id
    a.log.Info("service successfully registered with consul", "id", a.serviceId)
}

// Structured logging throughout
a.log.Info("starting app initialisation")
a.log.Info("service successfully registered with consul", "id", a.serviceId)
a.log.Info("shutting down")
a.log.Warn("failed to deregister service from consul", "error", err)

// Graceful shutdown with improved logging
func (a *ap) Run(name ...string) {
    fmt.Println()
    a.tryRegisterConsul(name...)
    signal.Notify(a.osc, os.Interrupt, syscall.SIGTERM)
    startupMsg := "started"
    if a.router != nil {
        startupMsg += " with http router " + a.conf.Address()
        go a.router.Run(a.conf.Address())
    }
    a.log.Info(startupMsg)
    <-a.osc
    a.cancel()
    fmt.Println()
    a.shutdown()

    if err := config.Consul.Deregister(a.serviceId); err != nil {
        a.log.Warn("failed to deregister service from consul", "error", err)
    } else if a.serviceId != "" {
        a.log.Info("service successfully deregistered from consul")
    }
    a.log.Info("shutdown complete")
}
```

## Practical Examples

### Example 1: Basic Service with Gin Framework

#### Before (Old API - Complex Setup)
```go
func main() {
    // Manual environment loading
    if err := godotenv.Load(); err != nil {
        log.Printf("error loading env files. %v", err)
    }

    // Manual router setup
    router := gin.New()
    router.Use(api.JSONLogMiddleware(logr.Log()), gin.Recovery())

    // Manual CORS configuration
    router.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"*"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }))

    // Manual Consul setup
    consulClient, err := consulapi.NewClient(consulapi.DefaultConfig())
    if err != nil {
        log.Fatal("Failed to create Consul client:", err)
    }

    serviceID := "my-service-" + os.Getenv("HOST") + "-1.0.0"
    err = consulClient.Agent().ServiceRegister(&consulapi.AgentServiceRegistration{
        ID:      serviceID,
        Name:    "my-service",
        Port:    8080,
        Address: os.Getenv("HOST"),
        Check: &consulapi.AgentServiceCheck{
            HTTP:     "http://" + os.Getenv("HOST") + ":8080/ok",
            Interval: "10s",
            Timeout:  "30s",
        },
    })
    if err != nil {
        log.Fatal("Failed to register service:", err)
    }

    // Manual health check endpoints
    router.GET("/ok", func(c *gin.Context) {
        c.JSON(http.StatusOK, "OK")
    })

    router.GET("/stats", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "app":     "my-service",
            "version": "1.0.0",
            "uptime":  "5m",
        })
    })

    // Business logic
    router.GET("/users", func(c *gin.Context) {
        users, err := userService.GetAll()
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
        c.JSON(200, users)
    })

    // Start server
    server := &http.Server{Addr: ":8080", Handler: router}
    go func() {
        log.Println("Starting server on :8080")
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal("Server failed:", err)
        }
    }()

    // Wait for shutdown signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    <-quit

    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    server.Shutdown(ctx)
    consulClient.Agent().ServiceDeregister(serviceID)
    log.Println("Server exited")
}
```

#### After (New API - Simplified Setup)
```go
func main() {
    // Initialize Gin framework
    gin.Setup()

    // Initialize application with heartbeat handlers
    application := app.Init(
        app.WithHeartbeatHandlers(), // Auto-adds /ok and /stats endpoints
    )

    // Register routes using unified router interface
    application.R().GET("/users", func(ctx api.RequestContext) {
        users, err := userService.GetAll(ctx.Context())
        if err != nil {
            ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        ctx.JSON(http.StatusOK, users)
    })

    application.R().POST("/users", func(ctx api.RequestContext) {
        var req users.CreateUserRequest
        if err := ctx.ShouldBindJSON(&req); err != nil {
            ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }

        user, err := userService.Create(ctx.Context(), req)
        if err != nil {
            ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        ctx.JSON(http.StatusCreated, user)
    })

    // Run application - handles Consul registration, graceful shutdown automatically
    application.Run("my-service")
}
```

### Example 2: Framework Configuration and Custom Middleware

#### Before (Old API - Hardcoded Framework Setup)
```go
func main() {
    // Hardcoded Gin setup with manual middleware
    router := gin.New()

    // Manual middleware configuration
    router.Use(api.JSONLogMiddleware(logr.Log()), gin.Recovery())
    router.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"*"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }))

    // Manual static file serving
    router.Static("/static", "./static")
    router.StaticFile("/favicon.ico", "./static/favicon.ico")

    // Manual route setup
    setupRoutes(router)

    // Start server manually
    server := &http.Server{Addr: ":8080", Handler: router}
    log.Fatal(server.ListenAndServe())
}
```

#### After (New API - Flexible Framework Configuration)
```go
func main() {
    // Configure Gin with custom options
    gin.SetupWithOptions(func(config *api.RouterConfig) {
        config.AllowOrigins = []string{"https://myapp.com", "https://admin.myapp.com"}
        config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
        config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "X-Requested-With"}
        config.EnableRecovery = true
        config.EnableLogging = true

        // Add custom middleware
        config.CustomMiddleware = []api.MiddlewareFunc{
            func(ctx contracts.RequestContext) {
                // Custom request ID middleware
                requestID := uuid.New().String()
                ctx.SetHeader("X-Request-ID", requestID)
                ctx.SetValue("request_id", requestID)
            },
            func(ctx contracts.RequestContext) {
                // Custom metrics middleware
                start := time.Now()
                defer func() {
                    duration := time.Since(start)
                    metrics.RecordRequestDuration(ctx.Method(), ctx.Path(), duration)
                }()
            },
        }
    })

    // Initialize application with custom options
    application := app.Init(
        app.WithHeartbeatHandlers(),
        app.WithStaticFileHandler("/static", "./static"),
    )

    // Register routes using the unified interface
    application.R().GET("/api/users", api.Handler[users.ListRequest, users.ListResponse](func(ctx context.Context, req users.ListRequest) (users.ListResponse, error) {
        return userService.List(ctx, req)
    }))

    application.R().POST("/api/users", api.Handler[users.CreateRequest, users.UserResponse](func(ctx context.Context, req users.CreateRequest) (users.UserResponse, error) {
        return userService.Create(ctx, req)
    }))

    // Run application
    application.Run("user-service")
}
```

### Example 3: Service Registration with Error Handling

#### Before (Old API - Manual Error Handling)
```go
func main() {
    // Manual environment loading with strict mode
    if err := godotenv.Load(); err != nil {
        if strictEnv {
            log.Fatalf("error loading env files. %v", err)
        }
        log.Printf("error loading env files. %v", err)
    }

    // Manual Consul setup with complex error handling
    env := utils.Env.Helper("CONSUL")
    if consulAddress := env.Get("HTTP_ADDR", ""); !consulAddress.Valid() {
        log.Printf("env var %s_HTTP_ADDR not set. skipping consul initialization", env.Prefix())
        return
    }

    config := consulapi.DefaultConfig()
    consulClient, err := consulapi.NewClient(config)
    if err != nil {
        log.Fatalf("consul client initialisation failed: %v", err)
    }

    serviceID := "my-service"
    err = consulClient.Agent().ServiceRegister(&consulapi.AgentServiceRegistration{
        ID:      serviceID,
        Name:    "my-service",
        Port:    8080,
        Address: "localhost",
        Check: &consulapi.AgentServiceCheck{
            HTTP:     "http://localhost:8080/ok",
            Interval: "10s",
            Timeout:  "30s",
        },
    })
    if err != nil {
        log.Fatalf("service registration failed: %v", err)
    }

    utils.Error.LogOK(logr.Log().Infof, "service successfully registered with consul")

    // Start server
    // ... server setup code
}
```

#### After (New API - Simplified Service Registration)
```go
func main() {
    // Initialize framework
    gin.Setup()

    // Initialize application - handles configuration automatically
    application := app.Init(
        app.WithHeartbeatHandlers(),
    )

    // Service registration is handled automatically with proper error handling
    // The app package will:
    // 1. Check if Consul is configured via environment variables
    // 2. Skip registration if not configured (development mode)
    // 3. Register with proper error handling if configured
    // 4. Log appropriate messages based on the outcome
    // 5. Handle graceful deregistration on shutdown

    // Register business routes
    setupRoutes(application)

    // Run - handles all service registration, server startup, and graceful shutdown
    application.Run("my-service")
}

// The actual service registration logic is now centralized and robust:
func (a *ap) tryRegisterConsul(name ...string) {
    id, err := config.Consul.RegisterFromEnv(name...)
    if err != nil {
        // Check if the error is because the endpoint is not configured
        if strings.Contains(err.Error(), "consul address not configured") {
            a.log.Warn("skipping consul registration", "reason", "consul address not configured")
        } else {
            a.log.Error("service registration failed", "error", err)
        }
        return
    }
    a.serviceId = id
    a.log.Info("service successfully registered with consul", "id", a.serviceId)
}
```

## Migration Benefits Summary

### 1. **Simplified Application Initialization**
- **Before**: Complex manual setup with environment loading, router configuration, middleware setup, and service registration
- **After**: Streamlined initialization with `app.Init()` that handles configuration, logging, and framework setup automatically

### 2. **Framework Flexibility**
- **Before**: Tightly coupled to Gin with direct `*gin.Engine` dependencies
- **After**: Framework-agnostic router interface supporting multiple web frameworks through adapters

### 3. **Centralized Configuration**
- **Before**: Scattered environment variable handling with `utils.Env.Helper()` throughout the code
- **After**: Centralized configuration through the `config` package with consistent environment variable handling

### 4. **Improved Error Handling**
- **Before**: Complex error handling patterns with `utils.Error.LogOK()` and manual Consul error management
- **After**: Structured error handling with proper logging levels and graceful degradation when services are unavailable

### 5. **Automated Service Management**
- **Before**: Manual Consul client setup, service registration, and deregistration with complex error handling
- **After**: Automated service registration with proper error handling, graceful degradation for development, and automatic cleanup

### 6. **Enhanced Logging**
- **Before**: Basic `log.Printf()` statements and manual log formatting
- **After**: Structured logging with `log/slog` integration, contextual logging, and proper log levels

### 7. **Reduced Boilerplate**
- **Before**: 100+ lines of manual setup code for each service
- **After**: ~10 lines of initialization code with automatic handling of common concerns

### 8. **Better Development Experience**
- **Before**: Complex setup that required understanding of multiple internal systems
- **After**: Simple initialization that works consistently across development and production environments

## Testing Your Migration

### 1. **Basic Integration Testing**
```go
func TestApplicationInitialization(t *testing.T) {
    // Test that application initializes correctly
    gin.Setup()

    application := app.Init(
        app.WithHeartbeatHandlers(),
    )

    // Test that router is accessible
    router := application.R()
    if router == nil {
        t.Fatal("Router should not be nil after initialization")
    }

    // Test that configuration is loaded
    config := application.Config()
    if config == nil {
        t.Fatal("Configuration should not be nil after initialization")
    }

    // Test that logger is initialized
    logger := application.Log()
    if logger == nil {
        t.Fatal("Logger should not be nil after initialization")
    }

    // Test heartbeat endpoints are registered (would require actual HTTP testing)
    t.Log("Application initialized successfully")
}
```

### 2. **Configuration Testing**
```go
func TestConfigurationLoading(t *testing.T) {
    // Test that configuration loads from environment
    os.Setenv("APP_NAME", "test-service")
    os.Setenv("APP_HTTP_PORT", "9025")
    defer os.Clearenv()

    config := app.Conf()

    if config.Name != "test-service" {
        t.Errorf("Expected name 'test-service', got '%s'", config.Name)
    }

    if config.HttpPort != 9025 {
        t.Errorf("Expected port 9025, got %d", config.HttpPort)
    }
}
```

### 3. **Router Interface Testing**
```go
func TestRouterInterface(t *testing.T) {
    gin.Setup()

    application := app.Init()
    router := application.R()

    // Test that we can register routes using the unified interface
    called := false
    router.GET("/test", func(ctx contracts.RequestContext) {
        called = true
        if appCtx, ok := ctx.(api.RequestContext); ok {
            appCtx.JSON(http.StatusOK, gin.H{"message": "test"})
        }
    })

    if !called {
        t.Error("Route handler should be callable")
    }
}
```

## Rollback Plan

If you encounter issues during migration:

1. **Revert to Manual Setup**: Temporarily use the old pattern of manual router initialization and service registration
2. **Skip Framework Setup**: If the new framework adapters cause issues, revert to direct Gin usage
3. **Disable Consul**: Set `CONSUL_HTTP_ADDR=""` to skip service registration if it causes problems
4. **Check Environment Variables**: Ensure all required environment variables are properly set
5. **Gradual Migration**: Migrate one service at a time rather than all at once

## Common Migration Issues

### 1. **Framework Initialization Order**
- **Issue**: Calling `app.Init()` before framework setup
- **Solution**: Always call framework setup (e.g., `gin.Setup()`) before `app.Init()`

### 2. **Missing Environment Variables**
- **Issue**: Configuration fails to load due to missing environment variables
- **Solution**: Check the `config` package documentation for required variables and their defaults

### 3. **Router Interface Changes**
- **Issue**: Code expecting `*gin.Engine` but getting `api.Router`
- **Solution**: Use the unified router interface or cast to framework-specific router when needed

### 4. **Consul Configuration**
- **Issue**: Service registration fails due to Consul not being configured
- **Solution**: The new API gracefully handles missing Consul configuration, but ensure logs show appropriate warnings

## Conclusion

The enhanced APP package provides significant improvements in application initialization, framework flexibility, and operational simplicity. The migration enables:

- **Simplified Setup**: Reduce ~100 lines of boilerplate to ~10 lines of initialization code
- **Framework Flexibility**: Easy to switch between web frameworks or support multiple frameworks
- **Better Error Handling**: Structured logging and graceful degradation when services are unavailable
- **Consistent Configuration**: Centralized configuration management across all services
- **Reduced Complexity**: Automatic handling of common concerns like service registration and health checks

The migration examples demonstrate how the new application structure eliminates operational overhead while making services more maintainable and easier to understand. The focus is on getting developers productive quickly with sensible defaults that work across development and production environments.