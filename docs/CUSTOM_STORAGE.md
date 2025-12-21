# Implementing Custom Storage Backends

This guide explains how to implement a custom storage backend for go-prompty. The storage layer follows a driver-based architecture similar to Go's `database/sql` package, making it straightforward to add support for any database or storage system.

## Overview

To create a custom storage backend, you need to:

1. Implement the `TemplateStorage` interface
2. Optionally implement `StorageDriver` for driver-based opening
3. Handle versioning, multi-tenancy, and error cases correctly

## The TemplateStorage Interface

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

## Implementation Checklist

When implementing a custom storage backend, ensure you handle:

| Requirement | Description |
|-------------|-------------|
| Thread Safety | All methods must be safe for concurrent access |
| Versioning | `Save()` must auto-increment version for existing templates |
| Not Found Errors | Return `NewStorageTemplateNotFoundError(name)` for missing templates |
| Context Cancellation | Respect `ctx.Done()` for long-running operations |
| Resource Cleanup | `Close()` must release all resources |
| Closed State | Operations on closed storage should return errors |

## Complete PostgreSQL Example

See the full implementation in `examples/custom_storage_postgres/`:

- `main.go` - Complete PostgreSQL storage implementation
- `schema.sql` - Database schema

### Key Implementation Patterns

#### 1. Connection Management

```go
type PostgresStorage struct {
    db     *sql.DB
    mu     sync.RWMutex // Protects closed state
    closed bool
}

func NewPostgresStorage(connectionString string) (*PostgresStorage, error) {
    db, err := sql.Open("postgres", connectionString)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // Verify connection
    if err := db.Ping(); err != nil {
        db.Close()
        return nil, fmt.Errorf("failed to connect: %w", err)
    }

    // Configure connection pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)

    return &PostgresStorage{db: db}, nil
}
```

#### 2. Version Management with Transactions

```go
func (s *PostgresStorage) Save(ctx context.Context, tmpl *prompty.StoredTemplate) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    // Get current max version atomically
    var maxVersion sql.NullInt64
    err = tx.QueryRowContext(ctx,
        `SELECT MAX(version) FROM templates WHERE name = $1`,
        tmpl.Name,
    ).Scan(&maxVersion)
    if err != nil {
        return fmt.Errorf("failed to get max version: %w", err)
    }

    // Calculate new version
    newVersion := 1
    if maxVersion.Valid {
        newVersion = int(maxVersion.Int64) + 1
    }

    // Insert with new version
    _, err = tx.ExecContext(ctx, `
        INSERT INTO templates (id, name, source, version, ...)
        VALUES ($1, $2, $3, $4, ...)`,
        tmpl.ID, tmpl.Name, tmpl.Source, newVersion, ...,
    )
    if err != nil {
        return fmt.Errorf("failed to insert: %w", err)
    }

    return tx.Commit()
}
```

#### 3. Proper Error Handling

```go
func (s *PostgresStorage) Get(ctx context.Context, name string) (*prompty.StoredTemplate, error) {
    // Check closed state
    s.mu.RLock()
    if s.closed {
        s.mu.RUnlock()
        return nil, fmt.Errorf("storage is closed")
    }
    s.mu.RUnlock()

    row := s.db.QueryRowContext(ctx, query, name)

    err := row.Scan(...)
    if err == sql.ErrNoRows {
        // Use the prompty error constructor for not-found
        return nil, prompty.NewStorageTemplateNotFoundError(name)
    }
    if err != nil {
        return nil, fmt.Errorf("failed to scan: %w", err)
    }

    return tmpl, nil
}
```

#### 4. JSON Storage for Flexible Data

```go
// Store metadata and tags as JSON
metadataJSON, _ := json.Marshal(tmpl.Metadata)
tagsJSON, _ := json.Marshal(tmpl.Tags)

// Query with JSON operators (PostgreSQL)
// Find by tag:
`SELECT * FROM templates WHERE tags @> $1::jsonb`

// Find by metadata:
`SELECT * FROM templates WHERE metadata @> $1::jsonb`
```

## Driver Registration (Optional)

Implement `StorageDriver` to enable driver-based opening:

```go
type PostgresDriver struct{}

func (d *PostgresDriver) Open(connectionString string) (prompty.TemplateStorage, error) {
    return NewPostgresStorage(connectionString)
}

// Register during initialization
func init() {
    prompty.RegisterStorageDriver("postgres", &PostgresDriver{})
}

// Usage:
storage, err := prompty.OpenStorage("postgres", "postgres://user:pass@host/db")
```

## Adapting to Other Databases

### MySQL

```go
// Key differences:
// 1. Parameter placeholders: $1 -> ?
// 2. JSON functions differ:
//    PostgreSQL: tags @> '["tag"]'::jsonb
//    MySQL: JSON_CONTAINS(tags, '"tag"')
// 3. Upsert syntax differs
```

### MongoDB

