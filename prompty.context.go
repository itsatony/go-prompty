package prompty

import (
	"strings"
	"sync"
)

// Context provides access to template variables and execution state.
// It supports dot-notation path resolution (e.g., "user.profile.name")
// and hierarchical scoping through parent-child relationships.
type Context struct {
	data       map[string]any
	parent     *Context
	mu         sync.RWMutex
	errorStrat ErrorStrategy
}

// NewContext creates a new execution context with the given data.
// If data is nil, an empty map is used.
func NewContext(data map[string]any) *Context {
	if data == nil {
		data = make(map[string]any)
	}
	return &Context{
		data:       data,
		errorStrat: ErrorStrategyThrow,
	}
}

// NewContextWithStrategy creates a context with a specific error strategy.
func NewContextWithStrategy(data map[string]any, strategy ErrorStrategy) *Context {
	ctx := NewContext(data)
	ctx.errorStrat = strategy
	return ctx
}

// Get retrieves a value by dot-notation path (e.g., "user.profile.name").
// Returns the value and true if found, or nil and false if not found.
func (c *Context) Get(path string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.getPath(path)
}

// getPath resolves a dot-notation path without locking (internal use).
func (c *Context) getPath(path string) (any, bool) {
	if path == "" {
		return nil, false
	}

	parts := strings.Split(path, PathSeparator)
	var current any = c.data

	for _, part := range parts {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]any:
			val, ok := v[part]
			if !ok {
				// Try parent context if not found
				if c.parent != nil {
					return c.parent.getPath(path)
				}
				return nil, false
			}
			current = val
		case map[string]string:
			val, ok := v[part]
			if !ok {
				if c.parent != nil {
					return c.parent.getPath(path)
				}
				return nil, false
			}
			current = val
		default:
			// Can't traverse further
			if c.parent != nil {
				return c.parent.getPath(path)
			}
			return nil, false
		}
	}

	return current, true
}

// GetString retrieves a string value by path.
// Returns empty string if not found or not a string.
func (c *Context) GetString(path string) string {
	val, ok := c.Get(path)
	if !ok {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

// GetDefault retrieves a value by path with a fallback default.
func (c *Context) GetDefault(path string, defaultVal any) any {
	val, ok := c.Get(path)
	if !ok {
		return defaultVal
	}
	return val
}

// GetStringDefault retrieves a string value with a fallback default.
func (c *Context) GetStringDefault(path, defaultVal string) string {
	val, ok := c.Get(path)
	if !ok {
		return defaultVal
	}
	if s, ok := val.(string); ok {
		return s
	}
	return defaultVal
}

// Set sets a value at the given path.
// Currently only supports simple keys (not nested paths).
func (c *Context) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = value
}

// Has checks if a value exists at the given path.
func (c *Context) Has(path string) bool {
	_, ok := c.Get(path)
	return ok
}

// Child creates a child context with additional data.
// The child inherits from the parent and can override values.
func (c *Context) Child(data map[string]any) *Context {
	if data == nil {
		data = make(map[string]any)
	}
	return &Context{
		data:       data,
		parent:     c,
		errorStrat: c.errorStrat,
	}
}

// Parent returns the parent context, or nil if this is a root context.
func (c *Context) Parent() *Context {
	return c.parent
}

// Data returns a copy of the context's direct data (not including parent).
func (c *Context) Data() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]any, len(c.data))
	for k, v := range c.data {
		result[k] = v
	}
	return result
}

// ErrorStrategy returns the current error handling strategy.
func (c *Context) ErrorStrategy() ErrorStrategy {
	return c.errorStrat
}

// Path separator for dot-notation
const PathSeparator = "."
