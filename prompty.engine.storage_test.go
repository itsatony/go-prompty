package prompty

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageEngine_NewStorageEngine(t *testing.T) {
	t.Run("creates with default engine", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, err := NewStorageEngine(StorageEngineConfig{
			Storage: storage,
		})
		require.NoError(t, err)
		require.NotNil(t, se)
		defer se.Close()

		assert.NotNil(t, se.Engine())
		assert.Equal(t, storage, se.Storage())
	})

	t.Run("creates with custom engine", func(t *testing.T) {
		storage := NewMemoryStorage()
		engine, _ := New()

		se, err := NewStorageEngine(StorageEngineConfig{
			Storage: storage,
			Engine:  engine,
		})
		require.NoError(t, err)
		require.NotNil(t, se)
		defer se.Close()

		assert.Equal(t, engine, se.Engine())
	})

	t.Run("rejects nil storage", func(t *testing.T) {
		_, err := NewStorageEngine(StorageEngineConfig{
			Storage: nil,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("enables caching by default", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, err := NewStorageEngine(StorageEngineConfig{
			Storage: storage,
		})
		require.NoError(t, err)
		defer se.Close()

		stats := se.ParsedCacheStats()
		assert.True(t, stats.Enabled)
	})

	t.Run("respects cache disabled config", func(t *testing.T) {
		storage := NewMemoryStorage()
		se, err := NewStorageEngine(StorageEngineConfig{
			Storage:                    storage,
			DisableParsedTemplateCache: true,
		})
		require.NoError(t, err)
		defer se.Close()

		stats := se.ParsedCacheStats()
		assert.False(t, stats.Enabled)
	})
}

func TestStorageEngine_MustNewStorageEngine(t *testing.T) {
	t.Run("succeeds with valid config", func(t *testing.T) {
		storage := NewMemoryStorage()
		se := MustNewStorageEngine(StorageEngineConfig{Storage: storage})
		require.NotNil(t, se)
		defer se.Close()
	})

	t.Run("panics with nil storage", func(t *testing.T) {
		assert.Panics(t, func() {
			MustNewStorageEngine(StorageEngineConfig{Storage: nil})
		})
	})
}

func TestStorageEngine_Execute(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	// Save a template
	err = se.Save(ctx, &StoredTemplate{
		Name:   "greeting",
		Source: "Hello, {~prompty.var name=\"user\" default=\"World\" /~}!",
	})
	require.NoError(t, err)

	t.Run("executes stored template", func(t *testing.T) {
		result, err := se.Execute(ctx, "greeting", map[string]any{
			"user": "Alice",
		})
		require.NoError(t, err)
		assert.Equal(t, "Hello, Alice!", result)
	})

	t.Run("uses default value when data missing", func(t *testing.T) {
		result, err := se.Execute(ctx, "greeting", nil)
		require.NoError(t, err)
		assert.Equal(t, "Hello, World!", result)
	})

	t.Run("returns error for nonexistent template", func(t *testing.T) {
		_, err := se.Execute(ctx, "nonexistent", nil)
		require.Error(t, err)
	})
}

func TestStorageEngine_ExecuteVersion(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	// Save multiple versions
	_ = se.Save(ctx, &StoredTemplate{Name: "versioned", Source: "Version 1"})
	_ = se.Save(ctx, &StoredTemplate{Name: "versioned", Source: "Version 2"})

	t.Run("executes specific version", func(t *testing.T) {
		result, err := se.ExecuteVersion(ctx, "versioned", 1, nil)
		require.NoError(t, err)
		assert.Equal(t, "Version 1", result)

		result, err = se.ExecuteVersion(ctx, "versioned", 2, nil)
		require.NoError(t, err)
		assert.Equal(t, "Version 2", result)
	})

	t.Run("returns error for nonexistent version", func(t *testing.T) {
		_, err := se.ExecuteVersion(ctx, "versioned", 99, nil)
		require.Error(t, err)
	})
}

func TestStorageEngine_ExecuteWithContext(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	_ = se.Save(ctx, &StoredTemplate{
		Name:   "ctx-test",
		Source: "{~prompty.var name=\"value\" /~}",
	})

	t.Run("executes with pre-built context", func(t *testing.T) {
		execCtx := NewContext(map[string]any{"value": "from context"})
		result, err := se.ExecuteWithContext(ctx, "ctx-test", execCtx)
		require.NoError(t, err)
		assert.Equal(t, "from context", result)
	})
}

func TestStorageEngine_Validate(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	t.Run("validates valid template", func(t *testing.T) {
		_ = storage.Save(ctx, &StoredTemplate{
			Name:   "valid",
			Source: "Hello {~prompty.var name=\"x\" /~}",
		})

		result, err := se.Validate(ctx, "valid")
		require.NoError(t, err)
		assert.True(t, result.IsValid())
	})

	t.Run("validates invalid template", func(t *testing.T) {
		_ = storage.Save(ctx, &StoredTemplate{
			Name:   "invalid",
			Source: "Hello {~prompty.var /~}", // missing required name attribute
		})

		result, err := se.Validate(ctx, "invalid")
		require.NoError(t, err)
		assert.False(t, result.IsValid())
	})

	t.Run("returns error for nonexistent template", func(t *testing.T) {
		_, err := se.Validate(ctx, "nonexistent")
		require.Error(t, err)
	})
}

func TestStorageEngine_ValidateVersion(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	_ = storage.Save(ctx, &StoredTemplate{Name: "versioned", Source: "v1"})
	_ = storage.Save(ctx, &StoredTemplate{Name: "versioned", Source: "v2"})

	t.Run("validates specific version", func(t *testing.T) {
		result, err := se.ValidateVersion(ctx, "versioned", 1)
		require.NoError(t, err)
		assert.True(t, result.IsValid())
	})

	t.Run("returns error for nonexistent version", func(t *testing.T) {
		_, err := se.ValidateVersion(ctx, "versioned", 99)
		require.Error(t, err)
	})
}

func TestStorageEngine_Save(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	t.Run("validates before saving", func(t *testing.T) {
		err := se.Save(ctx, &StoredTemplate{
			Name:   "valid",
			Source: "Hello World",
		})
		require.NoError(t, err)

		// Verify saved
		exists, _ := se.Exists(ctx, "valid")
		assert.True(t, exists)
	})

	t.Run("rejects invalid template", func(t *testing.T) {
		err := se.Save(ctx, &StoredTemplate{
			Name:   "invalid",
			Source: "{~prompty.if~}", // unclosed if
		})
		require.Error(t, err)

		// Verify not saved
		exists, _ := se.Exists(ctx, "invalid")
		assert.False(t, exists)
	})

	t.Run("invalidates parsed cache on save", func(t *testing.T) {
		// Save and execute to populate cache
		_ = se.Save(ctx, &StoredTemplate{Name: "cached", Source: "original"})
		_, _ = se.Execute(ctx, "cached", nil)

		stats := se.ParsedCacheStats()
		assert.Equal(t, 1, stats.Entries)

		// Save new version - should invalidate
		_ = se.Save(ctx, &StoredTemplate{Name: "cached", Source: "updated"})

		// Execute should get new version
		result, err := se.Execute(ctx, "cached", nil)
		require.NoError(t, err)
		assert.Equal(t, "updated", result)
	})
}

func TestStorageEngine_SaveWithoutValidation(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	t.Run("saves without validation", func(t *testing.T) {
		err := se.SaveWithoutValidation(ctx, &StoredTemplate{
			Name:   "unvalidated",
			Source: "{~prompty.if~}", // would fail validation
		})
		require.NoError(t, err)

		// Verify saved
		exists, _ := se.Exists(ctx, "unvalidated")
		assert.True(t, exists)
	})
}

func TestStorageEngine_Delete(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	// Save and cache
	_ = se.Save(ctx, &StoredTemplate{Name: "deleteme", Source: "content"})
	_, _ = se.Execute(ctx, "deleteme", nil)

	t.Run("deletes and invalidates cache", func(t *testing.T) {
		err := se.Delete(ctx, "deleteme")
		require.NoError(t, err)

		exists, _ := se.Exists(ctx, "deleteme")
		assert.False(t, exists)
	})

	t.Run("returns error for nonexistent", func(t *testing.T) {
		err := se.Delete(ctx, "nonexistent")
		require.Error(t, err)
	})
}

func TestStorageEngine_DeleteVersion(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	_ = se.Save(ctx, &StoredTemplate{Name: "multi", Source: "v1"})
	_ = se.Save(ctx, &StoredTemplate{Name: "multi", Source: "v2"})

	t.Run("deletes specific version", func(t *testing.T) {
		err := se.DeleteVersion(ctx, "multi", 1)
		require.NoError(t, err)

		versions, _ := se.ListVersions(ctx, "multi")
		assert.Equal(t, []int{2}, versions)
	})
}

func TestStorageEngine_StorageOperations(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "content"})

	t.Run("Get returns template", func(t *testing.T) {
		tmpl, err := se.Get(ctx, "test")
		require.NoError(t, err)
		assert.Equal(t, "content", tmpl.Source)
	})

	t.Run("GetVersion returns specific version", func(t *testing.T) {
		tmpl, err := se.GetVersion(ctx, "test", 1)
		require.NoError(t, err)
		assert.Equal(t, 1, tmpl.Version)
	})

	t.Run("Exists returns true", func(t *testing.T) {
		exists, err := se.Exists(ctx, "test")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("List returns templates", func(t *testing.T) {
		results, err := se.List(ctx, nil)
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("ListVersions returns versions", func(t *testing.T) {
		versions, err := se.ListVersions(ctx, "test")
		require.NoError(t, err)
		assert.Equal(t, []int{1}, versions)
	})
}

func TestStorageEngine_ParsedCache(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{
		Storage: storage,
		// Caching is enabled by default
	})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	_ = se.Save(ctx, &StoredTemplate{Name: "cache1", Source: "content1"})
	_ = se.Save(ctx, &StoredTemplate{Name: "cache2", Source: "content2"})

	t.Run("caches parsed templates", func(t *testing.T) {
		_, _ = se.Execute(ctx, "cache1", nil)
		_, _ = se.Execute(ctx, "cache2", nil)

		stats := se.ParsedCacheStats()
		assert.Equal(t, 2, stats.Entries)
	})

	t.Run("uses cached template on re-execution", func(t *testing.T) {
		// Execute multiple times
		for i := 0; i < 5; i++ {
			_, _ = se.Execute(ctx, "cache1", nil)
		}

		// Should still only have 2 entries
		stats := se.ParsedCacheStats()
		assert.Equal(t, 2, stats.Entries)
	})

	t.Run("ClearParsedCache clears all", func(t *testing.T) {
		se.ClearParsedCache()

		stats := se.ParsedCacheStats()
		assert.Equal(t, 0, stats.Entries)
	})
}

func TestStorageEngine_CacheDisabled(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{
		Storage:                    storage,
		DisableParsedTemplateCache: true,
	})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	_ = se.Save(ctx, &StoredTemplate{Name: "nocache", Source: "content"})

	// Execute multiple times
	for i := 0; i < 5; i++ {
		_, _ = se.Execute(ctx, "nocache", nil)
	}

	stats := se.ParsedCacheStats()
	assert.Equal(t, 0, stats.Entries)
	assert.False(t, stats.Enabled)
}

func TestStorageEngine_RegisterResolver(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	t.Run("registers resolver", func(t *testing.T) {
		resolver := &testResolver{name: "custom"}
		err := se.RegisterResolver(resolver)
		require.NoError(t, err)

		// Save template using custom resolver
		_ = se.Save(ctx, &StoredTemplate{
			Name:   "with-resolver",
			Source: "{~custom /~}",
		})

		result, err := se.Execute(ctx, "with-resolver", nil)
		require.NoError(t, err)
		assert.Equal(t, "resolved", result)
	})
}

func TestStorageEngine_MustRegisterResolver(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	t.Run("registers resolver without error", func(t *testing.T) {
		assert.NotPanics(t, func() {
			se.MustRegisterResolver(&testResolver{name: "must-custom"})
		})
	})
}

func TestStorageEngine_RegisterFunc(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	t.Run("registers function", func(t *testing.T) {
		err := se.RegisterFunc(&Func{
			Name:    "double",
			MinArgs: 1,
			MaxArgs: 1,
			Fn: func(args []any) (any, error) {
				if n, ok := args[0].(int); ok {
					return n * 2, nil
				}
				return nil, nil
			},
		})
		require.NoError(t, err)

		_ = se.Save(ctx, &StoredTemplate{
			Name:   "with-func",
			Source: "{~prompty.var name=\"result\" /~}",
		})

		// Function registered but we'd need an expression to test it
		// Just verify registration succeeded
	})
}

func TestStorageEngine_MustRegisterFunc(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	t.Run("registers function without error", func(t *testing.T) {
		assert.NotPanics(t, func() {
			se.MustRegisterFunc(&Func{
				Name:    "triple",
				MinArgs: 1,
				MaxArgs: 1,
				Fn: func(args []any) (any, error) {
					if n, ok := args[0].(int); ok {
						return n * 3, nil
					}
					return nil, nil
				},
			})
		})
	})
}

func TestStorageEngine_Close(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)

	ctx := context.Background()
	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "content"})
	_, _ = se.Execute(ctx, "test", nil)

	err = se.Close()
	require.NoError(t, err)

	// Verify cache cleared
	stats := se.ParsedCacheStats()
	assert.Equal(t, 0, stats.Entries)
}

func TestStorageEngine_ConcurrentAccess(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	// Pre-save some templates
	for i := 0; i < 5; i++ {
		_ = se.Save(ctx, &StoredTemplate{
			Name:   "concurrent-" + intToStr(i),
			Source: "Content " + intToStr(i),
		})
	}

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Concurrent executions
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := se.Execute(ctx, "concurrent-"+intToStr(id%5), nil)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	// Concurrent saves
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := se.Save(ctx, &StoredTemplate{
				Name:   "new-" + intToStr(id),
				Source: "New content " + intToStr(id),
			})
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent operation failed: %v", err)
	}
}

// testResolver is a simple resolver for testing
type testResolver struct {
	name string
}

func (r *testResolver) TagName() string {
	return r.name
}

func (r *testResolver) Resolve(ctx context.Context, execCtx *Context, attrs Attributes) (string, error) {
	return "resolved", nil
}

func (r *testResolver) Validate(attrs Attributes) error {
	return nil
}
