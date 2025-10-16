# List[T] - Enhanced Slices with Fluent API

The `List[T]` type is the cornerstone of the collections package, providing a powerful, fluent interface for working with slices in Go.

## Core Philosophy

Traditional Go slice operations often involve nested loops, temporary variables, and verbose code. `List[T]` transforms this experience into a chainable, expressive API that tells a story.

```go
// Traditional Go approach
var activeCustomers []Customer
for _, customer := range customers {
    if customer.IsActive && customer.Age >= 18 {
        activeCustomers = append(activeCustomers, customer)
    }
}

var customerNames []string
for _, customer := range activeCustomers {
    customerNames = append(customerNames, customer.Name)
}

// Collections approach
customerNames := collections.List(customers).
    Filter(func(c Customer) bool { return c.IsActive && c.Age >= 18 }).
    Map(func(c Customer) string { return c.Name })
```

## Basic Operations

### Creating Lists

```go
// From existing slice
customers := collections.List([]Customer{{ID: 1, Name: "Alice"}})

// From variadic arguments
numbers := collections.List(1, 2, 3, 4, 5)

// Empty list
emptyList := collections.List[int]()
```

### Accessing Elements

```go
// Get elements by index
first := numbers.At(0)         // First element
last := numbers.At(-1)         // Last element
third := numbers.At(2)         // Third element

// Safe access with defaults
value := numbers.AtOrElse(10, 42)   // Returns 42 if index 10 doesn't exist

// Get slices
firstThree := numbers.Take(3)
lastTwo := numbers.TakeLast(2)
```

### Basic Querying

```go
// Check if list contains elements
hasValue := numbers.Contains(42)
hasActive := customers.ContainsFunc(func(c Customer) bool {
    return c.IsActive
})

// Find elements
firstActive := customers.Find(func(c Customer) bool {
    return c.IsActive
})

allActive := customers.Where(func(c Customer) bool {
    return c.IsActive
})

// Check conditions
allAdults := customers.Every(func(c Customer) bool {
    return c.Age >= 18
})

someAdults := customers.Some(func(c Customer) bool {
    return c.Age >= 18
})
```

## Transformation Operations

### Map and Transform

```go
// Simple mapping
names := customers.Map(func(c Customer) string {
    return c.Name
})

// Mapping with index
namesWithIndex := customers.MapIndexed(func(i int, c Customer) string {
    return fmt.Sprintf("%d. %s", i+1, c.Name)
})

// Complex transformations
customerSummaries := customers.Map(func(c Customer) CustomerSummary {
    return CustomerSummary{
        Name:    c.Name,
        Age:     c.Age,
        Status:  c.GetStatus(),
        Balance: c.CalculateBalance(),
    }
})
```

### Filtering Operations

```go
// Simple filtering
activeCustomers := customers.Filter(func(c Customer) bool {
    return c.IsActive
})

// Complex filtering with multiple conditions
premiumCustomers := customers.Filter(func(c Customer) bool {
    return c.IsActive &&
           c.MembershipLevel == "premium" &&
           c.LastPurchase.After(time.Now().AddDate(0, -6, 0))
})

// Filtering with type conversion
recentOrders := orders.Filter(func(o Order) bool {
    return o.Status == "completed" && o.CompletedAt.After(time.Now().AddDate(0, -1, 0))
})
```

### Sorting Operations

```go
// Sort by single field
customersByName := customers.SortBy(func(c Customer) string {
    return c.Name
})

// Sort by multiple fields
customersByAgeAndName := customers.SortBy(func(c Customer) int {
    return c.Age
}).ThenBy(func(c Customer) string {
    return c.Name
})

// Custom sorting logic
vipCustomersFirst := customers.Sort(func(a, b Customer) bool {
    if a.IsVIP && !b.IsVIP {
        return true
    }
    if !a.IsVIP && b.IsVIP {
        return false
    }
    return a.Name < b.Name
})
```

## Aggregation Operations

### Numeric Aggregations

