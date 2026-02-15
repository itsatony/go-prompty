package prompty

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// FilesystemStorage stores templates as files on the filesystem.
// Each template is stored as a JSON file with metadata.
// Versioning is supported through separate files per version.
// Labels are stored in a labels.json file per template directory.
//
// Directory structure:
//
//	<root>/
//	  <template-name>/
//	    v1.json
//	    v2.json
//	    labels.json  # maps label names to version numbers
//	    ...
//
// FilesystemStorage implements ExtendedTemplateStorage (includes LabelStorage and StatusStorage).
type FilesystemStorage struct {
	mu     sync.RWMutex
	root   string
	closed bool
}

// filesystemLabelsFile is the name of the labels file in each template directory.
const filesystemLabelsFile = "labels.json"

// filesystemLabelData represents the structure of the labels.json file.
type filesystemLabelData struct {
	Labels map[string]filesystemLabelEntry `json:"labels"`
}

// filesystemLabelEntry represents a single label in the labels.json file.
type filesystemLabelEntry struct {
	Version    int       `json:"version"`
	AssignedAt time.Time `json:"assigned_at"`
	AssignedBy string    `json:"assigned_by,omitempty"`
}

// FilesystemStorageDriver is the driver for creating FilesystemStorage instances.
type FilesystemStorageDriver struct{}

func init() {
	RegisterStorageDriver(StorageDriverNameFilesystem, &FilesystemStorageDriver{})
}

// Open creates a new FilesystemStorage instance.
// The connection string is the root directory path.
func (d *FilesystemStorageDriver) Open(connectionString string) (TemplateStorage, error) {
	return NewFilesystemStorage(connectionString)
}

// NewFilesystemStorage creates a new filesystem-based template storage.
// The root directory will be created if it doesn't exist.
func NewFilesystemStorage(root string) (*FilesystemStorage, error) {
	if root == "" {
		return nil, &StorageError{Message: ErrMsgInvalidStorageRoot}
	}

	// Create root directory if it doesn't exist
	if err := os.MkdirAll(root, FilesystemDirPermissions); err != nil {
		return nil, &StorageError{
			Message: ErrMsgCreateStorageDir,
			Name:    root,
			Cause:   err,
		}
	}

	return &FilesystemStorage{
		root: root,
	}, nil
}

// Get retrieves the latest version of a template by name.
func (s *FilesystemStorage) Get(ctx context.Context, name string) (*StoredTemplate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Validate template name for security
	if err := validateTemplateNameForFilesystem(name); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, NewStorageClosedError()
	}

	versions, err := s.listVersionsInternal(name)
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return nil, NewStorageTemplateNotFoundError(name)
	}

	// Latest version is first (sorted descending)
	return s.loadTemplate(name, versions[0])
}

// GetByID retrieves a specific template version by ID.
func (s *FilesystemStorage) GetByID(ctx context.Context, id TemplateID) (*StoredTemplate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, NewStorageClosedError()
	}

	// Scan all templates to find by ID (inefficient but correct)
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, &StorageError{Message: ErrMsgReadStorageDir, Cause: err}
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		versions, err := s.listVersionsInternal(entry.Name())
		if err != nil {
			continue
		}

		for _, version := range versions {
			tmpl, err := s.loadTemplate(entry.Name(), version)
			if err != nil {
				continue
			}
			if tmpl.ID == id {
				return tmpl, nil
			}
		}
	}

	return nil, NewStorageTemplateNotFoundError(string(id))
}

// GetVersion retrieves a specific version of a template.
func (s *FilesystemStorage) GetVersion(ctx context.Context, name string, version int) (*StoredTemplate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Validate template name for security
	if err := validateTemplateNameForFilesystem(name); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, NewStorageClosedError()
	}

	return s.loadTemplate(name, version)
}

