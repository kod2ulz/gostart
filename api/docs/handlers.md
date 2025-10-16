# API Handlers and Request Processing

This guide covers the advanced patterns and techniques for building robust API handlers in GoStart.

## Handler Architecture

### Framework-Agnostic Handler Interface

The API package provides a unified handler interface that works with any web framework:

```go
// Basic handler signature
func CreateUserHandler(ctx contracts.RequestContext) (CreateUserResponse, ierrors.Error) {
    var req CreateUserRequest

    // Load and validate request
    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return CreateUserResponse{}, err
    }

    // Business logic
    user, err := userService.Create(req)
    if err != nil {
        return CreateUserResponse{}, err
    }

    // Return response
    return CreateUserResponse{
        ID:      user.ID,
        Message: "User created successfully",
    }, nil
}
```

### Handler Registration

```go
// Register handler with framework
router.POST("/users", CreateUserHandler)

// With middleware
router.POST("/users",
    middleware.RateLimit(100),
    middleware.Authenticate(),
    CreateUserHandler,
)
```

## Request Processing Patterns

### Automatic Request Loading

The API package automatically loads request data from multiple sources:

```go
type CreateOrderRequest struct {
    CustomerID string `json:"customer_id" validate:"required"`
    Items      []OrderItem `json:"items" validate:"required,min=1"`
    Priority   string `json:"priority" query:"priority" validate:"oneof=normal high urgent"`
    Discount   float64 `json:"discount" header:"X-Discount"`
}

func CreateOrderHandler(ctx contracts.RequestContext) (CreateOrderResponse, ierrors.Error) {
    var req CreateOrderRequest

    // Automatically loads from:
    // - JSON body (customer_id, items)
    // - Query params (priority)
    // - Headers (discount)
    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return CreateOrderResponse{}, err
    }

    // Process order
    order, err := orderService.Create(req)
    if err != nil {
        return CreateOrderResponse{}, err
    }

    return CreateOrderResponse{
        OrderID:    order.ID,
        Total:      order.Total,
        CreatedAt:  order.CreatedAt,
    }, nil
}
```

### Custom Request Loading

```go
type CustomRequest struct {
    Data       interface{}
    Metadata   RequestMetadata
    References map[string]interface{}
}

func (r *CustomRequest) RequestLoad(ctx contracts.RequestContext) (contracts.RequestParam, error) {
    // Custom loading logic
    var data struct {
        Name string `json:"name"`
        Age  int    `json:"age"`
    }

    if err := ctx.ShouldBindJSON(&data); err != nil {
        return nil, err
    }

    r.Data = data
    r.Metadata = RequestMetadata{
        IPAddress: ctx.Header("X-Forwarded-For"),
        UserAgent: ctx.Header("User-Agent"),
    }

    return r, nil
}

func (r *CustomRequest) Validate(ctx contracts.RequestContext) error {
    data := r.Data.(struct {
        Name string `json:"name"`
        Age  int    `json:"age"`
    })

    if data.Name == "" {
        return errors.New("name is required")
    }

    if data.Age < 0 || data.Age > 150 {
        return errors.New("age must be between 0 and 150")
    }

    return nil
}
```

## Response Handling Patterns

### Standardized Responses

```go
type SuccessResponse[T any] struct {
    Success bool   `json:"success"`
    Data    T      `json:"data"`
    Message string `json:"message,omitempty"`
}

type ErrorResponse struct {
    Success bool                   `json:"success"`
    Error   contracts.ErrorDetails `json:"error"`
}

func GetUserHandler(ctx contracts.RequestContext) (SuccessResponse[User], ierrors.Error) {
    var req GetUserRequest

    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return SuccessResponse[User]{}, err
    }

    user, err := userService.GetByID(req.UserID)
    if err != nil {
        return SuccessResponse[User]{}, err
    }

    return SuccessResponse[User]{
        Success: true,
        Data:    user,
        Message: "User retrieved successfully",
    }, nil
}
```

### Response Envelopes

