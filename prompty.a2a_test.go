package prompty

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileAgentCard_NilPrompt(t *testing.T) {
	var p *Prompt
	card, err := p.CompileAgentCard(context.Background(), &A2AAgentCardOptions{URL: "https://example.com"})
	require.Error(t, err)
	assert.Nil(t, card)
	assert.Contains(t, err.Error(), ErrMsgA2ACardNilPrompt)
}

func TestCompileAgentCard_NilOptions(t *testing.T) {
	p := &Prompt{Name: "test-agent", Description: "A test agent"}
	card, err := p.CompileAgentCard(context.Background(), nil)
	require.Error(t, err)
	assert.Nil(t, card)
	assert.Contains(t, err.Error(), ErrMsgA2ACardMissingURL)
}

func TestCompileAgentCard_MissingURL(t *testing.T) {
	p := &Prompt{Name: "test-agent", Description: "A test agent"}
	card, err := p.CompileAgentCard(context.Background(), &A2AAgentCardOptions{})
	require.Error(t, err)
	assert.Nil(t, card)
	assert.Contains(t, err.Error(), ErrMsgA2ACardMissingURL)
}

func TestCompileAgentCard_MissingName(t *testing.T) {
	p := &Prompt{Description: "No name"}
	card, err := p.CompileAgentCard(context.Background(), &A2AAgentCardOptions{URL: "https://example.com"})
	require.Error(t, err)
	assert.Nil(t, card)
	assert.Contains(t, err.Error(), ErrMsgA2ACardMissingName)
}

func TestCompileAgentCard_MinimalAgent(t *testing.T) {
	p := &Prompt{
		Name:        "simple-agent",
		Description: "A minimal agent",
	}

	card, err := p.CompileAgentCard(context.Background(), &A2AAgentCardOptions{
		URL: "https://agents.example.com/simple",
	})
	require.NoError(t, err)
	require.NotNil(t, card)

	assert.Equal(t, "simple-agent", card.Name)
	assert.Equal(t, "A minimal agent", card.Description)
	assert.Equal(t, "https://agents.example.com/simple", card.URL)
	assert.Equal(t, A2AProtocolVersionDefault, card.ProtocolVersion)
	assert.Equal(t, A2AVersionDefault, card.Version)
	assert.Nil(t, card.Provider)
	assert.Empty(t, card.Skills)
	// Default input/output modes
	assert.Equal(t, []string{A2AMIMETextPlain}, card.DefaultInputModes)
	assert.Equal(t, []string{A2AMIMETextPlain}, card.DefaultOutputModes)
}

