package prompty

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCachedStorage_NewCachedStorage(t *testing.T) {
	storage := NewMemoryStorage()
	cached := NewCachedStorage(storage, DefaultCacheConfig())

	require.NotNil(t, cached)
	assert.NotNil(t, cached.cache)
	assert.NotNil(t, cached.byID)
}

func TestCachedStorage_Get(t *testing.T) {
	storage := NewMemoryStorage()
	cached := NewCachedStorage(storage, CacheConfig{
		TTL:              1 * time.Hour,
		MaxEntries:       100,
		NegativeCacheTTL: 1 * time.Hour,
	})
	defer cached.Close()

	ctx := context.Background()

	// Save a template
	_ = storage.Save(ctx, &StoredTemplate{Name: "test", Source: "content"})

	t.Run("returns template and caches it", func(t *testing.T) {
		tmpl, err := cached.Get(ctx, "test")
		require.NoError(t, err)
		assert.Equal(t, "content", tmpl.Source)

		// Verify it's cached
		stats := cached.Stats()
		assert.Equal(t, 1, stats.ValidEntries)
	})

	t.Run("returns cached template on second call", func(t *testing.T) {
		// Modify underlying storage
		_ = storage.Save(ctx, &StoredTemplate{Name: "test", Source: "modified"})

		// Should still return cached version
		tmpl, err := cached.Get(ctx, "test")
		require.NoError(t, err)
		assert.Equal(t, "content", tmpl.Source) // Original content
	})

	t.Run("caches not found", func(t *testing.T) {
		_, err := cached.Get(ctx, "nonexistent")
		require.Error(t, err)

		stats := cached.Stats()
		assert.Equal(t, 1, stats.NegativeEntries)
	})
}

func TestCachedStorage_TTL(t *testing.T) {
	storage := NewMemoryStorage()
	cached := NewCachedStorage(storage, CacheConfig{
		TTL:              50 * time.Millisecond,
		MaxEntries:       100,
		NegativeCacheTTL: 50 * time.Millisecond,
	})
	defer cached.Close()

	ctx := context.Background()

	// Save a template
	_ = storage.Save(ctx, &StoredTemplate{Name: "test", Source: "original"})

	// Get and cache
	tmpl, err := cached.Get(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, "original", tmpl.Source)

	// Modify underlying storage
	_ = storage.Save(ctx, &StoredTemplate{Name: "test", Source: "modified"})

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Should now get modified version
	tmpl, err = cached.Get(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, "modified", tmpl.Source)
}

func TestCachedStorage_MaxEntries(t *testing.T) {
	storage := NewMemoryStorage()
	cached := NewCachedStorage(storage, CacheConfig{
		TTL:        1 * time.Hour,
		MaxEntries: 3,
	})
	defer cached.Close()

	ctx := context.Background()

	// Save templates
	for i := 0; i < 5; i++ {
		_ = storage.Save(ctx, &StoredTemplate{
			Name:   "tmpl" + intToStr(i),
			Source: "content" + intToStr(i),
		})
	}

	// Access all templates
	for i := 0; i < 5; i++ {
		_, _ = cached.Get(ctx, "tmpl"+intToStr(i))
	}

	// Cache should be limited to MaxEntries
	stats := cached.Stats()
	assert.LessOrEqual(t, stats.Entries, 3)
}

func TestCachedStorage_Invalidate(t *testing.T) {
	storage := NewMemoryStorage()
	cached := NewCachedStorage(storage, CacheConfig{
		TTL:        1 * time.Hour,
		MaxEntries: 100,
	})
	defer cached.Close()

	ctx := context.Background()

	// Save and cache
	_ = storage.Save(ctx, &StoredTemplate{Name: "test", Source: "original"})
	_, _ = cached.Get(ctx, "test")

	// Verify cached
	stats := cached.Stats()
	assert.Equal(t, 1, stats.ValidEntries)

	// Invalidate
	cached.Invalidate("test")

	// Verify removed
	stats = cached.Stats()
	assert.Equal(t, 0, stats.ValidEntries)

	// Modify and re-fetch
	_ = storage.Save(ctx, &StoredTemplate{Name: "test", Source: "modified"})
	tmpl, err := cached.Get(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, "modified", tmpl.Source)
}

func TestCachedStorage_InvalidateAll(t *testing.T) {
	storage := NewMemoryStorage()
	cached := NewCachedStorage(storage, DefaultCacheConfig())
	defer cached.Close()

	ctx := context.Background()

	// Save and cache multiple
	for i := 0; i < 5; i++ {
		_ = storage.Save(ctx, &StoredTemplate{Name: "tmpl" + intToStr(i)})
		_, _ = cached.Get(ctx, "tmpl"+intToStr(i))
	}

	stats := cached.Stats()
	assert.Equal(t, 5, stats.ValidEntries)

	// Invalidate all
	cached.InvalidateAll()

	stats = cached.Stats()
	assert.Equal(t, 0, stats.Entries)
}

