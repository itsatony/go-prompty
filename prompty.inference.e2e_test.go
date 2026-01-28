package prompty

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_FrontmatterBasicParsing tests basic YAML frontmatter parsing
func TestE2E_FrontmatterBasicParsing(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `---
name: test-template
description: A test template
version: 1.0.0
---
Hello World!`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)
	require.NotNil(t, tmpl)

	assert.True(t, tmpl.HasInferenceConfig())
	config := tmpl.InferenceConfig()
	require.NotNil(t, config)

	assert.Equal(t, "test-template", config.Name)
	assert.Equal(t, "A test template", config.Description)
	assert.Equal(t, "1.0.0", config.Version)

	// Template body should be just the content after frontmatter
	assert.Equal(t, "Hello World!", tmpl.TemplateBody())

	// Source should contain the full template including frontmatter
	assert.Contains(t, tmpl.Source(), "---")
}

// TestE2E_FrontmatterWithModel tests frontmatter with model configuration
func TestE2E_FrontmatterWithModel(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `---
name: chat-template
model:
  api: chat
  provider: openai
  name: gpt-4
  parameters:
    temperature: 0.7
    max_tokens: 2048
    top_p: 0.9
    stop:
      - "\n\n"
      - END
---
User: {~prompty.var name="query" /~}`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	config := tmpl.InferenceConfig()
	require.NotNil(t, config)
	require.True(t, config.HasModel())

	assert.Equal(t, ModelAPIChat, config.Model.API)
	assert.Equal(t, "openai", config.Model.Provider)
	assert.Equal(t, "gpt-4", config.Model.Name)

	temp, ok := config.GetTemperature()
	assert.True(t, ok)
	assert.Equal(t, 0.7, temp)

	maxTokens, ok := config.GetMaxTokens()
	assert.True(t, ok)
	assert.Equal(t, 2048, maxTokens)

	topP, ok := config.GetTopP()
	assert.True(t, ok)
	assert.Equal(t, 0.9, topP)

	assert.Equal(t, []string{"\n\n", "END"}, config.GetStopSequences())
}