// Save stores a template, creating a new version if one exists.
func (s *FilesystemStorage) Save(ctx context.Context, tmpl *StoredTemplate) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Validate template name for security
	if err := validateTemplateNameForFilesystem(tmpl.Name); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return NewStorageClosedError()
	}

	// Create template directory
	templateDir := filepath.Join(s.root, tmpl.Name)
	if err := os.MkdirAll(templateDir, FilesystemDirPermissions); err != nil {
		return &StorageError{Message: ErrMsgCreateStorageDir, Name: templateDir, Cause: err}
	}

	// Determine next version
	versions, _ := s.listVersionsInternal(tmpl.Name)
	nextVersion := 1
	if len(versions) > 0 {
		nextVersion = versions[0] + 1
	}

	now := time.Now()

	// Determine status - default to active if not specified
	status := tmpl.Status
	if status == "" {
		status = DeploymentStatusActive
	}

	// Create stored template with generated fields
	stored := &StoredTemplate{
		ID:           generateTemplateID(),
		Name:         tmpl.Name,
		Source:       tmpl.Source,
		Version:      nextVersion,
		Status:       status,
		Metadata:     copyStringMap(tmpl.Metadata),
		PromptConfig: tmpl.PromptConfig, // PromptConfig is immutable after parsing
		CreatedAt:    now,
		UpdatedAt:    now,
		CreatedBy:    tmpl.CreatedBy,
		TenantID:     tmpl.TenantID,
		Tags:         copyStringSlice(tmpl.Tags),
	}

	// Write to file
	filename := filepath.Join(templateDir, FilesystemVersionPrefix+intToStr(nextVersion)+FilesystemVersionSuffix)
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return &StorageError{Message: ErrMsgMarshalTemplate, Name: tmpl.Name, Cause: err}
	}

	if err := os.WriteFile(filename, data, FilesystemFilePermissions); err != nil {
		return &StorageError{Message: ErrMsgWriteTemplate, Name: filename, Cause: err}
	}

	// Update input template with generated values
	tmpl.ID = stored.ID
	tmpl.Version = stored.Version
	tmpl.Status = stored.Status
	tmpl.CreatedAt = stored.CreatedAt
	tmpl.UpdatedAt = stored.UpdatedAt

	return nil
}

// Delete removes all versions of a template by name.
func (s *FilesystemStorage) Delete(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Validate template name for security
	if err := validateTemplateNameForFilesystem(name); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return NewStorageClosedError()
	}

	templateDir := filepath.Join(s.root, name)
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return NewStorageTemplateNotFoundError(name)
	}

	if err := os.RemoveAll(templateDir); err != nil {
		return &StorageError{Message: ErrMsgDeleteTemplate, Name: name, Cause: err}
	}

	return nil
}

// DeleteVersion removes a specific version of a template.
func (s *FilesystemStorage) DeleteVersion(ctx context.Context, name string, version int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Validate template name for security
	if err := validateTemplateNameForFilesystem(name); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return NewStorageClosedError()
	}

	filename := filepath.Join(s.root, name, FilesystemVersionPrefix+intToStr(version)+FilesystemVersionSuffix)
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return NewStorageVersionNotFoundError(name, version)
	}

	if err := os.Remove(filename); err != nil {
		return &StorageError{Message: ErrMsgDeleteTemplate, Name: filename, Cause: err}
	}

	// Clean up labels pointing to this version
	s.cleanupLabelsForVersion(name, version)

	// Remove directory if empty (only version files remain, not labels.json)
	templateDir := filepath.Join(s.root, name)
	entries, err := os.ReadDir(templateDir)
	if err == nil {
		// Count non-label files
		versionFileCount := 0
		for _, entry := range entries {
			if !entry.IsDir() && entry.Name() != filesystemLabelsFile {
				versionFileCount++
			}
		}
		// If no version files remain, remove the entire directory (including labels.json)
		if versionFileCount == 0 {
			_ = os.RemoveAll(templateDir)
		}
	}

	return nil
}

// cleanupLabelsForVersion removes any labels pointing to a specific version.
// Called internally with lock held.
func (s *FilesystemStorage) cleanupLabelsForVersion(templateName string, version int) {
	labels, err := s.loadLabels(templateName)
	if err != nil {
		return // No labels file or error reading it
	}

	modified := false
	for label, entry := range labels.Labels {
		if entry.Version == version {
			delete(labels.Labels, label)
			modified = true
		}
	}

	if !modified {
		return
	}

	// Save updated labels or remove file if empty
	if len(labels.Labels) == 0 {
		labelsPath := filepath.Join(s.root, templateName, filesystemLabelsFile)
		_ = os.Remove(labelsPath)
	} else {
		_ = s.saveLabels(templateName, labels)
	}
}

