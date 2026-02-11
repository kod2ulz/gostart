# Performance Optimization Guide

The collections package is designed with performance in mind, providing various optimization strategies for different use cases.

## Understanding Performance Characteristics

### Lazy Evaluation

Most collections operations are lazy, meaning they create computation pipelines rather than immediate results:

```go
// This creates a pipeline, not intermediate lists
pipeline := collections.List(largeDataset).
    Filter(func(item Data) bool { return item.IsActive }).
    Map(func(item Data) Processed { return item.Process() }).
    SortBy(func(p Processed) int { return p.Priority })

// No computation happens until you call a terminal operation
results := pipeline.ToSlice() // Computation happens here
```

### Zero-Copy Views

Many operations create views rather than copies:

```go
// Efficient operations that don't create intermediate arrays
original := collections.List([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})

// Filter creates a view, not a new list
filtered := original.Filter(func(n int) bool { return n%2 == 0 })

// Map creates a view, not a new list
mapped := filtered.Map(func(n int) int { return n * 2 })

// Only when you call ToSlice() does the actual computation happen
results := mapped.Take(5).ToSlice() // [4, 8, 12, 16, 20]
```

## Memory Efficiency

### Chaining Operations

```go
// Efficient: Single pass with no intermediate allocations
result := collections.List(data).
    Filter(func(item Item) bool { return item.IsActive }).
    Map(func(item Item) Output { return transform(item) }).
    Take(100)

// Inefficient: Multiple passes with intermediate allocations
var activeItems []Item
for _, item := range data {
    if item.IsActive {
        activeItems = append(activeItems, item)
    }
}

var transformed []Output
for _, item := range activeItems {
    transformed = append(transformed, transform(item))
}

result = transformed[:100]
```

### Batch Processing

```go
// Process large datasets in chunks
func processLargeDataset(data []Item) []Result {
    const batchSize = 1000

    return collections.List(data).
        Chunk(batchSize).
        FlatMap(func(batch []Item) []Result {
            return processBatch(batch)
        }).
        ToSlice()
}

func processBatch(batch []Item) []Result {
    // Process batch in parallel
    results := make([]Result, len(batch))
    var wg sync.WaitGroup

    for i, item := range batch {
        wg.Add(1)
        go func(idx int, it Item) {
            defer wg.Done()
            results[idx] = processItem(it)
        }(i, item)
    }

    wg.Wait()
    return results
}
```

## Parallel Processing

### Parallel Map

```go
// Parallel processing with configurable worker count
func parallelProcess[T, R any](items []T, mapper func(T) R, workers int) []R {
    results := make([]R, len(items))
    jobs := make(chan int, len(items))
    resultsChan := make(chan struct {
        index int
        value R
    }, len(items))

    // Start workers
    var wg sync.WaitGroup
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for idx := range jobs {
                resultsChan <- struct {
                    index int
                    value R
                }{idx, mapper(items[idx])}
            }
        }()
    }

    // Send jobs
    go func() {
        for i := range items {
            jobs <- i
        }
        close(jobs)
    }()

    // Collect results
    go func() {
        wg.Wait()
        close(resultsChan)
    }()

    for result := range resultsChan {
        results[result.index] = result.value
    }

    return results
}

// Usage
results := parallelProcess(data, expensiveTransform, runtime.NumCPU())
```

### Parallel Filter

```go
func parallelFilter[T any](items []T, predicate func(T) bool, workers int) []T {
    var filtered []T
    var mu sync.Mutex

    collections.List(items).
        Chunk(len(items) / workers).
        ForEach(func(chunk []T) {
            var wg sync.WaitGroup
            wg.Add(1)
            go func() {
                defer wg.Done()
                localFiltered := collections.List(chunk).
                    Filter(predicate).
                    ToSlice()

                mu.Lock()
                filtered = append(filtered, localFiltered...)
                mu.Unlock()
            }()
        })

    return filtered
}
```

