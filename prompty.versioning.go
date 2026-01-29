package prompty

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// VersionInfo provides detailed information about a template version.
type VersionInfo struct {
	Version       int
	CreatedAt     time.Time
	UpdatedAt     time.Time
	CreatedBy     string
	Source        string
	SourceLen     int
	Tags          []string
	Metadata      map[string]string
	IsCurrent     bool
	TokenEstimate *TokenEstimate
	Status        DeploymentStatus // Deployment status (draft, active, deprecated, archived)
	Labels        []string         // Labels assigned to this version (e.g., "production", "staging")
}

// VersionDiff represents differences between two template versions.
type VersionDiff struct {
	OldVersion  int
	NewVersion  int
	OldSource   string
	NewSource   string
	AddedLines  []string
	RemovedLines []string
	ChangedLines int
	SameLines   int
	AddedTags   []string
	RemovedTags []string
}

// VersionHistory contains the complete version history for a template.
type VersionHistory struct {
	TemplateName      string
	CurrentVersion    int
	TotalVersions     int
	Versions          []VersionInfo
	OldestVersion     *VersionInfo
	NewestVersion     *VersionInfo
	ProductionVersion int            // Version labeled as "production", 0 if not set
	LabeledVersions   map[string]int // All label -> version mappings
}

// GetVersionHistory retrieves the complete version history for a template.
func (se *StorageEngine) GetVersionHistory(ctx context.Context, name string) (*VersionHistory, error) {
	// Get all version numbers
	versions, err := se.storage.ListVersions(ctx, name)
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, NewStorageTemplateNotFoundError(name)
	}

	// Get current (latest) template
	current, err := se.storage.Get(ctx, name)
	if err != nil {
		return nil, err
	}

	history := &VersionHistory{
		TemplateName:    name,
		CurrentVersion:  current.Version,
		TotalVersions:   len(versions),
		Versions:        make([]VersionInfo, 0, len(versions)),
		LabeledVersions: make(map[string]int),
	}

	// Get labels if storage supports them
	labelStorage, hasLabels := se.storage.(LabelStorage)
	if hasLabels {
		labels, err := labelStorage.ListLabels(ctx, name)
		if err == nil {
			for _, lbl := range labels {
				history.LabeledVersions[lbl.Label] = lbl.Version
				if lbl.Label == LabelProduction {
					history.ProductionVersion = lbl.Version
				}
			}
		}
	}

	// Create version labels map for quick lookup
	versionLabelsMap := make(map[int][]string)
	for label, version := range history.LabeledVersions {
		versionLabelsMap[version] = append(versionLabelsMap[version], label)
	}

	// Load each version
	for _, v := range versions {
		tmpl, err := se.storage.GetVersion(ctx, name, v)
		if err != nil {
			continue // Skip versions we can't load
		}

		info := VersionInfo{
			Version:       tmpl.Version,
			CreatedAt:     tmpl.CreatedAt,
			UpdatedAt:     tmpl.UpdatedAt,
			CreatedBy:     tmpl.CreatedBy,
			Source:        tmpl.Source,
			SourceLen:     len(tmpl.Source),
			Tags:          tmpl.Tags,
			Metadata:      tmpl.Metadata,
			IsCurrent:     tmpl.Version == current.Version,
			TokenEstimate: EstimateTokens(tmpl.Source),
			Status:        tmpl.Status,
			Labels:        versionLabelsMap[tmpl.Version],
		}
		if info.Labels == nil {
			info.Labels = []string{}
		}
		history.Versions = append(history.Versions, info)
	}

	// Set oldest and newest
	if len(history.Versions) > 0 {
		oldest := &history.Versions[len(history.Versions)-1]
		newest := &history.Versions[0]
		history.OldestVersion = oldest
		history.NewestVersion = newest
	}

	return history, nil
}

// CompareVersions compares two versions of a template.
func (se *StorageEngine) CompareVersions(ctx context.Context, name string, oldVersion, newVersion int) (*VersionDiff, error) {
	oldTmpl, err := se.storage.GetVersion(ctx, name, oldVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get version %d: %w", oldVersion, err)
	}

	newTmpl, err := se.storage.GetVersion(ctx, name, newVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get version %d: %w", newVersion, err)
	}

	return diffTemplates(oldTmpl, newTmpl), nil
}

