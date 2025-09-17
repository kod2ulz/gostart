# Query Package

The `query` package provides a fluent, type-safe, and secure API for building dynamic SQL queries from external inputs, such as URL query parameters. It enables developers to create flexible search and filtering capabilities while maintaining security and preventing SQL injection.

## Why We Have This Package

This package addresses several common challenges in web application development:

1. **Security**: Prevents SQL injection through definition-driven whitelisting and parameterized queries
2. **Productivity**: Eliminates manual parsing of URL parameters and building of SQL WHERE clauses
3. **Type Safety**: Ensures external inputs are validated and converted to proper Go types
4. **Consistency**: Provides a unified approach to query building across different transport layers (HTTP, MQ, etc.)
5. **Maintainability**: Centralizes query logic and makes it easy to modify search behavior

## Core Goals

- **Developer Productivity**: Abstract away the tedious and error-prone process of manually parsing URL parameters and building SQL `WHERE` clauses.
- **Type Safety**: Ensure that parameters from external sources are parsed and validated into their correct Go types at the earliest possible moment.
- **Security**: Prevent SQL injection and other vulnerabilities by using a definition-driven whitelisting approach. Only explicitly defined fields and operations are permitted.
- **Decoupling**: Keep the data layer (`query`) completely separate from the transport layer (`api`). This is achieved via the `UrlParameterProvider` function interface, which allows any source to provide query parameters.

## Core Concepts

The entire system is driven by a `FieldDefinitions` map, which serves as the single source of truth for your queryable API.

### 1. Definition-Driven API

You define every field that can be queried, sorted, or filtered. This is done using a fluent builder API that is both easy to read and write.

```go
var TransactionFields = query.NewDefinitions(
    query.Text("serviceType").Sortable(),
    query.Float("serviceAmount").Sortable(),
    query.Date("approvedAt", "2006-01-02").Sortable(),
    query.JSON("paymentDetails").WithSchema(
        query.Text("prn"),
        query.Text("item"),
    ),
)
```

### 2. Fluent Builders

Builders like `Text()`, `Int()`, `Float()`, `Date()`, and `JSON()` create a `FieldDefinition` with sensible defaults.
- The database column name is automatically inferred by converting the field name to `snake_case` (e.g., `serviceAmount` becomes `service_amount`).
- You can chain methods like `.Sortable()`, `.WithDBName()`, `.WithOperators()`, and `.WithSchema()` to customize the behavior.

### 3. Type-Safe Parsing

When a search is executed, the `query` package uses your `FieldDefinitions` to:
1. Find a corresponding URL parameter (e.g., `approvedAt_gt=2022-01-01`).
2. Check if the `gt` (`CompareGreaterThan`) operator is allowed for that field.
3. Parse the value ("2022-01-01") into a `time.Time` object using the defined format.
4. If any step fails, the parameter is safely ignored.

The SQL builder only ever deals with strongly-typed Go values.

## Complete Flow: URL Parameters to SQL Queries

The query library transforms URL parameters into SQL queries through a well-defined pipeline:

### 1. URL Parameter Parsing
```go
// Simulates reading from a URL query map
mockParameterProvider := func(params map[string]string) query.UrlParameterProvider {
    return func(ctx context.Context, name string, _default ...string) config.Value {
        if val, ok := params[name]; ok {
            return config.Value(val)
        }
        if len(_default) > 0 {
            return config.Value(_default[0])
        }
        return ""
    }
}
```

### 2. Field Definitions
```go
userFieldDefinitions := query.NewDefinitions(
    query.Text("username").Sortable().WithOperators(
        query.CompareEqual, query.CompareLike, query.CompareIn,
    ),
    query.Int("age").WithOperators(
        query.CompareEqual, query.CompareGreaterThan,
        query.CompareLessThan, query.CompareBetween,
    ),
    query.Bool("active"),
    query.Date("created_at", time.RFC3339).Sortable(),
    query.JSON("metadata").WithSchema(
        query.Text("city"),
        query.Int("zip_code"),
    ),
)
```

### 3. URL Search Parameter Processing
```go
urlParams := query.SearchURL(provider, userFieldDefinitions).Load(ctx)
```

### 4. SQL Builder Integration
```go
qb := query.SQLBuilder[User](db, rowScanner).FromUrlParams(urlParams)
sqlQuery, args := qb.Criteria()
```

