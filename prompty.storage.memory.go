package prompty

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryStorage is an in-memory implementation of TemplateStorage.
// It is primarily intended for testing and development.
// All data is lost when the process terminates.
type MemoryStorage struct {
	mu       sync.RWMutex
	templates map[string][]*StoredTemplate // name -> versions (sorted by version desc)
	byID     map[TemplateID]*StoredTemplate
	closed   bool
}

// MemoryStorageDriver is the driver for creating MemoryStorage instances.
type MemoryStorageDriver struct{}

func init() {
	RegisterStorageDriver("memory", &MemoryStorageDriver{})
}

// Open creates a new MemoryStorage instance.
// The connection string is ignored for memory storage.
func (d *MemoryStorageDriver) Open(connectionString string) (TemplateStorage, error) {
	return NewMemoryStorage(), nil
}

// NewMemoryStorage creates a new in-memory template storage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		templates: make(map[string][]*StoredTemplate),
		byID:     make(map[TemplateID]*StoredTemplate),
	}
}

// Get retrieves the latest version of a template by name.
func (s *MemoryStorage) Get(ctx context.Context, name string) (*StoredTemplate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, NewStorageClosedError()
	}

	versions, ok := s.templates[name]
	if !ok || len(versions) == 0 {
		return nil, NewStorageTemplateNotFoundError(name)
	}

	// Return copy of the latest version (first in slice, sorted desc)
	return copyStoredTemplate(versions[0]), nil
}

// GetByID retrieves a specific template version by ID.
func (s *MemoryStorage) GetByID(ctx context.Context, id TemplateID) (*StoredTemplate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, NewStorageClosedError()
	}

	tmpl, ok := s.byID[id]
	if !ok {
		return nil, NewStorageTemplateNotFoundError(string(id))
	}

	return copyStoredTemplate(tmpl), nil
}

// GetVersion retrieves a specific version of a template.
func (s *MemoryStorage) GetVersion(ctx context.Context, name string, version int) (*StoredTemplate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, NewStorageClosedError()
	}

	versions, ok := s.templates[name]
	if !ok {
		return nil, NewStorageVersionNotFoundError(name, version)
	}

	for _, tmpl := range versions {
		if tmpl.Version == version {
			return copyStoredTemplate(tmpl), nil
		}
	}

	return nil, NewStorageVersionNotFoundError(name, version)
}

// Save stores a template, creating a new version if one exists.
func (s *MemoryStorage) Save(ctx context.Context, tmpl *StoredTemplate) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if tmpl.Name == "" {
		return &StorageError{Message: ErrMsgInvalidTemplateName}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return NewStorageClosedError()
	}

	now := time.Now()
	versions := s.templates[tmpl.Name]

	// Determine next version number
	nextVersion := 1
	if len(versions) > 0 {
		nextVersion = versions[0].Version + 1
	}

	// Create new stored template with generated fields
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

	// Update input template with generated values
	tmpl.ID = stored.ID
	tmpl.Version = stored.Version
	tmpl.CreatedAt = stored.CreatedAt
	tmpl.UpdatedAt = stored.UpdatedAt

	// Insert at beginning (newest first)
	s.templates[tmpl.Name] = append([]*StoredTemplate{stored}, versions...)
	s.byID[stored.ID] = stored

	return nil
}

// Delete removes all versions of a template by name.
func (s *MemoryStorage) Delete(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return NewStorageClosedError()
	}

	versions, ok := s.templates[name]
	if !ok {
		return NewStorageTemplateNotFoundError(name)
	}

	// Remove all versions from byID index
	for _, tmpl := range versions {
		delete(s.byID, tmpl.ID)
	}

	delete(s.templates, name)
	return nil
}

