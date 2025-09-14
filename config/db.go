package config

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

// Dbtx is an interface for database transaction, compatible with *pgxpool.Pool.
type Dbtx interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

// DBLookupFunc defines the signature for a custom database lookup function.
// This allows users to provide their own logic for retrieving config values.
type DBLookupFunc func(ctx context.Context, db Dbtx, key string) (string, error)

// dbCacheEntry holds a cached value and its expiration time.
type dbCacheEntry struct {
	value      string
	expiration time.Time
}

// dbSource manages the database configuration source.
type dbSource struct {
	mu         sync.RWMutex
	db         Dbtx
	table      string
	cache      map[string]dbCacheEntry
	cacheTTL   time.Duration
	lookupFunc DBLookupFunc
}

// DB provides access to the database configuration source.
var DB dbSource

// From sets the database connection pool and table name to be used for the default query.
func (s *dbSource) From(db Dbtx, tableName string) *dbSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db = db
	s.table = tableName
	return s
}

// WithCache sets the cache TTL for database-retrieved values.
func (s *dbSource) WithCache(ttl time.Duration) *dbSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cacheTTL = ttl
	if s.cache == nil {
		s.cache = make(map[string]dbCacheEntry)
	}
	return s
}

// WithLookup sets a custom function for database lookups.
func (s *dbSource) WithLookup(fn DBLookupFunc) *dbSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lookupFunc = fn
	return s
}

// Get retrieves a value from the database, using a cache if configured.
func (s *dbSource) Get(key string, defaultValue ...interface{}) Value {
	s.mu.RLock()
	useCache := s.cache != nil && s.cacheTTL > 0
	if useCache {
		if entry, found := s.cache[key]; found && time.Now().Before(entry.expiration) {
			s.mu.RUnlock()
			return Value(entry.value)
		}
	}
	s.mu.RUnlock()

	// Value not in cache or expired, query the database
	var dbValue string
	var err error

	if s.db != nil {
		s.mu.RLock()
		lookup := s.lookupFunc
		table := s.table
		db := s.db
		s.mu.RUnlock()

		if lookup != nil {
			// Use custom lookup function
			dbValue, err = lookup(context.Background(), db, key)
		} else if table != "" {
			// Use default simple query
			query := fmt.Sprintf("SELECT value FROM %s WHERE key = $1", table)
			err = db.QueryRow(context.Background(), query, key).Scan(&dbValue)
		}
	}

	if err == nil && dbValue != "" {
		// Found in DB, update cache if enabled
		s.mu.Lock()
		if useCache {
			s.cache[key] = dbCacheEntry{
				value:      dbValue,
				expiration: time.Now().Add(s.cacheTTL),
			}
		}
		s.mu.Unlock()
		return Value(dbValue)
	}

	// Not found in DB or DB not configured, use default
	if len(defaultValue) > 0 {
		return Value(fmt.Sprint(defaultValue[0]))
	}

	return ""
}