```go
// Sum operations
totalRevenue := orders.Sum(func(o Order) float64 {
    return o.Amount
})

totalItems := orders.Sum(func(o Order) int {
    return o.ItemCount
})

// Average operations
avgOrderValue := orders.Average(func(o Order) float64 {
    return o.Amount
})

avgCustomerAge := customers.Average(func(c Customer) float64 {
    return float64(c.Age)
})

// Min/Max operations
oldestCustomer := customers.MinBy(func(c Customer) int {
    return c.Age
})

highestOrder := orders.MaxBy(func(o Order) float64 {
    return o.Amount
})
```

### Reduction Operations

```go
// Custom reduction
totalRevenueByRegion := orders.Reduce(map[string]float64{}, func(acc map[string]float64, order Order) map[string]float64 {
    acc[order.Region] += order.Amount
    return acc
})

// Count with conditions
activeCustomerCount := customers.Count(func(c Customer) bool {
    return c.IsActive
})

// Group and count
customersByRegion := customers.GroupBy(func(c Customer) string {
    return c.Region
}).Map(func(region string, customers []Customer) int {
    return len(customers)
})
```

### Grouping Operations

```go
// Simple grouping
customersByRegion := customers.GroupBy(func(c Customer) string {
    return c.Region
})

// Complex grouping with transformation
regionalStats := customers.GroupBy(func(c Customer) string {
    return c.Region
}).Map(func(region string, customers []Customer) RegionalStats {
    return RegionalStats{
        Region:       region,
        CustomerCount: len(customers),
        AvgAge:       collections.List(customers).Average(func(c Customer) float64 { return float64(c.Age) }),
        TotalRevenue: collections.List(customers).Sum(func(c Customer) float64 { return c.TotalSpent }),
    }
})

// Multi-level grouping
customersByRegionAndStatus := customers.GroupBy(func(c Customer) string {
    return fmt.Sprintf("%s_%s", c.Region, c.Status)
})
```

## Advanced Operations

### FlatMap and Flatten

```go
// Working with nested structures
orders := customers.FlatMap(func(c Customer) []Order {
    return c.Orders
})

// Flatten nested lists
allItems := orders.FlatMap(func(o Order) []OrderItem {
    return o.Items
})

// Complex transformations
productRecommendations := customers.FlatMap(func(c Customer) []Product {
    return c.GetRecommendedProducts()
}).Filter(func(p Product) bool {
    return p.InStock && p.Price > 0
})
```

### Partitioning

```go
// Split list based on condition
active, inactive := customers.Partition(func(c Customer) bool {
    return c.IsActive
})

// Multi-way partitioning
byMembershipLevel := customers.PartitionBy(func(c Customer) string {
    return c.MembershipLevel
})

// Returns: map[string][]Customer
// {
//     "basic": [...],
//     "premium": [...],
//     "vip": [...]
// }
```

### Chunking and Batching

```go
// Process large lists in chunks
batches := customers.Chunk(100) // Returns [][]Customer, each with max 100 customers

// Parallel processing
results := make([]Result, 0)
for i, batch := range batches {
    batchResult := processBatch(batch, i)
    results = append(results, batchResult...)
}

// Sliding windows
windows := numbers.SlidingWindow(3) // [[1,2,3], [2,3,4], [3,4,5], ...]
```

## Performance Considerations

### Lazy Evaluation

Most `List[T]` operations are lazy, meaning they create views rather than new lists:

```go
// This creates a pipeline, not intermediate lists
result := hugeList.
    Filter(func(item Item) bool { return item.IsActive }).
    Map(func(item Item) Output { return transform(item) }).
    Take(100) // Only processes the first 100 items!
```

### Memory Efficiency

```go
// Efficient chaining - no intermediate allocations
pipeline := collections.List(data).
    Filter(func(d Data) bool { return d.IsValid }).
    Map(func(d Data) Processed { return d.Process() }).
    SortBy(func(p Processed) int { return p.Priority })

// To realize the results, call a terminal operation
results := pipeline.ToSlice()
```

