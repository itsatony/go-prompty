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

// -----------------------------------------------------------------------------
// Deployment-Aware Versioning Tests (Labels and Status)
// -----------------------------------------------------------------------------

func TestStorageEngine_SetLabel(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	// Save a template with multiple versions
	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "v1"})
	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "v2"})

	t.Run("sets label on version", func(t *testing.T) {
		err := se.SetLabel(ctx, "test", "staging", 1)
		require.NoError(t, err)

		labels, err := se.ListLabels(ctx, "test")
		require.NoError(t, err)
		assert.Len(t, labels, 1)
		assert.Equal(t, "staging", labels[0].Label)
		assert.Equal(t, 1, labels[0].Version)
	})

	t.Run("updates existing label", func(t *testing.T) {
		err := se.SetLabel(ctx, "test", "staging", 2)
		require.NoError(t, err)

		labels, err := se.ListLabels(ctx, "test")
		require.NoError(t, err)
		assert.Len(t, labels, 1)
		assert.Equal(t, 2, labels[0].Version)
	})

	t.Run("returns error for nonexistent version", func(t *testing.T) {
		err := se.SetLabel(ctx, "test", "canary", 99)
		require.Error(t, err)
	})

	t.Run("returns error when storage doesn't support labels", func(t *testing.T) {
		// Create engine with mock storage that doesn't implement LabelStorage
		mockStorage := &minimalStorage{}
		se2, _ := NewStorageEngine(StorageEngineConfig{Storage: mockStorage})
		defer se2.Close()

		err := se2.SetLabel(ctx, "test", "staging", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "labels")
	})
}

func TestStorageEngine_SetLabelBy(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "v1"})

	t.Run("sets label with assignedBy", func(t *testing.T) {
		err := se.SetLabelBy(ctx, "test", "production", 1, "admin@example.com")
		require.NoError(t, err)

		labels, err := se.ListLabels(ctx, "test")
		require.NoError(t, err)
		require.Len(t, labels, 1)
		assert.Equal(t, "admin@example.com", labels[0].AssignedBy)
	})
}

func TestStorageEngine_RemoveLabel(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "v1"})
	_ = se.SetLabel(ctx, "test", "staging", 1)

	t.Run("removes existing label", func(t *testing.T) {
		err := se.RemoveLabel(ctx, "test", "staging")
		require.NoError(t, err)

		labels, err := se.ListLabels(ctx, "test")
		require.NoError(t, err)
		assert.Len(t, labels, 0)
	})

	t.Run("returns error for nonexistent label", func(t *testing.T) {
		err := se.RemoveLabel(ctx, "test", "nonexistent")
		require.Error(t, err)
	})
}

func TestStorageEngine_GetByLabel(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "version one"})
	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "version two"})
	_ = se.SetLabel(ctx, "test", "production", 1)
	_ = se.SetLabel(ctx, "test", "staging", 2)

	t.Run("retrieves correct version by label", func(t *testing.T) {
		tmpl, err := se.GetByLabel(ctx, "test", "production")
		require.NoError(t, err)
		assert.Equal(t, 1, tmpl.Version)
		assert.Equal(t, "version one", tmpl.Source)

		tmpl, err = se.GetByLabel(ctx, "test", "staging")
		require.NoError(t, err)
		assert.Equal(t, 2, tmpl.Version)
		assert.Equal(t, "version two", tmpl.Source)
	})

	t.Run("returns error for nonexistent label", func(t *testing.T) {
		_, err := se.GetByLabel(ctx, "test", "canary")
		require.Error(t, err)
	})
}

func TestStorageEngine_ExecuteLabeled(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	_ = se.Save(ctx, &StoredTemplate{Name: "greeting", Source: "Hello v1 {~prompty.var name=\"user\" /~}"})
	_ = se.Save(ctx, &StoredTemplate{Name: "greeting", Source: "Hello v2 {~prompty.var name=\"user\" /~}"})
	_ = se.SetLabel(ctx, "greeting", "production", 1)
	_ = se.SetLabel(ctx, "greeting", "staging", 2)

	t.Run("executes production version", func(t *testing.T) {
		result, err := se.ExecuteLabeled(ctx, "greeting", "production", map[string]any{"user": "Alice"})
		require.NoError(t, err)
		assert.Equal(t, "Hello v1 Alice", result)
	})

	t.Run("executes staging version", func(t *testing.T) {
		result, err := se.ExecuteLabeled(ctx, "greeting", "staging", map[string]any{"user": "Bob"})
		require.NoError(t, err)
		assert.Equal(t, "Hello v2 Bob", result)
	})

	t.Run("returns error for nonexistent label", func(t *testing.T) {
		_, err := se.ExecuteLabeled(ctx, "greeting", "canary", nil)
		require.Error(t, err)
	})
}

