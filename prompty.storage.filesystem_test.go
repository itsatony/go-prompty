package prompty

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesystemStorage_NewFilesystemStorage(t *testing.T) {
	t.Run("creates storage with new directory", func(t *testing.T) {
		dir := t.TempDir()
		root := filepath.Join(dir, "templates")

		storage, err := NewFilesystemStorage(root)
		require.NoError(t, err)
		require.NotNil(t, storage)
		defer storage.Close()

		// Verify directory was created
		info, err := os.Stat(root)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("uses existing directory", func(t *testing.T) {
		dir := t.TempDir()

		storage, err := NewFilesystemStorage(dir)
		require.NoError(t, err)
		require.NotNil(t, storage)
		defer storage.Close()
	})

	t.Run("rejects empty root", func(t *testing.T) {
		_, err := NewFilesystemStorage("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid storage root")
	})
}

func TestFilesystemStorage_Save(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage.Close()

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
		assert.Equal(t, 1, tmpl.Version)
		assert.False(t, tmpl.CreatedAt.IsZero())

		// Verify file was created
		filename := filepath.Join(dir, "greeting", "v1.json")
		_, err = os.Stat(filename)
		require.NoError(t, err)
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

		// Verify both files exist
		assert.FileExists(t, filepath.Join(dir, "versioned", "v1.json"))
		assert.FileExists(t, filepath.Join(dir, "versioned", "v2.json"))
	})

	t.Run("rejects empty name", func(t *testing.T) {
		tmpl := &StoredTemplate{Name: "", Source: "test"}
		err := storage.Save(ctx, tmpl)
		require.Error(t, err)
	})

	t.Run("rejects invalid characters in name", func(t *testing.T) {
		tmpl := &StoredTemplate{Name: "invalid/name", Source: "test"}
		err := storage.Save(ctx, tmpl)
		require.Error(t, err)
	})

	t.Run("rejects path traversal in name", func(t *testing.T) {
		traversalNames := []string{
			"../etc/passwd",
			"..\\windows\\system32",
			"foo/../bar",
			"foo/..\\bar",
			"..test",
			"test..",
			"te..st",
		}
		for _, name := range traversalNames {
			tmpl := &StoredTemplate{Name: name, Source: "test"}
			err := storage.Save(ctx, tmpl)
			require.Error(t, err, "should reject path traversal: %s", name)
		}
	})
}

func TestFilesystemStorage_Get(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage.Close()

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

	t.Run("returns error for nonexistent template", func(t *testing.T) {
		_, err := storage.Get(ctx, "nonexistent")
		require.Error(t, err)
	})

	t.Run("rejects path traversal", func(t *testing.T) {
		_, err := storage.Get(ctx, "../etc/passwd")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "path traversal")
	})
}

func TestFilesystemStorage_GetByID(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage.Close()

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

func TestFilesystemStorage_GetVersion(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage.Close()

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
	})
}

