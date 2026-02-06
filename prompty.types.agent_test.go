package prompty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- SkillRef tests ---

func TestSkillRef_Validate_EmptySlugAndInline(t *testing.T) {
	ref := &SkillRef{}
	err := ref.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgSkillRefEmpty)
}

func TestSkillRef_Validate_BothSlugAndInline(t *testing.T) {
	ref := &SkillRef{
		Slug:   "my-skill",
		Inline: &InlineSkill{Slug: "inline-skill", Body: "body"},
	}
	err := ref.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgSkillRefAmbiguous)
}

func TestSkillRef_Validate_ValidSlug(t *testing.T) {
	ref := &SkillRef{Slug: "my-skill"}
	err := ref.Validate()
	assert.NoError(t, err)
}

func TestSkillRef_Validate_ValidSlugWithVersion(t *testing.T) {
	ref := &SkillRef{Slug: "my-skill@v2"}
	err := ref.Validate()
	assert.NoError(t, err)
}

func TestSkillRef_Validate_ValidInline(t *testing.T) {
	ref := &SkillRef{
		Inline: &InlineSkill{Slug: "inline-skill", Body: "some body"},
	}
	err := ref.Validate()
	assert.NoError(t, err)
}

func TestSkillRef_Validate_InvalidInjection(t *testing.T) {
	ref := &SkillRef{
		Slug:      "my-skill",
		Injection: SkillInjection("invalid"),
	}
	err := ref.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgInvalidSkillInjection)
}

func TestSkillRef_Validate_ValidInjectionModes(t *testing.T) {
	modes := []SkillInjection{
		SkillInjectionNone,
		SkillInjectionSystemPrompt,
		SkillInjectionUserContext,
	}
	for _, mode := range modes {
		ref := &SkillRef{Slug: "my-skill", Injection: mode}
		assert.NoError(t, ref.Validate(), "mode: %s", mode)
	}
}

func TestSkillRef_GetSlug(t *testing.T) {
	tests := []struct {
		name     string
		ref      *SkillRef
		expected string
	}{
		{name: "nil ref", ref: nil, expected: ""},
		{name: "simple slug", ref: &SkillRef{Slug: "my-skill"}, expected: "my-skill"},
		{name: "slug with version", ref: &SkillRef{Slug: "my-skill@v2"}, expected: "my-skill"},
		{name: "inline", ref: &SkillRef{Inline: &InlineSkill{Slug: "inline-skill"}}, expected: "inline-skill"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.ref.GetSlug())
		})
	}
}

func TestSkillRef_GetVersion(t *testing.T) {
	tests := []struct {
		name     string
		ref      *SkillRef
		expected string
	}{
		{name: "nil ref", ref: nil, expected: RefVersionLatest},
		{name: "no version", ref: &SkillRef{Slug: "my-skill"}, expected: RefVersionLatest},
		{name: "slug@version", ref: &SkillRef{Slug: "my-skill@v2"}, expected: "v2"},
		{name: "explicit version overrides slug", ref: &SkillRef{Slug: "my-skill@v2", Version: "v3"}, expected: "v3"},
		{name: "explicit version only", ref: &SkillRef{Slug: "my-skill", Version: "v1"}, expected: "v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.ref.GetVersion())
		})
	}
}

func TestSkillRef_IsInline(t *testing.T) {
	assert.False(t, (*SkillRef)(nil).IsInline())
	assert.False(t, (&SkillRef{Slug: "my-skill"}).IsInline())
	assert.True(t, (&SkillRef{Inline: &InlineSkill{Slug: "x", Body: "y"}}).IsInline())
}

func TestSkillRef_Clone(t *testing.T) {
	original := &SkillRef{
		Slug:      "my-skill@v2",
		Version:   "v3",
		Injection: SkillInjectionSystemPrompt,
		Inline:    &InlineSkill{Slug: "inline", Description: "desc", Body: "body"},
		Execution: &ExecutionConfig{Provider: "openai", Model: "gpt-4"},
	}

	clone := original.Clone()
	require.NotNil(t, clone)
	assert.Equal(t, original.Slug, clone.Slug)
	assert.Equal(t, original.Version, clone.Version)
	assert.Equal(t, original.Injection, clone.Injection)
	assert.NotSame(t, original.Inline, clone.Inline)
	assert.Equal(t, original.Inline.Slug, clone.Inline.Slug)
	assert.NotSame(t, original.Execution, clone.Execution)
	assert.Equal(t, original.Execution.Provider, clone.Execution.Provider)

	// Nil clone
	assert.Nil(t, (*SkillRef)(nil).Clone())
}

