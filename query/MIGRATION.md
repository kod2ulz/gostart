# Query Package Migration Guide

This guide helps you migrate from the original query package (chore/202506-upgrades branch) to the enhanced version with improved field definitions, better type safety, and more flexible query capabilities.

## Key Changes

### 1. **New Fluent Field Definitions System**
The original package used simple `UrlFields` with string slices. The new version introduces a comprehensive fluent `FieldDefinition` system with type safety, validation, and better defaults.

### 2. **Enhanced Type System**
Split the generic "numeric" type into specific `Int` and `Float` types for better pgx compatibility and type safety.

### 3. **Improved JSON Schema Support**
Added support for nested JSON field schemas with proper validation and query capabilities.

### 4. **Configuration Management**
Moved from `utils.Env` to dedicated `config` package for better configuration management.

### 5. **Enhanced Query Builder Features**
Added new methods like `NoLimit()` and improved the overall query building experience.

## Migration Steps

### Step 1: Update Field Definitions (Major Change)

#### Old API (UrlFields)
```go
var CountryFields = query.UrlFields{
    Comparable: []string{"name", "continent_id", "parent_id", "capital", "a2", "a3", "timezone", "created_at", "updated_at"},
    Lookup:     []string{"name", "type", "capital", "continent_id", "parent_id", "a2", "a3", "timezone"},
    Sort:       []string{"name", "updated_at"},
}
```

#### New API (Fluent FieldDefinitions)
```go
var CountryFields = query.NewDefinitions(
    query.Text("name").Sortable().WithOperators(
        query.CompareEqual, query.CompareLike, query.CompareIn,
    ),
    query.Int("continent_id").Sortable(),
    query.Int("parent_id"),
    query.Text("capital").WithOperators(query.CompareEqual, query.CompareLike),
    query.Text("a2").Sortable(),
    query.Text("a3").Sortable(),
    query.Text("timezone"),
    query.Time("created_at", "2006-01-02T15:04:05Z").Sortable(),
    query.Time("updated_at", "2006-01-02T15:04:05Z").Sortable(),
)
```

This is a **significant improvement** as it provides:
- **Type Safety**: Each field has a specific type instead of just strings
- **Validation**: Automatic validation based on field types
- **Default Operators**: Sensible defaults for each field type
- **Fluent API**: Easy chaining of configuration options
- **Better Documentation**: Self-documenting field definitions

### Step 2: Update Numeric Field Types

#### Old API
```go
// Single numeric type
defs["age"] = query.FieldDef{
    Name: "age",
    Type: "numeric",
    Operators: []string{"eq", "gt", "lt"},
}
```

#### New API
```go
// Separate Int and Float types
defs := query.NewDefinitions(
    query.Int("age"),           // For integer values
    query.Float("score"),       // For floating-point values
)
```

### Step 3: Update JSON Field Definitions

#### Old API
```go
// Basic JSON field support
defs["metadata"] = query.FieldDef{
    Name: "metadata",
    Type: "json",
}
```

#### New API
```go
// Enhanced JSON field with schema support
defs := query.NewDefinitions(
    query.JSON("metadata").WithSchema(
        query.Text("city"),
        query.Int("age"),
        query.Float("rating"),
        query.Bool("active"),
    ),
)
```

### Step 4: Update Query Builder Usage

#### Old API
```go
builder := query.SQLBuilder[User](db, scanUser)
builder.Limit(10).Offset(0).Where(conditions...)
```

#### New API
```go
builder := query.SQLBuilder[User](db, scanUser)
builder.Limit(10).Offset(0).Where(conditions...)

// New NoLimit() method for unlimited queries
builder.NoLimit()  // Disables limit regardless of configuration
```

### Step 5: Update Condition Building

#### Old API
```go
conditions := []query.Condition{
    query.Equal("name", "John"),
    query.GreaterThan("age", 25),
}
```

#### New API
```go
// Conditions remain the same, but now work with proper field definitions
conditions := []query.Condition{
    query.Equal("name", "John"),
    query.GreaterThan("age", 25),
    query.Between("score", 80.5, 95.0),  // New between support
}
```

### Step 6: Update URL Search Integration

#### Old API
```go
// Basic URL parameter handling
params := query.ParseURLParams(r.URL.Query(), defs)
builder.FromUrlParams(params)
```

#### New API
```go
// Enhanced URL search with validation
urlSearch := query.SearchURL(
    func(ctx context.Context, name string, _default ...string) contracts.Value {
        // Custom parameter provider
        return contracts.Value(r.URL.Query().Get(name))
    },
    defs,
)

urlParams := urlSearch.Load(ctx)
builder.FromUrlParams(urlParams)
```

## Practical Examples

### Example 1: Country Search API Migration

