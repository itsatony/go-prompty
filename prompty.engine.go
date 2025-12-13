package prompty

import (
	"context"

	"github.com/itsatony/go-prompty/internal"
	"go.uber.org/zap"
)

// Engine is the main entry point for the prompty templating system.
// It manages parsing, execution, and resolver registration.
type Engine struct {
	registry *internal.Registry
	config   *engineConfig
	executor *internal.Executor
	logger   *zap.Logger
}

// New creates a new prompty Engine with the given options.
func New(opts ...Option) (*Engine, error) {
	config := defaultEngineConfig()
	for _, opt := range opts {
		opt(config)
	}

	logger := config.logger
	if logger == nil {
		logger = zap.NewNop()
	}

	registry := internal.NewRegistry(logger)
	internal.RegisterBuiltins(registry)

	executorConfig := internal.ExecutorConfig{
		MaxDepth: config.maxDepth,
	}
	executor := internal.NewExecutor(registry, executorConfig, logger)

	return &Engine{
		registry: registry,
		config:   config,
		executor: executor,
		logger:   logger,
	}, nil
}

// MustNew creates a new Engine and panics if there's an error.
func MustNew(opts ...Option) *Engine {
	engine, err := New(opts...)
	if err != nil {
		panic(err)
	}
	return engine
}

// Parse parses a template source string and returns a Template.
// The returned Template can be executed multiple times with different data.
func (e *Engine) Parse(source string) (*Template, error) {
	// Create lexer with configured delimiters
	lexerConfig := internal.LexerConfig{
		OpenDelim:  e.config.openDelim,
		CloseDelim: e.config.closeDelim,
	}
	lexer := internal.NewLexerWithConfig(source, lexerConfig, e.logger)

	// Tokenize
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, NewParseError(ErrMsgParseFailed, Position{}, err)
	}

	// Parse
	parser := internal.NewParser(tokens, e.logger)
	ast, err := parser.Parse()
	if err != nil {
		return nil, NewParseError(ErrMsgParseFailed, Position{}, err)
	}

	return newTemplate(source, ast, e.executor, e.config), nil
}

// Execute is a convenience method that parses and executes in one step.
// For templates that will be executed multiple times, use Parse() instead.
func (e *Engine) Execute(ctx context.Context, source string, data map[string]any) (string, error) {
	tmpl, err := e.Parse(source)
	if err != nil {
		return "", err
	}
	return tmpl.Execute(ctx, data)
}

// Register adds a custom resolver to the engine.
// Returns an error if a resolver for the same tag name is already registered.
func (e *Engine) Register(r Resolver) error {
	adapter := &resolverAdapter{resolver: r}
	return e.registry.Register(adapter)
}

// MustRegister adds a custom resolver and panics if registration fails.
func (e *Engine) MustRegister(r Resolver) {
	if err := e.Register(r); err != nil {
		panic(err)
	}
}

// resolverAdapter adapts the public Resolver interface to internal.InternalResolver
type resolverAdapter struct {
	resolver Resolver
}

func (a *resolverAdapter) TagName() string {
	return a.resolver.TagName()
}

func (a *resolverAdapter) Resolve(ctx context.Context, execCtx interface{}, attrs internal.Attributes) (string, error) {
	// Convert execCtx to *Context
	promptyCtx, ok := execCtx.(*Context)
	if !ok {
		return "", NewExecutionError(ErrMsgInvalidContextType, a.TagName(), Position{}, nil)
	}

	// Wrap internal.Attributes to satisfy public Attributes interface
	wrappedAttrs := &internalAttributesAdapter{attrs: attrs}
	return a.resolver.Resolve(ctx, promptyCtx, wrappedAttrs)
}

func (a *resolverAdapter) Validate(attrs internal.Attributes) error {
	wrappedAttrs := &internalAttributesAdapter{attrs: attrs}
	return a.resolver.Validate(wrappedAttrs)
}

// Additional error messages
const (
	ErrMsgInvalidContextType = "invalid context type"
)
