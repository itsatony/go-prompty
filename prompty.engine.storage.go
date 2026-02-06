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
// If the template source contains a config block and PromptConfig is not already set,
// the config is automatically extracted and populated.
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

	// Extract PromptConfig from source if not already set
	if tmpl.PromptConfig == nil && tmpl.Source != "" {
		parsed, err := se.engine.Parse(tmpl.Source)
		if err == nil && parsed.HasPrompt() {
			tmpl.PromptConfig = parsed.Prompt()
		}
		// If parsing fails, we skip setting PromptConfig (validation already passed)
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
	ErrMsgNilStorage                  = "storage is nil"
	ErrMsgInvalidTemplateSource       = "template source is invalid"
	ErrMsgStorageDoesNotSupportLabels = "storage backend does not support labels"
	ErrMsgStorageDoesNotSupportStatus = "storage backend does not support status"
)

// -----------------------------------------------------------------------------
// Label Operations
// -----------------------------------------------------------------------------

// labelStorage returns the storage as LabelStorage, or an error if unsupported.
func (se *StorageEngine) labelStorage() (LabelStorage, error) {
	if ls, ok := se.storage.(LabelStorage); ok {
		return ls, nil
	}
	return nil, &StorageError{Message: ErrMsgStorageDoesNotSupportLabels}
}

// statusStorage returns the storage as StatusStorage, or an error if unsupported.
func (se *StorageEngine) statusStorage() (StatusStorage, error) {
	if ss, ok := se.storage.(StatusStorage); ok {
		return ss, nil
	}
	return nil, &StorageError{Message: ErrMsgStorageDoesNotSupportStatus}
}

// SetLabel assigns a label to a specific template version.
func (se *StorageEngine) SetLabel(ctx context.Context, templateName, label string, version int) error {
	return se.SetLabelBy(ctx, templateName, label, version, "")
}

// SetLabelBy assigns a label to a specific template version with assignedBy metadata.
func (se *StorageEngine) SetLabelBy(ctx context.Context, templateName, label string, version int, assignedBy string) error {
	ls, err := se.labelStorage()
	if err != nil {
		return err
	}
	return ls.SetLabel(ctx, templateName, label, version, assignedBy)
}

// RemoveLabel removes a label from a template.
func (se *StorageEngine) RemoveLabel(ctx context.Context, templateName, label string) error {
	ls, err := se.labelStorage()
	if err != nil {
		return err
	}
	return ls.RemoveLabel(ctx, templateName, label)
}

// GetByLabel retrieves a template by its label.
func (se *StorageEngine) GetByLabel(ctx context.Context, templateName, label string) (*StoredTemplate, error) {
	ls, err := se.labelStorage()
	if err != nil {
		return nil, err
	}
	return ls.GetByLabel(ctx, templateName, label)
}

// ExecuteLabeled executes a template using a labeled version.
func (se *StorageEngine) ExecuteLabeled(ctx context.Context, templateName, label string, data map[string]any) (string, error) {
	ls, err := se.labelStorage()
	if err != nil {
		return "", err
	}

	// Get the labeled version
	stored, err := ls.GetByLabel(ctx, templateName, label)
	if err != nil {
		return "", err
	}

	// Parse and execute
	tmpl, err := se.engine.Parse(stored.Source)
	if err != nil {
		return "", err
	}

	return tmpl.Execute(ctx, data)
}

// ListLabels returns all labels for a template.
func (se *StorageEngine) ListLabels(ctx context.Context, templateName string) ([]*TemplateLabel, error) {
	ls, err := se.labelStorage()
	if err != nil {
		return nil, err
	}
	return ls.ListLabels(ctx, templateName)
}

// GetVersionLabels returns all labels assigned to a specific version.
func (se *StorageEngine) GetVersionLabels(ctx context.Context, templateName string, version int) ([]string, error) {
	ls, err := se.labelStorage()
	if err != nil {
		return nil, err
	}
	return ls.GetVersionLabels(ctx, templateName, version)
}

// -----------------------------------------------------------------------------
// Status Operations
// -----------------------------------------------------------------------------

