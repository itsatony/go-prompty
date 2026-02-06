package prompty

import (
	"context"
	"sync"
	"time"
)

// CachedStorage wraps any TemplateStorage with in-memory caching.
// It caches Get operations with configurable TTL and size limits.
type CachedStorage struct {
	storage TemplateStorage
	config  CacheConfig

	mu     sync.RWMutex
	cache  map[string]*cacheEntry
	byID   map[TemplateID]*cacheEntry
	closed bool
}

// CacheConfig configures the caching behavior.
type CacheConfig struct {
	// TTL is how long cached entries remain valid.
	// Default: 5 minutes.
	TTL time.Duration

	// MaxEntries is the maximum number of cached templates.
	// When exceeded, oldest entries are evicted.
	// Default: 1000.
	MaxEntries int

	// NegativeCacheTTL is how long to cache "not found" results.
	// Set to 0 to disable negative caching.
	// Default: 30 seconds.
	NegativeCacheTTL time.Duration
}

// DefaultCacheConfig returns the default caching configuration.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		TTL:              5 * time.Minute,
		MaxEntries:       1000,
		NegativeCacheTTL: 30 * time.Second,
	}
}

// cacheEntry represents a cached template.
type cacheEntry struct {
	template   *StoredTemplate
	notFound   bool
	cachedAt   time.Time
	accessedAt time.Time
	key        string
}

// NewCachedStorage wraps a storage with caching.
func NewCachedStorage(storage TemplateStorage, config CacheConfig) *CachedStorage {
	if config.TTL == 0 {
		config.TTL = 5 * time.Minute
	}
	if config.MaxEntries == 0 {
		config.MaxEntries = 1000
	}

	return &CachedStorage{
		storage: storage,
		config:  config,
		cache:   make(map[string]*cacheEntry),
		byID:    make(map[TemplateID]*cacheEntry),
	}
}

// Get retrieves a template, using cache when available.
func (s *CachedStorage) Get(ctx context.Context, name string) (*StoredTemplate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, NewStorageClosedError()
	}

	// Check cache
	entry, ok := s.cache[name]
	if ok && s.isValid(entry) {
		entry.accessedAt = time.Now()
		s.mu.RUnlock()

		if entry.notFound {
			return nil, NewStorageTemplateNotFoundError(name)
		}
		return copyStoredTemplate(entry.template), nil
	}
	s.mu.RUnlock()

	// Cache miss - fetch from storage
	tmpl, err := s.storage.Get(ctx, name)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, NewStorageClosedError()
	}

	if err != nil {
		// Cache negative result if configured
		if s.config.NegativeCacheTTL > 0 {
			s.addEntry(name, nil, true)
		}
		return nil, err
	}

	// Cache positive result
	s.addEntry(name, tmpl, false)
	return copyStoredTemplate(tmpl), nil
}

// GetByID retrieves a template by ID, using cache when available.
func (s *CachedStorage) GetByID(ctx context.Context, id TemplateID) (*StoredTemplate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, NewStorageClosedError()
	}

	// Check cache
	entry, ok := s.byID[id]
	if ok && s.isValid(entry) {
		entry.accessedAt = time.Now()
		s.mu.RUnlock()
		return copyStoredTemplate(entry.template), nil
	}
	s.mu.RUnlock()

	// Cache miss - fetch from storage
	return s.storage.GetByID(ctx, id)
}

// GetVersion retrieves a specific version (bypasses cache).
func (s *CachedStorage) GetVersion(ctx context.Context, name string, version int) (*StoredTemplate, error) {
	// Version-specific queries bypass cache for simplicity
	return s.storage.GetVersion(ctx, name, version)
}

// Save stores a template and invalidates cache.
func (s *CachedStorage) Save(ctx context.Context, tmpl *StoredTemplate) error {
	err := s.storage.Save(ctx, tmpl)
	if err != nil {
		return err
	}

	// Invalidate cache
	s.mu.Lock()
	s.invalidateName(tmpl.Name)
	s.mu.Unlock()

	return nil
}

