# Query Package

The `query` package provides a fluent, type-safe, and secure API for building dynamic SQL queries from external inputs, such as URL query parameters.

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
    "approvedAt": query.Date("approvedAt", "2006-01-02").Sortable(),
    "paymentDetails": query.JSON("paymentDetails").WithSchema(
        query.Text("prn"),
        query.Text("item"),
    ),
}
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

## Usage Example

Here is how to wire everything together.

#### 1. Define your fields (e.g., in your `db` package)

```go
// db/transactions.go

var TransactionFields = query.FieldDefinitions{
    "id": query.UUID("id").WithDBName("tracking_number"), // Override default DB name
    "serviceType": query.Text("serviceType").Sortable(),
    "serviceAmount": query.Numeric("serviceAmount").Sortable(),
    "approvedAt": query.Date("approvedAt", "2006-01-02").Sortable(),
}
```

#### 2. Create the `UrlParameterProvider` adapter

This function teaches the `query` package how to read values from your `api.ListRequest`.

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
