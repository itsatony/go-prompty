package prompty

import (
	"context"
)

// Resolver is the interface that custom tag handlers must implement.
// Each resolver handles a specific tag type and produces output during template execution.
type Resolver interface {
	// TagName returns the tag name this resolver handles (e.g., "prompty.var", "MyCustomTag")
	TagName() string

	// Resolve executes the tag and returns the output string.
	// ctx is the Go context for cancellation and deadlines.
	// execCtx provides access to template variables and execution state.
	// attrs contains the tag's attributes.
	Resolve(ctx context.Context, execCtx *Context, attrs Attributes) (string, error)

	// Validate checks if the provided attributes are valid for this resolver.
	// Called during parsing to catch errors early.
	// Return nil if attributes are valid, or an error describing the issue.
	Validate(attrs Attributes) error
}

// PromptResolver provides prompt lookup for reference resolution.
// Implement this interface to enable {~prompty.ref~} tag functionality.
type PromptResolver interface {
	// ResolvePrompt looks up a prompt by slug and version.
	// If version is empty or "latest", the most recent version should be returned.
	// Returns the prompt and its template body, or an error if not found.
	ResolvePrompt(ctx context.Context, slug string, version string) (*Prompt, string, error)
}

// Attributes provides read-only access to tag attributes.
// All attribute values are strings; resolvers must convert as needed.
type Attributes interface {
	// Get retrieves an attribute value.
	// Returns the value and true if found, or empty string and false if not.
	Get(key string) (string, bool)

	// GetDefault retrieves an attribute value with a fallback.
	// Returns the attribute value if it exists, or defaultVal if not.
	GetDefault(key, defaultVal string) string

	// Has checks if an attribute exists.
	Has(key string) bool

	// Keys returns all attribute keys in sorted order.
	Keys() []string

	// Map returns a copy of all attributes as a map.
	Map() map[string]string
}

// ResolverFunc is a convenience type for creating simple resolvers from functions.
// It implements Resolver with a configurable tag name and no validation.
type ResolverFunc struct {
	name     string
	fn       func(ctx context.Context, execCtx *Context, attrs Attributes) (string, error)
	validate func(attrs Attributes) error
}

// NewResolverFunc creates a new function-based resolver.
// If validate is nil, Validate() will always return nil.
func NewResolverFunc(
	name string,
	fn func(ctx context.Context, execCtx *Context, attrs Attributes) (string, error),
	validate func(attrs Attributes) error,
) *ResolverFunc {
	return &ResolverFunc{
		name:     name,
		fn:       fn,
		validate: validate,
	}
}

// TagName returns the resolver's tag name.
func (r *ResolverFunc) TagName() string {
	return r.name
}

// Resolve executes the resolver function.
func (r *ResolverFunc) Resolve(ctx context.Context, execCtx *Context, attrs Attributes) (string, error) {
	return r.fn(ctx, execCtx, attrs)
}

// Validate runs the validation function if provided.
func (r *ResolverFunc) Validate(attrs Attributes) error {
	if r.validate != nil {
		return r.validate(attrs)
	}
	return nil
}

// PromptResolverAdapter wraps a PromptResolver to implement PromptBodyResolver.
// This adapter extracts only the template body from the full PromptResolver response.
type PromptResolverAdapter struct {
	resolver PromptResolver
}

// NewPromptResolverAdapter creates an adapter that wraps a PromptResolver
// to implement the PromptBodyResolver interface used internally.
func NewPromptResolverAdapter(resolver PromptResolver) *PromptResolverAdapter {
	return &PromptResolverAdapter{resolver: resolver}
}

// ResolvePromptBody looks up a prompt by slug and version and returns its template body.
// This implements the PromptBodyResolver interface.
func (a *PromptResolverAdapter) ResolvePromptBody(ctx context.Context, slug string, version string) (string, error) {
	_, body, err := a.resolver.ResolvePrompt(ctx, slug, version)
	return body, err
}
