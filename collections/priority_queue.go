package collections

import (
	"container/heap"
	"sync"
)

// PriorityQueueItem represents an item in the priority queue
type PriorityQueueItem[T any] struct {
	Value    T
	Priority int
	index    int // index in the heap
}

// priorityQueueHeap implements heap.Interface for the priority queue
type priorityQueueHeap[T any] []*PriorityQueueItem[T]

func (pq priorityQueueHeap[T]) Len() int { return len(pq) }

func (pq priorityQueueHeap[T]) Less(i, j int) bool {
	// Higher priority value means higher priority (max heap)
	// To make it a min heap, reverse the comparison
	return pq[i].Priority > pq[j].Priority
}

func (pq priorityQueueHeap[T]) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueueHeap[T]) Push(x interface{}) {
	n := len(*pq)
	item := x.(*PriorityQueueItem[T])
	item.index = n
	*pq = append(*pq, item)
}

func (pq *priorityQueueHeap[T]) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

// PriorityQueue is a generic priority queue implementation
// Higher priority values are dequeued first
type PriorityQueue[T any] struct {
	heap priorityQueueHeap[T]
	mu   sync.RWMutex
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue[T any]() *PriorityQueue[T] {
	pq := &PriorityQueue[T]{
		heap: make(priorityQueueHeap[T], 0),
	}
	heap.Init(&pq.heap)
	return pq
}

// Push adds an item to the priority queue with the given priority
func (pq *PriorityQueue[T]) Push(value T, priority int) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	item := &PriorityQueueItem[T]{
		Value:    value,
		Priority: priority,
	}
	heap.Push(&pq.heap, item)
}

// Pop removes and returns the highest priority item from the queue
// Returns the zero value and false if the queue is empty
func (pq *PriorityQueue[T]) Pop() (T, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.heap.Len() == 0 {
		var zero T
		return zero, false
	}

	item := heap.Pop(&pq.heap).(*PriorityQueueItem[T])
	return item.Value, true
}

// Peek returns the highest priority item without removing it
// Returns the zero value and false if the queue is empty
func (pq *PriorityQueue[T]) Peek() (T, int, bool) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if pq.heap.Len() == 0 {
		var zero T
		return zero, 0, false
	}

	item := pq.heap[0]
	return item.Value, item.Priority, true
}

// Len returns the number of items in the queue
func (pq *PriorityQueue[T]) Len() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return pq.heap.Len()
}

// IsEmpty returns true if the queue is empty
func (pq *PriorityQueue[T]) IsEmpty() bool {
	return pq.Len() == 0
}

// Clear removes all items from the queue
func (pq *PriorityQueue[T]) Clear() {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.heap = make(priorityQueueHeap[T], 0)
	heap.Init(&pq.heap)
}

// UpdatePriority updates the priority of an item in the queue
// This is useful when you need to change priority of existing items
func (pq *PriorityQueue[T]) UpdatePriority(index int, newPriority int) bool {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if index < 0 || index >= pq.heap.Len() {
		return false
	}

	pq.heap[index].Priority = newPriority
	heap.Fix(&pq.heap, index)
	return true
}

// ToSlice returns all items in the queue as a slice (does not modify the queue)
// Items are returned in priority order (highest priority first)
func (pq *PriorityQueue[T]) ToSlice() []T {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	result := make([]T, 0, pq.heap.Len())

	// Create a copy to avoid modifying the original
	tempHeap := make(priorityQueueHeap[T], len(pq.heap))
	copy(tempHeap, pq.heap)

	for tempHeap.Len() > 0 {
		item := heap.Pop(&tempHeap).(*PriorityQueueItem[T])
		result = append(result, item.Value)
	}

	return result
}

// ForEach iterates over all items in the queue (in priority order) and applies the given function
// Does not modify the queue
func (pq *PriorityQueue[T]) ForEach(fn func(value T, priority int)) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	// Create a copy to avoid modifying the original
	tempHeap := make(priorityQueueHeap[T], len(pq.heap))
	copy(tempHeap, pq.heap)

	for tempHeap.Len() > 0 {
		item := heap.Pop(&tempHeap).(*PriorityQueueItem[T])
		fn(item.Value, item.Priority)
	}
}