#### Before (Old API - chore/202506-upgrades)
```go
var CountryFields = query.UrlFields{
    Comparable: []string{"name", "continent_id", "parent_id", "capital", "a2", "a3", "timezone", "created_at", "updated_at"},
    Lookup:     []string{"name", "type", "capital", "continent_id", "parent_id", "a2", "a3", "timezone"},
    Sort:       []string{"name", "updated_at"},
}

func SearchCountries(db sqlc.DBTX, params map[string]string) ([]Country, error) {
    // Manual parameter validation and parsing
    builder := query.SQLBuilder[Country](db, scanCountry)

    // Basic conditions
    var conditions []query.Condition
    for key, value := range params {
        if contains(CountryFields.Comparable, key) {
            conditions = append(conditions, query.Equal(key, value))
        }
    }

    builder.Where(conditions...)

    // Basic sorting
    if sort, ok := params["sort"]; ok && contains(CountryFields.Sort, sort) {
        builder.Order(query.Sort(sort, query.SortAsc))
    }

    return builder.Query()
}
```

#### After (New API - Current)
```go
var CountryFields = query.NewDefinitions(
    query.Text("name").Sortable().WithOperators(
        query.CompareEqual, query.CompareLike, query.CompareIn,
    ),
    query.Int("continent_id").Sortable(),
    query.Int("parent_id"),
    query.Text("capital").WithOperators(query.CompareEqual, query.CompareLike),
    query.Text("a2").Sortable(),
    query.Text("a3").Sortable(),
    query.Text("timezone"),
    query.Time("created_at", "2006-01-02T15:04:05Z").Sortable(),
    query.Time("updated_at", "2006-01-02T15:04:05Z").Sortable(),
)

func SearchCountries(db sqlc.DBTX, r *http.Request) ([]Country, error) {
    // Enhanced URL search with automatic validation
    urlSearch := query.SearchURL(
        func(ctx context.Context, name string, _default ...string) contracts.Value {
            return contracts.Value(r.URL.Query().Get(name))
        },
        CountryFields,
    )

    builder := query.SQLBuilder[Country](db, scanCountry)
    urlParams := urlSearch.Load(r.Context())
    builder.FromUrlParams(urlParams)

    return builder.Query()
}
```

**Key Improvements:**
- **Type Safety**: Each field has a specific type (Text, Int, Time) instead of just strings
- **Automatic Validation**: Invalid field names and operators are automatically rejected
- **Default Operators**: Sensible defaults for each field type (e.g., Int supports gt, lt, between)
- **Better Sorting**: Automatic sorting validation based on field definitions
- **Fluent API**: Clean, readable field definitions with method chaining

### Example 2: Product Catalog with Advanced Filtering

#### Before (Old API)
```go
func SearchProducts(db sqlc.DBTX, filters map[string]string) ([]Product, error) {
    // Limited field support
    defs := map[string]query.FieldDef{
        "name":        {Name: "name", Type: "text"},
        "price":       {Name: "price", Type: "numeric"},
        "category":    {Name: "category", Type: "text"},
    }

    conditions := make([]query.Condition, 0)
    for key, value := range filters {
        conditions = append(conditions, query.Equal(key, value))
    }

    builder := query.SQLBuilder[Product](db, scanProduct)
    builder.Where(conditions...)

    return builder.Query()
}
```

#### After (New API)
```go
func SearchProducts(db sqlc.DBTX, filters map[string]string) ([]Product, error) {
    // Comprehensive product field definitions
    defs := query.NewDefinitions(
        query.Text("name").Sortable().WithOperators(
            query.CompareEqual, query.CompareLike, query.CompareIn,
        ),
        query.Float("price").WithOperators(
            query.CompareEqual, query.CompareGreaterThan,
            query.CompareLessThan, query.CompareBetween,
        ),
        query.Text("category").WithOperators(query.CompareEqual, query.CompareIn),
        query.Bool("in_stock"),
        query.Date("created_at", "2006-01-02T15:04:05Z"),
        query.JSON("attributes").WithSchema(
            query.Text("color"),
            query.Text("size"),
            query.Float("weight"),
        ),
    )

    // Build conditions from filters with proper validation
    builder := query.SQLBuilder[Product](db, scanProduct)
    conditions := make([]query.Condition, 0)

    for key, value := range filters {
        // Validate that field exists in definitions
        if fieldDef, exists := defs[key]; exists {
            // Convert string value to appropriate type
            var convertedValue any
            switch fieldDef.Type {
            case query.TypeInt:
                if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
                    convertedValue = intValue
                }
            case query.TypeFloat:
                if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
                    convertedValue = floatValue
                }
            case query.TypeBool:
                convertedValue = strings.ToLower(value) == "true"
            default:
                convertedValue = value
            }

            if convertedValue != nil {
                conditions = append(conditions, query.Equal(key, convertedValue))
            }
        }
    }

    builder.Where(conditions...)

    // Allow unlimited results for export operations
    if unlimited, ok := filters["unlimited"]; ok && unlimited == "true" {
        builder.NoLimit()
    }

    return builder.Query()
}
```

