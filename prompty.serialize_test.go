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
			Provider:    ProviderOpenAI,
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
	assert.Contains(t, content, ProviderOpenAI)
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
			Provider: ProviderOpenAI,
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
			Provider: ProviderAnthropic,
		},
		Extensions: map[string]any{
			"custom_platform": map[string]any{
				"visibility": "public",
			},
		},
		Body: "template body",
	}

	data, err := p.ExportFull()
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "full-prompt")
	assert.Contains(t, content, ProviderAnthropic)
	assert.Contains(t, content, "public")
}

func TestPrompt_Serialize_Nil(t *testing.T) {
	var p *Prompt
	data, err := p.Serialize(nil)
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestPrompt_Serialize_ExtensionKeyConflict(t *testing.T) {
	// Extension keys that match known Prompt fields should be skipped
	// during serialization to prevent overwriting struct field values.
	p := &Prompt{
		Name:        "original-name",
		Description: "original-description",
		Extensions: map[string]any{
			"name":        "override-name",
			"description": "override-description",
			"inputs":      "override-inputs",
			"custom_key":  "custom-value",
		},
		Body: "body",
	}

	data, err := p.Serialize(nil)
	require.NoError(t, err)

	content := string(data)
	// Struct field values must win over conflicting extension keys
	assert.Contains(t, content, "original-name")
	assert.NotContains(t, content, "override-name")
	assert.Contains(t, content, "original-description")
	assert.NotContains(t, content, "override-description")
	assert.NotContains(t, content, "override-inputs")
	// Non-conflicting extension keys should still appear
	assert.Contains(t, content, "custom_key")
	assert.Contains(t, content, "custom-value")
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
