# Collections Package

This package provides a set of generic, thread-safe, and feature-rich data structures designed to improve upon the built-in Go types.

## Overview

Go's built-in slices and maps are powerful but can be verbose for common operations. This package provides enhanced data structures with more convenient methods while maintaining the performance characteristics you expect from Go.

The collections are designed to be:
- **Generic**: Work with any type using Go's generics
- **Thread-safe**: Concurrent access patterns for shared data
- **Intuitive**: Methods that follow familiar patterns from other languages
- **Performant**: Minimal overhead with efficient implementations

## Core Data Structures

### List[T] - Enhanced Slice Operations

The `List[T]` type extends Go slices with additional methods for common operations:

```go
// Basic operations
users := List([]User{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}})

// Add elements
users.Append(User{ID: 3, Name: "Charlie"})

// Check if empty
if users.Empty() {
    fmt.Println("No users")
}

// Get first and last elements
first := users.First()
last := users.Last()

// Find elements matching a condition
activeUsers := users.Filter(func(u User) bool {
    return u.IsActive
})

// Find first matching element
admin := users.Any(func(u User) bool {
    return u.Role == "admin"
})

// Sort with custom comparator
sorted := users.Sort(func(a, b User) bool {
    return a.Name < b.Name
})

// In-place stable sort
users.SortStable(func(a, b User) int {
    return strings.Compare(a.Name, b.Name)
})

// Iterate over elements
users.Iterate(func(i int, user User) error {
    fmt.Printf("User %d: %s\n", i, user.Name)
    return nil
})

// Apply transformation
userIDs := users.ForEach(func(i int, user User) string {
    return user.ID
})
```

### ConcurrentMap[K, V] - Thread-Safe Dictionary

For scenarios where multiple goroutines need to access shared data:

```go
// Create thread-safe map
sessionCache := NewConcurrentMap[string, Session]()

// Concurrent operations are safe
go func() {
    for _, session := range sessions {
        sessionCache.Set(session.ID, session)
    }
}()

go func() {
    if session, exists := sessionCache.Get("session123"); exists {
        processSession(session)
    }
}()

// Atomic operations
counter := NewConcurrentMap[string, int]()
counter.Increment("request_count")
counter.Decrement("active_connections")

// Range over entries safely
sessionCache.Range(func(key string, session Session) bool {
    fmt.Printf("Session %s: %v\n", key, session)
    return true // continue iteration
})
```

### Cache[K, T] - Generic Caching Interface

For caching data with automatic expiration and loading:

```go
// Cache interface for different implementations
type Cache[K comparable, T CacheModel[K, T], E error] interface {
    Get(key K) (T, error)
    Set(key K, value T) error
    Delete(key K) error
    Clear() error
}

// Memory-based implementation
userCache := &memoryCache[string, User, error]{
    items:    make(map[string]cacheItem[User]),
    ttl:      time.Hour,
    loadFn:   loadUserFromDB,
}

// Usage
user, err := userCache.Get("user123")
if err != nil {
    // Handle error or load from database
}
```

### Set[T] - Set Operations

For managing unique collections:

```go
// Create set from slice
tags := Set([]string{"golang", "api", "database"})

// Check membership
hasTag := tags.Has("golang")

// Add and remove elements
tags.Add("web")
tags.Remove("legacy")

// Set operations
allTags := tags.Union(otherTags)
commonTags := tags.Intersection(otherTags)
uniqueTags := tags.Difference(otherTags)
```

## Real-World Usage Patterns

### Data Processing Pipelines

```go
// Process log entries
func processLogs(logs []LogEntry) []ProcessedLog {
    return List(logs).
        Filter(func(entry LogEntry) bool {
            return entry.Level == "ERROR" || entry.Level == "WARN"
        }).
        Sort(func(a, b LogEntry) bool {
            return a.Timestamp.After(b.Timestamp)
        }).
        Slice(0, 1000) // Get first 1000 entries
}
```

### Configuration Management

```go
// Manage application configuration
type ConfigManager struct {
    sources *ConcurrentMap[string, ConfigSource]
    cache   *memoryCache[string, interface{}, error]
}

func (cm *ConfigManager) Get(key string) (interface{}, error) {
    // Try cache first
    if value, err := cm.cache.Get(key); err == nil {
        return value, nil
    }

    // Load from sources
    for _, source := range cm.sources.Values() {
        if value, exists := source.Get(key); exists {
            cm.cache.Set(key, value)
            return value, nil
        }
    }

    return nil, fmt.Errorf("config key not found: %s", key)
}
```

### Session Management

```go
// Thread-safe session storage
type SessionManager struct {
    sessions *ConcurrentMap[string, Session]
    cleanup  *time.Ticker
}

func (sm *SessionManager) Create(userID string) *Session {
    session := &Session{
        ID:        generateSessionID(),
        UserID:    userID,
        CreatedAt: time.Now(),
        ExpiresAt: time.Now().Add(24 * time.Hour),
    }

    sm.sessions.Set(session.ID, session)
    return session
}

func (sm *SessionManager) Validate(sessionID string) (*Session, error) {
    session, exists := sm.sessions.Get(sessionID)
    if !exists {
        return nil, errors.New("session not found")
    }

    if time.Now().After(session.ExpiresAt) {
        sm.sessions.Delete(sessionID)
        return nil, errors.New("session expired")
    }

    return session, nil
}
```

## Performance Considerations

### Memory Usage

All operations create new collections rather than modifying existing ones. This ensures thread safety but means you should be mindful of memory usage with large datasets:

