package prompty

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTempFilesystemStorage creates a temporary filesystem storage for testing.
func createTempFilesystemStorage(t *testing.T) (*FilesystemStorage, string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "prompty-e2e-*")
	require.NoError(t, err)

	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)

	cleanup := func() {
		if storage != nil {
			_ = storage.Close()
		}
		os.RemoveAll(dir)
	}

	return storage, dir, cleanup
}

func TestStorageEngine_FilesystemStorage_BasicOperations(t *testing.T) {
	ctx := context.Background()
	storage, dir, cleanup := createTempFilesystemStorage(t)
	defer cleanup()

	// Create engine with filesystem storage
	engine, err := NewStorageEngine(StorageEngineConfig{
		Storage: storage,
	})
	require.NoError(t, err)
	defer engine.Close()

	t.Run("save and retrieve template", func(t *testing.T) {
		tmpl := &StoredTemplate{
			Name:   "greeting",
			Source: `Hello {~prompty.var name="user" default="World" /~}!`,
			Tags:   []string{"production", "user-facing"},
			Metadata: map[string]string{
				"author": "test",
			},
		}

		err := engine.Save(ctx, tmpl)
		require.NoError(t, err)
		assert.NotEmpty(t, tmpl.ID)
		assert.Equal(t, 1, tmpl.Version)

		// Retrieve it
		retrieved, err := engine.Get(ctx, "greeting")
		require.NoError(t, err)
		assert.Equal(t, tmpl.Source, retrieved.Source)
		assert.Equal(t, tmpl.Tags, retrieved.Tags)
		assert.Equal(t, "test", retrieved.Metadata["author"])
	})

	t.Run("execute template", func(t *testing.T) {
		result, err := engine.Execute(ctx, "greeting", map[string]any{
			"user": "Alice",
		})
		require.NoError(t, err)
		assert.Equal(t, "Hello Alice!", result)
	})

	t.Run("execute with default", func(t *testing.T) {
		result, err := engine.Execute(ctx, "greeting", nil)
		require.NoError(t, err)
		assert.Equal(t, "Hello World!", result)
	})

	t.Run("files created on disk", func(t *testing.T) {
		// Check that files were actually written
		files, err := filepath.Glob(filepath.Join(dir, "*", "*.json"))
		require.NoError(t, err)
		assert.NotEmpty(t, files, "expected template files on disk")
	})

	t.Run("update creates new version", func(t *testing.T) {
		tmpl := &StoredTemplate{
			Name:   "greeting",
			Source: `Hi {~prompty.var name="user" default="there" /~}!`,
			Tags:   []string{"production", "v2"},
		}

		err := engine.Save(ctx, tmpl)
		require.NoError(t, err)
		assert.Equal(t, 2, tmpl.Version)

		// Verify new version
		retrieved, err := engine.Get(ctx, "greeting")
		require.NoError(t, err)
		assert.Equal(t, 2, retrieved.Version)
		assert.Contains(t, retrieved.Source, "Hi")
	})

	t.Run("list templates", func(t *testing.T) {
		templates, err := engine.List(ctx, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, templates)

		found := false
		for _, tmpl := range templates {
			if tmpl.Name == "greeting" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected to find 'greeting' template in list")
	})

	t.Run("delete template", func(t *testing.T) {
		err := engine.Save(ctx, &StoredTemplate{
			Name:   "to-delete",
			Source: "test",
		})
		require.NoError(t, err)

		err = engine.Delete(ctx, "to-delete")
		require.NoError(t, err)

		_, err = engine.Get(ctx, "to-delete")
		assert.Error(t, err)
	})
}

func TestStorageEngine_FilesystemStorage_Persistence(t *testing.T) {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "prompty-persistence-*")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	// Phase 1: Create templates
	t.Run("phase1_create_templates", func(t *testing.T) {
		storage, err := NewFilesystemStorage(dir)
		require.NoError(t, err)

		engine, err := NewStorageEngine(StorageEngineConfig{
			Storage: storage,
		})
		require.NoError(t, err)

		// Save multiple templates
		templates := []struct {
			name   string
			source string
			tags   []string
		}{
			{"header", "=== {~prompty.var name=\"title\" /~} ===", []string{"common"}},
			{"footer", "---\nCopyright 2024", []string{"common"}},
			{"greeting", "Hello {~prompty.var name=\"name\" /~}!", []string{"user"}},
		}

		for _, tmpl := range templates {
			err := engine.Save(ctx, &StoredTemplate{
				Name:   tmpl.name,
				Source: tmpl.source,
				Tags:   tmpl.tags,
			})
			require.NoError(t, err)
		}

		// Update one template to create version 2
		err = engine.Save(ctx, &StoredTemplate{
			Name:   "greeting",
			Source: "Hi {~prompty.var name=\"name\" /~}!",
			Tags:   []string{"user", "v2"},
		})
		require.NoError(t, err)

		engine.Close()
	})

	// Phase 2: Reopen and verify persistence
	t.Run("phase2_verify_persistence", func(t *testing.T) {
		storage, err := NewFilesystemStorage(dir)
		require.NoError(t, err)

		engine, err := NewStorageEngine(StorageEngineConfig{
			Storage: storage,
		})
		require.NoError(t, err)
		defer engine.Close()

		// Verify all templates exist
		templates := []string{"header", "footer", "greeting"}
		for _, name := range templates {
			tmpl, err := engine.Get(ctx, name)
			require.NoError(t, err, "template %s should exist", name)
			assert.Equal(t, name, tmpl.Name)
		}

		// Verify greeting has version 2
		greeting, err := engine.Get(ctx, "greeting")
		require.NoError(t, err)
		assert.Equal(t, 2, greeting.Version)
		assert.Contains(t, greeting.Source, "Hi")

		// Verify versions are accessible
		versions, err := storage.ListVersions(ctx, "greeting")
		require.NoError(t, err)
		assert.Len(t, versions, 2, "expected 2 versions")

		// Execute templates to verify they work
		result, err := engine.Execute(ctx, "header", map[string]any{"title": "Test"})
		require.NoError(t, err)
		assert.Equal(t, "=== Test ===", result)

		result, err = engine.Execute(ctx, "greeting", map[string]any{"name": "Bob"})
		require.NoError(t, err)
		assert.Equal(t, "Hi Bob!", result)
	})
}