// --- InlineSkill tests ---

func TestInlineSkill_Validate(t *testing.T) {
	// Nil is valid
	assert.NoError(t, (*InlineSkill)(nil).Validate())

	// Missing slug
	err := (&InlineSkill{Body: "body"}).Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgInlineSkillNoSlug)

	// Missing body
	err = (&InlineSkill{Slug: "my-skill"}).Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgInlineSkillNoBody)

	// Valid
	assert.NoError(t, (&InlineSkill{Slug: "my-skill", Body: "body"}).Validate())
}

func TestInlineSkill_Clone(t *testing.T) {
	original := &InlineSkill{Slug: "my-skill", Description: "desc", Body: "body"}
	clone := original.Clone()
	require.NotNil(t, clone)
	assert.Equal(t, original.Slug, clone.Slug)
	assert.Equal(t, original.Description, clone.Description)
	assert.Equal(t, original.Body, clone.Body)
	assert.NotSame(t, original, clone)

	assert.Nil(t, (*InlineSkill)(nil).Clone())
}

// --- ToolsConfig tests ---

func TestToolsConfig_HasTools(t *testing.T) {
	assert.False(t, (*ToolsConfig)(nil).HasTools())
	assert.False(t, (&ToolsConfig{}).HasTools())
	assert.True(t, (&ToolsConfig{Functions: []*FunctionDef{{Name: "f"}}}).HasTools())
	assert.True(t, (&ToolsConfig{MCPServers: []*MCPServer{{Name: "s", URL: "http://x"}}}).HasTools())
}

func TestToolsConfig_Validate(t *testing.T) {
	// Nil is valid
	assert.NoError(t, (*ToolsConfig)(nil).Validate())

	// Empty MCP server name
	err := (&ToolsConfig{MCPServers: []*MCPServer{{URL: "http://x"}}}).Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgMCPServerNameEmpty)

	// Empty MCP server URL
	err = (&ToolsConfig{MCPServers: []*MCPServer{{Name: "srv"}}}).Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgMCPServerURLEmpty)

	// Valid
	assert.NoError(t, (&ToolsConfig{
		Functions:  []*FunctionDef{{Name: "f"}},
		MCPServers: []*MCPServer{{Name: "srv", URL: "http://localhost:8080"}},
	}).Validate())
}

func TestToolsConfig_Clone(t *testing.T) {
	original := &ToolsConfig{
		Functions: []*FunctionDef{
			{Name: "func1", Parameters: map[string]any{"type": "object"}, Returns: map[string]any{"type": "string"}},
		},
		MCPServers: []*MCPServer{
			{Name: "srv", URL: "http://x", Transport: "sse", Tools: []string{"tool1"}},
		},
		ToolChoice: "auto",
	}

	clone := original.Clone()
	require.NotNil(t, clone)
	assert.Equal(t, original.ToolChoice, clone.ToolChoice)
	assert.Len(t, clone.Functions, 1)
	assert.NotSame(t, original.Functions[0], clone.Functions[0])
	assert.Equal(t, original.Functions[0].Name, clone.Functions[0].Name)
	assert.Len(t, clone.MCPServers, 1)
	assert.NotSame(t, original.MCPServers[0], clone.MCPServers[0])
	assert.Equal(t, original.MCPServers[0].Name, clone.MCPServers[0].Name)

	assert.Nil(t, (*ToolsConfig)(nil).Clone())
}

// --- ConstraintsConfig tests ---

