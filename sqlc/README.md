# SQLC Package

The `sqlc` package provides database interface definitions and utilities for GoStart applications. It offers a clean, standardized interface for database operations that works seamlessly with pgx and other PostgreSQL libraries.

## Overview

This package serves as the foundation for database interactions in GoStart applications by providing:

- **Standardized Database Interface**: Unified interface for database operations
- **pgx Integration**: Optimized for PostgreSQL with pgx/v5
- **Type Safety**: Strong typing for database operations
- **Framework Integration**: Seamless integration with other GoStart packages
- **Testing Support**: Mock-friendly interfaces for unit testing

## Core Interface

### DBTX Interface

```go
type DBTX interface {
    Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
    Query(context.Context, string, ...interface{}) (pgx.Rows, error)
    QueryRow(context.Context, string, ...interface{}) pgx.Row
}
```

This interface provides the essential database operations needed by most applications:

- **Exec**: Execute SQL commands that don't return rows (INSERT, UPDATE, DELETE)
- **Query**: Execute queries that return multiple rows
- **QueryRow**: Execute queries that return exactly one row

## Usage Examples

### Basic Database Operations

```go
import "github.com/kod2ulz/gostart/sqlc"

// Function that uses the DBTX interface
func CreateUser(ctx context.Context, db sqlc.DBTX, user *User) error {
    query := `
        INSERT INTO users (name, email, created_at)
        VALUES ($1, $2, $3)
        RETURNING id`

    return db.QueryRow(ctx, query, user.Name, user.Email, time.Now()).Scan(&user.ID)
}

func GetUserByID(ctx context.Context, db sqlc.DBTX, id string) (*User, error) {
    query := `
        SELECT id, name, email, created_at, updated_at
        FROM users
        WHERE id = $1`

    row := db.QueryRow(ctx, query, id)

    var user User
    err := row.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt, &user.UpdatedAt)
    if err != nil {
        return nil, err
    }

    return &user, nil
}

func UpdateUser(ctx context.Context, db sqlc.DBTX, user *User) error {
    query := `
        UPDATE users
        SET name = $1, email = $2, updated_at = $3
        WHERE id = $4`

    _, err := db.Exec(ctx, query, user.Name, user.Email, time.Now(), user.ID)
    return err
}
```

### Complex Query Operations

```go
func GetUsersWithFilters(ctx context.Context, db sqlc.DBTX, filters UserFilters) ([]User, error) {
    query := `
        SELECT id, name, email, created_at, updated_at
        FROM users
        WHERE 1=1`

    args := make([]interface{}, 0)
    argPos := 1

    if filters.Name != "" {
        query += fmt.Sprintf(" AND name ILIKE $%d", argPos)
        args = append(args, "%"+filters.Name+"%")
        argPos++
    }

    if filters.Email != "" {
        query += fmt.Sprintf(" AND email ILIKE $%d", argPos)
        args = append(args, "%"+filters.Email+"%")
        argPos++
    }

    if filters.Active != nil {
        query += fmt.Sprintf(" AND active = $%d", argPos)
        args = append(args, *filters.Active)
        argPos++
    }

    query += " ORDER BY created_at DESC"

    if filters.Limit > 0 {
        query += fmt.Sprintf(" LIMIT $%d", argPos)
        args = append(args, filters.Limit)
    }

    rows, err := db.Query(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var users []User
    for rows.Next() {
        var user User
        if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt, &user.UpdatedAt); err != nil {
            return nil, err
        }
        users = append(users, user)
    }

    return users, nil
}
```

## Integration with GoStart Framework

### Service Layer Integration