### Parallel Processing

```go
// Process items in parallel
results := customers.ParallelMap(func(c Customer) CustomerSummary {
    return processCustomer(c)
}, runtime.NumCPU())

// Parallel filtering
activeCustomers := customers.ParallelFilter(func(c Customer) bool {
    return c.IsActive && c.Age >= 18
}, 4)
```

## Real-World Examples

### E-commerce Analytics

```go
// Customer segmentation and analytics
customerSegments := collections.List(customers).
    Filter(func(c Customer) bool {
        return c.IsActive && c.TotalOrders > 0
    }).
    GroupBy(func(c Customer) string {
        switch {
        case c.TotalSpent > 10000:
            return "vip"
        case c.TotalSpent > 5000:
            return "premium"
        case c.TotalSpent > 1000:
            return "regular"
        default:
            return "basic"
        }
    }).
    Map(func(segment string, customers []Customer) SegmentReport {
        segmentCustomers := collections.List(customers)
        return SegmentReport{
            Segment:      segment,
            CustomerCount: len(customers),
            AvgOrderValue: segmentCustomers.Average(func(c Customer) float64 {
                return c.TotalSpent / float64(c.TotalOrders)
            }),
            TotalRevenue:  segmentCustomers.Sum(func(c Customer) float64 { return c.TotalSpent }),
            RetentionRate: segmentCustomers.Count(func(c Customer) bool {
                return c.LastOrder.After(time.Now().AddDate(0, -3, 0))
            }) / float64(len(customers)),
        }
    })
```

### Data Processing Pipeline

```go
// Clean and process transaction data
processedTransactions := collections.List(rawTransactions).
    Filter(func(t Transaction) bool {
        return t.Amount > 0 && t.Status != "failed"
    }).
    Map(func(t Transaction) ProcessedTransaction {
        return ProcessedTransaction{
            ID:          t.ID,
            Amount:      t.Amount,
            Category:    categorizeTransaction(t),
            Date:        t.Date,
            CustomerID:  t.CustomerID,
            ProcessedAt: time.Now(),
        }
    }).
    SortBy(func(pt ProcessedTransaction) time.Time {
        return pt.Date
    }).
    Chunk(1000) // Process in batches
```

### Search and Filtering

```go
// Advanced search functionality
searchResults := collections.List(products).
    Filter(func(p Product) bool {
        return p.IsActive && p.InStock
    }).
    Filter(func(p Product) bool {
        // Text search
        searchTerm := strings.ToLower(query)
        nameMatch := strings.Contains(strings.ToLower(p.Name), searchTerm)
        descMatch := strings.Contains(strings.ToLower(p.Description), searchTerm)
        return nameMatch || descMatch
    }).
    Filter(func(p Product) bool {
        // Price range
        return p.Price >= minPrice && p.Price <= maxPrice
    }).
    Filter(func(p Product) bool {
        // Category filter
        if len(categories) == 0 {
            return true
        }
        return collections.List(categories).Contains(p.Category)
    }).
    SortBy(func(p Product) float64 {
        // Sort by relevance (you could implement a scoring function)
        return calculateRelevanceScore(p, query)
    }).
    Take(limit)
```

## Error Handling

```go
// Safe operations with error handling
results, err := collections.List(data).
    Map(func(d Data) (Output, error) {
        result, err := processData(d)
        if err != nil {
            return Output{}, fmt.Errorf("failed to process data %v: %w", d.ID, err)
        }
        return result, nil
    }).
    FilterErrors() // Filters out errored items

// Error aggregation
allResults, errors := collections.List(data).
    SafeMap(func(d Data) (Output, error) {
        return processData(d)
    }).
    Partition(func(result SafeResult[Output]) bool {
        return result.Error == nil
    })

if len(errors) > 0 {
    log.Printf("Processing completed with %d errors", len(errors))
}
```

The `List[T]` type provides a powerful, expressive, and efficient way to work with collections in Go, making complex data operations readable and maintainable.