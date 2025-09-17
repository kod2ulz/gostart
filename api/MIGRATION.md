# API Package Migration Guide

This guide helps you migrate from the original API package to the enhanced version with improved error handling, centralized error management, and OpenAPI documentation support.

## Key Changes

### 1. **Centralized Error Handling**
The original package had scattered error handling with custom error types. The new version centralizes error management through the `errors` subpackage with standardized error responses.

### 2. **OpenAPI Documentation**
Added comprehensive OpenAPI/Swagger documentation support with automated schema generation and validation.

### 3. **Improved Request Handling**
Enhanced request model handling with better validation, serialization, and deserialization support.

### 4. **Framework Integration**
Better integration with web frameworks through standardized adapters and middleware.

## Migration Steps

### Step 1: Update Imports

Most imports remain the same, but you may need to add new imports:

```go
// Old imports
import "github.com/kod2ulz/gostart/api"

// New imports (additional)
import "github.com/kod2ulz/gostart/errors"
import "github.com/kod2ulz/gostart/api/openapi"
```

### Step 2: Update Error Handling

#### Old API
```go
func handleRequest(c *gin.Context) {
    // Manual error handling
    if err != nil {
        c.JSON(400, gin.H{
            "error": err.Error(),
            "success": false,
        })
        return
    }

    // Success response
    c.JSON(200, gin.H{
        "data": result,
        "success": true,
    })
}
```

#### New API
```go
func handleRequest(c *gin.Context) {
    // Centralized error handling
    if err != nil {
        errors.RespondWithError(c, err)
        return
    }

    // Standardized success response
    api.RespondSuccess(c, result)
}
```

### Step 3: Update Request Handler Patterns

#### Old API (Using RequestHandlerWithResponse Pattern)
```go
type RoleService struct {
    // service dependencies
}

func (u *RoleService) AdminAPI(router *gin.RouterGroup, middleware ...gin.HandlerFunc) {
    router.Use(middleware...).
        GET("", api.RequestHandlerWithListResponse(u.SearchRoles)).
        POST("", api.RequestHandlerWithResponse(u.CreateRole)).
        PUT(":id", api.RequestHandlerWithResponse(u.UpdateRole)).
        GET("/:id/users", api.RequestHandlerWithListResponse(u.ListRoleUsers)).
        GET("/:id/permissions", api.RequestHandlerWithListResponse(u.ListRolePermissions)).
        PUT("/:id/permissions", api.RequestHandlerWithResponse(u.SetRolePermissions)).
        DELETE(":id", api.RequestHandlerWithResponse(u.DeleteRole))
}

func (u *RoleService) SearchRoles(ctx context.Context) ([]Role, errors.Error) {
    // Business logic for searching roles
    return []Role{}, nil
}

func (u *RoleService) CreateRole(ctx context.Context) (Role, errors.Error) {
    // Business logic for creating role
    return Role{}, nil
}

func (u *RoleService) UpdateRole(ctx context.Context) (Role, errors.Error) {
    // Business logic for updating role
    return Role{}, nil
}
```

#### New API (Framework-Agnostic Handler Functions with Auto-Generated OpenAPI)
```go
type RoleService struct {
    // service dependencies (database, auth, etc.)
    db  *pgx.Conn
    auth auth.Service
}

func NewRoleService(db *pgx.Conn, auth auth.Service) *RoleService {
    return &RoleService{db: db, auth: auth}
}

// Service methods use contracts.RequestContext and contracts.RequestParam
// These methods can be used by HTTP handlers, MQ workers, and other services

func (s *RoleService) SearchRoles(ctx contracts.RequestContext, param contracts.RequestParam) ([]Role, ierrors.Error) {
    // Business logic for searching roles
    // Access request parameters through param contracts.RequestParam
    // Access request context (headers, auth info) through ctx contracts.RequestContext
    return []Role{}, nil
}

func (s *RoleService) CreateRole(ctx contracts.RequestContext, param contracts.RequestParam) (Role, ierrors.Error) {
    // Extract the actual request type from contracts.RequestParam
    req, ok := param.(*CreateRoleRequest)
    if !ok {
        return Role{}, errors.BadRequest("invalid request type")
    }

    // Business logic for creating role
    return Role{}, nil
}

func (s *RoleService) UpdateRole(ctx contracts.RequestContext, param contracts.RequestParam) (Role, ierrors.Error) {
    // Extract ID from path and request from param
    id := ctx.Param("id")
    req, ok := param.(*UpdateRoleRequest)
    if !ok {
        return Role{}, errors.BadRequest("invalid request type")
    }

    // Business logic for updating role
    return Role{}, nil
}
```

