package prompty

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_NewMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage()
	require.NotNil(t, storage)
	assert.NotNil(t, storage.templates)
	assert.NotNil(t, storage.byID)
	assert.False(t, storage.closed)
}

func TestMemoryStorage_Save(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	t.Run("saves new template", func(t *testing.T) {
		tmpl := &StoredTemplate{
			Name:      "greeting",
			Source:    "Hello, World!",
			CreatedBy: "test_user",
			TenantID:  "tenant_1",
			Tags:      []string{"public"},
			Metadata:  map[string]string{"author": "test"},
		}

		err := storage.Save(ctx, tmpl)
		require.NoError(t, err)

		// Verify generated fields
		assert.NotEmpty(t, tmpl.ID)
		assert.True(t, hasPrefix(string(tmpl.ID), "tmpl_"))
		assert.Equal(t, 1, tmpl.Version)
		assert.False(t, tmpl.CreatedAt.IsZero())
		assert.False(t, tmpl.UpdatedAt.IsZero())
	})

	t.Run("creates new version for existing template", func(t *testing.T) {
		tmpl1 := &StoredTemplate{Name: "versioned", Source: "v1"}
		err := storage.Save(ctx, tmpl1)
		require.NoError(t, err)
		assert.Equal(t, 1, tmpl1.Version)

		tmpl2 := &StoredTemplate{Name: "versioned", Source: "v2"}
		err = storage.Save(ctx, tmpl2)
		require.NoError(t, err)
		assert.Equal(t, 2, tmpl2.Version)

		tmpl3 := &StoredTemplate{Name: "versioned", Source: "v3"}
		err = storage.Save(ctx, tmpl3)
		require.NoError(t, err)
		assert.Equal(t, 3, tmpl3.Version)
	})

	t.Run("rejects empty name", func(t *testing.T) {
		tmpl := &StoredTemplate{Name: "", Source: "test"}
		err := storage.Save(ctx, tmpl)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "template name")
	})

	t.Run("rejects save on closed storage", func(t *testing.T) {
		s := NewMemoryStorage()
		s.Close()
		err := s.Save(ctx, &StoredTemplate{Name: "test"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "closed")
	})
}

