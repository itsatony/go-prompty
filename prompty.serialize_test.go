package prompty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrompt_Serialize_Default(t *testing.T) {
	temp := 0.7
	p := &Prompt{
		Name:        "test-prompt",
		Description: "A test prompt",
		Type:        DocumentTypeAgent,
		Execution: &ExecutionConfig{
			Provider:    "openai",
			Model:       "gpt-4",
			Temperature: &temp,
		},
		Skills: []SkillRef{
			{Slug: "search-skill", Injection: SkillInjectionSystemPrompt},
		},
		Tools: &ToolsConfig{
			Functions: []*FunctionDef{
				{Name: "search", Description: "Search the web"},
			},
		},
		Context: map[string]any{
			"company": "Acme",
		},
		Body: "You are a helpful assistant.",
	}

	data, err := p.Serialize(nil)
	require.NoError(t, err)
	require.NotNil(t, data)

	content := string(data)
	assert.Contains(t, content, "---")
	assert.Contains(t, content, "test-prompt")
	assert.Contains(t, content, "agent")
	assert.Contains(t, content, "openai")
	assert.Contains(t, content, "search-skill")
	assert.Contains(t, content, "Acme")
	assert.Contains(t, content, "You are a helpful assistant.")
}

func TestPrompt_Serialize_AgentSkillsExport(t *testing.T) {
	p := &Prompt{
		Name:        "test-prompt",
		Description: "A test prompt",
		Type:        DocumentTypeAgent,
		Execution: &ExecutionConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Skills: []SkillRef{
			{Slug: "skill-a"},
		},
		Body: "body content",
	}

	data, err := p.ExportAgentSkill()
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "test-prompt")
	// Should NOT contain execution or agent fields
	assert.NotContains(t, content, "openai")
	assert.NotContains(t, content, "skill-a")
}

func TestPrompt_Serialize_Full(t *testing.T) {
	p := &Prompt{
		Name:        "full-prompt",
		Description: "Full prompt with all fields",
		Type:        DocumentTypeSkill,
		Execution: &ExecutionConfig{
			Provider: "anthropic",
		},
		Skope: &SkopeConfig{
			Visibility: SkopeVisibilityPublic,
		},
		Body: "template body",
	}

	data, err := p.ExportFull()
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "full-prompt")
	assert.Contains(t, content, "anthropic")
	assert.Contains(t, content, "public")
}

func TestPrompt_Serialize_Nil(t *testing.T) {
	var p *Prompt
	data, err := p.Serialize(nil)
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestPrompt_Serialize_RoundTrip(t *testing.T) {
	original := &Prompt{
		Name:        "roundtrip-test",
		Description: "Tests serialization round-trip",
		Type:        DocumentTypeSkill,
		Body:        "Hello {~prompty.var name=\"name\" /~}!",
	}

	// Serialize
	data, err := original.Serialize(nil)
	require.NoError(t, err)

	// Parse back
	parsed, err := Parse(data)
	require.NoError(t, err)

	assert.Equal(t, original.Name, parsed.Name)
	assert.Equal(t, original.Description, parsed.Description)
	assert.Equal(t, original.EffectiveType(), parsed.EffectiveType())
	assert.Equal(t, original.Body, parsed.Body)
}