// TestE2E_FrontmatterWithVariables tests frontmatter with template variables
func TestE2E_FrontmatterWithVariables(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	// IMPORTANT: Use YAML single quotes for strings containing prompty tags
	// YAML double quotes require backslash escaping (e.g., \") which conflicts
	// with prompty tag parsing. Single quotes preserve literal characters.
	source := `---
name: dynamic-template
model:
  name: '{~prompty.var name="model_name" default="gpt-4" /~}'
---
Hello {~prompty.var name="user" /~}!`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	config := tmpl.InferenceConfig()
	require.NotNil(t, config)

	// The model name should be resolved from the default value
	assert.Equal(t, "gpt-4", config.Model.Name)

	// Execute the template to verify it works
	result, err := tmpl.Execute(context.Background(), map[string]any{
		"user": "Alice",
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello Alice!", result)
}

// TestE2E_FrontmatterWithEnvVars tests frontmatter with environment variables
func TestE2E_FrontmatterWithEnvVars(t *testing.T) {
	// Set up test environment variable
	testAPIKey := "sk-test-key-12345"
	os.Setenv("TEST_API_KEY", testAPIKey)
	defer os.Unsetenv("TEST_API_KEY")

	engine, err := New()
	require.NoError(t, err)

	// Use YAML single quotes for strings containing prompty tags
	source := `---
name: env-template
description: 'API Key: {~prompty.env name="TEST_API_KEY" /~}'
---
Template content`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	config := tmpl.InferenceConfig()
	require.NotNil(t, config)

	// The description should contain the resolved env var
	assert.Equal(t, "API Key: "+testAPIKey, config.Description)
}

// TestE2E_FrontmatterTemplateExecution tests that templates with frontmatter execute correctly
func TestE2E_FrontmatterTemplateExecution(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `---
name: greet-template
sample:
  name: World
---
Hello {~prompty.var name="name" /~}!

{~prompty.if eval="context.formal"~}
How do you do?
{~prompty.else~}
How are you?
{~/prompty.if~}`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	// Execute with provided data
	result, err := tmpl.Execute(context.Background(), map[string]any{
		"name": "Alice",
		"context": map[string]any{
			"formal": true,
		},
	})
	require.NoError(t, err)
	assert.Contains(t, result, "Hello Alice!")
	assert.Contains(t, result, "How do you do?")

	// Execute with informal context
	result, err = tmpl.Execute(context.Background(), map[string]any{
		"name": "Bob",
		"context": map[string]any{
			"formal": false,
		},
	})
	require.NoError(t, err)
	assert.Contains(t, result, "Hello Bob!")
	assert.Contains(t, result, "How are you?")
}

// TestE2E_FrontmatterInputValidation tests input validation
func TestE2E_FrontmatterInputValidation(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `---
inputs:
  name:
    type: string
    required: true
  age:
    type: number
    required: true
  active:
    type: boolean
    required: false
---
Name: {~prompty.var name="name" /~}, Age: {~prompty.var name="age" /~}`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	config := tmpl.InferenceConfig()
	require.NotNil(t, config)
	assert.True(t, config.HasInputs())

	// Valid inputs should pass validation
	err = config.ValidateInputs(map[string]any{
		"name": "Alice",
		"age":  30,
	})
	assert.NoError(t, err)

	// Missing required input should fail
	err = config.ValidateInputs(map[string]any{
		"name": "Alice",
	})
	assert.Error(t, err)

	// Wrong type should fail
	err = config.ValidateInputs(map[string]any{
		"name": "Alice",
		"age":  "thirty",
	})
	assert.Error(t, err)
}

// TestE2E_FrontmatterSampleData tests sample data extraction
func TestE2E_FrontmatterSampleData(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `---
sample:
  user: Alice
  items:
    - apple
    - banana
  count: 42
  active: true
---
Template`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	config := tmpl.InferenceConfig()
	require.NotNil(t, config)
	assert.True(t, config.HasSample())

	sample := config.GetSampleData()
	assert.Equal(t, "Alice", sample["user"])
	assert.Equal(t, 42, sample["count"]) // YAML preserves int
	assert.Equal(t, true, sample["active"])
}

// TestE2E_FrontmatterStorageRoundtrip tests storing and retrieving templates with config
func TestE2E_FrontmatterStorageRoundtrip(t *testing.T) {
	storage, err := OpenStorage(StorageDriverNameMemory, "")
	require.NoError(t, err)
	defer storage.Close()

	engine, err := New()
	require.NoError(t, err)

	se, err := NewStorageEngine(StorageEngineConfig{
		Storage: storage,
		Engine:  engine,
	})
	require.NoError(t, err)

	source := `---
name: stored-template
version: 1.0.0
model:
  api: chat
  name: gpt-4
---
Hello {~prompty.var name="user" /~}!`

	// Save template
	tmpl := &StoredTemplate{
		Name:   "test-template",
		Source: source,
	}
	err = se.Save(context.Background(), tmpl)
	require.NoError(t, err)

	// InferenceConfig should be automatically extracted
	require.NotNil(t, tmpl.InferenceConfig)
	assert.Equal(t, "stored-template", tmpl.InferenceConfig.Name)
	assert.Equal(t, "1.0.0", tmpl.InferenceConfig.Version)

	// Retrieve and verify
	retrieved, err := se.Get(context.Background(), "test-template")
	require.NoError(t, err)
	require.NotNil(t, retrieved.InferenceConfig)
	assert.Equal(t, "stored-template", retrieved.InferenceConfig.Name)

	// Execute the stored template
	result, err := se.Execute(context.Background(), "test-template", map[string]any{
		"user": "Alice",
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello Alice!", result)
}

// TestE2E_NoFrontmatterBackwardCompatible tests that templates without frontmatter work
func TestE2E_NoFrontmatterBackwardCompatible(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `Hello {~prompty.var name="user" /~}!`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	assert.False(t, tmpl.HasInferenceConfig())
	assert.Nil(t, tmpl.InferenceConfig())
	assert.Equal(t, source, tmpl.TemplateBody())

	result, err := tmpl.Execute(context.Background(), map[string]any{
		"user": "World",
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello World!", result)
}

// TestE2E_FrontmatterMalformedYAML tests error handling for malformed YAML
func TestE2E_FrontmatterMalformedYAML(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `---
invalid: yaml: content: [
---
Template`

	_, err = engine.Parse(source)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgFrontmatterParse)
}

// TestE2E_FrontmatterUnclosed tests error handling for unclosed frontmatter
func TestE2E_FrontmatterUnclosed(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `---
name: test
Template body without closing delimiter`

	_, err = engine.Parse(source)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgConfigBlockExtract)
}

// TestE2E_FrontmatterWithAuthors tests frontmatter with authors array
func TestE2E_FrontmatterWithAuthors(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `---
name: team-template
authors:
  - alice@example.com
  - bob@example.com
tags:
  - production
  - customer-service
---
Template`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	config := tmpl.InferenceConfig()
	require.NotNil(t, config)

	assert.Equal(t, []string{"alice@example.com", "bob@example.com"}, config.Authors)
	assert.Equal(t, []string{"production", "customer-service"}, config.Tags)
}

// TestE2E_FrontmatterModelParametersToMap tests model parameters ToMap conversion
func TestE2E_FrontmatterModelParametersToMap(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `---
model:
  parameters:
    temperature: 0.5
    max_tokens: 1024
    top_p: 0.95
    frequency_penalty: 0.1
    presence_penalty: 0.2
    seed: 12345
---
Template`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	config := tmpl.InferenceConfig()
	require.NotNil(t, config)
	require.NotNil(t, config.Model)
	require.NotNil(t, config.Model.Parameters)

	params := config.Model.Parameters.ToMap()
	assert.Equal(t, 0.5, params[ParamKeyTemperature])
	assert.Equal(t, 1024, params[ParamKeyMaxTokens])
	assert.Equal(t, 0.95, params[ParamKeyTopP])
	assert.Equal(t, 0.1, params[ParamKeyFrequencyPenalty])
	assert.Equal(t, 0.2, params[ParamKeyPresencePenalty])
	assert.Equal(t, int64(12345), params[ParamKeySeed])
}

// TestE2E_FrontmatterWithOutputs tests frontmatter with outputs schema
func TestE2E_FrontmatterWithOutputs(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `---
outputs:
  response:
    type: string
    description: The model response
  confidence:
    type: number
    description: Confidence score
---
Template`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	config := tmpl.InferenceConfig()
	require.NotNil(t, config)
	assert.True(t, config.HasOutputs())

	assert.Equal(t, SchemaTypeString, config.Outputs["response"].Type)
	assert.Equal(t, "The model response", config.Outputs["response"].Description)
	assert.Equal(t, SchemaTypeNumber, config.Outputs["confidence"].Type)
}

// TestE2E_FrontmatterWithLeadingWhitespace tests frontmatter after whitespace
func TestE2E_FrontmatterWithLeadingWhitespace(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `  ---
name: whitespace-test
---
Template`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	assert.True(t, tmpl.HasInferenceConfig())
	assert.Equal(t, "whitespace-test", tmpl.InferenceConfig().Name)
}

// TestE2E_FrontmatterInMiddleNotExtracted tests that frontmatter in middle are not extracted
func TestE2E_FrontmatterInMiddleNotExtracted(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	// Frontmatter after content should be treated as regular text
	source := `Hello World
---
name: middle-config
---
More content`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	// Should NOT have config because it's not at the start
	assert.False(t, tmpl.HasInferenceConfig())
	assert.Equal(t, source, tmpl.TemplateBody())
}

// TestE2E_FrontmatterEnvVarWithDefault tests env var with default value
func TestE2E_FrontmatterEnvVarWithDefault(t *testing.T) {
	// Make sure env var is not set
	os.Unsetenv("MISSING_VAR_FOR_TEST")

	engine, err := New()
	require.NoError(t, err)

	// Use YAML single quotes for strings containing prompty tags
	source := `---
name: env-default-test
description: '{~prompty.env name="MISSING_VAR_FOR_TEST" default="default-value" /~}'
---
Template`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	config := tmpl.InferenceConfig()
	require.NotNil(t, config)
	assert.Equal(t, "default-value", config.Description)
}

// TestE2E_FullPromptyTemplate tests a complete realistic template
func TestE2E_FullPromptyTemplate(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `---
name: customer-support-agent
description: Handles customer inquiries with empathetic responses
version: 1.0.0
authors:
  - support-team@example.com
tags:
  - production
  - customer-service
model:
  api: chat
  provider: openai
  name: gpt-4
  parameters:
    temperature: 0.7
    max_tokens: 2048
inputs:
  customer_name:
    type: string
    required: true
  query:
    type: string
    required: true
  priority:
    type: string
    required: false
outputs:
  response:
    type: string
sample:
  customer_name: Alice
  query: How do I reset my password?
  priority: normal
---
Hello {~prompty.var name="customer_name" /~},

Thank you for reaching out. I understand you need help with: {~prompty.var name="query" /~}

{~prompty.if eval="priority == 'high'"~}
I'm treating this as a priority request and will ensure quick resolution.
{~prompty.else~}
I'll do my best to help you today.
{~/prompty.if~}

Best regards,
Customer Support`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	// Verify config
	config := tmpl.InferenceConfig()
	require.NotNil(t, config)
	assert.Equal(t, "customer-support-agent", config.Name)
	assert.Equal(t, "1.0.0", config.Version)
	assert.True(t, config.HasModel())
	assert.True(t, config.HasInputs())
	assert.True(t, config.HasOutputs())
	assert.True(t, config.HasSample())

	// Validate inputs with sample data
	err = config.ValidateInputs(config.GetSampleData())
	assert.NoError(t, err)

	// Execute with provided data
	result, err := tmpl.Execute(context.Background(), map[string]any{
		"customer_name": "Bob",
		"query":         "I need to update my billing address",
		"priority":      "high",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "Hello Bob")
	assert.Contains(t, result, "update my billing address")
	assert.Contains(t, result, "priority request")
}

// TestE2E_InferenceConfigJSON tests JSON serialization
func TestE2E_InferenceConfigJSON(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `---
name: json-test
version: 1.0.0
---
Template`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	config := tmpl.InferenceConfig()
	require.NotNil(t, config)

	jsonStr, err := config.JSON()
	require.NoError(t, err)
	assert.Contains(t, jsonStr, `"name":"json-test"`)

	prettyJSON, err := config.JSONPretty()
	require.NoError(t, err)
	assert.Contains(t, prettyJSON, "\n")
	assert.Contains(t, prettyJSON, `"name": "json-test"`)
}

// TestE2E_InferenceConfigYAML tests YAML serialization
func TestE2E_InferenceConfigYAML(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `---
name: yaml-test
version: 2.0.0
model:
  name: gpt-4
---
Template`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	config := tmpl.InferenceConfig()
	require.NotNil(t, config)

	yamlStr, err := config.YAML()
	require.NoError(t, err)
	assert.Contains(t, yamlStr, "name: yaml-test")
	assert.Contains(t, yamlStr, "version: 2.0.0")
}

// TestE2E_LegacyJSONConfigBlockError tests that legacy JSON config blocks produce helpful error
func TestE2E_LegacyJSONConfigBlockError(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `{~prompty.config~}
{"name": "test"}
{~/prompty.config~}
Template`

	_, err = engine.Parse(source)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "legacy JSON config block detected")
	assert.Contains(t, err.Error(), "YAML frontmatter")
}

// TestE2E_FrontmatterWithNewFeatures tests the new features (response_format, tools, etc.)
func TestE2E_FrontmatterWithNewFeatures(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `---
name: advanced-template
model:
  api: chat
  name: gpt-4
  response_format:
    type: json_schema
    json_schema:
      name: entities
      strict: true
      schema:
        type: object
        properties:
          people:
            type: array
          places:
            type: array
        required:
          - people
          - places
  tools:
    - type: function
      function:
        name: get_weather
        description: Get current weather
        parameters:
          type: object
          properties:
            location:
              type: string
          required:
            - location
        strict: true
  tool_choice: auto
  streaming:
    enabled: true
  context_window: 8192
retry:
  max_attempts: 3
  backoff: exponential
cache:
  system_prompt: true
  ttl: 3600
---
Template`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	config := tmpl.InferenceConfig()
	require.NotNil(t, config)

	// Test response format
	assert.True(t, config.HasResponseFormat())
	rf := config.GetResponseFormat()
	assert.Equal(t, "json_schema", rf.Type)
	assert.NotNil(t, rf.JSONSchema)
	assert.Equal(t, "entities", rf.JSONSchema.Name)
	assert.True(t, rf.JSONSchema.Strict)

	// Test tools
	assert.True(t, config.HasTools())
	tools := config.GetTools()
	require.Len(t, tools, 1)
	assert.Equal(t, "function", tools[0].Type)
	assert.Equal(t, "get_weather", tools[0].Function.Name)
	assert.True(t, tools[0].Function.Strict)

	// Test tool choice
	assert.Equal(t, "auto", config.GetToolChoice())

	// Test streaming
	assert.True(t, config.HasStreaming())
	streaming := config.GetStreaming()
	assert.True(t, streaming.Enabled)

	// Test context window
	cw, ok := config.GetContextWindow()
	assert.True(t, ok)
	assert.Equal(t, 8192, cw)

	// Test retry config
	assert.True(t, config.HasRetry())
	retry := config.GetRetry()
	assert.Equal(t, 3, retry.MaxAttempts)
	assert.Equal(t, "exponential", retry.Backoff)

	// Test cache config
	assert.True(t, config.HasCache())
	cache := config.GetCache()
	assert.True(t, cache.SystemPrompt)
	assert.Equal(t, 3600, cache.TTL)
}

// TestE2E_MessageTagBasic tests basic message tag functionality
func TestE2E_MessageTagBasic(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `---
name: chat-test
model:
  api: chat
---
{~prompty.message role="system"~}
You are a helpful assistant.
{~/prompty.message~}

{~prompty.message role="user"~}
Hello!
{~/prompty.message~}`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	result, err := tmpl.Execute(context.Background(), nil)
	require.NoError(t, err)

	// Extract messages
	messages := ExtractMessagesFromOutput(result)
	require.Len(t, messages, 2)

	assert.Equal(t, "system", messages[0].Role)
	assert.Contains(t, messages[0].Content, "helpful assistant")

	assert.Equal(t, "user", messages[1].Role)
	assert.Contains(t, messages[1].Content, "Hello!")
}

// TestE2E_MessageTagWithVariables tests message tags with variable interpolation
func TestE2E_MessageTagWithVariables(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `---
name: dynamic-chat
---
{~prompty.message role="system"~}
You are a {~prompty.var name="assistant_type" /~} for {~prompty.var name="company" /~}.
{~/prompty.message~}

{~prompty.message role="user"~}
{~prompty.var name="query" /~}
{~/prompty.message~}`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	messages, err := tmpl.ExecuteAndExtractMessages(context.Background(), map[string]any{
		"assistant_type": "customer support agent",
		"company":        "Acme Corp",
		"query":          "How do I reset my password?",
	})
	require.NoError(t, err)
	require.Len(t, messages, 2)

	assert.Equal(t, "system", messages[0].Role)
	assert.Contains(t, messages[0].Content, "customer support agent")
	assert.Contains(t, messages[0].Content, "Acme Corp")

	assert.Equal(t, "user", messages[1].Role)
	assert.Contains(t, messages[1].Content, "reset my password")
}

// TestE2E_MessageTagWithConditionals tests message tags with conditionals inside
func TestE2E_MessageTagWithConditionals(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `---
name: conditional-chat
---
{~prompty.message role="system"~}
You are a helpful assistant.
{~prompty.if eval="use_guidelines"~}
Always follow the safety guidelines.
{~/prompty.if~}
{~/prompty.message~}

{~prompty.message role="user"~}
Hello
{~/prompty.message~}`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	// With guidelines
	messages, err := tmpl.ExecuteAndExtractMessages(context.Background(), map[string]any{
		"use_guidelines": true,
	})
	require.NoError(t, err)
	require.Len(t, messages, 2)
	assert.Contains(t, messages[0].Content, "safety guidelines")

	// Without guidelines
	messages, err = tmpl.ExecuteAndExtractMessages(context.Background(), map[string]any{
		"use_guidelines": false,
	})
	require.NoError(t, err)
	require.Len(t, messages, 2)
	assert.NotContains(t, messages[0].Content, "safety guidelines")
}
