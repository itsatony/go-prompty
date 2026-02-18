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

func TestPrompt_Serialize_CredentialsExcludedByDefault(t *testing.T) {
	p := &Prompt{
		Name:        "cred-test",
		Description: "Credential serialization test",
		Credential:  "main",
		Credentials: map[string]*CredentialRef{
			"main": {Provider: ProviderAnthropic, Ref: "${ANTHROPIC_KEY}"},
		},
		Body: "body",
	}

	// Default options should NOT include credentials
	data, err := p.Serialize(nil)
	require.NoError(t, err)
	content := string(data)
	assert.NotContains(t, content, "credentials")
	assert.NotContains(t, content, "${ANTHROPIC_KEY}")
}

func TestPrompt_Serialize_CredentialsWithExplicitFlag(t *testing.T) {
	p := &Prompt{
		Name:        "cred-test",
		Description: "Credential serialization test",
		Credential:  "main",
		Credentials: map[string]*CredentialRef{
			"main": {Provider: ProviderAnthropic, Ref: "${ANTHROPIC_KEY}"},
		},
		Body: "body",
	}

	// FullExportWithCredentials should include credentials
	data, err := p.Serialize(FullExportWithCredentials())
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "credentials")
	assert.Contains(t, content, "${ANTHROPIC_KEY}")
	assert.Contains(t, content, ProviderAnthropic)
}

func TestPrompt_Serialize_RequirementsWithAgentFields(t *testing.T) {
	p := &Prompt{
		Name:        "req-test",
		Description: "Requirements serialization test",
		Type:        DocumentTypeAgent,
		Requirements: &ExecutionRequirements{
			Modality:        ModalityImage,
			ProviderBinding: ProviderBindingRequired,
		},
		Body: "body",
	}

	// Default options include agent fields → requirements should appear
	data, err := p.Serialize(nil)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "requirements")
	assert.Contains(t, content, ProviderBindingRequired)
}

func TestFullExportWithCredentials(t *testing.T) {
	opts := FullExportWithCredentials()
	assert.True(t, opts.IncludeExecution)
	assert.True(t, opts.IncludeExtensions)
	assert.True(t, opts.IncludeAgentFields)
	assert.True(t, opts.IncludeContext)
	assert.True(t, opts.IncludeCredentials)
}

func TestPrompt_Serialize_RequirementsWithCredentialsFlag(t *testing.T) {
	// Requirements should also appear when IncludeCredentials is true
	// (even if IncludeAgentFields is false)
	p := &Prompt{
		Name:        "req-cred-test",
		Description: "Requirements via credentials flag",
		Requirements: &ExecutionRequirements{
			Modality:        ModalityEmbedding,
			ProviderBinding: ProviderBindingPreferred,
		},
		Body: "body",
	}

	opts := &SerializeOptions{
		IncludeExecution:   false,
		IncludeExtensions:  false,
		IncludeAgentFields: false,
		IncludeContext:     false,
		IncludeCredentials: true,
	}
	data, err := p.Serialize(opts)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "requirements")
	assert.Contains(t, content, ProviderBindingPreferred)
}

func TestPrompt_Serialize_CredentialExtensionKeyConflict(t *testing.T) {
	// Extension keys matching credential field names should be skipped
	p := &Prompt{
		Name:        "conflict-test",
		Description: "Extension key conflict with credentials",
		Extensions: map[string]any{
			PromptFieldCredentials:  "override-creds",
			PromptFieldCredential:   "override-cred",
			PromptFieldRequirements: "override-reqs",
		},
		Body: "body",
	}

	data, err := p.Serialize(nil)
	require.NoError(t, err)
	content := string(data)
	// Extension keys matching known fields should be skipped
	assert.NotContains(t, content, "override-creds")
	assert.NotContains(t, content, "override-cred")
	assert.NotContains(t, content, "override-reqs")
}
