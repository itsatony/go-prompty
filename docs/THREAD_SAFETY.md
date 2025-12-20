# Thread Safety Guide

This guide explains how to safely use go-prompty in concurrent applications, covering the "Parse Once, Execute Many" pattern, safe template sharing, and cache implementation strategies.

## Overview

go-prompty is designed to be thread-safe by default. All exported types can be safely accessed from multiple goroutines. However, understanding the concurrency model helps you write efficient code that avoids common pitfalls.

## Key Principles

### 1. Engine is Thread-Safe

The `Engine` type is safe for concurrent use. You should create one engine and share it across all goroutines:

```go
package main

import (
    "context"
    "sync"

    "github.com/itsatony/go-prompty"
)

func main() {
    // Create ONE engine for your application
    engine := prompty.MustNew()

    // Register templates once at startup
    engine.MustRegisterTemplate("greeting", "Hello, {~prompty.var name=\"name\" /~}!")
    engine.MustRegisterTemplate("farewell", "Goodbye, {~prompty.var name=\"name\" /~}!")

    // Use the same engine from multiple goroutines
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()

            data := map[string]any{"name": fmt.Sprintf("User%d", id)}
            result, _ := engine.Execute(context.Background(),
                "{~prompty.include template=\"greeting\" /~}", data)
            fmt.Println(result)
        }(i)
    }
    wg.Wait()
}
```

### 2. Parse Once, Execute Many

Templates are parsed into an AST (Abstract Syntax Tree) once and can be executed many times with different data. This is the recommended pattern for production use:

```go
package main

import (
    "context"
    "sync"

    "github.com/itsatony/go-prompty"
)

func main() {
    engine := prompty.MustNew()

    // Parse the template ONCE
    template, err := engine.Parse("Hello, {~prompty.var name=\"username\" /~}!")
    if err != nil {
        panic(err)
    }

    // Execute MANY times with different data
    var wg sync.WaitGroup
    for i := 0; i < 1000; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()

            // Each goroutine provides its own data
            data := prompty.NewContext(map[string]any{
                "username": fmt.Sprintf("user_%d", id),
            })

            // ExecuteTemplate is thread-safe
            result, _ := engine.ExecuteTemplate(context.Background(), template, data)
            _ = result
        }(i)
    }
    wg.Wait()
}
```

### 3. Context is Thread-Safe (But Create New Per Request)

While `Context` has internal locking for thread safety, the recommended pattern is to create a new context for each request. This avoids contention and makes data flow clearer:

```go
// RECOMMENDED: Create new context per request
func handleRequest(engine *prompty.Engine, userData map[string]any) string {
    ctx := prompty.NewContext(userData)
    result, _ := engine.Execute(context.Background(), template, ctx)
    return result
}

// AVOID: Sharing context across goroutines and mutating it
// This works but creates unnecessary contention
```

## Template Registration

### Safe Registration at Startup

Register all templates during application initialization, before starting concurrent operations:

```go
func initEngine() *prompty.Engine {
    engine := prompty.MustNew()

    // Register all templates at startup
    templates := map[string]string{
        "header":   "Welcome to {~prompty.var name=\"siteName\" /~}",
        "footer":   "Copyright 2024 {~prompty.var name=\"company\" /~}",
        "greeting": "Hello, {~prompty.var name=\"name\" /~}!",
    }

    for name, source := range templates {
        engine.MustRegisterTemplate(name, source)
    }

    return engine
}

// Use the initialized engine throughout your application
var appEngine = initEngine()
```

### Dynamic Registration (With Care)

If you need to register templates at runtime, use the non-panicking `RegisterTemplate` method and handle errors:

```go
func addTemplate(engine *prompty.Engine, name, source string) error {
    // RegisterTemplate is thread-safe but follows first-come-wins semantics
    // Duplicate registrations return an error
    return engine.RegisterTemplate(name, source)
}
```

## Caching Strategies

### In-Memory Template Cache

For applications that load templates from a database or filesystem, implement a cache layer:

