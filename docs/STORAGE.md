# Storage Layer Guide

go-prompty provides a pluggable storage layer for managing templates with versioning, metadata, and multi-tenant support. The storage layer follows a driver-based architecture similar to Go's `database/sql` package.

## Overview

The storage layer consists of:

- **TemplateStorage Interface**: Abstract interface for storage backends
- **StoredTemplate**: Template with metadata, versioning, and tenant support
- **Built-in Drivers**: Memory and filesystem implementations
- **CachedStorage**: Caching wrapper for any storage backend
- **StorageEngine**: Combines storage with template execution

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/itsatony/go-prompty"
)

func main() {
    ctx := context.Background()

    // Create storage engine with memory storage
    se, err := prompty.NewStorageEngine(prompty.StorageEngineConfig{
        Storage: prompty.NewMemoryStorage(),
    })
    if err != nil {
        panic(err)
    }
    defer se.Close()

    // Save a template
    err = se.Save(ctx, &prompty.StoredTemplate{
        Name:   "greeting",
        Source: "Hello, {~prompty.var name=\"user\" default=\"World\" /~}!",
        Tags:   []string{"public"},
    })
    if err != nil {
        panic(err)
    }

    // Execute the template
    result, err := se.Execute(ctx, "greeting", map[string]any{
        "user": "Alice",
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(result) // Output: Hello, Alice!
}
```

## Storage Interface

The `TemplateStorage` interface defines the contract for storage backends:

```go
type TemplateStorage interface {
    // Retrieval
    Get(ctx context.Context, name string) (*StoredTemplate, error)
    GetByID(ctx context.Context, id TemplateID) (*StoredTemplate, error)
    GetVersion(ctx context.Context, name string, version int) (*StoredTemplate, error)

    // Modification
    Save(ctx context.Context, tmpl *StoredTemplate) error
    Delete(ctx context.Context, name string) error
    DeleteVersion(ctx context.Context, name string, version int) error

    // Query
    List(ctx context.Context, query *TemplateQuery) ([]*StoredTemplate, error)
    Exists(ctx context.Context, name string) (bool, error)
    ListVersions(ctx context.Context, name string) ([]int, error)

    // Lifecycle
    Close() error
}
```

## StoredTemplate

Templates are stored with rich metadata:

```go
type StoredTemplate struct {
    ID          TemplateID         // Unique identifier (auto-generated)
    Name        string             // Human-readable name
    Source      string             // Template source code
    Version     int                // Auto-incremented version number
    Metadata    map[string]string  // Custom key-value metadata
    CreatedAt   time.Time          // Creation timestamp
    UpdatedAt   time.Time          // Last update timestamp
    CreatedBy   string             // Creator identifier
    TenantID    string             // Multi-tenant organization ID
    Tags        []string           // Categorization tags
}
```

## Built-in Drivers

### Memory Storage

In-memory storage for testing and development:

```go
storage := prompty.NewMemoryStorage()
```

Features:
- Fast, no persistence
- Thread-safe
- Supports all storage operations
- Ideal for testing

### Filesystem Storage

File-based storage with JSON serialization:

```go
storage, err := prompty.NewFilesystemStorage("/path/to/templates")
```

Features:
- Persistent storage
- Git-friendly format
- Directory structure: `<root>/<name>/v<N>.json`
- Thread-safe
- Automatic directory creation

Directory structure example:
```
/templates/
  greeting/
    v1.json
    v2.json
  farewell/
    v1.json
```

### Opening Storage by Driver Name

Storage can be opened using registered driver names:

```go
// List available drivers
drivers := prompty.ListStorageDrivers()
fmt.Println(drivers) // ["memory", "filesystem"]

// Open storage by driver name
storage, err := prompty.OpenStorage("filesystem", "/path/to/templates")
```

## Caching

Wrap any storage with caching for improved performance:

```go
storage := prompty.NewMemoryStorage()
cached := prompty.NewCachedStorage(storage, prompty.CacheConfig{
    TTL:              5 * time.Minute,   // How long entries stay valid
    MaxEntries:       1000,              // Maximum cached templates
    NegativeCacheTTL: 30 * time.Second,  // Cache "not found" results
})
```

The cache:
- Automatically invalidates on Save/Delete operations
- Uses LRU eviction when max entries exceeded
- Supports negative caching for missing templates
- Provides cache statistics via `Stats()`

```go
stats := cached.Stats()
fmt.Printf("Entries: %d, Valid: %d, Negative: %d\n",
    stats.Entries, stats.ValidEntries, stats.NegativeEntries)
```

## StorageEngine

`StorageEngine` combines storage with the template engine:

```go
se, err := prompty.NewStorageEngine(prompty.StorageEngineConfig{
    Storage: storage,
    Engine:  engine,  // Optional, creates default if nil
    DisableParsedTemplateCache: false,  // Enable parsed template caching
})
```

### Execute Templates

```go
// Execute latest version
result, err := se.Execute(ctx, "greeting", data)

// Execute specific version
result, err := se.ExecuteVersion(ctx, "greeting", 1, data)

// Execute with pre-built context
execCtx := prompty.NewContext(data)
result, err := se.ExecuteWithContext(ctx, "greeting", execCtx)
```

### Save and Validate

```go
// Save with validation
err := se.Save(ctx, &prompty.StoredTemplate{
    Name:   "greeting",
    Source: "Hello {~prompty.var name=\"user\" /~}!",
})

// Save without validation (use with caution)
err := se.SaveWithoutValidation(ctx, &prompty.StoredTemplate{
    Name:   "draft",
    Source: "incomplete template...",
})

// Validate without saving
result, err := se.Validate(ctx, "greeting")
if !result.IsValid() {
    for _, issue := range result.Issues() {
        fmt.Printf("Line %d: %s\n", issue.Position.Line, issue.Message)
    }
}
```

### Register Custom Resolvers and Functions

```go
// Register resolver
err := se.RegisterResolver(myResolver)

// Register function
err := se.RegisterFunc(&prompty.Func{
    Name:    "double",
    MinArgs: 1,
    MaxArgs: 1,
    Fn: func(args []any) (any, error) {
        if n, ok := args[0].(int); ok {
            return n * 2, nil
        }
        return nil, fmt.Errorf("expected int")
    },
})
```

### Parsed Template Caching

StorageEngine caches parsed templates to avoid re-parsing:

```go
// Check cache stats
stats := se.ParsedCacheStats()
fmt.Printf("Cached: %d, Enabled: %v\n", stats.Entries, stats.Enabled)

// Clear cache if needed
se.ClearParsedCache()
```

The cache automatically invalidates when:
- Template is saved (new version)
- Template is deleted
- Template version is deleted

## Querying Templates

Use `TemplateQuery` for flexible filtering:

```go
// Query by tenant
results, err := storage.List(ctx, &prompty.TemplateQuery{
    TenantID: "org_abc123",
})

// Query by name prefix
results, err := storage.List(ctx, &prompty.TemplateQuery{
    NamePrefix: "email-",
})

// Query by tags
results, err := storage.List(ctx, &prompty.TemplateQuery{
    Tags: []string{"public", "production"},
})

// Combined query with pagination
results, err := storage.List(ctx, &prompty.TemplateQuery{
    TenantID:   "org_abc123",
    NamePrefix: "notification-",
    Tags:       []string{"active"},
    Limit:      10,
    Offset:     20,
})

// Include all versions
results, err := storage.List(ctx, &prompty.TemplateQuery{
    IncludeAllVersions: true,
})
```

## Versioning

Templates are automatically versioned:

```go
// Save creates version 1
err := storage.Save(ctx, &prompty.StoredTemplate{
    Name:   "greeting",
    Source: "Hello!",
})

// Another save creates version 2
err := storage.Save(ctx, &prompty.StoredTemplate{
    Name:   "greeting",
    Source: "Hello, World!",
})

// Get latest version
tmpl, err := storage.Get(ctx, "greeting")
fmt.Println(tmpl.Version) // 2

// Get specific version
v1, err := storage.GetVersion(ctx, "greeting", 1)

// List all versions
versions, err := storage.ListVersions(ctx, "greeting")
// Returns: [2, 1] (newest first)

// Delete specific version
err := storage.DeleteVersion(ctx, "greeting", 1)
```

## Multi-Tenancy

Templates support multi-tenant isolation via `TenantID`:

```go
// Save tenant-specific template
err := storage.Save(ctx, &prompty.StoredTemplate{
    Name:     "welcome",
    Source:   "Welcome to Acme Corp!",
    TenantID: "tenant_acme",
})

// Query templates for a tenant
results, err := storage.List(ctx, &prompty.TemplateQuery{
    TenantID: "tenant_acme",
})
```

Note: The storage layer does not enforce tenant isolation automatically. Implement access control checks in your application layer.

## Implementing Custom Storage Drivers

Create custom storage backends by implementing `TemplateStorage`:

```go
type PostgresStorage struct {
    db *sql.DB
}

func (s *PostgresStorage) Get(ctx context.Context, name string) (*prompty.StoredTemplate, error) {
    // Your implementation
}

// ... implement other methods

// Register driver
func init() {
    prompty.RegisterStorageDriver("postgres", &PostgresDriver{})
}

type PostgresDriver struct{}

func (d *PostgresDriver) Open(connectionString string) (prompty.TemplateStorage, error) {
    db, err := sql.Open("postgres", connectionString)
    if err != nil {
        return nil, err
    }
    return &PostgresStorage{db: db}, nil
}
```

Usage:
```go
storage, err := prompty.OpenStorage("postgres", "postgres://user:pass@host/db")
```

## Error Handling

Storage operations return typed errors:

```go
tmpl, err := storage.Get(ctx, "nonexistent")
if err != nil {
    var storageErr *prompty.StorageError
    if errors.As(err, &storageErr) {
        fmt.Println(storageErr.Message) // "template not found"
        fmt.Println(storageErr.Name)    // "nonexistent"
    }
}
```

## Best Practices

### 1. Use StorageEngine for Production

```go
// Recommended: Use StorageEngine for combined storage + execution
se, err := prompty.NewStorageEngine(prompty.StorageEngineConfig{
    Storage: prompty.NewCachedStorage(
        prompty.MustNewFilesystemStorage("/templates"),
        prompty.DefaultCacheConfig(),
    ),
})
```

### 2. Validate Before Saving

```go
// StorageEngine.Save() validates automatically
err := se.Save(ctx, tmpl)

// Or validate manually
result, err := engine.Validate(tmpl.Source)
if !result.IsValid() {
    // Handle validation errors
}
```

### 3. Use Tags for Organization

```go
err := se.Save(ctx, &prompty.StoredTemplate{
    Name:   "notification-email",
    Source: "...",
    Tags:   []string{"notification", "email", "active", "v2"},
})
```

### 4. Use Metadata for Custom Data

```go
err := se.Save(ctx, &prompty.StoredTemplate{
    Name:   "report",
    Source: "...",
    Metadata: map[string]string{
        "author":      "team-a",
        "reviewed_by": "security",
        "model":       "gpt-4",
    },
})
```

### 5. Handle Concurrency

All built-in storage implementations are thread-safe:

```go
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        result, err := se.Execute(ctx, "greeting", data)
        // Safe for concurrent access
    }(i)
}
wg.Wait()
```

## See Also

- [Custom Storage Backends](CUSTOM_STORAGE.md) - Implementing PostgreSQL, MongoDB, Redis, etc.
- [Thread Safety Guide](THREAD_SAFETY.md)
- [README](../README.md)
