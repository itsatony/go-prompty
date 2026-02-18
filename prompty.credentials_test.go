package prompty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- CredentialRef tests ---

func TestCredentialRef_Validate(t *testing.T) {
	t.Run("nil is valid", func(t *testing.T) {
		var c *CredentialRef
		assert.NoError(t, c.Validate())
	})

	t.Run("valid with provider", func(t *testing.T) {
		c := &CredentialRef{Provider: ProviderOpenAI}
		assert.NoError(t, c.Validate())
	})

	t.Run("valid with all fields", func(t *testing.T) {
		c := &CredentialRef{
			Provider: ProviderAnthropic,
			Label:    "main-key",
			Ref:      "${ANTHROPIC_API_KEY}",
			Scopes:   []string{"chat", "embeddings"},
		}
		assert.NoError(t, c.Validate())
	})

	t.Run("missing provider", func(t *testing.T) {
		c := &CredentialRef{Label: "test"}
		err := c.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgCredentialMissingProvider)
	})
}

func TestCredentialRef_Clone(t *testing.T) {
	t.Run("nil clone", func(t *testing.T) {
		var c *CredentialRef
		assert.Nil(t, c.Clone())
	})

	t.Run("deep copy", func(t *testing.T) {
		c := &CredentialRef{
			Provider: ProviderOpenAI,
			Label:    "images",
			Ref:      "${KEY}",
			Scopes:   []string{"images", "audio"},
		}
		clone := c.Clone()
		require.NotNil(t, clone)
		assert.Equal(t, c.Provider, clone.Provider)
		assert.Equal(t, c.Label, clone.Label)
		assert.Equal(t, c.Ref, clone.Ref)
		assert.Equal(t, c.Scopes, clone.Scopes)

		// Verify isolation
		clone.Scopes[0] = "modified"
		assert.Equal(t, "images", c.Scopes[0])
	})
}

func TestCredentialRef_Getters(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var c *CredentialRef
		assert.Equal(t, "", c.GetProvider())
		assert.Equal(t, "", c.GetLabel())
		assert.Equal(t, "", c.GetRef())
		assert.Nil(t, c.GetScopes())
		assert.False(t, c.HasScopes())
	})

	t.Run("populated", func(t *testing.T) {
		c := &CredentialRef{
			Provider: ProviderMistral,
			Label:    "mistral-key",
			Ref:      "vault://secrets/mistral",
			Scopes:   []string{"chat"},
		}
		assert.Equal(t, ProviderMistral, c.GetProvider())
		assert.Equal(t, "mistral-key", c.GetLabel())
		assert.Equal(t, "vault://secrets/mistral", c.GetRef())
		assert.Equal(t, []string{"chat"}, c.GetScopes())
		assert.True(t, c.HasScopes())
	})
}

// --- Prompt credential methods tests ---

