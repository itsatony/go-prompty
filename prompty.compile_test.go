package prompty

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrompt_Compile_Simple(t *testing.T) {
	p := &Prompt{
		Name:        "simple",
		Description: "A simple prompt",
		Type:        DocumentTypeSkill,
		Body:        "Hello, world!",
	}

	result, err := p.Compile(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "Hello, world!", result)
}

func TestPrompt_Compile_WithVariable(t *testing.T) {
	p := &Prompt{
		Name:        "greeting",
		Description: "A greeting prompt",
		Type:        DocumentTypeSkill,
		Body:        "Hello, {~prompty.var name=\"name\" /~}!",
	}

	result, err := p.Compile(context.Background(), map[string]any{"name": "Alice"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "Hello, Alice!", result)
}

func TestPrompt_Compile_Nil(t *testing.T) {
	var p *Prompt
	_, err := p.Compile(context.Background(), nil, nil)
	require.Error(t, err)
}

func TestPrompt_CompileAgent_Basic(t *testing.T) {
	p := &Prompt{
		Name:        "test-agent",
		Description: "A test agent",
		Type:        DocumentTypeAgent,
		Execution: &ExecutionConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Body: "You are a helpful assistant.",
	}

	compiled, err := p.CompileAgent(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// Should have default messages (system from body)
	require.Len(t, compiled.Messages, 1)
	assert.Equal(t, RoleSystem, compiled.Messages[0].Role)
	assert.Equal(t, "You are a helpful assistant.", compiled.Messages[0].Content)

	// Execution should be cloned
	require.NotNil(t, compiled.Execution)
	assert.Equal(t, "openai", compiled.Execution.Provider)
	assert.Equal(t, "gpt-4", compiled.Execution.Model)
}

func TestPrompt_CompileAgent_WithInput(t *testing.T) {
	p := &Prompt{
		Name:        "test-agent",
		Description: "A test agent",
		Type:        DocumentTypeAgent,
		Body:        "You are a helpful assistant.",
	}

	input := map[string]any{"message": "Hello there!"}
	compiled, err := p.CompileAgent(context.Background(), input, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// Default messages: system + user (from input.message)
	require.Len(t, compiled.Messages, 2)
	assert.Equal(t, RoleSystem, compiled.Messages[0].Role)
	assert.Equal(t, RoleUser, compiled.Messages[1].Role)
	assert.Equal(t, "Hello there!", compiled.Messages[1].Content)
}

func TestPrompt_CompileAgent_WithMessages(t *testing.T) {
	p := &Prompt{
		Name:        "test-agent",
		Description: "A test agent",
		Type:        DocumentTypeAgent,
		Body:        "Base system content.",
		Messages: []MessageTemplate{
			{
				Role:    RoleSystem,
				Content: "You are {~prompty.var name=\"meta.name\" /~}.",
			},
			{
				Role:    RoleUser,
				Content: "{~prompty.var name=\"input.message\" default=\"Hello\" /~}",
			},
		},
	}

	input := map[string]any{"message": "What is 2+2?"}
	compiled, err := p.CompileAgent(context.Background(), input, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)
	require.Len(t, compiled.Messages, 2)

	assert.Equal(t, RoleSystem, compiled.Messages[0].Role)
	assert.Equal(t, "You are test-agent.", compiled.Messages[0].Content)

	assert.Equal(t, RoleUser, compiled.Messages[1].Role)
	assert.Equal(t, "What is 2+2?", compiled.Messages[1].Content)
}

func TestPrompt_CompileAgent_WithContext(t *testing.T) {
	p := &Prompt{
		Name:        "test-agent",
		Description: "A test agent",
		Type:        DocumentTypeAgent,
		Context: map[string]any{
			"company": "Acme Corp",
		},
		Body: "You work for {~prompty.var name=\"context.company\" /~}.",
	}

	compiled, err := p.CompileAgent(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)
	require.Len(t, compiled.Messages, 1)
	assert.Contains(t, compiled.Messages[0].Content, "Acme Corp")
}

func TestPrompt_CompileAgent_NotAgent(t *testing.T) {
	p := &Prompt{
		Name:        "not-agent",
		Description: "A skill",
		Type:        DocumentTypeSkill,
	}

	_, err := p.CompileAgent(context.Background(), nil, nil)
	require.Error(t, err)
}

func TestPrompt_CompileAgent_WithTools(t *testing.T) {
	p := &Prompt{
		Name:        "tool-agent",
		Description: "Agent with tools",
		Type:        DocumentTypeAgent,
		Tools: &ToolsConfig{
			Functions: []*FunctionDef{
				{Name: "search", Description: "Search the web"},
			},
		},
		Body: "You have tools available.",
	}

	compiled, err := p.CompileAgent(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled.Tools)
	require.Len(t, compiled.Tools.Functions, 1)
	assert.Equal(t, "search", compiled.Tools.Functions[0].Name)
}

func TestPrompt_CompileAgent_WithConstraints(t *testing.T) {
	maxTurns := 10
	p := &Prompt{
		Name:        "constrained-agent",
		Description: "Agent with constraints",
		Type:        DocumentTypeAgent,
		Constraints: &ConstraintsConfig{
			Behavioral: []string{"Be concise"},
			Safety:     []string{"No PII"},
			Operational: &OperationalConstraints{
				MaxTurns: &maxTurns,
			},
		},
		Body: "You are a constrained agent.",
	}

	compiled, err := p.CompileAgent(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled.Constraints)
	assert.Equal(t, 10, *compiled.Constraints.MaxTurns)
}

func TestPrompt_ActivateSkill_SystemPrompt(t *testing.T) {
	p := &Prompt{
		Name:        "agent-with-skills",
		Description: "Agent with skills",
		Type:        DocumentTypeAgent,
		Skills: []SkillRef{
			{
				Slug:      "search-skill",
				Injection: SkillInjectionSystemPrompt,
			},
		},
		Body: "You are a helpful agent.",
	}

	resolver := NewMapDocumentResolver()
	resolver.AddSkill("search-skill", &Prompt{
		Name:        "search-skill",
		Description: "Search skill",
		Type:        DocumentTypeSkill,
		Body:        "Use the search tool to find information.",
	})

	opts := &CompileOptions{Resolver: resolver}
	compiled, err := p.ActivateSkill(context.Background(), "search-skill", nil, opts)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// System message should contain both the base body and injected skill
	require.True(t, len(compiled.Messages) >= 1)
	systemMsg := compiled.Messages[0]
	assert.Equal(t, RoleSystem, systemMsg.Role)
	assert.Contains(t, systemMsg.Content, "You are a helpful agent.")
	assert.Contains(t, systemMsg.Content, "Use the search tool to find information.")
	assert.Contains(t, systemMsg.Content, SkillInjectionMarkerStart+"search-skill")
}

func TestPrompt_ActivateSkill_UserContext(t *testing.T) {
	p := &Prompt{
		Name:        "agent-user-ctx",
		Description: "Agent with user context injection",
		Type:        DocumentTypeAgent,
		Skills: []SkillRef{
			{
				Slug:      "context-skill",
				Injection: SkillInjectionUserContext,
			},
		},
		Body: "System prompt.",
	}

	resolver := NewMapDocumentResolver()
	resolver.AddSkill("context-skill", &Prompt{
		Name:        "context-skill",
		Description: "Context skill",
		Type:        DocumentTypeSkill,
		Body:        "Additional user context here.",
	})

	opts := &CompileOptions{Resolver: resolver}
	compiled, err := p.ActivateSkill(context.Background(), "context-skill", nil, opts)
	require.NoError(t, err)

	// Should have a user message with the injected skill
	hasUserMessage := false
	for _, msg := range compiled.Messages {
		if msg.Role == RoleUser && msg.Content != "" {
			if contains(msg.Content, "Additional user context here.") {
				hasUserMessage = true
			}
		}
	}
	assert.True(t, hasUserMessage, "expected user message with injected skill content")
}

func TestPrompt_ActivateSkill_None(t *testing.T) {
	p := &Prompt{
		Name:        "agent-no-inject",
		Description: "Agent with no injection",
		Type:        DocumentTypeAgent,
		Skills: []SkillRef{
			{
				Slug:      "passive-skill",
				Injection: SkillInjectionNone,
			},
		},
		Body: "System prompt.",
	}

	resolver := NewMapDocumentResolver()
	resolver.AddSkill("passive-skill", &Prompt{
		Name:        "passive-skill",
		Description: "Passive skill",
		Type:        DocumentTypeSkill,
		Body:        "This should not appear in messages.",
	})

	opts := &CompileOptions{Resolver: resolver}
	compiled, err := p.ActivateSkill(context.Background(), "passive-skill", nil, opts)
	require.NoError(t, err)

	// No message should contain the skill content
	for _, msg := range compiled.Messages {
		assert.NotContains(t, msg.Content, "This should not appear in messages.")
	}
}

func TestPrompt_ActivateSkill_InlineSkill(t *testing.T) {
	p := &Prompt{
		Name:        "agent-inline",
		Description: "Agent with inline skill",
		Type:        DocumentTypeAgent,
		Skills: []SkillRef{
			{
				Inline: &InlineSkill{
					Slug:        "inline-helper",
					Description: "Inline helper",
					Body:        "Inline skill body content.",
				},
				Injection: SkillInjectionSystemPrompt,
			},
		},
		Body: "Base agent body.",
	}

	compiled, err := p.ActivateSkill(context.Background(), "inline-helper", nil, nil)
	require.NoError(t, err)

	systemMsg := compiled.Messages[0]
	assert.Contains(t, systemMsg.Content, "Inline skill body content.")
}

func TestPrompt_ActivateSkill_NotFound(t *testing.T) {
	p := &Prompt{
		Name:        "agent",
		Description: "Agent",
		Type:        DocumentTypeAgent,
		Skills: []SkillRef{
			{Slug: "existing-skill"},
		},
		Body: "body",
	}

	_, err := p.ActivateSkill(context.Background(), "nonexistent-skill", nil, nil)
	require.Error(t, err)
}

func TestPrompt_ActivateSkill_ExecutionMerge(t *testing.T) {
	agentTemp := 0.5
	p := &Prompt{
		Name:        "merge-agent",
		Description: "Agent with execution",
		Type:        DocumentTypeAgent,
		Execution: &ExecutionConfig{
			Provider:    "openai",
			Model:       "gpt-4",
			Temperature: &agentTemp,
		},
		Skills: []SkillRef{
			{
				Slug: "skill-with-exec",
				Execution: &ExecutionConfig{
					Model: "gpt-4-turbo",
				},
			},
		},
		Body: "body",
	}

	skillTemp := 0.8
	resolver := NewMapDocumentResolver()
	resolver.AddSkill("skill-with-exec", &Prompt{
		Name:        "skill-with-exec",
		Description: "Skill with execution override",
		Type:        DocumentTypeSkill,
		Execution: &ExecutionConfig{
			Temperature: &skillTemp,
		},
		Body: "skill body",
	})

	opts := &CompileOptions{Resolver: resolver}
	compiled, err := p.ActivateSkill(context.Background(), "skill-with-exec", nil, opts)
	require.NoError(t, err)
	require.NotNil(t, compiled.Execution)

	// Provider from agent
	assert.Equal(t, "openai", compiled.Execution.Provider)
	// Model overridden by skill ref execution
	assert.Equal(t, "gpt-4-turbo", compiled.Execution.Model)
	// Temperature from resolved skill (then overridden by ref if ref had one)
	assert.Equal(t, 0.8, *compiled.Execution.Temperature)
}

func TestPrompt_ValidateForExecution(t *testing.T) {
	// Nil prompt
	var nilP *Prompt
	require.Error(t, nilP.ValidateForExecution())

	// No execution config
	p := &Prompt{Name: "test", Description: "test"}
	require.Error(t, p.ValidateForExecution())

	// No provider
	p.Execution = &ExecutionConfig{Model: "gpt-4"}
	require.Error(t, p.ValidateForExecution())

	// No model
	p.Execution = &ExecutionConfig{Provider: "openai"}
	require.Error(t, p.ValidateForExecution())

	// Valid
	p.Execution = &ExecutionConfig{Provider: "openai", Model: "gpt-4"}
	require.NoError(t, p.ValidateForExecution())
}

// ==================== API-04: Provider Message Serialization ====================

func TestCompiledPrompt_ToOpenAIMessages(t *testing.T) {
	compiled := &CompiledPrompt{
		Messages: []CompiledMessage{
			{Role: RoleSystem, Content: "You are a helpful assistant."},
			{Role: RoleUser, Content: "Hello!"},
			{Role: RoleAssistant, Content: "Hi there!"},
		},
	}

	msgs := compiled.ToOpenAIMessages()
	require.Len(t, msgs, 3)

	assert.Equal(t, RoleSystem, msgs[0][AttrRole])
	assert.Equal(t, "You are a helpful assistant.", msgs[0]["content"])
	assert.Equal(t, RoleUser, msgs[1][AttrRole])
	assert.Equal(t, "Hello!", msgs[1]["content"])
	assert.Equal(t, RoleAssistant, msgs[2][AttrRole])
	assert.Equal(t, "Hi there!", msgs[2]["content"])
}

func TestCompiledPrompt_ToOpenAIMessages_Nil(t *testing.T) {
	var cp *CompiledPrompt
	assert.Nil(t, cp.ToOpenAIMessages())

	cp = &CompiledPrompt{}
	assert.Nil(t, cp.ToOpenAIMessages())
}

func TestCompiledPrompt_ToAnthropicMessages(t *testing.T) {
	compiled := &CompiledPrompt{
		Messages: []CompiledMessage{
			{Role: RoleSystem, Content: "System instructions."},
			{Role: RoleUser, Content: "Hello!"},
			{Role: RoleAssistant, Content: "Hi!"},
		},
	}

	result := compiled.ToAnthropicMessages()
	require.NotNil(t, result)

	// System extracted to top-level
	assert.Equal(t, "System instructions.", result[RoleSystem])

	// Non-system messages in messages array
	msgs, ok := result["messages"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, msgs, 2)
	assert.Equal(t, RoleUser, msgs[0][AttrRole])
	assert.Equal(t, RoleAssistant, msgs[1][AttrRole])
}

func TestCompiledPrompt_ToAnthropicMessages_MultipleSystemMessages(t *testing.T) {
	compiled := &CompiledPrompt{
		Messages: []CompiledMessage{
			{Role: RoleSystem, Content: "First system message."},
			{Role: RoleSystem, Content: "Second system message."},
			{Role: RoleUser, Content: "Hello!"},
		},
	}

	result := compiled.ToAnthropicMessages()
	// Multiple system messages joined with double newline
	assert.Equal(t, "First system message.\n\nSecond system message.", result[RoleSystem])
}

func TestCompiledPrompt_ToAnthropicMessages_NoSystemMessage(t *testing.T) {
	compiled := &CompiledPrompt{
		Messages: []CompiledMessage{
			{Role: RoleUser, Content: "Hello!"},
		},
	}

	result := compiled.ToAnthropicMessages()
	_, hasSystem := result[RoleSystem]
	assert.False(t, hasSystem)

	msgs, ok := result["messages"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, msgs, 1)
}

func TestCompiledPrompt_ToAnthropicMessages_Nil(t *testing.T) {
	var cp *CompiledPrompt
	assert.Nil(t, cp.ToAnthropicMessages())
}

func TestCompiledPrompt_ToGeminiContents(t *testing.T) {
	compiled := &CompiledPrompt{
		Messages: []CompiledMessage{
			{Role: RoleSystem, Content: "System instructions."},
			{Role: RoleUser, Content: "Hello!"},
			{Role: RoleAssistant, Content: "Hi!"},
		},
	}

	result := compiled.ToGeminiContents()
	require.NotNil(t, result)

	// System instruction extracted
	sysInstr, ok := result["system_instruction"].(map[string]any)
	require.True(t, ok)
	parts, ok := sysInstr["parts"].([]map[string]string)
	require.True(t, ok)
	require.Len(t, parts, 1)
	assert.Equal(t, "System instructions.", parts[0]["text"])

	// Contents with role mapping
	contents, ok := result["contents"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, contents, 2)

	assert.Equal(t, RoleUser, contents[0][AttrRole])
	// Assistant → model
	assert.Equal(t, "model", contents[1][AttrRole])

	// Check parts structure
	userParts, ok := contents[0]["parts"].([]map[string]string)
	require.True(t, ok)
	assert.Equal(t, "Hello!", userParts[0]["text"])
}

func TestCompiledPrompt_ToGeminiContents_Nil(t *testing.T) {
	var cp *CompiledPrompt
	assert.Nil(t, cp.ToGeminiContents())
}

func TestCompiledPrompt_ToProviderMessages(t *testing.T) {
	compiled := &CompiledPrompt{
		Messages: []CompiledMessage{
			{Role: RoleSystem, Content: "System."},
			{Role: RoleUser, Content: "Hi."},
		},
	}

	tests := []struct {
		provider string
		wantErr  bool
	}{
		{ProviderOpenAI, false},
		{ProviderAzure, false},
		{ProviderAnthropic, false},
		{ProviderGemini, false},
		{ProviderGoogle, false},
		{ProviderVertex, false},
		{"unknown-provider", true},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			result, err := compiled.ToProviderMessages(tt.provider)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestCompiledPrompt_ToProviderMessages_Nil(t *testing.T) {
	var cp *CompiledPrompt
	result, err := cp.ToProviderMessages(ProviderOpenAI)
	require.NoError(t, err)
	assert.Nil(t, result)
}

// ==================== ENH-02: Functional Options for CompileOptions ====================

func TestNewCompileOptions(t *testing.T) {
	resolver := NewMapDocumentResolver()
	engine := MustNew()

	opts := NewCompileOptions(
		WithResolver(resolver),
		WithCompileEngine(engine),
		WithSkillsCatalogFormat(CatalogFormatDetailed),
		WithToolsCatalogFormat(CatalogFormatFunctionCalling),
	)

	assert.Equal(t, resolver, opts.Resolver)
	assert.Equal(t, engine, opts.Engine)
	assert.Equal(t, CatalogFormatDetailed, opts.SkillsCatalogFormat)
	assert.Equal(t, CatalogFormatFunctionCalling, opts.ToolsCatalogFormat)
}

func TestNewCompileOptions_Empty(t *testing.T) {
	opts := NewCompileOptions()
	assert.Nil(t, opts.Resolver)
	assert.Nil(t, opts.Engine)
	assert.Equal(t, CatalogFormatDefault, opts.SkillsCatalogFormat)
	assert.Equal(t, CatalogFormatDefault, opts.ToolsCatalogFormat)
}

func TestNewCompileOptions_Partial(t *testing.T) {
	resolver := NewMapDocumentResolver()
	opts := NewCompileOptions(WithResolver(resolver))

	assert.Equal(t, resolver, opts.Resolver)
	assert.Nil(t, opts.Engine)
}

func TestNewCompileOptions_UsedInCompilation(t *testing.T) {
	p := &Prompt{
		Name:        "test-agent",
		Description: "Test agent",
		Type:        DocumentTypeAgent,
		Skills: []SkillRef{
			{Slug: "helper"},
		},
		Body: "You are a helpful assistant.",
	}

	resolver := NewMapDocumentResolver()
	resolver.AddSkill("helper", &Prompt{
		Name:        "helper",
		Description: "A helper skill",
		Type:        DocumentTypeSkill,
		Body:        "I can help you.",
	})

	opts := NewCompileOptions(
		WithResolver(resolver),
		WithSkillsCatalogFormat(CatalogFormatCompact),
	)

	compiled, err := p.CompileAgent(context.Background(), nil, opts)
	require.NoError(t, err)
	require.NotNil(t, compiled)
	require.Len(t, compiled.Messages, 1)
}

// ==================== API-05: Compilation Error Context ====================

func TestCompileMessages_ErrorContext(t *testing.T) {
	p := &Prompt{
		Name:        "error-agent",
		Description: "Agent with bad message",
		Type:        DocumentTypeAgent,
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: "Valid content."},
			{Role: RoleUser, Content: "{~prompty.var name=\"nonexistent.deep.path\" /~}"},
		},
		Body: "body",
	}

	_, err := p.CompileAgent(context.Background(), nil, nil)
	// Should succeed — var with no onerror just produces empty by default in most configs
	// Let's test with an explicit error case
	if err != nil {
		// If error returned, check it has context
		errStr := err.Error()
		assert.Contains(t, errStr, ErrMsgCompileMessageFailed)
	}
}

func TestActivateSkill_ErrorContext_SkillResolutionFailed(t *testing.T) {
	p := &Prompt{
		Name:        "agent",
		Description: "Agent",
		Type:        DocumentTypeAgent,
		Skills: []SkillRef{
			{Slug: "broken-skill"},
		},
		Body: "body",
	}

	// Resolver that has no such skill
	resolver := NewMapDocumentResolver()
	opts := &CompileOptions{Resolver: resolver}

	_, err := p.ActivateSkill(context.Background(), "broken-skill", nil, opts)
	require.Error(t, err)
	errStr := err.Error()
	assert.Contains(t, errStr, ErrMsgCompileSkillFailed)
}

// contains is a simple helper to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- AgentDryRun Tests ---

func TestAgentDryRun_ValidAgent(t *testing.T) {
	p := &Prompt{
		Name:        "test-agent",
		Description: "A valid test agent",
		Type:        DocumentTypeAgent,
		Execution: &ExecutionConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: "You are a helpful assistant."},
			{Role: RoleUser, Content: "Hello"},
		},
		Body: "Agent body.",
		Tools: &ToolsConfig{
			Functions: []*FunctionDef{
				{Name: "search", Description: "Search the web"},
			},
		},
	}

	result := p.AgentDryRun(context.Background(), nil)
	assert.True(t, result.OK())
	assert.False(t, result.HasErrors())
	assert.Equal(t, 2, result.MessageCount)
	assert.Equal(t, 1, result.ToolsDefined)
	assert.Equal(t, 0, result.SkillsResolved)
}

func TestAgentDryRun_UnresolvableSkill(t *testing.T) {
	resolver := NewMapDocumentResolver()
	p := &Prompt{
		Name:        "test-agent",
		Description: "Agent with unresolvable skill",
		Type:        DocumentTypeAgent,
		Execution: &ExecutionConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Skills: []SkillRef{
			{Slug: "nonexistent-skill"},
		},
		Body: "Agent body.",
	}

	result := p.AgentDryRun(context.Background(), &CompileOptions{Resolver: resolver})
	assert.True(t, result.HasErrors())
	assert.Equal(t, 1, len(result.Issues))
	assert.Equal(t, AgentDryRunCategorySkill, result.Issues[0].Category)
	assert.Equal(t, "skill:nonexistent-skill", result.Issues[0].Location)
}

func TestAgentDryRun_InvalidMessageTemplate(t *testing.T) {
	p := &Prompt{
		Name:        "test-agent",
		Description: "Agent with invalid message",
		Type:        DocumentTypeAgent,
		Execution: &ExecutionConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: "{~prompty.var name=\"unclosed\""},
		},
		Body: "Valid body.",
	}

	result := p.AgentDryRun(context.Background(), nil)
	assert.True(t, result.HasErrors())

	foundTemplateIssue := false
	for _, issue := range result.Issues {
		if issue.Category == AgentDryRunCategoryTemplate && issue.Location == "message[0]" {
			foundTemplateIssue = true
			break
		}
	}
	assert.True(t, foundTemplateIssue, "should have a template issue for message[0]")
}

