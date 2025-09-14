package config

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Dbtx is an interface for database operations, compatible with *pgxpool.Pool.
type Dbtx interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

// DBLookupFunc defines the signature for a custom database lookup function.
type DBLookupFunc func(ctx context.Context, db Dbtx, key string) (string, error)

// DBSeederFunc defines the signature for a custom database seeding function.
// It should return the value that was persisted.
type DBSeederFunc func(ctx context.Context, db Dbtx, key, value string) (string, error)

// dbCacheEntry holds a cached value and its expiration time.
type dbCacheEntry struct {
	value      string
	expiration time.Time
}

// dbSource manages the database configuration source.
type dbSource struct {
	mu          sync.RWMutex
	db          Dbtx
	table       string
	cache       map[string]dbCacheEntry
	cacheTTL    time.Duration
	lookupFunc  DBLookupFunc
	seederFunc  DBSeederFunc
	seedMissing bool
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

// WithSeeder sets a custom function for seeding database values and enables seeding.
func (s *dbSource) WithSeeder(fn DBSeederFunc) *dbSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seederFunc = fn
	s.seedMissing = true // Implicitly enable seeding
	return s
}

// SeedMissing, if true, will cause the Get method to INSERT or UPDATE a row with the
// default value if a key is not found in the database. Use with caution.
func (s *dbSource) SeedMissing(seed bool) *dbSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seedMissing = seed
	return s
}

// Get retrieves a value from the database, using a cache if configured.
func (s *dbSource) Get(key string, defaultValue ...interface{}) Value {
	s.mu.RLock()
	// 1. Check cache
	useCache := s.cache != nil && s.cacheTTL > 0
	if useCache {
		if entry, found := s.cache[key]; found && time.Now().Before(entry.expiration) {
			s.mu.RUnlock()
			return Value(entry.value)
		}
	}
	db := s.db
	shouldSeed := s.seedMissing
	s.mu.RUnlock()

	// 2. Try to get from DB
	if db != nil {
		dbValue, err := s.getFromDB(context.Background(), key)

		// 2a. If successful, cache and return
		if err == nil {
			s.updateCache(key, dbValue)
			return Value(dbValue)
		}

		// 2b. If "not found" and seeding is enabled, seed the value
		if err == pgx.ErrNoRows && len(defaultValue) > 0 {
			defValue := fmt.Sprint(defaultValue[0])
			if shouldSeed {
				seededValue, seedErr := s.seedToDB(context.Background(), key, defValue)
				if seedErr == nil {
					s.updateCache(key, seededValue) // Cache the value that was actually seeded
					return Value(seededValue)
				}
			}
			return Value(defValue)
		}
	}

	// 3. Fallback to default
	if len(defaultValue) > 0 {
		return Value(fmt.Sprint(defaultValue[0]))
	}

	return ""
}

func (s *dbSource) getFromDB(ctx context.Context, key string) (string, error) {
	s.mu.RLock()
	lookup := s.lookupFunc
	table := s.table
	db := s.db
	s.mu.RUnlock()

	if lookup != nil {
		return lookup(ctx, db, key)
	} else if table != "" {
		query := fmt.Sprintf("SELECT value FROM %s WHERE key = $1", table)
		var value string
		err := db.QueryRow(ctx, query, key).Scan(&value)
		return value, err
	}
	return "", pgx.ErrNoRows
}

func (s *dbSource) seedToDB(ctx context.Context, key, value string) (string, error) {
	s.mu.RLock()
	db := s.db
	table := s.table
	seeder := s.seederFunc
	s.mu.RUnlock()

	if db == nil {
		return value, nil
	}

	if seeder != nil {
		return seeder(ctx, db, key, value)
	}

	if table != "" {
		// Default "upsert" logic
		query := fmt.Sprintf(`INSERT INTO %s (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, table)
		_, err := db.Exec(ctx, query, key, value)
		return value, err
	}

	return value, nil
}

func (s *dbSource) updateCache(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cache != nil && s.cacheTTL > 0 {
		s.cache[key] = dbCacheEntry{
			value:      value,
			expiration: time.Now().Add(s.cacheTTL),
		}
	}
}