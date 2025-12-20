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
//
// Directory structure:
//
//	<root>/
//	  <template-name>/
//	    v1.json
//	    v2.json
//	    ...
type FilesystemStorage struct {
	mu       sync.RWMutex
	root     string
	closed   bool
}

// FilesystemStorageDriver is the driver for creating FilesystemStorage instances.
type FilesystemStorageDriver struct{}

func init() {
	RegisterStorageDriver("filesystem", &FilesystemStorageDriver{})
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
	if err := os.MkdirAll(root, 0755); err != nil {
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

	if tmpl.Name == "" {
		return &StorageError{Message: ErrMsgInvalidTemplateName}
	}

	// Validate template name for filesystem safety
	if strings.ContainsAny(tmpl.Name, "/\\:*?\"<>|") {
		return &StorageError{Message: ErrMsgInvalidTemplateName, Name: tmpl.Name}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return NewStorageClosedError()
	}

	// Create template directory
	templateDir := filepath.Join(s.root, tmpl.Name)
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		return &StorageError{Message: ErrMsgCreateStorageDir, Name: templateDir, Cause: err}
	}

	// Determine next version
	versions, _ := s.listVersionsInternal(tmpl.Name)
	nextVersion := 1
	if len(versions) > 0 {
		nextVersion = versions[0] + 1
	}

	now := time.Now()

	// Create stored template with generated fields
	stored := &StoredTemplate{
		ID:        generateTemplateID(),
		Name:      tmpl.Name,
		Source:    tmpl.Source,
		Version:   nextVersion,
		Metadata:  copyStringMap(tmpl.Metadata),
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: tmpl.CreatedBy,
		TenantID:  tmpl.TenantID,
		Tags:      copyStringSlice(tmpl.Tags),
	}

	// Write to file
	filename := filepath.Join(templateDir, "v"+intToStr(nextVersion)+".json")
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return &StorageError{Message: ErrMsgMarshalTemplate, Name: tmpl.Name, Cause: err}
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return &StorageError{Message: ErrMsgWriteTemplate, Name: filename, Cause: err}
	}

	// Update input template with generated values
	tmpl.ID = stored.ID
	tmpl.Version = stored.Version
	tmpl.CreatedAt = stored.CreatedAt
	tmpl.UpdatedAt = stored.UpdatedAt

	return nil
}

// Delete removes all versions of a template by name.
func (s *FilesystemStorage) Delete(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
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

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return NewStorageClosedError()
	}

	filename := filepath.Join(s.root, name, "v"+intToStr(version)+".json")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return NewStorageVersionNotFoundError(name, version)
	}

	if err := os.Remove(filename); err != nil {
		return &StorageError{Message: ErrMsgDeleteTemplate, Name: filename, Cause: err}
	}

	// Remove directory if empty
	templateDir := filepath.Join(s.root, name)
	entries, err := os.ReadDir(templateDir)
	if err == nil && len(entries) == 0 {
		_ = os.Remove(templateDir)
	}

	return nil
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
		if strings.HasPrefix(filename, "v") && strings.HasSuffix(filename, ".json") {
			versionStr := filename[1 : len(filename)-5]
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
	filename := filepath.Join(s.root, name, "v"+intToStr(version)+".json")
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
)
