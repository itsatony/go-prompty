package prompty

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResultCache(t *testing.T) {
	cache := NewResultCache(DefaultResultCacheConfig())

	assert.NotNil(t, cache)
	assert.Equal(t, 5*time.Minute, cache.config.TTL)
	assert.Equal(t, 1000, cache.config.MaxEntries)
	assert.Equal(t, 1<<20, cache.config.MaxResultSize)
}

func TestNewResultCache_CustomConfig(t *testing.T) {
	config := ResultCacheConfig{
		TTL:           10 * time.Minute,
		MaxEntries:    500,
		MaxResultSize: 1024,
		KeyPrefix:     "test:",
	}
	cache := NewResultCache(config)

	assert.Equal(t, 10*time.Minute, cache.config.TTL)
	assert.Equal(t, 500, cache.config.MaxEntries)
	assert.Equal(t, 1024, cache.config.MaxResultSize)
	assert.Equal(t, "test:", cache.config.KeyPrefix)
}

func TestResultCache_GetSet(t *testing.T) {
	cache := NewResultCache(DefaultResultCacheConfig())

	data := map[string]any{"user": "Alice"}

	// Initially not in cache
	_, found := cache.Get("template1", data)
	assert.False(t, found)

	// Set and verify
	cache.Set("template1", data, "Hello Alice!")
	result, found := cache.Get("template1", data)
	assert.True(t, found)
	assert.Equal(t, "Hello Alice!", result)
}

func TestResultCache_DifferentData(t *testing.T) {
	cache := NewResultCache(DefaultResultCacheConfig())

	data1 := map[string]any{"user": "Alice"}
	data2 := map[string]any{"user": "Bob"}

	cache.Set("template1", data1, "Hello Alice!")
	cache.Set("template1", data2, "Hello Bob!")

	result1, _ := cache.Get("template1", data1)
	result2, _ := cache.Get("template1", data2)

	assert.Equal(t, "Hello Alice!", result1)
	assert.Equal(t, "Hello Bob!", result2)
}

func TestResultCache_Expiration(t *testing.T) {
	config := DefaultResultCacheConfig()
	config.TTL = 50 * time.Millisecond
	cache := NewResultCache(config)

	data := map[string]any{"user": "Alice"}
	cache.Set("template1", data, "Hello Alice!")

	// Should be in cache initially
	_, found := cache.Get("template1", data)
	assert.True(t, found)

	// Wait for expiration
	time.Sleep(60 * time.Millisecond)

	// Should be expired
	_, found = cache.Get("template1", data)
	assert.False(t, found)
}

func TestResultCache_MaxSize(t *testing.T) {
	config := DefaultResultCacheConfig()
	config.MaxResultSize = 10
	cache := NewResultCache(config)

	data := map[string]any{}

	// Small result should be cached
	cache.Set("small", data, "short")
	_, found := cache.Get("small", data)
	assert.True(t, found)

	// Large result should not be cached
	cache.Set("large", data, "this is a very long result that exceeds the max size")
	_, found = cache.Get("large", data)
	assert.False(t, found)
}

func TestResultCache_MaxEntries(t *testing.T) {
	config := DefaultResultCacheConfig()
	config.MaxEntries = 3
	cache := NewResultCache(config)

	// Fill cache
	cache.Set("t1", nil, "r1")
	cache.Set("t2", nil, "r2")
	cache.Set("t3", nil, "r3")

	// Add one more (should evict oldest)
	cache.Set("t4", nil, "r4")

	// t1 should be evicted
	_, found := cache.Get("t1", nil)
	assert.False(t, found)

	// Others should still be there
	_, found = cache.Get("t2", nil)
	assert.True(t, found)
	_, found = cache.Get("t4", nil)
	assert.True(t, found)
}

func TestResultCache_Invalidate(t *testing.T) {
	cache := NewResultCache(DefaultResultCacheConfig())

	data := map[string]any{"user": "Alice"}
	cache.Set("template1", data, "Hello Alice!")

	// Verify it's cached
	_, found := cache.Get("template1", data)
	assert.True(t, found)

	// Invalidate
	cache.Invalidate("template1", data)

	// Should be gone
	_, found = cache.Get("template1", data)
	assert.False(t, found)
}

func TestResultCache_InvalidateTemplate(t *testing.T) {
	cache := NewResultCache(DefaultResultCacheConfig())

	data1 := map[string]any{"user": "Alice"}
	data2 := map[string]any{"user": "Bob"}

	cache.Set("template1", data1, "Hello Alice!")
	cache.Set("template1", data2, "Hello Bob!")
	cache.Set("template2", data1, "Bye Alice!")

	// Invalidate all entries for template1
	cache.InvalidateTemplate("template1")

	// template1 entries should be gone
	_, found := cache.Get("template1", data1)
	assert.False(t, found)
	_, found = cache.Get("template1", data2)
	assert.False(t, found)

	// template2 should still be there
	_, found = cache.Get("template2", data1)
	assert.True(t, found)
}