### Work Stealing Pattern

```go
type WorkStealingPool struct {
    tasks    chan Task
    workers  int
    stop     chan struct{}
    wg       sync.WaitGroup
}

func NewWorkStealingPool(workers int) *WorkStealingPool {
    pool := &WorkStealingPool{
        tasks:   make(chan Task, 100),
        workers: workers,
        stop:    make(chan struct{}),
    }

    pool.start()
    return pool
}

func (p *WorkStealingPool) start() {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker()
    }
}

func (p *WorkStealingPool) worker() {
    defer p.wg.Done()

    for {
        select {
        case task := <-p.tasks:
            task.Execute()
        case <-p.stop:
            return
        }
    }
}

func (p *WorkStealingPool) Submit(task Task) {
    p.tasks <- task
}

func (p *WorkStealingPool) Stop() {
    close(p.stop)
    p.wg.Wait()
}
```

## Caching Strategies

### Memoization

```go
type MemoizedFunction[T, R any] struct {
    cache *collections.ConcurrentMap[T, R]
    fn    func(T) R
}

func NewMemoizedFunction[T, R any](fn func(T) R) *MemoizedFunction[T, R] {
    return &MemoizedFunction[T, R]{
        cache: collections.NewConcurrentMap[T, R](),
        fn:    fn,
    }
}

func (m *MemoizedFunction[T, R]) Call(input T) R {
    if result, exists := m.cache.Get(input); exists {
        return result
    }

    result := m.fn(input)
    m.cache.Set(input, result)
    return result
}

// Usage
expensiveCalc := NewMemoizedFunction(func(n int) int {
    time.Sleep(time.Second) // Simulate expensive operation
    return n * n
})

// First call: takes 1 second
result1 := expensiveCalc.Call(5)

// Second call: instantaneous
result2 := expensiveCalc.Call(5)
```

### Cache with Expiration

```go
type TTLCache[K comparable, V any] struct {
    items *collections.ConcurrentMap[K, cacheItem[V]]
    ttl   time.Duration
}

type cacheItem[V any] struct {
    value      V
    expiresAt  time.Time
    accessedAt time.Time
}

func (c *TTLCache[K, V]) Get(key K) (V, bool) {
    item, exists := c.items.Get(key)
    if !exists {
        var zero V
        return zero, false
    }

    if time.Now().After(item.expiresAt) {
        c.items.Delete(key)
        var zero V
        return zero, false
    }

    // Update access time
    item.accessedAt = time.Now()
    c.items.Set(key, *item)

    return item.value, true
}

func (c *TTLCache[K, V]) Set(key K, value V) {
    item := cacheItem[V]{
        value:      value,
        expiresAt:  time.Now().Add(c.ttl),
        accessedAt: time.Now(),
    }
    c.items.Set(key, item)
}

func (c *TTLCache[K, V]) Cleanup() {
    now := time.Now()
    c.items.Range(func(key K, item cacheItem[V]) bool {
        if now.After(item.expiresAt) {
            c.items.Delete(key)
        }
        return true
    })
}
```

## Algorithmic Optimizations

### Efficient Grouping

```go
// Efficient grouping with minimal allocations
func efficientGroupBy[T any, K comparable](items []T, keyFunc func(T) K) map[K][]T {
    groups := make(map[K][]T)

    // Pre-allocate slices based on estimated group sizes
    estimatedCount := make(map[K]int)
    for _, item := range items {
        key := keyFunc(item)
        estimatedCount[key]++
    }

    // Pre-allocate slices
    for key, count := range estimatedCount {
        groups[key] = make([]T, 0, count)
    }

    // Group items
    for _, item := range items {
        key := keyFunc(item)
        groups[key] = append(groups[key], item)
    }

    return groups
}

// Usage
customersByRegion := efficientGroupBy(customers, func(c Customer) string {
    return c.Region
})
```

### Sorting Optimization

