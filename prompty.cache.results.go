package prompty

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

// ResultCache provides caching for template execution results.
// It caches the output of Execute() calls based on template name/source
// and input data, avoiding redundant template execution.
type ResultCache struct {
	mu        sync.RWMutex
	entries   map[string]*resultCacheEntry
	config    ResultCacheConfig
	stats     ResultCacheStats
	evictList []string // LRU tracking
}

// resultCacheEntry holds a cached result with metadata.
type resultCacheEntry struct {
	Result    string
	CreatedAt time.Time
	ExpiresAt time.Time
	HitCount  int
	DataHash  string
}

// ResultCacheConfig configures the result cache behavior.
type ResultCacheConfig struct {
	// TTL is how long results are cached. Default: 5 minutes.
	TTL time.Duration

	// MaxEntries is the maximum number of cached results. Default: 1000.
	MaxEntries int

	// MaxResultSize is the maximum size of a result to cache (bytes). Default: 1MB.
	MaxResultSize int

	// KeyPrefix is prepended to all cache keys. Useful for namespacing.
	KeyPrefix string
}

// ResultCacheStats tracks cache performance metrics.
type ResultCacheStats struct {
	Hits       int64
	Misses     int64
	Evictions  int64
	TotalSize  int64
	EntryCount int
}

// DefaultResultCacheConfig returns sensible defaults for result caching.
func DefaultResultCacheConfig() ResultCacheConfig {
	return ResultCacheConfig{
		TTL:           5 * time.Minute,
		MaxEntries:    1000,
		MaxResultSize: 1 << 20, // 1MB
		KeyPrefix:     "",
	}
}

// NewResultCache creates a new result cache.
func NewResultCache(config ResultCacheConfig) *ResultCache {
	if config.TTL == 0 {
		config.TTL = 5 * time.Minute
	}
	if config.MaxEntries == 0 {
		config.MaxEntries = 1000
	}
	if config.MaxResultSize == 0 {
		config.MaxResultSize = 1 << 20
	}

	return &ResultCache{
		entries:   make(map[string]*resultCacheEntry),
		config:    config,
		evictList: make([]string, 0, config.MaxEntries),
	}
}

// Get retrieves a cached result if available and not expired.
func (c *ResultCache) Get(templateKey string, data map[string]any) (string, bool) {
	key := c.makeKey(templateKey, data)

	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		c.mu.Lock()
		c.stats.Misses++
		c.mu.Unlock()
		return "", false
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.stats.Misses++
		c.mu.Unlock()
		return "", false
	}

	c.mu.Lock()
	entry.HitCount++
	c.stats.Hits++
	c.mu.Unlock()

	return entry.Result, true
}

// Set stores a result in the cache.
func (c *ResultCache) Set(templateKey string, data map[string]any, result string) {
	// Don't cache results that exceed max size
	if len(result) > c.config.MaxResultSize {
		return
	}

	key := c.makeKey(templateKey, data)
	dataHash := c.hashData(data)
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict if at capacity
	if len(c.entries) >= c.config.MaxEntries {
		c.evictOldest()
	}

	c.entries[key] = &resultCacheEntry{
		Result:    result,
		CreatedAt: now,
		ExpiresAt: now.Add(c.config.TTL),
		HitCount:  0,
		DataHash:  dataHash,
	}
	c.evictList = append(c.evictList, key)
	c.stats.EntryCount = len(c.entries)
	c.stats.TotalSize += int64(len(result))
}

// Invalidate removes a specific cache entry.
func (c *ResultCache) Invalidate(templateKey string, data map[string]any) {
	key := c.makeKey(templateKey, data)

	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, exists := c.entries[key]; exists {
		c.stats.TotalSize -= int64(len(entry.Result))
		delete(c.entries, key)
		c.stats.EntryCount = len(c.entries)
	}
}

