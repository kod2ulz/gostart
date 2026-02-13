package collections

import (
	"fmt"
	"sync"
)

// Graph represents a generic graph data structure
type Graph[K comparable, V any] struct {
	vertices map[K]*Vertex[K, V]
	directed bool
	mu       sync.RWMutex
}

// Vertex represents a vertex in the graph
type Vertex[K comparable, V any] struct {
	Key   K
	Value V
	edges map[K]*Edge[K]
}

// Edge represents an edge in the graph
type Edge[K comparable] struct {
	From   K
	To     K
	Weight float64
}

// NewGraph creates a new graph
func NewGraph[K comparable, V any](directed bool) *Graph[K, V] {
	return &Graph[K, V]{
		vertices: make(map[K]*Vertex[K, V]),
		directed: directed,
	}
}

// AddVertex adds a vertex to the graph
func (g *Graph[K, V]) AddVertex(key K, value V) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.vertices[key]; !exists {
		g.vertices[key] = &Vertex[K, V]{
			Key:   key,
			Value: value,
			edges: make(map[K]*Edge[K]),
		}
	} else {
		// Update value if vertex exists
		g.vertices[key].Value = value
	}
}

// RemoveVertex removes a vertex from the graph
func (g *Graph[K, V]) RemoveVertex(key K) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.vertices[key]; !exists {
		return false
	}

	// Remove all edges pointing to this vertex
	for _, vertex := range g.vertices {
		delete(vertex.edges, key)
	}

	// Remove the vertex itself
	delete(g.vertices, key)
	return true
}

// AddEdge adds an edge between two vertices
func (g *Graph[K, V]) AddEdge(from, to K, weight float64) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	fromVertex, fromExists := g.vertices[from]
	_, toExists := g.vertices[to]

	if !fromExists || !toExists {
		return fmt.Errorf("one or both vertices do not exist")
	}

	// Add edge from -> to
	fromVertex.edges[to] = &Edge[K]{
		From:   from,
		To:     to,
		Weight: weight,
	}

	// If undirected, add edge to -> from
	if !g.directed {
		toVertex := g.vertices[to]
		toVertex.edges[from] = &Edge[K]{
			From:   to,
			To:     from,
			Weight: weight,
		}
	}

	return nil
}

// RemoveEdge removes an edge between two vertices
func (g *Graph[K, V]) RemoveEdge(from, to K) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	fromVertex, exists := g.vertices[from]
	if !exists {
		return false
	}

	if _, edgeExists := fromVertex.edges[to]; !edgeExists {
		return false
	}

	delete(fromVertex.edges, to)

	// If undirected, remove reverse edge
	if !g.directed {
		if toVertex, exists := g.vertices[to]; exists {
			delete(toVertex.edges, from)
		}
	}

	return true
}

// GetVertex returns the value of a vertex
func (g *Graph[K, V]) GetVertex(key K) (V, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	vertex, exists := g.vertices[key]
	if !exists {
		var zero V
		return zero, false
	}
	return vertex.Value, true
}

// HasVertex checks if a vertex exists in the graph
func (g *Graph[K, V]) HasVertex(key K) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	_, exists := g.vertices[key]
	return exists
}

// HasEdge checks if an edge exists between two vertices
func (g *Graph[K, V]) HasEdge(from, to K) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	fromVertex, exists := g.vertices[from]
	if !exists {
		return false
	}

	_, edgeExists := fromVertex.edges[to]
	return edgeExists
}

// GetEdgeWeight returns the weight of an edge
func (g *Graph[K, V]) GetEdgeWeight(from, to K) (float64, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	fromVertex, exists := g.vertices[from]
	if !exists {
		return 0, false
	}

	edge, edgeExists := fromVertex.edges[to]
	if !edgeExists {
		return 0, false
	}

	return edge.Weight, true
}

