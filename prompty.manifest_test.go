package prompty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileExecutionManifest_NilPrompt(t *testing.T) {
	var p *Prompt
	manifest, err := p.CompileExecutionManifest()
	require.Error(t, err)
	assert.Nil(t, manifest)
	assert.Contains(t, err.Error(), ErrMsgManifestNilPrompt)
}

func TestCompileExecutionManifest_MinimalPrompt(t *testing.T) {
	p := &Prompt{
		Name:        "simple",
		Description: "A simple prompt",
	}
	manifest, err := p.CompileExecutionManifest()
	require.NoError(t, err)
	require.NotNil(t, manifest)

	assert.Equal(t, "simple", manifest.Primary.Slug)
	assert.Nil(t, manifest.Primary.Execution)
	assert.Empty(t, manifest.Skills)
	// Default modality is text when none specified
	assert.Equal(t, []string{ModalityText}, manifest.Modalities)
}

func TestCompileExecutionManifest_SingleProvider(t *testing.T) {
	p := &Prompt{
		Name:        "single-provider",
		Description: "Single provider agent",
		Type:        DocumentTypeAgent,
		Credential:  "main",
		Credentials: map[string]*CredentialRef{
			"main": {Provider: ProviderAnthropic, Ref: "${ANTHROPIC_KEY}"},
		},
		Execution: &ExecutionConfig{
			Provider: ProviderAnthropic,
			Model:    "claude-sonnet-4-5",
		},
	}

	manifest, err := p.CompileExecutionManifest()
	require.NoError(t, err)
	require.NotNil(t, manifest)

	assert.Equal(t, []string{ProviderAnthropic}, manifest.Providers)
	assert.Equal(t, []string{"main"}, manifest.Credentials)
	assert.Equal(t, "main", manifest.Primary.Credential)
	assert.Equal(t, []string{ModalityText}, manifest.Modalities)
}

func TestCompileExecutionManifest_MultiProvider(t *testing.T) {
	p := &Prompt{
		Name:        "multi-model-agent",
		Description: "Research agent using multiple providers",
		Type:        DocumentTypeAgent,
		Credential:  "anthropic-main",
		Credentials: map[string]*CredentialRef{
			"anthropic-main":    {Provider: ProviderAnthropic, Ref: "${ANTHROPIC_API_KEY}"},
			"openai-images":     {Provider: ProviderOpenAI, Ref: "${OPENAI_IMAGES_KEY}", Scopes: []string{"images"}},
			"openai-embeddings": {Provider: ProviderOpenAI, Ref: "${OPENAI_EMBEDDINGS_KEY}", Scopes: []string{"embeddings"}},
		},
		Execution: &ExecutionConfig{
			Provider: ProviderAnthropic,
			Model:    "claude-sonnet-4-5",
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
				Credential: "openai-embeddings",
				Execution: &ExecutionConfig{
					Provider: ProviderOpenAI,
					Model:    "text-embedding-3-large",
					Modality: ModalityEmbedding,
				},
				Requirements: &ExecutionRequirements{
					Modality:           ModalityEmbedding,
					ProviderBinding:    ProviderBindingPreferred,
					EstimatedCost:      "$0.0001",
					EstimatedLatencyMs: 200,
				},
			},
		},
	}

	manifest, err := p.CompileExecutionManifest()
	require.NoError(t, err)
	require.NotNil(t, manifest)

	// Providers: anthropic + openai (deduplicated, sorted)
	assert.Equal(t, []string{ProviderAnthropic, ProviderOpenAI}, manifest.Providers)

	// Credentials: all 3 (sorted)
	assert.Equal(t, []string{"anthropic-main", "openai-embeddings", "openai-images"}, manifest.Credentials)

	// Modalities: embedding, image, text (sorted, deduplicated)
	assert.Equal(t, []string{ModalityEmbedding, ModalityImage, ModalityText}, manifest.Modalities)

	// Skills
	assert.Len(t, manifest.Skills, 2)

	imgSlot := manifest.Skills["image-gen"]
	assert.Equal(t, "image-gen", imgSlot.Slug)
	assert.Equal(t, "openai-images", imgSlot.Credential)
	assert.Equal(t, ProviderOpenAI, imgSlot.Execution.Provider)
	require.NotNil(t, imgSlot.Requirements)
	assert.Equal(t, ProviderBindingRequired, imgSlot.Requirements.ProviderBinding)

	embedSlot := manifest.Skills["embed-query"]
	assert.Equal(t, "embed-query", embedSlot.Slug)
	assert.Equal(t, "openai-embeddings", embedSlot.Credential)
	assert.Equal(t, ModalityEmbedding, embedSlot.Execution.Modality)
}

func TestExecutionManifest_HasProvider(t *testing.T) {
	t.Run("nil manifest", func(t *testing.T) {
		var m *ExecutionManifest
		assert.False(t, m.HasProvider(ProviderOpenAI))
	})

	t.Run("provider found", func(t *testing.T) {
		m := &ExecutionManifest{Providers: []string{ProviderAnthropic, ProviderOpenAI}}
		assert.True(t, m.HasProvider(ProviderOpenAI))
	})

	t.Run("provider not found", func(t *testing.T) {
		m := &ExecutionManifest{Providers: []string{ProviderAnthropic}}
		assert.False(t, m.HasProvider(ProviderOpenAI))
	})
}

