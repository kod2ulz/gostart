# API Middleware and Request Processing

This guide covers middleware patterns and request processing techniques for building robust APIs in GoStart.

## Middleware Architecture

### Chain of Responsibility Pattern

Middleware in GoStart follows the chain of responsibility pattern, allowing you to compose multiple middleware functions:

```go
func setupRouter() {
    router := gin.Default()

    // Global middleware
    router.Use(middleware.Logger())
    router.Use(middleware.Recovery())
    router.Use(middleware.CORS())

    // Route-specific middleware
    api := router.Group("/api")
    api.Use(middleware.Authenticate())
    api.Use(middleware.RateLimit(100))

    // Handler-specific middleware
    admin := api.Group("/admin")
    admin.Use(middleware.RequireRole("admin"))

    admin.GET("/users", GetUsersHandler)
}
```

### Framework-Agnostic Middleware

The API package provides framework-agnostic middleware interfaces:

```go
type Middleware interface {
    Process(next Handler) Handler
}

type MiddlewareFunc func(Handler) Handler

func (f MiddlewareFunc) Process(next Handler) Handler {
    return f(next)
}

// Usage
func LoggingMiddleware() MiddlewareFunc {
    return func(next Handler) Handler {
        return func(ctx contracts.RequestContext) (interface{}, ierrors.Error) {
            start := time.Now()
            log.Printf("Starting request: %s %s", ctx.Method(), ctx.Path())

            result, err := next(ctx)

            duration := time.Since(start)
            log.Printf("Request completed in %v", duration)

            return result, err
        }
    }
}
```

## Core Middleware Patterns

### Authentication and Authorization

```go
type AuthMiddleware struct {
    tokenValidator TokenValidator
    userResolver   UserResolver
}

func NewAuthMiddleware(validator TokenValidator, resolver UserResolver) *AuthMiddleware {
    return &AuthMiddleware{
        tokenValidator: validator,
        userResolver:   resolver,
    }
}

func (m *AuthMiddleware) Process(next Handler) Handler {
    return func(ctx contracts.RequestContext) (interface{}, ierrors.Error) {
        // Extract token from Authorization header
        token := ctx.Header("Authorization")
        if token == "" {
            return nil, errors.Unauthorized("missing authorization token")
        }

        // Remove "Bearer " prefix
        token = strings.TrimPrefix(token, "Bearer ")

        // Validate token
        claims, err := m.tokenValidator.Validate(token)
        if err != nil {
            return nil, errors.Unauthorized("invalid token")
        }

        // Resolve user
        user, err := m.userResolver.Resolve(claims.Subject)
        if err != nil {
            return nil, errors.Unauthorized("user not found")
        }

        // Add user to context
        ctx.Set("user", user)
        ctx.Set("user_id", user.ID)

        return next(ctx)
    }
}
```

### Role-Based Authorization

```go
type RoleMiddleware struct {
    requiredRoles []string
    roleChecker   RoleChecker
}

func NewRoleMiddleware(roles []string, checker RoleChecker) *RoleMiddleware {
    return &RoleMiddleware{
        requiredRoles: roles,
        roleChecker:   checker,
    }
}

func (m *RoleMiddleware) Process(next Handler) Handler {
    return func(ctx contracts.RequestContext) (interface{}, ierrors.Error) {
        user, exists := ctx.Get("user")
        if !exists {
            return nil, errors.Unauthorized("user not authenticated")
        }

        hasRequiredRole := false
        for _, requiredRole := range m.requiredRoles {
            if m.roleChecker.HasRole(user, requiredRole) {
                hasRequiredRole = true
                break
            }
        }

        if !hasRequiredRole {
            return nil, errors.Forbidden("insufficient permissions")
        }

        return next(ctx)
    }
}

// Usage
adminOnly := NewRoleMiddleware([]string{"admin"}, roleChecker)
editorOrAdmin := NewRoleMiddleware([]string{"editor", "admin"}, roleChecker)
```

