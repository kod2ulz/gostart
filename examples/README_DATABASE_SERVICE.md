# Database Service Example

This example demonstrates how to create a production-ready REST API service using GoStart with PostgreSQL database connectivity.

## Features Demonstrated

- **Connection Pooling**: Uses `pgxpool.Pool` instead of single connection
- **Type-Safe Handlers**: Uses `TypedHandler` for compile-time type safety
- **Service Layer**: Clean separation between handlers and business logic
- **Configuration**: Environment-based configuration with `storage.Config()`
- **Pagination**: Built-in support for paginated list responses
- **Validation**: Request validation using struct tags
- **Error Handling**: Centralized error handling with proper HTTP status codes
- **OpenAPI/Swagger**: Automatic API documentation at `/swagger`

## Prerequisites

1. **PostgreSQL database** running locally or accessible
2. **Go 1.21+** installed
3. **Environment variables** configured (see `.env.example`)

## Setup

### 1. Create Environment File

```bash
cp examples/.env.example .env
```

### 2. Update Environment Variables

Edit `.env` and update the database configuration:

```env
POSTGRES_DB_HOST=localhost
POSTGRES_DB_PORT=5432
POSTGRES_DB_USERNAME=postgres
POSTGRES_DB_PASSWORD=your_password
POSTGRES_DB_DATABASE=user_service
```

### 3. Create Database

```bash
createdb user_service
```

### 4. Run the Application

```bash
go run examples/database_service_example.go
```

The application will:
- Automatically create the `users` table
- Start the HTTP server on port 8080
- Expose Swagger UI at `http://localhost:8080/swagger`

## API Endpoints

### Create User

```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john.doe@example.com",
    "first_name": "John",
    "last_name": "Doe"
  }'
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "john.doe@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "full_name": "John Doe",
    "active": true,
    "created_at": "2025-01-04T12:34:56Z",
    "updated_at": "2025-01-04T12:34:56Z"
  }
}
```

### Get User

```bash
curl http://localhost:8080/users/{id}
```

### List Users (with pagination)

```bash
# Get all users
curl http://localhost:8080/users

# Get active users only
curl http://localhost:8080/users?active_only=true

# Search by email
curl http://localhost:8080/users?email=john

# Paginate
curl http://localhost:8080/users?limit=10&offset=0
```

**Response:**
```json
{
  "success": true,
  "data": [...],
  "total": 100,
  "limit": 10,
  "offset": 0
}
```

### Update User

```bash
curl -X PUT http://localhost:8080/users/{id} \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Jane",
    "active": false
  }'
```

Note: Use pointers (`*string`, `*bool`) for partial updates.

### Delete User

```bash
curl -X DELETE http://localhost:8080/users/{id}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "user deleted successfully",
    "id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

## Project Structure

```
examples/database_service_example.go
├── Domain Models (User)
├── Request Models (CreateUserRequest, UpdateUserRequest, etc.)
├── Response Models (UserResponse)
├── Service Layer (UserService)
│   ├── CreateUser
│   ├── GetUserByID
│   ├── UpdateUser
│   ├── ListUsers
│   └── DeleteUser
├── Handlers (TypedHandler functions)
└── Main (database initialization, routing)
```

## Key Concepts

### 1. Connection Pool with pgxpool

```go
// Instead of single connection (BAD)
db, err := pgx.Connect(ctx, connString)

// Use connection pool (GOOD)
pool, err := pgxpool.New(ctx, connString)
defer pool.Close()
```

### 2. Configuration from Environment

```go
// Reads from POSTGRES_DB_* environment variables
dbConf := storage.Config("POSTGRES_DB")
pool, err := pgxpool.New(ctx, dbConf.ConnectionString())
```

### 3. Type-Safe Handlers with TypedHandler

```go
// Define request struct
type CreateUserRequest struct {
    Email     string `json:"email" validate:"required,email"`
    FirstName string `json:"first_name" validate:"required"`
}