```go
type UserService struct {
    db sqlc.DBTX
}

func NewUserService(db sqlc.DBTX) *UserService {
    return &UserService{db: db}
}

func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, ierrors.Error) {
    user := &User{
        ID:        utils.UUID.New(),
        Name:      req.Name,
        Email:     req.Email,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    if err := CreateUser(ctx, s.db, user); err != nil {
        if isDuplicateKeyError(err) {
            return nil, errors.Conflict("user with this email already exists")
        }
        return nil, errors.SQLError[CreateUserRequest](err)
    }

    return user, nil
}

func (s *UserService) GetUser(ctx context.Context, userID string) (*User, ierrors.Error) {
    user, err := GetUserByID(ctx, s.db, userID)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, errors.NotFound[User](fmt.Sprintf("user not found: %s", userID))
        }
        return nil, errors.SQLError[string](err)
    }

    return user, nil
}
```

### Repository Pattern Implementation

```go
type UserRepository struct {
    db sqlc.DBTX
}

func NewUserRepository(db sqlc.DBTX) *UserRepository {
    return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *User) error {
    query := `
        INSERT INTO users (id, name, email, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5)`

    _, err := r.db.Exec(ctx, query, user.ID, user.Name, user.Email, user.CreatedAt, user.UpdatedAt)
    return err
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*User, error) {
    query := `
        SELECT id, name, email, created_at, updated_at
        FROM users
        WHERE id = $1`

    row := r.db.QueryRow(ctx, query, id)

    var user User
    err := row.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt, &user.UpdatedAt)
    if err != nil {
        return nil, err
    }

    return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *User) error {
    query := `
        UPDATE users
        SET name = $1, email = $2, updated_at = $3
        WHERE id = $4`

    user.UpdatedAt = time.Now()
    _, err := r.db.Exec(ctx, query, user.Name, user.Email, user.UpdatedAt, user.ID)
    return err
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
    query := `DELETE FROM users WHERE id = $1`
    _, err := r.db.Exec(ctx, query, id)
    return err
}
```

## Transaction Support

### Transaction Management

```go
type OrderService struct {
    db             sqlc.DBTX
    orderRepo      *OrderRepository
    inventoryRepo  *InventoryRepository
    notificationService *NotificationService
}

func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*Order, ierrors.Error) {
    // Begin transaction (this would be handled by the pgx pool)
    tx, err := s.db.Begin(ctx)
    if err != nil {
        return nil, errors.SQLError[CreateOrderRequest](err)
    }
    defer tx.Rollback(ctx)

    // Create order
    order := &Order{
        ID:         utils.UUID.New(),
        UserID:     req.UserID,
        Total:      req.Total,
        Status:     "pending",
        CreatedAt:  time.Now(),
    }

    if err := s.orderRepo.Create(ctx, tx, order); err != nil {
        return nil, errors.SQLError[CreateOrderRequest](err)
    }

    // Check inventory
    for _, item := range req.Items {
        inventory, err := s.inventoryRepo.GetByProductID(ctx, tx, item.ProductID)
        if err != nil {
            return nil, errors.SQLError[CreateOrderRequest](err)
        }

        if inventory.Quantity < item.Quantity {
            return nil, errors.InsufficientInventory(
                fmt.Sprintf("insufficient inventory for product %s", item.ProductID))
        }

        // Reserve inventory
        if err := s.inventoryRepo.Reserve(ctx, tx, item.ProductID, item.Quantity); err != nil {
            return nil, errors.SQLError[CreateOrderRequest](err)
        }
    }

    // Commit transaction
    if err := tx.Commit(ctx); err != nil {
        return nil, errors.SQLError[CreateOrderRequest](err)
    }

    // Send notification (async, outside transaction)
    go s.notificationService.SendOrderConfirmation(order.ID)

    return order, nil
}
```

## Testing Support

### Mock Database Interface