// MinPriorityQueue is a priority queue where lower priority values are dequeued first
type MinPriorityQueue[T any] struct {
	heap minPriorityQueueHeap[T]
	mu   sync.RWMutex
}

// minPriorityQueueHeap implements heap.Interface for min heap
type minPriorityQueueHeap[T any] []*PriorityQueueItem[T]

func (pq minPriorityQueueHeap[T]) Len() int { return len(pq) }

func (pq minPriorityQueueHeap[T]) Less(i, j int) bool {
	// Lower priority value means higher priority (min heap)
	return pq[i].Priority < pq[j].Priority
}

func (pq minPriorityQueueHeap[T]) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *minPriorityQueueHeap[T]) Push(x interface{}) {
	n := len(*pq)
	item := x.(*PriorityQueueItem[T])
	item.index = n
	*pq = append(*pq, item)
}

func (pq *minPriorityQueueHeap[T]) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

// NewMinPriorityQueue creates a new min priority queue
// Lower priority values are dequeued first
func NewMinPriorityQueue[T any]() *MinPriorityQueue[T] {
	pq := &MinPriorityQueue[T]{
		heap: make(minPriorityQueueHeap[T], 0),
	}
	heap.Init(&pq.heap)
	return pq
}

// Push adds an item to the min priority queue with the given priority
func (pq *MinPriorityQueue[T]) Push(value T, priority int) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	item := &PriorityQueueItem[T]{
		Value:    value,
		Priority: priority,
	}
	heap.Push(&pq.heap, item)
}

// Pop removes and returns the lowest priority item from the queue
// Returns the zero value and false if the queue is empty
func (pq *MinPriorityQueue[T]) Pop() (T, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.heap.Len() == 0 {
		var zero T
		return zero, false
	}

	item := heap.Pop(&pq.heap).(*PriorityQueueItem[T])
	return item.Value, true
}

// Peek returns the lowest priority item without removing it
// Returns the zero value and false if the queue is empty
func (pq *MinPriorityQueue[T]) Peek() (T, int, bool) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if pq.heap.Len() == 0 {
		var zero T
		return zero, 0, false
	}

	item := pq.heap[0]
	return item.Value, item.Priority, true
}

// Len returns the number of items in the queue
func (pq *MinPriorityQueue[T]) Len() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return pq.heap.Len()
}

// IsEmpty returns true if the queue is empty
func (pq *MinPriorityQueue[T]) IsEmpty() bool {
	return pq.Len() == 0
}

// Clear removes all items from the queue
func (pq *MinPriorityQueue[T]) Clear() {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.heap = make(minPriorityQueueHeap[T], 0)
	heap.Init(&pq.heap)
}

// ToSlice returns all items in the queue as a slice (does not modify the queue)
// Items are returned in priority order (lowest priority first)
func (pq *MinPriorityQueue[T]) ToSlice() []T {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	result := make([]T, 0, pq.heap.Len())

	// Create a copy to avoid modifying the original
	tempHeap := make(minPriorityQueueHeap[T], len(pq.heap))
	copy(tempHeap, pq.heap)

	for tempHeap.Len() > 0 {
		item := heap.Pop(&tempHeap).(*PriorityQueueItem[T])
		result = append(result, item.Value)
	}

	return result
}

// Example usage demonstrating priority queue functionality
func ExamplePriorityQueue() {
	// Create a max priority queue (higher priority = dequeued first)
	maxPQ := NewPriorityQueue[string]()

	// Add tasks with priorities
	maxPQ.Push("Low priority task", 1)
	maxPQ.Push("High priority task", 10)
	maxPQ.Push("Medium priority task", 5)

	// Pop items - they come out in priority order
	for !maxPQ.IsEmpty() {
		task, _ := maxPQ.Pop()
		_ = task // "High priority task", "Medium priority task", "Low priority task"
	}

	// Create a min priority queue (lower priority = dequeued first)
	minPQ := NewMinPriorityQueue[string]()

	// Add tasks with priorities
	minPQ.Push("Task A", 5)
	minPQ.Push("Task B", 1)
	minPQ.Push("Task C", 10)

	// Pop items - they come out in reverse priority order
	for !minPQ.IsEmpty() {
		task, _ := minPQ.Pop()
		_ = task // "Task B", "Task A", "Task C"
	}
}
