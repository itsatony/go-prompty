package prompty

import (
	"context"
	"sync"
	"time"
)

// TemplateID is a unique identifier for a stored template.
// Uses prefixed nanoID format (e.g., "tmpl_6ByTSYmGzT2c").
type TemplateID string

// StoredTemplate represents a template with metadata stored in a storage backend.
type StoredTemplate struct {
	// ID is the unique identifier for this template version.
	ID TemplateID `json:"id"`

	// Name is the template name used for lookups.
	Name string `json:"name"`

	// Source is the raw template source code.
	Source string `json:"source"`

	// Version is the version number (1, 2, 3, ...).
	// Higher versions are newer.
	Version int `json:"version"`

	// Metadata contains arbitrary key-value pairs for user-defined data.
	Metadata map[string]string `json:"metadata,omitempty"`

	// InferenceConfig holds parsed frontmatter configuration from the template source.
	// This coexists with Metadata and is automatically extracted when the template is parsed.
	// Use this for model configuration, input/output schemas, and sample data.
	InferenceConfig *InferenceConfig `json:"inference_config,omitempty"`

	// CreatedAt is when this version was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when this version was last modified.
	UpdatedAt time.Time `json:"updated_at"`

	// CreatedBy identifies who created this version (optional).
	CreatedBy string `json:"created_by,omitempty"`

	// TenantID for multi-tenant isolation (optional).
	TenantID string `json:"tenant_id,omitempty"`

	// Tags for categorization and querying.
	Tags []string `json:"tags,omitempty"`
}

// TemplateQuery defines filters for listing templates.
type TemplateQuery struct {
	// TenantID filters by tenant (empty matches all).
	TenantID string

	// Tags filters to templates having ALL specified tags.
	Tags []string

	// CreatedBy filters by creator.
	CreatedBy string

	// NamePrefix filters to names starting with this prefix.
	NamePrefix string

	// NameContains filters to names containing this substring.
	NameContains string

	// Limit is the maximum number of results (0 = no limit).
	Limit int

	// Offset is the number of results to skip (for pagination).
	Offset int

	// IncludeAllVersions includes all versions, not just latest.
	IncludeAllVersions bool
}

// TemplateStorage is the interface for pluggable storage backends.
// Implementations must be safe for concurrent use.
//
// The interface follows patterns from database/sql for familiarity:
// - Context for cancellation and timeouts
// - Explicit error returns
// - Close for resource cleanup
type TemplateStorage interface {
	// Get retrieves the latest version of a template by name.
	// Returns ErrTemplateNotFound if the template doesn't exist.
	Get(ctx context.Context, name string) (*StoredTemplate, error)

	// GetByID retrieves a specific template version by ID.
	// Returns ErrTemplateNotFound if the ID doesn't exist.
	GetByID(ctx context.Context, id TemplateID) (*StoredTemplate, error)

	// GetVersion retrieves a specific version of a template.
	// Returns ErrTemplateNotFound if the template or version doesn't exist.
	GetVersion(ctx context.Context, name string, version int) (*StoredTemplate, error)

	// Save stores a template. If a template with the same name exists,
	// a new version is created. The template's ID, Version, CreatedAt,
	// and UpdatedAt fields are set by the storage implementation.
	Save(ctx context.Context, tmpl *StoredTemplate) error

	// Delete removes all versions of a template by name.
	// Returns ErrTemplateNotFound if the template doesn't exist.
	Delete(ctx context.Context, name string) error

	// DeleteVersion removes a specific version of a template.
	// Returns ErrTemplateNotFound if the template or version doesn't exist.
	DeleteVersion(ctx context.Context, name string, version int) error

	// List returns templates matching the query.
	// Results are ordered by name, then by version (descending).
	List(ctx context.Context, query *TemplateQuery) ([]*StoredTemplate, error)

	// Exists checks if a template with the given name exists.
	Exists(ctx context.Context, name string) (bool, error)

	// ListVersions returns all version numbers for a template.
	// Returns empty slice if the template doesn't exist.
	ListVersions(ctx context.Context, name string) ([]int, error)

	// Close releases any resources held by the storage.
	// After Close, the storage should not be used.
	Close() error
}