// RollbackToVersion creates a new version based on an older version.
// This doesn't delete newer versions, it creates a new version from the old source.
func (se *StorageEngine) RollbackToVersion(ctx context.Context, name string, targetVersion int) (*StoredTemplate, error) {
	// Get the target version
	targetTmpl, err := se.storage.GetVersion(ctx, name, targetVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get version %d: %w", targetVersion, err)
	}

	// Create new template from target
	// Status is set to draft - rollbacks should be reviewed before activation
	newTmpl := &StoredTemplate{
		Name:     name,
		Source:   targetTmpl.Source,
		Tags:     targetTmpl.Tags,
		TenantID: targetTmpl.TenantID,
		Status:   DeploymentStatusDraft,
		Metadata: make(map[string]string),
	}

	// Copy metadata and mark as rollback
	for k, v := range targetTmpl.Metadata {
		newTmpl.Metadata[k] = v
	}
	newTmpl.Metadata[MetaKeyRollbackFromVersion] = fmt.Sprintf("%d", targetVersion)

	// Save creates a new version
	err = se.storage.Save(ctx, newTmpl)
	if err != nil {
		return nil, fmt.Errorf("failed to save rollback: %w", err)
	}

	return newTmpl, nil
}

// CloneVersion creates a new template from an existing template version.
func (se *StorageEngine) CloneVersion(ctx context.Context, sourceName string, sourceVersion int, newName string) (*StoredTemplate, error) {
	// Get source version
	sourceTmpl, err := se.storage.GetVersion(ctx, sourceName, sourceVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get source version: %w", err)
	}

	// Check new name doesn't exist
	exists, _ := se.storage.Exists(ctx, newName)
	if exists {
		return nil, fmt.Errorf("template '%s' already exists", newName)
	}

	// Create clone
	// Status is set to draft - cloned templates may need customization before activation
	clone := &StoredTemplate{
		Name:     newName,
		Source:   sourceTmpl.Source,
		Tags:     sourceTmpl.Tags,
		TenantID: sourceTmpl.TenantID,
		Status:   DeploymentStatusDraft,
		Metadata: make(map[string]string),
	}

	// Copy metadata and mark as clone
	for k, v := range sourceTmpl.Metadata {
		clone.Metadata[k] = v
	}
	clone.Metadata[MetaKeyClonedFrom] = sourceName
	clone.Metadata[MetaKeyClonedFromVersion] = fmt.Sprintf("%d", sourceVersion)

	err = se.storage.Save(ctx, clone)
	if err != nil {
		return nil, fmt.Errorf("failed to save clone: %w", err)
	}

	return clone, nil
}

// PruneOldVersions removes old versions, keeping only the most recent N versions.
func (se *StorageEngine) PruneOldVersions(ctx context.Context, name string, keepVersions int) (int, error) {
	if keepVersions < 1 {
		return 0, fmt.Errorf("must keep at least 1 version")
	}

	versions, err := se.storage.ListVersions(ctx, name)
	if err != nil {
		return 0, err
	}

	// Versions are sorted newest first, keep the first N
	if len(versions) <= keepVersions {
		return 0, nil // Nothing to prune
	}

	// Delete old versions
	toDelete := versions[keepVersions:]
	deleted := 0
	for _, v := range toDelete {
		err := se.storage.DeleteVersion(ctx, name, v)
		if err == nil {
			deleted++
		}
	}

	return deleted, nil
}

// GetVersionDelta returns changes between two consecutive versions.
func (se *StorageEngine) GetVersionDelta(ctx context.Context, name string, version int) (*VersionDiff, error) {
	if version <= 1 {
		return nil, fmt.Errorf("no previous version for version 1")
	}
	return se.CompareVersions(ctx, name, version-1, version)
}

// Helper functions