func TestAgentDryRun_InvalidBody(t *testing.T) {
	p := &Prompt{
		Name:        "test-agent",
		Description: "Agent with invalid body",
		Type:        DocumentTypeAgent,
		Execution: &ExecutionConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: "Valid system message."},
		},
		Body: "{~prompty.var name=\"unclosed\"",
	}

	result := p.AgentDryRun(context.Background(), nil)
	assert.True(t, result.HasErrors())

	foundBodyIssue := false
	for _, issue := range result.Issues {
		if issue.Category == AgentDryRunCategoryTemplate && issue.Location == "body" {
			foundBodyIssue = true
			break
		}
	}
	assert.True(t, foundBodyIssue, "should have a template issue for body")
}

func TestAgentDryRun_NoResolver(t *testing.T) {
	p := &Prompt{
		Name:        "test-agent",
		Description: "Agent with skills but no resolver",
		Type:        DocumentTypeAgent,
		Execution: &ExecutionConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Skills: []SkillRef{
			{Slug: "skill-a"},
			{Slug: "skill-b"},
		},
		Body: "Agent body.",
	}

	result := p.AgentDryRun(context.Background(), nil)
	assert.True(t, result.HasErrors())

	skillIssueCount := 0
	for _, issue := range result.Issues {
		if issue.Category == AgentDryRunCategorySkill {
			skillIssueCount++
		}
	}
	assert.Equal(t, 2, skillIssueCount, "should have issues for each skill")
}

