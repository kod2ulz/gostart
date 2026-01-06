# UUID Converter Functions Usage

The `contracts/convert.go` package now includes UUID converter functions following the same pattern as other type converters.

## Available Functions

### Value Converters (Database → Go)

#### `UUIDValue(val pgtype.UUID, fallback ...uuid.UUID) uuid.UUID`

Converts a `pgtype.UUID` from the database to a `uuid.UUID` in Go.

```go
// Convert database UUID to Go UUID
userID := contracts.UUIDValue(dbUser.ID)

// With fallback
userID := contracts.UUIDValue(dbUser.ID, uuid.MustParse("00000000-0000-0000-0000-000000000000"))
```

#### `NullableUUIDValue(val pgtype.UUID, fallback ...uuid.UUID) *uuid.UUID`

Converts a nullable `pgtype.UUID` to a `*uuid.UUID` pointer.

```go
// Returns *uuid.UUID or nil if NULL
userIDPtr := contracts.NullableUUIDValue(dbUser.ID)

// With fallback
userIDPtr := contracts.NullableUUIDValue(dbUser.ID, uuid.Nil)
```

### Parameter Converters (Go → Database)

#### `UUIDParam(val uuid.UUID) pgtype.UUID`

Converts a `uuid.UUID` to a `pgtype.UUID` for database operations.

```go
// Convert Go UUID to database UUID
id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
dbParam := contracts.UUIDParam(id)

// Use in database query
err := db.QueryRow(ctx, "SELECT * FROM users WHERE id = $1", contracts.UUIDParam(id))
```

#### `NullableUUIDParam(val *uuid.UUID, fallback ...uuid.UUID) pgtype.UUID`

Converts a `*uuid.UUID` pointer to `pgtype.UUID`.

```go
// Convert pointer to database parameter
var id *uuid.UUID
dbParam := contracts.NullableUUIDParam(id) // Returns NULL UUID if id is nil

// With fallback
dbParam := contracts.NullableUUIDParam(id, uuid.Nil)
```

#### `OptionalUUIDParam(val optional.String, fallback ...uuid.UUID) pgtype.UUID`

Converts an `optional.String` containing a UUID string to `pgtype.UUID`.

```go
// Convert optional string UUID to database parameter
optUUID := optional.NewString("550e8400-e29b-41d4-a716-446655440000")
dbParam := contracts.OptionalUUIDParam(optUUID)

// With fallback
dbParam := contracts.OptionalUUIDParam(optUUID, uuid.Nil)
```

## Usage Examples

### Example 1: Database Model Conversion

```go
type User struct {
    ID        uuid.UUID  `db:"id"`
    Name      string     `db:"name"`
    Email     *string    `db:"email"`
    ParentID  *uuid.UUID `db:"parent_id"`
}

type DBUser struct {
    ID        pgtype.UUID  `db:"id"`
    Name      pgtype.Text  `db:"name"`
    Email     pgtype.Text  `db:"email"`
    ParentID  pgtype.UUID  `db:"parent_id"`
}

// Convert from database model to Go model
func ToUser(dbUser DBUser) User {
    return User{
        ID:       contracts.UUIDValue(dbUser.ID),
        Name:     contracts.TextValue(dbUser.Name),
        Email:    contracts.NullableTextValue(dbUser.Email),
        ParentID: contracts.NullableUUIDValue(dbUser.ParentID),
    }
}

// Convert from Go model to database model
func ToDBUser(user User) DBUser {
    return DBUser{
        ID:       contracts.UUIDParam(user.ID),
        Name:     contracts.TextParam(user.Name),
        Email:    contracts.NullableTextParam(user.Email),
        ParentID: contracts.NullableUUIDParam(user.ParentID),
    }
}
```

### Example 2: Query Parameters

```go
func GetUserByID(ctx context.Context, db *pgx.Conn, id uuid.UUID) (*User, error) {
    var dbUser DBUser

    err := db.QueryRow(ctx,
        "SELECT id, name, email, parent_id FROM users WHERE id = $1",
        contracts.UUIDParam(id),
    ).Scan(
        &dbUser.ID, &dbUser.Name, &dbUser.Email, &dbUser.ParentID,
    )

    if err != nil {
        return nil, err
    }

    user := ToUser(dbUser)
    return &user, nil
}
```

### Example 3: Optional UUID Query Parameter

```go
func GetUsersByParent(
    ctx context.Context,
    db *pgx.Conn,
    parentID optional.String,
) ([]User, error) {
    rows, err := db.Query(ctx,
        "SELECT id, name, email, parent_id FROM users WHERE parent_id = $1",
        contracts.OptionalUUIDParam(parentID),
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var users []User
    for rows.Next() {
        var dbUser DBUser
        if err := rows.Scan(
            &dbUser.ID, &dbUser.Name, &dbUser.Email, &dbUser.ParentID,
        ); err != nil {
            return nil, err
        }
        users = append(users, ToUser(dbUser))
    }

    return users, nil
}
```

### Example 4: Insert with UUID

```go
func CreateUser(ctx context.Context, db *pgx.Conn, user User) error {
    id := uuid.New()

    _, err := db.Exec(ctx,
        `INSERT INTO users (id, name, email, parent_id)
         VALUES ($1, $2, $3, $4)`,
        contracts.UUIDParam(id),
        contracts.TextParam(user.Name),
        contracts.NullableTextParam(user.Email),
        contracts.NullableUUIDParam(user.ParentID),
    )

    return err
}
```

### Example 5: Update with Nullable UUID

```go
func SetUserParent(ctx context.Context, db *pgx.Conn, userID, parentID *uuid.UUID) error {
    _, err := db.Exec(ctx,
        "UPDATE users SET parent_id = $1 WHERE id = $2",
        contracts.NullableUUIDParam(parentID),
        contracts.NullableUUIDParam(userID),
    )
    return err
}
```

## Null UUID Handling

All UUID converters properly handle NULL/Nil values:

```go
// Database NULL → Go nil
var nullUUID pgtype.UUID
result := contracts.NullableUUIDValue(nullUUID) // Returns nil

// Go nil → Database NULL
var nilUUID *uuid.UUID
result := contracts.NullableUUIDParam(nilUUID) // Returns pgtype.UUID{Valid: false}

// Optional not present → Database NULL
optUUID := optional.String{}
result := contracts.OptionalUUIDParam(optUUID) // Returns pgtype.UUID{Valid: false}
```

## Fallback Values

You can provide fallback values for NULL UUIDs:

```go
var nullUUID pgtype.UUID

// With fallback to Nil
id := contracts.UUIDValue(nullUUID, uuid.Nil) // Returns uuid.Nil

// With custom fallback
fallbackID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
id := contracts.UUIDValue(nullUUID, fallbackID) // Returns fallbackID
```

## Pattern Consistency

These UUID converters follow the exact same pattern as all other type converters in the package:

- **Value functions**: Convert from database types (pgtype) to Go types
- **Param functions**: Convert from Go types to database types (pgtype)
- **Nullable functions**: Handle pointers and NULL values
- **Optional functions**: Handle optional types with fallback values
- **Fallback support**: All functions support optional fallback values
