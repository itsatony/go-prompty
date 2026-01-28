//go:build integration

package prompty

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupPostgresContainer creates an ephemeral PostgreSQL container for testing.
func setupPostgresContainer(t *testing.T) (*PostgresStorage, func()) {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:15",
		postgres.WithDatabase("prompty_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err, "failed to start postgres container")

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	storage, err := NewPostgresStorage(PostgresConfig{
		ConnectionString: connStr,
		AutoMigrate:      true,
		QueryTimeout:     30 * time.Second,
	})
	require.NoError(t, err, "failed to create postgres storage")

	cleanup := func() {
		if storage != nil {
			_ = storage.Close()
		}
		if container != nil {
			_ = container.Terminate(ctx)
		}
	}

	return storage, cleanup
}

// =============================================================================
// Basic CRUD Tests
// =============================================================================

func TestPostgres_E2E_BasicCRUD(t *testing.T) {
	storage, cleanup := setupPostgresContainer(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("Save", func(t *testing.T) {
		tmpl := &StoredTemplate{
			Name:      "test-template",
			Source:    "Hello {~prompty.var name=\"user\" /~}!",
			Metadata:  map[string]any{"author": "test"},
			Tags:      []string{"greeting", "test"},
			TenantID:  "tenant-1",
			CreatedBy: "user-1",
		}

		err := storage.Save(ctx, tmpl)
		require.NoError(t, err)
		assert.NotEmpty(t, tmpl.ID)
		assert.Equal(t, 1, tmpl.Version)
		assert.False(t, tmpl.CreatedAt.IsZero())
		assert.False(t, tmpl.UpdatedAt.IsZero())
	})

	t.Run("Get", func(t *testing.T) {
		tmpl, err := storage.Get(ctx, "test-template")
		require.NoError(t, err)
		assert.Equal(t, "test-template", tmpl.Name)
		assert.Contains(t, tmpl.Source, "prompty.var")
		assert.Equal(t, 1, tmpl.Version)
		assert.Equal(t, "tenant-1", tmpl.TenantID)
		assert.Equal(t, "user-1", tmpl.CreatedBy)
		assert.Contains(t, tmpl.Tags, "greeting")
	})

	t.Run("GetByID", func(t *testing.T) {
		tmpl, err := storage.Get(ctx, "test-template")
		require.NoError(t, err)

		retrieved, err := storage.GetByID(ctx, tmpl.ID)
		require.NoError(t, err)
		assert.Equal(t, tmpl.ID, retrieved.ID)
		assert.Equal(t, tmpl.Name, retrieved.Name)
	})

	t.Run("Exists", func(t *testing.T) {
		exists, err := storage.Exists(ctx, "test-template")
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = storage.Exists(ctx, "nonexistent")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("GetNotFound", func(t *testing.T) {
		_, err := storage.Get(ctx, "nonexistent-template")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Delete", func(t *testing.T) {
		// Save a template to delete
		tmpl := &StoredTemplate{
			Name:   "to-delete",
			Source: "delete me",
		}
		err := storage.Save(ctx, tmpl)
		require.NoError(t, err)

		// Delete it
		err = storage.Delete(ctx, "to-delete")
		require.NoError(t, err)

		// Verify it's gone
		exists, err := storage.Exists(ctx, "to-delete")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("DeleteNotFound", func(t *testing.T) {
		err := storage.Delete(ctx, "nonexistent-template")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// =============================================================================
// Versioning Tests
// =============================================================================

func TestPostgres_E2E_Versioning(t *testing.T) {
	storage, cleanup := setupPostgresContainer(t)
	defer cleanup()
	ctx := context.Background()

	// Save multiple versions
	for i := 1; i <= 5; i++ {
		tmpl := &StoredTemplate{
			Name:   "versioned-template",
			Source: fmt.Sprintf("Version %d content", i),
		}
		err := storage.Save(ctx, tmpl)
		require.NoError(t, err)
		assert.Equal(t, i, tmpl.Version)
	}

	t.Run("GetReturnsLatestVersion", func(t *testing.T) {
		tmpl, err := storage.Get(ctx, "versioned-template")
		require.NoError(t, err)
		assert.Equal(t, 5, tmpl.Version)
		assert.Contains(t, tmpl.Source, "Version 5")
	})

	t.Run("GetVersion", func(t *testing.T) {
		tmpl, err := storage.GetVersion(ctx, "versioned-template", 3)
		require.NoError(t, err)
		assert.Equal(t, 3, tmpl.Version)
		assert.Contains(t, tmpl.Source, "Version 3")
	})

	t.Run("GetVersionNotFound", func(t *testing.T) {
		_, err := storage.GetVersion(ctx, "versioned-template", 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("ListVersions", func(t *testing.T) {
		versions, err := storage.ListVersions(ctx, "versioned-template")
		require.NoError(t, err)
		assert.Len(t, versions, 5)
		// Should be in descending order
		assert.Equal(t, []int{5, 4, 3, 2, 1}, versions)
	})

	t.Run("DeleteVersion", func(t *testing.T) {
		err := storage.DeleteVersion(ctx, "versioned-template", 2)
		require.NoError(t, err)

		versions, err := storage.ListVersions(ctx, "versioned-template")
		require.NoError(t, err)
		assert.Len(t, versions, 4)
		assert.NotContains(t, versions, 2)
	})

	t.Run("DeleteVersionNotFound", func(t *testing.T) {
		err := storage.DeleteVersion(ctx, "versioned-template", 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// =============================================================================
// Concurrent Access Tests
// =============================================================================

func TestPostgres_E2E_ConcurrentSaves(t *testing.T) {
	storage, cleanup := setupPostgresContainer(t)
	defer cleanup()
	ctx := context.Background()

	const numGoroutines = 50
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)
	versionChan := make(chan int, numGoroutines)

	// 50 goroutines all saving the same template name
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			tmpl := &StoredTemplate{
				Name:     "concurrent-template",
				Source:   fmt.Sprintf("Content from goroutine %d", id),
				Metadata: map[string]any{"goroutine": id},
			}

			err := storage.Save(ctx, tmpl)
			if err != nil {
				errChan <- err
				return
			}
			versionChan <- tmpl.Version
		}(i)
	}

	wg.Wait()
	close(errChan)
	close(versionChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	// Collect versions
	var versions []int
	for v := range versionChan {
		versions = append(versions, v)
	}

	// All saves should succeed
	assert.Empty(t, errors, "expected no errors from concurrent saves")

	// All versions should be unique
	versionSet := make(map[int]bool)
	for _, v := range versions {
		assert.False(t, versionSet[v], "duplicate version detected: %d", v)
		versionSet[v] = true
	}

	// Should have 50 unique versions
	assert.Len(t, versionSet, numGoroutines)

	// Verify in database
	dbVersions, err := storage.ListVersions(ctx, "concurrent-template")
	require.NoError(t, err)
	assert.Len(t, dbVersions, numGoroutines)
}

func TestPostgres_E2E_ConcurrentReads(t *testing.T) {
	storage, cleanup := setupPostgresContainer(t)
	defer cleanup()
	ctx := context.Background()

	// Setup: Create a template
	tmpl := &StoredTemplate{
		Name:   "read-test",
		Source: "Read me concurrently",
	}
	err := storage.Save(ctx, tmpl)
	require.NoError(t, err)

	const numGoroutines = 100
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)

	// 100 concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			retrieved, err := storage.Get(ctx, "read-test")
			if err != nil {
				errChan <- err
				return
			}
			if retrieved.Name != "read-test" {
				errChan <- fmt.Errorf("unexpected template name: %s", retrieved.Name)
			}
		}()
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	assert.Empty(t, errors, "expected no errors from concurrent reads")
}

// =============================================================================
// List Filtering Tests
// =============================================================================

func TestPostgres_E2E_ListFiltering(t *testing.T) {
	storage, cleanup := setupPostgresContainer(t)
	defer cleanup()
	ctx := context.Background()

	// Setup: Create test templates
	testTemplates := []struct {
		name      string
		tenantID  string
		createdBy string
		tags      []string
	}{
		{"api/users/get", "tenant-a", "alice", []string{"api", "users"}},
		{"api/users/list", "tenant-a", "alice", []string{"api", "users"}},
		{"api/orders/get", "tenant-a", "bob", []string{"api", "orders"}},
		{"web/home", "tenant-b", "charlie", []string{"web", "public"}},
		{"web/about", "tenant-b", "charlie", []string{"web", "public"}},
		{"internal/admin", "tenant-b", "admin", []string{"internal", "admin"}},
	}

	for _, tt := range testTemplates {
		tmpl := &StoredTemplate{
			Name:      tt.name,
			Source:    "Source for " + tt.name,
			TenantID:  tt.tenantID,
			CreatedBy: tt.createdBy,
			Tags:      tt.tags,
		}
		err := storage.Save(ctx, tmpl)
		require.NoError(t, err)
	}

	t.Run("FilterByTenantID", func(t *testing.T) {
		results, err := storage.List(ctx, &TemplateQuery{
			TenantID: "tenant-a",
		})
		require.NoError(t, err)
		assert.Len(t, results, 3)

		for _, r := range results {
			assert.Equal(t, "tenant-a", r.TenantID)
		}
	})

	t.Run("FilterByCreatedBy", func(t *testing.T) {
		results, err := storage.List(ctx, &TemplateQuery{
			CreatedBy: "alice",
		})
		require.NoError(t, err)
		assert.Len(t, results, 2)

		for _, r := range results {
			assert.Equal(t, "alice", r.CreatedBy)
		}
	})

	t.Run("FilterByNamePrefix", func(t *testing.T) {
		results, err := storage.List(ctx, &TemplateQuery{
			NamePrefix: "api/",
		})
		require.NoError(t, err)
		assert.Len(t, results, 3)

		for _, r := range results {
			assert.True(t, len(r.Name) >= 4 && r.Name[:4] == "api/")
		}
	})

	t.Run("FilterByNameContains", func(t *testing.T) {
		results, err := storage.List(ctx, &TemplateQuery{
			NameContains: "users",
		})
		require.NoError(t, err)
		assert.Len(t, results, 2)

		for _, r := range results {
			assert.Contains(t, r.Name, "users")
		}
	})

	t.Run("FilterByTags_SingleTag", func(t *testing.T) {
		results, err := storage.List(ctx, &TemplateQuery{
			Tags: []string{"api"},
		})
		require.NoError(t, err)
		assert.Len(t, results, 3)

		for _, r := range results {
			assert.Contains(t, r.Tags, "api")
		}
	})

	t.Run("FilterByTags_MultipleTags", func(t *testing.T) {
		results, err := storage.List(ctx, &TemplateQuery{
			Tags: []string{"web", "public"},
		})
		require.NoError(t, err)
		assert.Len(t, results, 2)

		for _, r := range results {
			assert.Contains(t, r.Tags, "web")
			assert.Contains(t, r.Tags, "public")
		}
	})

	t.Run("FilterCombined", func(t *testing.T) {
		results, err := storage.List(ctx, &TemplateQuery{
			TenantID:   "tenant-a",
			NamePrefix: "api/users",
		})
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("Pagination", func(t *testing.T) {
		// Get first page
		page1, err := storage.List(ctx, &TemplateQuery{
			Limit:  2,
			Offset: 0,
		})
		require.NoError(t, err)
		assert.Len(t, page1, 2)

		// Get second page
		page2, err := storage.List(ctx, &TemplateQuery{
			Limit:  2,
			Offset: 2,
		})
		require.NoError(t, err)
		assert.Len(t, page2, 2)

		// Verify no overlap
		page1Names := make(map[string]bool)
		for _, t := range page1 {
			page1Names[t.Name] = true
		}
		for _, t := range page2 {
			assert.False(t, page1Names[t.Name], "pagination overlap detected")
		}
	})

	t.Run("IncludeAllVersions", func(t *testing.T) {
		// Save another version of an existing template
		tmpl := &StoredTemplate{
			Name:     "api/users/get",
			Source:   "Updated source",
			TenantID: "tenant-a",
		}
		err := storage.Save(ctx, tmpl)
		require.NoError(t, err)

		// Without IncludeAllVersions (default)
		results, err := storage.List(ctx, &TemplateQuery{
			NameContains: "api/users/get",
		})
		require.NoError(t, err)
		assert.Len(t, results, 1) // Only latest version

		// With IncludeAllVersions
		results, err = storage.List(ctx, &TemplateQuery{
			NameContains:       "api/users/get",
			IncludeAllVersions: true,
		})
		require.NoError(t, err)
		assert.Len(t, results, 2) // Both versions
	})
}

// =============================================================================
// Migration Tests
// =============================================================================

func TestPostgres_E2E_Migrations(t *testing.T) {
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:15",
		postgres.WithDatabase("prompty_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	defer func() { _ = container.Terminate(ctx) }()

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	t.Run("InitialMigration", func(t *testing.T) {
		// Create storage with auto-migrate
		storage, err := NewPostgresStorage(PostgresConfig{
			ConnectionString: connStr,
			AutoMigrate:      true,
		})
		require.NoError(t, err)
		defer storage.Close()

		// Check schema version
		version, err := storage.CurrentSchemaVersion(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, version)

		// Verify we can save templates
		tmpl := &StoredTemplate{
			Name:   "migration-test",
			Source: "test",
		}
		err = storage.Save(ctx, tmpl)
		require.NoError(t, err)
	})

	t.Run("IdempotentRerun", func(t *testing.T) {
		// Create another storage instance (should be idempotent)
		storage, err := NewPostgresStorage(PostgresConfig{
			ConnectionString: connStr,
			AutoMigrate:      true,
		})
		require.NoError(t, err)
		defer storage.Close()

		// Should still work
		version, err := storage.CurrentSchemaVersion(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, version)

		// Previous data should still exist
		exists, err := storage.Exists(ctx, "migration-test")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("ManualMigration", func(t *testing.T) {
		// Create storage without auto-migrate
		storage, err := NewPostgresStorage(PostgresConfig{
			ConnectionString: connStr,
			AutoMigrate:      false,
		})
		require.NoError(t, err)
		defer storage.Close()

		// Manually run migrations
		err = storage.RunMigrations(ctx)
		require.NoError(t, err)

		// Should be idempotent
		err = storage.RunMigrations(ctx)
		require.NoError(t, err)
	})
}

// =============================================================================
// Connection Pool Tests
// =============================================================================

func TestPostgres_E2E_ConnectionPooling(t *testing.T) {
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:15",
		postgres.WithDatabase("prompty_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	defer func() { _ = container.Terminate(ctx) }()

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	t.Run("CustomPoolConfig", func(t *testing.T) {
		storage, err := NewPostgresStorage(PostgresConfig{
			ConnectionString: connStr,
			MaxOpenConns:     5,
			MaxIdleConns:     2,
			ConnMaxLifetime:  1 * time.Minute,
			ConnMaxIdleTime:  30 * time.Second,
			AutoMigrate:      true,
		})
		require.NoError(t, err)
		defer storage.Close()

		// Should work with limited pool
		tmpl := &StoredTemplate{
			Name:   "pool-test",
			Source: "test",
		}
		err = storage.Save(ctx, tmpl)
		require.NoError(t, err)
	})

	t.Run("HighConcurrencyWithLimitedPool", func(t *testing.T) {
		storage, err := NewPostgresStorage(PostgresConfig{
			ConnectionString: connStr,
			MaxOpenConns:     3, // Very limited pool
			MaxIdleConns:     1,
			AutoMigrate:      false, // Already migrated
		})
		require.NoError(t, err)
		defer storage.Close()

		const numGoroutines = 20
		var wg sync.WaitGroup
		errChan := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// Mix of reads and writes
				if id%2 == 0 {
					tmpl := &StoredTemplate{
						Name:   fmt.Sprintf("pool-high-%d", id),
						Source: "test",
					}
					if err := storage.Save(ctx, tmpl); err != nil {
						errChan <- err
					}
				} else {
					_, err := storage.List(ctx, nil)
					if err != nil {
						errChan <- err
					}
				}
			}(i)
		}

		wg.Wait()
		close(errChan)

		var errors []error
		for err := range errChan {
			errors = append(errors, err)
		}

		assert.Empty(t, errors, "pool should handle high concurrency")
	})

	t.Run("TimeoutBehavior", func(t *testing.T) {
		storage, err := NewPostgresStorage(PostgresConfig{
			ConnectionString: connStr,
			QueryTimeout:     100 * time.Millisecond,
			AutoMigrate:      false,
		})
		require.NoError(t, err)
		defer storage.Close()

		// Normal operation should complete within timeout
		_, err = storage.List(ctx, nil)
		require.NoError(t, err)
	})
}

// =============================================================================
// Large Data Tests
// =============================================================================

func TestPostgres_E2E_LargeData(t *testing.T) {
	storage, cleanup := setupPostgresContainer(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("LargeMetadata", func(t *testing.T) {
		// Create metadata approaching 1MB
		largeMap := make(map[string]any)
		for i := 0; i < 1000; i++ {
			largeMap[fmt.Sprintf("key_%d", i)] = make([]byte, 500) // ~500KB total
		}

		tmpl := &StoredTemplate{
			Name:     "large-metadata",
			Source:   "test",
			Metadata: largeMap,
		}

		err := storage.Save(ctx, tmpl)
		require.NoError(t, err)

		// Retrieve and verify
		retrieved, err := storage.Get(ctx, "large-metadata")
		require.NoError(t, err)
		assert.Len(t, retrieved.Metadata, 1000)
	})

	t.Run("LargeSource", func(t *testing.T) {
		// Create a large template source
		largeSource := ""
		for i := 0; i < 10000; i++ {
			largeSource += fmt.Sprintf("{~prompty.var name=\"var%d\" /~}\n", i)
		}

		tmpl := &StoredTemplate{
			Name:   "large-source",
			Source: largeSource,
		}

		err := storage.Save(ctx, tmpl)
		require.NoError(t, err)

		retrieved, err := storage.Get(ctx, "large-source")
		require.NoError(t, err)
		assert.Equal(t, len(largeSource), len(retrieved.Source))
	})

	t.Run("ManyTemplates", func(t *testing.T) {
		const templateCount = 500

		// Create many templates
		for i := 0; i < templateCount; i++ {
			tmpl := &StoredTemplate{
				Name:   fmt.Sprintf("bulk/template-%04d", i),
				Source: fmt.Sprintf("Content %d", i),
				Tags:   []string{"bulk", fmt.Sprintf("group-%d", i%10)},
			}
			err := storage.Save(ctx, tmpl)
			require.NoError(t, err)
		}

		// List all with prefix
		results, err := storage.List(ctx, &TemplateQuery{
			NamePrefix: "bulk/",
		})
		require.NoError(t, err)
		assert.Len(t, results, templateCount)

		// Test pagination through all
		var allResults []*StoredTemplate
		pageSize := 50
		offset := 0

		for {
			page, err := storage.List(ctx, &TemplateQuery{
				NamePrefix: "bulk/",
				Limit:      pageSize,
				Offset:     offset,
			})
			require.NoError(t, err)

			if len(page) == 0 {
				break
			}

			allResults = append(allResults, page...)
			offset += pageSize
		}

		assert.Len(t, allResults, templateCount)
	})
}

// =============================================================================
// InferenceConfig Persistence Tests
// =============================================================================

func TestPostgres_E2E_InferenceConfigPersistence(t *testing.T) {
	storage, cleanup := setupPostgresContainer(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("SaveAndRetrieveInferenceConfig", func(t *testing.T) {
		cfg := &InferenceConfig{
			API: "anthropic",
			Model: &ModelConfig{
				Provider: "anthropic",
				Name:     "claude-3-opus",
				Parameters: map[string]any{
					"temperature": 0.7,
					"max_tokens":  4096,
					"top_p":       0.9,
				},
			},
			Input: &InputSchema{
				Type: "object",
				Properties: map[string]*PropertySchema{
					"query": {
						Type:        "string",
						Description: "User query",
					},
				},
				Required: []string{"query"},
			},
			Output: &OutputSchema{
				Type:   "string",
				Format: "text",
			},
			Sample: map[string]any{
				"query": "What is the meaning of life?",
			},
		}

		tmpl := &StoredTemplate{
			Name:            "config-test",
			Source:          "Answer: {~prompty.var name=\"response\" /~}",
			InferenceConfig: cfg,
		}

		err := storage.Save(ctx, tmpl)
		require.NoError(t, err)

		// Retrieve and verify
		retrieved, err := storage.Get(ctx, "config-test")
		require.NoError(t, err)
		require.NotNil(t, retrieved.InferenceConfig)

		assert.Equal(t, "anthropic", retrieved.InferenceConfig.API)
		assert.NotNil(t, retrieved.InferenceConfig.Model)
		assert.Equal(t, "claude-3-opus", retrieved.InferenceConfig.Model.Name)
		assert.Equal(t, 0.7, retrieved.InferenceConfig.Model.Parameters["temperature"])
		assert.NotNil(t, retrieved.InferenceConfig.Input)
		assert.Contains(t, retrieved.InferenceConfig.Input.Required, "query")
		assert.NotNil(t, retrieved.InferenceConfig.Output)
		assert.Equal(t, "text", retrieved.InferenceConfig.Output.Format)
	})

	t.Run("UpdateInferenceConfig", func(t *testing.T) {
		// Save new version with updated config
		cfg := &InferenceConfig{
			API: "openai",
			Model: &ModelConfig{
				Provider: "openai",
				Name:     "gpt-4",
				Parameters: map[string]any{
					"temperature": 0.5,
				},
			},
		}

		tmpl := &StoredTemplate{
			Name:            "config-test",
			Source:          "Updated answer: {~prompty.var name=\"response\" /~}",
			InferenceConfig: cfg,
		}

		err := storage.Save(ctx, tmpl)
		require.NoError(t, err)
		assert.Equal(t, 2, tmpl.Version)

		// Verify latest has new config
		latest, err := storage.Get(ctx, "config-test")
		require.NoError(t, err)
		assert.Equal(t, "openai", latest.InferenceConfig.API)

		// Verify old version still has old config
		v1, err := storage.GetVersion(ctx, "config-test", 1)
		require.NoError(t, err)
		assert.Equal(t, "anthropic", v1.InferenceConfig.API)
	})

	t.Run("NilInferenceConfig", func(t *testing.T) {
		tmpl := &StoredTemplate{
			Name:            "no-config",
			Source:          "Simple template",
			InferenceConfig: nil,
		}

		err := storage.Save(ctx, tmpl)
		require.NoError(t, err)

		retrieved, err := storage.Get(ctx, "no-config")
		require.NoError(t, err)
		assert.Nil(t, retrieved.InferenceConfig)
	})
}

// =============================================================================
// Edge Cases and Error Handling
// =============================================================================

func TestPostgres_E2E_EdgeCases(t *testing.T) {
	storage, cleanup := setupPostgresContainer(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("EmptyName", func(t *testing.T) {
		tmpl := &StoredTemplate{
			Name:   "",
			Source: "test",
		}
		err := storage.Save(ctx, tmpl)
		require.Error(t, err)
	})

	t.Run("EmptyTags", func(t *testing.T) {
		tmpl := &StoredTemplate{
			Name:   "empty-tags",
			Source: "test",
			Tags:   []string{},
		}
		err := storage.Save(ctx, tmpl)
		require.NoError(t, err)

		retrieved, err := storage.Get(ctx, "empty-tags")
		require.NoError(t, err)
		assert.Empty(t, retrieved.Tags)
	})

	t.Run("NilMetadata", func(t *testing.T) {
		tmpl := &StoredTemplate{
			Name:     "nil-metadata",
			Source:   "test",
			Metadata: nil,
		}
		err := storage.Save(ctx, tmpl)
		require.NoError(t, err)

		retrieved, err := storage.Get(ctx, "nil-metadata")
		require.NoError(t, err)
		// Should be empty map, not nil
		assert.NotNil(t, retrieved.Metadata)
	})

	t.Run("SpecialCharactersInName", func(t *testing.T) {
		names := []string{
			"template-with-dashes",
			"template_with_underscores",
			"template.with.dots",
			"template/with/slashes",
			"template:with:colons",
		}

		for _, name := range names {
			tmpl := &StoredTemplate{
				Name:   name,
				Source: "test",
			}
			err := storage.Save(ctx, tmpl)
			require.NoError(t, err, "failed to save template with name: %s", name)

			retrieved, err := storage.Get(ctx, name)
			require.NoError(t, err, "failed to get template with name: %s", name)
			assert.Equal(t, name, retrieved.Name)
		}
	})

	t.Run("UnicodeContent", func(t *testing.T) {
		tmpl := &StoredTemplate{
			Name:   "unicode-test",
			Source: "Hello 世界! Привет мир! مرحبا بالعالم 🎉",
			Metadata: map[string]any{
				"greeting": "こんにちは",
			},
			Tags: []string{"日本語", "русский"},
		}
		err := storage.Save(ctx, tmpl)
		require.NoError(t, err)

		retrieved, err := storage.Get(ctx, "unicode-test")
		require.NoError(t, err)
		assert.Contains(t, retrieved.Source, "世界")
		assert.Equal(t, "こんにちは", retrieved.Metadata["greeting"])
		assert.Contains(t, retrieved.Tags, "日本語")
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately

		_, err := storage.Get(cancelCtx, "any-template")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "canceled")
	})

	t.Run("OperationsAfterClose", func(t *testing.T) {
		// Create a new storage just for this test
		container, err := postgres.Run(ctx, "postgres:15",
			postgres.WithDatabase("close_test"),
			postgres.WithUsername("test"),
			postgres.WithPassword("test"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(60*time.Second),
			),
		)
		require.NoError(t, err)
		defer func() { _ = container.Terminate(ctx) }()

		connStr, err := container.ConnectionString(ctx, "sslmode=disable")
		require.NoError(t, err)

		tmpStorage, err := NewPostgresStorage(PostgresConfig{
			ConnectionString: connStr,
			AutoMigrate:      true,
		})
		require.NoError(t, err)

		// Close it
		err = tmpStorage.Close()
		require.NoError(t, err)

		// Operations should fail
		_, err = tmpStorage.Get(ctx, "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "closed")

		err = tmpStorage.Save(ctx, &StoredTemplate{Name: "test", Source: "test"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "closed")

		// Double close should error
		err = tmpStorage.Close()
		require.Error(t, err)
	})
}

// =============================================================================
// Label Storage Tests
// =============================================================================

func TestPostgres_E2E_Labels(t *testing.T) {
	storage, cleanup := setupPostgresContainer(t)
	defer cleanup()
	ctx := context.Background()

	// Create a template
	tmpl := &StoredTemplate{
		Name:   "label-test",
		Source: "Hello {~prompty.var name=\"name\" /~}",
	}
	err := storage.Save(ctx, tmpl)
	require.NoError(t, err)
	assert.Equal(t, 1, tmpl.Version)

	t.Run("SetAndGetLabel", func(t *testing.T) {
		err := storage.SetLabel(ctx, "label-test", "production", 1, "deploy-user")
		require.NoError(t, err)

		got, err := storage.GetByLabel(ctx, "label-test", "production")
		require.NoError(t, err)
		assert.Equal(t, 1, got.Version)
	})

	t.Run("ListLabels", func(t *testing.T) {
		// Add another label
		err := storage.SetLabel(ctx, "label-test", "staging", 1, "deploy-user")
		require.NoError(t, err)

		labels, err := storage.ListLabels(ctx, "label-test")
		require.NoError(t, err)
		assert.Len(t, labels, 2)

		labelNames := make([]string, len(labels))
		for i, l := range labels {
			labelNames[i] = l.Label
		}
		assert.Contains(t, labelNames, "production")
		assert.Contains(t, labelNames, "staging")
	})

	t.Run("GetVersionLabels", func(t *testing.T) {
		versionLabels, err := storage.GetVersionLabels(ctx, "label-test", 1)
		require.NoError(t, err)
		assert.Contains(t, versionLabels, "production")
		assert.Contains(t, versionLabels, "staging")
	})

	t.Run("ReassignLabel", func(t *testing.T) {
		// Save new version
		tmpl2 := &StoredTemplate{
			Name:   "label-test",
			Source: "Updated: Hello {~prompty.var name=\"name\" /~}",
		}
		err := storage.Save(ctx, tmpl2)
		require.NoError(t, err)
		assert.Equal(t, 2, tmpl2.Version)

		// Reassign production label
		err = storage.SetLabel(ctx, "label-test", "production", 2, "new-user")
		require.NoError(t, err)

		got, err := storage.GetByLabel(ctx, "label-test", "production")
		require.NoError(t, err)
		assert.Equal(t, 2, got.Version)
	})

	t.Run("RemoveLabel", func(t *testing.T) {
		err := storage.RemoveLabel(ctx, "label-test", "staging")
		require.NoError(t, err)

		_, err = storage.GetByLabel(ctx, "label-test", "staging")
		assert.Error(t, err)
	})

	t.Run("LabelValidation", func(t *testing.T) {
		// Invalid label - uppercase
		err := storage.SetLabel(ctx, "label-test", "Production", 1, "")
		assert.Error(t, err)

		// Invalid label - empty
		err = storage.SetLabel(ctx, "label-test", "", 1, "")
		assert.Error(t, err)

		// Label for non-existent version
		err = storage.SetLabel(ctx, "label-test", "test", 999, "")
		assert.Error(t, err)
	})
}

func TestPostgres_E2E_LabelCleanupOnDelete(t *testing.T) {
	storage, cleanup := setupPostgresContainer(t)
	defer cleanup()
	ctx := context.Background()

	// Create template with versions and labels
	err := storage.Save(ctx, &StoredTemplate{Name: "delete-label-test", Source: "v1"})
	require.NoError(t, err)
	err = storage.Save(ctx, &StoredTemplate{Name: "delete-label-test", Source: "v2"})
	require.NoError(t, err)

	err = storage.SetLabel(ctx, "delete-label-test", "production", 1, "")
	require.NoError(t, err)
	err = storage.SetLabel(ctx, "delete-label-test", "staging", 2, "")
	require.NoError(t, err)

	// Delete template
	err = storage.Delete(ctx, "delete-label-test")
	require.NoError(t, err)

	// Labels should be cleaned up - trying to list should return empty
	// since the template no longer exists
	labels, err := storage.ListLabels(ctx, "delete-label-test")
	require.NoError(t, err)
	assert.Empty(t, labels)
}

// =============================================================================
// Status Storage Tests
// =============================================================================

func TestPostgres_E2E_Status(t *testing.T) {
	storage, cleanup := setupPostgresContainer(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("DefaultStatus", func(t *testing.T) {
		tmpl := &StoredTemplate{
			Name:   "status-test",
			Source: "Hello",
		}
		err := storage.Save(ctx, tmpl)
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusActive, tmpl.Status)

		got, err := storage.Get(ctx, "status-test")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusActive, got.Status)
	})

	t.Run("StatusTransitions", func(t *testing.T) {
		// Transition to deprecated
		err := storage.SetStatus(ctx, "status-test", 1, DeploymentStatusDeprecated, "user1")
		require.NoError(t, err)

		got, err := storage.Get(ctx, "status-test")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusDeprecated, got.Status)

		// Transition back to active (allowed)
		err = storage.SetStatus(ctx, "status-test", 1, DeploymentStatusActive, "user1")
		require.NoError(t, err)

		// Transition to archived
		err = storage.SetStatus(ctx, "status-test", 1, DeploymentStatusArchived, "user1")
		require.NoError(t, err)

		// Try to transition from archived (should fail)
		err = storage.SetStatus(ctx, "status-test", 1, DeploymentStatusActive, "user1")
		assert.Error(t, err)
	})

	t.Run("ListByStatus", func(t *testing.T) {
		// Create templates with different statuses
		tmpl1 := &StoredTemplate{Name: "active-1", Source: "active"}
		err := storage.Save(ctx, tmpl1)
		require.NoError(t, err)

		tmpl2 := &StoredTemplate{Name: "active-2", Source: "active"}
		err = storage.Save(ctx, tmpl2)
		require.NoError(t, err)

		tmpl3 := &StoredTemplate{Name: "deprecated-1", Source: "deprecated"}
		err = storage.Save(ctx, tmpl3)
		require.NoError(t, err)
		err = storage.SetStatus(ctx, "deprecated-1", 1, DeploymentStatusDeprecated, "user")
		require.NoError(t, err)

		// List active
		results, err := storage.ListByStatus(ctx, DeploymentStatusActive, nil)
		require.NoError(t, err)
		assert.Len(t, results, 2) // active-1 and active-2

		// List deprecated
		results, err = storage.ListByStatus(ctx, DeploymentStatusDeprecated, nil)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "deprecated-1", results[0].Name)

		// List archived
		results, err = storage.ListByStatus(ctx, DeploymentStatusArchived, nil)
		require.NoError(t, err)
		assert.Len(t, results, 1) // status-test from previous test
	})

	t.Run("DraftStatus", func(t *testing.T) {
		tmpl := &StoredTemplate{
			Name:   "draft-template",
			Source: "WIP content",
			Status: DeploymentStatusDraft,
		}
		err := storage.Save(ctx, tmpl)
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusDraft, tmpl.Status)

		got, err := storage.Get(ctx, "draft-template")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusDraft, got.Status)

		// Draft to active is allowed
		err = storage.SetStatus(ctx, "draft-template", 1, DeploymentStatusActive, "reviewer")
		require.NoError(t, err)

		// Create another draft
		tmpl2 := &StoredTemplate{
			Name:   "draft-template",
			Source: "WIP content v2",
			Status: DeploymentStatusDraft,
		}
		err = storage.Save(ctx, tmpl2)
		require.NoError(t, err)

		// Draft to deprecated is NOT allowed
		err = storage.SetStatus(ctx, "draft-template", 2, DeploymentStatusDeprecated, "reviewer")
		assert.Error(t, err)
	})
}

func TestPostgres_E2E_StatusPersistence(t *testing.T) {
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:15",
		postgres.WithDatabase("status_persist_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	defer func() { _ = container.Terminate(ctx) }()

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// First storage instance
	storage1, err := NewPostgresStorage(PostgresConfig{
		ConnectionString: connStr,
		AutoMigrate:      true,
	})
	require.NoError(t, err)

	err = storage1.Save(ctx, &StoredTemplate{
		Name:   "persist-test",
		Source: "content",
	})
	require.NoError(t, err)

	err = storage1.SetStatus(ctx, "persist-test", 1, DeploymentStatusDeprecated, "admin")
	require.NoError(t, err)

	err = storage1.SetLabel(ctx, "persist-test", "production", 1, "deploy-bot")
	require.NoError(t, err)

	err = storage1.Close()
	require.NoError(t, err)

	// Second storage instance - verify persistence
	storage2, err := NewPostgresStorage(PostgresConfig{
		ConnectionString: connStr,
		AutoMigrate:      false,
	})
	require.NoError(t, err)
	defer storage2.Close()

	// Status should persist
	tmpl, err := storage2.Get(ctx, "persist-test")
	require.NoError(t, err)
	assert.Equal(t, DeploymentStatusDeprecated, tmpl.Status)

	// Label should persist
	got, err := storage2.GetByLabel(ctx, "persist-test", "production")
	require.NoError(t, err)
	assert.Equal(t, 1, got.Version)
}

// =============================================================================
// Integration with StorageEngine
// =============================================================================

func TestPostgres_E2E_StorageEngineIntegration(t *testing.T) {
	storage, cleanup := setupPostgresContainer(t)
	defer cleanup()
	ctx := context.Background()

	// Create engine with PostgreSQL storage
	engine, err := NewStorageEngine(StorageEngineConfig{
		Storage: storage,
	})
	require.NoError(t, err)

	t.Run("SaveAndExecute", func(t *testing.T) {
		source := `Hello {~prompty.var name="user" /~}! Today is {~prompty.var name="day" /~}.`

		// Save template
		err := engine.Save(ctx, "greeting", source)
		require.NoError(t, err)

		// Execute template
		data := map[string]any{
			"user": "Alice",
			"day":  "Monday",
		}
		result, err := engine.Execute(ctx, "greeting", data)
		require.NoError(t, err)
		assert.Equal(t, "Hello Alice! Today is Monday.", result)
	})

	t.Run("ComplexTemplate", func(t *testing.T) {
		source := `
{~prompty.if eval="isAdmin"~}
Admin Dashboard
{~prompty.for item="item" in="items"~}
- {~prompty.var name="item.name" /~}: {~prompty.var name="item.value" /~}
{~/prompty.for~}
{~prompty.else~}
Access Denied
{~/prompty.if~}`

		err := engine.Save(ctx, "admin-dashboard", source)
		require.NoError(t, err)

		// Admin case
		data := map[string]any{
			"isAdmin": true,
			"items": []map[string]any{
				{"name": "Users", "value": 100},
				{"name": "Orders", "value": 50},
			},
		}
		result, err := engine.Execute(ctx, "admin-dashboard", data)
		require.NoError(t, err)
		assert.Contains(t, result, "Admin Dashboard")
		assert.Contains(t, result, "Users: 100")

		// Non-admin case
		data["isAdmin"] = false
		result, err = engine.Execute(ctx, "admin-dashboard", data)
		require.NoError(t, err)
		assert.Contains(t, result, "Access Denied")
	})
}