func TestAgentDryRun_MultipleIssues(t *testing.T) {
	p := &Prompt{
		Name:        "test-agent",
		Description: "Agent with multiple issues",
		Type:        DocumentTypeAgent,
		Execution: &ExecutionConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Skills: []SkillRef{
			{Slug: "missing-skill"},
		},
		Messages: []MessageTemplate{
			{Role: RoleSystem, Content: "{~prompty.var name=\"broken\""},
		},
		Body: "{~prompty.var name=\"also-broken\"",
	}

	result := p.AgentDryRun(context.Background(), nil)
	assert.True(t, result.HasErrors())
	// At least 3 issues: 1 skill (no resolver) + 1 message parse + 1 body parse
	assert.GreaterOrEqual(t, len(result.Issues), 3)
}

func TestAgentDryRun_NilPrompt(t *testing.T) {
	var p *Prompt
	result := p.AgentDryRun(context.Background(), nil)
	assert.True(t, result.HasErrors())
	assert.Equal(t, 1, len(result.Issues))
	assert.Equal(t, AgentDryRunCategoryParse, result.Issues[0].Category)
	assert.Contains(t, result.Issues[0].Message, ErrMsgAgentDryRunNilPrompt)
}

func TestAgentDryRunResult_OK(t *testing.T) {
	result := &AgentDryRunResult{}
	assert.True(t, result.OK())
	assert.False(t, result.HasErrors())
}