func TestSecureStorageEngine_FilesystemStorage_AccessControl(t *testing.T) {
	ctx := context.Background()
	storage, _, cleanup := createTempFilesystemStorage(t)
	defer cleanup()

	// Create access checkers
	roleChecker := NewRoleChecker().
		WithOperationRoles(OpCreate, "admin", "editor").
		WithOperationRoles(OpUpdate, "admin", "editor").
		WithOperationRoles(OpDelete, "admin").
		WithOperationRoles(OpRead, "admin", "editor", "viewer").
		WithOperationRoles(OpExecute, "admin", "editor", "viewer").
		WithOperationRoles(OpList, "admin", "editor", "viewer")

	checker := MustChainedChecker(roleChecker)

	// Create secure engine
	engine, err := NewSecureStorageEngine(SecureStorageEngineConfig{
		StorageEngineConfig: StorageEngineConfig{
			Storage: storage,
		},
		AccessChecker: checker,
	})
	require.NoError(t, err)
	defer engine.Close()

	// Admin subject
	admin := NewAccessSubject("admin_123").WithRoles("admin")

	// Viewer subject
	viewer := NewAccessSubject("viewer_123").WithRoles("viewer")

	t.Run("admin can create templates", func(t *testing.T) {
		err := engine.SaveSecure(ctx, &StoredTemplate{
			Name:   "admin-template",
			Source: "Admin created this",
		}, admin)
		require.NoError(t, err)
	})

	t.Run("viewer cannot create templates", func(t *testing.T) {
		err := engine.SaveSecure(ctx, &StoredTemplate{
			Name:   "viewer-template",
			Source: "Viewer tries to create",
		}, viewer)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("viewer can execute templates", func(t *testing.T) {
		result, err := engine.ExecuteSecure(ctx, "admin-template", nil, viewer)
		require.NoError(t, err)
		assert.Equal(t, "Admin created this", result)
	})

	t.Run("viewer cannot delete templates", func(t *testing.T) {
		err := engine.DeleteSecure(ctx, "admin-template", viewer)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access denied")
	})

	t.Run("admin can delete templates", func(t *testing.T) {
		err := engine.DeleteSecure(ctx, "admin-template", admin)
		require.NoError(t, err)

		// Verify deleted
		_, err = engine.GetSecure(ctx, "admin-template", admin)
		require.Error(t, err)
	})
}

func TestStorageEngine_FilesystemStorage_CompleteWorkflow(t *testing.T) {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "prompty-workflow-*")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)

	engine, err := NewStorageEngine(StorageEngineConfig{
		Storage: storage,
	})
	require.NoError(t, err)

	t.Run("complete template lifecycle", func(t *testing.T) {
		// Step 1: Create initial template
		err := engine.Save(ctx, &StoredTemplate{
			Name:   "email-template",
			Source: "Dear {~prompty.var name=\"recipient\" /~}, Welcome!",
			Tags:   []string{"email", "onboarding"},
			Metadata: map[string]string{
				"created_by": "developer",
			},
		})
		require.NoError(t, err)

		// Step 2: Execute template
		result, err := engine.Execute(ctx, "email-template", map[string]any{
			"recipient": "John",
		})
		require.NoError(t, err)
		assert.Equal(t, "Dear John, Welcome!", result)

		// Step 3: Update template (creates version 2)
		err = engine.Save(ctx, &StoredTemplate{
			Name:   "email-template",
			Source: "Hi {~prompty.var name=\"recipient\" /~}, Welcome aboard!",
			Tags:   []string{"email", "onboarding", "v2"},
		})
		require.NoError(t, err)

		// Step 4: Verify new version
		tmpl, err := engine.Get(ctx, "email-template")
		require.NoError(t, err)
		assert.Equal(t, 2, tmpl.Version)

		// Step 5: Test rollback
		history, err := engine.GetVersionHistory(ctx, "email-template")
		require.NoError(t, err)
		assert.Equal(t, 2, history.TotalVersions)

		// Rollback to version 1
		_, err = engine.RollbackToVersion(ctx, "email-template", 1)
		require.NoError(t, err)

		// Verify version 3 created from version 1's content
		tmpl, err = engine.Get(ctx, "email-template")
		require.NoError(t, err)
		assert.Equal(t, 3, tmpl.Version)
		assert.Contains(t, tmpl.Source, "Dear")

		// Step 6: Prune old versions (keep only 2)
		pruned, err := engine.PruneOldVersions(ctx, "email-template", 2)
		require.NoError(t, err)
		assert.Equal(t, 1, pruned)

		// Verify only 2 versions remain
		versions, err := storage.ListVersions(ctx, "email-template")
		require.NoError(t, err)
		assert.Len(t, versions, 2)
	})

	// Phase 2: Close and reopen to verify persistence
	engine.Close()

	t.Run("verify persistence after restart", func(t *testing.T) {
		storage2, err := NewFilesystemStorage(dir)
		require.NoError(t, err)

		engine2, err := NewStorageEngine(StorageEngineConfig{
			Storage: storage2,
		})
		require.NoError(t, err)
		defer engine2.Close()

		// Verify template persisted
		tmpl, err := engine2.Get(ctx, "email-template")
		require.NoError(t, err)
		assert.Equal(t, 3, tmpl.Version)

		// Execute to verify it works
		result, err := engine2.Execute(ctx, "email-template", map[string]any{
			"recipient": "Jane",
		})
		require.NoError(t, err)
		assert.Equal(t, "Dear Jane, Welcome!", result)

		// Verify only 2 versions (after prune)
		versions, err := storage2.ListVersions(ctx, "email-template")
		require.NoError(t, err)
		assert.Len(t, versions, 2)
	})
}