func TestCachedStorage_Save(t *testing.T) {
	storage := NewMemoryStorage()
	cached := NewCachedStorage(storage, DefaultCacheConfig())
	defer cached.Close()

	ctx := context.Background()

	// Cache a template
	_ = storage.Save(ctx, &StoredTemplate{Name: "test", Source: "original"})
	_, _ = cached.Get(ctx, "test")

	// Save through cached storage should invalidate
	err := cached.Save(ctx, &StoredTemplate{Name: "test", Source: "updated"})
	require.NoError(t, err)

	// Next get should return new version
	tmpl, err := cached.Get(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, "updated", tmpl.Source)
}

func TestCachedStorage_Delete(t *testing.T) {
	storage := NewMemoryStorage()
	cached := NewCachedStorage(storage, DefaultCacheConfig())
	defer cached.Close()

	ctx := context.Background()

	// Save and cache
	_ = storage.Save(ctx, &StoredTemplate{Name: "test"})
	_, _ = cached.Get(ctx, "test")

	// Delete should invalidate cache
	err := cached.Delete(ctx, "test")
	require.NoError(t, err)

	// Should now return not found
	_, err = cached.Get(ctx, "test")
	assert.Error(t, err)
}

func TestCachedStorage_Exists(t *testing.T) {
	storage := NewMemoryStorage()
	cached := NewCachedStorage(storage, CacheConfig{
		TTL:              1 * time.Hour,
		MaxEntries:       100,
		NegativeCacheTTL: 1 * time.Hour,
	})
	defer cached.Close()

	ctx := context.Background()

	_ = storage.Save(ctx, &StoredTemplate{Name: "exists"})

	t.Run("uses cache for existing", func(t *testing.T) {
		// Populate cache
		_, _ = cached.Get(ctx, "exists")

		exists, err := cached.Exists(ctx, "exists")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("uses cache for not found", func(t *testing.T) {
		// Populate negative cache
		_, _ = cached.Get(ctx, "notfound")

		exists, err := cached.Exists(ctx, "notfound")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestCachedStorage_GetByID(t *testing.T) {
	storage := NewMemoryStorage()
	cached := NewCachedStorage(storage, DefaultCacheConfig())
	defer cached.Close()

	ctx := context.Background()

	tmpl := &StoredTemplate{Name: "test", Source: "content"}
	_ = storage.Save(ctx, tmpl)

	// Populate cache via Get
	_, _ = cached.Get(ctx, "test")

	// GetByID should use cached entry
	result, err := cached.GetByID(ctx, tmpl.ID)
	require.NoError(t, err)
	assert.Equal(t, "content", result.Source)
}

func TestCachedStorage_Close(t *testing.T) {
	storage := NewMemoryStorage()
	cached := NewCachedStorage(storage, DefaultCacheConfig())

	ctx := context.Background()
	_ = cached.Save(ctx, &StoredTemplate{Name: "test"})

	err := cached.Close()
	require.NoError(t, err)
	assert.True(t, cached.closed)

	// Operations should fail after close
	_, err = cached.Get(ctx, "test")
	assert.Error(t, err)
}

func TestCachedStorage_Stats(t *testing.T) {
	storage := NewMemoryStorage()
	cached := NewCachedStorage(storage, CacheConfig{
		TTL:              1 * time.Hour,
		MaxEntries:       100,
		NegativeCacheTTL: 1 * time.Hour,
	})
	defer cached.Close()

	ctx := context.Background()

	// Add positive entries
	_ = storage.Save(ctx, &StoredTemplate{Name: "tmpl1"})
	_ = storage.Save(ctx, &StoredTemplate{Name: "tmpl2"})
	_, _ = cached.Get(ctx, "tmpl1")
	_, _ = cached.Get(ctx, "tmpl2")

	// Add negative entry
	_, _ = cached.Get(ctx, "notfound")

	stats := cached.Stats()
	assert.Equal(t, 3, stats.Entries)
	assert.Equal(t, 2, stats.ValidEntries)
	assert.Equal(t, 1, stats.NegativeEntries)
}

func TestCachedStorage_DefaultConfig(t *testing.T) {
	config := DefaultCacheConfig()

	assert.Equal(t, 5*time.Minute, config.TTL)
	assert.Equal(t, 1000, config.MaxEntries)
	assert.Equal(t, 30*time.Second, config.NegativeCacheTTL)
}

func TestCachedStorage_PassThrough(t *testing.T) {
	storage := NewMemoryStorage()
	cached := NewCachedStorage(storage, DefaultCacheConfig())
	defer cached.Close()

	ctx := context.Background()

	// These operations pass through without caching
	_ = storage.Save(ctx, &StoredTemplate{Name: "test"})
	_ = storage.Save(ctx, &StoredTemplate{Name: "test"})

	t.Run("GetVersion bypasses cache", func(t *testing.T) {
		tmpl, err := cached.GetVersion(ctx, "test", 1)
		require.NoError(t, err)
		assert.Equal(t, 1, tmpl.Version)
	})

	t.Run("List bypasses cache", func(t *testing.T) {
		results, err := cached.List(ctx, nil)
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("ListVersions bypasses cache", func(t *testing.T) {
		versions, err := cached.ListVersions(ctx, "test")
		require.NoError(t, err)
		assert.Equal(t, []int{2, 1}, versions)
	})
}
