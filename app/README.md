# App Package

This package is the heart of a `gostart` application. It handles application initialization, lifecycle management, configuration, and routing.

## Overview

The `app.Init()` function is the main entry point. It sets up the core application instance, loads configuration from environment variables, initializes the logger, and prepares the Gin HTTP router.

The initialized `app` object provides access to the application context (`Ctx()`), the logger (`Log()`), and the router (`Router()`).

## Application Initialization Example

Here is a complete example demonstrating how to initialize a `gostart` application and its components, based on a real-world service.

```go
package main

import (
	"context"

	"github.com/kod2ulz/gostart/app"
	"github.com/kod2ulz/gostart/logr"
	"github.com/kod2ulz/gostart/mq"
	"github.com/kod2ulz/gostart/storage"
	"github.com/kod2ulz/gostart/utils"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Define a struct to hold your application's database connection
// This would typically be in its own package (e.g., /db)
type AppDatabase struct {
	Conn *pgxpool.Pool
	// You can add your sqlc Queries struct here
}

// This function initializes the database connection.
// It demonstrates how to use storage.Config and the pgx/v5 logger.
func InitDB(ctx context.Context, log *logr.Logger, conf *storage.Conf) (*AppDatabase, error) {
	connString := conf.ConnectionString()

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}

	// Use the new pgx/v5 logger from the storage package
	config.ConnConfig.Tracer = storage.NewPgxLogger(log)

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	return &AppDatabase{Conn: pool}, nil
}

func main() {
	// 1. Initialize the gostart application
	app := app.Init()
	ctx, log, router := app.Ctx(), app.Log(), app.Router()

	// 2. Initialize the Database
	// storage.Config reads environment variables with the given prefix (e.g., POSTGRES_DB_HOST)
	dbConf := storage.Config("POSTGRES_DB")
	dbx, err := InitDB(ctx, log, dbConf)
	utils.Error.Fail(log.Entry, err, "failed to connect to database")
	defer dbx.Conn.Close() // Ensure the connection is closed on shutdown

	// 3. Initialize the Message Queue
	rmqConf := mq.Config("RABBIT_MQ")
	rmq := mq.Load(ctx, rmqConf, log)
	defer utils.ErrorFunc[utils.ShFunc2](app, rmq.Close, "error closing rabbitMQ connection")

	// 4. (Optional) Register with Consul
	utils.Error.Fail(log.Entry, app.Register(), "failed to register with consul")

	// 5. Define API routes
	api := router.Group("/api")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 6. Run the application
	// This is a blocking call that will wait for a shutdown signal (e.g., Ctrl+C)
	app.Run()
}
```