```go
type MongoStorage struct {
    client     *mongo.Client
    collection *mongo.Collection
}

func (s *MongoStorage) Get(ctx context.Context, name string) (*prompty.StoredTemplate, error) {
    filter := bson.M{"name": name}
    opts := options.FindOne().SetSort(bson.M{"version": -1})

    var tmpl prompty.StoredTemplate
    err := s.collection.FindOne(ctx, filter, opts).Decode(&tmpl)
    if err == mongo.ErrNoDocuments {
        return nil, prompty.NewStorageTemplateNotFoundError(name)
    }
    return &tmpl, err
}
```

### Redis

```go
type RedisStorage struct {
    client *redis.Client
}

// Use hash structures:
// Key: "template:{name}:{version}"
// Hash fields: source, metadata, tags, etc.

func (s *RedisStorage) Get(ctx context.Context, name string) (*prompty.StoredTemplate, error) {
    // Get latest version number
    version, err := s.client.Get(ctx, "template:"+name+":latest").Int()
    if err == redis.Nil {
        return nil, prompty.NewStorageTemplateNotFoundError(name)
    }

    // Get template data
    key := fmt.Sprintf("template:%s:%d", name, version)
    data, err := s.client.HGetAll(ctx, key).Result()
    // ...
}
```

### DynamoDB

```go
// Use composite keys:
// Partition Key: name
// Sort Key: version (as number for sorting)

func (s *DynamoStorage) Get(ctx context.Context, name string) (*prompty.StoredTemplate, error) {
    input := &dynamodb.QueryInput{
        TableName: aws.String("templates"),
        KeyConditionExpression: aws.String("name = :name"),
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":name": {S: aws.String(name)},
        },
        ScanIndexForward: aws.Bool(false), // Descending by sort key
        Limit: aws.Int64(1),
    }
    // ...
}
```

## Using with StorageEngine

Wrap your custom storage with `StorageEngine` for template execution:

```go
// Basic usage
storage := NewMyCustomStorage(config)
engine, err := prompty.NewStorageEngine(prompty.StorageEngineConfig{
    Storage: storage,
})

// With caching (recommended for production)
cached := prompty.NewCachedStorage(storage, prompty.CacheConfig{
    TTL:        5 * time.Minute,
    MaxEntries: 1000,
})
engine, err := prompty.NewStorageEngine(prompty.StorageEngineConfig{
    Storage: cached,
})

// Execute templates
result, err := engine.Execute(ctx, "my-template", data)
```

## Testing Your Implementation

Use the storage test suite to verify your implementation:

```go
func TestMyStorage_Compliance(t *testing.T) {
    storage := NewMyStorage(testConfig)
    defer storage.Close()

    ctx := context.Background()

    t.Run("Save and Get", func(t *testing.T) {
        tmpl := &prompty.StoredTemplate{
            Name:   "test",
            Source: "Hello {~prompty.var name=\"x\" /~}",
        }

        err := storage.Save(ctx, tmpl)
        require.NoError(t, err)
        assert.Equal(t, 1, tmpl.Version)

        retrieved, err := storage.Get(ctx, "test")
        require.NoError(t, err)
        assert.Equal(t, "test", retrieved.Name)
        assert.Equal(t, 1, retrieved.Version)
    })

    t.Run("Versioning", func(t *testing.T) {
        // Save again - should create version 2
        err := storage.Save(ctx, &prompty.StoredTemplate{
            Name:   "test",
            Source: "Updated",
        })
        require.NoError(t, err)

        retrieved, err := storage.Get(ctx, "test")
        require.NoError(t, err)
        assert.Equal(t, 2, retrieved.Version)

        // Can still get version 1
        v1, err := storage.GetVersion(ctx, "test", 1)
        require.NoError(t, err)
        assert.Equal(t, 1, v1.Version)
    })

    t.Run("Not Found", func(t *testing.T) {
        _, err := storage.Get(ctx, "nonexistent")
        assert.Error(t, err)
        // Should be a StorageError with appropriate type
    })

    t.Run("Concurrent Access", func(t *testing.T) {
        var wg sync.WaitGroup
        for i := 0; i < 10; i++ {
            wg.Add(1)
            go func() {
                defer wg.Done()
                _, _ = storage.Get(ctx, "test")
            }()
        }
        wg.Wait()
    })
}
```

## Best Practices

1. **Connection Pooling**: Configure appropriate pool sizes for your workload
2. **Caching**: Use `CachedStorage` wrapper for read-heavy workloads
3. **Transactions**: Use transactions for version management to prevent race conditions
4. **Indexes**: Create indexes for common query patterns (name, tenant_id, tags)
5. **Error Wrapping**: Use `fmt.Errorf("context: %w", err)` for debugging
6. **Graceful Shutdown**: Ensure `Close()` properly releases all resources

## See Also

- [Storage Layer Guide](STORAGE.md) - Overview of storage features
- [examples/custom_storage_postgres](../examples/custom_storage_postgres/) - Complete PostgreSQL example
- [examples/storage_persistence](../examples/storage_persistence/) - Filesystem persistence example
