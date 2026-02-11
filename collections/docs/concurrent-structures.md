# Concurrent Data Structures

The collections package provides thread-safe implementations of common data structures for concurrent applications.

## ConcurrentMap[K, T]

The `ConcurrentMap[K, T]` is a thread-safe dictionary that supports high-performance concurrent access with fine-grained locking.

### Basic Usage

```go
// Create a concurrent map
sessionCache := collections.NewConcurrentMap[string, Session]()

// Concurrent writes and reads
go func() {
    for _, session := range userSessions {
        sessionCache.Set(session.ID, session)
    }
}()

go func() {
    if session, exists := sessionCache.Get("session123"); exists {
        processSession(session)
    }
}()
```

### Core Operations

```go
// Basic operations
userCache := collections.NewConcurrentMap[string, User]()

// Set and Get
userCache.Set("user123", User{ID: "user123", Name: "Alice"})
if user, exists := userCache.Get("user123"); exists {
    fmt.Printf("Found user: %s\n", user.Name)
}

// Delete
userCache.Delete("user123")

// Check existence
hasUser := userCache.Has("user123")

// Get or set with default function
user := userCache.GetOrSet("user456", func() User {
    return User{ID: "user456", Name: "Bob"}
})

// Get with default value
userName := userCache.GetOrElse("user789", User{Name: "Unknown"}).Name
```

### Atomic Operations

```go
// Atomic updates
counter := collections.NewConcurrentMap[string, int]()

// Increment atomically
counter.Increment("request_count")
counter.IncrementBy("request_count", 5)

// Decrement atomically
counter.Decrement("active_connections")

// Compare and swap
success := counter.CompareAndSwap("feature_flag", false, true)
if success {
    fmt.Println("Feature flag enabled")
}
```

### Bulk Operations

```go
// Get all keys and values
allKeys := userCache.Keys()
allValues := userCache.Values()
allEntries := userCache.Entries()

// Clear the map
userCache.Clear()

// Get map size
size := userCache.Size()

// Check if empty
isEmpty := userCache.IsEmpty()
```

### Range Operations

```go
// Safe iteration over map contents
userCache.Range(func(key string, user User) bool {
    fmt.Printf("Processing user %s: %s\n", key, user.Name)
    return true // continue iteration
})

// Conditional iteration
userCache.Range(func(key string, user User) bool {
    if !user.IsActive {
        return false // stop iteration
    }
    processActiveUser(user)
    return true
})
```

### Real-World Examples

#### Session Management

```go
type SessionManager struct {
    sessions *collections.ConcurrentMap[string, Session]
    timeouts *collections.ConcurrentMap[string, time.Time]
}

func NewSessionManager() *SessionManager {
    return &SessionManager{
        sessions: collections.NewConcurrentMap[string, Session](),
        timeouts: collections.NewConcurrentMap[string, time.Time](),
    }
}

func (sm *SessionManager) CreateSession(userID string) string {
    sessionID := generateSessionID()
    session := Session{
        ID:        sessionID,
        UserID:    userID,
        CreatedAt: time.Now(),
        ExpiresAt: time.Now().Add(24 * time.Hour),
    }

    sm.sessions.Set(sessionID, session)
    sm.timeouts.Set(sessionID, session.ExpiresAt)

    return sessionID
}

func (sm *SessionManager) ValidateSession(sessionID string) (*Session, bool) {
    session, exists := sm.sessions.Get(sessionID)
    if !exists {
        return nil, false
    }

    // Check if expired
    if time.Now().After(session.ExpiresAt) {
        sm.sessions.Delete(sessionID)
        sm.timeouts.Delete(sessionID)
        return nil, false
    }

    return &session, true
}

func (sm *SessionManager) CleanupExpiredSessions() {
    sm.timeouts.Range(func(sessionID string, expiry time.Time) bool {
        if time.Now().After(expiry) {
            sm.sessions.Delete(sessionID)
            sm.timeouts.Delete(sessionID)
        }
        return true
    })
}
```

