package prompty

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_ConfigBlockBasicParsing tests basic config block parsing
func TestE2E_ConfigBlockBasicParsing(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `{~prompty.config~}
{
  "name": "test-template",
  "description": "A test template",
  "version": "1.0.0"
}
{~/prompty.config~}
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

	// Template body should be just the content after config block
	assert.Equal(t, "Hello World!", tmpl.TemplateBody())

	// Source should contain the full template including config block
	assert.Contains(t, tmpl.Source(), "{~prompty.config~}")
}

// TestE2E_ConfigBlockWithModel tests config block with model configuration
func TestE2E_ConfigBlockWithModel(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `{~prompty.config~}
{
  "name": "chat-template",
  "model": {
    "api": "chat",
    "provider": "openai",
    "name": "gpt-4",
    "parameters": {
      "temperature": 0.7,
      "max_tokens": 2048,
      "top_p": 0.9,
      "stop": ["\n\n", "END"]
    }
  }
}
{~/prompty.config~}
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

// TestE2E_ConfigBlockWithVariables tests config block with template variables
func TestE2E_ConfigBlockWithVariables(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `{~prompty.config~}
{
  "name": "dynamic-template",
  "model": {
    "name": "{~prompty.var name="model_name" default="gpt-4" /~}"
  }
}
{~/prompty.config~}
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

// TestE2E_ConfigBlockWithEnvVars tests config block with environment variables
func TestE2E_ConfigBlockWithEnvVars(t *testing.T) {
	// Set up test environment variable
	testAPIKey := "sk-test-key-12345"
	os.Setenv("TEST_API_KEY", testAPIKey)
	defer os.Unsetenv("TEST_API_KEY")

	engine, err := New()
	require.NoError(t, err)

	source := `{~prompty.config~}
{
  "name": "env-template",
  "description": "API Key: {~prompty.env name="TEST_API_KEY" /~}"
}
{~/prompty.config~}
Template content`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	config := tmpl.InferenceConfig()
	require.NotNil(t, config)

	// The description should contain the resolved env var
	assert.Equal(t, "API Key: "+testAPIKey, config.Description)
}

// TestE2E_ConfigBlockTemplateExecution tests that templates with config blocks execute correctly
func TestE2E_ConfigBlockTemplateExecution(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `{~prompty.config~}
{
  "name": "greet-template",
  "sample": {
    "name": "World"
  }
}
{~/prompty.config~}
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

// TestE2E_ConfigBlockInputValidation tests input validation
func TestE2E_ConfigBlockInputValidation(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `{~prompty.config~}
{
  "inputs": {
    "name": {"type": "string", "required": true},
    "age": {"type": "number", "required": true},
    "active": {"type": "boolean", "required": false}
  }
}
{~/prompty.config~}
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

// TestE2E_ConfigBlockSampleData tests sample data extraction
func TestE2E_ConfigBlockSampleData(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `{~prompty.config~}
{
  "sample": {
    "user": "Alice",
    "items": ["apple", "banana"],
    "count": 42,
    "active": true
  }
}
{~/prompty.config~}
Template`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	config := tmpl.InferenceConfig()
	require.NotNil(t, config)
	assert.True(t, config.HasSample())

	sample := config.GetSampleData()
	assert.Equal(t, "Alice", sample["user"])
	assert.Equal(t, float64(42), sample["count"]) // JSON numbers are float64
	assert.Equal(t, true, sample["active"])
}

// TestE2E_ConfigBlockStorageRoundtrip tests storing and retrieving templates with config
func TestE2E_ConfigBlockStorageRoundtrip(t *testing.T) {
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

	source := `{~prompty.config~}
{
  "name": "stored-template",
  "version": "1.0.0",
  "model": {
    "api": "chat",
    "name": "gpt-4"
  }
}
{~/prompty.config~}
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

// TestE2E_NoConfigBlockBackwardCompatible tests that templates without config blocks work
func TestE2E_NoConfigBlockBackwardCompatible(t *testing.T) {
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

// TestE2E_ConfigBlockMalformedJSON tests error handling for malformed JSON
func TestE2E_ConfigBlockMalformedJSON(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `{~prompty.config~}
{invalid json}
{~/prompty.config~}
Template`

	_, err = engine.Parse(source)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgConfigBlockParse)
}

// TestE2E_ConfigBlockUnclosed tests error handling for unclosed config blocks
func TestE2E_ConfigBlockUnclosed(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `{~prompty.config~}
{"name": "test"}
Template body without closing tag`

	_, err = engine.Parse(source)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgConfigBlockExtract)
}

// TestE2E_ConfigBlockWithAuthors tests config with authors array
func TestE2E_ConfigBlockWithAuthors(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `{~prompty.config~}
{
  "name": "team-template",
  "authors": ["alice@example.com", "bob@example.com"],
  "tags": ["production", "customer-service"]
}
{~/prompty.config~}
Template`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	config := tmpl.InferenceConfig()
	require.NotNil(t, config)

	assert.Equal(t, []string{"alice@example.com", "bob@example.com"}, config.Authors)
	assert.Equal(t, []string{"production", "customer-service"}, config.Tags)
}

// TestE2E_ConfigBlockModelParametersToMap tests model parameters ToMap conversion
func TestE2E_ConfigBlockModelParametersToMap(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `{~prompty.config~}
{
  "model": {
    "parameters": {
      "temperature": 0.5,
      "max_tokens": 1024,
      "top_p": 0.95,
      "frequency_penalty": 0.1,
      "presence_penalty": 0.2,
      "seed": 12345
    }
  }
}
{~/prompty.config~}
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

// TestE2E_ConfigBlockWithOutputs tests config with outputs schema
func TestE2E_ConfigBlockWithOutputs(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `{~prompty.config~}
{
  "outputs": {
    "response": {"type": "string", "description": "The model response"},
    "confidence": {"type": "number", "description": "Confidence score"}
  }
}
{~/prompty.config~}
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

// TestE2E_ConfigBlockWithLeadingWhitespace tests config block after whitespace
func TestE2E_ConfigBlockWithLeadingWhitespace(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	source := `

  {~prompty.config~}
{"name": "whitespace-test"}
{~/prompty.config~}
Template`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	assert.True(t, tmpl.HasInferenceConfig())
	assert.Equal(t, "whitespace-test", tmpl.InferenceConfig().Name)
}

// TestE2E_ConfigBlockInMiddleNotExtracted tests that config blocks in middle are not extracted
func TestE2E_ConfigBlockInMiddleNotExtracted(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	// Config block after content should be treated as regular text
	source := `Hello World
{~prompty.config~}
{"name": "middle-config"}
{~/prompty.config~}
More content`

	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	// Should NOT have config because it's not at the start
	assert.False(t, tmpl.HasInferenceConfig())
	assert.Equal(t, source, tmpl.TemplateBody())
}

// TestE2E_ConfigBlockEnvVarWithDefault tests env var with default value
func TestE2E_ConfigBlockEnvVarWithDefault(t *testing.T) {
	// Make sure env var is not set
	os.Unsetenv("MISSING_VAR_FOR_TEST")

	engine, err := New()
	require.NoError(t, err)

	source := `{~prompty.config~}
{
  "name": "env-default-test",
  "description": "{~prompty.env name="MISSING_VAR_FOR_TEST" default="default-value" /~}"
}
{~/prompty.config~}
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

	source := `{~prompty.config~}
{
  "name": "customer-support-agent",
  "description": "Handles customer inquiries with empathetic responses",
  "version": "1.0.0",
  "authors": ["support-team@example.com"],
  "tags": ["production", "customer-service"],
  "model": {
    "api": "chat",
    "provider": "openai",
    "name": "gpt-4",
    "parameters": {
      "temperature": 0.7,
      "max_tokens": 2048
    }
  },
  "inputs": {
    "customer_name": {"type": "string", "required": true},
    "query": {"type": "string", "required": true},
    "priority": {"type": "string", "required": false}
  },
  "outputs": {
    "response": {"type": "string"}
  },
  "sample": {
    "customer_name": "Alice",
    "query": "How do I reset my password?",
    "priority": "normal"
  }
}
{~/prompty.config~}
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

	source := `{~prompty.config~}
{
  "name": "json-test",
  "version": "1.0.0"
}
{~/prompty.config~}
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
