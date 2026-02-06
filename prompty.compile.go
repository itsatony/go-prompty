package prompty

import (
	"context"
	"strings"
)

// Compilation error messages
const (
	ErrMsgCompileNotAgent       = "cannot compile non-agent document as agent"
	ErrMsgCompileBodyFailed     = "failed to compile body template"
	ErrMsgCompileMessageFailed  = "failed to compile message template"
	ErrMsgCompileSkillFailed    = "failed to compile skill for activation"
	ErrMsgCompileNoEngine       = "engine required for compilation"
	ErrMsgActivateSkillNotFound = "skill not found in agent for activation"
)

// CompileOptions configures agent compilation.
type CompileOptions struct {
	// Resolver resolves skill/prompt/agent references.
	Resolver DocumentResolver
	// SkillsCatalogFormat overrides the default skills catalog format.
	SkillsCatalogFormat CatalogFormat
	// ToolsCatalogFormat overrides the default tools catalog format.
	ToolsCatalogFormat CatalogFormat
}

// CompiledPrompt is the result of agent compilation.
type CompiledPrompt struct {
	// Messages contains the compiled messages ready for LLM API submission.
	Messages []CompiledMessage
	// Execution is the effective execution config (merged from agent + runtime).
	Execution *ExecutionConfig
	// Tools is the tools configuration for function calling.
	Tools *ToolsConfig
	// Constraints are the operational constraints for the agent.
	Constraints *OperationalConstraints
}

// CompiledMessage is a single message in the compiled output.
type CompiledMessage struct {
	// Role of the message: "system", "user", "assistant", "tool"
	Role string
	// Content is the fully resolved message content.
	Content string
	// Cache indicates whether this message should be cached.
	Cache bool
}

// Compile compiles a prompt or skill document by executing its body through the engine.
// For simple prompts/skills, this resolves all {~...~} tags in the body.
func (p *Prompt) Compile(ctx context.Context, input map[string]any, opts *CompileOptions) (string, error) {
	if p == nil {
		return "", NewCompilationError(ErrMsgCompilationFailed, nil)
	}

	// Build context data
	data := buildCompileContext(p, input)

	// Create engine and execute body
	engine := MustNew()
	result, err := engine.Execute(ctx, p.Body, data)
	if err != nil {
		return "", NewCompilationError(ErrMsgCompileBodyFailed, err)
	}

	return result, nil
}

// CompileAgent compiles an agent document into a CompiledPrompt.
// This resolves all templates, generates catalogs, and processes message templates.
func (p *Prompt) CompileAgent(ctx context.Context, input map[string]any, opts *CompileOptions) (*CompiledPrompt, error) {
	if p == nil {
		return nil, NewCompilationError(ErrMsgCompilationFailed, nil)
	}

	if !p.IsAgent() {
		return nil, NewCompilationError(ErrMsgCompileNotAgent, nil)
	}

	if opts == nil {
		opts = &CompileOptions{}
	}

	// Build context data
	data := buildCompileContext(p, input)

	// Generate catalogs and inject into context
	skillsCatalog, err := GenerateSkillsCatalog(ctx, p.Skills, opts.Resolver, opts.SkillsCatalogFormat)
	if err != nil {
		// Non-fatal: empty catalog on error
		skillsCatalog = ""
	}
	data[ContextKeySkills] = skillsCatalog

	toolsCatalog, err := GenerateToolsCatalog(p.Tools, opts.ToolsCatalogFormat)
	if err != nil {
		toolsCatalog = ""
	}
	data[ContextKeyTools] = toolsCatalog

	// Store body content for self-reference
	data[ContextKeySelfBody] = p.Body

	// Create engine for compilation
	engine := MustNew()

	// Register "self" template with the body content
	if p.Body != "" {
		_ = engine.RegisterTemplate(TemplateNameSelf, p.Body)
	}

	// Compile body
	compiledBody, err := engine.Execute(ctx, p.Body, data)
	if err != nil {
		return nil, NewCompilationError(ErrMsgCompileBodyFailed, err)
	}

	// Process messages
	var messages []CompiledMessage
	if len(p.Messages) > 0 {
		messages, err = compileMessages(ctx, engine, p.Messages, data, compiledBody)
		if err != nil {
			return nil, err
		}
	} else {
		// Default messages: system (compiled body) + user (input message if present)
		messages = buildDefaultMessages(compiledBody, input)
	}

	// Build result
	result := &CompiledPrompt{
		Messages: messages,
	}

	if p.Execution != nil {
		result.Execution = p.Execution.Clone()
	}

	if p.Tools != nil {
		result.Tools = p.Tools.Clone()
	}

	if p.Constraints != nil && p.Constraints.Operational != nil {
		result.Constraints = p.Constraints.Operational.Clone()
	}

	return result, nil
}