// DeleteVersion removes a specific version of a template.
func (s *MemoryStorage) DeleteVersion(ctx context.Context, name string, version int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return NewStorageClosedError()
	}

	versions, ok := s.templates[name]
	if !ok {
		return NewStorageVersionNotFoundError(name, version)
	}

	for i, tmpl := range versions {
		if tmpl.Version == version {
			// Remove from byID index
			delete(s.byID, tmpl.ID)

			// Remove from versions slice
			s.templates[name] = append(versions[:i], versions[i+1:]...)

			// Clean up if no versions left
			if len(s.templates[name]) == 0 {
				delete(s.templates, name)
			}

			return nil
		}
	}

	return NewStorageVersionNotFoundError(name, version)
}

// List returns templates matching the query.
func (s *MemoryStorage) List(ctx context.Context, query *TemplateQuery) ([]*StoredTemplate, error) {
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

	var results []*StoredTemplate

	// Collect matching templates
	for name, versions := range s.templates {
		if !matchesQuery(name, versions, query) {
			continue
		}

		if query.IncludeAllVersions {
			for _, tmpl := range versions {
				if matchesTemplateQuery(tmpl, query) {
					results = append(results, copyStoredTemplate(tmpl))
				}
			}
		} else if len(versions) > 0 {
			// Only include latest version
			if matchesTemplateQuery(versions[0], query) {
				results = append(results, copyStoredTemplate(versions[0]))
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
func (s *MemoryStorage) Exists(ctx context.Context, name string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return false, NewStorageClosedError()
	}

	versions, ok := s.templates[name]
	return ok && len(versions) > 0, nil
}

// ListVersions returns all version numbers for a template.
func (s *MemoryStorage) ListVersions(ctx context.Context, name string) ([]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, NewStorageClosedError()
	}

	versions, ok := s.templates[name]
	if !ok {
		return []int{}, nil
	}

	result := make([]int, len(versions))
	for i, tmpl := range versions {
		result[i] = tmpl.Version
	}

	return result, nil
}

// Close marks the storage as closed.
func (s *MemoryStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed = true
	s.templates = nil
	s.byID = nil
	return nil
}

// matchesQuery checks if a template name matches the query filters.
func matchesQuery(name string, versions []*StoredTemplate, query *TemplateQuery) bool {
	if query.NamePrefix != "" && !strings.HasPrefix(name, query.NamePrefix) {
		return false
	}
	if query.NameContains != "" && !strings.Contains(name, query.NameContains) {
		return false
	}
	return true
}

// matchesTemplateQuery checks if a template matches additional query filters.
func matchesTemplateQuery(tmpl *StoredTemplate, query *TemplateQuery) bool {
	if query.TenantID != "" && tmpl.TenantID != query.TenantID {
		return false
	}
	if query.CreatedBy != "" && tmpl.CreatedBy != query.CreatedBy {
		return false
	}
	if len(query.Tags) > 0 {
		for _, tag := range query.Tags {
			if !containsString(tmpl.Tags, tag) {
				return false
			}
		}
	}
	return true
}

// containsString checks if a slice contains a string.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// generateTemplateID generates a unique template ID.
func generateTemplateID() TemplateID {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	id := base64.RawURLEncoding.EncodeToString(b)
	return TemplateID("tmpl_" + id)
}

// copyStoredTemplate creates a deep copy of a StoredTemplate.
func copyStoredTemplate(tmpl *StoredTemplate) *StoredTemplate {
	if tmpl == nil {
		return nil
	}
	return &StoredTemplate{
		ID:        tmpl.ID,
		Name:      tmpl.Name,
		Source:    tmpl.Source,
		Version:   tmpl.Version,
		Metadata:  copyStringMap(tmpl.Metadata),
		CreatedAt: tmpl.CreatedAt,
		UpdatedAt: tmpl.UpdatedAt,
		CreatedBy: tmpl.CreatedBy,
		TenantID:  tmpl.TenantID,
		Tags:      copyStringSlice(tmpl.Tags),
	}
}

// copyStringMap creates a copy of a string map.
func copyStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// copyStringSlice creates a copy of a string slice.
func copyStringSlice(s []string) []string {
	if s == nil {
		return nil
	}
	result := make([]string, len(s))
	copy(result, s)
	return result
}
