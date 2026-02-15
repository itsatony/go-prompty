package prompty

import (
	"context"

	"github.com/itsatony/go-prompty/v2/internal"
)

// Template represents a parsed template that can be executed multiple times.
type Template struct {
	source          string
	templateBody    string // Template body without config block
	ast             *internal.RootNode
	executor        *internal.Executor
	config          *engineConfig
	engine          TemplateExecutor          // Engine reference for nested template execution
	prompt          *Prompt                   // Parsed prompt configuration from frontmatter
	inheritanceInfo *internal.InheritanceInfo // Inheritance info (nil if no extends)
}

// newTemplateWithConfig creates a new template with prompt configuration (internal use).
func newTemplateWithConfig(source, templateBody string, ast *internal.RootNode, executor *internal.Executor, config *engineConfig, engine TemplateExecutor, prompt *Prompt) *Template {
	// Extract inheritance info from AST
	inheritanceInfo, _ := internal.ExtractInheritanceInfo(ast)

	return &Template{
		source:          source,
		templateBody:    templateBody,
		ast:             ast,
		executor:        executor,
		config:          config,
		engine:          engine,
		prompt:          prompt,
		inheritanceInfo: inheritanceInfo,
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
// The engine reference is injected into the context for nested template support.
// If the template uses extends (template inheritance), inheritance is resolved before execution.
func (t *Template) ExecuteWithContext(ctx context.Context, execCtx *Context) (string, error) {
	// Inject engine reference into context for nested template resolution
	if t.engine != nil && execCtx.Engine() == nil {
		execCtx = execCtx.WithEngine(t.engine)
	}

	// Resolve inheritance if the template extends another template
	astToExecute := t.ast
	if t.inheritanceInfo != nil && t.engine != nil {
		// Create an adapter that wraps the engine for TemplateSourceResolver interface
		sourceResolver := &engineSourceAdapter{engine: t.engine}
		resolver := internal.NewInheritanceResolver(nil, sourceResolver, t.config.maxDepth)
		resolvedAST, err := resolver.ResolveInheritance(ctx, t.ast, t.inheritanceInfo, 0)
		if err != nil {
			return "", err
		}
		astToExecute = resolvedAST
	}

	return t.executor.Execute(ctx, astToExecute, execCtx)
}

// engineSourceAdapter adapts TemplateExecutor to TemplateSourceResolver
type engineSourceAdapter struct {
	engine TemplateExecutor
}

func (a *engineSourceAdapter) GetTemplateSource(name string) (string, bool) {
	return a.engine.GetTemplateSource(name)
}

// Source returns the original template source string (including config block if present).
func (t *Template) Source() string {
	return t.source
}

// TemplateBody returns the template body without the config block.
// This is the portion of the template that is actually executed.
func (t *Template) TemplateBody() string {
	return t.templateBody
}

// Prompt returns the v2.0 prompt configuration from the frontmatter.
// Returns nil if the template has no frontmatter or if it's a v1 template.
func (t *Template) Prompt() *Prompt {
	return t.prompt
}

// HasPrompt returns true if the template has a v2.0 prompt configuration.
func (t *Template) HasPrompt() bool {
	return t.prompt != nil
}

// Compile compiles the template's prompt by executing its body through an engine.
// Returns an error if the template has no prompt configuration.
func (t *Template) Compile(ctx context.Context, input map[string]any, opts *CompileOptions) (string, error) {
	if t.prompt == nil {
		return "", NewCompilationError(ErrMsgCompilationFailed, nil)
	}
	return t.prompt.Compile(ctx, input, opts)
}

// CompileAgent compiles the template's agent prompt into a CompiledPrompt.
// Returns an error if the template has no prompt configuration or is not an agent.
func (t *Template) CompileAgent(ctx context.Context, input map[string]any, opts *CompileOptions) (*CompiledPrompt, error) {
	if t.prompt == nil {
		return nil, NewCompilationError(ErrMsgCompilationFailed, nil)
	}
	return t.prompt.CompileAgent(ctx, input, opts)
}

// ExecuteAndExtractMessages executes the template and extracts structured messages from the output.
// This is useful for chat/conversation templates that use {~prompty.message~} tags.
// Returns the messages array and any error from execution.
func (t *Template) ExecuteAndExtractMessages(ctx context.Context, data map[string]any) ([]Message, error) {
	output, err := t.Execute(ctx, data)
	if err != nil {
		return nil, err
	}
	return ExtractMessagesFromOutput(output), nil
}

// ExtractMessagesFromOutput parses executed template output and extracts structured messages.
// Messages are marked by special markers inserted by the prompty.message tag resolver.
// This is a standalone function for when you already have the executed output.
func ExtractMessagesFromOutput(output string) []Message {
	internalMessages := internal.ExtractMessages(output)
	if internalMessages == nil {
		return nil
	}

	messages := make([]Message, len(internalMessages))
	for i, m := range internalMessages {
		messages[i] = Message{
			Role:    m.Role,
			Content: m.Content,
			Cache:   m.Cache,
		}
	}
	return messages
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