```go
// Hybrid sort for large datasets
func hybridSort[T any](items []T, less func(a, b T) bool) {
    const quickSortThreshold = 32

    if len(items) <= quickSortThreshold {
        insertionSort(items, less)
        return
    }

    // Quick sort for large datasets
    quickSort(items, 0, len(items)-1, less)
}

func insertionSort[T any](items []T, less func(a, b T) bool) {
    for i := 1; i < len(items); i++ {
        key := items[i]
        j := i - 1

        for j >= 0 && less(key, items[j]) {
            items[j+1] = items[j]
            j--
        }
        items[j+1] = key
    }
}

func quickSort[T any](items []T, low, high int, less func(a, b T) bool) {
    if low < high {
        pi := partition(items, low, high, less)
        quickSort(items, low, pi-1, less)
        quickSort(items, pi+1, high, less)
    }
}
```

### Memory Pooling

```go
type ObjectPool[T any] struct {
    pool       chan T
    factory    func() T
    reset      func(T)
    maxSize    int
}

func NewObjectPool[T any](factory func() T, reset func(T), maxSize int) *ObjectPool[T] {
    pool := &ObjectPool[T]{
        pool:    make(chan T, maxSize),
        factory: factory,
        reset:   reset,
        maxSize: maxSize,
    }

    // Pre-populate pool
    for i := 0; i < maxSize/2; i++ {
        pool.pool <- factory()
    }

    return pool
}

func (p *ObjectPool[T]) Get() T {
    select {
    case item := <-p.pool:
        return item
    default:
        return p.factory()
    }
}

func (p *ObjectPool[T]) Put(item T) {
    p.reset(item)
    select {
    case p.pool <- item:
        // Item returned to pool
    default:
        // Pool is full, item will be garbage collected
    }
}

// Usage for expensive objects
bufferPool := NewObjectPool(func() []byte {
    return make([]byte, 1024*1024) // 1MB buffer
}, func(buf []byte) {
    buf = buf[:0] // Reset buffer
}, 100)

func processData(data []byte) {
    buffer := bufferPool.Get()
    defer bufferPool.Put(buffer)

    // Use buffer for processing
    copy(buffer, data)
    // ... processing logic
}
```

## Streaming Processing

### Iterator Pattern

```go
type Iterator[T any] interface {
    Next() bool
    Value() T
    Close() error
}

type StreamIterator[T any] struct {
    items    []T
    position int
}

func NewStreamIterator[T any](items []T) *StreamIterator[T] {
    return &StreamIterator[T]{
        items:    items,
        position: 0,
    }
}

func (it *StreamIterator[T]) Next() bool {
    if it.position < len(it.items) {
        it.position++
        return true
    }
    return false
}

func (it *StreamIterator[T]) Value() T {
    if it.position <= 0 || it.position > len(it.items) {
        var zero T
        return zero
    }
    return it.items[it.position-1]
}

func (it *StreamIterator[T]) Close() error {
    it.items = nil
    it.position = 0
    return nil
}

// Streaming processing
func processStream[T, R any](iterator Iterator[T], processor func(T) (R, error)) ([]R, error) {
    var results []R

    for iterator.Next() {
        result, err := processor(iterator.Value())
        if err != nil {
            return results, err
        }
        results = append(results, result)
    }

    return results, nil
}
```

### Pipeline Processing

