package prompty

import (
	"context"
	"os"
)

// Agent executor error message constants
const (
	ErrMsgAgentExecNilPrompt   = "prompt cannot be nil"
	ErrMsgAgentExecReadFile    = "failed to read agent file"
	ErrMsgAgentExecParseFailed = "failed to parse agent source"
)

// AgentExecutor is a high-level convenience wrapper that combines parsing,
// validation, and compilation of agent documents into a single workflow.
//
// It provides a simpler API for the common pattern of:
//
//	prompt, _ := prompty.Parse(source)
//	_ = prompt.ValidateAsAgent()
//	compiled, _ := prompt.CompileAgent(ctx, input, opts)
//
// Example:
//
//	executor := prompty.NewAgentExecutor(
//	    prompty.WithAgentResolver(myResolver),
//	    prompty.WithAgentSkillsCatalogFormat(prompty.CatalogFormatDetailed),
//	)
//	compiled, err := executor.Execute(ctx, agentYAML, input)
type AgentExecutor struct {
	resolver            DocumentResolver
	engine              *Engine
	skillsCatalogFormat CatalogFormat
	toolsCatalogFormat  CatalogFormat
}

// AgentExecutorOption is a functional option for configuring AgentExecutor.
type AgentExecutorOption func(*AgentExecutor)

// WithAgentResolver sets the DocumentResolver for agent compilation.
func WithAgentResolver(r DocumentResolver) AgentExecutorOption {
	return func(ae *AgentExecutor) {
		ae.resolver = r
	}
}

// WithAgentEngine sets a pre-configured engine for agent compilation.
func WithAgentEngine(e *Engine) AgentExecutorOption {
	return func(ae *AgentExecutor) {
		ae.engine = e
	}
}

// WithAgentSkillsCatalogFormat sets the skills catalog output format.
func WithAgentSkillsCatalogFormat(f CatalogFormat) AgentExecutorOption {
	return func(ae *AgentExecutor) {
		ae.skillsCatalogFormat = f
	}
}

// WithAgentToolsCatalogFormat sets the tools catalog output format.
func WithAgentToolsCatalogFormat(f CatalogFormat) AgentExecutorOption {
	return func(ae *AgentExecutor) {
		ae.toolsCatalogFormat = f
	}
}

// NewAgentExecutor creates a new AgentExecutor with the given options.
func NewAgentExecutor(options ...AgentExecutorOption) *AgentExecutor {
	ae := &AgentExecutor{}
	for _, opt := range options {
		opt(ae)
	}
	return ae
}

// Execute parses agent source, validates it as an agent, and compiles it.
// Returns the compiled prompt ready for LLM API submission.
func (ae *AgentExecutor) Execute(ctx context.Context, source string, input map[string]any) (*CompiledPrompt, error) {
	prompt, err := Parse([]byte(source))
	if err != nil {
		return nil, NewCompilationError(ErrMsgAgentExecParseFailed, err)
	}

	return ae.ExecutePrompt(ctx, prompt, input)
}

// ExecuteFile reads a file and compiles it as an agent.
func (ae *AgentExecutor) ExecuteFile(ctx context.Context, path string, input map[string]any) (*CompiledPrompt, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, NewCompilationError(ErrMsgAgentExecReadFile, err)
	}

	return ae.Execute(ctx, string(data), input)
}

// ExecutePrompt validates a pre-parsed prompt as an agent and compiles it.
func (ae *AgentExecutor) ExecutePrompt(ctx context.Context, prompt *Prompt, input map[string]any) (*CompiledPrompt, error) {
	if prompt == nil {
		return nil, NewCompilationError(ErrMsgAgentExecNilPrompt, nil)
	}

	if err := prompt.ValidateAsAgent(); err != nil {
		return nil, err
	}

	return prompt.CompileAgent(ctx, input, ae.compileOptions())
}

// ActivateSkill parses agent source, validates it, and activates a specific skill.
// If runtimeExec is provided, it is merged into the compiled execution config.
func (ae *AgentExecutor) ActivateSkill(ctx context.Context, source string, skillSlug string, input map[string]any, runtimeExec *ExecutionConfig) (*CompiledPrompt, error) {
	prompt, err := Parse([]byte(source))
	if err != nil {
		return nil, NewCompilationError(ErrMsgAgentExecParseFailed, err)
	}

	if err := prompt.ValidateAsAgent(); err != nil {
		return nil, err
	}

	compiled, err := prompt.ActivateSkill(ctx, skillSlug, input, ae.compileOptions())
	if err != nil {
		return nil, err
	}

	// Merge runtime execution config if provided
	if runtimeExec != nil && compiled.Execution != nil {
		compiled.Execution = compiled.Execution.Merge(runtimeExec)
	} else if runtimeExec != nil {
		compiled.Execution = runtimeExec.Clone()
	}

	return compiled, nil
}

// compileOptions builds CompileOptions from the executor's configuration.
func (ae *AgentExecutor) compileOptions() *CompileOptions {
	return &CompileOptions{
		Resolver:            ae.resolver,
		Engine:              ae.engine,
		SkillsCatalogFormat: ae.skillsCatalogFormat,
		ToolsCatalogFormat:  ae.toolsCatalogFormat,
	}
}