### Rate Limiting

```go
type RateLimitMiddleware struct {
    limiter      RateLimiter
    keyExtractor func(ctx contracts.RequestContext) string
    limit        int
    window       time.Duration
}

func NewRateLimitMiddleware(limiter RateLimiter, keyExtractor func(ctx contracts.RequestContext) string, limit int, window time.Duration) *RateLimitMiddleware {
    return &RateLimitMiddleware{
        limiter:      limiter,
        keyExtractor: keyExtractor,
        limit:        limit,
        window:       window,
    }
}

func (m *RateLimitMiddleware) Process(next Handler) Handler {
    return func(ctx contracts.RequestContext) (interface{}, ierrors.Error) {
        key := m.keyExtractor(ctx)

        if !m.limiter.Allow(key, m.limit, m.window) {
            return nil, errors.TooManyRequests("rate limit exceeded")
        }

        return next(ctx)
    }
}

// Key extractors
func ClientIPExtractor(ctx contracts.RequestContext) string {
    return ctx.Header("X-Forwarded-For")
}

func UserIDExtractor(ctx contracts.RequestContext) string {
    if userID, exists := ctx.Get("user_id"); exists {
        return userID.(string)
    }
    return ClientIPExtractor(ctx)
}
```

### Request Validation

```go
type ValidationMiddleware struct {
    validator RequestValidator
}

func NewValidationMiddleware(validator RequestValidator) *ValidationMiddleware {
    return &ValidationMiddleware{
        validator: validator,
    }
}

func (m *ValidationMiddleware) Process(next Handler) Handler {
    return func(ctx contracts.RequestContext) (interface{}, ierrors.Error) {
        // Load request parameters
        var req contracts.RequestParam
        if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
            return nil, errors.BadRequest(err)
        }

        // Validate request
        if err := m.validator.Validate(req); err != nil {
            return nil, errors.BadRequest(err)
        }

        return next(ctx)
    }
}
```

## Advanced Middleware Patterns

### Request/Response Transformation

```go
type TransformMiddleware struct {
    requestTransformer  func(contracts.RequestContext) contracts.RequestContext
    responseTransformer func(interface{}) interface{}
}

func NewTransformMiddleware(
    requestTrans func(contracts.RequestContext) contracts.RequestContext,
    responseTrans func(interface{}) interface{},
) *TransformMiddleware {
    return &TransformMiddleware{
        requestTransformer:  requestTrans,
        responseTransformer: responseTrans,
    }
}

func (m *TransformMiddleware) Process(next Handler) Handler {
    return func(ctx contracts.RequestContext) (interface{}, ierrors.Error) {
        // Transform request
        if m.requestTransformer != nil {
            ctx = m.requestTransformer(ctx)
        }

        // Call next handler
        result, err := next(ctx)
        if err != nil {
            return nil, err
        }

        // Transform response
        if m.responseTransformer != nil {
            result = m.responseTransformer(result)
        }

        return result, nil
    }
}
```

### Caching Middleware

```go
type CacheMiddleware struct {
    cache       Cache
    keyGenerator func(ctx contracts.RequestContext) string
    ttl         time.Duration
}

func NewCacheMiddleware(cache Cache, keyGenerator func(ctx contracts.RequestContext) string, ttl time.Duration) *CacheMiddleware {
    return &CacheMiddleware{
        cache:       cache,
        keyGenerator: keyGenerator,
        ttl:         ttl,
    }
}

func (m *CacheMiddleware) Process(next Handler) Handler {
    return func(ctx contracts.RequestContext) (interface{}, ierrors.Error) {
        key := m.keyGenerator(ctx)

        // Try to get from cache
        if cached, exists := m.cache.Get(key); exists {
            return cached, nil
        }

        // Call next handler
        result, err := next(ctx)
        if err != nil {
            return nil, err
        }

        // Cache result
        m.cache.Set(key, result, m.ttl)

        return result, nil
    }
}
```

