# Storage Package

This package provides utilities for data storage, including an in-memory caching system and a factory for creating Redis clients.

## Redis Client

A Redis client can be easily created from a `storage.Conf` object. This is typically handled by the main application setup.

### Usage

```go
import "github.com/kod2ulz/gostart/storage"

// Assuming 'conf' is a loaded *storage.Conf object
redisClient := storage.Redis(conf)
```

## In-Memory Cache

The package includes a generic, thread-safe in-memory cache (`memoryCache`) that features automatic data loading and configurable TTLs.

### Core Concepts

- **`CacheModel[K, T]`**: Your cached objects must implement this interface, which requires a `Key() K` method to uniquely identify the object.
- **Fetcher Function**: The cache is write-through and read-through. It requires a "fetcher" function (`fetchCacheKeysFromStorageFunc`) that it can call to load objects from the persistent data store (e.g., a database) when they are not in the cache or have expired.

### Initialization

Here's how to create a new `memoryCache`.

```go
import (
    "context"
    "github.com/kod2ulz/gostart/storage"
)

// 1. Define your model
type User struct {
    ID   string
    Name string
}

func (u User) Key() string { // Implement the CacheModel interface
    return u.ID
}

// 2. Define your fetcher function
fetcher := func(ctx context.Context, keys []string) ([]User, error) {
    // In a real application, you would fetch these users from a database
    // users, err := myUserRepo.FindUsersByIDs(ctx, keys)
    // return users, err
    
    // Dummy implementation:
    var users []User
    for _, key := range keys {
        users = append(users, User{ID: key, Name: "User " + key})
    }
    return users, nil
}

// 3. Create the cache instance
userCache := storage.NewMemoryCache[string, User, error](
    logger, // Your application logger
    storage.WithFetcherFunc[string, User, error](fetcher),
    storage.WithDefaultTTL[string, User, error](5 * time.Minute),
)
```

### Usage

Once initialized, you can get objects from the cache. If an object is missing or expired, the cache will automatically call your fetcher function to load it.

```go
// Get a single user. If not in cache, it will be fetched.
user := userCache.Get("user-123")
if user != nil {
    fmt.Printf("Got user: %s", user.Name)
}

// Pre-load multiple users into the cache
users, err := userCache.Fetch(context.Background(), "user-456", "user-789")
```

## Roadmap

- **Configuration Backends:** Support for loading configurations from sources other than environment variables, such as HashiCorp Vault, YAML, or JSON files.
- **Dynamic Configuration:** Introduce a caching layer for configuration that can be refreshed without restarting the application.
- **Expanded Cache Support:** Add support for other caching backends besides the default in-memory cache.

## End-to-End Initialization Example

The `storage.Config` function is a key part of bootstrapping a service. It reads environment variables based on a prefix to configure a component.

Here is an example of how `storage.Config` is used in a `main.go` file to set up a database connection.

```go
package main

import (
	"context"

	"github.com/kod2ulz/gostart/app"
	"github.com/kod2ulz/gostart/logr"
	"github.com/kod2ulz/gostart/storage"
	"github.com/kod2ulz/gostart/utils"

	"github.com/jackc/pgx/v5/pgxpool"
)

// This function would live in your project's database package
func InitDB(ctx context.Context, log *logr.Logger, conf *storage.Conf) (*pgxpool.Pool, error) {
    connString := conf.ConnectionString()
    config, err := pgxpool.ParseConfig(connString)
    if err != nil {
        return nil, err
    }

    // Attach the pgx/v5 logger
    config.ConnConfig.Tracer = storage.NewPgxLogger(log)

    return pgxpool.NewWithConfig(ctx, config)
}

func main() {
	// Initialize the core application
	app := app.Init()
	ctx, log := app.Ctx(), app.Log()

	// Use storage.Config to get database configuration from environment variables
	// (e.g., POSTGRES_DB_HOST, POSTGRES_DB_PORT, etc.)
	dbConf := storage.Config("POSTGRES_DB")

	// Pass the config to your database initializer
	db_conn, err := InitDB(ctx, log, dbConf)
	utils.Error.Fail(log.Entry, err, "failed to connect to database")
	defer db_conn.Close()

	log.Info("Database connection successful!")

	// ... rest of your application
	app.Run()
}
```