func TestStorageEngine_ExecuteProduction(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	_ = se.Save(ctx, &StoredTemplate{Name: "greeting", Source: "Production: {~prompty.var name=\"user\" /~}"})
	_ = se.SetLabel(ctx, "greeting", "production", 1)

	t.Run("executes production-labeled version", func(t *testing.T) {
		result, err := se.ExecuteProduction(ctx, "greeting", map[string]any{"user": "VIP"})
		require.NoError(t, err)
		assert.Equal(t, "Production: VIP", result)
	})

	t.Run("returns error when no production label", func(t *testing.T) {
		_ = se.Save(ctx, &StoredTemplate{Name: "no-prod", Source: "content"})
		_, err := se.ExecuteProduction(ctx, "no-prod", nil)
		require.Error(t, err)
	})
}

func TestStorageEngine_PromoteToProduction(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "v1"})
	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "v2"})
	_ = se.SetLabel(ctx, "test", "production", 1)

	t.Run("moves production label to new version", func(t *testing.T) {
		err := se.PromoteToProduction(ctx, "test", 2)
		require.NoError(t, err)

		tmpl, err := se.GetProduction(ctx, "test")
		require.NoError(t, err)
		assert.Equal(t, 2, tmpl.Version)
	})
}

func TestStorageEngine_PromoteToProductionBy(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "v1"})

	t.Run("promotes with promotedBy tracking", func(t *testing.T) {
		err := se.PromoteToProductionBy(ctx, "test", 1, "release-manager")
		require.NoError(t, err)

		labels, err := se.ListLabels(ctx, "test")
		require.NoError(t, err)
		require.Len(t, labels, 1)
		assert.Equal(t, "release-manager", labels[0].AssignedBy)
	})
}

func TestStorageEngine_GetProduction(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "content"})
	_ = se.SetLabel(ctx, "test", "production", 1)

	t.Run("returns production-labeled template", func(t *testing.T) {
		tmpl, err := se.GetProduction(ctx, "test")
		require.NoError(t, err)
		assert.Equal(t, 1, tmpl.Version)
	})

	t.Run("returns error when no production label", func(t *testing.T) {
		_ = se.Save(ctx, &StoredTemplate{Name: "no-prod", Source: "content"})
		_, err := se.GetProduction(ctx, "no-prod")
		require.Error(t, err)
	})
}

func TestStorageEngine_ListLabels(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "v1"})
	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "v2"})
	_ = se.SetLabel(ctx, "test", "production", 1)
	_ = se.SetLabel(ctx, "test", "staging", 2)
	_ = se.SetLabel(ctx, "test", "canary", 2)

	t.Run("lists all labels", func(t *testing.T) {
		labels, err := se.ListLabels(ctx, "test")
		require.NoError(t, err)
		assert.Len(t, labels, 3)

		labelMap := make(map[string]int)
		for _, l := range labels {
			labelMap[l.Label] = l.Version
		}
		assert.Equal(t, 1, labelMap["production"])
		assert.Equal(t, 2, labelMap["staging"])
		assert.Equal(t, 2, labelMap["canary"])
	})

	t.Run("returns empty for template without labels", func(t *testing.T) {
		_ = se.Save(ctx, &StoredTemplate{Name: "no-labels", Source: "content"})
		labels, err := se.ListLabels(ctx, "no-labels")
		require.NoError(t, err)
		assert.Len(t, labels, 0)
	})
}

func TestStorageEngine_GetVersionLabels(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	// Create template with multiple versions
	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "v1"})
	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "v2"})
	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "v3"})

	// Set labels: production->v1, staging->v2, canary->v2
	_ = se.SetLabel(ctx, "test", "production", 1)
	_ = se.SetLabel(ctx, "test", "staging", 2)
	_ = se.SetLabel(ctx, "test", "canary", 2)

	t.Run("returns labels for version with multiple labels", func(t *testing.T) {
		labels, err := se.GetVersionLabels(ctx, "test", 2)
		require.NoError(t, err)
		assert.Len(t, labels, 2)
		assert.Contains(t, labels, "staging")
		assert.Contains(t, labels, "canary")
	})

	t.Run("returns labels for version with single label", func(t *testing.T) {
		labels, err := se.GetVersionLabels(ctx, "test", 1)
		require.NoError(t, err)
		assert.Len(t, labels, 1)
		assert.Contains(t, labels, "production")
	})

	t.Run("returns empty for version without labels", func(t *testing.T) {
		labels, err := se.GetVersionLabels(ctx, "test", 3)
		require.NoError(t, err)
		assert.Len(t, labels, 0)
	})

	t.Run("returns error when storage doesn't support labels", func(t *testing.T) {
		mockStorage := &minimalStorage{}
		se2, _ := NewStorageEngine(StorageEngineConfig{Storage: mockStorage})
		defer se2.Close()

		_, err := se2.GetVersionLabels(ctx, "test", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "labels")
	})
}