#### Route Registration with New API

```go
func (s *RoleService) RegisterRoutes(router *gin.Engine) {
    api := router.Group("/api/v1/roles")

    // Middleware for authentication/authorization
    api.Use(auth.Middleware())

    // Use Handler wrapper for automatic request binding and response formatting
    api.GET("", api.Handler(s.SearchRoles))
    api.POST("", api.Handler(s.CreateRole))
    api.GET("/:id", api.Handler(s.GetRole))
    api.PUT("/:id", api.Handler(s.UpdateRole))
    api.DELETE(":id", api.Handler(s.DeleteRole))
    api.GET("/:id/users", api.ListHandler(s.ListRoleUsers))
    api.GET("/:id/permissions", api.ListHandler(s.ListRolePermissions))
    api.PUT("/:id/permissions", api.Handler(s.SetRolePermissions))
}
```

#### Usage in MQ Workers (Same Service Method)

```go
func (s *RoleService) SetupMQWorkers(mqManager *mq.UnifiedHandler) error {
    // Use the same service method for MQ message processing
    worker, err := mqManager.CreateWorker("roles", "role.created", s.CreateRole)
    if err != nil {
        return err
    }

    return worker.Start()
}
```

The service methods are now reusable across HTTP, MQ, and any other transport layer.

### Step 4: Update Response Handling

#### Old API
```go
func getUserHandler(c *gin.Context) {
    user, err := userService.Get(id)
    if err != nil {
        c.JSON(404, gin.H{
            "error": "user not found",
            "success": false,
        })
        return
    }

    c.JSON(200, gin.H{
        "data": user,
        "success": true,
    })
}
```

#### New API
```go
func getUserHandler(c *gin.Context) {
    user, err := userService.Get(id)
    if err != nil {
        if errors.IsNotFound(err) {
            errors.RespondWithError(c, errors.NotFound("user not found"))
        } else {
            errors.RespondWithError(c, err)
        }
        return
    }

    api.RespondSuccess(c, user)
}
```

### Step 5: OpenAPI Documentation (Auto-Generated)

#### Old API
```go
// No built-in documentation support
func setupRouter() *gin.Engine {
    r := gin.Default()

    r.POST("/users", handleUserRequest)
    r.GET("/users/:id", getUserHandler)

    return r
}
```

#### New API
```go
func setupRouter() *gin.Engine {
    r := gin.Default()

    // OpenAPI documentation is automatically generated from:
    // - Route handlers and their parameters
    // - Request/response struct definitions
    // - Validation tags and field types
    // - HTTP methods and status codes
    // - Struct field documentation and examples

    // Enable OpenAPI middleware for auto-documentation
    r.Use(openapi.AutoDocumentation())

    // Interactive documentation UI (automatically available)
    // GET /docs/swagger - Interactive Swagger UI
    // GET /docs/openapi.json - OpenAPI specification
    // GET /docs/redoc - ReDoc documentation

    // API routes (automatically documented)
    api := r.Group("/api/v1")
    {
        api.POST("/users", handleUserRequest)
        api.GET("/users/:id", getUserHandler)
    }

    return r
}
```

**Key Points:**
- **Zero Configuration**: OpenAPI documentation is generated automatically from your code
- **Schema Inference**: Request/response types are inferred from Go structs
- **Validation Integration**: Validation tags become OpenAPI validation rules
- **Interactive UI**: Swagger UI and ReDoc are automatically available
- **No Manual Annotations**: Unlike traditional approaches, you don't need to add manual OpenAPI annotations
- **Context Usage**: Business logic continues to use `context.Context` (via `c.Request.Context()`) for proper resource management, while the framework handles HTTP-specific concerns

## Practical Examples

### Example 1: CRUD API Migration

#### Before (Old API - Manual Everything)
```go
type UserController struct {
    userService *UserService
}

func (ctrl *UserController) Create(c *gin.Context) {
    var user User
    if err := c.ShouldBindJSON(&user); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    if user.Name == "" {
        c.JSON(400, gin.H{"error": "name is required"})
        return
    }

    if user.Email == "" {
        c.JSON(400, gin.H{"error": "email is required"})
        return
    }

    created, err := ctrl.userService.Create(user)
    if err != nil {
        c.JSON(500, gin.H{"error": "failed to create user"})
        return
    }

    c.JSON(201, gin.H{
        "data": created,
        "success": true,
    })
}

func (ctrl *UserController) Get(c *gin.Context) {
    id := c.Param("id")

    user, err := ctrl.userService.Get(id)
    if err != nil {
        c.JSON(404, gin.H{"error": "user not found"})
        return
    }

    c.JSON(200, gin.H{
        "data": user,
        "success": true,
    })
}
```

