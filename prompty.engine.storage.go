package prompty

import (
	"context"
	"sync"
)

// StorageEngine combines template storage with the execution engine.
// It provides a unified API for loading, caching, and executing
// templates from any storage backend.
type StorageEngine struct {
	engine  *Engine
	storage TemplateStorage

	// Parsed template cache
	mu           sync.RWMutex
	parsedCache  map[string]*parsedCacheEntry
	cacheEnabled bool
}

// parsedCacheEntry caches a parsed template with its version.
type parsedCacheEntry struct {
	template *Template
	version  int
}

// StorageEngineConfig configures the StorageEngine.
type StorageEngineConfig struct {
	// Storage is the template storage backend (required).
	Storage TemplateStorage

	// Engine is the template engine to use.
	// If nil, a new engine with default options is created.
	Engine *Engine

	// DisableParsedTemplateCache disables caching of parsed templates.
	// By default (false), templates are cached and only re-parsed when their version changes.
	// Set to true to disable caching and always re-parse templates.
	DisableParsedTemplateCache bool
}

// NewStorageEngine creates a new StorageEngine with the given configuration.
func NewStorageEngine(config StorageEngineConfig) (*StorageEngine, error) {
	if config.Storage == nil {
		return nil, &StorageError{Message: ErrMsgNilStorage}
	}

	engine := config.Engine
	if engine == nil {
		var err error
		engine, err = New()
		if err != nil {
			return nil, err
		}
	}

	// Caching is enabled by default (disabled only if explicitly set)
	cacheEnabled := !config.DisableParsedTemplateCache

	return &StorageEngine{
		engine:       engine,
		storage:      config.Storage,
		parsedCache:  make(map[string]*parsedCacheEntry),
		cacheEnabled: cacheEnabled,
	}, nil
}

// MustNewStorageEngine creates a new StorageEngine, panicking on error.
func MustNewStorageEngine(config StorageEngineConfig) *StorageEngine {
	se, err := NewStorageEngine(config)
	if err != nil {
		panic(err)
	}
	return se
}

// Execute executes a stored template by name with the given data.
// This is the primary method for executing templates from storage.
func (se *StorageEngine) Execute(ctx context.Context, templateName string, data map[string]any) (string, error) {
	// Load and parse template
	tmpl, err := se.loadAndParse(ctx, templateName)
	if err != nil {
		return "", err
	}

	// Execute the template
	return tmpl.Execute(ctx, data)
}

// ExecuteVersion executes a specific version of a stored template.
func (se *StorageEngine) ExecuteVersion(ctx context.Context, templateName string, version int, data map[string]any) (string, error) {
	// Load specific version (bypasses cache)
	stored, err := se.storage.GetVersion(ctx, templateName, version)
	if err != nil {
		return "", err
	}

	// Parse the template
	tmpl, err := se.engine.Parse(stored.Source)
	if err != nil {
		return "", err
	}

	// Execute the template
	return tmpl.Execute(ctx, data)
}

// ExecuteWithContext executes a stored template with a pre-built context.
func (se *StorageEngine) ExecuteWithContext(ctx context.Context, templateName string, execCtx *Context) (string, error) {
	tmpl, err := se.loadAndParse(ctx, templateName)
	if err != nil {
		return "", err
	}

	return tmpl.ExecuteWithContext(ctx, execCtx)
}

// Validate validates a stored template without executing it.
func (se *StorageEngine) Validate(ctx context.Context, templateName string) (*ValidationResult, error) {
	stored, err := se.storage.Get(ctx, templateName)
	if err != nil {
		return nil, err
	}

	return se.engine.Validate(stored.Source)
}

// ValidateVersion validates a specific version of a stored template.
func (se *StorageEngine) ValidateVersion(ctx context.Context, templateName string, version int) (*ValidationResult, error) {
	stored, err := se.storage.GetVersion(ctx, templateName, version)
	if err != nil {
		return nil, err
	}

	return se.engine.Validate(stored.Source)
}

// Save stores a new template or creates a new version.
// The template source is validated before saving.
func (se *StorageEngine) Save(ctx context.Context, tmpl *StoredTemplate) error {
	// Validate source before saving
	result, err := se.engine.Validate(tmpl.Source)
	if err != nil {
		return err
	}
	if !result.IsValid() {
		return &StorageError{
			Message: ErrMsgInvalidTemplateSource,
			Name:    tmpl.Name,
		}
	}

	// Save to storage
	if err := se.storage.Save(ctx, tmpl); err != nil {
		return err
	}

	// Invalidate parsed cache
	se.invalidateParsedCache(tmpl.Name)

	return nil
}

// SaveWithoutValidation stores a template without validation.
// Use with caution - invalid templates will fail at execution time.
func (se *StorageEngine) SaveWithoutValidation(ctx context.Context, tmpl *StoredTemplate) error {
	if err := se.storage.Save(ctx, tmpl); err != nil {
		return err
	}

	se.invalidateParsedCache(tmpl.Name)
	return nil
}