// StorageDriver is a factory for creating storage instances.
// Drivers register themselves during init().
type StorageDriver interface {
	// Open creates a new storage instance with the given connection string.
	// The format of the connection string is driver-specific.
	Open(connectionString string) (TemplateStorage, error)
}

// Storage driver registry
var (
	storageDriversMu sync.RWMutex
	storageDrivers   = make(map[string]StorageDriver)
)

// RegisterStorageDriver registers a storage driver by name.
// This is typically called from a driver's init() function.
// Panics if a driver with the same name is already registered.
func RegisterStorageDriver(name string, driver StorageDriver) {
	storageDriversMu.Lock()
	defer storageDriversMu.Unlock()

	if driver == nil {
		panic(ErrMsgNilStorageDriver)
	}
	if _, exists := storageDrivers[name]; exists {
		panic(ErrMsgDriverAlreadyRegistered + ": " + name)
	}
	storageDrivers[name] = driver
}

// OpenStorage opens a storage connection using the named driver.
// The connection string format is driver-specific.
//
// Example:
//
//	storage, err := prompty.OpenStorage("memory", "")
//	storage, err := prompty.OpenStorage("filesystem", "/path/to/templates")
func OpenStorage(driverName, connectionString string) (TemplateStorage, error) {
	storageDriversMu.RLock()
	driver, ok := storageDrivers[driverName]
	storageDriversMu.RUnlock()

	if !ok {
		return nil, NewStorageDriverNotFoundError(driverName)
	}

	return driver.Open(connectionString)
}

// ListStorageDrivers returns the names of all registered storage drivers.
func ListStorageDrivers() []string {
	storageDriversMu.RLock()
	defer storageDriversMu.RUnlock()

	names := make([]string, 0, len(storageDrivers))
	for name := range storageDrivers {
		names = append(names, name)
	}
	return names
}

// Storage error message constants
const (
	ErrMsgNilStorageDriver        = "storage driver is nil"
	ErrMsgDriverAlreadyRegistered = "storage driver already registered"
	ErrMsgStorageDriverNotFound   = "storage driver not found"
	ErrMsgStorageClosed           = "storage is closed"
	ErrMsgInvalidTemplateID       = "invalid template ID"
	ErrMsgVersionNotFound         = "template version not found"
)

// Storage metadata key constants
const (
	MetaKeyDriverName = "driver"
)

// NewStorageDriverNotFoundError creates an error for missing storage driver.
func NewStorageDriverNotFoundError(name string) error {
	return &StorageError{
		Message: ErrMsgStorageDriverNotFound,
		Name:    name,
	}
}

// NewStorageTemplateNotFoundError creates an error for template not found in storage.
func NewStorageTemplateNotFoundError(name string) error {
	return NewTemplateNotFoundError(name)
}

// NewStorageVersionNotFoundError creates an error for version not found.
func NewStorageVersionNotFoundError(name string, version int) error {
	return &StorageError{
		Message:  ErrMsgVersionNotFound,
		Name:     name,
		Version:  version,
	}
}

// NewStorageClosedError creates an error for operations on closed storage.
func NewStorageClosedError() error {
	return &StorageError{
		Message: ErrMsgStorageClosed,
	}
}

// StorageError represents a storage-related error.
type StorageError struct {
	Message string
	Name    string
	Version int
	Cause   error
}

// Error implements the error interface.
func (e *StorageError) Error() string {
	if e.Name != "" && e.Version > 0 {
		return e.Message + ": " + e.Name + " v" + intToStr(e.Version)
	}
	if e.Name != "" {
		return e.Message + ": " + e.Name
	}
	return e.Message
}

// Unwrap returns the underlying cause.
func (e *StorageError) Unwrap() error {
	return e.Cause
}

// intToStr converts an int to string without importing strconv.
func intToStr(i int) string {
	if i == 0 {
		return "0"
	}

	var neg bool
	if i < 0 {
		neg = true
		i = -i
	}

	buf := make([]byte, 20)
	pos := len(buf)

	for i > 0 {
		pos--
		buf[pos] = byte(i%10) + '0'
		i /= 10
	}

	if neg {
		pos--
		buf[pos] = '-'
	}

	return string(buf[pos:])
}
