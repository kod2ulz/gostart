package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the base interface for all repositories
type Repository[T any, ID comparable] interface {
	// Create creates a new entity
	Create(ctx context.Context, entity *T) error

	// Get retrieves an entity by ID
	Get(ctx context.Context, id ID) (*T, error)

	// Update updates an existing entity
	Update(ctx context.Context, entity *T) error

	// Delete deletes an entity by ID
	Delete(ctx context.Context, id ID) error

	// List retrieves multiple entities with pagination
	List(ctx context.Context, opts ListOptions) ([]*T, error)

	// Count returns the total count of entities
	Count(ctx context.Context, filter Filter) (int64, error)

	// Exists checks if an entity exists
	Exists(ctx context.Context, id ID) (bool, error)
}

// ReadRepository defines read-only operations
type ReadRepository[T any, ID comparable] interface {
	Get(ctx context.Context, id ID) (*T, error)
	List(ctx context.Context, opts ListOptions) ([]*T, error)
	Count(ctx context.Context, filter Filter) (int64, error)
	Exists(ctx context.Context, id ID) (bool, error)
}

// WriteRepository defines write-only operations
type WriteRepository[T any, ID comparable] interface {
	Create(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id ID) error
}

// TransactionalRepository adds transaction support
type TransactionalRepository[T any, ID comparable] interface {
	Repository[T, ID]

	// WithTransaction executes a function within a transaction
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// CacheableRepository adds caching support
type CacheableRepository[T any, ID comparable] interface {
	Repository[T, ID]

	// InvalidateCache invalidates the cache for an entity
	InvalidateCache(ctx context.Context, id ID) error

	// ClearCache clears all cached entities
	ClearCache(ctx context.Context) error
}

// ListOptions defines options for listing entities
type ListOptions struct {
	Offset int
	Limit  int
	Sort   []SortOption
	Filter Filter
}

// SortOption defines sorting parameters
type SortOption struct {
	Field      string
	Descending bool
}

// Filter defines filtering criteria
type Filter interface {
	// ToSQL converts the filter to SQL WHERE clause
	ToSQL() (string, []interface{})

	// ToMongo converts the filter to MongoDB query
	ToMongo() map[string]interface{}
}

// SimpleFilter implements basic filtering
type SimpleFilter struct {
	Field    string
	Operator string
	Value    interface{}
}

func (f SimpleFilter) ToSQL() (string, []interface{}) {
	return f.Field + " " + f.Operator + " ?", []interface{}{f.Value}
}

func (f SimpleFilter) ToMongo() map[string]interface{} {
	return map[string]interface{}{
		f.Field: map[string]interface{}{
			"$" + f.Operator: f.Value,
		},
	}
}

// AndFilter combines multiple filters with AND
type AndFilter struct {
	Filters []Filter
}

func (f AndFilter) ToSQL() (string, []interface{}) {
	var clauses []string
	var args []interface{}

	for _, filter := range f.Filters {
		sql, filterArgs := filter.ToSQL()
		clauses = append(clauses, "("+sql+")")
		args = append(args, filterArgs...)
	}

	return "(" + join(clauses, " AND ") + ")", args
}

func (f AndFilter) ToMongo() map[string]interface{} {
	conditions := make([]map[string]interface{}, len(f.Filters))
	for i, filter := range f.Filters {
		conditions[i] = filter.ToMongo()
	}
	return map[string]interface{}{"$and": conditions}
}

// OrFilter combines multiple filters with OR
type OrFilter struct {
	Filters []Filter
}

func (f OrFilter) ToSQL() (string, []interface{}) {
	var clauses []string
	var args []interface{}

	for _, filter := range f.Filters {
		sql, filterArgs := filter.ToSQL()
		clauses = append(clauses, "("+sql+")")
		args = append(args, filterArgs...)
	}

	return "(" + join(clauses, " OR ") + ")", args
}

func (f OrFilter) ToMongo() map[string]interface{} {
	conditions := make([]map[string]interface{}, len(f.Filters))
	for i, filter := range f.Filters {
		conditions[i] = filter.ToMongo()
	}
	return map[string]interface{}{"$or": conditions}
}

// CacheStore defines the interface for caching
type CacheStore interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
	Exists(ctx context.Context, key string) (bool, error)
}

// EventStore defines the interface for event sourcing
type EventStore interface {
	// AppendEvent appends an event to the stream
	AppendEvent(ctx context.Context, streamID string, event Event) error

	// GetEvents retrieves events from a stream
	GetEvents(ctx context.Context, streamID string, fromVersion int64) ([]Event, error)

	// GetAllEvents retrieves all events from a stream
	GetAllEvents(ctx context.Context, streamID string) ([]Event, error)
}

// Event represents a domain event
type Event interface {
	GetID() uuid.UUID
	GetType() string
	GetTimestamp() time.Time
	GetData() interface{}
	GetMetadata() map[string]interface{}
}

// BaseEvent provides a basic event implementation
type BaseEvent struct {
	ID        uuid.UUID              `json:"id"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      interface{}            `json:"data"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func (e BaseEvent) GetID() uuid.UUID                   { return e.ID }
func (e BaseEvent) GetType() string                    { return e.Type }
func (e BaseEvent) GetTimestamp() time.Time            { return e.Timestamp }
func (e BaseEvent) GetData() interface{}               { return e.Data }
func (e BaseEvent) GetMetadata() map[string]interface{} { return e.Metadata }

// Helper function to join strings
func join(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