### Metrics Collection

```go
type MetricsMiddleware struct {
    metrics MetricsCollector
}

func NewMetricsMiddleware(metrics MetricsCollector) *MetricsMiddleware {
    return &MetricsMiddleware{
        metrics: metrics,
    }
}

func (m *MetricsMiddleware) Process(next Handler) Handler {
    return func(ctx contracts.RequestContext) (interface{}, ierrors.Error) {
        start := time.Now()

        // Collect request metrics
        m.metrics.Increment("requests_total", map[string]string{
            "method": ctx.Method(),
            "path":   ctx.Path(),
        })

        // Call next handler
        result, err := next(ctx)

        // Collect response metrics
        duration := time.Since(start)
        status := "success"
        if err != nil {
            status = "error"
        }

        m.metrics.Histogram("request_duration_seconds", duration.Seconds(), map[string]string{
            "method": ctx.Method(),
            "path":   ctx.Path(),
            "status": status,
        })

        m.metrics.Increment("responses_total", map[string]string{
            "method": ctx.Method(),
            "path":   ctx.Path(),
            "status": status,
        })

        return result, err
    }
}
```

### Request Tracing

```go
type TracingMiddleware struct {
    tracer Tracer
}

func NewTracingMiddleware(tracer Tracer) *TracingMiddleware {
    return &TracingMiddleware{
        tracer: tracer,
    }
}

func (m *TracingMiddleware) Process(next Handler) Handler {
    return func(ctx contracts.RequestContext) (interface{}, ierrors.Error) {
        // Start span
        span := m.tracer.StartSpan(ctx.Path())
        defer span.Finish()

        // Add context to span
        span.SetTag("http.method", ctx.Method())
        span.SetTag("http.url", ctx.Path())
        span.SetTag("component", "api")

        // Add span to context
        ctx = ctx.WithContext(m.tracer.ContextWithSpan(ctx.Context(), span))

        // Call next handler
        result, err := next(ctx)

        // Add result to span
        if err != nil {
            span.SetTag("error", true)
            span.SetTag("error.message", err.Error())
        }

        return result, err
    }
}
```

## Security Middleware

### CSRF Protection

```go
type CSRFMiddleware struct {
    tokenGenerator TokenGenerator
    tokenValidator TokenValidator
    excludedPaths  []string
}

func NewCSRFMiddleware(generator TokenGenerator, validator TokenValidator, excludedPaths []string) *CSRFMiddleware {
    return &CSRFMiddleware{
        tokenGenerator: generator,
        tokenValidator: validator,
        excludedPaths:  excludedPaths,
    }
}

func (m *CSRFMiddleware) Process(next Handler) Handler {
    return func(ctx contracts.RequestContext) (interface{}, ierrors.Error) {
        // Skip excluded paths
        for _, path := range m.excludedPaths {
            if ctx.Path() == path {
                return next(ctx)
            }
        }

        // For GET requests, generate CSRF token
        if ctx.Method() == "GET" {
            token := m.tokenGenerator.Generate()
            ctx.Set("csrf_token", token)
            ctx.Header("X-CSRF-Token", token)
            return next(ctx)
        }

        // For state-changing requests, validate CSRF token
        token := ctx.Header("X-CSRF-Token")
        if token == "" {
            token = ctx.FormValue("csrf_token")
        }

        if token == "" {
            return nil, errors.BadRequest("CSRF token required")
        }

        if !m.tokenValidator.Validate(token) {
            return nil, errors.BadRequest("Invalid CSRF token")
        }

        return next(ctx)
    }
}
```

### Security Headers