// InvalidateTemplate removes all cache entries for a template.
func (c *ResultCache) InvalidateTemplate(templateKey string) {
	prefix := c.config.KeyPrefix + templateKey + ":"

	c.mu.Lock()
	defer c.mu.Unlock()

	for key, entry := range c.entries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			c.stats.TotalSize -= int64(len(entry.Result))
			delete(c.entries, key)
		}
	}
	c.stats.EntryCount = len(c.entries)
}

// Clear removes all entries from the cache.
func (c *ResultCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*resultCacheEntry)
	c.evictList = make([]string, 0, c.config.MaxEntries)
	c.stats.TotalSize = 0
	c.stats.EntryCount = 0
}

// Stats returns current cache statistics.
func (c *ResultCache) Stats() ResultCacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// HitRate returns the cache hit rate (0.0 to 1.0).
func (c *ResultCache) HitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.stats.Hits + c.stats.Misses
	if total == 0 {
		return 0
	}
	return float64(c.stats.Hits) / float64(total)
}

// Cleanup removes expired entries. Call periodically for long-running applications.
func (c *ResultCache) Cleanup() int {
	now := time.Now()
	removed := 0

	c.mu.Lock()
	defer c.mu.Unlock()

	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			c.stats.TotalSize -= int64(len(entry.Result))
			delete(c.entries, key)
			removed++
		}
	}
	c.stats.EntryCount = len(c.entries)
	return removed
}

// makeKey creates a cache key from template key and data.
func (c *ResultCache) makeKey(templateKey string, data map[string]any) string {
	dataHash := c.hashData(data)
	return c.config.KeyPrefix + templateKey + ":" + dataHash
}

// hashData creates a hash of the input data for cache keying.
func (c *ResultCache) hashData(data map[string]any) string {
	if data == nil {
		return "nil"
	}

	// Serialize data to JSON for consistent hashing
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		// Fallback to empty hash on marshal error
		return "error"
	}

	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes (16 hex chars)
}

// evictOldest removes the oldest entry (simple FIFO for now).
func (c *ResultCache) evictOldest() {
	if len(c.evictList) == 0 {
		return
	}

	// Remove oldest entry
	oldestKey := c.evictList[0]
	c.evictList = c.evictList[1:]

	if entry, exists := c.entries[oldestKey]; exists {
		c.stats.TotalSize -= int64(len(entry.Result))
		delete(c.entries, oldestKey)
		c.stats.Evictions++
	}
}

// CachedEngine wraps an Engine with result caching.
type CachedEngine struct {
	engine *Engine
	cache  *ResultCache
}

// NewCachedEngine creates an engine wrapper with result caching.
func NewCachedEngine(engine *Engine, cacheConfig ResultCacheConfig) *CachedEngine {
	return &CachedEngine{
		engine: engine,
		cache:  NewResultCache(cacheConfig),
	}
}

// Execute executes a template with caching.
func (ce *CachedEngine) Execute(ctx context.Context, source string, data map[string]any) (string, error) {
	// Try cache first
	if result, ok := ce.cache.Get(source, data); ok {
		return result, nil
	}

	// Execute template
	result, err := ce.engine.Execute(ctx, source, data)
	if err != nil {
		return "", err
	}

	// Cache the result
	ce.cache.Set(source, data, result)
	return result, nil
}

// ExecuteTemplate executes a registered template with caching.
func (ce *CachedEngine) ExecuteTemplate(ctx context.Context, name string, data map[string]any) (string, error) {
	cacheKey := "named:" + name

	// Try cache first
	if result, ok := ce.cache.Get(cacheKey, data); ok {
		return result, nil
	}

	// Execute template
	result, err := ce.engine.ExecuteTemplate(ctx, name, data)
	if err != nil {
		return "", err
	}

	// Cache the result
	ce.cache.Set(cacheKey, data, result)
	return result, nil
}

// Parse delegates to the underlying engine (no caching for parsing).
func (ce *CachedEngine) Parse(source string) (*Template, error) {
	return ce.engine.Parse(source)
}

// InvalidateCache clears the result cache.
func (ce *CachedEngine) InvalidateCache() {
	ce.cache.Clear()
}