func TestMemoryStorage_Get(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Save multiple versions
	for i := 0; i < 3; i++ {
		_ = storage.Save(ctx, &StoredTemplate{
			Name:   "test",
			Source: "version " + intToStr(i+1),
		})
	}

	t.Run("returns latest version", func(t *testing.T) {
		tmpl, err := storage.Get(ctx, "test")
		require.NoError(t, err)
		assert.Equal(t, 3, tmpl.Version)
		assert.Equal(t, "version 3", tmpl.Source)
	})

	t.Run("returns copy not reference", func(t *testing.T) {
		tmpl1, _ := storage.Get(ctx, "test")
		tmpl2, _ := storage.Get(ctx, "test")
		assert.NotSame(t, tmpl1, tmpl2)

		tmpl1.Source = "modified"
		tmpl3, _ := storage.Get(ctx, "test")
		assert.Equal(t, "version 3", tmpl3.Source)
	})

	t.Run("returns error for nonexistent template", func(t *testing.T) {
		_, err := storage.Get(ctx, "nonexistent")
		require.Error(t, err)
	})

	t.Run("returns error on closed storage", func(t *testing.T) {
		s := NewMemoryStorage()
		_ = s.Save(ctx, &StoredTemplate{Name: "test"})
		_ = s.Close()
		_, err := s.Get(ctx, "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "closed")
	})
}

func TestMemoryStorage_GetByID(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	tmpl := &StoredTemplate{Name: "test", Source: "content"}
	_ = storage.Save(ctx, tmpl)

	t.Run("returns template by ID", func(t *testing.T) {
		result, err := storage.GetByID(ctx, tmpl.ID)
		require.NoError(t, err)
		assert.Equal(t, tmpl.ID, result.ID)
		assert.Equal(t, "content", result.Source)
	})

	t.Run("returns error for nonexistent ID", func(t *testing.T) {
		_, err := storage.GetByID(ctx, "tmpl_nonexistent")
		require.Error(t, err)
	})
}

func TestMemoryStorage_GetVersion(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Save multiple versions
	for i := 0; i < 3; i++ {
		_ = storage.Save(ctx, &StoredTemplate{
			Name:   "versioned",
			Source: "v" + intToStr(i+1),
		})
	}

	t.Run("returns specific version", func(t *testing.T) {
		tmpl, err := storage.GetVersion(ctx, "versioned", 2)
		require.NoError(t, err)
		assert.Equal(t, 2, tmpl.Version)
		assert.Equal(t, "v2", tmpl.Source)
	})

	t.Run("returns error for nonexistent version", func(t *testing.T) {
		_, err := storage.GetVersion(ctx, "versioned", 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "version not found")
	})

	t.Run("returns error for nonexistent template", func(t *testing.T) {
		_, err := storage.GetVersion(ctx, "nonexistent", 1)
		require.Error(t, err)
	})
}

func TestMemoryStorage_Delete(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Save multiple versions
	_ = storage.Save(ctx, &StoredTemplate{Name: "delete-me", Source: "v1"})
	_ = storage.Save(ctx, &StoredTemplate{Name: "delete-me", Source: "v2"})

	t.Run("deletes all versions", func(t *testing.T) {
		err := storage.Delete(ctx, "delete-me")
		require.NoError(t, err)

		exists, _ := storage.Exists(ctx, "delete-me")
		assert.False(t, exists)
	})

	t.Run("returns error for nonexistent template", func(t *testing.T) {
		err := storage.Delete(ctx, "nonexistent")
		require.Error(t, err)
	})
}

func TestMemoryStorage_DeleteVersion(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Save multiple versions
	_ = storage.Save(ctx, &StoredTemplate{Name: "partial-delete", Source: "v1"})
	_ = storage.Save(ctx, &StoredTemplate{Name: "partial-delete", Source: "v2"})
	_ = storage.Save(ctx, &StoredTemplate{Name: "partial-delete", Source: "v3"})

	t.Run("deletes specific version", func(t *testing.T) {
		err := storage.DeleteVersion(ctx, "partial-delete", 2)
		require.NoError(t, err)

		versions, _ := storage.ListVersions(ctx, "partial-delete")
		assert.Equal(t, []int{3, 1}, versions)
	})

	t.Run("removes template when last version deleted", func(t *testing.T) {
		s := NewMemoryStorage()
		_ = s.Save(ctx, &StoredTemplate{Name: "single", Source: "only"})

		err := s.DeleteVersion(ctx, "single", 1)
		require.NoError(t, err)

		exists, _ := s.Exists(ctx, "single")
		assert.False(t, exists)
	})

	t.Run("returns error for nonexistent version", func(t *testing.T) {
		err := storage.DeleteVersion(ctx, "partial-delete", 99)
		require.Error(t, err)
	})
}

func TestMemoryStorage_List(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Setup test data
	templates := []struct {
		name      string
		tenant    string
		createdBy string
		tags      []string
		versions  int
	}{
		{"greeting-en", "tenant1", "user1", []string{"public", "english"}, 2},
		{"greeting-es", "tenant1", "user2", []string{"public", "spanish"}, 1},
		{"farewell-en", "tenant2", "user1", []string{"public", "english"}, 1},
		{"internal", "tenant1", "user1", []string{"private"}, 1},
	}

	for _, t := range templates {
		for i := 0; i < t.versions; i++ {
			_ = storage.Save(ctx, &StoredTemplate{
				Name:      t.name,
				Source:    t.name + " v" + intToStr(i+1),
				TenantID:  t.tenant,
				CreatedBy: t.createdBy,
				Tags:      t.tags,
			})
		}
	}

	t.Run("returns all templates with nil query", func(t *testing.T) {
		results, err := storage.List(ctx, nil)
		require.NoError(t, err)
		assert.Len(t, results, 4) // One per unique name (latest version)
	})

	t.Run("filters by tenant", func(t *testing.T) {
		results, err := storage.List(ctx, &TemplateQuery{TenantID: "tenant1"})
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("filters by name prefix", func(t *testing.T) {
		results, err := storage.List(ctx, &TemplateQuery{NamePrefix: "greeting"})
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("filters by name contains", func(t *testing.T) {
		results, err := storage.List(ctx, &TemplateQuery{NameContains: "-en"})
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("filters by created by", func(t *testing.T) {
		results, err := storage.List(ctx, &TemplateQuery{CreatedBy: "user1"})
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("filters by tags (all must match)", func(t *testing.T) {
		results, err := storage.List(ctx, &TemplateQuery{Tags: []string{"public", "english"}})
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("includes all versions when requested", func(t *testing.T) {
		results, err := storage.List(ctx, &TemplateQuery{
			NamePrefix:         "greeting-en",
			IncludeAllVersions: true,
		})
		require.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, 2, results[0].Version) // Newest first
		assert.Equal(t, 1, results[1].Version)
	})

	t.Run("applies limit", func(t *testing.T) {
		results, err := storage.List(ctx, &TemplateQuery{Limit: 2})
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("applies offset", func(t *testing.T) {
		all, _ := storage.List(ctx, nil)
		results, err := storage.List(ctx, &TemplateQuery{Offset: 2})
		require.NoError(t, err)
		assert.Len(t, results, len(all)-2)
	})

	t.Run("applies limit and offset together", func(t *testing.T) {
		results, err := storage.List(ctx, &TemplateQuery{Offset: 1, Limit: 2})
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("returns empty for offset beyond results", func(t *testing.T) {
		results, err := storage.List(ctx, &TemplateQuery{Offset: 100})
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})

	t.Run("results are sorted by name then version desc", func(t *testing.T) {
		results, err := storage.List(ctx, &TemplateQuery{IncludeAllVersions: true})
		require.NoError(t, err)

		// Check order
		for i := 1; i < len(results); i++ {
			prev := results[i-1]
			curr := results[i]
			if prev.Name == curr.Name {
				assert.Greater(t, prev.Version, curr.Version, "versions should be descending")
			} else {
				assert.Less(t, prev.Name, curr.Name, "names should be ascending")
			}
		}
	})
}

func TestMemoryStorage_Exists(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	_ = storage.Save(ctx, &StoredTemplate{Name: "exists"})

	t.Run("returns true for existing template", func(t *testing.T) {
		exists, err := storage.Exists(ctx, "exists")
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false for nonexistent template", func(t *testing.T) {
		exists, err := storage.Exists(ctx, "nonexistent")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestMemoryStorage_ListVersions(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Save multiple versions
	for i := 0; i < 3; i++ {
		_ = storage.Save(ctx, &StoredTemplate{Name: "multi"})
	}

	t.Run("returns all version numbers", func(t *testing.T) {
		versions, err := storage.ListVersions(ctx, "multi")
		require.NoError(t, err)
		assert.Equal(t, []int{3, 2, 1}, versions)
	})

	t.Run("returns empty for nonexistent template", func(t *testing.T) {
		versions, err := storage.ListVersions(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Empty(t, versions)
	})
}

func TestMemoryStorage_Close(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	_ = storage.Save(ctx, &StoredTemplate{Name: "test"})

	err := storage.Close()
	require.NoError(t, err)
	assert.True(t, storage.closed)

	// All operations should fail after close
	_, err = storage.Get(ctx, "test")
	assert.Error(t, err)

	_, err = storage.List(ctx, nil)
	assert.Error(t, err)

	err = storage.Save(ctx, &StoredTemplate{Name: "new"})
	assert.Error(t, err)
}

func TestMemoryStorage_ConcurrentAccess(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Concurrent writes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := storage.Save(ctx, &StoredTemplate{
				Name:   "concurrent-" + intToStr(id%10),
				Source: "data from goroutine " + intToStr(id),
			})
			if err != nil {
				errors <- err
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, _ = storage.Get(ctx, "concurrent-"+intToStr(id%10))
			_, _ = storage.List(ctx, nil)
			_, _ = storage.Exists(ctx, "concurrent-"+intToStr(id%10))
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent operation failed: %v", err)
	}
}

func TestMemoryStorage_ContextCancellation(t *testing.T) {
	storage := NewMemoryStorage()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	t.Run("Get respects context", func(t *testing.T) {
		_, err := storage.Get(ctx, "test")
		assert.Error(t, err)
	})

	t.Run("Save respects context", func(t *testing.T) {
		err := storage.Save(ctx, &StoredTemplate{Name: "test"})
		assert.Error(t, err)
	})

	t.Run("List respects context", func(t *testing.T) {
		_, err := storage.List(ctx, nil)
		assert.Error(t, err)
	})
}

func TestMemoryStorageDriver_Open(t *testing.T) {
	driver := &MemoryStorageDriver{}

	storage, err := driver.Open("")
	require.NoError(t, err)
	require.NotNil(t, storage)

	// Verify it's a working MemoryStorage
	ctx := context.Background()
	err = storage.Save(ctx, &StoredTemplate{Name: "test", Source: "content"})
	require.NoError(t, err)

	tmpl, err := storage.Get(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, "content", tmpl.Source)
}

func TestMemoryStorage_OpenViaRegistry(t *testing.T) {
	// The memory driver should be registered via init()
	drivers := ListStorageDrivers()
	assert.Contains(t, drivers, "memory")

	storage, err := OpenStorage("memory", "")
	require.NoError(t, err)
	require.NotNil(t, storage)

	defer storage.Close()
}

func TestGenerateTemplateID(t *testing.T) {
	ids := make(map[TemplateID]bool)

	for i := 0; i < 100; i++ {
		id := generateTemplateID()
		assert.True(t, hasPrefix(string(id), "tmpl_"))
		assert.False(t, ids[id], "generated duplicate ID")
		ids[id] = true
	}
}

func TestCopyStoredTemplate(t *testing.T) {
	original := &StoredTemplate{
		ID:        "tmpl_test",
		Name:      "test",
		Source:    "content",
		Version:   1,
		Metadata:  map[string]string{"key": "value"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: "user",
		TenantID:  "tenant",
		Tags:      []string{"tag1", "tag2"},
	}

	copy := copyStoredTemplate(original)

	t.Run("copies all fields", func(t *testing.T) {
		assert.Equal(t, original.ID, copy.ID)
		assert.Equal(t, original.Name, copy.Name)
		assert.Equal(t, original.Source, copy.Source)
		assert.Equal(t, original.Version, copy.Version)
		assert.Equal(t, original.CreatedBy, copy.CreatedBy)
		assert.Equal(t, original.TenantID, copy.TenantID)
	})

	t.Run("deep copies metadata", func(t *testing.T) {
		copy.Metadata["new"] = "added"
		assert.NotContains(t, original.Metadata, "new")
	})

	t.Run("deep copies tags", func(t *testing.T) {
		copy.Tags[0] = "modified"
		assert.Equal(t, "tag1", original.Tags[0])
	})

	t.Run("handles nil input", func(t *testing.T) {
		assert.Nil(t, copyStoredTemplate(nil))
	})
}

// Helper function to check string prefix
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