```go
type ResponseEnvelope[T any] struct {
    Success    bool                   `json:"success"`
    Data       T                      `json:"data,omitempty"`
    Error      *contracts.ErrorDetails `json:"error,omitempty"`
    Pagination *PaginationInfo        `json:"pagination,omitempty"`
    Meta       map[string]interface{} `json:"meta,omitempty"`
}

type PaginationInfo struct {
    Page       int `json:"page"`
    PageSize   int `json:"page_size"`
    TotalItems int `json:"total_items"`
    TotalPages int `json:"total_pages"`
}

func GetUsersHandler(ctx contracts.RequestContext) (ResponseEnvelope[[]User], ierrors.Error) {
    var req GetUsersRequest

    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return ResponseEnvelope[[]User]{}, err
    }

    users, total, err := userService.GetUsers(req)
    if err != nil {
        return ResponseEnvelope[[]User]{}, err
    }

    totalPages := (total + req.PageSize - 1) / req.PageSize

    return ResponseEnvelope[[]User]{
        Success: true,
        Data:    users,
        Pagination: &PaginationInfo{
            Page:       req.Page,
            PageSize:   req.PageSize,
            TotalItems: total,
            TotalPages: totalPages,
        },
        Meta: map[string]interface{}{
            "timestamp": time.Now(),
            "version":   "1.0",
        },
    }, nil
}
```

## Error Handling Patterns

### Structured Error Responses

```go
type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Code    string `json:"code"`
}

type BusinessError struct {
    Type    string `json:"type"`
    Message string `json:"message"`
    Code    string `json:"code"`
    Details map[string]interface{} `json:"details,omitempty"`
}

func ValidateUserHandler(ctx contracts.RequestContext) (SuccessResponse[User], ierrors.Error) {
    var req CreateUserRequest

    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return SuccessResponse[User]{}, errors.BadRequest(err)
    }

    // Business validation
    if userExists, _ := userService.EmailExists(req.Email); userExists {
        return SuccessResponse[User]{},
            errors.Conflict("user with this email already exists").
            WithDetails(map[string]interface{}{
                "field":   "email",
                "email":   req.Email,
                "suggestions": []string{
                    "Try logging in instead",
                    "Use password reset if you forgot your password",
                },
            })
    }

    // Age validation
    if req.Age < 18 {
        return SuccessResponse[User]{},
            errors.Forbidden("users must be at least 18 years old").
            WithCode("AGE_RESTRICTION")
    }

    user, err := userService.Create(req)
    if err != nil {
        return SuccessResponse[User]{}, err
    }

    return SuccessResponse[User]{
        Success: true,
        Data:    user,
    }, nil
}
```

### Error Aggregation

```go
type BulkOperationRequest struct {
    Operations []Operation `json:"operations"`
}

type BulkOperationResponse struct {
    SuccessCount int           `json:"success_count"`
    ErrorCount   int           `json:"error_count"`
    Results      []OperationResult `json:"results"`
    Errors       []OperationError `json:"errors"`
}

type OperationResult struct {
    ID      string      `json:"id"`
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
}

type OperationError struct {
    ID      string                 `json:"id"`
    Error   contracts.ErrorDetails `json:"error"`
}

func BulkCreateHandler(ctx contracts.RequestContext) (BulkOperationResponse, ierrors.Error) {
    var req BulkOperationRequest

    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return BulkOperationResponse{}, err
    }

    var wg sync.WaitGroup
    results := make([]OperationResult, len(req.Operations))
    errors := make([]OperationError, 0)
    errorsChan := make(chan OperationError, len(req.Operations))

    for i, operation := range req.Operations {
        wg.Add(1)
        go func(idx int, op Operation) {
            defer wg.Done()

            result, err := processOperation(op)
            if err != nil {
                errorsChan <- OperationError{
                    ID:    op.ID,
                    Error: err.ToErrorDetails(),
                }
                return
            }

            results[idx] = OperationResult{
                ID:      op.ID,
                Success: true,
                Data:    result,
            }
        }(i, operation)
    }

    wg.Wait()
    close(errorsChan)

    for err := range errorsChan {
        errors = append(errors, err)
    }

    return BulkOperationResponse{
        SuccessCount: len(results),
        ErrorCount:   len(errors),
        Results:      results,
        Errors:       errors,
    }, nil
}
```

## Advanced Handler Patterns

### Streaming Responses