// Handler receives concrete type
func CreateUserHandler(ctx contracts.RequestContext, req CreateUserRequest) (UserResponse, ierrors.Error) {
    // No type assertions needed!
    user, err := userService.CreateUser(req)
    return user.ToResponse(), err
}

// Register with TypedHandler
router.POST("/users", api.TypedHandler(CreateUserRequest, CreateUserHandler))
```

### 4. Service Layer Pattern

```go
type UserService struct {
    db  *pgxpool.Pool
    log *logr.Logger
    ctx context.Context
}

func (s *UserService) CreateUser(req CreateUserRequest) (*User, ierrors.Error) {
    // Business logic here
    // Database operations here
    // Validation here
}
```

### 5. Pagination Support

```go
type ListUsersRequest struct {
    api.ListRequest // Embed for pagination
    SearchEmail     string `query:"email"`
    ActiveOnly      bool   `query:"active_only"`
}

// Handler returns ([]UserResponse, *int64, ierrors.Error)
func ListUsersHandler(ctx contracts.RequestContext, req ListUsersRequest) ([]UserResponse, *int64, ierrors.Error) {
    users, total, err := userService.ListUsers(req)
    return responses, total, err
}
```

### 6. Partial Updates with Pointers

```go
type UpdateUserRequest struct {
    Email     *string `json:"email" validate:"omitempty,email"`     // Pointer
    FirstName *string `json:"first_name" validate:"omitempty"`      // Pointer
    Active    *bool   `json:"active"`                               // Pointer
}

// Only update fields that are provided (non-nil)
if req.FirstName != nil {
    user.FirstName = *req.FirstName
}
```

## Database Migrations

The example includes automatic migration in `runMigrations()`:

```go
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
    createTableSQL := `
    CREATE TABLE IF NOT EXISTS users (
        id VARCHAR(36) PRIMARY KEY,
        email VARCHAR(255) NOT NULL UNIQUE,
        first_name VARCHAR(100) NOT NULL,
        last_name VARCHAR(100) NOT NULL,
        active BOOLEAN DEFAULT true,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
    CREATE INDEX IF NOT EXISTS idx_users_active ON users(active);
    `

    if _, err := pool.Exec(ctx, createTableSQL); err != nil {
        return fmt.Errorf("failed to create users table: %w", err)
    }

    return nil
}
```

For production, consider using a migration tool like:
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [goose](https://github.com/pressly/goose)
- [sqlc](https://sqlc.dev/)

## Error Handling

The service uses GoStart's centralized error handling:

```go
import "github.com/kod2ulz/gostart/errors"

// Not found (404)
return nil, errors.NotFound("user not found: %s", id)

// Conflict (409)
return nil, errors.Conflict("email %s already exists", email)

// Bad request (400)
return nil, errors.BadRequest("invalid email format")

// General failure (500)
return nil, errors.GeneralFailure("database error")
```

## Testing

You can test the API using:
- **Swagger UI**: `http://localhost:8080/swagger`
- **curl**: See examples above
- **Postman**: Import the OpenAPI spec from `http://localhost:8080/openapi.json`

## Next Steps

1. **Add authentication**: Use the `auth/` package for JWT/PASETO tokens
2. **Add caching**: Use Redis for frequently accessed data
3. **Add message queue**: Use `mq/` package for async processing
4. **Add metrics**: Use Prometheus for observability
5. **Add tests**: Write unit and integration tests

## Production Checklist

- [ ] Use environment variables for all configuration
- [ ] Enable SSL/TLS for database connections
- [ ] Set appropriate connection pool sizes
- [ ] Add request timeout middleware
- [ ] Add rate limiting
- [ ] Add authentication/authorization
- [ ] Add structured logging with correlation IDs
- [ ] Add health check endpoints (already included with `app.WithHeartbeatHandlers()`)
- [ ] Add metrics/monitoring
- [ ] Use a proper migration tool
- [ ] Add comprehensive tests
- [ ] Set up CI/CD pipeline