// InvalidateTemplate removes cached results for a specific template.
func (ce *CachedEngine) InvalidateTemplate(templateKey string) {
	ce.cache.InvalidateTemplate(templateKey)
}

// CacheStats returns the result cache statistics.
func (ce *CachedEngine) CacheStats() ResultCacheStats {
	return ce.cache.Stats()
}

// CacheHitRate returns the cache hit rate.
func (ce *CachedEngine) CacheHitRate() float64 {
	return ce.cache.HitRate()
}

// Engine returns the underlying engine for direct access.
func (ce *CachedEngine) Engine() *Engine {
	return ce.engine
}

// CachedStorageEngine wraps a StorageEngine with result caching.
type CachedStorageEngine struct {
	engine *StorageEngine
	cache  *ResultCache
}

// NewCachedStorageEngine creates a storage engine wrapper with result caching.
func NewCachedStorageEngine(engine *StorageEngine, cacheConfig ResultCacheConfig) *CachedStorageEngine {
	return &CachedStorageEngine{
		engine: engine,
		cache:  NewResultCache(cacheConfig),
	}
}

// Execute executes a stored template with caching.
func (cse *CachedStorageEngine) Execute(ctx context.Context, templateName string, data map[string]any) (string, error) {
	cacheKey := "stored:" + templateName

	// Try cache first
	if result, ok := cse.cache.Get(cacheKey, data); ok {
		return result, nil
	}

	// Execute template
	result, err := cse.engine.Execute(ctx, templateName, data)
	if err != nil {
		return "", err
	}

	// Cache the result
	cse.cache.Set(cacheKey, data, result)
	return result, nil
}

// ExecuteVersion executes a specific version with caching.
func (cse *CachedStorageEngine) ExecuteVersion(ctx context.Context, templateName string, version int, data map[string]any) (string, error) {
	cacheKey := "stored:" + templateName + ":v" + string(rune('0'+version%10))

	// Try cache first
	if result, ok := cse.cache.Get(cacheKey, data); ok {
		return result, nil
	}

	// Execute template
	result, err := cse.engine.ExecuteVersion(ctx, templateName, version, data)
	if err != nil {
		return "", err
	}

	// Cache the result
	cse.cache.Set(cacheKey, data, result)
	return result, nil
}

// InvalidateCache clears all cached results.
func (cse *CachedStorageEngine) InvalidateCache() {
	cse.cache.Clear()
}

// InvalidateTemplate removes cached results for a template.
func (cse *CachedStorageEngine) InvalidateTemplate(templateName string) {
	cse.cache.InvalidateTemplate("stored:" + templateName)
}

// CacheStats returns cache statistics.
func (cse *CachedStorageEngine) CacheStats() ResultCacheStats {
	return cse.cache.Stats()
}

// Engine returns the underlying storage engine.
func (cse *CachedStorageEngine) Engine() *StorageEngine {
	return cse.engine
}

// Save saves a template and invalidates its cache.
func (cse *CachedStorageEngine) Save(ctx context.Context, tmpl *StoredTemplate) error {
	err := cse.engine.Save(ctx, tmpl)
	if err == nil {
		// Invalidate cache for this template since it changed
		cse.InvalidateTemplate(tmpl.Name)
	}
	return err
}

// Delete deletes a template and invalidates its cache.
func (cse *CachedStorageEngine) Delete(ctx context.Context, name string) error {
	err := cse.engine.Delete(ctx, name)
	if err == nil {
		cse.InvalidateTemplate(name)
	}
	return err
}

// Get retrieves a template (delegated, no result caching).
func (cse *CachedStorageEngine) Get(ctx context.Context, name string) (*StoredTemplate, error) {
	return cse.engine.Get(ctx, name)
}

// List lists templates (delegated, no result caching).
func (cse *CachedStorageEngine) List(ctx context.Context, query *TemplateQuery) ([]*StoredTemplate, error) {
	return cse.engine.List(ctx, query)
}

// Close closes the underlying engine.
func (cse *CachedStorageEngine) Close() error {
	return cse.engine.Close()
}
