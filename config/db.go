package config

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kod2ulz/gostart/collections"
)

// Dbtx is an interface for database operations, compatible with *pgxpool.Pool.
type Dbtx interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

// DBLookupFunc defines the signature for a custom database lookup function.
type DBLookupFunc func(ctx context.Context, db Dbtx, key string) (string, error)

// DBSeederFunc defines the signature for a custom database seeding function.
// It should return the value that was persisted.
type DBSeederFunc func(ctx context.Context, db Dbtx, key, value string) (string, error)

// configValue is a wrapper to make plain string values compatible with collections.Cache.
type configValue struct {
	key   string
	value string
}

func (v configValue) Key() string { return v.key }

// DBSource manages the database configuration source.
type DBSource struct {
	mu          sync.RWMutex
	db          Dbtx
	table       string
	cache       collections.Cache[string, configValue, error]
	cacheTTL    time.Duration
	lookupFunc  DBLookupFunc
	seederFunc  DBSeederFunc
	seedMissing bool
}

// DB provides access to the database configuration source.
var DB DBSource

// From sets the database connection pool and table name, and initializes the cache.
func (s *DBSource) From(db Dbtx, tableName string) *DBSource {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.db = db
	s.table = tableName

	fetcher := func(ctx context.Context, keys []string) ([]configValue, error) {
		// We need to read-lock here to safely access lookupFunc
		s.mu.RLock()
		lookup := s.lookupFunc
		s.mu.RUnlock()

		if lookup != nil {
			// If a custom lookup is provided, use it.
			values := make([]configValue, 0, len(keys))
			for _, key := range keys {
				val, err := lookup(ctx, s.db, key)
				if err == nil {
					values = append(values, configValue{key: key, value: val})
				} else if err != pgx.ErrNoRows {
					return nil, err
				}
			}
			return values, nil
		}
		// Otherwise, use the default table-based lookup.
		return s.getManyFromDB(ctx, keys)
	}

	// Use the configured TTL, or a default.
	ttl := s.cacheTTL
	if ttl == 0 {
		ttl = collections.DefaultCacheKeyTTL
	}

	s.cache = collections.NewMemoryCache[string, configValue, error](
		collections.WithFetcherFunc[string, configValue, error](fetcher),
		collections.WithDefaultTTL[string, configValue, error](ttl),
	)

	return s
}

// WithCacheTTL sets the cache TTL for the DB source. Must be called before From().
func (s *DBSource) WithCacheTTL(ttl time.Duration) *DBSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cacheTTL = ttl
	return s
}

// WithLookup sets a custom function for database lookups.
func (s *DBSource) WithLookup(fn DBLookupFunc) *DBSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lookupFunc = fn
	return s
}

// WithSeeder sets a custom function for seeding database values and enables seeding.
func (s *DBSource) WithSeeder(fn DBSeederFunc) *DBSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seederFunc = fn
	s.seedMissing = true // Implicitly enable seeding
	return s
}

// SeedMissing, if true, will cause the Get method to INSERT or UPDATE a row with the
// default value if a key is not found in the database. Use with caution.
func (s *DBSource) SeedMissing(seed bool) *DBSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seedMissing = seed
	return s
}

// Get retrieves a value from the database, using the cache.
func (s *DBSource) Get(key string, defaultValue ...interface{}) Value {
	if s.cache == nil {
		// If From() hasn't been called, we can't get a value.
		// Return invalid to allow fallback.
		return ""
	}

	val, err := s.cache.Get(context.Background(), key)
	if err == nil && val != nil {
		return Value(val.value)
	}

	// Value was not in cache and fetcher failed or returned empty.
	// Now, handle seeding.
	if s.seedMissing && len(defaultValue) > 0 {
		defValueStr := fmt.Sprint(defaultValue[0])
		seededValue, seedErr := s.seedToDB(context.Background(), key, defValueStr)
		if seedErr == nil {
			// Manually put the newly seeded value in the cache
			s.cache.Set(key, configValue{key: key, value: seededValue})
			return Value(seededValue)
		}
	}

	return ""
}

// getManyFromDB fetches multiple keys from the database.
// NOTE: This uses a simple loop. For better performance with many keys,
// this could be optimized to use a single `WHERE key = ANY($1)` query.
func (s *DBSource) getManyFromDB(ctx context.Context, keys []string) ([]configValue, error) {
	s.mu.RLock()
	db := s.db
	table := s.table
	s.mu.RUnlock()

	if db == nil || table == "" {
		return nil, fmt.Errorf("database source not configured")
	}

	// Inefficient loop, but simple. Can be replaced with a single query.
	values := make([]configValue, 0, len(keys))
	for _, key := range keys {
		query := fmt.Sprintf("SELECT value FROM %s WHERE key = $1", table)
		var value string
		err := db.QueryRow(ctx, query, key).Scan(&value)
		if err == nil {
			values = append(values, configValue{key: key, value: value})
		} else if err != pgx.ErrNoRows {
			return nil, err // Return on actual errors
		}
	}
	return values, nil
}

func (s *DBSource) seedToDB(ctx context.Context, key, value string) (string, error) {
	s.mu.RLock()
	db := s.db
	table := s.table
	seeder := s.seederFunc
	s.mu.RUnlock()

	if db == nil {
		return value, fmt.Errorf("database source not configured")
	}

	if seeder != nil {
		return seeder(ctx, db, key, value)
	}

	if table != "" {
		query := fmt.Sprintf(`INSERT INTO %s (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, table)
		_, err := db.Exec(ctx, query, key, value)
		return value, err
	}

	return value, nil
}