// List returns templates matching the query.
func (s *FilesystemStorage) List(ctx context.Context, query *TemplateQuery) ([]*StoredTemplate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, NewStorageClosedError()
	}

	if query == nil {
		query = &TemplateQuery{}
	}

	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, &StorageError{Message: ErrMsgReadStorageDir, Cause: err}
	}

	var results []*StoredTemplate

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Apply name filters
		if query.NamePrefix != "" && !strings.HasPrefix(name, query.NamePrefix) {
			continue
		}
		if query.NameContains != "" && !strings.Contains(name, query.NameContains) {
			continue
		}

		versions, err := s.listVersionsInternal(name)
		if err != nil || len(versions) == 0 {
			continue
		}

		if query.IncludeAllVersions {
			for _, version := range versions {
				tmpl, err := s.loadTemplate(name, version)
				if err != nil {
					continue
				}
				if matchesTemplateQuery(tmpl, query) {
					results = append(results, tmpl)
				}
			}
		} else {
			// Only include latest version
			tmpl, err := s.loadTemplate(name, versions[0])
			if err != nil {
				continue
			}
			if matchesTemplateQuery(tmpl, query) {
				results = append(results, tmpl)
			}
		}
	}

	// Sort by name, then version descending
	sort.Slice(results, func(i, j int) bool {
		if results[i].Name != results[j].Name {
			return results[i].Name < results[j].Name
		}
		return results[i].Version > results[j].Version
	})

	// Apply offset and limit
	if query.Offset > 0 {
		if query.Offset >= len(results) {
			return []*StoredTemplate{}, nil
		}
		results = results[query.Offset:]
	}

	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}

	return results, nil
}

// Exists checks if a template with the given name exists.
func (s *FilesystemStorage) Exists(ctx context.Context, name string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return false, NewStorageClosedError()
	}

	templateDir := filepath.Join(s.root, name)
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return false, nil
	}

	versions, _ := s.listVersionsInternal(name)
	return len(versions) > 0, nil
}

// ListVersions returns all version numbers for a template.
func (s *FilesystemStorage) ListVersions(ctx context.Context, name string) ([]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, NewStorageClosedError()
	}

	return s.listVersionsInternal(name)
}

// Close marks the storage as closed.
func (s *FilesystemStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed = true
	return nil
}

// listVersionsInternal lists version numbers for a template (no locking).
func (s *FilesystemStorage) listVersionsInternal(name string) ([]int, error) {
	templateDir := filepath.Join(s.root, name)
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []int{}, nil
		}
		return nil, err
	}

	var versions []int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		if strings.HasPrefix(filename, FilesystemVersionPrefix) && strings.HasSuffix(filename, FilesystemVersionSuffix) {
			versionStr := filename[len(FilesystemVersionPrefix) : len(filename)-len(FilesystemVersionSuffix)]
			version := parseVersionNumber(versionStr)
			if version > 0 {
				versions = append(versions, version)
			}
		}
	}

	// Sort descending
	sort.Sort(sort.Reverse(sort.IntSlice(versions)))
	return versions, nil
}

// loadTemplate loads a template from disk.
func (s *FilesystemStorage) loadTemplate(name string, version int) (*StoredTemplate, error) {
	filename := filepath.Join(s.root, name, FilesystemVersionPrefix+intToStr(version)+FilesystemVersionSuffix)
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewStorageVersionNotFoundError(name, version)
		}
		return nil, &StorageError{Message: ErrMsgReadTemplate, Name: filename, Cause: err}
	}

	var tmpl StoredTemplate
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return nil, &StorageError{Message: ErrMsgUnmarshalTemplate, Name: filename, Cause: err}
	}

	return &tmpl, nil
}

// parseVersionNumber parses a version number string.
func parseVersionNumber(s string) int {
	if s == "" {
		return 0
	}
	result := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		result = result*10 + int(c-'0')
	}
	return result
}

// -----------------------------------------------------------------------------
// LabelStorage Implementation
// -----------------------------------------------------------------------------