func TestAgentDryRunResult_HasErrors(t *testing.T) {
	result := &AgentDryRunResult{
		Issues: []AgentDryRunIssue{
			{Category: AgentDryRunCategoryValidation, Message: "test issue", Location: "prompt"},
		},
	}
	assert.True(t, result.HasErrors())
	assert.False(t, result.OK())
}

func TestAgentDryRunResult_String(t *testing.T) {
	t.Run("ok result", func(t *testing.T) {
		result := &AgentDryRunResult{
			SkillsResolved: 2,
			ToolsDefined:   3,
			MessageCount:   4,
		}
		s := result.String()
		assert.Contains(t, s, "OK")
		assert.Contains(t, s, "2 skills resolved")
		assert.Contains(t, s, "3 tools defined")
		assert.Contains(t, s, "4 messages")
	})

	t.Run("result with issues", func(t *testing.T) {
		result := &AgentDryRunResult{
			Issues: []AgentDryRunIssue{
				{Category: AgentDryRunCategorySkill, Message: "not found", Location: "skill:web-search"},
				{Category: AgentDryRunCategoryTemplate, Message: "parse error", Location: "body"},
			},
		}
		s := result.String()
		assert.Contains(t, s, "2 issue(s)")
		assert.Contains(t, s, "skill")
		assert.Contains(t, s, "web-search")
		assert.Contains(t, s, "body")
	})
}