```go
type Pipeline[T, R any] struct {
    stages []func(Iterator[T]) Iterator[R]
}

func NewPipeline[T, R any]() *Pipeline[T, R] {
    return &Pipeline[T, R]{}
}

func (p *Pipeline[T, R]) AddStage(stage func(Iterator[T]) Iterator[R]) *Pipeline[T, R] {
    p.stages = append(p.stages, stage)
    return p
}

func (p *Pipeline[T, R]) Process(input Iterator[T]) Iterator[R] {
    var current Iterator[T] = input

    for _, stage := range p.stages {
        // Type assertion needed here in real implementation
        // This is simplified for the example
        current = stage(current).(Iterator[T])
    }

    return current.(Iterator[R])
}

// Usage
pipeline := NewPipeline[RawData, ProcessedData]().
    AddStage(func(it Iterator[RawData]) Iterator[ProcessedData] {
        return &MapIterator[RawData, ProcessedData]{
            source: it,
            mapper: func(data RawData) ProcessedData {
                return data.Process()
            },
        }
    }).
    AddStage(func(it Iterator[ProcessedData]) Iterator[ProcessedData] {
        return &FilterIterator[ProcessedData]{
            source: it,
            filter: func(data ProcessedData) bool {
                return data.IsValid()
            },
        }
    })
```

## Performance Monitoring

### Metrics Collection

```go
type PerformanceMetrics struct {
    Operations    int64
    Duration      time.Duration
    MemoryAllocated int64
    ItemsProcessed int64
}

type MonitoredList[T any] struct {
    inner   *collections.List[T]
    metrics *collections.ConcurrentMap[string, PerformanceMetrics]
}

func NewMonitoredList[T any](items []T) *MonitoredList[T] {
    return &MonitoredList[T]{
        inner:   collections.List(items),
        metrics: collections.NewConcurrentMap[string, PerformanceMetrics](),
    }
}

func (m *MonitoredList[T]) trackOperation(name string, fn func()) {
    start := time.Now()
    var memStatsBefore, memStatsAfter runtime.MemStats

    runtime.ReadMemStats(&memStatsBefore)
    fn()
    runtime.ReadMemStats(&memStatsAfter)

    metrics := PerformanceMetrics{
        Operations:    1,
        Duration:      time.Since(start),
        MemoryAllocated: int64(memStatsAfter.TotalAlloc - memStatsBefore.TotalAlloc),
    }

    existing := m.metrics.GetOrSet(name, func() PerformanceMetrics {
        return PerformanceMetrics{}
    })

    metrics.Operations += existing.Operations
    metrics.Duration += existing.Duration
    metrics.MemoryAllocated += existing.MemoryAllocated

    m.metrics.Set(name, metrics)
}

func (m *MonitoredList[T]) Filter(predicate func(T) bool) *collections.List[T] {
    var result *collections.List[T]
    m.trackOperation("filter", func() {
        result = m.inner.Filter(predicate)
    })
    return result
}

func (m *MonitoredList[T]) GetMetrics() map[string]PerformanceMetrics {
    result := make(map[string]PerformanceMetrics)
    m.metrics.Range(func(name string, metrics PerformanceMetrics) bool {
        result[name] = metrics
        return true
    })
    return result
}
```

### Performance Profiling

```go
func profileListOperations[T any](items []T, iterations int) {
    list := collections.List(items)

    // Profile Filter operation
    start := time.Now()
    for i := 0; i < iterations; i++ {
        list.Filter(func(item T) bool {
            // Simple condition
            return true
        })
    }
    filterDuration := time.Since(start)

    // Profile Map operation
    start = time.Now()
    for i := 0; i < iterations; i++ {
        list.Map(func(item T) T {
            return item
        })
    }
    mapDuration := time.Since(start)

    // Profile Sort operation
    start = time.Now()
    for i := 0; i < iterations; i++ {
        list.SortBy(func(item T) int {
            return 0
        })
    }
    sortDuration := time.Since(start)

    fmt.Printf("Filter: %v (%.2fns/op)\n", filterDuration, float64(filterDuration.Nanoseconds())/float64(iterations))
    fmt.Printf("Map: %v (%.2fns/op)\n", mapDuration, float64(mapDuration.Nanoseconds())/float64(iterations))
    fmt.Printf("Sort: %v (%.2fns/op)\n", sortDuration, float64(sortDuration.Nanoseconds())/float64(iterations))
}
```

By understanding and applying these performance optimization techniques, you can build high-performance applications that efficiently handle large datasets and complex processing requirements.