```go
type StreamHandler struct {
    dataSource DataSource
}

func (h *StreamHandler) Handle(ctx contracts.RequestContext) {
    // Set up streaming response
    ctx.Header("Content-Type", "text/event-stream")
    ctx.Header("Cache-Control", "no-cache")
    ctx.Header("Connection", "keep-alive")

    flusher, ok := ctx.ResponseWriter().(http.Flusher)
    if !ok {
        ctx.JSON(http.StatusInternalServerError, ErrorResponse{
            Error: "Streaming not supported",
        })
        return
    }

    // Stream data
    dataChan := make(chan DataItem, 100)
    errorChan := make(chan error, 1)

    go h.dataSource.Stream(dataChan, errorChan)

    for {
        select {
        case item := <-dataChan:
            json.NewEncoder(ctx).Encode(map[string]interface{}{
                "type": "data",
                "data": item,
            })
            flusher.Flush()

        case err := <-errorChan:
            json.NewEncoder(ctx).Encode(map[string]interface{}{
                "type": "error",
                "error": err.Error(),
            })
            return

        case <-ctx.Request().Context().Done():
            // Client disconnected
            return
        }
    }
}
```

### File Upload/Download

```go
type FileUploadRequest struct {
    File       multipart.File `json:"-" validate:"required"`
    Filename   string         `json:"filename"`
    ContentType string         `json:"content_type"`
    Size       int64          `json:"size"`
}

func (r *FileUploadRequest) RequestLoad(ctx contracts.RequestContext) (contracts.RequestParam, error) {
    file, header, err := ctx.Request().FormFile("file")
    if err != nil {
        return nil, err
    }

    r.File = file
    r.Filename = header.Filename
    r.ContentType = header.Header.Get("Content-Type")
    r.Size = header.Size

    return r, nil
}

func (r *FileUploadRequest) Validate(ctx contracts.RequestContext) error {
    if r.Size > 100*1024*1024 { // 100MB limit
        return errors.New("file size exceeds limit")
    }

    allowedTypes := map[string]bool{
        "image/jpeg": true,
        "image/png":  true,
        "application/pdf": true,
    }

    if !allowedTypes[r.ContentType] {
        return errors.New("file type not allowed")
    }

    return nil
}

func UploadFileHandler(ctx contracts.RequestContext) (SuccessResponse[FileInfo], ierrors.Error) {
    var req FileUploadRequest

    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return SuccessResponse[FileInfo]{}, err
    }

    defer req.File.Close()

    // Process file upload
    fileInfo, err := fileService.Upload(req.File, req.Filename, req.ContentType)
    if err != nil {
        return SuccessResponse[FileInfo]{}, err
    }

    return SuccessResponse[FileInfo]{
        Success: true,
        Data:    fileInfo,
    }, nil
}
```

### WebSocket Handlers

```go
type WebSocketHandler struct {
    connections *collections.ConcurrentMap[string, *WebSocketConnection]
    hub         *MessageHub
}

type WebSocketConnection struct {
    Conn     *websocket.Conn
    UserID   string
    Send     chan []byte
    Close    chan struct{}
}

func (h *WebSocketHandler) HandleWebSocket(ctx contracts.RequestContext) {
    conn, err := upgrader.Upgrade(ctx.ResponseWriter(), ctx.Request(), nil)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, ErrorResponse{
            Error: "Failed to upgrade connection",
        })
        return
    }
    defer conn.Close()

    // Get user ID from context (set by auth middleware)
    userID := ctx.Value("user_id").(string)

    // Create connection
    wsConn := &WebSocketConnection{
        Conn:  conn,
        UserID: userID,
        Send:   make(chan []byte, 256),
        Close:  make(chan struct{}),
    }

    // Register connection
    h.connections.Set(userID, wsConn)

    // Start message pump
    go h.readPump(wsConn)
    go h.writePump(wsConn)

    // Wait for connection to close
    <-wsConn.Close
    h.connections.Delete(userID)
}

func (h *WebSocketHandler) readPump(conn *WebSocketConnection) {
    defer close(conn.Close)

    for {
        _, message, err := conn.Conn.ReadMessage()
        if err != nil {
            return
        }

        // Handle incoming message
        h.hub.HandleMessage(conn.UserID, message)
    }
}

func (h *WebSocketHandler) writePump(conn *WebSocketConnection) {
    ticker := time.NewTicker(30 * time.Second)
    defer func() {
        ticker.Stop()
        conn.Conn.Close()
    }()

    for {
        select {
        case message, ok := <-conn.Send:
            if !ok {
                conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            conn.Conn.WriteMessage(websocket.TextMessage, message)

        case <-ticker.C:
            conn.Conn.WriteMessage(websocket.PingMessage, []byte{})

        case <-conn.Close:
            return
        }
    }
}
```

## Middleware Integration

### Authentication Middleware