func TestResultCache_Clear(t *testing.T) {
	cache := NewResultCache(DefaultResultCacheConfig())

	cache.Set("t1", nil, "r1")
	cache.Set("t2", nil, "r2")
	cache.Set("t3", nil, "r3")

	cache.Clear()

	stats := cache.Stats()
	assert.Equal(t, 0, stats.EntryCount)

	_, found := cache.Get("t1", nil)
	assert.False(t, found)
}

func TestResultCache_Stats(t *testing.T) {
	cache := NewResultCache(DefaultResultCacheConfig())

	data := map[string]any{"user": "Alice"}

	// Miss
	cache.Get("template1", data)

	// Set
	cache.Set("template1", data, "Hello Alice!")

	// Hit
	cache.Get("template1", data)
	cache.Get("template1", data)

	stats := cache.Stats()
	assert.Equal(t, int64(2), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, 1, stats.EntryCount)
}

func TestResultCache_HitRate(t *testing.T) {
	cache := NewResultCache(DefaultResultCacheConfig())

	// Empty cache - 0 hit rate
	assert.Equal(t, 0.0, cache.HitRate())

	data := map[string]any{"user": "Alice"}

	// One miss
	cache.Get("template1", data)
	assert.Equal(t, 0.0, cache.HitRate())

	// Set and hit
	cache.Set("template1", data, "Hello!")
	cache.Get("template1", data)

	// 1 hit, 1 miss = 50%
	assert.Equal(t, 0.5, cache.HitRate())
}

func TestResultCache_Cleanup(t *testing.T) {
	config := DefaultResultCacheConfig()
	config.TTL = 50 * time.Millisecond
	cache := NewResultCache(config)

	cache.Set("t1", nil, "r1")
	cache.Set("t2", nil, "r2")

	// Wait for expiration
	time.Sleep(60 * time.Millisecond)

	removed := cache.Cleanup()
	assert.Equal(t, 2, removed)

	stats := cache.Stats()
	assert.Equal(t, 0, stats.EntryCount)
}

func TestResultCache_NilData(t *testing.T) {
	cache := NewResultCache(DefaultResultCacheConfig())

	cache.Set("template1", nil, "result")
	result, found := cache.Get("template1", nil)

	assert.True(t, found)
	assert.Equal(t, "result", result)
}

func TestCachedEngine_Execute(t *testing.T) {
	engine := MustNew()
	cachedEngine := NewCachedEngine(engine, DefaultResultCacheConfig())

	source := `Hello {~prompty.var name="user" /~}!`
	data := map[string]any{"user": "Alice"}

	// First call - cache miss
	result1, err := cachedEngine.Execute(context.Background(), source, data)
	require.NoError(t, err)
	assert.Equal(t, "Hello Alice!", result1)

	// Second call - cache hit
	result2, err := cachedEngine.Execute(context.Background(), source, data)
	require.NoError(t, err)
	assert.Equal(t, "Hello Alice!", result2)

	// Verify cache was used
	stats := cachedEngine.CacheStats()
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
}

func TestCachedEngine_ExecuteTemplate(t *testing.T) {
	engine := MustNew()
	err := engine.RegisterTemplate("greeting", `Hello {~prompty.var name="user" /~}!`)
	require.NoError(t, err)

	cachedEngine := NewCachedEngine(engine, DefaultResultCacheConfig())

	data := map[string]any{"user": "Bob"}

	// First call - cache miss
	result1, err := cachedEngine.ExecuteTemplate(context.Background(), "greeting", data)
	require.NoError(t, err)
	assert.Equal(t, "Hello Bob!", result1)

	// Second call - cache hit
	result2, err := cachedEngine.ExecuteTemplate(context.Background(), "greeting", data)
	require.NoError(t, err)
	assert.Equal(t, "Hello Bob!", result2)

	assert.Equal(t, int64(1), cachedEngine.CacheStats().Hits)
}

func TestCachedEngine_InvalidateCache(t *testing.T) {
	engine := MustNew()
	cachedEngine := NewCachedEngine(engine, DefaultResultCacheConfig())

	source := `Hello World`
	_, _ = cachedEngine.Execute(context.Background(), source, nil)

	assert.Equal(t, 1, cachedEngine.CacheStats().EntryCount)

	cachedEngine.InvalidateCache()

	assert.Equal(t, 0, cachedEngine.CacheStats().EntryCount)
}

