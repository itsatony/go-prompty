package prompty

import (
	"context"

	"github.com/itsatony/go-prompty/internal"
)

// Template represents a parsed template that can be executed multiple times.
type Template struct {
	source   string
	ast      *internal.RootNode
	executor *internal.Executor
	config   *engineConfig
}

// newTemplate creates a new template (internal use).
func newTemplate(source string, ast *internal.RootNode, executor *internal.Executor, config *engineConfig) *Template {
	return &Template{
		source:   source,
		ast:      ast,
		executor: executor,
		config:   config,
	}
}

// Execute renders the template with the given data.
// This is a convenience method that creates a Context from the data map.
func (t *Template) Execute(ctx context.Context, data map[string]any) (string, error) {
	execCtx := NewContextWithStrategy(data, t.config.errorStrategy)
	return t.ExecuteWithContext(ctx, execCtx)
}

// ExecuteWithContext renders the template with the given execution context.
// Use this when you need more control over the context (e.g., parent scoping).
func (t *Template) ExecuteWithContext(ctx context.Context, execCtx *Context) (string, error) {
	return t.executor.Execute(ctx, t.ast, execCtx)
}

// Source returns the original template source string.
func (t *Template) Source() string {
	return t.source
}

// internalAttributesAdapter wraps internal.Attributes to implement Attributes interface
type internalAttributesAdapter struct {
	attrs internal.Attributes
}

func (a *internalAttributesAdapter) Get(key string) (string, bool) {
	return a.attrs.Get(key)
}

func (a *internalAttributesAdapter) GetDefault(key, defaultVal string) string {
	return a.attrs.GetDefault(key, defaultVal)
}

func (a *internalAttributesAdapter) Has(key string) bool {
	return a.attrs.Has(key)
}

func (a *internalAttributesAdapter) Keys() []string {
	return a.attrs.Keys()
}

func (a *internalAttributesAdapter) Map() map[string]string {
	return a.attrs.Map()
}
