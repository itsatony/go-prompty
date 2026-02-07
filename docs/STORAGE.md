# Storage Layer Guide

go-prompty provides a pluggable storage layer for managing templates with versioning, metadata, and multi-tenant support. The storage layer follows a driver-based architecture similar to Go's `database/sql` package.

## Overview

The storage layer consists of:

- **TemplateStorage Interface**: Abstract interface for storage backends
- **StoredTemplate**: Template with metadata, versioning, and tenant support
- **Built-in Drivers**: Memory, filesystem, and PostgreSQL implementations
- **CachedStorage**: Caching wrapper for any storage backend
- **StorageEngine**: Combines storage with template execution

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/itsatony/go-prompty/v2"
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

### PostgreSQL Storage

Production-ready PostgreSQL storage with connection pooling and migrations:

```go
// Simple: Open via driver registry
storage, err := prompty.OpenStorage("postgres",
    "postgres://user:password@localhost:5432/prompty?sslmode=disable")

// Full control: Use PostgresConfig
storage, err := prompty.NewPostgresStorage(prompty.PostgresConfig{
    ConnectionString: os.Getenv("DATABASE_URL"),
    MaxOpenConns:     25,
    MaxIdleConns:     5,
    ConnMaxLifetime:  5 * time.Minute,
    ConnMaxIdleTime:  5 * time.Minute,
    TablePrefix:      "prompty_",
    AutoMigrate:      true,
    QueryTimeout:     30 * time.Second,
})
if err != nil {
    log.Fatal(err)
}
defer storage.Close()
```

Features:
- Automatic schema migrations with version tracking
- Connection pooling with configurable limits
- JSONB storage for metadata, inference config, and tags
- GIN indexes for efficient tag queries
- SERIALIZABLE transactions for version safety
- Context-aware query timeouts
- Thread-safe for concurrent access

#### Database Schema

The driver creates the following tables (with configurable prefix):

```sql
-- Templates table
CREATE TABLE prompty_templates (
    id               VARCHAR(255) PRIMARY KEY,
    name             VARCHAR(255) NOT NULL,
    source           TEXT NOT NULL,
    version          INTEGER NOT NULL DEFAULT 1,
    metadata         JSONB DEFAULT '{}',
    inference_config JSONB DEFAULT NULL,
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by       VARCHAR(255),
    tenant_id        VARCHAR(255),
    tags             JSONB DEFAULT '[]',
    CONSTRAINT prompty_templates_name_version_unique UNIQUE (name, version)
);

-- Schema migrations tracking
CREATE TABLE prompty_schema_migrations (
    version     INTEGER PRIMARY KEY,
    applied_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    description VARCHAR(255)
);
```

Indexes are automatically created for:
- `name` - Fast lookup by template name
- `name, version DESC` - Efficient latest version queries
- `tenant_id` - Multi-tenant filtering (partial index)
- `created_by` - Creator filtering (partial index)
- `tags` - GIN index for tag containment queries
- `created_at DESC` - Chronological listing

#### Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `ConnectionString` | (required) | PostgreSQL connection DSN |
| `MaxOpenConns` | 25 | Maximum open connections |
| `MaxIdleConns` | 5 | Maximum idle connections |
| `ConnMaxLifetime` | 5m | Maximum connection lifetime |
| `ConnMaxIdleTime` | 5m | Maximum idle time |
| `TablePrefix` | `prompty_` | Table name prefix |
| `AutoMigrate` | false | Run migrations on open |
| `QueryTimeout` | 30s | Default query timeout |

#### Running Migrations

Migrations run automatically with `AutoMigrate: true`, or manually:

```go
storage, err := prompty.NewPostgresStorage(prompty.PostgresConfig{
    ConnectionString: connStr,
    AutoMigrate:      false,
})
if err != nil {
    log.Fatal(err)
}

// Run migrations manually
if err := storage.RunMigrations(ctx); err != nil {
    log.Fatal(err)
}
```

#### Query Performance

The PostgreSQL driver optimizes common query patterns:

```go
// Efficient: Uses name index
tmpl, _ := storage.Get(ctx, "greeting")

// Efficient: Uses GIN index for tag containment
results, _ := storage.List(ctx, &prompty.TemplateQuery{
    Tags: []string{"production", "email"},
})

// Efficient: Uses partial index for tenant
results, _ := storage.List(ctx, &prompty.TemplateQuery{
    TenantID: "org_123",
})

// Efficient: Uses DISTINCT ON with name index
results, _ := storage.List(ctx, &prompty.TemplateQuery{
    IncludeAllVersions: false, // Default: only latest versions
})
```

### Opening Storage by Driver Name

Storage can be opened using registered driver names:

