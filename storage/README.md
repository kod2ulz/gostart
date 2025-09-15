# Storage Package

This package provides utilities for data storage, primarily a factory for creating Redis clients.

## Redis Client

A Redis client can be easily created by providing a configuration prefix. The factory will then use the main `gostart/config` package to retrieve the necessary connection details.

### Usage

```go
import "github.com/kod2ulz/gostart/storage"

// Create a redis client using settings from keys prefixed with "MY_APP_REDIS_"
// (e.g., MY_APP_REDIS_HOST, MY_APP_REDIS_PORT)
redisClient := storage.Redis("MY_APP_REDIS")
```

## Datastore Configuration (`storage.Config`)

The `storage.Config` function is a helper for bootstrapping datastore configurations. It reads environment variables based on a prefix to populate a `storage.Conf` struct, which can then be used to generate a `ConnectionString`.

This is particularly useful for initializing database connections like PostgreSQL.

### End-to-End Example

Here is an example of how `storage.Config` is used in a `main.go` file to set up a database connection. Note how it uses the main `config` package to load a `config.yaml` file first, and `storage.Config` can then draw from those values if they are present, falling back to environment variables if they are not.

```go
package main

import (
	"context"

	"github.com/kod2ulz/gostart/app"
	"github.com/kod2ulz/gostart/config"
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
	// Load hierarchical config from file/env/etc.
	if err := config.Yaml.Load("config.yaml"); err != nil {
		// handle error
	}

	// Initialize the core application
	app := app.Init()
	ctx, log := app.Ctx(), app.Log()

	// Use storage.Config to get database configuration.
	// It will pull from the values loaded by the main config package.
	// Keys would be e.g., POSTGRES_DB_HOST, POSTGRES_DB_PORT, etc.
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