// ActivateSkill compiles the agent and then activates a specific skill.
// The skill body is resolved, executed, and injected into the messages per injection mode.
func (p *Prompt) ActivateSkill(ctx context.Context, skillSlug string, input map[string]any, opts *CompileOptions) (*CompiledPrompt, error) {
	// First, compile the base agent
	compiled, err := p.CompileAgent(ctx, input, opts)
	if err != nil {
		return nil, err
	}

	// Find the skill ref by slug
	var skillRef *SkillRef
	for i := range p.Skills {
		if p.Skills[i].GetSlug() == skillSlug {
			skillRef = &p.Skills[i]
			break
		}
	}
	if skillRef == nil {
		return nil, NewSkillNotFoundError(skillSlug)
	}

	// Resolve skill body
	var skillBody string
	var skillExec *ExecutionConfig

	if skillRef.IsInline() {
		skillBody = skillRef.Inline.Body
	} else if opts != nil && opts.Resolver != nil {
		resolved, err := opts.Resolver.ResolveSkill(ctx, skillRef.Slug)
		if err != nil {
			return nil, NewCompilationError(ErrMsgCompileSkillFailed, err)
		}
		skillBody = resolved.Body
		if resolved.Execution != nil {
			skillExec = resolved.Execution
		}
	} else {
		return nil, NewSkillNotFoundError(skillSlug)
	}

	// Compile skill body through engine
	data := buildCompileContext(p, input)
	engine := MustNew()
	compiledSkillBody, err := engine.Execute(ctx, skillBody, data)
	if err != nil {
		return nil, NewCompilationError(ErrMsgCompileSkillFailed, err)
	}

	// Inject into messages per injection mode
	injection := skillRef.Injection
	if injection == "" {
		injection = SkillInjectionSystemPrompt
	}

	switch injection {
	case SkillInjectionSystemPrompt:
		injectSkillIntoSystemPrompt(compiled, skillSlug, compiledSkillBody)
	case SkillInjectionUserContext:
		injectSkillIntoUserContext(compiled, skillSlug, compiledSkillBody)
	case SkillInjectionNone:
		// No injection
	}

	// Merge execution configs: agent → skill resolved → skill ref override → runtime
	if skillExec != nil && compiled.Execution != nil {
		compiled.Execution = compiled.Execution.Merge(skillExec)
	} else if skillExec != nil {
		compiled.Execution = skillExec.Clone()
	}
	if skillRef.Execution != nil && compiled.Execution != nil {
		compiled.Execution = compiled.Execution.Merge(skillRef.Execution)
	} else if skillRef.Execution != nil {
		compiled.Execution = skillRef.Execution.Clone()
	}

	return compiled, nil
}

// ValidateForExecution checks that the prompt has sufficient configuration for execution.
func (p *Prompt) ValidateForExecution() error {
	if p == nil {
		return NewCompilationError(ErrMsgCompilationFailed, nil)
	}
	if p.Execution == nil {
		return NewCompilationError(ErrMsgNoExecutionConfig, nil)
	}
	if p.Execution.Provider == "" {
		return NewCompilationError(ErrMsgNoProvider, nil)
	}
	if p.Execution.Model == "" {
		return NewCompilationError(ErrMsgNoModel, nil)
	}
	return nil
}