// diffTemplates creates a diff between two templates.
func diffTemplates(oldTmpl, newTmpl *StoredTemplate) *VersionDiff {
	diff := &VersionDiff{
		OldVersion: oldTmpl.Version,
		NewVersion: newTmpl.Version,
		OldSource:  oldTmpl.Source,
		NewSource:  newTmpl.Source,
	}

	// Simple line-by-line diff
	oldLines := strings.Split(oldTmpl.Source, "\n")
	newLines := strings.Split(newTmpl.Source, "\n")

	oldSet := make(map[string]bool)
	newSet := make(map[string]bool)

	for _, line := range oldLines {
		oldSet[line] = true
	}
	for _, line := range newLines {
		newSet[line] = true
	}

	// Find added and removed lines
	for _, line := range newLines {
		if !oldSet[line] {
			diff.AddedLines = append(diff.AddedLines, line)
		} else {
			diff.SameLines++
		}
	}
	for _, line := range oldLines {
		if !newSet[line] {
			diff.RemovedLines = append(diff.RemovedLines, line)
		}
	}

	diff.ChangedLines = len(diff.AddedLines) + len(diff.RemovedLines)

	// Find tag changes
	oldTagSet := make(map[string]bool)
	newTagSet := make(map[string]bool)
	for _, tag := range oldTmpl.Tags {
		oldTagSet[tag] = true
	}
	for _, tag := range newTmpl.Tags {
		newTagSet[tag] = true
	}

	for _, tag := range newTmpl.Tags {
		if !oldTagSet[tag] {
			diff.AddedTags = append(diff.AddedTags, tag)
		}
	}
	for _, tag := range oldTmpl.Tags {
		if !newTagSet[tag] {
			diff.RemovedTags = append(diff.RemovedTags, tag)
		}
	}

	return diff
}

// String returns a human-readable summary of the version diff.
func (d *VersionDiff) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Version %d -> %d\n", d.OldVersion, d.NewVersion))
	sb.WriteString(fmt.Sprintf("Lines: +%d -%d (=%d unchanged)\n",
		len(d.AddedLines), len(d.RemovedLines), d.SameLines))

	if len(d.AddedTags) > 0 {
		sb.WriteString(fmt.Sprintf("Tags added: %s\n", strings.Join(d.AddedTags, ", ")))
	}
	if len(d.RemovedTags) > 0 {
		sb.WriteString(fmt.Sprintf("Tags removed: %s\n", strings.Join(d.RemovedTags, ", ")))
	}

	if len(d.AddedLines) > 0 && len(d.AddedLines) <= 10 {
		sb.WriteString("\nAdded:\n")
		for _, line := range d.AddedLines {
			if line != "" {
				sb.WriteString(fmt.Sprintf("  + %s\n", line))
			}
		}
	}

	if len(d.RemovedLines) > 0 && len(d.RemovedLines) <= 10 {
		sb.WriteString("\nRemoved:\n")
		for _, line := range d.RemovedLines {
			if line != "" {
				sb.WriteString(fmt.Sprintf("  - %s\n", line))
			}
		}
	}

	return sb.String()
}

// String returns a human-readable summary of the version history.
func (h *VersionHistory) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("=== Version History: %s ===\n", h.TemplateName))
	sb.WriteString(fmt.Sprintf("Current: v%d | Total: %d versions\n", h.CurrentVersion, h.TotalVersions))
	if h.ProductionVersion > 0 {
		sb.WriteString(fmt.Sprintf("Production: v%d\n", h.ProductionVersion))
	}
	sb.WriteString("\n")

	for _, v := range h.Versions {
		current := ""
		if v.IsCurrent {
			current = " [CURRENT]"
		}
		sb.WriteString(fmt.Sprintf("v%d%s\n", v.Version, current))
		sb.WriteString(fmt.Sprintf("  Created: %s\n", v.CreatedAt.Format(time.RFC3339)))
		if v.CreatedBy != "" {
			sb.WriteString(fmt.Sprintf("  By: %s\n", v.CreatedBy))
		}
		if v.Status != "" {
			sb.WriteString(fmt.Sprintf("  Status: %s\n", v.Status))
		}
		if len(v.Labels) > 0 {
			sb.WriteString(fmt.Sprintf("  Labels: %s\n", strings.Join(v.Labels, ", ")))
		}
		sb.WriteString(fmt.Sprintf("  Size: %d chars (~%d tokens)\n", v.SourceLen, v.TokenEstimate.EstimatedGeneric))
		if len(v.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("  Tags: %s\n", strings.Join(v.Tags, ", ")))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// HasChanges returns true if there are any changes between versions.
func (d *VersionDiff) HasChanges() bool {
	return len(d.AddedLines) > 0 || len(d.RemovedLines) > 0 ||
		len(d.AddedTags) > 0 || len(d.RemovedTags) > 0
}

// IsSignificantChange returns true if changes exceed the threshold.
func (d *VersionDiff) IsSignificantChange(lineThreshold int) bool {
	return d.ChangedLines >= lineThreshold
}