// Neighbors returns the keys of all vertices adjacent to the given vertex
func (g *Graph[K, V]) Neighbors(key K) []K {
	g.mu.RLock()
	defer g.mu.RUnlock()

	vertex, exists := g.vertices[key]
	if !exists {
		return nil
	}

	neighbors := make([]K, 0, len(vertex.edges))
	for neighborKey := range vertex.edges {
		neighbors = append(neighbors, neighborKey)
	}

	return neighbors
}

// VertexCount returns the number of vertices in the graph
func (g *Graph[K, V]) VertexCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.vertices)
}

// EdgeCount returns the number of edges in the graph
func (g *Graph[K, V]) EdgeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	count := 0
	for _, vertex := range g.vertices {
		count += len(vertex.edges)
	}

	// If undirected, divide by 2 since each edge is counted twice
	if !g.directed {
		count = count / 2
	}

	return count
}

// BFS performs a breadth-first search from the start vertex
func (g *Graph[K, V]) BFS(start K, visit func(key K, value V)) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, exists := g.vertices[start]; !exists {
		return
	}

	visited := make(map[K]bool)
	queue := []K{start}
	visited[start] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		vertex := g.vertices[current]
		visit(vertex.Key, vertex.Value)

		for neighborKey := range vertex.edges {
			if !visited[neighborKey] {
				visited[neighborKey] = true
				queue = append(queue, neighborKey)
			}
		}
	}
}

// DFS performs a depth-first search from the start vertex
func (g *Graph[K, V]) DFS(start K, visit func(key K, value V)) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, exists := g.vertices[start]; !exists {
		return
	}

	visited := make(map[K]bool)
	g.dfsHelper(start, visited, visit)
}

// dfsHelper is a recursive helper for DFS
func (g *Graph[K, V]) dfsHelper(key K, visited map[K]bool, visit func(key K, value V)) {
	visited[key] = true
	vertex := g.vertices[key]
	visit(vertex.Key, vertex.Value)

	for neighborKey := range vertex.edges {
		if !visited[neighborKey] {
			g.dfsHelper(neighborKey, visited, visit)
		}
	}
}

// HasPath checks if there's a path between two vertices using BFS
func (g *Graph[K, V]) HasPath(from, to K) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, exists := g.vertices[from]; !exists {
		return false
	}
	if _, exists := g.vertices[to]; !exists {
		return false
	}

	if from == to {
		return true
	}

	visited := make(map[K]bool)
	queue := []K{from}
	visited[from] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == to {
			return true
		}

		vertex := g.vertices[current]
		for neighborKey := range vertex.edges {
			if !visited[neighborKey] {
				visited[neighborKey] = true
				queue = append(queue, neighborKey)
			}
		}
	}

	return false
}

// FindPath finds a path between two vertices using BFS
// Returns the path as a slice of keys, or nil if no path exists
func (g *Graph[K, V]) FindPath(from, to K) []K {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, exists := g.vertices[from]; !exists {
		return nil
	}
	if _, exists := g.vertices[to]; !exists {
		return nil
	}

	if from == to {
		return []K{from}
	}

	visited := make(map[K]bool)
	parent := make(map[K]K)
	queue := []K{from}
	visited[from] = true

	found := false
	for len(queue) > 0 && !found {
		current := queue[0]
		queue = queue[1:]

		vertex := g.vertices[current]
		for neighborKey := range vertex.edges {
			if !visited[neighborKey] {
				visited[neighborKey] = true
				parent[neighborKey] = current
				queue = append(queue, neighborKey)

				if neighborKey == to {
					found = true
					break
				}
			}
		}
	}

	if !found {
		return nil
	}

	// Reconstruct path
	path := []K{}
	current := to
	for current != from {
		path = append([]K{current}, path...)
		current = parent[current]
	}
	path = append([]K{from}, path...)

	return path
}

