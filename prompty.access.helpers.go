package prompty

import (
	"context"
	"sync"
	"time"
)

// AllowAllChecker allows all access requests.
// Use for development, testing, or public endpoints.
type AllowAllChecker struct{}

// Check always returns an allow decision.
func (c *AllowAllChecker) Check(ctx context.Context, req *AccessRequest) (*AccessDecision, error) {
	return Allow("allow all checker"), nil
}

// BatchCheck allows all requests.
func (c *AllowAllChecker) BatchCheck(ctx context.Context, reqs []*AccessRequest) ([]*AccessDecision, error) {
	decisions := make([]*AccessDecision, len(reqs))
	for i := range reqs {
		decisions[i] = Allow("allow all checker")
	}
	return decisions, nil
}

// DenyAllChecker denies all access requests.
// Use for maintenance mode or locked-down endpoints.
type DenyAllChecker struct {
	// Reason is the denial reason to include in decisions.
	Reason string
}

// NewDenyAllChecker creates a deny-all checker with the given reason.
func NewDenyAllChecker(reason string) *DenyAllChecker {
	if reason == "" {
		reason = "deny all checker"
	}
	return &DenyAllChecker{Reason: reason}
}

// Check always returns a deny decision.
func (c *DenyAllChecker) Check(ctx context.Context, req *AccessRequest) (*AccessDecision, error) {
	return Deny(c.Reason), nil
}

// BatchCheck denies all requests.
func (c *DenyAllChecker) BatchCheck(ctx context.Context, reqs []*AccessRequest) ([]*AccessDecision, error) {
	decisions := make([]*AccessDecision, len(reqs))
	for i := range reqs {
		decisions[i] = Deny(c.Reason)
	}
	return decisions, nil
}

// ChainedChecker chains multiple checkers together.
// All checkers must allow for the request to be allowed (AND logic).
// The first denial stops the chain.
type ChainedChecker struct {
	checkers []AccessChecker
}

// NewChainedChecker creates a checker that chains multiple checkers.
// All checkers must allow for the request to be allowed.
func NewChainedChecker(checkers ...AccessChecker) (*ChainedChecker, error) {
	if len(checkers) == 0 {
		return nil, &AccessError{Message: ErrMsgNoCheckersInChain}
	}
	return &ChainedChecker{checkers: checkers}, nil
}

// MustChainedChecker creates a chained checker, panicking on error.
func MustChainedChecker(checkers ...AccessChecker) *ChainedChecker {
	c, err := NewChainedChecker(checkers...)
	if err != nil {
		panic(err)
	}
	return c
}

// Check evaluates all checkers in order.
// Returns the first denial or an allow if all pass.
func (c *ChainedChecker) Check(ctx context.Context, req *AccessRequest) (*AccessDecision, error) {
	for _, checker := range c.checkers {
		decision, err := checker.Check(ctx, req)
		if err != nil {
			return Deny(ErrMsgAccessCheckFailed + ": " + err.Error()), err
		}
		if !decision.Allowed {
			return decision, nil
		}
	}
	return Allow("all checkers passed"), nil
}

// BatchCheck evaluates all requests through the chain.
func (c *ChainedChecker) BatchCheck(ctx context.Context, reqs []*AccessRequest) ([]*AccessDecision, error) {
	decisions := make([]*AccessDecision, len(reqs))
	for i, req := range reqs {
		decision, err := c.Check(ctx, req)
		if err != nil {
			return nil, err
		}
		decisions[i] = decision
	}
	return decisions, nil
}

// AddChecker adds a checker to the chain.
func (c *ChainedChecker) AddChecker(checker AccessChecker) {
	c.checkers = append(c.checkers, checker)
}

// CachedChecker wraps a checker with caching for performance.
// Caches both allow and deny decisions with configurable TTL.
type CachedChecker struct {
	checker AccessChecker
	config  CachedCheckerConfig

	mu    sync.RWMutex
	cache map[string]*cachedDecision
}

