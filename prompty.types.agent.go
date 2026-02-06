package prompty

import "strings"

// SkillRef represents a reference to a skill, either by slug or inline.
type SkillRef struct {
	// Slug is the skill identifier (e.g., "my-skill" or "my-skill@v2")
	Slug string `yaml:"slug,omitempty" json:"slug,omitempty"`
	// Version overrides the version parsed from slug@version syntax
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
	// Injection controls how the skill is injected into the agent
	Injection SkillInjection `yaml:"injection,omitempty" json:"injection,omitempty"`
	// Inline contains an inline skill definition (alternative to Slug)
	Inline *InlineSkill `yaml:"inline,omitempty" json:"inline,omitempty"`
	// Execution overrides for this skill activation
	Execution *ExecutionConfig `yaml:"execution,omitempty" json:"execution,omitempty"`
}

// InlineSkill defines a skill inline within an agent definition.
type InlineSkill struct {
	// Slug is the identifier for the inline skill
	Slug string `yaml:"slug" json:"slug"`
	// Description of the inline skill
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	// Body is the skill template content
	Body string `yaml:"body" json:"body"`
}

// ToolsConfig configures the tools available to an agent.
type ToolsConfig struct {
	// Functions is a list of function definitions for tool calling
	Functions []*FunctionDef `yaml:"functions,omitempty" json:"functions,omitempty"`
	// MCPServers lists MCP (Model Context Protocol) server configurations
	MCPServers []*MCPServer `yaml:"mcp_servers,omitempty" json:"mcp_servers,omitempty"`
	// ToolChoice controls tool selection strategy: "auto", "none", "required"
	ToolChoice string `yaml:"tool_choice,omitempty" json:"tool_choice,omitempty"`
}

// MCPServer configures a Model Context Protocol server.
type MCPServer struct {
	// Name is the server identifier
	Name string `yaml:"name" json:"name"`
	// URL is the server endpoint
	URL string `yaml:"url" json:"url"`
	// Transport is the connection type (e.g., "sse", "stdio")
	Transport string `yaml:"transport,omitempty" json:"transport,omitempty"`
	// Tools lists specific tools to expose from this server
	Tools []string `yaml:"tools,omitempty" json:"tools,omitempty"`
}

// ConstraintsConfig defines agent behavioral and operational constraints.
type ConstraintsConfig struct {
	// Behavioral constraints (e.g., "Be concise", "Use formal language")
	Behavioral []string `yaml:"behavioral,omitempty" json:"behavioral,omitempty"`
	// Safety constraints (e.g., "Never share PII", "Refuse harmful requests")
	Safety []string `yaml:"safety,omitempty" json:"safety,omitempty"`
	// Operational constraints with structured settings
	Operational *OperationalConstraints `yaml:"operational,omitempty" json:"operational,omitempty"`
}

// OperationalConstraints defines structured operational limits.
type OperationalConstraints struct {
	// MaxTurns limits the number of conversation turns
	MaxTurns *int `yaml:"max_turns,omitempty" json:"max_turns,omitempty"`
	// MaxTokensPerTurn limits tokens per individual turn
	MaxTokensPerTurn *int `yaml:"max_tokens_per_turn,omitempty" json:"max_tokens_per_turn,omitempty"`
	// AllowedDomains restricts which domains the agent can access
	AllowedDomains []string `yaml:"allowed_domains,omitempty" json:"allowed_domains,omitempty"`
	// BlockedDomains lists domains the agent must not access
	BlockedDomains []string `yaml:"blocked_domains,omitempty" json:"blocked_domains,omitempty"`
}

// MessageTemplate represents a message template defined in YAML frontmatter.
// This is distinct from Message which is the extracted chat message after execution.
type MessageTemplate struct {
	// Role of the message: "system", "user", "assistant", "tool"
	Role string `yaml:"role" json:"role"`
	// Content is the template content (may contain {~...~} tags)
	Content string `yaml:"content" json:"content"`
	// Cache indicates whether this message should be cached
	Cache bool `yaml:"cache,omitempty" json:"cache,omitempty"`
}