```go
func AuthMiddleware(validator TokenValidator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := r.Header.Get("Authorization")
            if token == "" {
                http.Error(w, "Authorization required", http.StatusUnauthorized)
                return
            }

            user, err := validator.Validate(token)
            if err != nil {
                http.Error(w, "Invalid token", http.StatusUnauthorized)
                return
            }

            // Add user to context
            ctx := context.WithValue(r.Context(), "user", user)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### Rate Limiting Middleware

```go
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            clientID := getClientID(r)

            if !limiter.Allow(clientID, 100, time.Minute) {
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

### Logging Middleware

```go
func LoggingMiddleware(logger *logr.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()

            // Wrap response writer to capture status code
            wrapped := wrapResponseWriter(w)

            defer func() {
                duration := time.Since(start)
                logger.Info("Request completed",
                    "method", r.Method,
                    "path", r.URL.Path,
                    "status", wrapped.status,
                    "duration_ms", duration.Milliseconds(),
                    "user_agent", r.UserAgent(),
                )
            }()

            next.ServeHTTP(wrapped, r)
        })
    }
}

type responseWriter struct {
    http.ResponseWriter
    status int
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
    return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.status = code
    rw.ResponseWriter.WriteHeader(code)
}
```

## Testing Patterns

### Handler Testing

```go
func TestCreateUserHandler(t *testing.T) {
    // Setup mock service
    mockUserService := &MockUserService{
        CreateFunc: func(req CreateUserRequest) (User, error) {
            return User{
                ID:    "user123",
                Name:  req.Name,
                Email: req.Email,
            }, nil
        },
    }

    // Create handler
    handler := CreateUserHandler
    handler.UserService = mockUserService

    // Test cases
    tests := []struct {
        name           string
        request        CreateUserRequest
        expectedStatus int
        expectedError  string
    }{
        {
            name: "valid request",
            request: CreateUserRequest{
                Name:  "John Doe",
                Email: "john@example.com",
                Age:   25,
            },
            expectedStatus: http.StatusCreated,
        },
        {
            name: "invalid email",
            request: CreateUserRequest{
                Name:  "John Doe",
                Email: "invalid-email",
                Age:   25,
            },
            expectedStatus: http.StatusBadRequest,
            expectedError:  "invalid email format",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create test context
            ctx := createTestContext(tt.request)

            // Call handler
            response, err := handler(ctx)

            // Assert results
            if tt.expectedError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedError)
            } else {
                assert.NoError(t, err)
                assert.NotEmpty(t, response.ID)
                assert.Equal(t, tt.request.Name, response.Name)
            }
        })
    }
}

func createTestContext(request interface{}) contracts.RequestContext {
    // Serialize request to JSON
    requestBody, _ := json.Marshal(request)

    // Create HTTP request
    req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(requestBody))
    req.Header.Set("Content-Type", "application/json")

    // Create test context
    ctx := &gin.Context{
        Request: req,
        Writer:  httptest.NewRecorder(),
    }

    return &api.Context{Context: ctx}
}
```

### Integration Testing

```go
func TestUserAPIIntegration(t *testing.T) {
    // Setup test server
    router := setupTestRouter()

    // Test user creation flow
    t.Run("create and get user", func(t *testing.T) {
        // Create user
        createUserReq := map[string]interface{}{
            "name":  "Jane Doe",
            "email": "jane@example.com",
            "age":   30,
        }

        reqBody, _ := json.Marshal(createUserReq)
        req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(reqBody))
        req.Header.Set("Content-Type", "application/json")

        w := httptest.NewRecorder()
        router.ServeHTTP(w, req)

        assert.Equal(t, http.StatusCreated, w.Code)

        var createUserResp map[string]interface{}
        json.Unmarshal(w.Body.Bytes(), &createUserResp)
        userID := createUserResp["data"].(map[string]interface{})["id"].(string)

        // Get user
        getReq := httptest.NewRequest("GET", "/users/"+userID, nil)
        w = httptest.NewRecorder()
        router.ServeHTTP(w, getReq)

        assert.Equal(t, http.StatusOK, w.Code)

        var getUserResp map[string]interface{}
        json.Unmarshal(w.Body.Bytes(), &getUserResp)
        assert.Equal(t, userID, getUserResp["data"].(map[string]interface{})["id"])
        assert.Equal(t, "Jane Doe", getUserResp["data"].(map[string]interface{})["name"])
    })
}
```

These patterns provide a comprehensive foundation for building robust, maintainable API handlers in GoStart applications.