# Storage Package

The `storage` package provides a unified abstraction layer for configuring and initializing various data storage technologies. It serves as the central configuration point for database connections, caching systems, and other persistence layers, enabling consistent setup patterns across different storage backends.

## Overview

This package is designed to be **storage-technology agnostic**, providing a common interface for configuration management while supporting specific implementations for different storage systems. Currently supports:

- **PostgreSQL** (via pgx/v5)
- **Redis** (caching and pub/sub)
- **Extensible architecture** for future additions (DGraph, MinIO, Neo4j, etc.)

## Core Philosophy

The storage package follows these principles:

- **Configuration Unification**: Single entry point for all storage configuration
- **Environment-Aware**: Seamless integration with GoStart's hierarchical configuration system
- **Technology Agnostic**: Consistent patterns regardless of underlying storage technology
- **Production-Ready**: Built-in connection pooling, logging, and health checking

## Current Implementations

### PostgreSQL Integration

The package provides streamlined PostgreSQL configuration and initialization:

```go
import "github.com/kod2ulz/gostart/storage"

// Configure PostgreSQL connection
dbConf := storage.Config("POSTGRES_DB")
connString := dbConf.ConnectionString()

// Use with pgx/v5 for connection pooling
pool, err := pgxpool.New(context.Background(), connString)
```

### Redis Integration

Redis client creation with automatic configuration management:

```go
import "github.com/kod2ulz/gostart/storage"

// Create Redis client using configuration prefix
redisClient := storage.Redis("MY_APP_REDIS")
// Reads from MY_APP_REDIS_HOST, MY_APP_REDIS_PORT, etc.
```

## Configuration Pattern

All storage technologies follow the same configuration pattern:

1. **Define a configuration prefix** (e.g., "POSTGRES_DB", "REDIS_CACHE")
2. **Use the appropriate factory function** (`storage.Config()`, `storage.Redis()`, etc.)
3. **Get configured client/connection** ready for use

The system automatically reads from GoStart's hierarchical configuration:
- YAML configuration files
- Environment variables
- Database settings (for dynamic configuration)
- HashiCorp Vault (for secrets)
- Default values

## Usage Examples

### Database Configuration

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

func InitDB(ctx context.Context, log *logr.Logger, conf *storage.Conf) (*pgxpool.Pool, error) {
    connString := conf.ConnectionString()
    config, err := pgxpool.ParseConfig(connString)
    if err != nil {
        return nil, err
    }

    // Attach structured logging
    config.ConnConfig.Tracer = storage.NewPgxLogger(log)

    return pgxpool.NewWithConfig(ctx, config)
}

func main() {
    // Load hierarchical configuration
    if err := config.Yaml.Load("config.yaml"); err != nil {
        // handle error
    }

    app := app.Init()
    ctx, log := app.Ctx(), app.Log()

    // Configure PostgreSQL
    dbConf := storage.Config("POSTGRES_DB")
    dbConn, err := InitDB(ctx, log, dbConf)
    utils.Error.Fail(log.Entry, err, "failed to connect to database")
    defer dbConn.Close()

    // Configure Redis for caching
    redisClient := storage.Redis("CACHE_REDIS")

    log.Info("Storage systems initialized successfully!")
    app.Run()
}
```

## Extensibility

The storage package is designed to be easily extended with new storage technologies:

1. **Configuration Structure**: Define configuration parameters for the new technology
2. **Factory Function**: Create a factory function that uses `storage.Config()`
3. **Integration**: Follow the established patterns for logging and health checking

### Example: Adding DGraph Support

```go
// Future extension example
func DGraph(configPrefix string) (*dgraph.Client, error) {
    conf := storage.Config(configPrefix)
    // Configure DGraph client using conf.Host, conf.Port, etc.
    return dgraphClient, nil
}
```

## Integration with GoStart Ecosystem

The storage package seamlessly integrates with other GoStart packages:

- **[`config`](../config/README.md)**: Hierarchical configuration management
- **[`logr`](../logr/README.md)**: Structured logging with pgx integration
- **[`errors`](../errors/README.md)**: Enhanced error handling for storage operations
- **[`query`](../query/README.md)**: Query builders and criteria patterns

## Implementation Status

### ⚠️ **Current State: Basic Configuration Only**

**IMPORTANT:** This package provides only basic configuration utilities. No actual storage abstractions or query builders are implemented.

### ✅ **Implemented Features**
- **Redis Client Factory** - Basic Redis client creation with authentication
- **PostgreSQL Configuration** - Basic database configuration structure
- **Connection String Generation** - Utility for building database connection strings
- **pgx Logger Integration** - Basic logging integration for PostgreSQL

### ❌ **NOT Implemented (Documentation Overstates Capabilities)**
- **Storage Abstractions** - No generic storage interfaces exist
- **Query Builders** - No type-safe query builders implemented
- **Caching Layer** - No intelligent caching system exists
- **Database Connection Management** - Basic configuration only, no pooling or health monitoring
- **Multiple Storage Technologies** - Only basic Redis and PostgreSQL setup
- **Migration System** - No schema migration tools
- **Transaction Management** - No transaction support beyond basic pgx

### 🚧 **What This Package Actually Provides**
This package is currently just a configuration utility that helps set up Redis and PostgreSQL clients. It does NOT provide:
- Generic storage interfaces
- Query builders or ORM functionality
- Advanced caching systems
- Multi-technology storage abstractions
- Health monitoring or automatic reconnection

## Roadmap

### 🚧 **Planned Enhancements**
- **Storage Interface Abstractions** - Generic interfaces for multiple storage backends
- **Query Builder System** - Type-safe query construction
- **Advanced Caching Layer** - Intelligent caching with invalidation strategies
- **Additional Storage Technologies** - DGraph, MinIO, Neo4j, MongoDB, Cassandra support
- **Connection Management** - Connection pooling, health monitoring, automatic reconnection
- **Migration Tools** - Database schema migration utilities
- **Performance Monitoring** - Query metrics and optimization insights