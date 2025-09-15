# Collections (`collections`)

This package provides a set of generic, thread-safe, and feature-rich data structures designed to improve upon the built-in Go types.

## Overview

The collections are designed with a fluent, chainable API, making complex data manipulations more expressive and readable.

- **`List[T]`**: A generic wrapper for a slice (`[]T`) with a rich set of utility methods.
- **`Map[K, T]`**: A generic wrapper for a map (`map[K]T`) with helpful extensions.
- **`Set[T]`**: A generic implementation of a Set data structure, built on top of `Map`.
- **`Tree[ID, T]`**: A powerful tool for converting a flat list of items into a hierarchical tree structure.
- **`ConcurrentMap[K, T]`**: A thread-safe implementation of the `Map`.
- **`ConcurrentList[T]`**: A thread-safe implementation of the `List`.

---

## `List[T]`

A `List` is an enhanced slice.

### Example

```go
import "github.com/kod2ulz/gostart/collections"

// Create a new list
myList := collections.List[int]{1, 2, 3, 4, 5}

// Add items
myList.Add(6, 7)

// Filter for even numbers
evens := myList.Filter(func(i int, val int) bool {
    return val%2 == 0
}) // Result: [2, 4, 6]

// Get the first item
first := evens.First() // Result: 2

// Sort descending
sorted := myList.Sort(func(a, b int) bool {
    return a > b
}) // Result: [7, 6, 5, 4, 3, 2, 1]
```

---

## `Map[K, T]`

A `Map` is an enhanced map.

### Example

```go
// Create a new map
myMap := collections.Map[string, string]{"a": "apple", "b": "banana"}

// Add an item
myMap.Add("c", "cherry")

// Get all keys
keys := myMap.Keys() // Result: ["a", "b", "c"]

// Get all values
values := myMap.Values() // Result: ["apple", "banana", "cherry"]

// Get a single value (returns a pointer)
val := myMap.Get("b") // *val is "banana"
```

---

## `Set[T]`

A `Set` stores unique, comparable values.

### Example

```go
mySet := collections.Set[string]{}
mySet.Add("apple")
mySet.Add("banana")
mySet.Add("apple") // Duplicate is ignored

// Check for existence
hasApple := mySet.Has("apple") // true
hasCherry := mySet.Has("cherry") // false

// Get all unique values
values := mySet.Values() // ["apple", "banana"] (order not guaranteed)

// Remove an item
mySet.Remove("apple")

// Union: a new set with all items from both
set1 := collections.Set[string]{"a": {}, "b": {}}
set2 := collections.Set[string]{"b": {}, "c": {}}
union := set1.Union(set2) // Contains "a", "b", "c"

// Intersection: a new set with only common items
intersection := set1.Intersection(set2) // Contains just "b"

// Difference: a new set with items in the first set but not the second
difference := set1.Difference(set2) // Contains just "a"
```

---

## `Tree[ID, T]`

The `Tree` is used to build a hierarchy from a flat slice of objects. Each object must implement the `TreeDataInterface`.

For improved performance and control over JSON serialization, your data structure can optionally implement the `TreeDataMapper` interface.

### Tree Methods

- **`Flatten()`**: Returns a flat `List[T]` of all nodes in the tree.
- **`Walk(fn func(node *TreeNode[ID, T]))`**: Traverses the tree in breadth-first order and applies a function to each node.

### JSON Marshaling

The `TreeNode` type implements the `json.Marshaler` interface, so you can serialize a tree directly to JSON. The output will be a nested structure representing the hierarchy.

### Example

```go
import (
    "encoding/json"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/kod2ulz/gostart/collections"
)

// 1. Define your data structure
type Category struct {
    ID       int     `json:"id"`
    Name     string  `json:"name"`
    ParentID *int    `json:"parentId,omitempty"`
    CreatedAt time.Time `json:"createdAt"`
}

// 2. Implement the TreeDataInterface
func (c Category) Identifier() int {
    return c.ID
}
func (c Category) ParentIdentifier() *int {
    return c.ParentID
}

// 3. Optionally implement the TreeDataMapper interface for custom JSON output
func (c Category) ToMap() map[string]any {
    return map[string]any{
        "id": c.ID,
        "name": c.Name,
        "parent_id": c.ParentID,
        "created_at": c.CreatedAt.Format(time.RFC3339),
        "_custom_field": "example", // Add custom fields
    }
}

// 4. Your flat data
data := []Category{
    {ID: 1, Name: "Electronics", CreatedAt: time.Now()},
    {ID: 2, Name: "Computers", ParentID: collections.AsPointer(1), CreatedAt: time.Now()},
    {ID: 3, Name: "Laptops", ParentID: collections.AsPointer(2), CreatedAt: time.Now()},
    {ID: 4, Name: "Phones", ParentID: collections.AsPointer(1), CreatedAt: time.Now()},
}

// 5. Build the tree
tree := collections.TreeOf(data...)

// 6. Flatten the tree
flatList := tree.Flatten()
fmt.Printf("Found %d items in the flattened list\n", flatList.Size())

// 7. Walk the tree
var count int
tree.Walk(func(node *collections.TreeNode[int, Category]) {
	count++
})
fmt.Printf("Walked %d nodes\n", count)

// 8. Serialize to JSON
jsonBytes, _ := json.MarshalIndent(tree, "", "  ")
fmt.Println(string(jsonBytes))
/* Example Output (structure will vary based on actual data and ToMap implementation):
{
  "id": 1,
  "name": "Electronics",
  "parent_id": null,
  "created_at": "2023-10-27T10:00:00Z",
  "_custom_field": "example",
  "children": [
    {
      "id": 2,
      "name": "Computers",
      "parent_id": 1,
      "created_at": "2023-10-27T10:00:00Z",
      "_custom_field": "example",
      "children": [
        {
          "id": 3,
          "name": "Laptops",
          "parent_id": 2,
          "created_at": "2023-10-27T10:00:00Z",
          "_custom_field": "example"
        }
      ]
    },
    {
      "id": 4,
      "name": "Phones",
      "parent_id": 1,
      "created_at": "2023-10-27T10:00:00Z",
      "_custom_field": "example"
    }
  ]
}
*/
```

---

## Concurrent Collections

For use in concurrent applications, the package provides thread-safe versions of `Map` and `List`.

- **`ConcurrentMap[K, T]`**: A map where read and write operations are protected by a mutex.
- **`ConcurrentList[T]`**: A list where read and write operations are protected by a mutex.

These can be used as direct, thread-safe replacements for their non-concurrent counterparts.