// CachedCheckerConfig configures the cached checker.
type CachedCheckerConfig struct {
	// TTL is how long cached decisions remain valid.
	// Default: 5 minutes.
	TTL time.Duration

	// MaxEntries is the maximum number of cached decisions.
	// Default: 10000.
	MaxEntries int

	// KeyFunc generates cache keys from requests.
	// Default uses subject ID + operation + template name.
	KeyFunc func(*AccessRequest) string
}

// DefaultCachedCheckerConfig returns sensible defaults.
func DefaultCachedCheckerConfig() CachedCheckerConfig {
	return CachedCheckerConfig{
		TTL:        5 * time.Minute,
		MaxEntries: 10000,
		KeyFunc:    defaultCacheKey,
	}
}

type cachedDecision struct {
	decision  *AccessDecision
	cachedAt  time.Time
	expiresAt time.Time
}

// NewCachedChecker wraps a checker with caching.
func NewCachedChecker(checker AccessChecker, config CachedCheckerConfig) *CachedChecker {
	if config.TTL == 0 {
		config.TTL = 5 * time.Minute
	}
	if config.MaxEntries == 0 {
		config.MaxEntries = 10000
	}
	if config.KeyFunc == nil {
		config.KeyFunc = defaultCacheKey
	}

	return &CachedChecker{
		checker: checker,
		config:  config,
		cache:   make(map[string]*cachedDecision),
	}
}

func defaultCacheKey(req *AccessRequest) string {
	subjectID := ""
	if req.Subject != nil {
		subjectID = req.Subject.ID
	}
	return subjectID + ":" + string(req.Operation) + ":" + req.TemplateName
}

// Check evaluates the request, using cache when available.
func (c *CachedChecker) Check(ctx context.Context, req *AccessRequest) (*AccessDecision, error) {
	key := c.config.KeyFunc(req)

	// Check cache
	c.mu.RLock()
	cached, ok := c.cache[key]
	c.mu.RUnlock()

	if ok && time.Now().Before(cached.expiresAt) {
		return cached.decision, nil
	}

	// Cache miss - call underlying checker
	decision, err := c.checker.Check(ctx, req)
	if err != nil {
		return decision, err
	}

	// Cache the result
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict if at capacity
	if len(c.cache) >= c.config.MaxEntries {
		c.evictOldest()
	}

	now := time.Now()
	expiresAt := now.Add(c.config.TTL)

	// Use decision's expiry if sooner
	if decision.ExpiresAt != nil && decision.ExpiresAt.Before(expiresAt) {
		expiresAt = *decision.ExpiresAt
	}

	c.cache[key] = &cachedDecision{
		decision:  decision,
		cachedAt:  now,
		expiresAt: expiresAt,
	}

	return decision, nil
}

// BatchCheck evaluates multiple requests.
func (c *CachedChecker) BatchCheck(ctx context.Context, reqs []*AccessRequest) ([]*AccessDecision, error) {
	decisions := make([]*AccessDecision, len(reqs))
	for i, req := range reqs {
		decision, err := c.Check(ctx, req)
		if err != nil {
			return nil, err
		}
		decisions[i] = decision
	}
	return decisions, nil
}

// Invalidate removes a specific cache entry.
func (c *CachedChecker) Invalidate(req *AccessRequest) {
	key := c.config.KeyFunc(req)
	c.mu.Lock()
	delete(c.cache, key)
	c.mu.Unlock()
}

// InvalidateAll clears the entire cache.
func (c *CachedChecker) InvalidateAll() {
	c.mu.Lock()
	c.cache = make(map[string]*cachedDecision)
	c.mu.Unlock()
}

// Stats returns cache statistics.
func (c *CachedChecker) Stats() CachedCheckerStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	var validCount int
	for _, entry := range c.cache {
		if now.Before(entry.expiresAt) {
			validCount++
		}
	}

	return CachedCheckerStats{
		Entries:      len(c.cache),
		ValidEntries: validCount,
	}
}

// CachedCheckerStats contains cache statistics.
type CachedCheckerStats struct {
	Entries      int
	ValidEntries int
}