func TestConstraintsConfig_Clone(t *testing.T) {
	maxTurns := 10
	maxTokens := 1000
	original := &ConstraintsConfig{
		Behavioral: []string{"be concise", "use formal language"},
		Safety:     []string{"no PII"},
		Operational: &OperationalConstraints{
			MaxTurns:         &maxTurns,
			MaxTokensPerTurn: &maxTokens,
			AllowedDomains:   []string{"example.com"},
			BlockedDomains:   []string{"evil.com"},
		},
	}

	clone := original.Clone()
	require.NotNil(t, clone)
	assert.Equal(t, original.Behavioral, clone.Behavioral)
	assert.Equal(t, original.Safety, clone.Safety)
	require.NotNil(t, clone.Operational)
	assert.Equal(t, *original.Operational.MaxTurns, *clone.Operational.MaxTurns)
	assert.Equal(t, *original.Operational.MaxTokensPerTurn, *clone.Operational.MaxTokensPerTurn)
	assert.Equal(t, original.Operational.AllowedDomains, clone.Operational.AllowedDomains)
	assert.Equal(t, original.Operational.BlockedDomains, clone.Operational.BlockedDomains)

	// Verify independence
	clone.Behavioral[0] = "changed"
	assert.NotEqual(t, original.Behavioral[0], clone.Behavioral[0])

	assert.Nil(t, (*ConstraintsConfig)(nil).Clone())
}

// --- MessageTemplate tests ---

func TestMessageTemplate_Validate(t *testing.T) {
	// Nil is valid
	assert.NoError(t, (*MessageTemplate)(nil).Validate())

	// Missing role
	err := (&MessageTemplate{Content: "hello"}).Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgMessageTemplateNoRole)

	// Missing content
	err = (&MessageTemplate{Role: RoleSystem}).Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgMessageTemplateNoBody)

	// Invalid role
	err = (&MessageTemplate{Role: "invalid", Content: "hello"}).Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgMessageInvalidRole)

	// Valid roles
	validRoles := []string{RoleSystem, RoleUser, RoleAssistant, RoleTool}
	for _, role := range validRoles {
		mt := &MessageTemplate{Role: role, Content: "content"}
		assert.NoError(t, mt.Validate(), "role: %s", role)
	}
}

// --- DocumentType tests ---

func TestDocumentType_Validation(t *testing.T) {
	assert.True(t, isValidDocumentType(DocumentTypePrompt))
	assert.True(t, isValidDocumentType(DocumentTypeSkill))
	assert.True(t, isValidDocumentType(DocumentTypeAgent))
	assert.True(t, isValidDocumentType("")) // empty is valid (defaults to skill)
	assert.False(t, isValidDocumentType(DocumentType("invalid")))
}

// --- Prompt type-specific validation tests ---

func TestPrompt_Validate_PromptTypeNoSkills(t *testing.T) {
	p := &Prompt{
		Name:        "my-prompt",
		Description: "A simple prompt",
		Type:        DocumentTypePrompt,
		Skills:      []SkillRef{{Slug: "some-skill"}},
	}
	err := p.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgPromptNoSkillsAllowed)
}

func TestPrompt_Validate_PromptTypeNoTools(t *testing.T) {
	p := &Prompt{
		Name:        "my-prompt",
		Description: "A simple prompt",
		Type:        DocumentTypePrompt,
		Tools:       &ToolsConfig{Functions: []*FunctionDef{{Name: "f"}}},
	}
	err := p.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgPromptNoToolsAllowed)
}

func TestPrompt_Validate_PromptTypeNoConstraints(t *testing.T) {
	p := &Prompt{
		Name:        "my-prompt",
		Description: "A simple prompt",
		Type:        DocumentTypePrompt,
		Constraints: &ConstraintsConfig{Behavioral: []string{"be nice"}},
	}
	err := p.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgPromptNoConstraints)
}

func TestPrompt_Validate_SkillTypeNoSkills(t *testing.T) {
	p := &Prompt{
		Name:        "my-skill",
		Description: "A skill",
		Type:        DocumentTypeSkill,
		Skills:      []SkillRef{{Slug: "nested-skill"}},
	}
	err := p.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgSkillNoSkillsAllowed)
}

func TestPrompt_Validate_AgentTypeValid(t *testing.T) {
	p := &Prompt{
		Name:        "my-agent",
		Description: "An agent",
		Type:        DocumentTypeAgent,
		Skills:      []SkillRef{{Slug: "search-skill"}},
		Tools: &ToolsConfig{
			Functions: []*FunctionDef{{Name: "search"}},
		},
		Constraints: &ConstraintsConfig{
			Behavioral: []string{"be helpful"},
		},
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: "You are a helpful assistant."},
		},
	}
	assert.NoError(t, p.Validate())
}

func TestPrompt_Validate_InvalidDocumentType(t *testing.T) {
	p := &Prompt{
		Name:        "my-doc",
		Description: "A document",
		Type:        DocumentType("bogus"),
	}
	err := p.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgInvalidDocumentType)
}