#### Rate Limiting

```go
type RateLimiter struct {
    requests *collections.ConcurrentMap[string, []time.Time]
    limits   *collections.ConcurrentMap[string, int]
}

func NewRateLimiter() *RateLimiter {
    return &RateLimiter{
        requests: collections.NewConcurrentMap[string, []time.Time](),
        limits:   collections.NewConcurrentMap[string, int](),
    }
}

func (rl *RateLimiter) Allow(clientID string, limit int, window time.Duration) bool {
    now := time.Now()
    cutoff := now.Add(-window)

    // Get or create request history
    history := rl.requests.GetOrSet(clientID, func() []time.Time {
        return []time.Time{}
    })

    // Filter old requests
    recent := collections.List(history).Filter(func(t time.Time) bool {
        return t.After(cutoff)
    }).ToSlice()

    // Check if limit exceeded
    if len(recent) >= limit {
        return false
    }

    // Add current request
    recent = append(recent, now)
    rl.requests.Set(clientID, recent)

    return true
}
```

#### Cache with Expiration

```go
type CacheItem struct {
    Value      interface{}
    ExpiresAt  time.Time
    AccessCount int
}

type TTLCache struct {
    items *collections.ConcurrentMap[string, CacheItem]
    stats *collections.ConcurrentMap[string, CacheStats]
}

func (c *TTLCache) Get(key string) (interface{}, bool) {
    item, exists := c.items.Get(key)
    if !exists {
        return nil, false
    }

    // Check expiration
    if time.Now().After(item.ExpiresAt) {
        c.items.Delete(key)
        return nil, false
    }

    // Update access count
    item.AccessCount++
    c.items.Set(key, *item)

    return item.Value, true
}

func (c *TTLCache) Set(key string, value interface{}, ttl time.Duration) {
    item := CacheItem{
        Value:      value,
        ExpiresAt:  time.Now().Add(ttl),
        AccessCount: 0,
    }
    c.items.Set(key, item)
}
```

## ConcurrentList[T]

The `ConcurrentList[T]` provides thread-safe operations for lists, perfect for concurrent data processing.

### Basic Operations

```go
// Create concurrent list
taskQueue := collections.NewConcurrentList[Task]()

// Thread-safe operations
go func() {
    taskQueue.Append(Task{ID: 1, Action: "process"})
}()

go func() {
    taskQueue.Prepend(Task{ID: 2, Action: "analyze"})
}()

// Safe iteration
taskQueue.ForEach(func(task Task) {
    fmt.Printf("Processing task: %d\n", task.ID)
})
```

### Queue Operations

```go
// Producer-consumer pattern
taskQueue := collections.NewConcurrentList[Task]()

// Producer
go func() {
    for i := 0; i < 100; i++ {
        task := Task{ID: i, Action: fmt.Sprintf("task_%d", i)}
        taskQueue.Append(task)
        time.Sleep(time.Millisecond * 10)
    }
}()

// Consumer
go func() {
    for {
        if task, ok := taskQueue.Shift(); ok {
            processTask(task)
        } else {
            time.Sleep(time.Millisecond * 100)
        }
    }
}()
```

### Stack Operations

```go
// Stack usage for LIFO processing
callStack := collections.NewConcurrentList[CallFrame]()

// Push
callStack.Append(CallFrame{Function: "main", Line: 10})
callStack.Append(CallFrame{Function: "process", Line: 25})

// Pop
if frame, ok := callStack.Pop(); ok {
    fmt.Printf("Returning from: %s\n", frame.Function)
}
```

### Concurrent Processing

```go
// Parallel processing of list items
func processConcurrently(items []WorkItem) []Result {
    results := collections.NewConcurrentList[Result]()
    wg := sync.WaitGroup{}

    for _, item := range items {
        wg.Add(1)
        go func(workItem WorkItem) {
            defer wg.Done()
            result := processWorkItem(workItem)
            results.Append(result)
        }(item)
    }

    wg.Wait()
    return results.ToSlice()
}
```