#### After (New API - Structured and Documented)
```go
type UserController struct {
    userService *UserService
}

type CreateUserRequest struct {
    Name  string `json:"name" validate:"required" example:"John Doe"`
    Email string `json:"email" validate:"required,email" example:"john@example.com"`
    Age   int    `json:"age" validate:"gte=18" example:"25"`
}

type UserResponse struct {
    ID        string    `json:"id" example:"123"`
    Name      string    `json:"name" example:"John Doe"`
    Email     string    `json:"email" example:"john@example.com"`
    Age       int       `json:"age" example:"25"`
    CreatedAt time.Time `json:"created_at" example:"2023-01-01T00:00:00Z"`
    UpdatedAt time.Time `json:"updated_at" example:"2023-01-01T00:00:00Z"`
}

// @Summary Create a new user
// @Description Create a new user account with validation
// @Tags users
// @Accept json
// @Produce json
// @Param request body CreateUserRequest true "User creation data"
// @Success 201 {object} api.Response[UserResponse]
// @Failure 400 {object} api.ErrorResponse
// @Failure 500 {object} api.ErrorResponse
// @Router /users [post]
func (ctrl *UserController) Create(c *gin.Context) {
    var req CreateUserRequest
    if err := api.BindAndValidate(c, &req); err != nil {
        errors.RespondWithError(c, err)
        return
    }

    user, err := ctrl.userService.Create(req)
    if err != nil {
        errors.RespondWithError(c, errors.Internal("failed to create user", err))
        return
    }

    response := UserResponse{
        ID:        user.ID,
        Name:      user.Name,
        Email:     user.Email,
        Age:       user.Age,
        CreatedAt: user.CreatedAt,
        UpdatedAt: user.UpdatedAt,
    }

    api.RespondSuccess(c, response)
}

// @Summary Get user by ID
// @Description Get user details by their unique identifier
// @Tags users
// @Produce json
// @Param id path string true "User ID" example("123")
// @Success 200 {object} api.Response[UserResponse]
// @Failure 404 {object} api.ErrorResponse
// @Router /users/{id} [get]
func (ctrl *UserController) Get(c *gin.Context) {
    id := c.Param("id")

    user, err := ctrl.userService.Get(id)
    if err != nil {
        errors.RespondWithError(c, errors.NotFound("user not found"))
        return
    }

    response := UserResponse{
        ID:        user.ID,
        Name:      user.Name,
        Email:     user.Email,
        Age:       user.Age,
        CreatedAt: user.CreatedAt,
        UpdatedAt: user.UpdatedAt,
    }

    api.RespondSuccess(c, response)
}
```

### Example 2: Authentication Integration

#### Before (Old API - Manual Auth)
```go
func authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.JSON(401, gin.H{"error": "missing authorization token"})
            c.Abort()
            return
        }

        user, err := validateToken(token)
        if err != nil {
            c.JSON(401, gin.H{"error": "invalid token"})
            c.Abort()
            return
        }

        c.Set("user", user)
        c.Next()
    }
}

func protectedHandler(c *gin.Context) {
    user := c.MustGet("user").(User)
    c.JSON(200, gin.H{
        "data": user,
        "success": true,
    })
}
```

#### After (New API - Structured Auth)
```go
func authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            errors.RespondWithError(c, errors.Unauthorized("missing authorization token"))
            c.Abort()
            return
        }

        user, err := authService.ValidateToken(token)
        if err != nil {
            errors.RespondWithError(c, errors.Unauthorized("invalid token"))
            c.Abort()
            return
        }

        c.Set("user", user)
        c.Next()
    }
}

// @Summary Get current user profile
// @Description Get the profile of the currently authenticated user
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} api.Response[UserResponse]
// @Failure 401 {object} api.ErrorResponse
// @Router /users/me [get]
func protectedHandler(c *gin.Context) {
    user := c.MustGet("user").(User)

    response := UserResponse{
        ID:        user.ID,
        Name:      user.Name,
        Email:     user.Email,
        Age:       user.Age,
        CreatedAt: user.CreatedAt,
        UpdatedAt: user.UpdatedAt,
    }

    api.RespondSuccess(c, response)
}
```

### Example 3: File Upload Handling