func TestStorageEngine_FilesystemStorage_IncludeTemplates(t *testing.T) {
	ctx := context.Background()
	storage, _, cleanup := createTempFilesystemStorage(t)
	defer cleanup()

	engine, err := NewStorageEngine(StorageEngineConfig{
		Storage: storage,
	})
	require.NoError(t, err)
	defer engine.Close()

	// Create component templates - also register them with base engine for include resolution
	err = engine.Save(ctx, &StoredTemplate{
		Name:   "header",
		Source: "=== {~prompty.var name=\"title\" default=\"Untitled\" /~} ===\n",
	})
	require.NoError(t, err)
	engine.Engine().MustRegisterTemplate("header", "=== {~prompty.var name=\"title\" default=\"Untitled\" /~} ===\n")

	err = engine.Save(ctx, &StoredTemplate{
		Name:   "footer",
		Source: "\n---\nEnd of document",
	})
	require.NoError(t, err)
	engine.Engine().MustRegisterTemplate("footer", "\n---\nEnd of document")

	// Create main template that includes others
	err = engine.Save(ctx, &StoredTemplate{
		Name:   "document",
		Source: `{~prompty.include template="header" title="Report" /~}Content here: {~prompty.var name="content" /~}{~prompty.include template="footer" /~}`,
	})
	require.NoError(t, err)

	// Execute with includes
	result, err := engine.Execute(ctx, "document", map[string]any{
		"content": "This is the report content.",
	})
	require.NoError(t, err)

	assert.Contains(t, result, "=== Report ===")
	assert.Contains(t, result, "This is the report content.")
	assert.Contains(t, result, "End of document")
}