// SetLabel assigns a label to a specific template version.
func (s *FilesystemStorage) SetLabel(ctx context.Context, templateName, label string, version int, assignedBy string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Validate label
	if err := ValidateLabel(label); err != nil {
		return err
	}

	// Validate template name for security
	if err := validateTemplateNameForFilesystem(templateName); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return NewStorageClosedError()
	}

	// Verify template and version exist
	_, err := s.loadTemplate(templateName, version)
	if err != nil {
		return err
	}

	// Load existing labels
	labels, err := s.loadLabels(templateName)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if labels == nil {
		labels = &filesystemLabelData{Labels: make(map[string]filesystemLabelEntry)}
	}

	// Update label
	labels.Labels[label] = filesystemLabelEntry{
		Version:    version,
		AssignedAt: time.Now(),
		AssignedBy: assignedBy,
	}

	// Save labels
	return s.saveLabels(templateName, labels)
}

// RemoveLabel removes a label from a template.
func (s *FilesystemStorage) RemoveLabel(ctx context.Context, templateName, label string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Validate template name for security
	if err := validateTemplateNameForFilesystem(templateName); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return NewStorageClosedError()
	}

	// Load existing labels
	labels, err := s.loadLabels(templateName)
	if err != nil {
		if os.IsNotExist(err) {
			return NewStorageLabelNotFoundError(templateName, label)
		}
		return err
	}

	if _, exists := labels.Labels[label]; !exists {
		return NewStorageLabelNotFoundError(templateName, label)
	}

	delete(labels.Labels, label)

	// Save labels (or remove file if empty)
	if len(labels.Labels) == 0 {
		labelsPath := filepath.Join(s.root, templateName, filesystemLabelsFile)
		_ = os.Remove(labelsPath)
		return nil
	}

	return s.saveLabels(templateName, labels)
}

// GetByLabel retrieves a template by its label.
func (s *FilesystemStorage) GetByLabel(ctx context.Context, templateName, label string) (*StoredTemplate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Validate template name for security
	if err := validateTemplateNameForFilesystem(templateName); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, NewStorageClosedError()
	}

	// Load labels
	labels, err := s.loadLabels(templateName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewStorageLabelNotFoundError(templateName, label)
		}
		return nil, err
	}

	entry, exists := labels.Labels[label]
	if !exists {
		return nil, NewStorageLabelNotFoundError(templateName, label)
	}

	return s.loadTemplate(templateName, entry.Version)
}

// ListLabels returns all labels for a template.
func (s *FilesystemStorage) ListLabels(ctx context.Context, templateName string) ([]*TemplateLabel, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Validate template name for security
	if err := validateTemplateNameForFilesystem(templateName); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, NewStorageClosedError()
	}

	labels, err := s.loadLabels(templateName)
	if err != nil {
		if os.IsNotExist(err) {
			return []*TemplateLabel{}, nil
		}
		return nil, err
	}

	result := make([]*TemplateLabel, 0, len(labels.Labels))
	for label, entry := range labels.Labels {
		result = append(result, &TemplateLabel{
			TemplateName: templateName,
			Label:        label,
			Version:      entry.Version,
			AssignedAt:   entry.AssignedAt,
			AssignedBy:   entry.AssignedBy,
		})
	}

	return result, nil
}

// GetVersionLabels returns all labels assigned to a specific version.
func (s *FilesystemStorage) GetVersionLabels(ctx context.Context, templateName string, version int) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Validate template name for security
	if err := validateTemplateNameForFilesystem(templateName); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, NewStorageClosedError()
	}

	labels, err := s.loadLabels(templateName)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var result []string
	for label, entry := range labels.Labels {
		if entry.Version == version {
			result = append(result, label)
		}
	}

	if result == nil {
		result = []string{}
	}

	return result, nil
}

// loadLabels loads labels from the labels.json file.
func (s *FilesystemStorage) loadLabels(templateName string) (*filesystemLabelData, error) {
	labelsPath := filepath.Join(s.root, templateName, filesystemLabelsFile)
	data, err := os.ReadFile(labelsPath)
	if err != nil {
		return nil, err
	}

	var labels filesystemLabelData
	if err := json.Unmarshal(data, &labels); err != nil {
		return nil, &StorageError{Message: ErrMsgUnmarshalLabels, Name: labelsPath, Cause: err}
	}

	return &labels, nil
}