// evictOldest removes the oldest cache entry.
// Caller must hold write lock.
func (c *CachedChecker) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.cache {
		if oldestKey == "" || entry.cachedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.cachedAt
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
	}
}

// AnyOfChecker allows if ANY of the checkers allow (OR logic).
// Useful for combining multiple grant sources.
type AnyOfChecker struct {
	checkers []AccessChecker
}

// NewAnyOfChecker creates a checker that allows if any checker allows.
func NewAnyOfChecker(checkers ...AccessChecker) (*AnyOfChecker, error) {
	if len(checkers) == 0 {
		return nil, &AccessError{Message: ErrMsgNoCheckersInChain}
	}
	return &AnyOfChecker{checkers: checkers}, nil
}

// MustAnyOfChecker creates an any-of checker, panicking on error.
func MustAnyOfChecker(checkers ...AccessChecker) *AnyOfChecker {
	c, err := NewAnyOfChecker(checkers...)
	if err != nil {
		panic(err)
	}
	return c
}

// Check evaluates all checkers, allowing if any allows.
func (c *AnyOfChecker) Check(ctx context.Context, req *AccessRequest) (*AccessDecision, error) {
	var lastDenial *AccessDecision

	for _, checker := range c.checkers {
		decision, err := checker.Check(ctx, req)
		if err != nil {
			continue // Try next checker on error
		}
		if decision.Allowed {
			return decision, nil
		}
		lastDenial = decision
	}

	if lastDenial != nil {
		return lastDenial, nil
	}
	return Deny("no checker allowed access"), nil
}

// BatchCheck evaluates all requests.
func (c *AnyOfChecker) BatchCheck(ctx context.Context, reqs []*AccessRequest) ([]*AccessDecision, error) {
	decisions := make([]*AccessDecision, len(reqs))
	for i, req := range reqs {
		decision, err := c.Check(ctx, req)
		if err != nil {
			return nil, err
		}
		decisions[i] = decision
	}
	return decisions, nil
}

// OperationChecker allows specific operations.
// Useful for operation-based access control.
type OperationChecker struct {
	allowed map[Operation]bool
}

// NewOperationChecker creates a checker that allows specific operations.
func NewOperationChecker(ops ...Operation) *OperationChecker {
	allowed := make(map[Operation]bool)
	for _, op := range ops {
		allowed[op] = true
	}
	return &OperationChecker{allowed: allowed}
}

// Check allows if the operation is in the allowed set.
func (c *OperationChecker) Check(ctx context.Context, req *AccessRequest) (*AccessDecision, error) {
	if c.allowed[req.Operation] {
		return Allow("operation " + string(req.Operation) + " is allowed"), nil
	}
	return Deny("operation " + string(req.Operation) + " is not allowed"), nil
}

// BatchCheck evaluates all requests.
// Errors from individual Check calls result in Deny decisions (fail-safe).
func (c *OperationChecker) BatchCheck(ctx context.Context, reqs []*AccessRequest) ([]*AccessDecision, error) {
	decisions := make([]*AccessDecision, len(reqs))
	for i, req := range reqs {
		decision, err := c.Check(ctx, req)
		if err != nil {
			// Fail-safe: deny access on error
			decisions[i] = Deny(ErrMsgAccessCheckFailed)
		} else {
			decisions[i] = decision
		}
	}
	return decisions, nil
}

// TenantChecker enforces tenant isolation.
// Ensures subjects can only access templates in their tenant.
type TenantChecker struct {
	// AllowCrossTenant allows subjects to access templates from other tenants.
	// Default is false (strict isolation).
	AllowCrossTenant bool

	// SystemTenantID is a special tenant that can access all templates.
	// Empty means no system tenant.
	SystemTenantID string
}

// NewTenantChecker creates a tenant isolation checker.
func NewTenantChecker() *TenantChecker {
	return &TenantChecker{}
}