func TestStorageEngine_FilesystemStorage_Conditionals(t *testing.T) {
	ctx := context.Background()
	storage, _, cleanup := createTempFilesystemStorage(t)
	defer cleanup()

	engine, err := NewStorageEngine(StorageEngineConfig{
		Storage: storage,
	})
	require.NoError(t, err)
	defer engine.Close()

	// Create template with conditionals
	err = engine.Save(ctx, &StoredTemplate{
		Name: "notification",
		Source: `{~prompty.if eval="priority == 'high'"~}URGENT: {~/prompty.if~}{~prompty.var name="message" /~}{~prompty.if eval="includeFooter"~}

-- Sent by System{~/prompty.if~}`,
	})
	require.NoError(t, err)

	t.Run("high priority", func(t *testing.T) {
		result, err := engine.Execute(ctx, "notification", map[string]any{
			"priority":      "high",
			"message":       "Server is down!",
			"includeFooter": true,
		})
		require.NoError(t, err)
		assert.Contains(t, result, "URGENT:")
		assert.Contains(t, result, "Server is down!")
		assert.Contains(t, result, "Sent by System")
	})

	t.Run("normal priority", func(t *testing.T) {
		result, err := engine.Execute(ctx, "notification", map[string]any{
			"priority":      "normal",
			"message":       "Daily report ready",
			"includeFooter": false,
		})
		require.NoError(t, err)
		assert.NotContains(t, result, "URGENT:")
		assert.Contains(t, result, "Daily report ready")
		assert.NotContains(t, result, "Sent by System")
	})
}

func TestStorageEngine_FilesystemStorage_Loops(t *testing.T) {
	ctx := context.Background()
	storage, _, cleanup := createTempFilesystemStorage(t)
	defer cleanup()

	engine, err := NewStorageEngine(StorageEngineConfig{
		Storage: storage,
	})
	require.NoError(t, err)
	defer engine.Close()

	// Create template with loop
	err = engine.Save(ctx, &StoredTemplate{
		Name: "item-list",
		Source: `Items:{~prompty.for item="item" index="i" in="items"~}
{~prompty.var name="i" /~}. {~prompty.var name="item.name" /~} - ${~prompty.var name="item.price" /~}{~/prompty.for~}`,
	})
	require.NoError(t, err)

	result, err := engine.Execute(ctx, "item-list", map[string]any{
		"items": []map[string]any{
			{"name": "Apple", "price": "1.50"},
			{"name": "Banana", "price": "0.75"},
			{"name": "Orange", "price": "2.00"},
		},
	})
	require.NoError(t, err)

	assert.Contains(t, result, "0. Apple - $1.50")
	assert.Contains(t, result, "1. Banana - $0.75")
	assert.Contains(t, result, "2. Orange - $2.00")
}

