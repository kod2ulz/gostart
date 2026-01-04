# Progressive Route Configuration with Options

This guide shows how to use the new progressive route configuration system that supports adding middleware, annotations, and handlers incrementally.

## Quick Example

Here's how to add comprehensive Swagger documentation to a search endpoint:

```go
func (s *categoryService) AdminAPI(middleware ...api.MiddlewareFunc) func(api.Router) {
	return func(router api.Router) {
		// Apply middleware to all routes
		router.Use(middleware...)

		// 1. Start with a simple route
		// router.GET("", api.TypedListHandler(s.SearchCategories))

		// 2. Add OpenAPI documentation progressively
		searchParams := query.GenerateOpenAPIParameters(db.CategorySearchFields)
		searchDescription := `Search and filter attribute categories with powerful query parameters.

` + query.GenerateSearchDescription(db.CategorySearchFields)

		router.GET("", api.TypedListHandler(s.SearchCategories),
			api.WithAnnotation(openapi.Annotation{
				Summary:     "Search categories",
				Description: searchDescription,
				Parameters:  searchParams,
			}),
		)

		// 3. Add route guards (middleware) to specific routes
		router.POST("", api.TypedHandler(s.CreateCategory),
			api.WithMiddleware(adminOnlyMiddleware),
		)

		// 4. Combine multiple options
		router.PUT(":id", api.TypedHandler(s.UpdateCategory),
			api.WithAnnotation(openapi.Annotation{
				Summary: "Update a category",
			}),
			api.WithMiddleware(adminOnlyMiddleware, auditLoggingMiddleware),
		)

		// 5. Chain additional handlers before the main handler
		router.DELETE(":id", api.TypedHandler(s.DeleteCategory),
			api.WithHandler(validatePathParam),
			api.WithMiddleware(adminOnlyMiddleware),
			api.WithAnnotation(openapi.Annotation{
				Summary: "Delete a category",
			}),
		)
	}
}
```

## Route Options

The router supports three types of optional arguments that can be combined in any order:

### 1. `WithMiddleware(middleware ...MiddlewareFunc) RouteOption`

Adds route-specific middleware (router guards). These are executed before the main handler.

```go
router.GET("/admin", handler,
	api.WithMiddleware(authMiddleware, adminOnlyMiddleware),
)
```

### 2. `WithAnnotation(annotation openapi.Annotation) RouteOption`

Adds custom OpenAPI/Swagger documentation to the route.

```go
router.GET("/search", searchHandler,
	api.WithAnnotation(openapi.Annotation{
		Summary:     "Search items",
		Description: "Full-text search with filters",
		Parameters:  searchParams,
		Tags:        []string{"search", "items"},
	}),
)
```

### 3. `WithHandler(handler HandlerFunc) RouteOption`

Adds an additional handler to the chain (executed before the main handler).

```go
router.POST("/users", createUserHandler,
	api.WithHandler(validateRequest),
	api.WithHandler(rateLimitCheck),
)
```

## Auto-Generated Documentation

The router automatically generates documentation from:

1. **Handler Name**: The function name is extracted and used as the default summary
   - `SearchCategories` → "Search Categories"
   - `createUserHandler` → "Create User Handler"

2. **Typed Handlers**: Request and response types are automatically documented
   - Query parameters from `ListRequest` fields
   - JSON body from request struct tags
   - Response schemas from return types

3. **Path Tags**: Automatic tag generation from route path
   - `/admin/categories` → `["admin", "categories"]`

## Progressive Development Pattern

Start simple, add complexity as needed:

```go
func (s *service) API(router api.Router) {
	// Week 1: Simple routes
	router.GET("", listHandler)
	router.POST("", createHandler)

	// Week 2: Add documentation
	router.GET("", listHandler,
		api.WithAnnotation(openapi.Annotation{
			Summary: "List all items",
		}),
	)

	// Week 3: Add middleware guards
	router.POST("", createHandler,
		api.WithMiddleware(authMiddleware),
		api.WithAnnotation(openapi.Annotation{
			Summary: "Create a new item",
		}),
	)

	// Week 4: Add validation handler chain
	router.POST("", createHandler,
		api.WithHandler(validateRequest),
		api.WithMiddleware(authMiddleware),
		api.WithAnnotation(openapi.Annotation{
			Summary:     "Create a new item",
			Description: "Creates an item after validation",
		}),
	)
}
```

## Real-World Examples

### Search Endpoint with Full Documentation

```go
func (s *categoryService) SearchCategories(ctx apictx.RequestContext, param contracts.ListRequest[string]) (out []contracts.Category, count *int64, er ierrors.Error) {
	var err error
	var total int64
	var res []db.AttributeCategory

	if total, res, err = db.SearchCategories(s.db, ctx.Context(), contracts.QueryParamSearchProvider(ctx)); err != nil {
		return nil, nil, errors.DatabaseFailure[db.AttributeCategory](err)
	}
	return collections.MapList(res, contracts.ToCategory), &total, nil
}

func (s *categoryService) AdminAPI(middleware ...api.MiddlewareFunc) func(api.Router) {
	return func(router api.Router) {
		router.Use(middleware...)

		// Richly documented search endpoint
		searchParams := query.GenerateOpenAPIParameters(db.CategorySearchFields)
		searchDescription := query.GenerateSearchDescription(db.CategorySearchFields)

		router.GET("", api.TypedListHandler(s.SearchCategories),
			api.WithAnnotation(openapi.Annotation{
				Summary:     "Search categories",
				Description: searchDescription,
				Parameters:  searchParams,
			}),
		)

		// Other simple routes
		router.POST("", api.TypedHandler(s.CreateCategory))
		router.PUT(":id", api.TypedHandler(s.UpdateCategory))
	}
}
```

### Protected Admin Endpoint

```go
// Only admins can access, with audit logging
router.DELETE("/users/:id", deleteUserHandler,
	api.WithMiddleware(authMiddleware, adminOnlyMiddleware, auditMiddleware),
	api.WithAnnotation(openapi.Annotation{
		Summary:        "Delete user",
		Description:    "Permanently delete a user account (admin only)",
		Deprecated:     false,
	}),
)
```

### Rate-Limited Public Endpoint

```go
// Public but rate-limited
router.GET("/popular", popularItemsHandler,
	api.WithMiddleware(rateLimitMiddleware),
	api.WithAnnotation(openapi.Annotation{
		Summary:        "Get popular items",
		Description:    "Returns the most popular items (rate limited: 100 req/min)",
	}),
)
```

### Chained Validation

```go
// Validate → Transform → Create
router.POST("/complex", createComplexResourceHandler,
	api.WithHandler(validateSchemaHandler),
	api.WithHandler(transformRequestHandler),
	api.WithMiddleware(authMiddleware),
	api.WithAnnotation(openapi.Annotation{
		Summary: "Create complex resource",
	}),
)
```

## Annotation Structure

The `openapi.Annotation` struct supports:

```go
type Annotation struct {
	Summary     string                        // Auto-generated from handler name if empty
	Description string                        // Full markdown description
	Tags        []string                      // Auto-generated from path if empty
	Parameters  []ParameterAnnotation         // Query, path, header parameters
	RequestBody *RequestBodyAnnotation        // Request body schema
	Responses   map[string]ResponseAnnotation // Response schemas by status code
	Deprecated  bool                          // Mark as deprecated
}
```

## Result

When you visit Swagger UI (typically at `/swagger`), you'll see:

1. **Auto-generated summaries** from handler names
2. **All custom parameters** from field definitions
3. **Rich descriptions** for search capabilities
4. **Deprecation warnings** where applicable
5. **Tags grouping** related endpoints
6. **Request/response schemas** from typed handlers

The documentation automatically stays in sync with your code, and you can enhance it progressively as your API evolves!