// --- SkillRef methods ---

// Validate checks the skill reference for consistency.
func (s *SkillRef) Validate() error {
	if s == nil {
		return nil
	}

	// Either slug or inline must be provided
	if s.Slug == "" && s.Inline == nil {
		return NewAgentError(ErrMsgSkillRefEmpty, nil)
	}

	// If both slug and inline are set, that's ambiguous
	if s.Slug != "" && s.Inline != nil {
		return NewAgentError(ErrMsgSkillRefAmbiguous, nil)
	}

	// Validate inline skill if present
	if s.Inline != nil {
		if err := s.Inline.Validate(); err != nil {
			return err
		}
	}

	// Validate injection mode if set
	if s.Injection != "" {
		if !isValidSkillInjection(s.Injection) {
			return NewAgentError(ErrMsgInvalidSkillInjection, nil)
		}
	}

	// Validate execution override if present
	if s.Execution != nil {
		if err := s.Execution.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// GetSlug returns the resolved slug, parsing slug@version syntax if needed.
func (s *SkillRef) GetSlug() string {
	if s == nil {
		return ""
	}
	if s.Inline != nil {
		return s.Inline.Slug
	}
	slug := s.Slug
	if atIdx := strings.LastIndex(slug, "@"); atIdx > 0 {
		slug = slug[:atIdx]
	}
	return slug
}

// GetVersion returns the resolved version.
// Explicit Version field takes precedence over slug@version syntax.
func (s *SkillRef) GetVersion() string {
	if s == nil {
		return RefVersionLatest
	}
	if s.Version != "" {
		return s.Version
	}
	if atIdx := strings.LastIndex(s.Slug, "@"); atIdx > 0 {
		return s.Slug[atIdx+1:]
	}
	return RefVersionLatest
}

// IsInline returns true if this is an inline skill definition.
func (s *SkillRef) IsInline() bool {
	return s != nil && s.Inline != nil
}

// Clone creates a deep copy of the SkillRef.
func (s *SkillRef) Clone() *SkillRef {
	if s == nil {
		return nil
	}

	clone := &SkillRef{
		Slug:      s.Slug,
		Version:   s.Version,
		Injection: s.Injection,
	}

	if s.Inline != nil {
		clone.Inline = s.Inline.Clone()
	}
	if s.Execution != nil {
		clone.Execution = s.Execution.Clone()
	}

	return clone
}

// --- InlineSkill methods ---

// Validate checks the inline skill definition.
func (is *InlineSkill) Validate() error {
	if is == nil {
		return nil
	}
	if is.Slug == "" {
		return NewAgentError(ErrMsgInlineSkillNoSlug, nil)
	}
	if is.Body == "" {
		return NewAgentError(ErrMsgInlineSkillNoBody, nil)
	}
	return nil
}

// Clone creates a deep copy of the InlineSkill.
func (is *InlineSkill) Clone() *InlineSkill {
	if is == nil {
		return nil
	}
	return &InlineSkill{
		Slug:        is.Slug,
		Description: is.Description,
		Body:        is.Body,
	}
}

// --- ToolsConfig methods ---

// HasTools returns true if any tools are configured.
func (tc *ToolsConfig) HasTools() bool {
	if tc == nil {
		return false
	}
	return len(tc.Functions) > 0 || len(tc.MCPServers) > 0
}

// Validate checks the tools configuration.
func (tc *ToolsConfig) Validate() error {
	if tc == nil {
		return nil
	}

	for _, srv := range tc.MCPServers {
		if srv.Name == "" {
			return NewAgentError(ErrMsgMCPServerNameEmpty, nil)
		}
		if srv.URL == "" {
			return NewAgentError(ErrMsgMCPServerURLEmpty, nil)
		}
	}

	return nil
}

// Clone creates a deep copy of the ToolsConfig.
func (tc *ToolsConfig) Clone() *ToolsConfig {
	if tc == nil {
		return nil
	}

	clone := &ToolsConfig{
		ToolChoice: tc.ToolChoice,
	}

	if tc.Functions != nil {
		clone.Functions = make([]*FunctionDef, len(tc.Functions))
		for i, f := range tc.Functions {
			cloned := *f
			if f.Parameters != nil {
				cloned.Parameters = copySchema(f.Parameters)
			}
			if f.Returns != nil {
				cloned.Returns = copySchema(f.Returns)
			}
			clone.Functions[i] = &cloned
		}
	}

	if tc.MCPServers != nil {
		clone.MCPServers = make([]*MCPServer, len(tc.MCPServers))
		for i, srv := range tc.MCPServers {
			clonedSrv := *srv
			if srv.Tools != nil {
				clonedSrv.Tools = make([]string, len(srv.Tools))
				copy(clonedSrv.Tools, srv.Tools)
			}
			clone.MCPServers[i] = &clonedSrv
		}
	}

	return clone
}

// --- ConstraintsConfig methods ---

// Clone creates a deep copy of the ConstraintsConfig.
func (cc *ConstraintsConfig) Clone() *ConstraintsConfig {
	if cc == nil {
		return nil
	}

	clone := &ConstraintsConfig{}

	if cc.Behavioral != nil {
		clone.Behavioral = make([]string, len(cc.Behavioral))
		copy(clone.Behavioral, cc.Behavioral)
	}

	if cc.Safety != nil {
		clone.Safety = make([]string, len(cc.Safety))
		copy(clone.Safety, cc.Safety)
	}

	if cc.Operational != nil {
		clone.Operational = cc.Operational.Clone()
	}

	return clone
}

// --- OperationalConstraints methods ---

// Clone creates a deep copy of the OperationalConstraints.
func (oc *OperationalConstraints) Clone() *OperationalConstraints {
	if oc == nil {
		return nil
	}

	clone := &OperationalConstraints{}

	if oc.MaxTurns != nil {
		mt := *oc.MaxTurns
		clone.MaxTurns = &mt
	}
	if oc.MaxTokensPerTurn != nil {
		mt := *oc.MaxTokensPerTurn
		clone.MaxTokensPerTurn = &mt
	}
	if oc.AllowedDomains != nil {
		clone.AllowedDomains = make([]string, len(oc.AllowedDomains))
		copy(clone.AllowedDomains, oc.AllowedDomains)
	}
	if oc.BlockedDomains != nil {
		clone.BlockedDomains = make([]string, len(oc.BlockedDomains))
		copy(clone.BlockedDomains, oc.BlockedDomains)
	}

	return clone
}

// --- MessageTemplate methods ---

// Validate checks the message template.
func (mt *MessageTemplate) Validate() error {
	if mt == nil {
		return nil
	}
	if mt.Role == "" {
		return NewAgentError(ErrMsgMessageTemplateNoRole, nil)
	}
	if mt.Content == "" {
		return NewAgentError(ErrMsgMessageTemplateNoBody, nil)
	}
	// Validate role
	switch mt.Role {
	case RoleSystem, RoleUser, RoleAssistant, RoleTool:
		// Valid
	default:
		return NewAgentError(ErrMsgMessageInvalidRole, nil)
	}
	return nil
}

// --- Helper functions ---

// isValidSkillInjection checks if a skill injection mode is valid.
func isValidSkillInjection(mode SkillInjection) bool {
	switch mode {
	case SkillInjectionNone, SkillInjectionSystemPrompt, SkillInjectionUserContext:
		return true
	default:
		return false
	}
}

// isValidDocumentType checks if a document type is valid.
func isValidDocumentType(dt DocumentType) bool {
	switch dt {
	case DocumentTypePrompt, DocumentTypeSkill, DocumentTypeAgent, "":
		return true
	default:
		return false
	}
}
