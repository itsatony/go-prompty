package prompty

import (
	"context"
	"time"
)

// Operation represents an access control operation type.
type Operation string

// Operation constants for access control.
const (
	OpRead    Operation = "read"
	OpExecute Operation = "execute"
	OpCreate  Operation = "create"
	OpUpdate  Operation = "update"
	OpDelete  Operation = "delete"
	OpList    Operation = "list"
)

// AccessChecker is the interface for access control implementations.
// Users implement this to integrate their authentication/authorization systems.
//
// The interface is intentionally minimal - Check handles individual requests,
// BatchCheck handles bulk operations efficiently.
type AccessChecker interface {
	// Check evaluates whether the request should be allowed.
	// Returns an AccessDecision with the result and reasoning.
	// Implementations should fail closed (deny on error).
	Check(ctx context.Context, req *AccessRequest) (*AccessDecision, error)

	// BatchCheck evaluates multiple requests efficiently.
	// Default implementation calls Check for each request.
	// Override for bulk optimization (e.g., single database query).
	BatchCheck(ctx context.Context, reqs []*AccessRequest) ([]*AccessDecision, error)
}

// AccessRequest carries all context needed for an access decision.
// The Subject identifies who is requesting access, and the operation
// specifies what they want to do.
type AccessRequest struct {
	// Operation is the type of access being requested.
	Operation Operation

	// TemplateName is the name of the template being accessed.
	TemplateName string

	// TemplateID is the unique identifier of the template (if known).
	TemplateID TemplateID

	// Subject identifies who is requesting access.
	Subject *AccessSubject

	// Resource is the template being accessed (if already loaded).
	// May be nil for existence checks or before loading.
	Resource *StoredTemplate

	// Metadata contains additional context for the access decision.
	// Use for custom attributes specific to your access control model.
	Metadata map[string]any

	// ExecutionData contains the data being passed to template execution.
	// Only populated for OpExecute operations.
	ExecutionData map[string]any
}

// AccessSubject identifies the entity requesting access.
// The struct is intentionally broad to support various auth models:
// RBAC (roles), ABAC (attrs), OAuth2 (scopes), custom (extra).
type AccessSubject struct {
	// ID is the unique identifier for this subject (user ID, service ID, etc.).
	ID string

	// Type categorizes the subject (e.g., "user", "service", "system", "anonymous").
	Type string

	// TenantID is the multi-tenant organization identifier.
	// Use for tenant isolation in multi-tenant systems.
	TenantID string

	// Roles contains RBAC role assignments (e.g., "admin", "editor", "viewer").
	Roles []string

	// Groups contains group memberships for group-based access control.
	Groups []string

	// Scopes contains OAuth2-style permission scopes (e.g., "templates:read").
	Scopes []string

	// Attrs contains ABAC attributes for attribute-based access control.
	// Keys and values are strings for simple serialization.
	Attrs map[string]string

	// Extra contains any additional data needed by custom access checkers.
	Extra map[string]any
}

// AccessDecision is the result of an access check.
type AccessDecision struct {
	// Allowed indicates whether access is granted.
	Allowed bool

	// Reason provides a human-readable explanation for the decision.
	// Useful for debugging and audit logs.
	Reason string

	// Conditions lists any conditions attached to the grant.
	// For conditional access (e.g., "allowed during business hours").
	Conditions []string

	// ExpiresAt indicates when this decision expires (for caching).
	// Nil means the decision doesn't expire.
	ExpiresAt *time.Time
}

// NewAccessRequest creates a new access request with the given parameters.
func NewAccessRequest(op Operation, templateName string, subject *AccessSubject) *AccessRequest {
	return &AccessRequest{
		Operation:    op,
		TemplateName: templateName,
		Subject:      subject,
		Metadata:     make(map[string]any),
	}
}

// WithTemplateID sets the template ID on the request.
func (r *AccessRequest) WithTemplateID(id TemplateID) *AccessRequest {
	r.TemplateID = id
	return r
}

// WithResource sets the resource (loaded template) on the request.
func (r *AccessRequest) WithResource(tmpl *StoredTemplate) *AccessRequest {
	r.Resource = tmpl
	if tmpl != nil {
		r.TemplateID = tmpl.ID
		r.TemplateName = tmpl.Name
	}
	return r
}

// WithExecutionData sets the execution data on the request.
func (r *AccessRequest) WithExecutionData(data map[string]any) *AccessRequest {
	r.ExecutionData = data
	return r
}

// WithMetadata adds metadata to the request.
func (r *AccessRequest) WithMetadata(key string, value any) *AccessRequest {
	if r.Metadata == nil {
		r.Metadata = make(map[string]any)
	}
	r.Metadata[key] = value
	return r
}

// NewAccessSubject creates a new access subject with the given ID.
func NewAccessSubject(id string) *AccessSubject {
	return &AccessSubject{
		ID:    id,
		Type:  SubjectTypeUser,
		Attrs: make(map[string]string),
		Extra: make(map[string]any),
	}
}