func TestStorageEngine_SetStatus(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "content"})

	t.Run("sets status", func(t *testing.T) {
		err := se.SetStatus(ctx, "test", 1, DeploymentStatusDeprecated)
		require.NoError(t, err)

		tmpl, err := se.GetVersion(ctx, "test", 1)
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusDeprecated, tmpl.Status)
	})

	t.Run("returns error for invalid status", func(t *testing.T) {
		err := se.SetStatus(ctx, "test", 1, DeploymentStatus("invalid"))
		require.Error(t, err)
	})

	t.Run("returns error when storage doesn't support status", func(t *testing.T) {
		mockStorage := &minimalStorage{}
		se2, _ := NewStorageEngine(StorageEngineConfig{Storage: mockStorage})
		defer se2.Close()

		err := se2.SetStatus(ctx, "test", 1, DeploymentStatusActive)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status")
	})
}

func TestStorageEngine_SetStatusBy(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "content"})

	t.Run("sets status with changedBy", func(t *testing.T) {
		err := se.SetStatusBy(ctx, "test", 1, DeploymentStatusDeprecated, "admin")
		require.NoError(t, err)

		tmpl, err := se.GetVersion(ctx, "test", 1)
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusDeprecated, tmpl.Status)
	})
}

func TestStorageEngine_ListByStatus(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	// Create templates with different statuses
	_ = se.Save(ctx, &StoredTemplate{Name: "active1", Source: "content"})
	_ = se.Save(ctx, &StoredTemplate{Name: "active2", Source: "content"})
	_ = se.Save(ctx, &StoredTemplate{Name: "deprecated", Source: "content"})
	_ = se.SetStatus(ctx, "deprecated", 1, DeploymentStatusDeprecated)

	t.Run("lists active templates", func(t *testing.T) {
		active, err := se.ListByStatus(ctx, DeploymentStatusActive, nil)
		require.NoError(t, err)
		assert.Len(t, active, 2)
	})

	t.Run("lists deprecated templates", func(t *testing.T) {
		deprecated, err := se.ListByStatus(ctx, DeploymentStatusDeprecated, nil)
		require.NoError(t, err)
		assert.Len(t, deprecated, 1)
		assert.Equal(t, "deprecated", deprecated[0].Name)
	})

	t.Run("returns empty for unused status", func(t *testing.T) {
		archived, err := se.ListByStatus(ctx, DeploymentStatusArchived, nil)
		require.NoError(t, err)
		assert.Len(t, archived, 0)
	})
}

func TestStorageEngine_SupportsLabels(t *testing.T) {
	t.Run("returns true for memory storage", func(t *testing.T) {
		se, _ := NewStorageEngine(StorageEngineConfig{Storage: NewMemoryStorage()})
		defer se.Close()
		assert.True(t, se.SupportsLabels())
	})

	t.Run("returns false for minimal storage", func(t *testing.T) {
		se, _ := NewStorageEngine(StorageEngineConfig{Storage: &minimalStorage{}})
		defer se.Close()
		assert.False(t, se.SupportsLabels())
	})
}

func TestStorageEngine_SupportsStatus(t *testing.T) {
	t.Run("returns true for memory storage", func(t *testing.T) {
		se, _ := NewStorageEngine(StorageEngineConfig{Storage: NewMemoryStorage()})
		defer se.Close()
		assert.True(t, se.SupportsStatus())
	})

	t.Run("returns false for minimal storage", func(t *testing.T) {
		se, _ := NewStorageEngine(StorageEngineConfig{Storage: &minimalStorage{}})
		defer se.Close()
		assert.False(t, se.SupportsStatus())
	})
}

func TestStorageEngine_PromoteToStaging(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()
	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "v1"})
	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "v2"})

	t.Run("sets staging label on version", func(t *testing.T) {
		err := se.PromoteToStaging(ctx, "test", 2)
		require.NoError(t, err)

		tmpl, err := se.GetByLabel(ctx, "test", LabelStaging)
		require.NoError(t, err)
		assert.Equal(t, 2, tmpl.Version)
	})

	t.Run("returns error for non-existent version", func(t *testing.T) {
		err := se.PromoteToStaging(ctx, "test", 99)
		assert.Error(t, err)
	})
}