func TestPrompt_ValidateCredentialRefs(t *testing.T) {
	t.Run("nil prompt", func(t *testing.T) {
		var p *Prompt
		assert.NoError(t, p.ValidateCredentialRefs())
	})

	t.Run("no credentials", func(t *testing.T) {
		p := &Prompt{Name: "test", Description: "test"}
		assert.NoError(t, p.ValidateCredentialRefs())
	})

	t.Run("valid credentials map", func(t *testing.T) {
		p := &Prompt{
			Name:        "test",
			Description: "test",
			Credentials: map[string]*CredentialRef{
				"main": {Provider: ProviderOpenAI, Ref: "${KEY}"},
			},
		}
		assert.NoError(t, p.ValidateCredentialRefs())
	})

	t.Run("credential missing provider", func(t *testing.T) {
		p := &Prompt{
			Name:        "test",
			Description: "test",
			Credentials: map[string]*CredentialRef{
				"bad": {Label: "no-provider"},
			},
		}
		err := p.ValidateCredentialRefs()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgCredentialMissingProvider)
	})

	t.Run("nil credential in map", func(t *testing.T) {
		p := &Prompt{
			Name:        "test",
			Description: "test",
			Credentials: map[string]*CredentialRef{
				"nil-entry": nil,
			},
		}
		err := p.ValidateCredentialRefs()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgCredentialMissingProvider)
	})

	t.Run("default credential label found", func(t *testing.T) {
		p := &Prompt{
			Name:        "test",
			Description: "test",
			Credential:  "main",
			Credentials: map[string]*CredentialRef{
				"main": {Provider: ProviderAnthropic},
			},
		}
		assert.NoError(t, p.ValidateCredentialRefs())
	})

	t.Run("default credential label not found", func(t *testing.T) {
		p := &Prompt{
			Name:        "test",
			Description: "test",
			Credential:  "missing",
			Credentials: map[string]*CredentialRef{
				"main": {Provider: ProviderAnthropic},
			},
		}
		err := p.ValidateCredentialRefs()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgCredentialNotFound)
	})

	t.Run("skill credential label found", func(t *testing.T) {
		p := &Prompt{
			Name:        "test",
			Description: "test",
			Credentials: map[string]*CredentialRef{
				"main":   {Provider: ProviderAnthropic},
				"images": {Provider: ProviderOpenAI},
			},
			Skills: []SkillRef{
				{Slug: "img-gen", Credential: "images"},
			},
		}
		assert.NoError(t, p.ValidateCredentialRefs())
	})

	t.Run("skill credential label not found", func(t *testing.T) {
		p := &Prompt{
			Name:        "test",
			Description: "test",
			Credentials: map[string]*CredentialRef{
				"main": {Provider: ProviderAnthropic},
			},
			Skills: []SkillRef{
				{Slug: "img-gen", Credential: "missing-images"},
			},
		}
		err := p.ValidateCredentialRefs()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgCredentialNotFound)
	})

	t.Run("skill without credential is valid", func(t *testing.T) {
		p := &Prompt{
			Name:        "test",
			Description: "test",
			Credentials: map[string]*CredentialRef{
				"main": {Provider: ProviderAnthropic},
			},
			Skills: []SkillRef{
				{Slug: "no-cred-skill"},
			},
		}
		assert.NoError(t, p.ValidateCredentialRefs())
	})
}

func TestPrompt_GetCredentialRef(t *testing.T) {
	t.Run("nil prompt", func(t *testing.T) {
		var p *Prompt
		assert.Nil(t, p.GetCredentialRef("test"))
	})

	t.Run("found", func(t *testing.T) {
		cred := &CredentialRef{Provider: ProviderOpenAI}
		p := &Prompt{
			Credentials: map[string]*CredentialRef{"openai": cred},
		}
		assert.Equal(t, cred, p.GetCredentialRef("openai"))
	})

	t.Run("not found", func(t *testing.T) {
		p := &Prompt{
			Credentials: map[string]*CredentialRef{"openai": {Provider: ProviderOpenAI}},
		}
		assert.Nil(t, p.GetCredentialRef("missing"))
	})
}

func TestPrompt_HasCredentialRef(t *testing.T) {
	t.Run("nil prompt", func(t *testing.T) {
		var p *Prompt
		assert.False(t, p.HasCredentialRef("test"))
	})

	t.Run("exists", func(t *testing.T) {
		p := &Prompt{
			Credentials: map[string]*CredentialRef{"x": {Provider: ProviderOpenAI}},
		}
		assert.True(t, p.HasCredentialRef("x"))
	})

	t.Run("not exists", func(t *testing.T) {
		p := &Prompt{
			Credentials: map[string]*CredentialRef{"x": {Provider: ProviderOpenAI}},
		}
		assert.False(t, p.HasCredentialRef("y"))
	})
}

