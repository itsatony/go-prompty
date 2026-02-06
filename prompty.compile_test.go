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