## Cache[K, T]

The `Cache[K, T]` provides intelligent caching with automatic expiration and background refresh.

### Basic Usage

```go
// Create cache with TTL and loader function
userCache := collections.NewCache[string, User](time.Hour, func(userID string) (User, error) {
    return database.GetUserByID(userID)
})

// Get value (cached or fresh)
user, err := userCache.Get("user123")
if err != nil {
    log.Printf("Failed to get user: %v", err)
    return
}

fmt.Printf("User: %s\n", user.Name)
```

### Advanced Configuration

```go
// Cache with custom options
config := CacheConfig{
    TTL:           time.Hour,
    CleanupInterval: 5 * time.Minute,
    MaxSize:       10000,
    Loader: func(key string) (interface{}, error) {
        return loadExpensiveData(key)
    },
}

cache := collections.NewCacheWithConfig(config)
```

### Cache Strategies

```go
// Cache with refresh ahead
type RefreshAheadCache struct {
    cache *collections.Cache[string, Data]
}

func (rac *RefreshAheadCache) Get(key string) (Data, error) {
    data, err := rac.cache.Get(key)
    if err != nil {
        return Data{}, err
    }

    // Refresh if close to expiration
    if time.Until(data.ExpiresAt) < time.Minute*5 {
        go rac.refreshInBackground(key)
    }

    return data, nil
}
```

### Real-World Example: API Rate Limiting

```go
type APIRateLimiter struct {
    limits   *collections.ConcurrentMap[string, RateLimit]
    requests *collections.ConcurrentMap[string, []time.Time]
}

type RateLimit struct {
    RequestsPerMinute int
    BurstSize        int
}

func NewAPIRateLimiter() *APIRateLimiter {
    return &APIRateLimiter{
        limits:   collections.NewConcurrentMap[string, RateLimit](),
        requests: collections.NewConcurrentMap[string, []time.Time](),
    }
}

func (rl *APIRateLimiter) SetLimit(clientID string, requestsPerMinute, burstSize int) {
    rl.limits.Set(clientID, RateLimit{
        RequestsPerMinute: requestsPerMinute,
        BurstSize:        burstSize,
    })
}

func (rl *APIRateLimiter) AllowRequest(clientID string) bool {
    limit, exists := rl.limits.Get(clientID)
    if !exists {
        return true // No limit set
    }

    now := time.Now()
    windowStart := now.Add(-time.Minute)

    // Get request history
    history := rl.requests.GetOrSet(clientID, func() []time.Time {
        return []time.Time{}
    })

    // Filter to current window
    recent := collections.List(history).Filter(func(t time.Time) bool {
        return t.After(windowStart)
    }).ToSlice()

    // Check limits
    if len(recent) >= limit.RequestsPerMinute {
        return false
    }

    // Check burst limit
    if len(recent) >= limit.BurstSize {
        // Check if requests are spread out
        oldest := recent[0]
        if now.Sub(oldest) < time.Second {
            return false
        }
    }

    // Add current request
    recent = append(recent, now)
    rl.requests.Set(clientID, recent)

    return true
}
```

### Real-World Example: Session Store