func TestCachedEngine_CacheHitRate(t *testing.T) {
	engine := MustNew()
	cachedEngine := NewCachedEngine(engine, DefaultResultCacheConfig())

	source := `Hello`

	// Miss
	_, _ = cachedEngine.Execute(context.Background(), source, nil)
	assert.Equal(t, 0.0, cachedEngine.CacheHitRate())

	// Hit
	_, _ = cachedEngine.Execute(context.Background(), source, nil)
	assert.Equal(t, 0.5, cachedEngine.CacheHitRate())
}

func TestCachedEngine_Engine(t *testing.T) {
	engine := MustNew()
	cachedEngine := NewCachedEngine(engine, DefaultResultCacheConfig())

	assert.Equal(t, engine, cachedEngine.Engine())
}

func TestCachedStorageEngine_Execute(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	// Save a template
	err = engine.Save(context.Background(), &StoredTemplate{
		Name:   "greeting",
		Source: `Hello {~prompty.var name="user" /~}!`,
	})
	require.NoError(t, err)

	cachedEngine := NewCachedStorageEngine(engine, DefaultResultCacheConfig())

	data := map[string]any{"user": "Charlie"}

	// First call - miss
	result1, err := cachedEngine.Execute(context.Background(), "greeting", data)
	require.NoError(t, err)
	assert.Equal(t, "Hello Charlie!", result1)

	// Second call - hit
	result2, err := cachedEngine.Execute(context.Background(), "greeting", data)
	require.NoError(t, err)
	assert.Equal(t, "Hello Charlie!", result2)

	assert.Equal(t, int64(1), cachedEngine.CacheStats().Hits)
}

func TestCachedStorageEngine_Save_InvalidatesCache(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	cachedEngine := NewCachedStorageEngine(engine, DefaultResultCacheConfig())

	// Save initial template
	err = cachedEngine.Save(context.Background(), &StoredTemplate{
		Name:   "greeting",
		Source: `Hello World`,
	})
	require.NoError(t, err)

	// Execute to populate cache
	_, err = cachedEngine.Execute(context.Background(), "greeting", nil)
	require.NoError(t, err)

	assert.Equal(t, 1, cachedEngine.CacheStats().EntryCount)

	// Save updated template - should invalidate cache
	err = cachedEngine.Save(context.Background(), &StoredTemplate{
		Name:   "greeting",
		Source: `Hello Universe`,
	})
	require.NoError(t, err)

	// Cache should be invalidated for this template
	// Execute again to get fresh result
	result, err := cachedEngine.Execute(context.Background(), "greeting", nil)
	require.NoError(t, err)
	assert.Equal(t, "Hello Universe", result)
}

func TestCachedStorageEngine_Delete_InvalidatesCache(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	cachedEngine := NewCachedStorageEngine(engine, DefaultResultCacheConfig())

	// Save template
	err = cachedEngine.Save(context.Background(), &StoredTemplate{
		Name:   "greeting",
		Source: `Hello`,
	})
	require.NoError(t, err)

	// Execute to populate cache
	_, _ = cachedEngine.Execute(context.Background(), "greeting", nil)

	// Delete
	err = cachedEngine.Delete(context.Background(), "greeting")
	require.NoError(t, err)

	// Execute should fail
	_, err = cachedEngine.Execute(context.Background(), "greeting", nil)
	assert.Error(t, err)
}

func TestResultCache_KeyPrefix(t *testing.T) {
	config := DefaultResultCacheConfig()
	config.KeyPrefix = "myapp:"
	cache := NewResultCache(config)

	cache.Set("template1", nil, "result")

	// Should find with same cache
	result, found := cache.Get("template1", nil)
	assert.True(t, found)
	assert.Equal(t, "result", result)
}

func TestResultCache_ConcurrentAccess(t *testing.T) {
	cache := NewResultCache(DefaultResultCacheConfig())

	// Run concurrent gets and sets
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			data := map[string]any{"id": id}
			cache.Set("template", data, "result")
			cache.Get("template", data)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and should have entries
	stats := cache.Stats()
	assert.True(t, stats.EntryCount > 0)
}

func BenchmarkResultCache_Get(b *testing.B) {
	cache := NewResultCache(DefaultResultCacheConfig())
	data := map[string]any{"user": "Alice", "count": 42}
	cache.Set("template", data, "Hello Alice!")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("template", data)
	}
}

func BenchmarkResultCache_Set(b *testing.B) {
	cache := NewResultCache(DefaultResultCacheConfig())
	data := map[string]any{"user": "Alice", "count": 42}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("template", data, "Hello Alice!")
	}
}

func BenchmarkCachedEngine_Execute_Hit(b *testing.B) {
	engine := MustNew()
	cachedEngine := NewCachedEngine(engine, DefaultResultCacheConfig())

	source := `Hello {~prompty.var name="user" /~}!`
	data := map[string]any{"user": "Alice"}

	// Warm up cache
	_, _ = cachedEngine.Execute(context.Background(), source, data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cachedEngine.Execute(context.Background(), source, data)
	}
}