## URL Parameter Patterns

### Field Name Resolution Rules

The query library supports intelligent field name resolution with these rules:

1. **Separator Variations**: `first_name`, `firstName`, and `first-name` are treated as the same field
2. **Word Casing Matters**: `firstName` and `firstname` are different fields
3. **JSONB Fields**: For dotted field names like `person.first_name`, each part gets separator variations applied separately:
   - `person.first_name` matches `person.firstName`, `person.first-name`, etc.
   - Results in SQL: `person ->> 'first_name'`
4. **Database Source of Truth**: Database field names are always lowercase snake_case, variations are generated from there

#### Examples:
- Field definition: `first_name` matches parameters: `first_name`, `firstName`, `first-name`
- Field definition: `firstName` matches parameters: `firstName` only (not `firstname`)
- Field definition: `person.first_name` matches: `person.first_name`, `person.firstName`, `person.first-name`, etc.

### Basic Comparisons
- `username=john` → `username = 'john'`
- `age_gt=25` → `age > 25`
- `age_lt=30` → `age < 30`
- `age_gte=25` → `age >= 25`
- `age_lte=30` → `age <= 30`
- `age_neq=25` → `age != 25`

### Text Operations (Tilde Wildcard Syntax)
- `~username~=john` → `username ILIKE '%john%'` (contains)
- `~username=john` → `username ILIKE '%john'` (starts with)
- `username~=john` → `username ILIKE 'john%'` (ends with)
- `username=john` → `username = 'john'` (exact match)
- `username_in=john,jane,bob` → `username IN ('john', 'jane', 'bob')`

### OR Operations
- **Within-field OR**: `username=john|jane|bob` → `username IN ('john', 'jane', 'bob')`
- **Across-field OR**: `*role=admin&accountId=123` → `(account_id = 123 OR role = 'admin')`
- **Multiple star fields**: `*role=admin&*accountId=123` → `(role = 'admin' OR account_id = 123)`
- **Single star field**: `*role=admin` → (ignored, no conditions added - OR only makes sense with multiple fields)

### Range Operations
- `age_bt=25:35` → `age BETWEEN 25 AND 35`
- `created_at_bt=2023-01-01:2023-12-31` → `created_at BETWEEN '2023-01-01' AND '2023-12-31'`

### JSON Fields
- `metadata.city=New York` → `(metadata ->> 'city') = 'New York'`
- `metadata.zip_code_gt=10000` → `(metadata ->> 'zip_code') > 10000`

### Sorting (Both Formats Supported)
- **Legacy Format**: `sort_username=desc&sort_age=asc` → `ORDER BY username DESC, age ASC`
- **New Format**: `sort=-username,+age` → `ORDER BY username DESC, age ASC`
- `sort=username` → `ORDER BY username ASC`
- `sort=-username` → `ORDER BY username DESC`
- `sort=username,-created_at` → `ORDER BY username ASC, created_at DESC`

### Pagination
- `limit=10` → `LIMIT 10`
- `offset=20` → `OFFSET 20`
- `page=3` → `OFFSET 40` (assuming limit=20)

## Usage Examples

### Basic Usage

#### 1. Define your fields (e.g., in your `db` package)
```go
// db/transactions.go

var TransactionFields = query.NewDefinitions(
    query.UUID("id").WithDBName("tracking_number"), // Override default DB name
    query.Text("serviceType").Sortable(),
    query.Float("serviceAmount").Sortable(),
    query.Date("approvedAt", "2006-01-02").Sortable(),
)
```

#### 2. Create the `UrlParameterProvider` adapter
```go
// In your service/api layer

func queryReader(ctx context.Context, name string, _default ...string) config.Value {
    param, _ := api.ListRequestFrom(ctx)
    return param.Query(ctx, name, _default...)
}
```

#### 3. Use it in your search function
```go
// db/transactions.go

func (q *Queries) SearchTransactions(ctx context.Context, param api.ListRequest) (count int64, out []ServiceTransactionResult, err error) {

    // Initialize the parser with the reader and definitions
    urlParams := query.SearchURL(queryReader, TransactionFields).Load(ctx)

    // Build the query from the parsed params
    queryBuilder := query.SQLBuilder(q.db, transactionResultScanner).
        FromUrlParams(urlParams)

    // Set default sort order if none is provided
    if len(urlParams.GetSorts()) == 0 {
        queryBuilder.Order(query.Desc("approved_at"))
    }

    return queryBuilder.Select(ctx, "analytics.partner_application_overview", "id", "approved_at", "service_type", "application_total")
}
```

