# Using OpenAPI Annotations with Routes

This guide shows how to add custom OpenAPI/Swagger documentation to your routes using the new `GETWithAnnotation`, `POSTWithAnnotation`, etc. methods.

## Quick Example

Here's how to add comprehensive Swagger documentation to a search endpoint:

```go
func (s *categoryService) AdminAPI(middleware ...api.MiddlewareFunc) func(api.Router) {
	return func(router api.Router) {
		// Generate OpenAPI parameters from your field definitions
		searchParams := query.GenerateOpenAPIParameters(db.CategorySearchFields)

		// Create a detailed description explaining search capabilities
		searchDescription := `Search and filter attribute categories with powerful query parameters.

` + query.GenerateSearchDescription(db.CategorySearchFields)

		// Apply middleware to all routes
		router.Use(middleware...)

		// Register search route with custom OpenAPI documentation
		router.GETWithAnnotation("", api.TypedListHandler(s.SearchCategories), openapi.Annotation{
			Summary:     "Search categories",
			Description: searchDescription,
			Parameters:  searchParams,
		})

		// Register other routes (without custom annotations)
		router.POST("", api.TypedHandler(s.CreateCategory)).
			PUT(":id", api.TypedHandler(s.UpdateCategory)).
			PUT(":id/disable", api.TypedHandler(s.DisableCategory)).
			PUT(":id/enable", api.TypedHandler(s.EnableCategory))
	}
}
```

## Available Methods

The Router interface now provides these methods for each HTTP verb:

- `GETWithAnnotation(path, handler, annotation)`
- `POSTWithAnnotation(path, handler, annotation)`
- `PUTWithAnnotation(path, handler, annotation)`
- `DELETEWithAnnotation(path, handler, annotation)`
- `PATCHWithAnnotation(path, handler, annotation)`
- `OPTIONSWithAnnotation(path, handler, annotation)`
- `HEADWithAnnotation(path, handler, annotation)`

## Annotation Structure

The `openapi.Annotation` struct supports:

```go
type Annotation struct {
	Summary     string
	Description string
	Tags        []string
	Parameters  []ParameterAnnotation
	RequestBody *RequestBodyAnnotation
	Responses   map[string]ResponseAnnotation
	Deprecated  bool
}
```

## Auto-Generated Defaults

When using `*WithAnnotation` methods:

- **Tags**: Automatically generated from the path (e.g., `admin/categories` → `["admin", "categories"]`)
  - Override by providing your own `Tags` field
- **Summary**: Default is `METHOD path` (e.g., `GET /admin/categories`)
  - Override by providing your own `Summary` field

## Common Use Cases

### 1. Documenting Search Endpoints

```go
searchParams := query.GenerateOpenAPIParameters(db.MySearchFields)
searchDescription := query.GenerateSearchDescription(db.MySearchFields)

router.GETWithAnnotation("", searchHandler, openapi.Annotation{
	Summary:     "Search items",
	Description: searchDescription,
	Parameters:  searchParams,
})
```

### 2. Custom Request Body Documentation

```go
router.POSTWithAnnotation("", createHandler, openapi.Annotation{
	Summary: "Create a new item",
	RequestBody: &openapi.RequestBodyAnnotation{
		Description: "Item to create",
		Required:    true,
		Content: map[string]openapi.MediaType{
			"application/json": {
				Schema: &openapi.Schema{
					Type:       "object",
					Properties: map[string]*openapi.Schema{
						"name": {Type: "string"},
						"value": {Type: "integer"},
					},
				},
			},
		},
	},
})
```

### 3. Custom Response Documentation

```go
router.GETWithAnnotation("", getHandler, openapi.Annotation{
	Summary: "Get item by ID",
	Responses: map[string]openapi.ResponseAnnotation{
		"200": {
			Description: "Item found",
			Content: map[string]openapi.MediaType{
				"application/json": {
					Schema: &openapi.Schema{
						Type: "object",
					},
				},
			},
		},
		"404": {
			Description: "Item not found",
		},
	},
})
```

### 4. Deprecating Routes

```go
router.GETWithAnnotation("", oldHandler, openapi.Annotation{
	Summary:    "Old endpoint (deprecated)",
	Deprecated: true,
})
```

## Mix and Match

You can mix annotated and non-annotated routes:

```go
func (s *service) API(router api.Router) {
	// Custom documentation
	router.GETWithAnnotation("/search", searchHandler, openapi.Annotation{
		Summary: "Search items",
		Parameters: searchParams,
	})

	// Auto-generated documentation
	router.GET("/:id", getHandler)
	router.POST("", createHandler)

	// Custom documentation
	router.DELETEWithAnnotation("/:id", deleteHandler, openapi.Annotation{
		Summary: "Permanently delete an item",
		Responses: responseDocs,
	})
}
```

## Result

When you visit Swagger UI (typically at `/swagger`), you'll see:

1. All your custom parameters documented
2. Rich descriptions for search capabilities
3. Custom request/response schemas
4. Deprecation warnings where applicable
5. Tags grouping related endpoints

The documentation automatically stays in sync with your code!