// Delete removes all versions of a template from storage.
func (se *StorageEngine) Delete(ctx context.Context, templateName string) error {
	if err := se.storage.Delete(ctx, templateName); err != nil {
		return err
	}

	se.invalidateParsedCache(templateName)
	return nil
}

// DeleteVersion removes a specific version of a template.
func (se *StorageEngine) DeleteVersion(ctx context.Context, templateName string, version int) error {
	if err := se.storage.DeleteVersion(ctx, templateName, version); err != nil {
		return err
	}

	se.invalidateParsedCache(templateName)
	return nil
}

// Get retrieves the latest version of a stored template.
func (se *StorageEngine) Get(ctx context.Context, templateName string) (*StoredTemplate, error) {
	return se.storage.Get(ctx, templateName)
}

// GetVersion retrieves a specific version of a stored template.
func (se *StorageEngine) GetVersion(ctx context.Context, templateName string, version int) (*StoredTemplate, error) {
	return se.storage.GetVersion(ctx, templateName, version)
}

// List returns templates matching the query.
func (se *StorageEngine) List(ctx context.Context, query *TemplateQuery) ([]*StoredTemplate, error) {
	return se.storage.List(ctx, query)
}

// Exists checks if a template exists in storage.
func (se *StorageEngine) Exists(ctx context.Context, templateName string) (bool, error) {
	return se.storage.Exists(ctx, templateName)
}

// ListVersions returns all version numbers for a template.
func (se *StorageEngine) ListVersions(ctx context.Context, templateName string) ([]int, error) {
	return se.storage.ListVersions(ctx, templateName)
}

// Engine returns the underlying template engine.
// Use this to access engine-specific functionality like registering resolvers.
func (se *StorageEngine) Engine() *Engine {
	return se.engine
}

// Storage returns the underlying storage backend.
func (se *StorageEngine) Storage() TemplateStorage {
	return se.storage
}

// Close closes the storage engine and underlying storage.
func (se *StorageEngine) Close() error {
	se.mu.Lock()
	se.parsedCache = nil
	se.mu.Unlock()

	return se.storage.Close()
}

// ClearParsedCache clears the parsed template cache.
func (se *StorageEngine) ClearParsedCache() {
	se.mu.Lock()
	se.parsedCache = make(map[string]*parsedCacheEntry)
	se.mu.Unlock()
}

// ParsedCacheStats returns statistics about the parsed template cache.
func (se *StorageEngine) ParsedCacheStats() ParsedCacheStats {
	se.mu.RLock()
	defer se.mu.RUnlock()

	return ParsedCacheStats{
		Entries: len(se.parsedCache),
		Enabled: se.cacheEnabled,
	}
}

// ParsedCacheStats contains parsed cache statistics.
type ParsedCacheStats struct {
	Entries int
	Enabled bool
}

// loadAndParse loads a template from storage and parses it.
// Uses caching to avoid re-parsing unchanged templates.
func (se *StorageEngine) loadAndParse(ctx context.Context, name string) (*Template, error) {
	// Load from storage
	stored, err := se.storage.Get(ctx, name)
	if err != nil {
		return nil, err
	}

	// Check parsed cache
	if se.cacheEnabled {
		se.mu.RLock()
		entry, ok := se.parsedCache[name]
		se.mu.RUnlock()

		if ok && entry.version == stored.Version {
			return entry.template, nil
		}
	}

	// Parse the template
	tmpl, err := se.engine.Parse(stored.Source)
	if err != nil {
		return nil, err
	}

	// Cache the parsed template
	if se.cacheEnabled {
		se.mu.Lock()
		se.parsedCache[name] = &parsedCacheEntry{
			template: tmpl,
			version:  stored.Version,
		}
		se.mu.Unlock()
	}

	return tmpl, nil
}

// invalidateParsedCache removes a template from the parsed cache.
func (se *StorageEngine) invalidateParsedCache(name string) {
	se.mu.Lock()
	delete(se.parsedCache, name)
	se.mu.Unlock()
}

// RegisterResolver registers a custom resolver with the underlying engine.
func (se *StorageEngine) RegisterResolver(resolver Resolver) error {
	return se.engine.Register(resolver)
}

// MustRegisterResolver registers a resolver, panicking on error.
func (se *StorageEngine) MustRegisterResolver(resolver Resolver) {
	se.engine.MustRegister(resolver)
}

// RegisterFunc registers a custom function with the underlying engine.
func (se *StorageEngine) RegisterFunc(f *Func) error {
	return se.engine.RegisterFunc(f)
}

// MustRegisterFunc registers a function, panicking on error.
func (se *StorageEngine) MustRegisterFunc(f *Func) {
	se.engine.MustRegisterFunc(f)
}

// Storage error messages
const (
	ErrMsgNilStorage            = "storage is nil"
	ErrMsgInvalidTemplateSource = "template source is invalid"
)