```go
// Efficient for small to medium datasets
result := largeList.Filter(func(item Item) bool {
    return item.IsActive
})

// For very large datasets, consider streaming or batch processing
func processLargeDataset[T any](items []T, batchSize int, processor func([]T)) {
    List(items).Chunk(batchSize).ForEach(func(i int, batch []T) {
        processor(batch)
    })
}
```

### Concurrency

The thread-safe structures use appropriate locking strategies:

```go
// ConcurrentMap uses fine-grained locking
var counter ConcurrentMap[string, int]

// Multiple goroutines can safely update different keys
go func() {
    counter.Increment("counter1")
}()

go func() {
    counter.Increment("counter2")
}()

// Operations on the same key are serialized
go func() {
    counter.Increment("counter1") // This will wait if counter1 is being updated
}()
```

## Integration with Other Packages

The collections package integrates well with other GoStart packages:

```go
import (
    "github.com/kod2ulz/gostart/collections"
    "github.com/kod2ulz/gostart/config"
    "github.com/kod2ulz/gostart/errors"
)

// Use with configuration
func loadUsers() []User {
    dbHost := config.Get("database.host", "localhost").String()
    users := fetchUsersFromDB(dbHost)

    return collections.List(users).Filter(func(u User) bool {
        return u.IsActive
    })
}

// Use with error handling
func safeProcess(items []Item) ([]ProcessedItem, error) {
    results := collections.List(items).ForEach(func(i int, item Item) ProcessedItem {
        result, err := processItem(item)
        if err != nil {
            return ProcessedItem{Error: err}
        }
        return result
    })

    // Check for errors
    if collections.List(results).Any(func(p ProcessedItem) bool {
        return p.Error != nil
    }) != nil {
        return nil, errors.GeneralFailure("processing failed")
    }

    return results, nil
}
```

## When to Use This Package

### Well-Suited For:

- **Data transformation pipelines**: Filter, sort, and process collections
- **Concurrent applications**: Shared data structures accessed by multiple goroutines
- **API response processing**: Clean, readable data manipulation
- **Session management**: Thread-safe user session storage
- **Configuration systems**: Cached configuration with multiple sources

### Consider Standard Go For:

- **Simple operations**: Basic slice operations where built-ins suffice
- **Performance-critical code**: Raw loops may be faster for simple cases
- **Memory-constrained environments**: The additional features have some overhead
- **Small datasets**: Overhead may not justify benefits for trivial use cases

## Migration from Standard Go

### From Slices to List[T]

```go
// Before: Standard Go
var activeUsers []User
for _, user := range users {
    if user.IsActive {
        activeUsers = append(activeUsers, user)
    }
}

sort.Slice(activeUsers, func(i, j int) bool {
    return activeUsers[i].Name < activeUsers[j].Name
})

// After: Collections
activeUsers := collections.List(users).
    Filter(func(u User) bool { return u.IsActive }).
    Sort(func(a, b User) bool { return a.Name < b.Name })
```

### From Maps to ConcurrentMap[K, V]

```go
// Before: Standard Go with mutex
var (
    sessions = make(map[string]Session)
    mutex    sync.RWMutex
)

func getSession(id string) (Session, bool) {
    mutex.RLock()
    defer mutex.RUnlock()
    session, exists := sessions[id]
    return session, exists
}

// After: ConcurrentMap
sessions := collections.NewConcurrentMap[string, Session]()
session, exists := sessions.Get(id)
```

## Implementation Status

### ✅ **Fully Implemented Data Structures**
- **Generic Cache System** - Thread-safe caching with TTL, background eviction, and prefix-based management
- **ConcurrentMap[K, V]** - Thread-safe dictionary with atomic operations, range iteration, and fine-grained locking
- **List[T]** - Enhanced slice operations with `Filter()`, `Sort()`, `SortStable()`, `Any()`, `ForEach()`, `First()`, `Last()`
- **Set[T]** - Set operations with union, intersection, difference, and membership testing
- **Iterator Pattern** - Generic iteration support for all collections
- **Cache Model Interface** - Extensible caching system for custom types

### ✅ **Well-Implemented Utility Methods**
- **List Operations**: `Filter()`, `Sort(lessFn)`, `SortStable(comp)`, `Any(predicate)`, `ForEach(fn)`, `First()`, `Last()`, `Empty()`, `Append()`, `Iterate()`
- **ConcurrentMap Operations**: `Set()`, `Get()`, `Delete()`, `Increment()`, `Decrement()`, `Range()`, `Values()`
- **Set Operations**: `Has()`, `Add()`, `Remove()`, `Union()`, `Intersection()`, `Difference()`
- **Cache Operations**: `Get()`, `Set()`, `Delete()`, `Clear()`, TTL management, background cleanup

### ⚠️ **Needs Verification/Completion**
- **Tree Structures** - Referenced in file structure but implementation needs verification
- **Advanced List Methods** - Some utility methods may be incomplete
- **Performance Optimization** - Further optimization for large datasets may be needed

### 🚧 **Planned Enhancements**
- **Additional Data Structures** - Trees, graphs, priority queues
- **Performance Benchmarks** - Comprehensive performance testing and optimization
- **Memory Pooling** - Object pooling for reduced GC pressure
- **Stream Processing** - Real-time data stream processing capabilities

## Detailed Documentation

For comprehensive guides on specific features:

- **[List Operations](./docs/list.md)** - Complete reference for List[T] methods
- **[Concurrent Data Structures](./docs/concurrent-structures.md)** - Thread-safe patterns and examples
- **[Performance Optimization](./docs/performance.md)** - Performance characteristics and best practices

This package provides practical enhancements to Go's built-in collections while maintaining the language's performance characteristics and idioms.