```go
// Mock implementation for testing
type MockDB struct {
    execFunc    func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error)
    queryFunc   func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error)
    queryRowFunc func(ctx context.Context, query string, args ...interface{}) pgx.Row
}

func (m *MockDB) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
    if m.execFunc != nil {
        return m.execFunc(ctx, query, args...)
    }
    return pgconn.NewCommandTag("SELECT 0"), nil
}

func (m *MockDB) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
    if m.queryFunc != nil {
        return m.queryFunc(ctx, query, args...)
    }
    return pgx.Rows{}, nil
}

func (m *MockDB) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
    if m.queryRowFunc != nil {
        return m.queryRowFunc(ctx, query, args...)
    }
    return pgx.Row{}
}

// Test example
func TestUserService_CreateUser(t *testing.T) {
    mockDB := &MockDB{
        queryRowFunc: func(ctx context.Context, query string, args ...interface{}) pgx.Row {
            return &mockRow{scanFunc: func(dest ...interface{}) error {
                if id, ok := dest[0].(*string); ok {
                    *id = "test-user-id"
                }
                return nil
            }}
        },
    }

    service := NewUserService(mockDB)
    req := &CreateUserRequest{
        Name:  "John Doe",
        Email: "john@example.com",
    }

    user, err := service.CreateUser(context.Background(), req)
    assert.NoError(t, err)
    assert.Equal(t, "test-user-id", user.ID)
    assert.Equal(t, "John Doe", user.Name)
}
```

## Best Practices

### Error Handling

```go
func safeDBOperation(ctx context.Context, db sqlc.DBTX, operation string) error {
    // Add context timeout
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    // Add logging
    start := time.Now()
    defer func() {
        logr.Info("Database operation completed",
            "operation", operation,
            "duration_ms", time.Since(start).Milliseconds(),
        )
    }()

    // Execute operation with error handling
    _, err := db.Exec(ctx, "SELECT 1")
    if err != nil {
        logr.Error("Database operation failed",
            "operation", operation,
            "error", err.Error(),
            "duration_ms", time.Since(start).Milliseconds(),
        )

        // Handle specific error types
        if isConnectionError(err) {
            return errors.ServiceUnavailable("database connection failed")
        }
        if isTimeoutError(err) {
            return errors.ServiceTimeout("database operation timed out")
        }

        return errors.SQLError[string](err)
    }

    return nil
}
```

### Query Optimization

```go
// Use prepared statements for repeated queries
var getUserStmt *pgx.PreparedStatement

func GetUserPrepared(ctx context.Context, db sqlc.DBTX, id string) (*User, error) {
    if getUserStmt == nil {
        var err error
        getUserStmt, err = db.Prepare(ctx, "get_user", `
            SELECT id, name, email, created_at, updated_at
            FROM users WHERE id = $1`)
        if err != nil {
            return nil, err
        }
    }

    row := getUserStmt.QueryRow(ctx, id)
    // Scan row...
}

// Use batch operations for multiple queries
func GetUsersBatch(ctx context.Context, db sqlc.DBTX, ids []string) ([]User, error) {
    batch := &pgx.Batch{}

    for _, id := range ids {
        batch.Queue("SELECT id, name, email FROM users WHERE id = $1", id)
    }

    results := db.SendBatch(ctx, defer results.Close())

    var users []User
    for _, id := range ids {
        row := results.QueryRow()
        var user User
        if err := row.Scan(&user.ID, &user.Name, &user.Email); err != nil {
            return nil, err
        }
        users = append(users, user)
    }

    return users, nil
}
```

## Configuration and Setup

### Database Connection Setup

```go
func setupDatabase(config *config.DatabaseConfig) (sqlc.DBTX, error) {
    // Parse connection string
    connString := fmt.Sprintf(
        "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
        config.Host,
        config.Port,
        config.Username,
        config.Password,
        config.Database,
        config.SSLMode,
    )

    // Create connection pool
    poolConfig, err := pgxpool.ParseConfig(connString)
    if err != nil {
        return nil, fmt.Errorf("failed to parse database config: %w", err)
    }

    // Configure pool settings
    poolConfig.MaxConns = config.MaxConnections
    poolConfig.MinConns = config.MinConnections
    poolConfig.MaxConnLifetime = config.MaxConnLifetime
    poolConfig.HealthCheckPeriod = time.Minute

    // Create pool
    pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create connection pool: %w", err)
    }

    // Test connection
    if err := pool.Ping(context.Background()); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    return pool, nil
}
```

The sqlc package provides a clean, minimal interface for database operations that serves as the foundation for robust data access layers in GoStart applications.