func TestPrompt_HasCredentials(t *testing.T) {
	var nilP *Prompt
	assert.False(t, nilP.HasCredentials())
	assert.False(t, (&Prompt{}).HasCredentials())
	assert.True(t, (&Prompt{
		Credentials: map[string]*CredentialRef{"x": {Provider: ProviderOpenAI}},
	}).HasCredentials())
}

func TestPrompt_HasRequirements(t *testing.T) {
	var nilP *Prompt
	assert.False(t, nilP.HasRequirements())
	assert.False(t, (&Prompt{}).HasRequirements())
	assert.True(t, (&Prompt{Requirements: &ExecutionRequirements{}}).HasRequirements())
}

func TestPrompt_GetRequirements(t *testing.T) {
	var nilP *Prompt
	assert.Nil(t, nilP.GetRequirements())

	req := &ExecutionRequirements{Modality: ModalityImage}
	p := &Prompt{Requirements: req}
	assert.Equal(t, req, p.GetRequirements())
}

func TestPrompt_Clone_WithCredentials(t *testing.T) {
	p := &Prompt{
		Name:        "test",
		Description: "test desc",
		Credential:  "main",
		Credentials: map[string]*CredentialRef{
			"main":   {Provider: ProviderAnthropic, Scopes: []string{"chat"}},
			"images": {Provider: ProviderOpenAI, Ref: "${KEY}"},
		},
		Requirements: &ExecutionRequirements{
			Modality:        ModalityText,
			ProviderBinding: ProviderBindingRequired,
			Capabilities:    []string{"function_calling"},
		},
	}

	clone := p.Clone()
	require.NotNil(t, clone)

	// Verify values
	assert.Equal(t, "main", clone.Credential)
	assert.Len(t, clone.Credentials, 2)
	assert.Equal(t, ProviderAnthropic, clone.Credentials["main"].Provider)
	assert.Equal(t, ProviderOpenAI, clone.Credentials["images"].Provider)
	assert.NotNil(t, clone.Requirements)
	assert.Equal(t, ModalityText, clone.Requirements.Modality)

	// Verify isolation
	clone.Credentials["main"].Scopes[0] = "modified"
	assert.Equal(t, "chat", p.Credentials["main"].Scopes[0])

	clone.Requirements.Capabilities[0] = "modified"
	assert.Equal(t, "function_calling", p.Requirements.Capabilities[0])
}

func TestPrompt_Validate_WithCredentials(t *testing.T) {
	t.Run("valid agent with credentials", func(t *testing.T) {
		p := &Prompt{
			Name:        "test-agent",
			Description: "A test agent",
			Type:        DocumentTypeAgent,
			Credential:  "main",
			Credentials: map[string]*CredentialRef{
				"main": {Provider: ProviderAnthropic},
			},
			Execution: &ExecutionConfig{
				Provider: ProviderAnthropic,
				Model:    "claude-sonnet-4-5",
			},
			Messages: []MessageTemplate{
				{Role: RoleSystem, Content: "You are helpful."},
			},
			Body: "test body",
		}
		assert.NoError(t, p.Validate())
	})

	t.Run("invalid credential in map", func(t *testing.T) {
		p := &Prompt{
			Name:        "test-agent",
			Description: "A test agent",
			Type:        DocumentTypeAgent,
			Credentials: map[string]*CredentialRef{
				"bad": {Label: "no-provider"},
			},
			Execution: &ExecutionConfig{
				Provider: ProviderAnthropic,
				Model:    "claude-sonnet-4-5",
			},
			Messages: []MessageTemplate{
				{Role: RoleSystem, Content: "You are helpful."},
			},
			Body: "test body",
		}
		err := p.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgCredentialMissingProvider)
	})
}

func TestPrompt_ValidateOptional_WithCredentials(t *testing.T) {
	// Credentials should trigger full validation
	p := &Prompt{
		Credentials: map[string]*CredentialRef{
			"main": {Provider: ProviderOpenAI},
		},
	}
	// Should fail because Name is required when credentials are present
	err := p.ValidateOptional()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgPromptNameRequired)
}
