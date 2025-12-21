# Performance Guide

This guide covers performance characteristics, optimization strategies, and tuning recommendations for go-prompty.

## Table of Contents

1. [Performance Characteristics](#performance-characteristics)
2. [Running Benchmarks](#running-benchmarks)
3. [Parse vs Execute](#parse-vs-execute)
4. [Caching Strategies](#caching-strategies)
5. [Concurrent Usage](#concurrent-usage)
6. [Memory Optimization](#memory-optimization)
7. [Template Design Tips](#template-design-tips)
8. [Storage Performance](#storage-performance)
9. [Production Tuning](#production-tuning)

---

## Performance Characteristics

### Baseline Performance

Typical performance on modern hardware (results may vary):

| Operation | Time | Memory | Notes |
|-----------|------|--------|-------|
| Parse simple template | ~2-3μs | ~7KB | Single variable |
| Parse complex template | ~30-40μs | ~50KB | Conditionals + loops |
| Execute simple template | ~1-2μs | ~2KB | Pre-parsed |
| Execute with caching (hit) | ~50-100ns | ~0.5KB | Cache lookup only |
| Context path lookup | ~100-200ns | ~0 | Nested path resolution |

### What Affects Performance

1. **Template Complexity**: More tags = more parsing time
2. **Data Depth**: Deeply nested data paths take longer to resolve
3. **Loop Size**: Large loops scale linearly with item count
4. **Expression Complexity**: Complex expressions with functions are slower
5. **Include Depth**: Nested template includes add overhead

---

## Running Benchmarks

### Run All Benchmarks

```bash
# Basic benchmark run
go test -bench=. -benchmem ./...

# With specific benchtime for more accurate results
go test -bench=. -benchtime=3s -benchmem ./...

# Filter to specific benchmarks
go test -bench=BenchmarkExecute -benchmem ./...

# Output to file for comparison
go test -bench=. -benchmem ./... > benchmark_results.txt
```

### Compare Benchmarks

Use `benchstat` for comparison:

```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Run benchmarks before changes
go test -bench=. -count=5 ./... > old.txt

# Make changes...

# Run benchmarks after changes
go test -bench=. -count=5 ./... > new.txt

# Compare
benchstat old.txt new.txt
```

### Key Benchmarks to Watch

| Benchmark | Purpose |
|-----------|---------|
| `BenchmarkParse_*` | Template parsing overhead |
| `BenchmarkExecute_PreParsed` | Pure execution without parsing |
| `BenchmarkCachedEngine_Hit` | Cache effectiveness |
| `BenchmarkExecute_Loop_*` | Loop scaling behavior |
| `BenchmarkExecute_Concurrent` | Parallel scaling |
| `BenchmarkParallelScaling` | Goroutine overhead |

---

## Parse vs Execute

### The Golden Rule: Parse Once, Execute Many

```go
// SLOW: Parsing on every request
func handleRequest(source string, data map[string]any) string {
    result, _ := engine.Execute(ctx, source, data)  // Parses every time
    return result
}

// FAST: Parse once, reuse template
var tmpl *prompty.Template

func init() {
    tmpl, _ = engine.Parse(source)  // Parse once at startup
}

func handleRequest(data map[string]any) string {
    result, _ := tmpl.Execute(ctx, data)  // Execute only
    return result
}
```

### Performance Comparison

```
BenchmarkParse_Simple:       2,272 ns/op
BenchmarkExecute_PreParsed:    800 ns/op  // 2.8x faster
```

For a simple template:
- Parsing takes ~2-3μs
- Execution takes ~1μs
- Parsing is 2-3x more expensive than execution

### Registered Templates

For templates that need dynamic lookup, use `RegisterTemplate`:

```go
// Register once at startup
engine.MustRegisterTemplate("greeting", `Hello {~prompty.var name="user" /~}!`)
engine.MustRegisterTemplate("farewell", `Goodbye {~prompty.var name="user" /~}!`)

// Execute by name (no re-parsing)
result, _ := engine.ExecuteTemplate(ctx, "greeting", data)
```

---

## Caching Strategies

### Result Caching

Use `CachedEngine` to cache execution results:

```go
// Create cached engine
config := prompty.ResultCacheConfig{
    TTL:           5 * time.Minute,
    MaxEntries:    1000,
    MaxResultSize: 1 << 20,  // 1MB
}
cached := prompty.NewCachedEngine(engine, config)

// First call: cache miss, executes template
result1, _ := cached.Execute(ctx, source, data)  // ~1000ns

// Second call: cache hit, returns cached result
result2, _ := cached.Execute(ctx, source, data)  // ~50ns
```

### When to Use Result Caching

**Good candidates for caching:**
- Templates with static or slowly-changing data
- Expensive expressions or functions
- High-frequency templates with predictable inputs

**Avoid caching when:**
- Data changes frequently
- Each request has unique data
- Templates include timestamps or random values
- Memory is constrained

### Cache Configuration

```go
config := prompty.ResultCacheConfig{
    // TTL: How long results stay cached
    TTL: 5 * time.Minute,  // Default

    // MaxEntries: Maximum cached results
    // Higher = more memory, better hit rate
    MaxEntries: 1000,  // Default

    // MaxResultSize: Skip caching large results
    // Prevents memory bloat from large outputs
    MaxResultSize: 1 << 20,  // 1MB default

    // KeyPrefix: Namespace cache keys
    // Useful for multi-tenant or per-user caches
    KeyPrefix: "tenant_123:",
}
```

### Monitoring Cache Effectiveness

```go
stats := cached.CacheStats()
fmt.Printf("Hit Rate: %.2f%%\n", cached.CacheHitRate()*100)
fmt.Printf("Hits: %d, Misses: %d\n", stats.Hits, stats.Misses)
fmt.Printf("Entries: %d, Size: %d bytes\n", stats.EntryCount, stats.TotalSize)

// Target: >70% hit rate for effective caching
// Below 30%: Caching may not be beneficial
```

---

## Concurrent Usage

### Thread Safety

go-prompty is designed for concurrent use:

```go
// Engine and templates are safe for concurrent access
var engine = prompty.MustNew()
var tmpl, _ = engine.Parse(source)

// Safe to call from multiple goroutines
go func() { tmpl.Execute(ctx, data1) }()
go func() { tmpl.Execute(ctx, data2) }()
go func() { tmpl.Execute(ctx, data3) }()
```

### Parallel Scaling

Benchmark results for parallel execution:

```
BenchmarkParallelScaling/Goroutines-1:   1,000,000 ops
BenchmarkParallelScaling/Goroutines-2:   1,800,000 ops  // ~1.8x
BenchmarkParallelScaling/Goroutines-4:   3,200,000 ops  // ~3.2x
BenchmarkParallelScaling/Goroutines-8:   5,500,000 ops  // ~5.5x
```

Scaling is nearly linear up to CPU core count.

### Avoiding Contention

**Don't share mutable data between goroutines:**

```go
// BAD: Shared data map
sharedData := map[string]any{"counter": 0}
for i := 0; i < 10; i++ {
    go func() {
        sharedData["counter"]++  // RACE CONDITION
        tmpl.Execute(ctx, sharedData)
    }()
}

// GOOD: Each goroutine gets its own data
for i := 0; i < 10; i++ {
    go func(id int) {
        data := map[string]any{"id": id}  // Local copy
        tmpl.Execute(ctx, data)
    }(i)
}
```

---

## Memory Optimization

### Reducing Allocations

Track allocations with benchmarks:

```bash
go test -bench=BenchmarkExecute -benchmem ./...

# Output:
# BenchmarkExecute_PreParsed    1234567   810 ns/op   2048 B/op   12 allocs/op
```

### Tips for Reducing Memory

1. **Reuse templates**: Parse once, execute many times
2. **Use caching**: Avoid redundant string allocations
3. **Limit loop sizes**: Set reasonable limits
4. **Avoid deep nesting**: Flatter data structures are faster
5. **Pre-size slices**: If building large outputs, pre-allocate

### Memory Limits

go-prompty has built-in limits:

| Limit | Default | Config |
|-------|---------|--------|
| Max Output Size | 10MB | `WithMaxOutputSize()` |
| Max Loop Iterations | 10,000 | `WithMaxLoopIterations()` |
| Max Depth | 10 | `WithMaxDepth()` |

```go
engine, _ := prompty.New(
    prompty.WithMaxOutputSize(5 << 20),     // 5MB
    prompty.WithMaxLoopIterations(5000),
    prompty.WithMaxDepth(5),
)
```

---

## Template Design Tips

### Keep Templates Simple

```go
// SLOW: Complex expressions evaluated every iteration
source := `{~prompty.for item="x" in="items"~}
{~prompty.if eval="complexFunction(x) && len(otherList) > 0"~}
...
{~/prompty.if~}
{~/prompty.for~}`

// FAST: Pre-compute in data
data := map[string]any{
    "items":         items,
    "showContent":   complexFunction(items[0]) && len(otherList) > 0,
    "filteredItems": filterItems(items),  // Filter in Go
}
source := `{~prompty.for item="x" in="filteredItems"~}
...
{~/prompty.for~}`
```

### Minimize Variable Lookups

```go
// SLOW: Multiple lookups for same path
`{~prompty.var name="user.profile.settings.theme.color" /~}
{~prompty.var name="user.profile.settings.theme.font" /~}
{~prompty.var name="user.profile.settings.theme.size" /~}`

// FAST: Flatten in data
data := map[string]any{
    "themeColor": user.Profile.Settings.Theme.Color,
    "themeFont":  user.Profile.Settings.Theme.Font,
    "themeSize":  user.Profile.Settings.Theme.Size,
}
```

### Use Includes Wisely

```go
// SLOW: Deep nesting of includes
// outer → middle → inner → innermost

// BETTER: Flatten where possible
// Consider if the abstraction is worth the overhead
```

---

## Storage Performance

### Memory vs Filesystem Storage

| Storage | Read | Write | Best For |
|---------|------|-------|----------|
| Memory | ~100ns | ~200ns | Development, testing, small sets |
| Filesystem | ~1ms | ~2ms | Persistence, moderate traffic |

### Caching Storage

Always wrap storage with caching for production:

```go
storage := prompty.NewMemoryStorage()  // or FilesystemStorage
cached := prompty.NewCachedStorage(storage, prompty.CacheConfig{
    TTL:         5 * time.Minute,
    MaxEntries:  1000,
    NegativeTTL: 30 * time.Second,
})
engine, _ := prompty.NewStorageEngine(prompty.StorageEngineConfig{
    Storage: cached,
})
```

### Result + Storage Caching

For maximum performance, cache both template loading AND results:

```go
// Layer 1: Cache template storage
cachedStorage := prompty.NewCachedStorage(storage, storageConfig)

// Layer 2: Cache execution results
storageEngine, _ := prompty.NewStorageEngine(prompty.StorageEngineConfig{
    Storage: cachedStorage,
})
cachedEngine := prompty.NewCachedStorageEngine(storageEngine, resultConfig)

// Now both template loading and execution are cached
result, _ := cachedEngine.Execute(ctx, "template-name", data)
```

---

## Production Tuning

### Recommended Configuration

```go
// Engine with production limits
engine, _ := prompty.New(
    prompty.WithDefaultTimeout(5 * time.Second),
    prompty.WithMaxDepth(10),
    prompty.WithMaxLoopIterations(10000),
    prompty.WithMaxOutputSize(10 << 20),  // 10MB
)

// Result caching for high-traffic templates
resultCache := prompty.ResultCacheConfig{
    TTL:           5 * time.Minute,
    MaxEntries:    10000,
    MaxResultSize: 1 << 20,
}
cachedEngine := prompty.NewCachedEngine(engine, resultCache)
```

### Monitoring Checklist

1. **Track cache hit rates**: Target >70%
2. **Monitor execution times**: P99 latency
3. **Watch memory usage**: Cache size growth
4. **Log errors**: Timeout and limit errors
5. **Track template complexity**: Parsing times

### Warning Signs

| Symptom | Possible Cause | Solution |
|---------|---------------|----------|
| High latency | No caching | Add result caching |
| Memory growth | No cache limits | Set MaxEntries |
| Timeouts | Complex expressions | Simplify or increase timeout |
| High CPU | Re-parsing | Use registered templates |
| Low hit rate | High data variance | Review caching strategy |

### Benchmarking Your Templates

Create custom benchmarks for your specific templates:

```go
func BenchmarkMyProductionTemplate(b *testing.B) {
    engine := prompty.MustNew()
    tmpl, _ := engine.Parse(myProductionTemplate)
    data := loadTypicalProductionData()
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = tmpl.Execute(ctx, data)
    }
}
```

---

## Quick Reference

### Performance Hierarchy

From fastest to slowest:

1. **Cache hit**: ~50-100ns
2. **Execute pre-parsed**: ~1-2μs
3. **Execute with parsing**: ~3-5μs
4. **Execute with loops**: ~10-100μs (depends on size)
5. **Execute with includes**: +overhead per include

### Optimization Priority

1. Parse once, execute many
2. Enable result caching for repeated inputs
3. Use registered templates
4. Simplify complex expressions
5. Pre-compute in Go where possible
6. Monitor and tune cache configuration

### Memory Budget Guidelines

| Use Case | Suggested Cache Size |
|----------|---------------------|
| Small app | 100-500 entries |
| Medium app | 1,000-5,000 entries |
| High-traffic | 10,000+ entries |
| Per-tenant | 100-1,000 per tenant |

---

## See Also

- [Thread Safety Guide](THREAD_SAFETY.md)
- [Error Strategies Guide](ERROR_STRATEGIES.md)
- [Common Pitfalls](COMMON_PITFALLS.md)
