package internal

import (
	"context"
	"fmt"
	"strconv"
)

// ContextAccessor is the interface for accessing context data.
// This mirrors prompty.Context to avoid import cycles.
type ContextAccessor interface {
	Get(path string) (any, bool)
	GetString(path string) string
	GetStringDefault(path, defaultVal string) string
	Has(path string) bool
}

// TemplateContextAccessor extends ContextAccessor with template execution capabilities.
// Resolvers that need to execute nested templates should check if their context
// implements this interface.
type TemplateContextAccessor interface {
	ContextAccessor
	// Engine returns the template executor for nested template resolution.
	// Returns nil if no engine is available.
	Engine() interface{}
	// Depth returns the current nesting depth for template includes.
	Depth() int
}

// VarResolver handles the prompty.var built-in tag.
// It retrieves variable values from the execution context.
type VarResolver struct{}

// NewVarResolver creates a new VarResolver.
func NewVarResolver() *VarResolver {
	return &VarResolver{}
}

// TagName returns the tag name for this resolver.
func (r *VarResolver) TagName() string {
	return TagNameVar
}

// Resolve retrieves the variable value from the context.
func (r *VarResolver) Resolve(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error) {
	accessor, ok := execCtx.(ContextAccessor)
	if !ok {
		return "", NewBuiltinError(ErrMsgInvalidContext, TagNameVar)
	}

	// Get required 'name' attribute
	name, ok := attrs.Get(AttrName)
	if !ok {
		return "", NewBuiltinError(ErrMsgMissingNameAttr, TagNameVar)
	}

	// Try to get the value
	val, found := accessor.Get(name)
	if !found {
		// Check for default attribute
		if defaultVal, hasDefault := attrs.Get(AttrDefault); hasDefault {
			return defaultVal, nil
		}
		return "", NewBuiltinError(fmt.Sprintf(ErrMsgVariableNotFoundFmt, name), TagNameVar)
	}

	// Convert value to string
	return valueToString(val), nil
}

// Validate checks that the required attributes are present.
func (r *VarResolver) Validate(attrs Attributes) error {
	if !attrs.Has(AttrName) {
		return NewBuiltinError(ErrMsgMissingNameAttr, TagNameVar)
	}
	return nil
}

// valueToString converts any value to its string representation.
func valueToString(val any) string {
	switch v := val.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case nil:
		return ""
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// RawResolver handles the prompty.raw built-in tag.
// This is a marker resolver - the executor handles raw blocks specially
// by preserving their content without parsing.
type RawResolver struct{}

// NewRawResolver creates a new RawResolver.
func NewRawResolver() *RawResolver {
	return &RawResolver{}
}

// TagName returns the tag name for this resolver.
func (r *RawResolver) TagName() string {
	return TagNameRaw
}

// Resolve returns an error because raw blocks should be handled by the executor.
// The raw block content is stored in TagNode.RawContent and should be
// returned directly by the executor without calling this resolver.
func (r *RawResolver) Resolve(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error) {
	// This should never be called - executor handles raw blocks specially
	return "", NewBuiltinError(ErrMsgRawResolverCalled, TagNameRaw)
}

// Validate always returns nil since raw blocks don't have required attributes.
func (r *RawResolver) Validate(attrs Attributes) error {
	return nil
}

// RegisterBuiltins registers all built-in resolvers with the registry.
func RegisterBuiltins(registry *Registry) {
	registry.MustRegister(NewVarResolver())
	registry.MustRegister(NewRawResolver())
	registry.MustRegister(NewIncludeResolver())
}

// BuiltinError represents an error from a built-in resolver.
type BuiltinError struct {
	Message string
	TagName string
}

// NewBuiltinError creates a new builtin error.
func NewBuiltinError(message, tagName string) *BuiltinError {
	return &BuiltinError{
		Message: message,
		TagName: tagName,
	}
}

// Error implements the error interface.
func (e *BuiltinError) Error() string {
	return fmt.Sprintf(ErrFmtTagMessage, e.TagName, e.Message)
}

// Builtin error message constants
const (
	ErrMsgInvalidContext      = "invalid execution context type"
	ErrMsgMissingNameAttr     = "missing required 'name' attribute"
	ErrMsgVariableNotFoundFmt = "variable not found: %s"
	ErrMsgRawResolverCalled   = "raw resolver should not be called directly"
)