// WithSystemTenant sets the system tenant ID that can access all templates.
func (c *TenantChecker) WithSystemTenant(tenantID string) *TenantChecker {
	c.SystemTenantID = tenantID
	return c
}

// Check enforces tenant isolation.
func (c *TenantChecker) Check(ctx context.Context, req *AccessRequest) (*AccessDecision, error) {
	// Anonymous subjects denied
	if req.Subject == nil || req.Subject.TenantID == "" {
		return Deny("tenant ID required"), nil
	}

	// System tenant can access everything
	if c.SystemTenantID != "" && req.Subject.TenantID == c.SystemTenantID {
		return Allow("system tenant access"), nil
	}

	// If resource is loaded, check tenant match
	if req.Resource != nil && req.Resource.TenantID != "" {
		if req.Resource.TenantID != req.Subject.TenantID {
			if !c.AllowCrossTenant {
				return Deny("tenant mismatch"), nil
			}
		}
	}

	return Allow("tenant check passed"), nil
}

// BatchCheck evaluates all requests.
// Errors from individual Check calls result in Deny decisions (fail-safe).
func (c *TenantChecker) BatchCheck(ctx context.Context, reqs []*AccessRequest) ([]*AccessDecision, error) {
	decisions := make([]*AccessDecision, len(reqs))
	for i, req := range reqs {
		decision, err := c.Check(ctx, req)
		if err != nil {
			// Fail-safe: deny access on error
			decisions[i] = Deny(ErrMsgAccessCheckFailed)
		} else {
			decisions[i] = decision
		}
	}
	return decisions, nil
}

// RoleChecker allows access based on roles.
// Configurable required roles per operation.
type RoleChecker struct {
	// RolesPerOperation maps operations to required roles.
	// Subject must have at least one of the roles for the operation.
	RolesPerOperation map[Operation][]string

	// DefaultRoles are required if no operation-specific roles defined.
	DefaultRoles []string

	// RequireAllRoles requires subject to have ALL roles (default: any).
	RequireAllRoles bool
}

// NewRoleChecker creates a role-based access checker.
func NewRoleChecker() *RoleChecker {
	return &RoleChecker{
		RolesPerOperation: make(map[Operation][]string),
	}
}

// WithOperationRoles sets the roles required for an operation.
func (c *RoleChecker) WithOperationRoles(op Operation, roles ...string) *RoleChecker {
	c.RolesPerOperation[op] = roles
	return c
}

// WithDefaultRoles sets the default roles required.
func (c *RoleChecker) WithDefaultRoles(roles ...string) *RoleChecker {
	c.DefaultRoles = roles
	return c
}

// Check evaluates role requirements.
func (c *RoleChecker) Check(ctx context.Context, req *AccessRequest) (*AccessDecision, error) {
	if req.Subject == nil {
		return Deny("subject required for role check"), nil
	}

	// Get required roles for operation
	requiredRoles := c.RolesPerOperation[req.Operation]
	if len(requiredRoles) == 0 {
		requiredRoles = c.DefaultRoles
	}

	// No roles required means allow
	if len(requiredRoles) == 0 {
		return Allow("no roles required"), nil
	}

	// Check role membership
	if c.RequireAllRoles {
		if req.Subject.HasAllRoles(requiredRoles...) {
			return Allow("has all required roles"), nil
		}
		return Deny("missing required roles"), nil
	}

	if req.Subject.HasAnyRole(requiredRoles...) {
		return Allow("has required role"), nil
	}
	return Deny("missing required role"), nil
}

// BatchCheck evaluates all requests.
// Errors from individual Check calls result in Deny decisions (fail-safe).
func (c *RoleChecker) BatchCheck(ctx context.Context, reqs []*AccessRequest) ([]*AccessDecision, error) {
	decisions := make([]*AccessDecision, len(reqs))
	for i, req := range reqs {
		decision, err := c.Check(ctx, req)
		if err != nil {
			// Fail-safe: deny access on error
			decisions[i] = Deny(ErrMsgAccessCheckFailed)
		} else {
			decisions[i] = decision
		}
	}
	return decisions, nil
}