func TestCompileAgentCard_FullOptions(t *testing.T) {
	p := &Prompt{
		Name:        "research-agent",
		Description: "AI research assistant",
		Type:        DocumentTypeAgent,
		Execution: &ExecutionConfig{
			Provider: ProviderAnthropic,
			Model:    "claude-sonnet-4-5",
			Streaming: &StreamingConfig{
				Enabled: true,
				Method:  StreamMethodSSE,
			},
		},
	}

	card, err := p.CompileAgentCard(context.Background(), &A2AAgentCardOptions{
		URL:                  "https://agents.example.com/research",
		ProviderOrganization: "Acme Corp",
		ProviderURL:          "https://acme.example.com",
		Version:              "2.1.0",
		ProtocolVersion:      "0.4.0",
		DefaultInputModes:    []string{A2AMIMETextPlain, A2AMIMEApplicationJSON},
		DefaultOutputModes:   []string{A2AMIMETextMarkdown},
		SecuritySchemes: map[string]any{
			"bearer": map[string]any{
				"type":   "http",
				"scheme": "bearer",
			},
		},
		Security: []map[string][]string{
			{"bearer": {}},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, card)

	assert.Equal(t, "research-agent", card.Name)
	assert.Equal(t, "2.1.0", card.Version)
	assert.Equal(t, "0.4.0", card.ProtocolVersion)

	require.NotNil(t, card.Provider)
	assert.Equal(t, "Acme Corp", card.Provider.Organization)
	assert.Equal(t, "https://acme.example.com", card.Provider.URL)

	// Streaming auto-detected
	require.NotNil(t, card.Capabilities)
	assert.True(t, card.Capabilities.Streaming)

	// Overridden input/output modes
	assert.Equal(t, []string{A2AMIMETextPlain, A2AMIMEApplicationJSON}, card.DefaultInputModes)
	assert.Equal(t, []string{A2AMIMETextMarkdown}, card.DefaultOutputModes)

	// Security
	require.NotNil(t, card.SecuritySchemes)
	assert.Contains(t, card.SecuritySchemes, "bearer")
	require.Len(t, card.Security, 1)
}

func TestCompileAgentCard_StreamingAutoDetect(t *testing.T) {
	ctx := context.Background()

	t.Run("streaming enabled", func(t *testing.T) {
		p := &Prompt{
			Name:        "stream-agent",
			Description: "Streaming agent",
			Execution: &ExecutionConfig{
				Streaming: &StreamingConfig{Enabled: true},
			},
		}
		card, err := p.CompileAgentCard(ctx, &A2AAgentCardOptions{URL: "https://example.com/a"})
		require.NoError(t, err)
		require.NotNil(t, card.Capabilities)
		assert.True(t, card.Capabilities.Streaming)
	})

	t.Run("streaming disabled", func(t *testing.T) {
		p := &Prompt{
			Name:        "nostream-agent",
			Description: "Non-streaming agent",
			Execution: &ExecutionConfig{
				Streaming: &StreamingConfig{Enabled: false},
			},
		}
		card, err := p.CompileAgentCard(ctx, &A2AAgentCardOptions{URL: "https://example.com/b"})
		require.NoError(t, err)
		require.NotNil(t, card.Capabilities)
		assert.False(t, card.Capabilities.Streaming)
	})

	t.Run("no streaming config", func(t *testing.T) {
		p := &Prompt{
			Name:        "basic-agent",
			Description: "Basic agent",
		}
		card, err := p.CompileAgentCard(ctx, &A2AAgentCardOptions{URL: "https://example.com/c"})
		require.NoError(t, err)
		require.NotNil(t, card.Capabilities)
		assert.False(t, card.Capabilities.Streaming)
	})

	t.Run("capabilities override", func(t *testing.T) {
		p := &Prompt{
			Name:        "override-agent",
			Description: "Override caps",
			Execution: &ExecutionConfig{
				Streaming: &StreamingConfig{Enabled: true},
			},
		}
		card, err := p.CompileAgentCard(ctx, &A2AAgentCardOptions{
			URL: "https://example.com/d",
			Capabilities: &A2ACapabilities{
				Streaming:         false,
				PushNotifications: true,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, card.Capabilities)
		assert.False(t, card.Capabilities.Streaming)
		assert.True(t, card.Capabilities.PushNotifications)
	})
}

func TestCompileAgentCard_Skills(t *testing.T) {
	ctx := context.Background()
	resolver := NewMapDocumentResolver()
	resolver.AddSkill("web-search", &Prompt{
		Name:        "web-search",
		Description: "Search the web for information",
	})

	p := &Prompt{
		Name:        "multi-skill-agent",
		Description: "Agent with multiple skills",
		Type:        DocumentTypeAgent,
		Skills: []SkillRef{
			{
				Slug:      "web-search",
				Injection: SkillInjectionSystemPrompt,
			},
			{
				Slug: "summarizer",
				Execution: &ExecutionConfig{
					Modality: ModalityText,
				},
			},
			{
				Slug: "image-gen",
				Execution: &ExecutionConfig{
					Provider: ProviderOpenAI,
					Model:    "dall-e-3",
					Modality: ModalityImage,
				},
			},
		},
	}

	card, err := p.CompileAgentCard(ctx, &A2AAgentCardOptions{
		URL:      "https://agents.example.com/multi",
		Resolver: resolver,
	})
	require.NoError(t, err)
	require.NotNil(t, card)
	require.Len(t, card.Skills, 3)

	// web-search: resolved description and name from resolver
	assert.Equal(t, "web-search", card.Skills[0].ID)
	assert.Equal(t, "web-search", card.Skills[0].Name)
	assert.Equal(t, "Search the web for information", card.Skills[0].Description)

	// summarizer: no resolver match, empty description, text output mode
	assert.Equal(t, "summarizer", card.Skills[1].ID)
	assert.Equal(t, "", card.Skills[1].Description)
	assert.Equal(t, []string{A2AMIMETextPlain}, card.Skills[1].OutputModes)

	// image-gen: image output mode
	assert.Equal(t, "image-gen", card.Skills[2].ID)
	assert.Equal(t, []string{A2AMIMEImagePNG}, card.Skills[2].OutputModes)
}

func TestCompileAgentCard_SkillNameFromResolver(t *testing.T) {
	// When resolver returns a Prompt with a different Name, skill.Name should use it
	resolver := NewMapDocumentResolver()
	resolver.AddSkill("search", &Prompt{
		Name:        "Web Search Engine",
		Description: "Searches the web",
	})

	p := &Prompt{
		Name:        "name-test-agent",
		Description: "Tests skill name resolution",
		Skills: []SkillRef{
			{Slug: "search"},
		},
	}

	card, err := p.CompileAgentCard(context.Background(), &A2AAgentCardOptions{
		URL:      "https://example.com/name",
		Resolver: resolver,
	})
	require.NoError(t, err)
	require.Len(t, card.Skills, 1)
	assert.Equal(t, "search", card.Skills[0].ID)
	assert.Equal(t, "Web Search Engine", card.Skills[0].Name)
	assert.Equal(t, "Searches the web", card.Skills[0].Description)
}

func TestCompileAgentCard_InlineSkill(t *testing.T) {
	p := &Prompt{
		Name:        "inline-agent",
		Description: "Agent with inline skill",
		Skills: []SkillRef{
			{
				Inline: &InlineSkill{
					Slug:        "my-inline",
					Description: "An inline skill",
					Body:        "Hello",
				},
			},
		},
	}

	card, err := p.CompileAgentCard(context.Background(), &A2AAgentCardOptions{URL: "https://example.com/inline"})
	require.NoError(t, err)
	require.Len(t, card.Skills, 1)
	assert.Equal(t, "my-inline", card.Skills[0].ID)
	assert.Equal(t, "An inline skill", card.Skills[0].Description)
}

func TestCompileAgentCard_SkillWithRequirementsModality(t *testing.T) {
	p := &Prompt{
		Name:        "req-agent",
		Description: "Agent with requirements modality on skill",
		Skills: []SkillRef{
			{
				Slug: "audio-skill",
				Requirements: &ExecutionRequirements{
					Modality: ModalityAudioSpeech,
				},
			},
		},
	}

	card, err := p.CompileAgentCard(context.Background(), &A2AAgentCardOptions{URL: "https://example.com/req"})
	require.NoError(t, err)
	require.Len(t, card.Skills, 1)
	assert.Equal(t, []string{A2AMIMEAudioMPEG}, card.Skills[0].OutputModes)
}

func TestCompileAgentCard_InputModesFromInputDefs(t *testing.T) {
	ctx := context.Background()

	t.Run("string inputs", func(t *testing.T) {
		p := &Prompt{
			Name:        "string-input",
			Description: "String input agent",
			Inputs: map[string]*InputDef{
				"query": {Type: SchemaTypeString, Required: true},
			},
		}
		card, err := p.CompileAgentCard(ctx, &A2AAgentCardOptions{URL: "https://example.com/s"})
		require.NoError(t, err)
		assert.Equal(t, []string{A2AMIMETextPlain}, card.DefaultInputModes)
	})

	t.Run("object inputs", func(t *testing.T) {
		p := &Prompt{
			Name:        "object-input",
			Description: "Object input agent",
			Inputs: map[string]*InputDef{
				"config": {Type: SchemaTypeObject},
			},
		}
		card, err := p.CompileAgentCard(ctx, &A2AAgentCardOptions{URL: "https://example.com/o"})
		require.NoError(t, err)
		assert.Equal(t, []string{A2AMIMEApplicationJSON}, card.DefaultInputModes)
	})

	t.Run("mixed inputs", func(t *testing.T) {
		p := &Prompt{
			Name:        "mixed-input",
			Description: "Mixed input agent",
			Inputs: map[string]*InputDef{
				"query":  {Type: SchemaTypeString},
				"config": {Type: SchemaTypeObject},
			},
		}
		card, err := p.CompileAgentCard(ctx, &A2AAgentCardOptions{URL: "https://example.com/m"})
		require.NoError(t, err)
		// Both MIME types, sorted
		assert.Equal(t, []string{A2AMIMEApplicationJSON, A2AMIMETextPlain}, card.DefaultInputModes)
	})

	t.Run("all nil input defs", func(t *testing.T) {
		p := &Prompt{
			Name:        "nil-defs",
			Description: "Agent with nil InputDef entries",
			Inputs: map[string]*InputDef{
				"a": nil,
				"b": nil,
			},
		}
		card, err := p.CompileAgentCard(ctx, &A2AAgentCardOptions{URL: "https://example.com/nil"})
		require.NoError(t, err)
		// All nil defs are skipped, mimeSet is empty → defaults to text/plain
		assert.Equal(t, []string{A2AMIMETextPlain}, card.DefaultInputModes)
	})
}

func TestCompileAgentCard_OutputModesFromModality(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		modality string
		expected []string
	}{
		{"text", ModalityText, []string{A2AMIMETextPlain}},
		{"image", ModalityImage, []string{A2AMIMEImagePNG}},
		{"image_edit", ModalityImageEdit, []string{A2AMIMEImagePNG}},
		{"audio_speech", ModalityAudioSpeech, []string{A2AMIMEAudioMPEG}},
		{"audio_transcription", ModalityAudioTranscription, []string{A2AMIMEAudioMPEG}},
		{"music", ModalityMusic, []string{A2AMIMEAudioMPEG}},
		{"sound_effects", ModalitySoundEffects, []string{A2AMIMEAudioMPEG}},
		{"embedding", ModalityEmbedding, []string{A2AMIMEApplicationJSON}},
		{"video", ModalityVideo, []string{A2AMIMEApplicationJSON}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Prompt{
				Name:        "modality-agent",
				Description: "Modality test",
				Execution: &ExecutionConfig{
					Modality: tt.modality,
				},
			}
			card, err := p.CompileAgentCard(ctx, &A2AAgentCardOptions{URL: "https://example.com/mod"})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, card.DefaultOutputModes)
		})
	}

	t.Run("requirements modality takes precedence", func(t *testing.T) {
		p := &Prompt{
			Name:        "req-modality",
			Description: "Requirements modality",
			Execution: &ExecutionConfig{
				Modality: ModalityText,
			},
			Requirements: &ExecutionRequirements{
				Modality: ModalityImage,
			},
		}
		card, err := p.CompileAgentCard(ctx, &A2AAgentCardOptions{URL: "https://example.com/req"})
		require.NoError(t, err)
		assert.Equal(t, []string{A2AMIMEImagePNG}, card.DefaultOutputModes)
	})
}

func TestCompileAgentCard_Metadata(t *testing.T) {
	ctx := context.Background()

	t.Run("prompt metadata only", func(t *testing.T) {
		p := &Prompt{
			Name:        "meta-agent",
			Description: "Metadata test",
			Metadata: map[string]any{
				"author":  "alice",
				"version": "1.0",
			},
		}
		card, err := p.CompileAgentCard(ctx, &A2AAgentCardOptions{URL: "https://example.com/meta"})
		require.NoError(t, err)
		require.NotNil(t, card.Metadata)
		assert.Equal(t, "alice", card.Metadata["author"])
		assert.Equal(t, "1.0", card.Metadata["version"])
	})

	t.Run("a2a extensions merged", func(t *testing.T) {
		p := &Prompt{
			Name:        "ext-agent",
			Description: "Extensions test",
			Metadata: map[string]any{
				"author": "bob",
			},
			Extensions: map[string]any{
				"a2a.custom_field": "custom_value",
				"a2a.tags":         []string{"research", "ai"},
				"non_a2a_key":      "should be excluded",
			},
		}
		card, err := p.CompileAgentCard(ctx, &A2AAgentCardOptions{URL: "https://example.com/ext"})
		require.NoError(t, err)
		require.NotNil(t, card.Metadata)
		assert.Equal(t, "bob", card.Metadata["author"])
		assert.Equal(t, "custom_value", card.Metadata["a2a.custom_field"])
		assert.NotContains(t, card.Metadata, "non_a2a_key")
	})

	t.Run("no metadata", func(t *testing.T) {
		p := &Prompt{
			Name:        "no-meta",
			Description: "No metadata",
		}
		card, err := p.CompileAgentCard(ctx, &A2AAgentCardOptions{URL: "https://example.com/nometa"})
		require.NoError(t, err)
		assert.Nil(t, card.Metadata)
	})

	t.Run("a2a extensions only no prompt metadata", func(t *testing.T) {
		// Covers the branch: len(p.Metadata)==0 but a2a-prefixed Extensions exist
		p := &Prompt{
			Name:        "ext-only",
			Description: "Extensions only agent",
			Extensions: map[string]any{
				"a2a.network": "production",
				"other.key":   "ignored",
			},
		}
		card, err := p.CompileAgentCard(ctx, &A2AAgentCardOptions{URL: "https://example.com/extonly"})
		require.NoError(t, err)
		require.NotNil(t, card.Metadata)
		assert.Equal(t, "production", card.Metadata["a2a.network"])
		assert.NotContains(t, card.Metadata, "other.key")
		assert.Len(t, card.Metadata, 1)
	})
}

func TestCompileAgentCard_ToJSON(t *testing.T) {
	ctx := context.Background()
	p := &Prompt{
		Name:        "json-agent",
		Description: "JSON serialization test",
		Execution: &ExecutionConfig{
			Provider: ProviderAnthropic,
			Model:    "claude-sonnet-4-5",
		},
		Skills: []SkillRef{
			{Slug: "search"},
		},
	}

	card, err := p.CompileAgentCard(ctx, &A2AAgentCardOptions{
		URL:                  "https://agents.example.com/json",
		ProviderOrganization: "Test Corp",
	})
	require.NoError(t, err)

	// Compact JSON
	data, err := card.ToJSON()
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// Verify it's valid JSON and round-trips
	var parsed A2AAgentCard
	require.NoError(t, json.Unmarshal(data, &parsed))
	assert.Equal(t, "json-agent", parsed.Name)
	assert.Equal(t, "https://agents.example.com/json", parsed.URL)
	assert.Equal(t, A2AProtocolVersionDefault, parsed.ProtocolVersion)
	require.NotNil(t, parsed.Provider)
	assert.Equal(t, "Test Corp", parsed.Provider.Organization)
	require.Len(t, parsed.Skills, 1)
	assert.Equal(t, "search", parsed.Skills[0].ID)

	// Pretty JSON uses JSONIndentDefault
	prettyData, err := card.ToJSONPretty()
	require.NoError(t, err)
	assert.Contains(t, string(prettyData), "\n")
	assert.Contains(t, string(prettyData), JSONIndentDefault)
}

func TestCompileAgentCard_ToJSON_Nil(t *testing.T) {
	var card *A2AAgentCard
	data, err := card.ToJSON()
	require.NoError(t, err)
	assert.Nil(t, data)

	data, err = card.ToJSONPretty()
	require.NoError(t, err)
	assert.Nil(t, data)
}

func TestModalityToMIME(t *testing.T) {
	tests := []struct {
		modality string
		expected string
	}{
		{ModalityText, A2AMIMETextPlain},
		{ModalityImage, A2AMIMEImagePNG},
		{ModalityImageEdit, A2AMIMEImagePNG},
		{ModalityAudioSpeech, A2AMIMEAudioMPEG},
		{ModalityAudioTranscription, A2AMIMEAudioMPEG},
		{ModalityMusic, A2AMIMEAudioMPEG},
		{ModalitySoundEffects, A2AMIMEAudioMPEG},
		{ModalityEmbedding, A2AMIMEApplicationJSON},
		{ModalityVideo, A2AMIMEApplicationJSON},
		{"unknown", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.modality, func(t *testing.T) {
			assert.Equal(t, tt.expected, modalityToMIME(tt.modality))
		})
	}
}

func TestInputTypeToMIME(t *testing.T) {
	tests := []struct {
		inputType string
		expected  string
	}{
		{SchemaTypeString, A2AMIMETextPlain},
		{SchemaTypeNumber, A2AMIMETextPlain},
		{SchemaTypeBoolean, A2AMIMETextPlain},
		{SchemaTypeObject, A2AMIMEApplicationJSON},
		{SchemaTypeArray, A2AMIMEApplicationJSON},
		{"unknown", A2AMIMETextPlain},
	}

	for _, tt := range tests {
		t.Run(tt.inputType, func(t *testing.T) {
			assert.Equal(t, tt.expected, inputTypeToMIME(tt.inputType))
		})
	}
}

func TestCompileAgentCard_MultiModelAgent(t *testing.T) {
	ctx := context.Background()
	// Integration test: multi-model agent from v2.10 → A2A card
	resolver := NewMapDocumentResolver()
	resolver.AddSkill("image-gen", &Prompt{
		Name:        "image-gen",
		Description: "Generate images using DALL-E",
	})
	resolver.AddSkill("embed-query", &Prompt{
		Name:        "embed-query",
		Description: "Embed text for semantic search",
	})

	p := &Prompt{
		Name:        "multi-model-agent",
		Description: "Research agent using multiple providers",
		Type:        DocumentTypeAgent,
		Credential:  "anthropic-main",
		Credentials: map[string]*CredentialRef{
			"anthropic-main": {Provider: ProviderAnthropic, Ref: "${ANTHROPIC_API_KEY}"},
			"openai-images":  {Provider: ProviderOpenAI, Ref: "${OPENAI_IMAGES_KEY}", Scopes: []string{"images"}},
		},
		Execution: &ExecutionConfig{
			Provider: ProviderAnthropic,
			Model:    "claude-sonnet-4-5",
			Streaming: &StreamingConfig{
				Enabled: true,
			},
		},
		Requirements: &ExecutionRequirements{
			Modality:        ModalityText,
			ProviderBinding: ProviderBindingRequired,
		},
		Skills: []SkillRef{
			{
				Slug:       "image-gen",
				Credential: "openai-images",
				Execution: &ExecutionConfig{
					Provider: ProviderOpenAI,
					Model:    "dall-e-3",
					Modality: ModalityImage,
				},
				Requirements: &ExecutionRequirements{
					Modality:           ModalityImage,
					ProviderBinding:    ProviderBindingRequired,
					Capabilities:       []string{"image_generation"},
					EstimatedCost:      "$0.04",
					EstimatedLatencyMs: 8000,
				},
			},
			{
				Slug:       "embed-query",
				Credential: "openai-images",
				Execution: &ExecutionConfig{
					Provider: ProviderOpenAI,
					Model:    "text-embedding-3-large",
					Modality: ModalityEmbedding,
				},
			},
		},
		Inputs: map[string]*InputDef{
			"query":   {Type: SchemaTypeString, Required: true},
			"options": {Type: SchemaTypeObject},
		},
	}

	card, err := p.CompileAgentCard(ctx, &A2AAgentCardOptions{
		URL:                  "https://agents.example.com/multi-model",
		ProviderOrganization: "Research Lab",
		ProviderURL:          "https://research.example.com",
		Resolver:             resolver,
	})
	require.NoError(t, err)
	require.NotNil(t, card)

	// Core fields
	assert.Equal(t, "multi-model-agent", card.Name)
	assert.Equal(t, "Research agent using multiple providers", card.Description)
	assert.Equal(t, "https://agents.example.com/multi-model", card.URL)

	// Provider
	require.NotNil(t, card.Provider)
	assert.Equal(t, "Research Lab", card.Provider.Organization)

	// Streaming auto-detected
	require.NotNil(t, card.Capabilities)
	assert.True(t, card.Capabilities.Streaming)

	// Skills with resolved descriptions
	require.Len(t, card.Skills, 2)
	assert.Equal(t, "image-gen", card.Skills[0].ID)
	assert.Equal(t, "Generate images using DALL-E", card.Skills[0].Description)
	assert.Equal(t, []string{A2AMIMEImagePNG}, card.Skills[0].OutputModes)

	assert.Equal(t, "embed-query", card.Skills[1].ID)
	assert.Equal(t, "Embed text for semantic search", card.Skills[1].Description)
	assert.Equal(t, []string{A2AMIMEApplicationJSON}, card.Skills[1].OutputModes)

	// Input modes from InputDefs (sorted: application/json, text/plain)
	assert.Equal(t, []string{A2AMIMEApplicationJSON, A2AMIMETextPlain}, card.DefaultInputModes)

	// Output mode from requirements modality
	assert.Equal(t, []string{A2AMIMETextPlain}, card.DefaultOutputModes)

	// JSON round-trip
	data, err := card.ToJSON()
	require.NoError(t, err)
	var roundTrip A2AAgentCard
	require.NoError(t, json.Unmarshal(data, &roundTrip))
	assert.Equal(t, card.Name, roundTrip.Name)
	assert.Equal(t, card.URL, roundTrip.URL)
	assert.Len(t, roundTrip.Skills, 2)
}

func TestCompileAgentCard_NoSkills(t *testing.T) {
	p := &Prompt{
		Name:        "no-skills-agent",
		Description: "Agent without skills",
	}
	card, err := p.CompileAgentCard(context.Background(), &A2AAgentCardOptions{URL: "https://example.com/ns"})
	require.NoError(t, err)
	assert.Nil(t, card.Skills)
}

func TestCompileAgentCard_NilResolverFallback(t *testing.T) {
	// Skills should still compile even without a resolver (descriptions empty)
	p := &Prompt{
		Name:        "no-resolver-agent",
		Description: "Agent without resolver",
		Skills: []SkillRef{
			{Slug: "search"},
			{Slug: "summarize"},
		},
	}
	card, err := p.CompileAgentCard(context.Background(), &A2AAgentCardOptions{URL: "https://example.com/nr"})
	require.NoError(t, err)
	require.Len(t, card.Skills, 2)
	assert.Equal(t, "search", card.Skills[0].ID)
	assert.Equal(t, "", card.Skills[0].Description)
	assert.Equal(t, "summarize", card.Skills[1].ID)
	assert.Equal(t, "", card.Skills[1].Description)
}

func TestA2AAgentCard_JSONOmitsEmptyFields(t *testing.T) {
	p := &Prompt{
		Name:        "sparse-agent",
		Description: "Test omitempty",
	}
	card, err := p.CompileAgentCard(context.Background(), &A2AAgentCardOptions{URL: "https://example.com/sparse"})
	require.NoError(t, err)

	data, err := card.ToJSON()
	require.NoError(t, err)

	// Verify omitempty works - these fields should not appear
	jsonStr := string(data)
	assert.NotContains(t, jsonStr, "\"skills\"")
	assert.NotContains(t, jsonStr, "\"securitySchemes\"")
	assert.NotContains(t, jsonStr, "\"security\"")
	assert.NotContains(t, jsonStr, "\"metadata\"")
	assert.NotContains(t, jsonStr, "\"provider\"")
}

func TestNewA2AError(t *testing.T) {
	t.Run("without cause", func(t *testing.T) {
		err := NewA2AError(ErrMsgA2ACardMissingURL, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgA2ACardMissingURL)
	})

	t.Run("with cause", func(t *testing.T) {
		cause := assert.AnError
		err := NewA2AError(ErrMsgA2ACardNilPrompt, cause)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgA2ACardNilPrompt)
	})
}