func TestExecutionManifest_RequiresCredential(t *testing.T) {
	t.Run("nil manifest", func(t *testing.T) {
		var m *ExecutionManifest
		assert.False(t, m.RequiresCredential("main"))
	})

	t.Run("credential found", func(t *testing.T) {
		m := &ExecutionManifest{Credentials: []string{"main", "images"}}
		assert.True(t, m.RequiresCredential("main"))
		assert.True(t, m.RequiresCredential("images"))
	})

	t.Run("credential not found", func(t *testing.T) {
		m := &ExecutionManifest{Credentials: []string{"main"}}
		assert.False(t, m.RequiresCredential("missing"))
	})
}

func TestExecutionManifest_SkillsForProvider(t *testing.T) {
	t.Run("nil manifest", func(t *testing.T) {
		var m *ExecutionManifest
		assert.Nil(t, m.SkillsForProvider(ProviderOpenAI))
	})

	t.Run("no matching skills", func(t *testing.T) {
		m := &ExecutionManifest{
			Skills: map[string]ExecutionSlot{
				"skill-a": {
					Slug:      "skill-a",
					Execution: &ExecutionConfig{Provider: ProviderAnthropic, Model: "claude-sonnet-4-5"},
				},
			},
		}
		result := m.SkillsForProvider(ProviderOpenAI)
		assert.Empty(t, result)
	})

	t.Run("matching skills sorted", func(t *testing.T) {
		m := &ExecutionManifest{
			Skills: map[string]ExecutionSlot{
				"embed": {
					Slug:      "embed",
					Execution: &ExecutionConfig{Provider: ProviderOpenAI, Model: "text-embedding-3-large"},
				},
				"images": {
					Slug:      "images",
					Execution: &ExecutionConfig{Provider: ProviderOpenAI, Model: "dall-e-3"},
				},
				"chat": {
					Slug:      "chat",
					Execution: &ExecutionConfig{Provider: ProviderAnthropic, Model: "claude-sonnet-4-5"},
				},
			},
		}
		result := m.SkillsForProvider(ProviderOpenAI)
		assert.Equal(t, []string{"embed", "images"}, result)
	})
}

func TestCompileExecutionManifest_SkillWithoutExecution(t *testing.T) {
	p := &Prompt{
		Name:        "agent",
		Description: "Agent with skill without execution",
		Type:        DocumentTypeAgent,
		Execution: &ExecutionConfig{
			Provider: ProviderAnthropic,
			Model:    "claude-sonnet-4-5",
		},
		Skills: []SkillRef{
			{Slug: "no-exec-skill"},
		},
	}

	manifest, err := p.CompileExecutionManifest()
	require.NoError(t, err)
	require.NotNil(t, manifest)

	assert.Len(t, manifest.Skills, 1)
	slot := manifest.Skills["no-exec-skill"]
	assert.Equal(t, "no-exec-skill", slot.Slug)
	assert.Nil(t, slot.Execution)
	// Only the primary provider
	assert.Equal(t, []string{ProviderAnthropic}, manifest.Providers)
}

func TestCompileExecutionManifest_RequirementsModality(t *testing.T) {
	// Requirements modality should appear in the manifest even if execution modality is not set
	p := &Prompt{
		Name:        "agent",
		Description: "Agent with requirements modality",
		Requirements: &ExecutionRequirements{
			Modality: ModalityImage,
		},
	}

	manifest, err := p.CompileExecutionManifest()
	require.NoError(t, err)
	require.NotNil(t, manifest)

	assert.Contains(t, manifest.Modalities, ModalityImage)
}

func TestCompileExecutionManifest_CredentialsDeclaredButNotReferenced(t *testing.T) {
	// All declared credentials should appear in manifest even if not referenced by skills
	p := &Prompt{
		Name:        "agent",
		Description: "Agent with extra credentials",
		Credentials: map[string]*CredentialRef{
			"unused-key": {Provider: ProviderCohere},
			"main":       {Provider: ProviderAnthropic},
		},
		Credential: "main",
		Execution: &ExecutionConfig{
			Provider: ProviderAnthropic,
			Model:    "claude-sonnet-4-5",
		},
	}

	manifest, err := p.CompileExecutionManifest()
	require.NoError(t, err)
	require.NotNil(t, manifest)

	assert.Equal(t, []string{"main", "unused-key"}, manifest.Credentials)
}

func TestSortedKeys(t *testing.T) {
	t.Run("nil map", func(t *testing.T) {
		assert.Nil(t, sortedKeys(nil))
	})

	t.Run("empty map", func(t *testing.T) {
		assert.Nil(t, sortedKeys(map[string]bool{}))
	})

	t.Run("sorted output", func(t *testing.T) {
		m := map[string]bool{"c": true, "a": true, "b": true}
		assert.Equal(t, []string{"a", "b", "c"}, sortedKeys(m))
	})
}