// buildCompileContext creates the context data map for compilation.
func buildCompileContext(p *Prompt, input map[string]any) map[string]any {
	data := make(map[string]any)

	// Input data
	if input != nil {
		data[ContextKeyInput] = input
		// Also flatten input keys at top level for convenience
		for k, v := range input {
			data[k] = v
		}
	}

	// Meta information
	meta := map[string]any{
		"name":        p.Name,
		"description": p.Description,
		"type":        string(p.EffectiveType()),
	}
	data[ContextKeyMeta] = meta

	// Context
	if p.Context != nil {
		data[ContextKeyContext] = p.Context
		// Also make context values accessible via dot notation
		for k, v := range p.Context {
			if _, exists := data[k]; !exists {
				data[k] = v
			}
		}
	}

	// Constraints
	if p.Constraints != nil {
		constraintData := make(map[string]any)
		if p.Constraints.Behavioral != nil {
			// Convert to []any for template iteration
			behavioral := make([]any, len(p.Constraints.Behavioral))
			for i, b := range p.Constraints.Behavioral {
				behavioral[i] = b
			}
			constraintData["behavioral"] = behavioral
		}
		if p.Constraints.Safety != nil {
			safety := make([]any, len(p.Constraints.Safety))
			for i, s := range p.Constraints.Safety {
				safety[i] = s
			}
			constraintData["safety"] = safety
		}
		data[ContextKeyConstraints] = constraintData
	}

	return data
}

// compileMessages compiles message templates through the engine.
func compileMessages(ctx context.Context, engine *Engine, templates []MessageTemplate, data map[string]any, compiledBody string) ([]CompiledMessage, error) {
	messages := make([]CompiledMessage, 0, len(templates))

	// Add compiled body to context for {~prompty.include template="self" /~}
	data[ContextKeySelfBody] = compiledBody

	for i := range templates {
		mt := &templates[i]
		content, err := engine.Execute(ctx, mt.Content, data)
		if err != nil {
			return nil, NewCompilationError(ErrMsgCompileMessageFailed, err)
		}

		messages = append(messages, CompiledMessage{
			Role:    mt.Role,
			Content: strings.TrimSpace(content),
			Cache:   mt.Cache,
		})
	}

	return messages, nil
}

// buildDefaultMessages creates default messages when no explicit messages are defined.
func buildDefaultMessages(compiledBody string, input map[string]any) []CompiledMessage {
	messages := make([]CompiledMessage, 0, 2)

	// System message from compiled body
	if compiledBody != "" {
		messages = append(messages, CompiledMessage{
			Role:    RoleSystem,
			Content: strings.TrimSpace(compiledBody),
		})
	}

	// User message from input.message if present
	if input != nil {
		if msg, ok := input["message"]; ok {
			if msgStr, ok := msg.(string); ok && msgStr != "" {
				messages = append(messages, CompiledMessage{
					Role:    RoleUser,
					Content: msgStr,
				})
			}
		}
	}

	return messages
}

// injectSkillIntoSystemPrompt appends skill content to the system message.
func injectSkillIntoSystemPrompt(compiled *CompiledPrompt, slug string, content string) {
	marker := SkillInjectionMarkerStart + slug + SkillInjectionMarkerClose + "\n" +
		content + "\n" +
		SkillInjectionMarkerEnd + slug + SkillInjectionMarkerClose

	// Find system message and append
	for i := range compiled.Messages {
		if compiled.Messages[i].Role == RoleSystem {
			compiled.Messages[i].Content += "\n\n" + marker
			return
		}
	}

	// No system message found — create one
	compiled.Messages = append([]CompiledMessage{{
		Role:    RoleSystem,
		Content: marker,
	}}, compiled.Messages...)
}

// injectSkillIntoUserContext adds skill content as a user message.
func injectSkillIntoUserContext(compiled *CompiledPrompt, slug string, content string) {
	marker := SkillInjectionMarkerStart + slug + SkillInjectionMarkerClose + "\n" +
		content + "\n" +
		SkillInjectionMarkerEnd + slug + SkillInjectionMarkerClose

	// Insert before the last user message, or append
	compiled.Messages = append(compiled.Messages, CompiledMessage{
		Role:    RoleUser,
		Content: marker,
	})
}