```go
package main

import (
    "context"
    "sync"
    "time"

    "github.com/itsatony/go-prompty"
)

// TemplateCache provides thread-safe caching of parsed templates
type TemplateCache struct {
    engine   *prompty.Engine
    cache    map[string]*cacheEntry
    mu       sync.RWMutex
    ttl      time.Duration
    loader   TemplateLoader
}

type cacheEntry struct {
    template  *prompty.Template
    loadedAt  time.Time
}

type TemplateLoader func(name string) (string, error)

func NewTemplateCache(engine *prompty.Engine, loader TemplateLoader, ttl time.Duration) *TemplateCache {
    return &TemplateCache{
        engine: engine,
        cache:  make(map[string]*cacheEntry),
        ttl:    ttl,
        loader: loader,
    }
}

func (tc *TemplateCache) Get(name string) (*prompty.Template, error) {
    // Fast path: check cache with read lock
    tc.mu.RLock()
    entry, exists := tc.cache[name]
    tc.mu.RUnlock()

    if exists && time.Since(entry.loadedAt) < tc.ttl {
        return entry.template, nil
    }

    // Slow path: load and parse template
    tc.mu.Lock()
    defer tc.mu.Unlock()

    // Double-check after acquiring write lock
    entry, exists = tc.cache[name]
    if exists && time.Since(entry.loadedAt) < tc.ttl {
        return entry.template, nil
    }

    // Load from source
    source, err := tc.loader(name)
    if err != nil {
        return nil, err
    }

    // Parse and cache
    template, err := tc.engine.Parse(source)
    if err != nil {
        return nil, err
    }

    tc.cache[name] = &cacheEntry{
        template: template,
        loadedAt: time.Now(),
    }

    return template, nil
}

func (tc *TemplateCache) Execute(ctx context.Context, name string, data map[string]any) (string, error) {
    template, err := tc.Get(name)
    if err != nil {
        return "", err
    }

    execCtx := prompty.NewContext(data)
    return tc.engine.ExecuteTemplate(ctx, template, execCtx)
}

func (tc *TemplateCache) Invalidate(name string) {
    tc.mu.Lock()
    delete(tc.cache, name)
    tc.mu.Unlock()
}

func (tc *TemplateCache) Clear() {
    tc.mu.Lock()
    tc.cache = make(map[string]*cacheEntry)
    tc.mu.Unlock()
}
```

### Usage Example

```go
func main() {
    engine := prompty.MustNew()

    // Create a loader that fetches templates from a database
    loader := func(name string) (string, error) {
        return db.GetTemplate(name) // Your database call
    }

    // Create cache with 5-minute TTL
    cache := NewTemplateCache(engine, loader, 5*time.Minute)

    // Use from multiple goroutines
    http.HandleFunc("/render", func(w http.ResponseWriter, r *http.Request) {
        templateName := r.URL.Query().Get("template")
        data := map[string]any{"user": r.Context().Value("user")}

        result, err := cache.Execute(r.Context(), templateName, data)
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }

        w.Write([]byte(result))
    })
}
```

## Best Practices

### Do

1. **Create one Engine per application** - Engines are designed to be shared
2. **Register templates at startup** - Avoid registration during request handling
3. **Create new Context per request** - This is the cleanest pattern for data isolation
4. **Use Parse + ExecuteTemplate for hot paths** - Pre-parse templates for better performance
5. **Implement caching for dynamic templates** - If templates come from external sources

### Don't

1. **Don't create a new Engine per request** - This wastes resources
2. **Don't share Context data between requests** - Each request should have its own data
3. **Don't modify template source after registration** - Templates are immutable once registered
4. **Don't rely on Context mutation across goroutines** - While safe, it's error-prone

## Performance Considerations

### Parsing Cost

Parsing templates has a cost. For templates used repeatedly, always use the Parse Once, Execute Many pattern:

```go
// GOOD: Parse once
template, _ := engine.Parse(complexTemplate)
for _, user := range users {
    result, _ := engine.ExecuteTemplate(ctx, template, prompty.NewContext(user))
    // ...
}

// BAD: Parse every time
for _, user := range users {
    result, _ := engine.Execute(ctx, complexTemplate, user)
    // ...
}
```

### Context Creation

`NewContext` creates a new context with the provided data. For high-performance scenarios, consider reusing context structures:

```go
// For bulk processing, reuse the data map structure
dataTemplate := map[string]any{
    "name": "",
    "email": "",
    "role": "",
}

for _, user := range users {
    dataTemplate["name"] = user.Name
    dataTemplate["email"] = user.Email
    dataTemplate["role"] = user.Role

    result, _ := engine.ExecuteTemplate(ctx, template, prompty.NewContext(dataTemplate))
    // ...
}
```

### Resolver Isolation

Custom resolvers run in isolated goroutines with timeout protection. This means:

1. **Resolver panics don't crash your application** - They're recovered and converted to errors
2. **Slow resolvers don't block forever** - Configurable timeout per resolver
3. **Context cancellation is respected** - Resolvers can be cancelled

```go
// Configure timeouts via engine options
engine, _ := prompty.New(
    prompty.WithResolverTimeout(5 * time.Second),  // Per-resolver timeout
    prompty.WithMaxExecutionTime(30 * time.Second), // Overall timeout
)
```

## Common Patterns

### HTTP Handler with Template

```go
func NewTemplateHandler(engine *prompty.Engine, templateName string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Extract data from request (user, params, etc.)
        data := map[string]any{
            "user":      r.Context().Value("user"),
            "path":      r.URL.Path,
            "method":    r.Method,
            "timestamp": time.Now().Format(time.RFC3339),
        }

        // Execute with request context for cancellation support
        result, err := engine.Execute(r.Context(),
            `{~prompty.include template="`+templateName+`" /~}`, data)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "text/plain")
        w.Write([]byte(result))
    }
}
```

### Worker Pool Processing

```go
func ProcessBatch(engine *prompty.Engine, template *prompty.Template, items []Item) []string {
    results := make([]string, len(items))

    var wg sync.WaitGroup
    sem := make(chan struct{}, 10) // Limit concurrent workers

    for i, item := range items {
        wg.Add(1)
        go func(idx int, it Item) {
            defer wg.Done()

            sem <- struct{}{}        // Acquire
            defer func() { <-sem }() // Release

            ctx := prompty.NewContext(map[string]any{
                "item": it,
                "index": idx,
            })

            result, _ := engine.ExecuteTemplate(context.Background(), template, ctx)
            results[idx] = result
        }(i, item)
    }

    wg.Wait()
    return results
}
```

## Testing Concurrent Code

When testing code that uses go-prompty concurrently, use the race detector:

```bash
go test -race ./...
```

Example concurrent test:

```go
func TestConcurrentExecution(t *testing.T) {
    engine := prompty.MustNew()
    template, _ := engine.Parse("Hello, {~prompty.var name=\"name\" /~}!")

    var wg sync.WaitGroup
    errors := make(chan error, 100)

    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()

            ctx := prompty.NewContext(map[string]any{
                "name": fmt.Sprintf("User%d", id),
            })

            result, err := engine.ExecuteTemplate(context.Background(), template, ctx)
            if err != nil {
                errors <- err
                return
            }

            expected := fmt.Sprintf("Hello, User%d!", id)
            if result != expected {
                errors <- fmt.Errorf("got %q, want %q", result, expected)
            }
        }(i)
    }

    wg.Wait()
    close(errors)

    for err := range errors {
        t.Error(err)
    }
}
```

## Summary

- **Engine**: Thread-safe, share across goroutines
- **Template**: Immutable after parsing, safe to share
- **Context**: Thread-safe but create new per request for clarity
- **Registry**: Thread-safe, register at startup
- **Resolvers**: Isolated execution with timeout protection

Following these patterns ensures your go-prompty usage is both safe and efficient in concurrent applications.