// Delete removes a template and invalidates cache.
func (s *CachedStorage) Delete(ctx context.Context, name string) error {
	err := s.storage.Delete(ctx, name)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.invalidateName(name)
	s.mu.Unlock()

	return nil
}

// DeleteVersion removes a specific version and invalidates cache.
func (s *CachedStorage) DeleteVersion(ctx context.Context, name string, version int) error {
	err := s.storage.DeleteVersion(ctx, name, version)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.invalidateName(name)
	s.mu.Unlock()

	return nil
}

// List returns templates matching the query (bypasses cache).
func (s *CachedStorage) List(ctx context.Context, query *TemplateQuery) ([]*StoredTemplate, error) {
	return s.storage.List(ctx, query)
}

// Exists checks if a template exists (may use cache).
func (s *CachedStorage) Exists(ctx context.Context, name string) (bool, error) {
	// Check cache first
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return false, NewStorageClosedError()
	}

	entry, ok := s.cache[name]
	if ok && s.isValid(entry) {
		s.mu.RUnlock()
		return !entry.notFound, nil
	}
	s.mu.RUnlock()

	return s.storage.Exists(ctx, name)
}

// ListVersions returns version numbers (bypasses cache).
func (s *CachedStorage) ListVersions(ctx context.Context, name string) ([]int, error) {
	return s.storage.ListVersions(ctx, name)
}

// Close closes the cache and underlying storage.
func (s *CachedStorage) Close() error {
	s.mu.Lock()
	s.closed = true
	s.cache = nil
	s.byID = nil
	s.mu.Unlock()

	return s.storage.Close()
}

// Invalidate removes a template from the cache.
func (s *CachedStorage) Invalidate(name string) {
	s.mu.Lock()
	s.invalidateName(name)
	s.mu.Unlock()
}

// InvalidateAll clears the entire cache.
func (s *CachedStorage) InvalidateAll() {
	s.mu.Lock()
	s.cache = make(map[string]*cacheEntry)
	s.byID = make(map[TemplateID]*cacheEntry)
	s.mu.Unlock()
}

// Stats returns cache statistics.
func (s *CachedStorage) Stats() CacheStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var validCount, negativeCount int
	for _, entry := range s.cache {
		if s.isValid(entry) {
			if entry.notFound {
				negativeCount++
			} else {
				validCount++
			}
		}
	}

	return CacheStats{
		Entries:         len(s.cache),
		ValidEntries:    validCount,
		NegativeEntries: negativeCount,
	}
}

// CacheStats contains cache statistics.
type CacheStats struct {
	Entries         int
	ValidEntries    int
	NegativeEntries int
}

// isValid checks if a cache entry is still valid.
func (s *CachedStorage) isValid(entry *cacheEntry) bool {
	ttl := s.config.TTL
	if entry.notFound {
		ttl = s.config.NegativeCacheTTL
	}
	return time.Since(entry.cachedAt) < ttl
}

// addEntry adds an entry to the cache, evicting if necessary.
// Caller must hold write lock.
func (s *CachedStorage) addEntry(name string, tmpl *StoredTemplate, notFound bool) {
	// Evict if at capacity
	if len(s.cache) >= s.config.MaxEntries {
		s.evictOldest()
	}

	now := time.Now()
	entry := &cacheEntry{
		template:   tmpl,
		notFound:   notFound,
		cachedAt:   now,
		accessedAt: now,
		key:        name,
	}

	s.cache[name] = entry
	if tmpl != nil {
		s.byID[tmpl.ID] = entry
	}
}

// invalidateName removes a name from the cache.
// Caller must hold write lock.
func (s *CachedStorage) invalidateName(name string) {
	entry, ok := s.cache[name]
	if !ok {
		return
	}

	if entry.template != nil {
		delete(s.byID, entry.template.ID)
	}
	delete(s.cache, name)
}

// evictOldest removes the oldest accessed entry.
// Caller must hold write lock.
func (s *CachedStorage) evictOldest() {
	var oldest *cacheEntry
	for _, entry := range s.cache {
		if oldest == nil || entry.accessedAt.Before(oldest.accessedAt) {
			oldest = entry
		}
	}

	if oldest != nil {
		s.invalidateName(oldest.key)
	}
}
