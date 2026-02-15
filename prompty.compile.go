package prompty

import (
	"context"
	"fmt"
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
	ErrMsgAgentDryRunNilPrompt  = "prompt is nil"
)

// AgentDryRunCategory categorizes the type of issue found during an agent dry run.
type AgentDryRunCategory string

const (
	// AgentDryRunCategoryParse indicates a parsing failure.
	AgentDryRunCategoryParse AgentDryRunCategory = "parse"
	// AgentDryRunCategoryValidation indicates a validation failure.
	AgentDryRunCategoryValidation AgentDryRunCategory = "validation"
	// AgentDryRunCategoryResolver indicates a resolver failure (e.g., skill not found).
	AgentDryRunCategoryResolver AgentDryRunCategory = "resolver"
	// AgentDryRunCategoryTemplate indicates a template parsing failure in messages or body.
	AgentDryRunCategoryTemplate AgentDryRunCategory = "template"
	// AgentDryRunCategorySkill indicates a skill resolution failure.
	AgentDryRunCategorySkill AgentDryRunCategory = "skill"
)

// AgentDryRunIssue represents a single issue found during an agent dry run.
type AgentDryRunIssue struct {
	// Category is the type of issue.
	Category AgentDryRunCategory
	// Message is a human-readable description of the issue.
	Message string
	// Location identifies where the issue was found (e.g., "message[0]", "skill:web-search", "body").
	Location string
	// Err is the underlying error, if any.
	Err error
}

// AgentDryRunResult contains all issues found during an agent dry run.
// Unlike the template-level DryRunResult (from Template.DryRun), this validates
// agent-specific concerns: skill resolution, message template parsing, and body parsing.
type AgentDryRunResult struct {
	// Issues is the list of all issues found.
	Issues []AgentDryRunIssue
	// SkillsResolved is the number of skills that were successfully resolved.
	SkillsResolved int
	// ToolsDefined is the number of tool functions defined.
	ToolsDefined int
	// MessageCount is the number of messages defined.
	MessageCount int
}

// OK returns true if no issues were found.
func (r *AgentDryRunResult) OK() bool {
	return len(r.Issues) == 0
}

// HasErrors returns true if any issues were found.
func (r *AgentDryRunResult) HasErrors() bool {
	return !r.OK()
}

// String returns a human-readable summary of the agent dry run result.
func (r *AgentDryRunResult) String() string {
	if r.OK() {
		return fmt.Sprintf(AgentDryRunSummaryOK, r.SkillsResolved, r.ToolsDefined, r.MessageCount)
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf(AgentDryRunSummaryIssues, len(r.Issues)))
	for i := range r.Issues {
		issue := &r.Issues[i]
		b.WriteString(fmt.Sprintf(AgentDryRunIssueFormat, issue.Category, issue.Location, issue.Message))
	}
	return b.String()
}

// Agent dry run summary format constants
const (
	AgentDryRunSummaryOK     = "agent dry run OK: %d skills resolved, %d tools defined, %d messages"
	AgentDryRunSummaryIssues = "agent dry run found %d issue(s):\n"
	AgentDryRunIssueFormat   = "  [%s] %s: %s\n"
)

