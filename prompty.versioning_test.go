package prompty

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetVersionHistory(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	ctx := context.Background()

	// Create multiple versions
	err = engine.Save(ctx, &StoredTemplate{
		Name:   "test",
		Source: "Version 1 content",
		Tags:   []string{"v1"},
	})
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	err = engine.Save(ctx, &StoredTemplate{
		Name:   "test",
		Source: "Version 2 content with more text",
		Tags:   []string{"v2", "latest"},
	})
	require.NoError(t, err)

	// Get history
	history, err := engine.GetVersionHistory(ctx, "test")
	require.NoError(t, err)

	assert.Equal(t, "test", history.TemplateName)
	assert.Equal(t, 2, history.TotalVersions)
	assert.Equal(t, 2, history.CurrentVersion)
	assert.Len(t, history.Versions, 2)

	// Check version info
	assert.Equal(t, 2, history.Versions[0].Version)
	assert.True(t, history.Versions[0].IsCurrent)
	assert.Contains(t, history.Versions[0].Tags, "latest")

	assert.Equal(t, 1, history.Versions[1].Version)
	assert.False(t, history.Versions[1].IsCurrent)

	// Check oldest/newest
	assert.NotNil(t, history.OldestVersion)
	assert.NotNil(t, history.NewestVersion)
	assert.Equal(t, 1, history.OldestVersion.Version)
	assert.Equal(t, 2, history.NewestVersion.Version)
}