#### Before (Old API - Manual Upload)
```go
func uploadHandler(c *gin.Context) {
    file, err := c.FormFile("file")
    if err != nil {
        c.JSON(400, gin.H{"error": "file is required"})
        return
    }

    if file.Size > 10*1024*1024 { // 10MB
        c.JSON(400, gin.H{"error": "file too large"})
        return
    }

    if !strings.HasSuffix(file.Filename, ".jpg") && !strings.HasSuffix(file.Filename, ".png") {
        c.JSON(400, gin.H{"error": "invalid file type"})
        return
    }

    // Save file...

    c.JSON(200, gin.H{
        "message": "file uploaded successfully",
        "success": true,
    })
}
```

#### After (New API - Validated Upload)
```go
type FileUploadRequest struct {
    File *multipart.FileHeader `form:"file" validate:"required"`
}

// @Summary Upload file
// @Description Upload a file with validation
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to upload (max 10MB, JPG/PNG only)"
// @Success 200 {object} api.Response[FileUploadResponse]
// @Failure 400 {object} api.ErrorResponse
// @Router /files/upload [post]
func uploadHandler(c *gin.Context) {
    var req FileUploadRequest
    if err := c.ShouldBind(&req); err != nil {
        errors.RespondWithError(c, err)
        return
    }

    // Custom validation
    if req.File.Size > 10*1024*1024 {
        errors.RespondWithError(c, errors.Validation("file size exceeds 10MB limit"))
        return
    }

    allowedTypes := map[string]bool{
        ".jpg":  true,
        ".jpeg": true,
        ".png":  true,
    }

    ext := strings.ToLower(filepath.Ext(req.File.Filename))
    if !allowedTypes[ext] {
        errors.RespondWithError(c, errors.Validation("invalid file type. Only JPG and PNG are allowed"))
        return
    }

    // Save file...
    filename, err := saveFile(req.File)
    if err != nil {
        errors.RespondWithError(c, errors.Internal("failed to save file", err))
        return
    }

    response := FileUploadResponse{
        Filename: filename,
        Size:     req.File.Size,
        URL:      fmt.Sprintf("/files/%s", filename),
    }

    api.RespondSuccess(c, response)
}
```

## Migration Benefits Summary

### 1. **Error Handling Improvements**
- **Before**: Manual error handling with inconsistent response formats
- **After**: Centralized error management with standardized responses

### 2. **Documentation Quality**
- **Before**: No built-in API documentation
- **After**: Comprehensive OpenAPI/Swagger documentation

### 3. **Validation**
- **Before**: Manual validation scattered throughout handlers
- **After**: Structured validation with clear error messages

### 4. **Code Organization**
- **Before**: Inconsistent patterns across handlers
- **After**: Standardized patterns with better separation of concerns

### 5. **Developer Experience**
- **Before**: Manual testing and debugging
- **After**: Interactive API documentation and better tooling support

## Testing Your Migration

### 1. **Unit Testing**
```go
func TestCreateUser(t *testing.T) {
    // Setup test router
    router := setupTestRouter()

    // Test successful creation
    userReq := CreateUserRequest{
        Name:  "John Doe",
        Email: "john@example.com",
        Age:   25,
    }

    jsonValue, _ := json.Marshal(userReq)
    req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(jsonValue))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusCreated, w.Code)

    var response api.Response[UserResponse]
    json.Unmarshal(w.Body.Bytes(), &response)
    assert.True(t, response.Success)
    assert.Equal(t, "John Doe", response.Data.Name)
}
```

### 2. **Integration Testing**
```go
func TestUserAPIDocumentation(t *testing.T) {
    router := setupRouter()

    // Test that OpenAPI documentation is accessible
    req, _ := http.NewRequest("GET", "/docs/swagger.json", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
    assert.Contains(t, w.Body.String(), "openapi")
}
```

## Rollback Plan

If you encounter issues during migration:

1. **Revert Error Handling**: Temporarily comment out centralized error handling and restore manual responses
2. **Disable OpenAPI**: Remove OpenAPI middleware if it causes issues
3. **Gradual Migration**: Migrate one endpoint at a time rather than all at once

## Common Migration Issues

### 1. **Import Errors**
- Ensure all new imports are correctly added
- Check for conflicts with existing error handling

### 2. **Validation Errors**
- Update struct tags to use the new validation format
- Test validation with various input scenarios

### 3. **Documentation Issues**
- Ensure all routes have proper OpenAPI annotations
- Test that the Swagger UI is accessible

## Conclusion

The enhanced API package provides significant improvements in error handling, documentation, and overall code quality. While the migration requires some changes to existing code, the benefits include:

- Consistent error responses across all endpoints
- Interactive API documentation for better developer experience
- Better validation and request handling
- Improved maintainability and testability

The migration examples above demonstrate how the new API eliminates common pain points while making the code more professional and maintainable.