// CompileOptions configures agent compilation.
//
// Compilation resolves all {~...~} tags in the agent's body and message templates,
// generates skills/tools catalogs, and produces a CompiledPrompt ready for LLM API submission.
//
// Example:
//
//	compiled, err := agent.CompileAgent(ctx, input, &prompty.CompileOptions{
//	    Resolver:            myDocumentResolver,
//	    SkillsCatalogFormat: prompty.CatalogFormatDetailed,
//	    Engine:              engineWithCustomResolvers,
//	})
type CompileOptions struct {
	// Resolver resolves skill/prompt/agent references during compilation.
	// Used by: catalog generation (to get skill descriptions), ActivateSkill (to get skill body),
	// and the {~prompty.ref~} tag resolver (to inline referenced prompts).
	// When nil, unresolvable references produce empty output (non-fatal for catalogs)
	// or errors (fatal for ActivateSkill).
	Resolver DocumentResolver
	// SkillsCatalogFormat controls the output format of {~prompty.skills_catalog~} tags.
	// Supported: "" (default markdown), "detailed", "compact".
	// "function_calling" is not supported for skills catalogs.
	SkillsCatalogFormat CatalogFormat
	// ToolsCatalogFormat controls the output format of {~prompty.tools_catalog~} tags.
	// Supported: "" (default markdown), "detailed", "compact", "function_calling" (JSON schema).
	ToolsCatalogFormat CatalogFormat
	// Engine is an optional pre-configured engine to use for compilation.
	// When set, user-registered resolvers, functions, and templates are available during compilation.
	// When nil, a new engine with default options is created for each compilation.
	Engine *Engine
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

// CompileOption is a functional option for configuring CompileOptions.
type CompileOption func(*CompileOptions)

// WithResolver sets the DocumentResolver for compilation.
func WithResolver(r DocumentResolver) CompileOption {
	return func(o *CompileOptions) {
		o.Resolver = r
	}
}

// WithCompileEngine sets a pre-configured engine for compilation.
func WithCompileEngine(e *Engine) CompileOption {
	return func(o *CompileOptions) {
		o.Engine = e
	}
}

// WithSkillsCatalogFormat sets the skills catalog output format.
func WithSkillsCatalogFormat(f CatalogFormat) CompileOption {
	return func(o *CompileOptions) {
		o.SkillsCatalogFormat = f
	}
}

// WithToolsCatalogFormat sets the tools catalog output format.
func WithToolsCatalogFormat(f CatalogFormat) CompileOption {
	return func(o *CompileOptions) {
		o.ToolsCatalogFormat = f
	}
}

// NewCompileOptions creates a CompileOptions from functional options.
//
// Example:
//
//	opts := prompty.NewCompileOptions(
//	    prompty.WithResolver(myResolver),
//	    prompty.WithSkillsCatalogFormat(prompty.CatalogFormatDetailed),
//	)
//	compiled, err := agent.CompileAgent(ctx, input, opts)
func NewCompileOptions(options ...CompileOption) *CompileOptions {
	opts := &CompileOptions{}
	for _, o := range options {
		o(opts)
	}
	return opts
}

// compileEngine returns the engine from options, or creates a new default engine.
func compileEngine(opts *CompileOptions) *Engine {
	if opts != nil && opts.Engine != nil {
		return opts.Engine
	}
	return MustNew()
}

// Compile compiles a prompt or skill document by executing its body through the engine.
// For simple prompts/skills, this resolves all {~...~} tags in the body.
func (p *Prompt) Compile(ctx context.Context, input map[string]any, opts *CompileOptions) (string, error) {
	if p == nil {
		return "", NewCompilationError(ErrMsgCompilationFailed, nil)
	}

	// Build context data
	data := buildCompileContext(p, input)

	// Get or create engine
	engine := compileEngine(opts)
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

	// Get or create engine for compilation
	engine := compileEngine(opts)

	// Register "self" template with the body content
	if p.Body != "" {
		if err := engine.RegisterTemplate(TemplateNameSelf, p.Body); err != nil {
			return nil, NewCompilationError(ErrMsgCompileBodyFailed, err)
		}
	}

	// Compile body
	compiledBody, err := engine.Execute(ctx, p.Body, data)
	if err != nil {
		return nil, NewCompileBodyError(err)
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
			return nil, NewCompileSkillError(skillSlug, err)
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
	compiledSkillBody, err := compileEngine(opts).Execute(ctx, skillBody, data)
	if err != nil {
		return nil, NewCompileSkillError(skillSlug, err)
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
			return nil, NewCompileMessageError(i, mt.Role, err)
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

// ToOpenAIMessages converts compiled messages to OpenAI Chat Completions API format.
// Returns a slice of message objects with "role" and "content" keys.
func (cp *CompiledPrompt) ToOpenAIMessages() []map[string]any {
	if cp == nil || len(cp.Messages) == 0 {
		return nil
	}

	result := make([]map[string]any, 0, len(cp.Messages))
	for _, msg := range cp.Messages {
		result = append(result, map[string]any{
			AttrRole:  msg.Role,
			"content": msg.Content,
		})
	}
	return result
}

// ToAnthropicMessages converts compiled messages to Anthropic Messages API format.
// Returns a map with "system" (string) and "messages" (slice of role/content maps).
// System messages are extracted and concatenated into the top-level "system" field,
// as required by the Anthropic API.
func (cp *CompiledPrompt) ToAnthropicMessages() map[string]any {
	if cp == nil || len(cp.Messages) == 0 {
		return nil
	}

	var systemParts []string
	messages := make([]map[string]any, 0, len(cp.Messages))

	for _, msg := range cp.Messages {
		if msg.Role == RoleSystem {
			systemParts = append(systemParts, msg.Content)
			continue
		}
		messages = append(messages, map[string]any{
			AttrRole:  msg.Role,
			"content": msg.Content,
		})
	}

	result := make(map[string]any, 2)
	if len(systemParts) > 0 {
		result[RoleSystem] = strings.Join(systemParts, "\n\n")
	}
	result["messages"] = messages
	return result
}

// ToGeminiContents converts compiled messages to Gemini/Vertex AI API format.
// System messages are returned separately in the "system_instruction" key.
// Other messages use Gemini roles: "user" and "model" (instead of "assistant").
func (cp *CompiledPrompt) ToGeminiContents() map[string]any {
	if cp == nil || len(cp.Messages) == 0 {
		return nil
	}

	var systemParts []string
	contents := make([]map[string]any, 0, len(cp.Messages))

	for _, msg := range cp.Messages {
		if msg.Role == RoleSystem {
			systemParts = append(systemParts, msg.Content)
			continue
		}

		role := msg.Role
		if role == RoleAssistant {
			role = "model"
		}

		contents = append(contents, map[string]any{
			AttrRole: role,
			"parts":  []map[string]string{{"text": msg.Content}},
		})
	}

	result := make(map[string]any, 2)
	if len(systemParts) > 0 {
		result["system_instruction"] = map[string]any{
			"parts": []map[string]string{{"text": strings.Join(systemParts, "\n\n")}},
		}
	}
	result["contents"] = contents
	return result
}

// ToProviderMessages converts compiled messages to the format required by the given provider.
// Supported providers: "openai", "azure", "anthropic", "gemini", "google", "vertex".
// Returns the provider-specific message structure, or an error for unsupported providers.
func (cp *CompiledPrompt) ToProviderMessages(provider string) (any, error) {
	if cp == nil {
		return nil, nil
	}

	switch provider {
	case ProviderOpenAI, ProviderAzure:
		return cp.ToOpenAIMessages(), nil
	case ProviderAnthropic:
		return cp.ToAnthropicMessages(), nil
	case ProviderGoogle, ProviderGemini, ProviderVertex:
		return cp.ToGeminiContents(), nil
	default:
		return nil, NewProviderMessageError(provider)
	}
}

// AgentDryRun validates all references and templates in an agent document without producing output.
// It collects ALL issues rather than stopping at the first error, making it ideal for
// pre-flight checks before compilation.
//
// The method checks:
//   - Prompt validation (Validate or ValidateAsAgent for agents)
//   - Skill reference resolution via opts.Resolver
//   - Message template parseability through the engine
//   - Body template parseability through the engine
//   - Tool definition counts
//
// Returns an AgentDryRunResult with all issues collected. Use result.OK() to check success.
func (p *Prompt) AgentDryRun(ctx context.Context, opts *CompileOptions) *AgentDryRunResult {
	result := &AgentDryRunResult{}

	// Handle nil prompt
	if p == nil {
		result.Issues = append(result.Issues, AgentDryRunIssue{
			Category: AgentDryRunCategoryParse,
			Message:  ErrMsgAgentDryRunNilPrompt,
			Location: "prompt",
		})
		return result
	}

	// Step 1: Validate prompt
	if p.IsAgent() {
		if err := p.ValidateAsAgent(); err != nil {
			result.Issues = append(result.Issues, AgentDryRunIssue{
				Category: AgentDryRunCategoryValidation,
				Message:  err.Error(),
				Location: "prompt",
				Err:      err,
			})
		}
	} else {
		if err := p.Validate(); err != nil {
			result.Issues = append(result.Issues, AgentDryRunIssue{
				Category: AgentDryRunCategoryValidation,
				Message:  err.Error(),
				Location: "prompt",
				Err:      err,
			})
		}
	}

	// Step 2: Resolve skills
	if len(p.Skills) > 0 {
		for i := range p.Skills {
			skill := &p.Skills[i]
			slug := skill.GetSlug()
			location := fmt.Sprintf("skill:%s", slug)

			if skill.IsInline() {
				result.SkillsResolved++
				continue
			}

			if opts == nil || opts.Resolver == nil {
				result.Issues = append(result.Issues, AgentDryRunIssue{
					Category: AgentDryRunCategorySkill,
					Message:  ErrMsgNoDocumentResolver,
					Location: location,
				})
				continue
			}

			_, err := opts.Resolver.ResolveSkill(ctx, skill.Slug)
			if err != nil {
				result.Issues = append(result.Issues, AgentDryRunIssue{
					Category: AgentDryRunCategorySkill,
					Message:  err.Error(),
					Location: location,
					Err:      err,
				})
			} else {
				result.SkillsResolved++
			}
		}
	}

	// Step 3: Parse message templates
	engine := compileEngine(opts)
	result.MessageCount = len(p.Messages)
	for i := range p.Messages {
		mt := &p.Messages[i]
		location := fmt.Sprintf("message[%d]", i)

		_, err := engine.Parse(mt.Content)
		if err != nil {
			result.Issues = append(result.Issues, AgentDryRunIssue{
				Category: AgentDryRunCategoryTemplate,
				Message:  err.Error(),
				Location: location,
				Err:      err,
			})
		}
	}

	// Step 4: Parse body template
	if p.Body != "" {
		_, err := engine.Parse(p.Body)
		if err != nil {
			result.Issues = append(result.Issues, AgentDryRunIssue{
				Category: AgentDryRunCategoryTemplate,
				Message:  err.Error(),
				Location: "body",
				Err:      err,
			})
		}
	}

	// Step 5: Count tools
	if p.Tools != nil && len(p.Tools.Functions) > 0 {
		result.ToolsDefined = len(p.Tools.Functions)
	}

	return result
}