// SetStatus updates the deployment status of a specific template version.
func (se *StorageEngine) SetStatus(ctx context.Context, templateName string, version int, status DeploymentStatus) error {
	return se.SetStatusBy(ctx, templateName, version, status, "")
}

// SetStatusBy updates the deployment status with changedBy metadata.
func (se *StorageEngine) SetStatusBy(ctx context.Context, templateName string, version int, status DeploymentStatus, changedBy string) error {
	ss, err := se.statusStorage()
	if err != nil {
		return err
	}
	return ss.SetStatus(ctx, templateName, version, status, changedBy)
}

// ListByStatus returns templates matching the given deployment status.
func (se *StorageEngine) ListByStatus(ctx context.Context, status DeploymentStatus, query *TemplateQuery) ([]*StoredTemplate, error) {
	ss, err := se.statusStorage()
	if err != nil {
		return nil, err
	}
	return ss.ListByStatus(ctx, status, query)
}

// -----------------------------------------------------------------------------
// Convenience Methods
// -----------------------------------------------------------------------------

// ExecuteProduction executes a template's "production" labeled version.
// This is a convenience wrapper around ExecuteLabeled with LabelProduction.
func (se *StorageEngine) ExecuteProduction(ctx context.Context, templateName string, data map[string]any) (string, error) {
	return se.ExecuteLabeled(ctx, templateName, LabelProduction, data)
}

// PromoteToProduction assigns the "production" label to a specific version.
// This is a convenience wrapper around SetLabel with LabelProduction.
func (se *StorageEngine) PromoteToProduction(ctx context.Context, templateName string, version int) error {
	return se.SetLabel(ctx, templateName, LabelProduction, version)
}

// PromoteToProductionBy assigns the "production" label with audit information.
func (se *StorageEngine) PromoteToProductionBy(ctx context.Context, templateName string, version int, promotedBy string) error {
	return se.SetLabelBy(ctx, templateName, LabelProduction, version, promotedBy)
}

// GetProduction retrieves a template's "production" labeled version.
// This is a convenience wrapper around GetByLabel with LabelProduction.
func (se *StorageEngine) GetProduction(ctx context.Context, templateName string) (*StoredTemplate, error) {
	return se.GetByLabel(ctx, templateName, LabelProduction)
}

// PromoteToStaging assigns the "staging" label to a specific version.
func (se *StorageEngine) PromoteToStaging(ctx context.Context, templateName string, version int) error {
	return se.SetLabel(ctx, templateName, LabelStaging, version)
}

// ExecuteStaging executes a template's "staging" labeled version.
func (se *StorageEngine) ExecuteStaging(ctx context.Context, templateName string, data map[string]any) (string, error) {
	return se.ExecuteLabeled(ctx, templateName, LabelStaging, data)
}

// GetActiveTemplates returns all templates with "active" status.
func (se *StorageEngine) GetActiveTemplates(ctx context.Context, query *TemplateQuery) ([]*StoredTemplate, error) {
	return se.ListByStatus(ctx, DeploymentStatusActive, query)
}

// ArchiveVersion marks a template version as archived.
func (se *StorageEngine) ArchiveVersion(ctx context.Context, templateName string, version int) error {
	return se.SetStatus(ctx, templateName, version, DeploymentStatusArchived)
}

// DeprecateVersion marks a template version as deprecated.
func (se *StorageEngine) DeprecateVersion(ctx context.Context, templateName string, version int) error {
	return se.SetStatus(ctx, templateName, version, DeploymentStatusDeprecated)
}

// ActivateVersion marks a template version as active.
func (se *StorageEngine) ActivateVersion(ctx context.Context, templateName string, version int) error {
	return se.SetStatus(ctx, templateName, version, DeploymentStatusActive)
}

// SupportsLabels returns true if the underlying storage supports labels.
func (se *StorageEngine) SupportsLabels() bool {
	_, ok := se.storage.(LabelStorage)
	return ok
}

// SupportsStatus returns true if the underlying storage supports deployment status.
func (se *StorageEngine) SupportsStatus() bool {
	_, ok := se.storage.(StatusStorage)
	return ok
}