#### 4. Make a request
A request to `GET /transactions?serviceType=REGISTRATION&approvedAt_gt=2023-01-01&sort_serviceAmount=desc` will now be safely parsed and converted into the appropriate SQL query.

### Advanced Usage with Parameter Overrides

The library supports adding service-layer constraints and business logic overrides:

```go
func (s *UserService) SearchUsers(ctx context.Context, params map[string]string, currentUser auth.User) ([]User, error) {
    // Define field definitions
    userDefs := query.NewDefinitions(
        query.Text("username").Sortable(),
        query.Int("age"),
        query.Bool("active"),
        query.Text("account_id"),
    )

    // Parse URL parameters
    provider := mockParameterProvider(params)
    urlParams := query.SearchURL(provider, userDefs).Load(ctx)

    // Add service layer security constraint
    if !currentUser.IsAdmin() {
        urlParams = urlParams.AddField("account_id", currentUser.AccountID)
    }

    // Add business logic constraint
    urlParams = urlParams.AddCondition("active", query.CompareEqual, true)

    // Build query
    qb := query.SQLBuilder[User](s.db, s.scanUser).FromUrlParams(urlParams)

    count, users, err := qb.Select(ctx, "users")
    if err != nil {
        return nil, err
    }

    return users, nil
}
```

### Multi-Tenant Security Pattern

```go
func (s *TransactionService) SearchTransactions(ctx context.Context, params map[string]string, currentUser auth.User) ([]Transaction, error) {
    transactionDefs := query.NewDefinitions(
        query.Float("amount").WithOperators(
            query.CompareEqual, query.CompareGreaterThan,
            query.CompareLessThan, query.CompareBetween,
        ),
        query.Text("status").WithOperators(
            query.CompareEqual, query.CompareIn,
        ),
        query.Date("transaction_date", "2006-01-02").Sortable(),
        query.UUID("account_id"),
    )

    provider := mockParameterProvider(params)
    urlParams := query.SearchURL(provider, transactionDefs).Load(ctx)

    // Multi-tenant security: regular users can only see their own account's transactions
    if !currentUser.IsAdmin {
        urlParams = urlParams.AddField("account_id", currentUser.AccountID)
    }

    qb := query.SQLBuilder[Transaction](s.db, s.scanTransaction).FromUrlParams(urlParams)

    count, transactions, err := qb.Select(ctx, "transactions")
    if err != nil {
        return nil, err
    }

    return transactions, nil
}
```

### Complex Query with JSON Fields

```go
func (s *ProductService) SearchProducts(ctx context.Context, params map[string]string) ([]Product, error) {
    productDefs := query.NewDefinitions(
        query.Text("name").Sortable().WithOperators(
            query.CompareEqual, query.CompareLike, query.CompareIn,
        ),
        query.Float("price").WithOperators(
            query.CompareEqual, query.CompareGreaterThan,
            query.CompareLessThan, query.CompareBetween,
        ),
        query.JSON("attributes").WithSchema(
            query.Text("color"),
            query.Text("size"),
            query.Float("weight"),
        ),
        query.Bool("in_stock"),
    )

    provider := mockParameterProvider(params)
    urlParams := query.SearchURL(provider, productDefs).Load(ctx)

    // Add business logic: only show in-stock products for regular users
    if !currentUser.IsAdmin {
        urlParams = urlParams.AddField("in_stock", true)
    }

    qb := query.SQLBuilder[Product](s.db, s.scanProduct).FromUrlParams(urlParams)

    count, products, err := qb.Select(ctx, "products")
    if err != nil {
        return nil, err
    }

    return products, nil
}
```

## Security Considerations

### SQL Injection Prevention
- All values are parameterized, preventing SQL injection
- Field names are validated against field definitions
- Only allowed operators are processed

### Type Safety
- All values are parsed according to their field type definitions
- Invalid values are silently ignored
- Date/time values are parsed using specified formats

### Field Access Control
- Only fields defined in FieldDefinitions are processed
- Each field can specify which operators are allowed
- JSON field access is controlled through schema definitions

## Performance Considerations

### Indexing
Ensure database indexes exist for frequently filtered fields:
- Primary filter fields
- Sort fields
- JSON field paths (if using JSONB)

