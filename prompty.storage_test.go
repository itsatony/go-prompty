package prompty

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoredTemplate_Fields(t *testing.T) {
	tmpl := &StoredTemplate{
		ID:        "tmpl_abc123",
		Name:      "greeting",
		Source:    "Hello, {~prompty.var name=\"name\" /~}!",
		Version:   1,
		Metadata:  map[string]string{"author": "test"},
		CreatedBy: "user_123",
		TenantID:  "tenant_456",
		Tags:      []string{"public", "greeting"},
	}

	assert.Equal(t, TemplateID("tmpl_abc123"), tmpl.ID)
	assert.Equal(t, "greeting", tmpl.Name)
	assert.Equal(t, 1, tmpl.Version)
	assert.Equal(t, "test", tmpl.Metadata["author"])
	assert.Equal(t, "user_123", tmpl.CreatedBy)
	assert.Equal(t, "tenant_456", tmpl.TenantID)
	assert.Contains(t, tmpl.Tags, "public")
}

func TestTemplateQuery_Fields(t *testing.T) {
	query := &TemplateQuery{
		TenantID:           "tenant_123",
		Tags:               []string{"public"},
		CreatedBy:          "user_456",
		NamePrefix:         "greet",
		NameContains:       "ing",
		Limit:              10,
		Offset:             5,
		IncludeAllVersions: true,
	}

	assert.Equal(t, "tenant_123", query.TenantID)
	assert.Contains(t, query.Tags, "public")
	assert.Equal(t, "user_456", query.CreatedBy)
	assert.Equal(t, "greet", query.NamePrefix)
	assert.Equal(t, "ing", query.NameContains)
	assert.Equal(t, 10, query.Limit)
	assert.Equal(t, 5, query.Offset)
	assert.True(t, query.IncludeAllVersions)
}

// mockStorageDriver implements StorageDriver for testing
type mockStorageDriver struct {
	storage TemplateStorage
	err     error
}

func (d *mockStorageDriver) Open(connectionString string) (TemplateStorage, error) {
	if d.err != nil {
		return nil, d.err
	}
	return d.storage, nil
}

// mockTemplateStorage implements TemplateStorage for testing
type mockTemplateStorage struct {
	closed bool
}

func (s *mockTemplateStorage) Get(ctx context.Context, name string) (*StoredTemplate, error) {
	return nil, NewStorageTemplateNotFoundError(name)
}

func (s *mockTemplateStorage) GetByID(ctx context.Context, id TemplateID) (*StoredTemplate, error) {
	return nil, NewStorageTemplateNotFoundError(string(id))
}

func (s *mockTemplateStorage) GetVersion(ctx context.Context, name string, version int) (*StoredTemplate, error) {
	return nil, NewStorageVersionNotFoundError(name, version)
}

func (s *mockTemplateStorage) Save(ctx context.Context, tmpl *StoredTemplate) error {
	return nil
}

func (s *mockTemplateStorage) Delete(ctx context.Context, name string) error {
	return nil
}

func (s *mockTemplateStorage) DeleteVersion(ctx context.Context, name string, version int) error {
	return nil
}

func (s *mockTemplateStorage) List(ctx context.Context, query *TemplateQuery) ([]*StoredTemplate, error) {
	return nil, nil
}

func (s *mockTemplateStorage) Exists(ctx context.Context, name string) (bool, error) {
	return false, nil
}

func (s *mockTemplateStorage) ListVersions(ctx context.Context, name string) ([]int, error) {
	return nil, nil
}

func (s *mockTemplateStorage) Close() error {
	s.closed = true
	return nil
}

func TestRegisterStorageDriver(t *testing.T) {
	// Clean up after test
	defer func() {
		storageDriversMu.Lock()
		delete(storageDrivers, "test-driver")
		storageDriversMu.Unlock()
	}()

	driver := &mockStorageDriver{storage: &mockTemplateStorage{}}
	RegisterStorageDriver("test-driver", driver)

	// Verify it's registered
	drivers := ListStorageDrivers()
	assert.Contains(t, drivers, "test-driver")
}

func TestRegisterStorageDriver_PanicsOnNil(t *testing.T) {
	assert.Panics(t, func() {
		RegisterStorageDriver("nil-driver", nil)
	})
}

func TestRegisterStorageDriver_PanicsOnDuplicate(t *testing.T) {
	// Clean up after test
	defer func() {
		storageDriversMu.Lock()
		delete(storageDrivers, "dup-driver")
		storageDriversMu.Unlock()
	}()

	driver := &mockStorageDriver{storage: &mockTemplateStorage{}}
	RegisterStorageDriver("dup-driver", driver)

	assert.Panics(t, func() {
		RegisterStorageDriver("dup-driver", driver)
	})
}

func TestOpenStorage(t *testing.T) {
	// Clean up after test
	defer func() {
		storageDriversMu.Lock()
		delete(storageDrivers, "open-test")
		storageDriversMu.Unlock()
	}()

	mockStorage := &mockTemplateStorage{}
	driver := &mockStorageDriver{storage: mockStorage}
	RegisterStorageDriver("open-test", driver)

	storage, err := OpenStorage("open-test", "connection-string")
	require.NoError(t, err)
	assert.Equal(t, mockStorage, storage)
}

func TestOpenStorage_DriverNotFound(t *testing.T) {
	_, err := OpenStorage("nonexistent-driver", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "storage driver not found")
	assert.Contains(t, err.Error(), "nonexistent-driver")
}

func TestListStorageDrivers(t *testing.T) {
	// Clean up after test
	defer func() {
		storageDriversMu.Lock()
		delete(storageDrivers, "list-test-1")
		delete(storageDrivers, "list-test-2")
		storageDriversMu.Unlock()
	}()

	driver := &mockStorageDriver{storage: &mockTemplateStorage{}}
	RegisterStorageDriver("list-test-1", driver)
	RegisterStorageDriver("list-test-2", driver)

	drivers := ListStorageDrivers()
	assert.Contains(t, drivers, "list-test-1")
	assert.Contains(t, drivers, "list-test-2")
}

func TestStorageError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *StorageError
		expected string
	}{
		{
			name:     "message only",
			err:      &StorageError{Message: "test error"},
			expected: "test error",
		},
		{
			name:     "with name",
			err:      &StorageError{Message: "not found", Name: "greeting"},
			expected: "not found: greeting",
		},
		{
			name:     "with name and version",
			err:      &StorageError{Message: "not found", Name: "greeting", Version: 3},
			expected: "not found: greeting v3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestStorageError_Unwrap(t *testing.T) {
	cause := assert.AnError
	err := &StorageError{
		Message: "wrapped error",
		Cause:   cause,
	}

	assert.Equal(t, cause, err.Unwrap())
}

func TestNewStorageVersionNotFoundError(t *testing.T) {
	err := NewStorageVersionNotFoundError("greeting", 5)
	assert.Contains(t, err.Error(), "version not found")
	assert.Contains(t, err.Error(), "greeting")
	assert.Contains(t, err.Error(), "v5")
}

func TestNewStorageClosedError(t *testing.T) {
	err := NewStorageClosedError()
	assert.Contains(t, err.Error(), "closed")
}

func TestIntToStr(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{123456, "123456"},
		{-1, "-1"},
		{-42, "-42"},
	}

	for _, tt := range tests {
		result := intToStr(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}