func TestStorageEngine_ExecuteStaging(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()
	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "Hello {~prompty.var name=\"name\" /~}"})
	_ = se.SetLabel(ctx, "test", LabelStaging, 1)

	t.Run("executes staging labeled version", func(t *testing.T) {
		result, err := se.ExecuteStaging(ctx, "test", map[string]any{"name": "World"})
		require.NoError(t, err)
		assert.Equal(t, "Hello World", result)
	})

	t.Run("returns error when staging label not set", func(t *testing.T) {
		_, err := se.ExecuteStaging(ctx, "other", nil)
		assert.Error(t, err)
	})
}

func TestStorageEngine_GetActiveTemplates(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()

	// Create templates with different statuses
	_ = se.Save(ctx, &StoredTemplate{Name: "active1", Source: "v1"})
	_ = se.Save(ctx, &StoredTemplate{Name: "active2", Source: "v1"})
	_ = se.Save(ctx, &StoredTemplate{Name: "draft1", Source: "v1", Status: DeploymentStatusDraft})

	t.Run("returns only active templates", func(t *testing.T) {
		templates, err := se.GetActiveTemplates(ctx, nil)
		require.NoError(t, err)
		assert.Len(t, templates, 2)

		names := make([]string, len(templates))
		for i, t := range templates {
			names[i] = t.Name
		}
		assert.Contains(t, names, "active1")
		assert.Contains(t, names, "active2")
		assert.NotContains(t, names, "draft1")
	})
}

func TestStorageEngine_ArchiveVersion(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()
	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "v1"})

	t.Run("sets status to archived", func(t *testing.T) {
		err := se.ArchiveVersion(ctx, "test", 1)
		require.NoError(t, err)

		tmpl, err := se.Get(ctx, "test")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusArchived, tmpl.Status)
	})

	t.Run("returns error for non-existent version", func(t *testing.T) {
		err := se.ArchiveVersion(ctx, "test", 99)
		assert.Error(t, err)
	})
}

func TestStorageEngine_DeprecateVersion(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()
	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "v1"})

	t.Run("sets status to deprecated", func(t *testing.T) {
		err := se.DeprecateVersion(ctx, "test", 1)
		require.NoError(t, err)

		tmpl, err := se.Get(ctx, "test")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusDeprecated, tmpl.Status)
	})

	t.Run("returns error for non-existent version", func(t *testing.T) {
		err := se.DeprecateVersion(ctx, "test", 99)
		assert.Error(t, err)
	})
}

func TestStorageEngine_ActivateVersion(t *testing.T) {
	storage := NewMemoryStorage()
	se, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer se.Close()

	ctx := context.Background()
	_ = se.Save(ctx, &StoredTemplate{Name: "test", Source: "v1", Status: DeploymentStatusDraft})

	t.Run("sets status to active", func(t *testing.T) {
		// First verify it's in draft status
		tmpl, err := se.Get(ctx, "test")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusDraft, tmpl.Status)

		err = se.ActivateVersion(ctx, "test", 1)
		require.NoError(t, err)

		tmpl, err = se.Get(ctx, "test")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusActive, tmpl.Status)
	})

	t.Run("returns error for non-existent version", func(t *testing.T) {
		err := se.ActivateVersion(ctx, "test", 99)
		assert.Error(t, err)
	})
}

// minimalStorage implements only TemplateStorage, not LabelStorage or StatusStorage
type minimalStorage struct{}

func (s *minimalStorage) Get(ctx context.Context, name string) (*StoredTemplate, error) {
	return nil, NewStorageTemplateNotFoundError(name)
}
func (s *minimalStorage) GetByID(ctx context.Context, id TemplateID) (*StoredTemplate, error) {
	return nil, NewStorageTemplateNotFoundError(string(id))
}
func (s *minimalStorage) GetVersion(ctx context.Context, name string, version int) (*StoredTemplate, error) {
	return nil, NewStorageVersionNotFoundError(name, version)
}
func (s *minimalStorage) Save(ctx context.Context, tmpl *StoredTemplate) error { return nil }
func (s *minimalStorage) Delete(ctx context.Context, name string) error        { return nil }
func (s *minimalStorage) DeleteVersion(ctx context.Context, name string, version int) error {
	return nil
}
func (s *minimalStorage) List(ctx context.Context, query *TemplateQuery) ([]*StoredTemplate, error) {
	return nil, nil
}
func (s *minimalStorage) Exists(ctx context.Context, name string) (bool, error) { return false, nil }
func (s *minimalStorage) ListVersions(ctx context.Context, name string) ([]int, error) {
	return nil, nil
}
func (s *minimalStorage) Close() error { return nil }

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
