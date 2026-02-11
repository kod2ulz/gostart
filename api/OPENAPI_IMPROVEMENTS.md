# OpenAPI Documentation Improvements

This document describes the improvements made to OpenAPI/Swagger documentation generation in gostart.

## Issues Fixed

### 1. Tag Generation Based on Route Groups
**Before**: Tags were generated from every segment in the path (e.g., `["api", "admin", "attribute", "categories"]`)

**After**: Tags respect the route group hierarchy and skip common prefixes (e.g., `["admin", "attribute/categories"]`)

**Result**: Swagger UI groups endpoints more logically by functional area.

### 2. Type Schema Generation for Typed Handlers
**Before**: Request and response schemas were not automatically extracted from typed handlers.

**After**: Typed handlers can now provide explicit type information for accurate schema generation.

**Result**: Request bodies and response types are now properly documented in Swagger UI.

### 3. Annotation Parameters
**Before**: Custom annotation parameters (from `SearchableRouteDocs`, etc.) were being ignored.

**After**: Annotation parameters are now properly included in the OpenAPI specification.

**Result**: Search parameters and other custom metadata appear correctly in Swagger UI.

## New Features

### TypedHandlerWithTypes and TypedListHandlerWithTypes

New helper functions that capture type information for automatic schema generation:

```go
// Instead of:
router.GET("/users", api.TypedListHandler(ListUsers), api.WithAnnotation(...))

// Use:
handler, typeOpt := api.TypedListHandlerWithTypes(ListUsers)
router.GET("/users", handler, api.WithAnnotation(...), typeOpt)
```

### Automatic Request Body Schema

For POST/PUT/PATCH endpoints, request body schemas are now automatically generated from the request struct:

```go
type UpdateUserRequest struct {
    ID       string `json:"id" param:"id"`
    Name     string `json:"name" validate:"required"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"gte=0"`
}

func UpdateUser(ctx contracts.RequestContext, req UpdateUserRequest) (User, ierrors.Error) {
    // Handler logic
}

// Usage:
handler, typeOpt := api.TypedHandlerWithTypes(UpdateUser)
router.PUT("/users/:id", handler, typeOpt)
```

This automatically generates:
- Request body schema with all fields
- Field types and validation rules
- Required field indicators

### Automatic Query Parameter Generation

For GET endpoints with list handlers, query parameters are automatically extracted:

```go
type ListUsersRequest struct {
    api.ListRequest
    Search string `json:"search" query:"search"`
    Active bool   `json:"active" query:"active"`
}

func ListUsers(ctx contracts.RequestContext, req ListUsersRequest) ([]User, *int64, ierrors.Error) {
    // Handler logic
}

// Usage:
handler, typeOpt := api.TypedListHandlerWithTypes(ListUsers)
router.GET("/users", handler, typeOpt)
```

This automatically generates:
- Query parameters for fields with `query` tag
- Parameter types from struct field types
- Required flags from `validate:"required"` tag

### Enhanced Tag Generation

Tags are now generated based on route group structure:

- `/api/admin/geo/attributes` → `["admin", "geo/attributes"]`
- `/api/admin/attribute/categories` → `["admin", "attribute/categories"]`
- `/api/users` → `["users"]`

## Migration Guide

### For Existing Code

Your existing code will continue to work without changes. However, to get better documentation:

#### Step 1: Update Handler Registration

Change from:
```go
router.GET("/search", api.TypedListHandler(s.SearchCategories),
    api.WithAnnotation(contracts.SearchableRouteDocs("Search Categories", db.CategorySearchFields, "admin", "categories")))
```

To:
```go
handler, typeOpt := api.TypedListHandlerWithTypes(s.SearchCategories)
router.GET("/search", handler,
    api.WithAnnotation(contracts.SearchableRouteDocs("Search Categories", db.CategorySearchFields, "admin", "categories")),
    typeOpt)
```

#### Step 2: Verify Swagger UI

1. Start your application
2. Navigate to `/swagger` or `/docs`
3. Verify that:
   - Endpoints are grouped by tags
   - Request bodies show proper schemas for POST/PUT
   - Query parameters appear for GET endpoints
   - Search parameters from `SearchableRouteDocs` are included

## Advanced Usage

### Custom Type Information

You can also manually provide type information using `WithTypes`:

```go
router.POST("/users", CreateUserHandler,
    api.WithTypes(reflect.TypeOf(CreateUserRequest{}), reflect.TypeOf(User{}), false))
```

### Combining with Existing Annotations

Type information works seamlessly with existing annotations:

```go
handler, typeOpt := api.TypedHandlerWithTypes(UpdateUser)
router.PUT("/users/:id", handler,
    api.WithAnnotation(openapi.Annotation{
        Summary:     "Update a user",
        Description: "Updates an existing user account",
        Tags:        []string{"users", "admin"},
    }),
    typeOpt)
```

## Technical Details

### Type Capture

The `TypedHandlerWithTypes` and `TypedListHandlerWithTypes` functions:
1. Create zero-value instances of the generic type parameters
2. Use `reflect.TypeOf()` to capture type information
3. Return both the handler function and a `RouteOption` with type metadata

### Schema Generation

The router's `enhanceAnnotationFromTypes` method:
1. For GET requests: Extracts query and path parameters from struct tags
2. For POST/PUT/PATCH: Generates request body schema from struct fields
3. Respects `json`, `query`, `param`, and `validate` tags
4. Properly handles required fields and validation rules

### Tag Hierarchy

The `generateTagsFromPath` method:
1. Splits the path into segments
2. Removes common prefixes (like "api")
3. Creates hierarchical tags for nested resources
4. Preserves the logical grouping from route definitions

## Benefits

1. **Better Developer Experience**: Swagger UI now shows accurate request/response schemas
2. **Reduced Manual Documentation**: Most documentation is auto-generated from code
3. **Type Safety**: Schemas stay in sync with your Go types
4. **Better Organization**: Logical grouping of endpoints in Swagger UI
5. **Backward Compatible**: Existing code continues to work without changes

## Limitations

1. Type information requires explicit use of `TypedHandlerWithTypes` or `TypedListHandlerWithTypes`
2. Complex nested types may need additional manual annotation
3. Response type generation is currently limited (uses default response structure)
4. Array and slice types show as generic arrays without item type details

## Future Enhancements

- Automatic response type extraction from handler return types
- Support for generic types in schema generation
- Better handling of nested and embedded structs
- Enum value extraction from constants
- Example value generation from `example` tags