```go
type SessionStore struct {
    sessions  *collections.ConcurrentMap[string, Session]
    userIndex *collections.ConcurrentMap[string, []string] // user_id -> session_ids
    stats     *collections.ConcurrentMap[string, SessionStats]
}

func NewSessionStore() *SessionStore {
    return &SessionStore{
        sessions:  collections.NewConcurrentMap[string, Session](),
        userIndex: collections.NewConcurrentMap[string, []string](),
        stats:     collections.NewConcurrentMap[string, SessionStats](),
    }
}

func (ss *SessionStore) CreateSession(userID string, data SessionData) (string, error) {
    sessionID := generateSessionID()
    session := Session{
        ID:        sessionID,
        UserID:    userID,
        Data:      data,
        CreatedAt: time.Now(),
        ExpiresAt: time.Now().Add(24 * time.Hour),
    }

    // Store session
    ss.sessions.Set(sessionID, session)

    // Update user index
    ss.userIndex.GetOrSet(userID, func() []string {
        return []string{}
    })
    userSessions := ss.userIndex.Get(userID)
    if userSessions != nil {
        userSessions = append(userSessions, sessionID)
        ss.userIndex.Set(userID, userSessions)
    }

    // Update stats
    ss.stats.GetOrSet(userID, func() SessionStats {
        return SessionStats{SessionCount: 0}
    })
    stats := ss.stats.Get(userID)
    if stats != nil {
        stats.SessionCount++
        ss.stats.Set(userID, *stats)
    }

    return sessionID, nil
}

func (ss *SessionStore) GetUserSessions(userID string) []Session {
    sessionIDs := ss.userIndex.Get(userID)
    if sessionIDs == nil {
        return []Session{}
    }

    var sessions []Session
    for _, sessionID := range sessionIDs {
        if session, exists := ss.sessions.Get(sessionID); exists {
            if time.Now().Before(session.ExpiresAt) {
                sessions = append(sessions, session)
            }
        }
    }

    return sessions
}
```

### Performance Considerations

#### Lock Granularity

```go
// Fine-grained locking for better performance
type OptimizedConcurrentMap[K, V] struct {
    shards []*collections.ConcurrentMap[K, V]
    count  int
}

func NewOptimizedConcurrentMap[K, V](shardCount int) *OptimizedConcurrentMap[K, V] {
    shards := make([]*collections.ConcurrentMap[K, V], shardCount)
    for i := range shards {
        shards[i] = collections.NewConcurrentMap[K, V]()
    }
    return &OptimizedConcurrentMap[K, V]{
        shards: shards,
        count:  shardCount,
    }
}

func (m *OptimizedConcurrentMap[K, V]) getShard(key K) *collections.ConcurrentMap[K, V] {
    hash := fnv.New32a()
    hash.Write([]byte(fmt.Sprintf("%v", key)))
    return m.shards[int(hash.Sum32())%m.count]
}

func (m *OptimizedConcurrentMap[K, V]) Set(key K, value V) {
    shard := m.getShard(key)
    shard.Set(key, value)
}

func (m *OptimizedConcurrentMap[K, V]) Get(key K) (V, bool) {
    shard := m.getShard(key)
    return shard.Get(key)
}
```

#### Memory Management

```go
// Cache with memory limits and eviction
type MemoryAwareCache[K, V] struct {
    cache     *collections.ConcurrentMap[K, cacheItem[V]]
    maxSize   int
    currentSize int64
    mu        sync.RWMutex
}

type cacheItem[V] struct {
    value     V
    size      int64
    lastAccess time.Time
}

func (c *MemoryAwareCache[K, V]) Set(key K, value V, size int64) {
    item := cacheItem[V]{
        value:      value,
        size:       size,
        lastAccess: time.Now(),
    }

    // Check if we need to evict
    c.mu.Lock()
    if c.currentSize+size > int64(c.maxSize) {
        c.evictLRU(int64(c.maxSize) - (c.currentSize + size))
    }
    c.mu.Unlock()

    c.cache.Set(key, item)
}

func (c *MemoryAwareCache[K, V]) evictLRU(required int64) {
    // Find least recently used items
    var itemsToEvict []K
    c.cache.Range(func(key K, item cacheItem[V]) bool {
        if len(itemsToEvict) < 10 { // Evict in batches
            itemsToEvict = append(itemsToEvict, key)
        }
        return true
    })

    // Evict items
    for _, key := range itemsToEvict {
        if item, exists := c.cache.Get(key); exists {
            c.cache.Delete(key)
            c.currentSize -= item.size
            if c.currentSize <= required {
                break
            }
        }
    }
}
```

Concurrent data structures in the collections package provide the foundation for building high-performance, thread-safe applications with complex data processing requirements.