func TestFilesystemStorage_Delete(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Save multiple versions
	_ = storage.Save(ctx, &StoredTemplate{Name: "delete-me", Source: "v1"})
	_ = storage.Save(ctx, &StoredTemplate{Name: "delete-me", Source: "v2"})

	t.Run("deletes all versions", func(t *testing.T) {
		err := storage.Delete(ctx, "delete-me")
		require.NoError(t, err)

		exists, _ := storage.Exists(ctx, "delete-me")
		assert.False(t, exists)

		// Verify directory was removed
		templateDir := filepath.Join(dir, "delete-me")
		_, err = os.Stat(templateDir)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("returns error for nonexistent template", func(t *testing.T) {
		err := storage.Delete(ctx, "nonexistent")
		require.Error(t, err)
	})
}

func TestFilesystemStorage_DeleteVersion(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage.Close()

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

		// Verify file was removed
		assert.NoFileExists(t, filepath.Join(dir, "partial-delete", "v2.json"))
	})

	t.Run("removes directory when last version deleted", func(t *testing.T) {
		s, _ := NewFilesystemStorage(t.TempDir())
		defer s.Close()

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

func TestFilesystemStorage_List(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Setup test data
	templates := []struct {
		name      string
		tenant    string
		createdBy string
		tags      []string
	}{
		{"greeting-en", "tenant1", "user1", []string{"public", "english"}},
		{"greeting-es", "tenant1", "user2", []string{"public", "spanish"}},
		{"farewell-en", "tenant2", "user1", []string{"public", "english"}},
		{"internal", "tenant1", "user1", []string{"private"}},
	}

	for _, tmpl := range templates {
		_ = storage.Save(ctx, &StoredTemplate{
			Name:      tmpl.name,
			Source:    tmpl.name + " content",
			TenantID:  tmpl.tenant,
			CreatedBy: tmpl.createdBy,
			Tags:      tmpl.tags,
		})
	}

	t.Run("returns all templates with nil query", func(t *testing.T) {
		results, err := storage.List(ctx, nil)
		require.NoError(t, err)
		assert.Len(t, results, 4)
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

	t.Run("filters by tags", func(t *testing.T) {
		results, err := storage.List(ctx, &TemplateQuery{Tags: []string{"public", "english"}})
		require.NoError(t, err)
		assert.Len(t, results, 2)
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
}

func TestFilesystemStorage_Exists(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage.Close()

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

func TestFilesystemStorage_ListVersions(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage.Close()

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

func TestFilesystemStorage_Close(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)

	ctx := context.Background()
	_ = storage.Save(ctx, &StoredTemplate{Name: "test"})

	err = storage.Close()
	require.NoError(t, err)
	assert.True(t, storage.closed)

	// All operations should fail after close
	_, err = storage.Get(ctx, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestFilesystemStorage_ConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Concurrent writes
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := storage.Save(ctx, &StoredTemplate{
				Name:   "concurrent-" + intToStr(id%5),
				Source: "data from goroutine " + intToStr(id),
			})
			if err != nil {
				errors <- err
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, _ = storage.Get(ctx, "concurrent-"+intToStr(id%5))
			_, _ = storage.List(ctx, nil)
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent operation failed: %v", err)
	}
}

func TestFilesystemStorage_OpenViaRegistry(t *testing.T) {
	dir := t.TempDir()

	// The filesystem driver should be registered via init()
	drivers := ListStorageDrivers()
	assert.Contains(t, drivers, "filesystem")

	storage, err := OpenStorage("filesystem", dir)
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer storage.Close()

	// Test basic operation
	ctx := context.Background()
	err = storage.Save(ctx, &StoredTemplate{Name: "test", Source: "content"})
	require.NoError(t, err)

	tmpl, err := storage.Get(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, "content", tmpl.Source)
}

func TestFilesystemStorage_Persistence(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	// Create storage and save template
	storage1, err := NewFilesystemStorage(dir)
	require.NoError(t, err)

	_ = storage1.Save(ctx, &StoredTemplate{
		Name:   "persistent",
		Source: "original content",
		Tags:   []string{"tag1"},
	})
	_ = storage1.Close()

	// Create new storage instance and verify data persists
	storage2, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage2.Close()

	tmpl, err := storage2.Get(ctx, "persistent")
	require.NoError(t, err)
	assert.Equal(t, "original content", tmpl.Source)
	assert.Equal(t, []string{"tag1"}, tmpl.Tags)
}

func TestParseVersionNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"0", 0},
		{"1", 1},
		{"42", 42},
		{"123", 123},
		{"abc", 0},
		{"1a", 0},
		{"a1", 0},
	}

	for _, tt := range tests {
		result := parseVersionNumber(tt.input)
		assert.Equal(t, tt.expected, result, "input: %q", tt.input)
	}
}

// -----------------------------------------------------------------------------
// LabelStorage Tests
// -----------------------------------------------------------------------------

func TestFilesystemStorage_Labels(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Create a template
	tmpl := &StoredTemplate{
		Name:   "test-template",
		Source: "Hello {~prompty.var name=\"name\" /~}",
	}
	err = storage.Save(ctx, tmpl)
	require.NoError(t, err)
	assert.Equal(t, 1, tmpl.Version)

	// Set a label
	err = storage.SetLabel(ctx, "test-template", "production", 1, "user1")
	require.NoError(t, err)

	// Get by label
	got, err := storage.GetByLabel(ctx, "test-template", "production")
	require.NoError(t, err)
	assert.Equal(t, 1, got.Version)

	// List labels
	labels, err := storage.ListLabels(ctx, "test-template")
	require.NoError(t, err)
	assert.Len(t, labels, 1)
	assert.Equal(t, "production", labels[0].Label)
	assert.Equal(t, 1, labels[0].Version)
	assert.Equal(t, "user1", labels[0].AssignedBy)

	// Get version labels
	versionLabels, err := storage.GetVersionLabels(ctx, "test-template", 1)
	require.NoError(t, err)
	assert.Contains(t, versionLabels, "production")

	// Reassign label to new version
	tmpl2 := &StoredTemplate{
		Name:   "test-template",
		Source: "Updated: Hello {~prompty.var name=\"name\" /~}",
	}
	err = storage.Save(ctx, tmpl2)
	require.NoError(t, err)
	assert.Equal(t, 2, tmpl2.Version)

	err = storage.SetLabel(ctx, "test-template", "production", 2, "user2")
	require.NoError(t, err)

	got, err = storage.GetByLabel(ctx, "test-template", "production")
	require.NoError(t, err)
	assert.Equal(t, 2, got.Version)

	// Remove label
	err = storage.RemoveLabel(ctx, "test-template", "production")
	require.NoError(t, err)

	_, err = storage.GetByLabel(ctx, "test-template", "production")
	assert.Error(t, err)
}

func TestFilesystemStorage_LabelPersistence(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	// Create storage and save template with label
	storage1, err := NewFilesystemStorage(dir)
	require.NoError(t, err)

	err = storage1.Save(ctx, &StoredTemplate{
		Name:   "persistent",
		Source: "content",
	})
	require.NoError(t, err)

	err = storage1.SetLabel(ctx, "persistent", "production", 1, "deploy-user")
	require.NoError(t, err)

	err = storage1.Close()
	require.NoError(t, err)

	// Create new storage instance and verify label persists
	storage2, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage2.Close()

	// Verify label exists
	tmpl, err := storage2.GetByLabel(ctx, "persistent", "production")
	require.NoError(t, err)
	assert.Equal(t, 1, tmpl.Version)

	// Verify label details
	labels, err := storage2.ListLabels(ctx, "persistent")
	require.NoError(t, err)
	require.Len(t, labels, 1)
	assert.Equal(t, "production", labels[0].Label)
	assert.Equal(t, "deploy-user", labels[0].AssignedBy)
}

func TestFilesystemStorage_LabelValidation(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Create a template
	tmpl := &StoredTemplate{
		Name:   "test-template",
		Source: "Hello",
	}
	err = storage.Save(ctx, tmpl)
	require.NoError(t, err)

	// Invalid label - uppercase
	err = storage.SetLabel(ctx, "test-template", "Production", 1, "")
	assert.Error(t, err)

	// Invalid label - empty
	err = storage.SetLabel(ctx, "test-template", "", 1, "")
	assert.Error(t, err)

	// Label for non-existent version
	err = storage.SetLabel(ctx, "test-template", "production", 999, "")
	assert.Error(t, err)

	// Label for non-existent template
	err = storage.SetLabel(ctx, "non-existent", "production", 1, "")
	assert.Error(t, err)
}

func TestFilesystemStorage_LabelCleanupOnDelete(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Create template and versions
	err = storage.Save(ctx, &StoredTemplate{Name: "delete-test", Source: "v1"})
	require.NoError(t, err)
	err = storage.Save(ctx, &StoredTemplate{Name: "delete-test", Source: "v2"})
	require.NoError(t, err)

	// Set labels
	err = storage.SetLabel(ctx, "delete-test", "production", 1, "")
	require.NoError(t, err)
	err = storage.SetLabel(ctx, "delete-test", "staging", 2, "")
	require.NoError(t, err)

	// Delete template
	err = storage.Delete(ctx, "delete-test")
	require.NoError(t, err)

	// Verify labels are gone (by checking that labels.json doesn't exist)
	labelsFile := filepath.Join(dir, "delete-test", "labels.json")
	_, err = os.Stat(labelsFile)
	assert.True(t, os.IsNotExist(err))
}

func TestFilesystemStorage_LabelCleanupOnDeleteVersion(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Create template with multiple versions
	err = storage.Save(ctx, &StoredTemplate{Name: "version-delete", Source: "v1"})
	require.NoError(t, err)
	err = storage.Save(ctx, &StoredTemplate{Name: "version-delete", Source: "v2"})
	require.NoError(t, err)
	err = storage.Save(ctx, &StoredTemplate{Name: "version-delete", Source: "v3"})
	require.NoError(t, err)

	// Set labels on different versions
	err = storage.SetLabel(ctx, "version-delete", "production", 1, "")
	require.NoError(t, err)
	err = storage.SetLabel(ctx, "version-delete", "staging", 2, "")
	require.NoError(t, err)
	err = storage.SetLabel(ctx, "version-delete", "canary", 3, "")
	require.NoError(t, err)

	// Delete version 2 (which has staging label)
	err = storage.DeleteVersion(ctx, "version-delete", 2)
	require.NoError(t, err)

	// Verify staging label is gone
	_, err = storage.GetByLabel(ctx, "version-delete", "staging")
	assert.Error(t, err)

	// Verify other labels still exist
	labels, err := storage.ListLabels(ctx, "version-delete")
	require.NoError(t, err)
	assert.Len(t, labels, 2)

	labelMap := make(map[string]int)
	for _, l := range labels {
		labelMap[l.Label] = l.Version
	}
	assert.Equal(t, 1, labelMap["production"])
	assert.Equal(t, 3, labelMap["canary"])
}

func TestFilesystemStorage_DeleteVersionCleansAllLabels(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Create template with single version
	err = storage.Save(ctx, &StoredTemplate{Name: "single-version", Source: "content"})
	require.NoError(t, err)

	// Set multiple labels on the same version
	err = storage.SetLabel(ctx, "single-version", "production", 1, "")
	require.NoError(t, err)
	err = storage.SetLabel(ctx, "single-version", "staging", 1, "")
	require.NoError(t, err)

	// Delete the only version
	err = storage.DeleteVersion(ctx, "single-version", 1)
	require.NoError(t, err)

	// Verify directory is completely gone (including labels.json)
	templateDir := filepath.Join(dir, "single-version")
	_, err = os.Stat(templateDir)
	assert.True(t, os.IsNotExist(err))
}

// -----------------------------------------------------------------------------
// StatusStorage Tests
// -----------------------------------------------------------------------------

func TestFilesystemStorage_Status(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Create a template - should default to active
	tmpl := &StoredTemplate{
		Name:   "test-template",
		Source: "Hello",
	}
	err = storage.Save(ctx, tmpl)
	require.NoError(t, err)
	assert.Equal(t, DeploymentStatusActive, tmpl.Status)

	// Verify stored status
	got, err := storage.Get(ctx, "test-template")
	require.NoError(t, err)
	assert.Equal(t, DeploymentStatusActive, got.Status)

	// Transition to deprecated
	err = storage.SetStatus(ctx, "test-template", 1, DeploymentStatusDeprecated, "user1")
	require.NoError(t, err)

	got, err = storage.Get(ctx, "test-template")
	require.NoError(t, err)
	assert.Equal(t, DeploymentStatusDeprecated, got.Status)

	// Transition back to active (allowed)
	err = storage.SetStatus(ctx, "test-template", 1, DeploymentStatusActive, "user1")
	require.NoError(t, err)

	// Transition to archived
	err = storage.SetStatus(ctx, "test-template", 1, DeploymentStatusArchived, "user1")
	require.NoError(t, err)

	// Try to transition from archived (should fail)
	err = storage.SetStatus(ctx, "test-template", 1, DeploymentStatusActive, "user1")
	assert.Error(t, err)

	// List by status
	results, err := storage.ListByStatus(ctx, DeploymentStatusArchived, nil)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "test-template", results[0].Name)
}

func TestFilesystemStorage_StatusPersistence(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	// Create storage and save template
	storage1, err := NewFilesystemStorage(dir)
	require.NoError(t, err)

	err = storage1.Save(ctx, &StoredTemplate{
		Name:   "persistent",
		Source: "content",
	})
	require.NoError(t, err)

	err = storage1.SetStatus(ctx, "persistent", 1, DeploymentStatusDeprecated, "admin")
	require.NoError(t, err)

	err = storage1.Close()
	require.NoError(t, err)

	// Create new storage instance and verify status persists
	storage2, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage2.Close()

	tmpl, err := storage2.Get(ctx, "persistent")
	require.NoError(t, err)
	assert.Equal(t, DeploymentStatusDeprecated, tmpl.Status)
}

func TestFilesystemStorage_StatusWithDraft(t *testing.T) {
	dir := t.TempDir()
	storage, err := NewFilesystemStorage(dir)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()

	// Create a template with draft status
	tmpl := &StoredTemplate{
		Name:   "draft-template",
		Source: "WIP content",
		Status: DeploymentStatusDraft,
	}
	err = storage.Save(ctx, tmpl)
	require.NoError(t, err)
	assert.Equal(t, DeploymentStatusDraft, tmpl.Status)

	// Verify stored status
	got, err := storage.Get(ctx, "draft-template")
	require.NoError(t, err)
	assert.Equal(t, DeploymentStatusDraft, got.Status)

	// Draft to active is allowed
	err = storage.SetStatus(ctx, "draft-template", 1, DeploymentStatusActive, "reviewer")
	require.NoError(t, err)

	// Create another draft and try invalid transition
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
}