func TestPrompt_EffectiveType(t *testing.T) {
	assert.Equal(t, DocumentTypeSkill, (*Prompt)(nil).EffectiveType())
	assert.Equal(t, DocumentTypeSkill, (&Prompt{}).EffectiveType())
	assert.Equal(t, DocumentTypeAgent, (&Prompt{Type: DocumentTypeAgent}).EffectiveType())
	assert.Equal(t, DocumentTypePrompt, (&Prompt{Type: DocumentTypePrompt}).EffectiveType())
}

func TestPrompt_IsAgent(t *testing.T) {
	assert.False(t, (*Prompt)(nil).IsAgent())
	assert.False(t, (&Prompt{}).IsAgent())
	assert.True(t, (&Prompt{Type: DocumentTypeAgent}).IsAgent())
}

func TestPrompt_IsSkill(t *testing.T) {
	assert.False(t, (*Prompt)(nil).IsSkill())
	assert.True(t, (&Prompt{}).IsSkill()) // default type
	assert.True(t, (&Prompt{Type: DocumentTypeSkill}).IsSkill())
}

func TestPrompt_IsPrompt(t *testing.T) {
	assert.False(t, (*Prompt)(nil).IsPrompt())
	assert.False(t, (&Prompt{}).IsPrompt())
	assert.True(t, (&Prompt{Type: DocumentTypePrompt}).IsPrompt())
}

func TestPrompt_Clone_WithAgentFields(t *testing.T) {
	original := &Prompt{
		Name:        "my-agent",
		Description: "An agent",
		Type:        DocumentTypeAgent,
		Body:        "agent body content",
		Skills: []SkillRef{
			{Slug: "skill-1", Injection: SkillInjectionSystemPrompt},
		},
		Tools: &ToolsConfig{
			Functions: []*FunctionDef{{Name: "search"}},
		},
		Context: map[string]any{
			"company": "Acme Corp",
		},
		Constraints: &ConstraintsConfig{
			Behavioral: []string{"be helpful"},
		},
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: "system message"},
		},
	}

	clone := original.Clone()
	require.NotNil(t, clone)
	assert.Equal(t, original.Type, clone.Type)
	assert.Equal(t, original.Body, clone.Body)

	// Skills cloned
	require.Len(t, clone.Skills, 1)
	assert.Equal(t, original.Skills[0].Slug, clone.Skills[0].Slug)

	// Tools cloned
	require.NotNil(t, clone.Tools)
	assert.NotSame(t, original.Tools, clone.Tools)

	// Context cloned
	require.NotNil(t, clone.Context)
	assert.Equal(t, original.Context["company"], clone.Context["company"])

	// Constraints cloned
	require.NotNil(t, clone.Constraints)
	assert.NotSame(t, original.Constraints, clone.Constraints)

	// Messages cloned
	require.Len(t, clone.Messages, 1)
	assert.Equal(t, original.Messages[0].Role, clone.Messages[0].Role)
}

// --- FunctionDef tool conversion tests ---

func TestFunctionDef_ToOpenAITool(t *testing.T) {
	f := &FunctionDef{
		Name:        "search",
		Description: "Search the web",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string"},
			},
			"required": []any{"query"},
		},
		Strict: true,
	}

	result := f.ToOpenAITool()
	require.NotNil(t, result)
	assert.Equal(t, "function", result["type"])

	funcDef, ok := result["function"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "search", funcDef["name"])
	assert.Equal(t, "Search the web", funcDef["description"])
	assert.Equal(t, true, funcDef["strict"])
	assert.NotNil(t, funcDef["parameters"])
}

func TestFunctionDef_ToAnthropicTool(t *testing.T) {
	f := &FunctionDef{
		Name:        "search",
		Description: "Search the web",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string"},
			},
		},
	}

	result := f.ToAnthropicTool()
	require.NotNil(t, result)
	assert.Equal(t, "search", result["name"])
	assert.Equal(t, "Search the web", result["description"])
	assert.NotNil(t, result["input_schema"])
}

func TestFunctionDef_ToOpenAITool_Nil(t *testing.T) {
	assert.Nil(t, (*FunctionDef)(nil).ToOpenAITool())
}

func TestFunctionDef_ToAnthropicTool_Nil(t *testing.T) {
	assert.Nil(t, (*FunctionDef)(nil).ToAnthropicTool())
}