### Query Complexity
- Complex queries with many conditions may impact performance
- JSON field operations are generally slower than regular fields
- Consider denormalizing frequently accessed JSON fields

### Pagination
- Use appropriate limit values (typically 20-100)
- Consider cursor-based pagination for large datasets
- Avoid very high offset values

## Integration Patterns

### 1. Standard API Handler
```go
func ListUsersHandler(ctx contracts.RequestContext) ([]User, ierrors.Error) {
    var req api.ListRequest
    var modal api.RequestModal[api.ListRequest]
    if loadError := modal.FromContext(ctx.Context(), &req); loadError != nil {
        return nil, errors.RequestLoadError[api.ListRequest](loadError)
    }

    userDefs := query.NewDefinitions(
        query.Text("username").Sortable(),
        query.Int("age"),
        query.Bool("active"),
    )

    urlParams := query.SearchURL(req, userDefs).Load(ctx.Context())
    qb := query.SQLBuilder[User](db, scanUser).FromUrlParams(urlParams)

    count, users, err := qb.Select(ctx.Context(), "users")
    if err != nil {
        return nil, errors.GeneralError[[]User](err)
    }

    return users, nil
}
```

### 2. MQ Handler Integration
```go
func HandleUserSearchRequest(ctx context.Context, msg mq.Message) error {
    var params map[string]string
    if err := json.Unmarshal(msg.Body, &params); err != nil {
        return err
    }

    userDefs := query.NewDefinitions(
        query.Text("username").Sortable(),
        query.Int("age"),
        query.Bool("active"),
    )

    provider := mockParameterProvider(params)
    urlParams := query.SearchURL(provider, userDefs).Load(ctx)

    qb := query.SQLBuilder[User](db, scanUser).FromUrlParams(urlParams)

    count, users, err := qb.Select(ctx, "users")
    if err != nil {
        return err
    }

    // Send results back via MQ
    return mq.Publish(ctx, "user.search.results", users)
}
```

## Error Handling

### Invalid Parameters
Invalid or malformed parameters are silently ignored:
- Non-numeric values for numeric fields
- Invalid date formats
- Unknown field names
- Invalid operator combinations

### Type Conversion
Values are converted according to field type definitions:
- Text fields accept any string
- Integer fields require valid integer values
- Float fields require valid floating-point values
- Boolean fields accept "true", "false", "1", "0"
- Date fields require valid date/time strings

## Development Roadmap

### Completed Features ✅
- [x] Fluent field definition API
- [x] Type-safe parameter parsing
- [x] SQL injection prevention
- [x] JSON field support with schemas
- [x] Parameter override functionality
- [x] Comprehensive test coverage
- [x] Migration documentation
- [x] Tilde wildcard syntax for LIKE operations
- [x] Case-insensitive field name resolution with separator variations
- [x] Dual sort parameter format support (legacy and new)
- [x] OR operator functionality (within-field and across-field)
- [x] Pipe-separated value processing for IN operations

### Planned Features 📋
- [ ] Advanced join support
- [ ] Subquery support
- [ ] Aggregate function support
- [ ] Full-text search integration
- [ ] Database-specific optimizations
- [ ] Query caching
- [ ] Performance monitoring
- [ ] Advanced pagination (cursor-based)
- [ ] Field-level access control
- [ ] Query validation middleware

### Long-term Vision 🔮
- [ ] Multi-database support (MySQL, SQLite)
- [ ] GraphQL integration
- [ ] Real-time query subscriptions
- [ ] Query performance analytics
- [ ] Automatic index recommendations
- [ ] Query optimization suggestions

## Migration from chore/202506-upgrades

For developers migrating from the previous implementation in the `chore/202506-upgrades` branch, please refer to the [Migration Guide](MIGRATION.md) for detailed instructions and examples.

The new implementation provides significant improvements in type safety, validation, and query capabilities while maintaining backward compatibility where possible.

## Contributing

This package is designed to be extensible and maintainable. When contributing:

1. **Security First**: Always consider SQL injection and access control implications
2. **Type Safety**: Leverage Go's type system to catch errors at compile time
3. **Performance**: Consider the impact on query performance and database load
4. **Documentation**: Keep examples and documentation up to date
5. **Testing**: Maintain comprehensive test coverage for all features

## License

This package is part of the gostart project and follows the same license terms.