```go
// List available drivers
drivers := prompty.ListStorageDrivers()
fmt.Println(drivers) // ["memory", "filesystem", "postgres"]

// Open storage by driver name
storage, err := prompty.OpenStorage("filesystem", "/path/to/templates")

// Open PostgreSQL storage
storage, err := prompty.OpenStorage("postgres",
    "postgres://user:pass@localhost/prompty?sslmode=disable")
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

## Deployment-Aware Versioning

go-prompty supports deployment-aware versioning with labels and status for production workflows.

### Labels

Labels are named pointers to specific template versions, perfect for deployment workflows:

```go
// Set label to point to a specific version
err := se.SetLabel(ctx, "greeting", "production", 2)
err := se.SetLabel(ctx, "greeting", "staging", 3)
err := se.SetLabel(ctx, "greeting", "canary", 4)

// Execute template by label
result, err := se.ExecuteLabeled(ctx, "greeting", "production", data)

// Convenience: Execute the "production" labeled version
result, err := se.ExecuteProduction(ctx, "greeting", data)

// Get template by label
tmpl, err := se.GetByLabel(ctx, "greeting", "production")
tmpl, err := se.GetProduction(ctx, "greeting")

// Promote version to production
err := se.PromoteToProduction(ctx, "greeting", 5)

// List all labels for a template
labels, err := se.ListLabels(ctx, "greeting")
for _, l := range labels {
    fmt.Printf("  %s -> v%d (assigned by %s at %s)\n",
        l.Label, l.Version, l.AssignedBy, l.AssignedAt)
}
```

**Reserved Labels**:
- `production` - The live production version
- `staging` - Pre-production testing
- `canary` - Limited production rollout

**Label Rules**:
- Must start with lowercase letter
- Can contain lowercase letters, numbers, hyphens, underscores
- Maximum 64 characters
- Pattern: `^[a-z][a-z0-9_-]*$`

### Deployment Status

Status tracks the lifecycle of template versions:

```go
// Set status on a version
err := se.SetStatus(ctx, "greeting", 2, prompty.DeploymentStatusDeprecated)

// List templates by status
deprecated, err := se.ListByStatus(ctx, prompty.DeploymentStatusDeprecated, nil)
active, err := se.ListByStatus(ctx, prompty.DeploymentStatusActive, nil)
```

**Status Values**:
- `draft` - Work in progress, not ready for use
- `active` - Ready for production use (default for new templates)
- `deprecated` - Usable but scheduled for removal
- `archived` - No longer usable (terminal state)

**Status Transitions**:
| From | Allowed To |
|------|------------|
| draft | active, archived |
| active | deprecated, archived |
| deprecated | active, archived |
| archived | (terminal - no transitions allowed) |

### Version History with Labels

```go
history, err := se.GetVersionHistory(ctx, "greeting")

fmt.Printf("Template: %s\n", history.TemplateName)
fmt.Printf("Production version: %d\n", history.ProductionVersion)

for _, v := range history.Versions {
    fmt.Printf("  v%d: status=%s, labels=%v\n", v.Version, v.Status, v.Labels)
}
```

### Rollback and Clone Behavior

When rolling back or cloning, new versions start in `draft` status:

```go
// Rollback creates a new version with draft status
rolled, err := se.RollbackToVersion(ctx, "greeting", 1)
// rolled.Status == DeploymentStatusDraft

// Clone also creates with draft status
cloned, err := se.CloneVersion(ctx, "greeting", 1, "greeting-copy")
// cloned.Status == DeploymentStatusDraft

// Activate after review
err = se.SetStatus(ctx, "greeting", rolled.Version, prompty.DeploymentStatusActive)
```

### Checking Feature Support

```go
// Check if storage supports labels
if se.SupportsLabels() {
    err := se.SetLabel(ctx, "greeting", "production", 1)
}

// Check if storage supports status
if se.SupportsStatus() {
    err := se.SetStatus(ctx, "greeting", 1, prompty.DeploymentStatusActive)
}
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
type RedisStorage struct {
    client *redis.Client
}

func (s *RedisStorage) Get(ctx context.Context, name string) (*prompty.StoredTemplate, error) {
    // Your implementation
}

// ... implement other methods

// Register driver
func init() {
    prompty.RegisterStorageDriver("redis", &RedisDriver{})
}

type RedisDriver struct{}

func (d *RedisDriver) Open(connectionString string) (prompty.TemplateStorage, error) {
    opts, err := redis.ParseURL(connectionString)
    if err != nil {
        return nil, err
    }
    return &RedisStorage{client: redis.NewClient(opts)}, nil
}
```

Usage:
```go
storage, err := prompty.OpenStorage("redis", "redis://localhost:6379/0")
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

- [Custom Storage Backends](CUSTOM_STORAGE.md) - Implementing MongoDB, Redis, or other custom backends
- [Thread Safety Guide](THREAD_SAFETY.md)
- [README](../README.md)