func TestGetVersionHistory_NotFound(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	_, err = engine.GetVersionHistory(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestGetVersionHistory_TokenEstimate(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	ctx := context.Background()

	err = engine.Save(ctx, &StoredTemplate{
		Name:   "test",
		Source: "This is a sample template with some content for token estimation.",
	})
	require.NoError(t, err)

	history, err := engine.GetVersionHistory(ctx, "test")
	require.NoError(t, err)

	// Verify token estimate is populated
	assert.NotNil(t, history.Versions[0].TokenEstimate)
	assert.True(t, history.Versions[0].TokenEstimate.EstimatedGeneric > 0)
}

func TestCompareVersions(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	ctx := context.Background()

	// Create v1
	err = engine.Save(ctx, &StoredTemplate{
		Name:   "test",
		Source: "Line 1\nLine 2\nLine 3",
		Tags:   []string{"original"},
	})
	require.NoError(t, err)

	// Create v2 with changes
	err = engine.Save(ctx, &StoredTemplate{
		Name:   "test",
		Source: "Line 1\nLine 2 modified\nLine 4",
		Tags:   []string{"modified"},
	})
	require.NoError(t, err)

	// Compare versions
	diff, err := engine.CompareVersions(ctx, "test", 1, 2)
	require.NoError(t, err)

	assert.Equal(t, 1, diff.OldVersion)
	assert.Equal(t, 2, diff.NewVersion)
	assert.True(t, diff.HasChanges())

	// Check line changes
	assert.Contains(t, diff.AddedLines, "Line 2 modified")
	assert.Contains(t, diff.AddedLines, "Line 4")
	assert.Contains(t, diff.RemovedLines, "Line 2")
	assert.Contains(t, diff.RemovedLines, "Line 3")

	// Check tag changes
	assert.Contains(t, diff.AddedTags, "modified")
	assert.Contains(t, diff.RemovedTags, "original")
}

func TestCompareVersions_NoChanges(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	ctx := context.Background()

	// Create v1
	err = engine.Save(ctx, &StoredTemplate{
		Name:   "test",
		Source: "Same content",
		Tags:   []string{"tag1"},
	})
	require.NoError(t, err)

	// Create v2 with same content
	err = engine.Save(ctx, &StoredTemplate{
		Name:   "test",
		Source: "Same content",
		Tags:   []string{"tag1"},
	})
	require.NoError(t, err)

	diff, err := engine.CompareVersions(ctx, "test", 1, 2)
	require.NoError(t, err)

	assert.False(t, diff.HasChanges())
	assert.Empty(t, diff.AddedLines)
	assert.Empty(t, diff.RemovedLines)
	assert.Empty(t, diff.AddedTags)
	assert.Empty(t, diff.RemovedTags)
}

func TestCompareVersions_InvalidVersion(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	ctx := context.Background()

	err = engine.Save(ctx, &StoredTemplate{
		Name:   "test",
		Source: "content",
	})
	require.NoError(t, err)

	// Try to compare with nonexistent version
	_, err = engine.CompareVersions(ctx, "test", 1, 99)
	assert.Error(t, err)
}

func TestRollbackToVersion(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	ctx := context.Background()

	// Create v1
	err = engine.Save(ctx, &StoredTemplate{
		Name:   "test",
		Source: "Original content",
		Tags:   []string{"v1"},
		Metadata: map[string]string{
			"author": "Alice",
		},
	})
	require.NoError(t, err)

	// Create v2
	err = engine.Save(ctx, &StoredTemplate{
		Name:   "test",
		Source: "Modified content",
		Tags:   []string{"v2"},
	})
	require.NoError(t, err)

	// Rollback to v1
	rolled, err := engine.RollbackToVersion(ctx, "test", 1)
	require.NoError(t, err)

	assert.Equal(t, "Original content", rolled.Source)
	assert.Contains(t, rolled.Tags, "v1")
	assert.Equal(t, "1", rolled.Metadata[MetaKeyRollbackFromVersion])
	assert.Equal(t, "Alice", rolled.Metadata["author"])

	// Verify current version is now v3 (rollback creates new version)
	current, err := engine.Get(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, 3, current.Version)
	assert.Equal(t, "Original content", current.Source)
}

func TestRollbackToVersion_InvalidVersion(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	ctx := context.Background()

	err = engine.Save(ctx, &StoredTemplate{
		Name:   "test",
		Source: "content",
	})
	require.NoError(t, err)

	_, err = engine.RollbackToVersion(ctx, "test", 99)
	assert.Error(t, err)
}

func TestRollbackToVersion_StatusIsDraft(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	ctx := context.Background()

	// Create v1 with active status (default)
	err = engine.Save(ctx, &StoredTemplate{
		Name:   "test",
		Source: "Original content",
	})
	require.NoError(t, err)

	// Create v2
	err = engine.Save(ctx, &StoredTemplate{
		Name:   "test",
		Source: "Modified content",
	})
	require.NoError(t, err)

	// Rollback to v1
	rolled, err := engine.RollbackToVersion(ctx, "test", 1)
	require.NoError(t, err)

	// Verify status is draft (rollbacks need review before activation)
	assert.Equal(t, DeploymentStatusDraft, rolled.Status)

	// Verify stored version also has draft status
	current, err := engine.Get(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, 3, current.Version)
	assert.Equal(t, DeploymentStatusDraft, current.Status)
}

func TestCloneVersion(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	ctx := context.Background()

	// Create source template
	err = engine.Save(ctx, &StoredTemplate{
		Name:   "source",
		Source: "Template content",
		Tags:   []string{"production"},
		Metadata: map[string]string{
			"author": "Alice",
		},
	})
	require.NoError(t, err)

	// Clone to new template
	cloned, err := engine.CloneVersion(ctx, "source", 1, "cloned")
	require.NoError(t, err)

	assert.Equal(t, "cloned", cloned.Name)
	assert.Equal(t, "Template content", cloned.Source)
	assert.Contains(t, cloned.Tags, "production")
	assert.Equal(t, "source", cloned.Metadata[MetaKeyClonedFrom])
	assert.Equal(t, "1", cloned.Metadata[MetaKeyClonedFromVersion])
	assert.Equal(t, "Alice", cloned.Metadata["author"])

	// Verify it exists
	retrieved, err := engine.Get(ctx, "cloned")
	require.NoError(t, err)
	assert.Equal(t, "Template content", retrieved.Source)
}

func TestCloneVersion_SourceNotFound(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	_, err = engine.CloneVersion(context.Background(), "nonexistent", 1, "new")
	assert.Error(t, err)
}

func TestCloneVersion_TargetExists(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	ctx := context.Background()

	// Create both templates
	err = engine.Save(ctx, &StoredTemplate{Name: "source", Source: "content"})
	require.NoError(t, err)

	err = engine.Save(ctx, &StoredTemplate{Name: "target", Source: "existing"})
	require.NoError(t, err)

	// Try to clone to existing name
	_, err = engine.CloneVersion(ctx, "source", 1, "target")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestCloneVersion_StatusIsDraft(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	ctx := context.Background()

	// Create source template with active status (default)
	err = engine.Save(ctx, &StoredTemplate{
		Name:   "source",
		Source: "Template content",
	})
	require.NoError(t, err)

	// Clone to new template
	cloned, err := engine.CloneVersion(ctx, "source", 1, "cloned")
	require.NoError(t, err)

	// Verify status is draft (clones may need customization before activation)
	assert.Equal(t, DeploymentStatusDraft, cloned.Status)

	// Verify stored template also has draft status
	retrieved, err := engine.Get(ctx, "cloned")
	require.NoError(t, err)
	assert.Equal(t, DeploymentStatusDraft, retrieved.Status)
}

func TestPruneOldVersions(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	ctx := context.Background()

	// Create 5 versions
	for i := 1; i <= 5; i++ {
		err = engine.Save(ctx, &StoredTemplate{
			Name:   "test",
			Source: "Content v" + string(rune('0'+i)),
		})
		require.NoError(t, err)
	}

	// Verify 5 versions exist
	versions, err := engine.storage.ListVersions(ctx, "test")
	require.NoError(t, err)
	assert.Len(t, versions, 5)

	// Prune to keep only 2
	deleted, err := engine.PruneOldVersions(ctx, "test", 2)
	require.NoError(t, err)
	assert.Equal(t, 3, deleted)

	// Verify only 2 remain
	versions, err = engine.storage.ListVersions(ctx, "test")
	require.NoError(t, err)
	assert.Len(t, versions, 2)

	// Verify kept versions are the newest
	assert.Equal(t, 5, versions[0])
	assert.Equal(t, 4, versions[1])
}

func TestPruneOldVersions_KeepAll(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	ctx := context.Background()

	// Create 3 versions
	for i := 1; i <= 3; i++ {
		err = engine.Save(ctx, &StoredTemplate{
			Name:   "test",
			Source: "Content",
		})
		require.NoError(t, err)
	}

	// Try to keep more than exist
	deleted, err := engine.PruneOldVersions(ctx, "test", 10)
	require.NoError(t, err)
	assert.Equal(t, 0, deleted)
}

func TestPruneOldVersions_InvalidKeepCount(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	_, err = engine.PruneOldVersions(context.Background(), "test", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must keep at least 1")
}

func TestGetVersionDelta(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	ctx := context.Background()

	// Create v1
	err = engine.Save(ctx, &StoredTemplate{
		Name:   "test",
		Source: "First version",
	})
	require.NoError(t, err)

	// Create v2
	err = engine.Save(ctx, &StoredTemplate{
		Name:   "test",
		Source: "Second version",
	})
	require.NoError(t, err)

	// Get delta between v1 and v2
	diff, err := engine.GetVersionDelta(ctx, "test", 2)
	require.NoError(t, err)

	assert.Equal(t, 1, diff.OldVersion)
	assert.Equal(t, 2, diff.NewVersion)
	assert.True(t, diff.HasChanges())
}

func TestGetVersionDelta_Version1(t *testing.T) {
	storage := NewMemoryStorage()
	engine, err := NewStorageEngine(StorageEngineConfig{Storage: storage})
	require.NoError(t, err)
	defer engine.Close()

	ctx := context.Background()

	err = engine.Save(ctx, &StoredTemplate{
		Name:   "test",
		Source: "content",
	})
	require.NoError(t, err)

	// Can't get delta for version 1 (no previous version)
	_, err = engine.GetVersionDelta(ctx, "test", 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no previous version")
}

func TestVersionDiff_String(t *testing.T) {
	diff := &VersionDiff{
		OldVersion:   1,
		NewVersion:   2,
		AddedLines:   []string{"new line 1", "new line 2"},
		RemovedLines: []string{"old line"},
		SameLines:    5,
		AddedTags:    []string{"new-tag"},
		RemovedTags:  []string{"old-tag"},
	}

	str := diff.String()

	assert.Contains(t, str, "Version 1 -> 2")
	assert.Contains(t, str, "+2")
	assert.Contains(t, str, "-1")
	assert.Contains(t, str, "=5 unchanged")
	assert.Contains(t, str, "Tags added: new-tag")
	assert.Contains(t, str, "Tags removed: old-tag")
	assert.Contains(t, str, "+ new line 1")
	assert.Contains(t, str, "- old line")
}

func TestVersionDiff_String_ManyChanges(t *testing.T) {
	// When there are more than 10 added/removed lines, don't show details
	diff := &VersionDiff{
		OldVersion:   1,
		NewVersion:   2,
		AddedLines:   make([]string, 15),
		RemovedLines: make([]string, 15),
	}

	str := diff.String()

	// Should show summary but not individual lines
	assert.Contains(t, str, "Version 1 -> 2")
	assert.NotContains(t, str, "Added:")
	assert.NotContains(t, str, "Removed:")
}

func TestVersionHistory_String(t *testing.T) {
	history := &VersionHistory{
		TemplateName:   "my-template",
		CurrentVersion: 2,
		TotalVersions:  2,
		Versions: []VersionInfo{
			{
				Version:   2,
				CreatedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
				CreatedBy: "Alice",
				SourceLen: 100,
				Tags:      []string{"production"},
				IsCurrent: true,
				TokenEstimate: &TokenEstimate{
					EstimatedGeneric: 25,
				},
			},
			{
				Version:   1,
				CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				SourceLen: 50,
				IsCurrent: false,
				TokenEstimate: &TokenEstimate{
					EstimatedGeneric: 12,
				},
			},
		},
	}

	str := history.String()

	assert.Contains(t, str, "my-template")
	assert.Contains(t, str, "Current: v2")
	assert.Contains(t, str, "Total: 2 versions")
	assert.Contains(t, str, "[CURRENT]")
	assert.Contains(t, str, "By: Alice")
	assert.Contains(t, str, "Tags: production")
	assert.Contains(t, str, "100 chars")
	assert.Contains(t, str, "~25 tokens")
}

func TestVersionDiff_IsSignificantChange(t *testing.T) {
	diff := &VersionDiff{
		ChangedLines: 5,
	}

	assert.True(t, diff.IsSignificantChange(5))
	assert.True(t, diff.IsSignificantChange(3))
	assert.False(t, diff.IsSignificantChange(10))
}

func BenchmarkGetVersionHistory(b *testing.B) {
	storage := NewMemoryStorage()
	engine, _ := NewStorageEngine(StorageEngineConfig{Storage: storage})
	defer engine.Close()

	ctx := context.Background()

	// Create 10 versions
	for i := 0; i < 10; i++ {
		_ = engine.Save(ctx, &StoredTemplate{
			Name:   "bench",
			Source: "Content version " + string(rune('0'+i)),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.GetVersionHistory(ctx, "bench")
	}
}

func BenchmarkCompareVersions(b *testing.B) {
	storage := NewMemoryStorage()
	engine, _ := NewStorageEngine(StorageEngineConfig{Storage: storage})
	defer engine.Close()

	ctx := context.Background()

	_ = engine.Save(ctx, &StoredTemplate{
		Name:   "bench",
		Source: "Original content with multiple lines\nLine 2\nLine 3",
	})
	_ = engine.Save(ctx, &StoredTemplate{
		Name:   "bench",
		Source: "Modified content with different lines\nLine 2 changed\nLine 4",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.CompareVersions(ctx, "bench", 1, 2)
	}
}
