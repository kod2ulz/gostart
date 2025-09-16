package app

// Legacy types for backward compatibility
// Context is deprecated - use RequestContext instead
type Context interface {
	Get(key string) (value any, exists bool)
}

// HandlerFunc is deprecated - use app.HandlerFunc instead
type HandlerFunc func(Context)

// RouterEngine is deprecated - use Router instead
type RouterEngine[ctx Context] interface {
	// Empty interface for backward compatibility
}