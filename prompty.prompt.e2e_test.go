package prompty

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_V2PromptParsing tests v2.0 prompt parsing with execution config.
func TestE2E_V2PromptParsing(t *testing.T) {
	source := `---
name: my-prompt
description: A test prompt for v2.0 parsing
execution:
  provider: openai
  model: gpt-4
  temperature: 0.7
  max_tokens: 1000
---
Hello {~prompty.var name="user" /~}!`

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	// Should be recognized as v2 prompt
	assert.True(t, tmpl.HasPrompt())
	prompt := tmpl.Prompt()
	require.NotNil(t, prompt)

	assert.Equal(t, "my-prompt", prompt.Name)
	assert.Equal(t, "A test prompt for v2.0 parsing", prompt.Description)

	// Execution config should be present
	require.NotNil(t, prompt.Execution)
	assert.Equal(t, "openai", prompt.Execution.Provider)
	assert.Equal(t, "gpt-4", prompt.Execution.Model)

	temp, ok := prompt.Execution.GetTemperature()
	require.True(t, ok)
	assert.Equal(t, 0.7, temp)

	maxTokens, ok := prompt.Execution.GetMaxTokens()
	require.True(t, ok)
	assert.Equal(t, 1000, maxTokens)

	// Execute should still work
	result, err := tmpl.Execute(context.Background(), map[string]any{
		"user": "Alice",
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello Alice!", result)
}

// TestE2E_V2PromptWithSkope tests v2.0 prompt parsing with skope config.
func TestE2E_V2PromptWithSkope(t *testing.T) {
	source := `---
name: my-prompt
description: A test prompt with skope
skope:
  visibility: public
  projects:
    - project1
    - project2
---
Content here`

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	// Should be recognized as v2 prompt
	assert.True(t, tmpl.HasPrompt())
	prompt := tmpl.Prompt()
	require.NotNil(t, prompt)

	assert.Equal(t, "my-prompt", prompt.Name)

	// Skope config should be present
	require.NotNil(t, prompt.Skope)
	assert.Equal(t, SkopeVisibilityPublic, prompt.Skope.Visibility)
	assert.Equal(t, []string{"project1", "project2"}, prompt.Skope.Projects)
}

// TestE2E_V2PromptWithFullConfig tests v2.0 prompt with all fields.
func TestE2E_V2PromptWithFullConfig(t *testing.T) {
	source := `---
name: full-config-prompt
description: A comprehensive test prompt
license: MIT
compatibility: gpt-4,claude-3
allowed_tools: calculator,search
metadata:
  author: test
  version: 1.0.0
execution:
  provider: anthropic
  model: claude-sonnet-4-5
  temperature: 0.5
  max_tokens: 2000
  top_k: 40
  thinking:
    enabled: true
    budget_tokens: 5000
skope:
  slug: full-config
  visibility: team
  version_number: 3
inputs:
  query:
    type: string
    required: true
    description: The user query
outputs:
  result:
    type: object
    description: The response
sample:
  query: "What is 2+2?"
---
Process: {~prompty.var name="query" /~}`

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	assert.True(t, tmpl.HasPrompt())
	prompt := tmpl.Prompt()
	require.NotNil(t, prompt)

	// Basic fields
	assert.Equal(t, "full-config-prompt", prompt.Name)
	assert.Equal(t, "A comprehensive test prompt", prompt.Description)
	assert.Equal(t, "MIT", prompt.License)
	assert.Equal(t, "gpt-4,claude-3", prompt.Compatibility)
	assert.Equal(t, "calculator,search", prompt.AllowedTools)

	// Metadata
	require.NotNil(t, prompt.Metadata)
	assert.Equal(t, "test", prompt.Metadata["author"])

	// Execution
	require.NotNil(t, prompt.Execution)
	assert.Equal(t, "anthropic", prompt.Execution.Provider)
	assert.Equal(t, "claude-sonnet-4-5", prompt.Execution.Model)
	assert.True(t, prompt.Execution.HasThinking())
	require.NotNil(t, prompt.Execution.Thinking)
	assert.True(t, prompt.Execution.Thinking.Enabled)

	// Skope
	require.NotNil(t, prompt.Skope)
	assert.Equal(t, "full-config", prompt.Skope.Slug)
	assert.Equal(t, SkopeVisibilityTeam, prompt.Skope.Visibility)
	assert.Equal(t, 3, prompt.Skope.VersionNumber)

	// Inputs/Outputs
	require.NotNil(t, prompt.Inputs)
	assert.Contains(t, prompt.Inputs, "query")
	assert.True(t, prompt.Inputs["query"].Required)

	require.NotNil(t, prompt.Outputs)
	assert.Contains(t, prompt.Outputs, "result")

	// Sample
	require.NotNil(t, prompt.Sample)
	assert.Equal(t, "What is 2+2?", prompt.Sample["query"])

	// Slug should come from skope config
	assert.Equal(t, "full-config", prompt.GetSlug())

	// Execute with sample data
	result, err := tmpl.Execute(context.Background(), prompt.Sample)
	require.NoError(t, err)
	assert.Contains(t, result, "What is 2+2?")
}

// TestE2E_V2PromptValidation tests that invalid v2 prompts are rejected.
func TestE2E_V2PromptValidation(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr string
	}{
		{
			name: "missing name",
			source: `---
description: A prompt without name
execution:
  provider: openai
---
content`,
			wantErr: ErrMsgPromptNameRequired,
		},
		{
			name: "missing description",
			source: `---
name: prompt-without-desc
execution:
  provider: openai
---
content`,
			wantErr: ErrMsgPromptDescriptionRequired,
		},
		{
			name: "invalid name format",
			source: `---
name: MyPrompt
description: Invalid name format
execution:
  provider: openai
---
content`,
			wantErr: ErrMsgPromptNameInvalidFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := MustNew()
			_, err := engine.Parse(tt.source)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestE2E_V2PromptProviderConversion tests provider-specific format conversion.
func TestE2E_V2PromptProviderConversion(t *testing.T) {
	source := `---
name: format-test
description: Test provider format conversion
execution:
  provider: openai
  model: gpt-4
  temperature: 0.7
  max_tokens: 1000
  response_format:
    type: json_schema
    json_schema:
      name: test_response
      strict: true
      schema:
        type: object
        properties:
          answer:
            type: string
        required:
          - answer
---
content`

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	prompt := tmpl.Prompt()
	require.NotNil(t, prompt)
	require.NotNil(t, prompt.Execution)

	// Test OpenAI conversion
	openAIFormat := prompt.Execution.ToOpenAI()
	require.NotNil(t, openAIFormat)
	assert.Equal(t, "gpt-4", openAIFormat["model"])
	assert.Equal(t, 0.7, openAIFormat[ParamKeyTemperature])

	// Test Anthropic conversion
	anthropicFormat := prompt.Execution.ToAnthropic()
	require.NotNil(t, anthropicFormat)
	assert.Equal(t, "gpt-4", anthropicFormat["model"])

	// Test ProviderFormat
	openAIRF, err := prompt.Execution.ProviderFormat(ProviderOpenAI)
	require.NoError(t, err)
	require.NotNil(t, openAIRF)
}

// TestE2E_V2PromptInputValidation tests input validation.
func TestE2E_V2PromptInputValidation(t *testing.T) {
	source := `---
name: input-validation-test
description: Test input validation
execution:
  provider: openai
inputs:
  query:
    type: string
    required: true
  limit:
    type: number
    required: false
---
Query: {~prompty.var name="query" /~}`

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	prompt := tmpl.Prompt()
	require.NotNil(t, prompt)

	// Valid inputs
	err = prompt.ValidateInputs(map[string]any{
		"query": "test",
		"limit": 10,
	})
	assert.NoError(t, err)

	// Missing required input
	err = prompt.ValidateInputs(map[string]any{})
	assert.Error(t, err)

	// Wrong type
	err = prompt.ValidateInputs(map[string]any{
		"query": 123, // should be string
	})
	assert.Error(t, err)
}

// mockPromptResolver implements PromptBodyResolver for testing.
type mockPromptResolver struct {
	prompts map[string]string // slug -> body
}

func (m *mockPromptResolver) ResolvePromptBody(_ context.Context, slug string, _ string) (string, error) {
	body, ok := m.prompts[slug]
	if !ok {
		return "", NewRefNotFoundError(slug, "latest")
	}
	return body, nil
}

// TestE2E_V2RefTag tests the prompty.ref tag for prompt composition.
func TestE2E_V2RefTag(t *testing.T) {
	// Create mock resolver with some prompts
	resolver := &mockPromptResolver{
		prompts: map[string]string{
			"greeting":   "Hello, welcome!",
			"signature":  "Best regards, Bot",
			"nested-ref": "Start: {~prompty.ref slug=\"greeting\" /~} End",
		},
	}

	engine := MustNew()

	t.Run("basic reference", func(t *testing.T) {
		source := `Message: {~prompty.ref slug="greeting" /~}`
		tmpl, err := engine.Parse(source)
		require.NoError(t, err)

		// Create context with prompt resolver
		execCtx := NewContext(nil).WithPromptResolver(resolver)

		result, err := tmpl.ExecuteWithContext(context.Background(), execCtx)
		require.NoError(t, err)
		assert.Equal(t, "Message: Hello, welcome!", result)
	})

	t.Run("multiple references", func(t *testing.T) {
		source := `{~prompty.ref slug="greeting" /~}

Content here.

{~prompty.ref slug="signature" /~}`
		tmpl, err := engine.Parse(source)
		require.NoError(t, err)

		execCtx := NewContext(nil).WithPromptResolver(resolver)

		result, err := tmpl.ExecuteWithContext(context.Background(), execCtx)
		require.NoError(t, err)
		assert.Contains(t, result, "Hello, welcome!")
		assert.Contains(t, result, "Best regards, Bot")
	})

	t.Run("reference with version", func(t *testing.T) {
		source := `{~prompty.ref slug="greeting" version="v1" /~}`
		tmpl, err := engine.Parse(source)
		require.NoError(t, err)

		execCtx := NewContext(nil).WithPromptResolver(resolver)

		result, err := tmpl.ExecuteWithContext(context.Background(), execCtx)
		require.NoError(t, err)
		assert.Equal(t, "Hello, welcome!", result)
	})

	t.Run("slug@version syntax", func(t *testing.T) {
		source := `{~prompty.ref slug="greeting@v2" /~}`
		tmpl, err := engine.Parse(source)
		require.NoError(t, err)

		execCtx := NewContext(nil).WithPromptResolver(resolver)

		result, err := tmpl.ExecuteWithContext(context.Background(), execCtx)
		require.NoError(t, err)
		assert.Equal(t, "Hello, welcome!", result)
	})

	t.Run("not found error", func(t *testing.T) {
		source := `{~prompty.ref slug="nonexistent" /~}`
		tmpl, err := engine.Parse(source)
		require.NoError(t, err)

		execCtx := NewContext(nil).WithPromptResolver(resolver)

		_, err = tmpl.ExecuteWithContext(context.Background(), execCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "referenced prompt not found")
	})

	t.Run("missing slug error", func(t *testing.T) {
		source := `{~prompty.ref /~}`
		tmpl, err := engine.Parse(source)
		require.NoError(t, err) // Parse succeeds

		execCtx := NewContext(nil).WithPromptResolver(resolver)

		// Execution should fail with missing slug error
		_, err = tmpl.ExecuteWithContext(context.Background(), execCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "slug")
	})

	t.Run("invalid slug format", func(t *testing.T) {
		source := `{~prompty.ref slug="Invalid-Slug" /~}`
		tmpl, err := engine.Parse(source)
		require.NoError(t, err) // Parse succeeds

		execCtx := NewContext(nil).WithPromptResolver(resolver)

		_, err = tmpl.ExecuteWithContext(context.Background(), execCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid prompt slug")
	})

	t.Run("no resolver available", func(t *testing.T) {
		source := `{~prompty.ref slug="greeting" /~}`
		tmpl, err := engine.Parse(source)
		require.NoError(t, err)

		// Context without prompt resolver
		execCtx := NewContext(nil)

		_, err = tmpl.ExecuteWithContext(context.Background(), execCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "prompt resolver not available")
	})
}

// TestE2E_V2RefTagCircularDetection tests circular reference detection.
func TestE2E_V2RefTagCircularDetection(t *testing.T) {
	// Create resolver with circular references
	resolver := &mockPromptResolver{
		prompts: map[string]string{
			"prompt-a": "A includes: {~prompty.ref slug=\"prompt-b\" /~}",
			"prompt-b": "B includes: {~prompty.ref slug=\"prompt-a\" /~}",
		},
	}

	engine := MustNew()
	source := `{~prompty.ref slug="prompt-a" /~}`
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	// Set up context with resolver and initial chain
	execCtx := NewContext(nil).
		WithPromptResolver(resolver).
		WithRefChain([]string{"prompt-a"}) // Simulate that we're already resolving prompt-a

	_, err = tmpl.ExecuteWithContext(context.Background(), execCtx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular reference")
}

// TestE2E_V1StyleTemplateParsesAsPrompt tests that old v1-style frontmatter is
// now parsed as a Prompt (v1 InferenceConfig fallback has been removed).
func TestE2E_V1StyleTemplateParsesAsPrompt(t *testing.T) {
	// Old v1 template with model: block — now parsed as Prompt
	source := `---
name: v1-template
description: A v1 template
---
Hello {~prompty.var name="user" /~}!`

	engine := MustNew()
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	// Now parsed as Prompt (v1 fallback removed)
	assert.True(t, tmpl.HasPrompt())
	prompt := tmpl.Prompt()
	require.NotNil(t, prompt)
	assert.Equal(t, "v1-template", prompt.Name)

	// Execute should still work
	result, err := tmpl.Execute(context.Background(), map[string]any{
		"user": "Alice",
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello Alice!", result)
}