func TestStorageEngine_FilesystemStorage_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	storage, _, cleanup := createTempFilesystemStorage(t)
	defer cleanup()

	engine, err := NewStorageEngine(StorageEngineConfig{
		Storage: storage,
	})
	require.NoError(t, err)
	defer engine.Close()

	// Create a template
	err = engine.Save(ctx, &StoredTemplate{
		Name:   "counter",
		Source: "Count: {~prompty.var name=\"n\" /~}",
	})
	require.NoError(t, err)

	// Run concurrent executions
	done := make(chan bool)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(n int) {
			result, err := engine.Execute(ctx, "counter", map[string]any{
				"n": n,
			})
			if err != nil {
				errors <- err
			} else if result != "Count: "+intToStr(n) {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	select {
	case err := <-errors:
		t.Fatalf("concurrent execution error: %v", err)
	default:
		// All good
	}
}

func TestStorageEngine_FilesystemStorage_EngineOptions(t *testing.T) {
	ctx := context.Background()
	storage, _, cleanup := createTempFilesystemStorage(t)
	defer cleanup()

	// Create a pre-configured engine with custom options
	baseEngine, err := New(WithMaxDepth(5))
	require.NoError(t, err)

	// Create storage engine with the pre-configured engine
	engine, err := NewStorageEngine(StorageEngineConfig{
		Storage: storage,
		Engine:  baseEngine,
	})
	require.NoError(t, err)
	defer engine.Close()

	// Create template
	err = engine.Save(ctx, &StoredTemplate{
		Name:   "simple",
		Source: "Hello {~prompty.var name=\"name\" /~}!",
	})
	require.NoError(t, err)

	// Normal execution should work
	result, err := engine.Execute(ctx, "simple", map[string]any{"name": "World"})
	require.NoError(t, err)
	assert.Equal(t, "Hello World!", result)
}

func TestCachedStorageEngine_FilesystemStorage(t *testing.T) {
	ctx := context.Background()
	storage, _, cleanup := createTempFilesystemStorage(t)
	defer cleanup()

	// Create base engine
	baseEngine, err := NewStorageEngine(StorageEngineConfig{
		Storage: storage,
	})
	require.NoError(t, err)

	// Wrap with result caching
	cachedEngine := NewCachedStorageEngine(baseEngine, ResultCacheConfig{
		TTL:        5 * time.Minute,
		MaxEntries: 100,
	})
	defer cachedEngine.Close()

	// Create template
	err = cachedEngine.Save(ctx, &StoredTemplate{
		Name:   "cached-template",
		Source: "Value: {~prompty.var name=\"value\" /~}",
	})
	require.NoError(t, err)

	// First execution - cache miss
	result1, err := cachedEngine.Execute(ctx, "cached-template", map[string]any{"value": "test"})
	require.NoError(t, err)
	assert.Equal(t, "Value: test", result1)

	stats := cachedEngine.CacheStats()
	assert.Equal(t, int64(1), stats.Misses)

	// Second execution - cache hit
	result2, err := cachedEngine.Execute(ctx, "cached-template", map[string]any{"value": "test"})
	require.NoError(t, err)
	assert.Equal(t, "Value: test", result2)

	stats = cachedEngine.CacheStats()
	assert.Equal(t, int64(1), stats.Hits)

	// Verify cache hit rate (computed manually from stats)
	total := stats.Hits + stats.Misses
	var hitRate float64
	if total > 0 {
		hitRate = float64(stats.Hits) / float64(total)
	}
	assert.Equal(t, 0.5, hitRate, "expected 50%% hit rate (1 hit, 1 miss)")
}