// IsAcyclic checks if the graph is acyclic (has no cycles)
// Only works for directed graphs
func (g *Graph[K, V]) IsAcyclic() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.directed {
		// For undirected graphs, check if it's a forest (no cycles)
		return g.isForest()
	}

	visited := make(map[K]bool)
	recStack := make(map[K]bool)

	for key := range g.vertices {
		if !visited[key] {
			if g.hasCycle(key, visited, recStack) {
				return false
			}
		}
	}

	return true
}

// hasCycle detects cycles in directed graphs using DFS
func (g *Graph[K, V]) hasCycle(key K, visited, recStack map[K]bool) bool {
	visited[key] = true
	recStack[key] = true

	vertex := g.vertices[key]
	for neighborKey := range vertex.edges {
		if !visited[neighborKey] {
			if g.hasCycle(neighborKey, visited, recStack) {
				return true
			}
		} else if recStack[neighborKey] {
			return true
		}
	}

	recStack[key] = false
	return false
}

// isForest checks if undirected graph is a forest (no cycles)
func (g *Graph[K, V]) isForest() bool {
	visited := make(map[K]bool)

	for key := range g.vertices {
		if !visited[key] {
			var defaultKey K
			if g.hasUndirectedCycle(key, defaultKey, visited) {
				return false
			}
		}
	}

	return true
}

// hasUndirectedCycle detects cycles in undirected graphs
func (g *Graph[K, V]) hasUndirectedCycle(key, parent K, visited map[K]bool) bool {
	visited[key] = true

	vertex := g.vertices[key]
	for neighborKey := range vertex.edges {
		if !visited[neighborKey] {
			if g.hasUndirectedCycle(neighborKey, key, visited) {
				return true
			}
		} else if neighborKey != parent {
			return true
		}
	}

	return false
}

// TopologicalSort returns a topological ordering of vertices
// Only works for directed acyclic graphs
func (g *Graph[K, V]) TopologicalSort() ([]K, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.directed {
		return nil, fmt.Errorf("topological sort only works on directed graphs")
	}

	if !g.IsAcyclic() {
		return nil, fmt.Errorf("graph contains cycles")
	}

	visited := make(map[K]bool)
	stack := []K{}

	for key := range g.vertices {
		if !visited[key] {
			g.topologicalSortHelper(key, visited, &stack)
		}
	}

	return stack, nil
}

// topologicalSortHelper is a recursive helper for topological sort
func (g *Graph[K, V]) topologicalSortHelper(key K, visited map[K]bool, stack *[]K) {
	visited[key] = true

	vertex := g.vertices[key]
	for neighborKey := range vertex.edges {
		if !visited[neighborKey] {
			g.topologicalSortHelper(neighborKey, visited, stack)
		}
	}

	*stack = append([]K{key}, *stack...)
}

// Clear removes all vertices and edges from the graph
func (g *Graph[K, V]) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.vertices = make(map[K]*Vertex[K, V])
}

// Example usage demonstrating graph functionality
func ExampleGraph() {
	// Create a directed graph
	graph := NewGraph[string, int](true)

	// Add vertices
	graph.AddVertex("A", 1)
	graph.AddVertex("B", 2)
	graph.AddVertex("C", 3)
	graph.AddVertex("D", 4)

	// Add edges
	graph.AddEdge("A", "B", 1.0)
	graph.AddEdge("A", "C", 2.0)
	graph.AddEdge("B", "D", 3.0)
	graph.AddEdge("C", "D", 4.0)

	// Find path from A to D
	path := graph.FindPath("A", "D")
	_ = path // ["A", "B", "D"] or ["A", "C", "D"]

	// BFS traversal
	graph.BFS("A", func(key string, value int) {
		// Process vertex
	})

	// DFS traversal
	graph.DFS("A", func(key string, value int) {
		// Process vertex
	})

	// Check if graph is acyclic
	isDAG := graph.IsAcyclic()
	_ = isDAG

	// Topological sort
	sorted, _ := graph.TopologicalSort()
	_ = sorted
}