```go
type SecurityHeadersMiddleware struct {
    headers map[string]string
}

func NewSecurityHeadersMiddleware() *SecurityHeadersMiddleware {
    return &SecurityHeadersMiddleware{
        headers: map[string]string{
            "X-Content-Type-Options":   "nosniff",
            "X-Frame-Options":          "DENY",
            "X-XSS-Protection":         "1; mode=block",
            "Strict-Transport-Security": "max-age=31536000; includeSubDomains",
            "Content-Security-Policy":   "default-src 'self'",
            "Referrer-Policy":          "strict-origin-when-cross-origin",
        },
    }
}

func (m *SecurityHeadersMiddleware) Process(next Handler) Handler {
    return func(ctx contracts.RequestContext) (interface{}, ierrors.Error) {
        // Set security headers
        for key, value := range m.headers {
            ctx.Header(key, value)
        }

        return next(ctx)
    }
}
```

### Request Size Limiting

```go
type SizeLimitMiddleware struct {
    maxSize int64
}

func NewSizeLimitMiddleware(maxSize int64) *SizeLimitMiddleware {
    return &SizeLimitMiddleware{
        maxSize: maxSize,
    }
}

func (m *SizeLimitMiddleware) Process(next Handler) Handler {
    return func(ctx contracts.RequestContext) (interface{}, ierrors.Error) {
        // Check Content-Length header
        if contentLength := ctx.Header("Content-Length"); contentLength != "" {
            if size, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
                if size > m.maxSize {
                    return nil, errors.RequestEntityTooLarge("request entity too large")
                }
            }
        }

        // Wrap response writer to track actual size
        wrappedWriter := &sizeLimitWriter{
            ResponseWriter: ctx.ResponseWriter(),
            maxSize:       m.maxSize,
        }

        // Update context with wrapped writer
        ctx = ctx.WithResponseWriter(wrappedWriter)

        return next(ctx)
    }
}

type sizeLimitWriter struct {
    http.ResponseWriter
    maxSize    int64
    bytesWritten int64
}

func (w *sizeLimitWriter) Write(b []byte) (int, error) {
    if w.bytesWritten+int64(len(b)) > w.maxSize {
        return 0, errors.RequestEntityTooLarge("request entity too large")
    }

    n, err := w.ResponseWriter.Write(b)
    w.bytesWritten += int64(n)
    return n, err
}
```

## Performance Optimization

### Response Compression

```go
type CompressionMiddleware struct {
    level int
    types []string
}

func NewCompressionMiddleware(level int, contentTypes []string) *CompressionMiddleware {
    return &CompressionMiddleware{
        level: level,
        types: contentTypes,
    }
}

func (m *CompressionMiddleware) Process(next Handler) Handler {
    return func(ctx contracts.RequestContext) (interface{}, ierrors.Error) {
        // Check if client accepts compression
        acceptEncoding := ctx.Header("Accept-Encoding")
        if !strings.Contains(acceptEncoding, "gzip") {
            return next(ctx)
        }

        // Call next handler
        result, err := next(ctx)
        if err != nil {
            return nil, err
        }

        // Compress response if applicable
        if m.shouldCompress(ctx) {
            compressed, err := m.compress(result)
            if err == nil {
                ctx.Header("Content-Encoding", "gzip")
                ctx.Header("Content-Length", strconv.Itoa(len(compressed)))
                return compressed, nil
            }
        }

        return result, nil
    }
}

func (m *CompressionMiddleware) shouldCompress(ctx contracts.RequestContext) bool {
    contentType := ctx.Header("Content-Type")
    for _, t := range m.types {
        if strings.Contains(contentType, t) {
            return true
        }
    }
    return false
}

func (m *CompressionMiddleware) compress(data interface{}) ([]byte, error) {
    var buf bytes.Buffer
    writer, err := gzip.NewWriterLevel(&buf, m.level)
    if err != nil {
        return nil, err
    }

    defer writer.Close()

    switch v := data.(type) {
    case []byte:
        writer.Write(v)
    case string:
        writer.Write([]byte(v))
    default:
        jsonData, err := json.Marshal(v)
        if err != nil {
            return nil, err
        }
        writer.Write(jsonData)
    }

    return buf.Bytes(), nil
}
```

### Connection Pooling