// Subject type constants.
const (
	SubjectTypeUser      = "user"
	SubjectTypeService   = "service"
	SubjectTypeSystem    = "system"
	SubjectTypeAnonymous = "anonymous"
)

// WithType sets the subject type.
func (s *AccessSubject) WithType(t string) *AccessSubject {
	s.Type = t
	return s
}

// WithTenant sets the tenant ID.
func (s *AccessSubject) WithTenant(tenantID string) *AccessSubject {
	s.TenantID = tenantID
	return s
}

// WithRoles sets the subject's roles.
func (s *AccessSubject) WithRoles(roles ...string) *AccessSubject {
	s.Roles = roles
	return s
}

// WithGroups sets the subject's groups.
func (s *AccessSubject) WithGroups(groups ...string) *AccessSubject {
	s.Groups = groups
	return s
}

// WithScopes sets the subject's OAuth2 scopes.
func (s *AccessSubject) WithScopes(scopes ...string) *AccessSubject {
	s.Scopes = scopes
	return s
}

// WithAttr adds an ABAC attribute.
func (s *AccessSubject) WithAttr(key, value string) *AccessSubject {
	if s.Attrs == nil {
		s.Attrs = make(map[string]string)
	}
	s.Attrs[key] = value
	return s
}

// WithExtra adds extra data to the subject.
func (s *AccessSubject) WithExtra(key string, value any) *AccessSubject {
	if s.Extra == nil {
		s.Extra = make(map[string]any)
	}
	s.Extra[key] = value
	return s
}

// HasRole checks if the subject has the specified role.
func (s *AccessSubject) HasRole(role string) bool {
	for _, r := range s.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the subject has any of the specified roles.
func (s *AccessSubject) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if s.HasRole(role) {
			return true
		}
	}
	return false
}

// HasAllRoles checks if the subject has all of the specified roles.
func (s *AccessSubject) HasAllRoles(roles ...string) bool {
	for _, role := range roles {
		if !s.HasRole(role) {
			return false
		}
	}
	return true
}

// HasScope checks if the subject has the specified scope.
func (s *AccessSubject) HasScope(scope string) bool {
	for _, sc := range s.Scopes {
		if sc == scope {
			return true
		}
	}
	return false
}

// HasGroup checks if the subject is a member of the specified group.
func (s *AccessSubject) HasGroup(group string) bool {
	for _, g := range s.Groups {
		if g == group {
			return true
		}
	}
	return false
}

// IsAnonymous checks if the subject is anonymous.
func (s *AccessSubject) IsAnonymous() bool {
	return s == nil || s.ID == "" || s.Type == SubjectTypeAnonymous
}

// Allow creates an AccessDecision that grants access.
func Allow(reason string) *AccessDecision {
	return &AccessDecision{
		Allowed: true,
		Reason:  reason,
	}
}

// Deny creates an AccessDecision that denies access.
func Deny(reason string) *AccessDecision {
	return &AccessDecision{
		Allowed: false,
		Reason:  reason,
	}
}

// AllowWithConditions creates an AccessDecision that grants access with conditions.
func AllowWithConditions(reason string, conditions ...string) *AccessDecision {
	return &AccessDecision{
		Allowed:    true,
		Reason:     reason,
		Conditions: conditions,
	}
}

// AllowWithExpiry creates an AccessDecision that grants access with an expiration.
func AllowWithExpiry(reason string, expiresAt time.Time) *AccessDecision {
	return &AccessDecision{
		Allowed:   true,
		Reason:    reason,
		ExpiresAt: &expiresAt,
	}
}

// Access control error messages.
const (
	ErrMsgAccessDenied      = "access denied"
	ErrMsgAccessCheckFailed = "access check failed"
	ErrMsgNilSubject        = "subject is nil"
	ErrMsgNilChecker        = "access checker is nil"
	ErrMsgNoCheckersInChain = "no checkers in chain"
)

// AccessError represents an access control error.
type AccessError struct {
	Message   string
	Operation Operation
	Subject   *AccessSubject
	Template  string
	Cause     error
}

// Error implements the error interface.
func (e *AccessError) Error() string {
	msg := e.Message
	if e.Template != "" {
		msg += " (template: " + e.Template + ")"
	}
	if e.Operation != "" {
		msg += " (operation: " + string(e.Operation) + ")"
	}
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}

// Unwrap returns the underlying error.
func (e *AccessError) Unwrap() error {
	return e.Cause
}

// NewAccessDeniedError creates an access denied error.
func NewAccessDeniedError(op Operation, template string, subject *AccessSubject) *AccessError {
	return &AccessError{
		Message:   ErrMsgAccessDenied,
		Operation: op,
		Subject:   subject,
		Template:  template,
	}
}

// NewAccessCheckError creates an error for failed access checks.
func NewAccessCheckError(op Operation, template string, cause error) *AccessError {
	return &AccessError{
		Message:   ErrMsgAccessCheckFailed,
		Operation: op,
		Template:  template,
		Cause:     cause,
	}
}