### Example 3: Analytics Dashboard with Complex Queries

#### Before (Old API)
```go
func GetAnalyticsData(db sqlc.DBTX, startDate, endDate time.Time) ([]Analytics, error) {
    builder := query.SQLBuilder[Analytics](db, scanAnalytics)

    // Manual condition building
    conditions := []query.Condition{
        query.GreaterThanOrEqual("created_at", startDate),
        query.LessThanOrEqual("created_at", endDate),
    }

    builder.Where(conditions...)
    builder.Group("user_id", "event_type")

    return builder.Query()
}
```

#### After (New API)
```go
func GetAnalyticsData(db sqlc.DBTX, startDate, endDate time.Time) ([]Analytics, error) {
    // Comprehensive analytics field definitions
    defs := query.NewDefinitions(
        query.Text("event_type"),
        query.Text("user_id"),
        query.Time("created_at", "2006-01-02T15:04:05Z"),
        query.JSON("metadata").WithSchema(
            query.Text("action"),
            query.Text("category"),
            query.Float("value"),
        ),
    )

    builder := query.SQLBuilder[Analytics](db, scanAnalytics)

    // Enhanced date filtering with validation
    conditions := []query.Condition{
        query.Between("created_at", startDate, endDate),
        query.In("event_type", "login", "purchase", "view"),
    }

    builder.Where(conditions...)
    builder.Group("user_id", "event_type")
    builder.Order(
        query.Sort("created_at", query.SortDesc),
        query.Sort("user_id", query.SortAsc),
    )

    return builder.Query()
}
```

## Migration Benefits Summary

### 1. **Type Safety Improvements**
- **Before**: Single "numeric" type for all numbers
- **After**: Separate `Int` and `Float` types with proper validation

### 2. **Better JSON Support**
- **Before**: Basic JSON field handling
- **After**: Nested JSON schemas with validation and querying

### 3. **Enhanced Validation**
- **Before**: Manual field validation
- **After**: Automatic validation based on field definitions

### 4. **Improved Query Flexibility**
- **Before**: Basic query building
- **After**: Advanced features like `NoLimit()`, better sorting, and grouping

### 5. **Configuration Management**
- **Before**: Using utils.Env
- **After**: Dedicated config package with better type safety

## Testing Your Migration

### 1. **Unit Testing**
```go
func TestFieldDefinitions(t *testing.T) {
    defs := query.NewDefinitions(
        query.Text("name"),
        query.Int("age"),
        query.Float("price"),
    )

    // Test field existence
    if _, exists := defs["name"]; !exists {
        t.Error("Expected 'name' field to exist")
    }

    // Test field properties
    ageField := defs["age"]
    if ageField.Type != query.TypeInt {
        t.Errorf("Expected age field type to be Int, got %s", ageField.Type)
    }
}
```

### 2. **Integration Testing**
```go
func TestQueryBuilderWithNewDefinitions(t *testing.T) {
    db := setupTestDB()
    defs := query.NewDefinitions(
        query.Text("name"),
        query.Int("age"),
        query.Float("score"),
    )

    builder := query.SQLBuilder[User](db, scanUser)
    conditions := []query.Condition{
        query.Equal("name", "John Doe"),
        query.GreaterThan("age", 25),
    }

    builder.Where(conditions...)
    users, err := builder.Query()

    assert.NoError(t, err)
    assert.NotEmpty(t, users)
}
```

## Common Migration Issues

### 1. **Field Definition Errors**
- Ensure all field names match your database schema
- Check that field types are correctly specified (Int vs Float)
- Verify operator compatibility with field types

### 2. **JSON Schema Issues**
- JSON fields must use `TypeJSON` type
- Nested fields must be defined using `WithSchema()`
- Ensure JSON field paths are correctly formatted

### 3. **Configuration Changes**
- Update imports from `utils.Env` to `config`
- Ensure configuration values are properly typed
- Check environment variable names

### 4. **Query Builder Method Names**
- `sqlBuilder` is now `SqlBuild` (capitalized)
- New methods like `NoLimit()` are available
- Check method signatures for changes

## Rollback Plan

If you encounter issues during migration:

1. **Revert Field Definitions**: Temporarily use the old map-based approach
2. **Disable New Features**: Avoid using new methods like `NoLimit()`
3. **Gradual Migration**: Migrate one query at a time rather than all at once

## Conclusion

The enhanced query package provides significant improvements in type safety, validation, and query capabilities. While the migration requires some changes to existing code, the benefits include:

- Better type safety with separate numeric types
- Enhanced JSON field support with nested schemas
- Improved validation and error handling
- More flexible query building options
- Better configuration management

The migration examples above demonstrate how the new API eliminates common pain points while making the code more maintainable and type-safe.