```go
type ConnectionPoolMiddleware struct {
    pool *ConnectionPool
}

func NewConnectionPoolMiddleware(pool *ConnectionPool) *ConnectionPoolMiddleware {
    return &ConnectionPoolMiddleware{
        pool: pool,
    }
}

func (m *ConnectionPoolMiddleware) Process(next Handler) Handler {
    return func(ctx contracts.RequestContext) (interface{}, ierrors.Error) {
        // Get connection from pool
        conn, err := m.pool.Get()
        if err != nil {
            return nil, errors.ServiceUnavailable("database connection unavailable")
        }
        defer m.pool.Put(conn)

        // Add connection to context
        ctx = ctx.WithContext(context.WithValue(ctx.Context(), "db_conn", conn))

        return next(ctx)
    }
}
```

## Error Handling Middleware

### Global Error Handling

```go
type ErrorHandlingMiddleware struct {
    errorHandlers map[error.Type]ErrorHandler
    defaultHandler ErrorHandler
}

type ErrorHandler func(err ierrors.Error) (interface{}, int)

func NewErrorHandlingMiddleware() *ErrorHandlingMiddleware {
    return &ErrorHandlingMiddleware{
        errorHandlers: map[error.Type]ErrorHandler{
            error.TypeValidation:   handleValidationError,
            error.TypeNotFound:     handleNotFoundError,
            error.TypeUnauthorized: handleUnauthorizedError,
            error.TypeForbidden:    handleForbiddenError,
        },
        defaultHandler: handleDefaultError,
    }
}

func (m *ErrorHandlingMiddleware) Process(next Handler) Handler {
    return func(ctx contracts.RequestContext) (interface{}, ierrors.Error) {
        result, err := next(ctx)

        if err != nil {
            // Log error
            log.Printf("Error in %s %s: %v", ctx.Method(), ctx.Path(), err)

            // Handle error
            handler, exists := m.errorHandlers[err.Type()]
            if !exists {
                handler = m.defaultHandler
            }

            response, statusCode := handler(err)
            ctx.Status(statusCode)
            return response, nil
        }

        return result, nil
    }
}

func handleValidationError(err ierrors.Error) (interface{}, int) {
    return map[string]interface{}{
        "success": false,
        "error": map[string]interface{}{
            "type":    "validation_error",
            "message": err.Message(),
            "details": err.Details(),
        },
    }, http.StatusBadRequest
}

func handleNotFoundError(err ierrors.Error) (interface{}, int) {
    return map[string]interface{}{
        "success": false,
        "error": map[string]interface{}{
            "type":    "not_found",
            "message": err.Message(),
        },
    }, http.StatusNotFound
}
```

### Request Timeout Handling

```go
type TimeoutMiddleware struct {
    timeout time.Duration
}

func NewTimeoutMiddleware(timeout time.Duration) *TimeoutMiddleware {
    return &TimeoutMiddleware{
        timeout: timeout,
    }
}

func (m *TimeoutMiddleware) Process(next Handler) Handler {
    return func(ctx contracts.RequestContext) (interface{}, ierrors.Error) {
        // Create context with timeout
        timeoutCtx, cancel := context.WithTimeout(ctx.Context(), m.timeout)
        defer cancel()

        // Update context
        ctx = ctx.WithContext(timeoutCtx)

        // Create channel for result
        resultChan := make(chan interface{}, 1)
        errorChan := make(chan ierrors.Error, 1)

        // Execute handler in goroutine
        go func() {
            result, err := next(ctx)
            if err != nil {
                errorChan <- err
            } else {
                resultChan <- result
            }
        }()

        // Wait for result or timeout
        select {
        case result := <-resultChan:
            return result, nil
        case err := <-errorChan:
            return nil, err
        case <-timeoutCtx.Done():
            return nil, errors.RequestTimeout("request timeout")
        }
    }
}
```

These middleware patterns provide a comprehensive toolkit for building secure, performant, and maintainable APIs in GoStart.