// saveLabels saves labels to the labels.json file.
func (s *FilesystemStorage) saveLabels(templateName string, labels *filesystemLabelData) error {
	labelsPath := filepath.Join(s.root, templateName, filesystemLabelsFile)

	data, err := json.MarshalIndent(labels, "", "  ")
	if err != nil {
		return &StorageError{Message: ErrMsgMarshalLabels, Name: templateName, Cause: err}
	}

	if err := os.WriteFile(labelsPath, data, FilesystemFilePermissions); err != nil {
		return &StorageError{Message: ErrMsgWriteLabels, Name: labelsPath, Cause: err}
	}

	return nil
}

// -----------------------------------------------------------------------------
// StatusStorage Implementation
// -----------------------------------------------------------------------------

// SetStatus updates the deployment status of a specific version.
func (s *FilesystemStorage) SetStatus(ctx context.Context, templateName string, version int, status DeploymentStatus, changedBy string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Validate status
	if !status.IsValid() {
		return NewInvalidDeploymentStatusError(string(status))
	}

	// Validate template name for security
	if err := validateTemplateNameForFilesystem(templateName); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return NewStorageClosedError()
	}

	// Load the template
	tmpl, err := s.loadTemplate(templateName, version)
	if err != nil {
		return err
	}

	// Check if current status is archived (terminal state)
	if tmpl.Status == DeploymentStatusArchived {
		return NewArchivedVersionError(templateName, version)
	}

	// Validate status transition
	if tmpl.Status != "" && !CanTransitionStatus(tmpl.Status, status) {
		return NewInvalidStatusTransitionError(tmpl.Status, status)
	}

	// Update status
	tmpl.Status = status
	tmpl.UpdatedAt = time.Now()

	// Save back to file
	filename := filepath.Join(s.root, templateName, FilesystemVersionPrefix+intToStr(version)+FilesystemVersionSuffix)
	data, err := json.MarshalIndent(tmpl, "", "  ")
	if err != nil {
		return &StorageError{Message: ErrMsgMarshalTemplate, Name: templateName, Cause: err}
	}

	if err := os.WriteFile(filename, data, FilesystemFilePermissions); err != nil {
		return &StorageError{Message: ErrMsgWriteTemplate, Name: filename, Cause: err}
	}

	return nil
}

// ListByStatus returns templates matching the given deployment status.
func (s *FilesystemStorage) ListByStatus(ctx context.Context, status DeploymentStatus, query *TemplateQuery) ([]*StoredTemplate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Create a query with the status filter
	if query == nil {
		query = &TemplateQuery{}
	}

	// Create a copy of query with status set
	queryCopy := *query
	queryCopy.Status = status

	return s.List(ctx, &queryCopy)
}

// Ensure FilesystemStorage implements ExtendedTemplateStorage
var _ ExtendedTemplateStorage = (*FilesystemStorage)(nil)

// Additional storage error messages
const (
	ErrMsgInvalidStorageRoot = "invalid storage root path"
	ErrMsgCreateStorageDir   = "failed to create storage directory"
	ErrMsgReadStorageDir     = "failed to read storage directory"
	ErrMsgMarshalTemplate    = "failed to marshal template"
	ErrMsgUnmarshalTemplate  = "failed to unmarshal template"
	ErrMsgWriteTemplate      = "failed to write template file"
	ErrMsgReadTemplate       = "failed to read template file"
	ErrMsgDeleteTemplate     = "failed to delete template"
	ErrMsgMarshalLabels      = "failed to marshal labels"
	ErrMsgUnmarshalLabels    = "failed to unmarshal labels"
	ErrMsgWriteLabels        = "failed to write labels file"
)

// validateTemplateNameForFilesystem validates a template name for filesystem safety.
// Prevents path traversal attacks and invalid filesystem characters.
func validateTemplateNameForFilesystem(name string) error {
	if name == "" {
		return &StorageError{Message: ErrMsgInvalidTemplateName}
	}
	// Check for path traversal attempts
	if strings.Contains(name, "..") {
		return &StorageError{Message: ErrMsgPathTraversalDetected, Name: name}
	}
	// Check for invalid filesystem characters
	if strings.ContainsAny(name, "/\\:*?\"<>|") {
		return &StorageError{Message: ErrMsgInvalidTemplateName, Name: name}
	}
	return nil
}
