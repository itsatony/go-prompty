package prompty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInferenceConfig_Empty(t *testing.T) {
	config, err := ParseInferenceConfig("")
	require.NoError(t, err)
	assert.Nil(t, config)
}

func TestParseInferenceConfig_Basic(t *testing.T) {
	jsonData := `{
		"name": "test-template",
		"description": "A test template",
		"version": "1.0.0",
		"authors": ["Alice", "Bob"]
	}`

	config, err := ParseInferenceConfig(jsonData)
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "test-template", config.Name)
	assert.Equal(t, "A test template", config.Description)
	assert.Equal(t, "1.0.0", config.Version)
	assert.Equal(t, []string{"Alice", "Bob"}, config.Authors)
}

func TestParseInferenceConfig_FullModel(t *testing.T) {
	jsonData := `{
		"name": "chat-template",
		"model": {
			"api": "chat",
			"provider": "openai",
			"name": "gpt-4",
			"parameters": {
				"temperature": 0.7,
				"max_tokens": 2048,
				"top_p": 0.9,
				"frequency_penalty": 0.5,
				"presence_penalty": 0.5,
				"stop": ["\n\n", "END"],
				"seed": 42
			}
		}
	}`

	config, err := ParseInferenceConfig(jsonData)
	require.NoError(t, err)
	require.NotNil(t, config)
	require.NotNil(t, config.Model)

	assert.Equal(t, "chat", config.Model.API)
	assert.Equal(t, "openai", config.Model.Provider)
	assert.Equal(t, "gpt-4", config.Model.Name)

	require.NotNil(t, config.Model.Parameters)
	temp, ok := config.GetTemperature()
	assert.True(t, ok)
	assert.Equal(t, 0.7, temp)

	maxTokens, ok := config.GetMaxTokens()
	assert.True(t, ok)
	assert.Equal(t, 2048, maxTokens)

	topP, ok := config.GetTopP()
	assert.True(t, ok)
	assert.Equal(t, 0.9, topP)

	freqPenalty, ok := config.GetFrequencyPenalty()
	assert.True(t, ok)
	assert.Equal(t, 0.5, freqPenalty)

	presencePenalty, ok := config.GetPresencePenalty()
	assert.True(t, ok)
	assert.Equal(t, 0.5, presencePenalty)

	assert.Equal(t, []string{"\n\n", "END"}, config.GetStopSequences())

	seed, ok := config.GetSeed()
	assert.True(t, ok)
	assert.Equal(t, int64(42), seed)
}

func TestParseInferenceConfig_InputsOutputs(t *testing.T) {
	jsonData := `{
		"inputs": {
			"name": {"type": "string", "required": true, "description": "User name"},
			"age": {"type": "number", "required": false, "default": 0}
		},
		"outputs": {
			"greeting": {"type": "string", "description": "The greeting message"}
		}
	}`

	config, err := ParseInferenceConfig(jsonData)
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.True(t, config.HasInputs())
	assert.True(t, config.HasOutputs())

	nameInput := config.Inputs["name"]
	require.NotNil(t, nameInput)
	assert.Equal(t, "string", nameInput.Type)
	assert.True(t, nameInput.Required)
	assert.Equal(t, "User name", nameInput.Description)

	ageInput := config.Inputs["age"]
	require.NotNil(t, ageInput)
	assert.Equal(t, "number", ageInput.Type)
	assert.False(t, ageInput.Required)

	greetingOutput := config.Outputs["greeting"]
	require.NotNil(t, greetingOutput)
	assert.Equal(t, "string", greetingOutput.Type)
}

func TestParseInferenceConfig_Sample(t *testing.T) {
	jsonData := `{
		"sample": {
			"name": "Alice",
			"count": 42,
			"active": true
		}
	}`

	config, err := ParseInferenceConfig(jsonData)
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.True(t, config.HasSample())
	sample := config.GetSampleData()
	assert.Equal(t, "Alice", sample["name"])
	assert.Equal(t, float64(42), sample["count"]) // JSON numbers are float64
	assert.Equal(t, true, sample["active"])
}

func TestParseInferenceConfig_InvalidJSON(t *testing.T) {
	_, err := ParseInferenceConfig(`{invalid json}`)
	require.Error(t, err)
}

func TestInferenceConfig_NilSafe(t *testing.T) {
	var config *InferenceConfig

	// All getters should be nil-safe
	temp, ok := config.GetTemperature()
	assert.False(t, ok)
	assert.Equal(t, float64(0), temp)

	maxTokens, ok := config.GetMaxTokens()
	assert.False(t, ok)
	assert.Equal(t, 0, maxTokens)

	assert.Empty(t, config.GetModelName())
	assert.Empty(t, config.GetAPIType())
	assert.Empty(t, config.GetProvider())
	assert.Nil(t, config.GetSampleData())
	assert.Nil(t, config.GetStopSequences())

	assert.False(t, config.HasModel())
	assert.False(t, config.HasInputs())
	assert.False(t, config.HasOutputs())
	assert.False(t, config.HasSample())
}

func TestInferenceConfig_ValidateInputs(t *testing.T) {
	config := &InferenceConfig{
		Inputs: map[string]*InputDef{
			"name":    {Type: "string", Required: true},
			"count":   {Type: "number", Required: true},
			"active":  {Type: "boolean", Required: false},
			"tags":    {Type: "array", Required: false},
			"options": {Type: "object", Required: false},
		},
	}

	tests := []struct {
		name    string
		data    map[string]any
		wantErr bool
	}{
		{
			name:    "valid all required",
			data:    map[string]any{"name": "Alice", "count": 42},
			wantErr: false,
		},
		{
			name:    "valid with optional",
			data:    map[string]any{"name": "Alice", "count": 42, "active": true},
			wantErr: false,
		},
		{
			name:    "missing required name",
			data:    map[string]any{"count": 42},
			wantErr: true,
		},
		{
			name:    "missing required count",
			data:    map[string]any{"name": "Alice"},
			wantErr: true,
		},
		{
			name:    "wrong type for name",
			data:    map[string]any{"name": 123, "count": 42},
			wantErr: true,
		},
		{
			name:    "wrong type for count",
			data:    map[string]any{"name": "Alice", "count": "not a number"},
			wantErr: true,
		},
		{
			name:    "array type valid",
			data:    map[string]any{"name": "Alice", "count": 42, "tags": []string{"a", "b"}},
			wantErr: false,
		},
		{
			name:    "object type valid",
			data:    map[string]any{"name": "Alice", "count": 42, "options": map[string]any{"key": "value"}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.ValidateInputs(tt.data)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInferenceConfig_ValidateInputs_NilConfig(t *testing.T) {
	var config *InferenceConfig
	err := config.ValidateInputs(map[string]any{"anything": "goes"})
	assert.NoError(t, err)
}

func TestModelParameters_ToMap(t *testing.T) {
	temp := 0.7
	maxTokens := 2048
	topP := 0.9
	freqPenalty := 0.5
	presPenalty := 0.3
	seed := int64(42)

	params := &ModelParameters{
		Temperature:      &temp,
		MaxTokens:        &maxTokens,
		TopP:             &topP,
		FrequencyPenalty: &freqPenalty,
		PresencePenalty:  &presPenalty,
		Stop:             []string{"END"},
		Seed:             &seed,
	}

	m := params.ToMap()
	assert.Equal(t, 0.7, m[ParamKeyTemperature])
	assert.Equal(t, 2048, m[ParamKeyMaxTokens])
	assert.Equal(t, 0.9, m[ParamKeyTopP])
	assert.Equal(t, 0.5, m[ParamKeyFrequencyPenalty])
	assert.Equal(t, 0.3, m[ParamKeyPresencePenalty])
	assert.Equal(t, []string{"END"}, m[ParamKeyStop])
	assert.Equal(t, int64(42), m[ParamKeySeed])
}

func TestModelParameters_ToMap_Partial(t *testing.T) {
	temp := 0.5
	params := &ModelParameters{
		Temperature: &temp,
	}

	m := params.ToMap()
	assert.Equal(t, 0.5, m[ParamKeyTemperature])
	assert.Nil(t, m[ParamKeyMaxTokens])
	assert.Len(t, m, 1)
}

func TestModelParameters_ToMap_Nil(t *testing.T) {
	var params *ModelParameters
	m := params.ToMap()
	assert.Nil(t, m)
}

func TestInferenceConfig_JSON(t *testing.T) {
	config := &InferenceConfig{
		Name:    "test",
		Version: "1.0.0",
	}

	jsonStr, err := config.JSON()
	require.NoError(t, err)
	assert.Contains(t, jsonStr, `"name":"test"`)
	assert.Contains(t, jsonStr, `"version":"1.0.0"`)
}

func TestInferenceConfig_JSONPretty(t *testing.T) {
	config := &InferenceConfig{
		Name:    "test",
		Version: "1.0.0",
	}

	jsonStr, err := config.JSONPretty()
	require.NoError(t, err)
	assert.Contains(t, jsonStr, `"name": "test"`)
	assert.Contains(t, jsonStr, "\n")
}

func TestInferenceConfig_JSON_Nil(t *testing.T) {
	var config *InferenceConfig
	jsonStr, err := config.JSON()
	require.NoError(t, err)
	assert.Empty(t, jsonStr)
}

func TestValidateInputType_AllTypes(t *testing.T) {
	tests := []struct {
		name         string
		value        any
		expectedType string
		wantErr      bool
	}{
		// String
		{"string valid", "hello", SchemaTypeString, false},
		{"string invalid", 123, SchemaTypeString, true},

		// Number
		{"number int", 42, SchemaTypeNumber, false},
		{"number int64", int64(42), SchemaTypeNumber, false},
		{"number float64", 3.14, SchemaTypeNumber, false},
		{"number float32", float32(3.14), SchemaTypeNumber, false},
		{"number invalid", "not a number", SchemaTypeNumber, true},

		// Boolean
		{"boolean true", true, SchemaTypeBoolean, false},
		{"boolean false", false, SchemaTypeBoolean, false},
		{"boolean invalid", "true", SchemaTypeBoolean, true},

		// Array
		{"array any", []any{1, 2, 3}, SchemaTypeArray, false},
		{"array string", []string{"a", "b"}, SchemaTypeArray, false},
		{"array int", []int{1, 2, 3}, SchemaTypeArray, false},
		{"array float64", []float64{1.1, 2.2}, SchemaTypeArray, false},
		{"array invalid", "not an array", SchemaTypeArray, true},

		// Object
		{"object map any", map[string]any{"k": "v"}, SchemaTypeObject, false},
		{"object map string", map[string]string{"k": "v"}, SchemaTypeObject, false},
		{"object invalid", "not an object", SchemaTypeObject, true},

		// Unknown type
		{"unknown type", "anything", "unknown", false},

		// Nil
		{"nil value", nil, SchemaTypeString, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInputType("test", tt.value, tt.expectedType)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJSONSchemaSpec_AdditionalProperties(t *testing.T) {
	falseVal := false
	trueVal := true

	tests := []struct {
		name     string
		spec     *JSONSchemaSpec
		expected *bool
	}{
		{
			name: "additionalProperties not set",
			spec: &JSONSchemaSpec{
				Name:   "test",
				Schema: map[string]any{},
			},
			expected: nil,
		},
		{
			name: "additionalProperties set to false",
			spec: &JSONSchemaSpec{
				Name:                 "test",
				Schema:               map[string]any{},
				AdditionalProperties: &falseVal,
			},
			expected: &falseVal,
		},
		{
			name: "additionalProperties set to true",
			spec: &JSONSchemaSpec{
				Name:                 "test",
				Schema:               map[string]any{},
				AdditionalProperties: &trueVal,
			},
			expected: &trueVal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.spec.AdditionalProperties)
		})
	}
}

func TestJSONSchemaSpec_PropertyOrdering(t *testing.T) {
	spec := &JSONSchemaSpec{
		Name:             "test",
		Schema:           map[string]any{},
		PropertyOrdering: []string{"first", "second", "third"},
	}

	assert.Equal(t, []string{"first", "second", "third"}, spec.PropertyOrdering)
}

func TestEnumConstraint(t *testing.T) {
	enum := &EnumConstraint{
		Values:      []string{"positive", "negative", "neutral"},
		Description: "Sentiment classification",
	}

	assert.Len(t, enum.Values, 3)
	assert.Contains(t, enum.Values, "positive")
	assert.Contains(t, enum.Values, "negative")
	assert.Contains(t, enum.Values, "neutral")
	assert.Equal(t, "Sentiment classification", enum.Description)
}

func TestResponseFormat_WithEnum(t *testing.T) {
	rf := &ResponseFormat{
		Type: ResponseFormatEnum,
		Enum: &EnumConstraint{
			Values: []string{"yes", "no"},
		},
	}

	assert.Equal(t, ResponseFormatEnum, rf.Type)
	require.NotNil(t, rf.Enum)
	assert.Equal(t, []string{"yes", "no"}, rf.Enum.Values)
}

func TestOutputFormat(t *testing.T) {
	of := &OutputFormat{
		Format: &OutputFormatSpec{
			Type: ResponseFormatJSONSchema,
			Schema: map[string]any{
				SchemaKeyType: SchemaTypeObject,
				SchemaKeyProperties: map[string]any{
					"result": map[string]any{SchemaKeyType: SchemaTypeString},
				},
			},
		},
	}

	assert.Equal(t, ResponseFormatJSONSchema, of.Format.Type)
	require.NotNil(t, of.Format.Schema)
}

func TestGuidedDecoding(t *testing.T) {
	tests := []struct {
		name string
		gd   *GuidedDecoding
	}{
		{
			name: "json constraint",
			gd: &GuidedDecoding{
				Backend: GuidedBackendXGrammar,
				JSON: map[string]any{
					SchemaKeyType: SchemaTypeObject,
				},
			},
		},
		{
			name: "regex constraint",
			gd: &GuidedDecoding{
				Regex: "^[a-z]+$",
			},
		},
		{
			name: "choice constraint",
			gd: &GuidedDecoding{
				Choice: []string{"option1", "option2"},
			},
		},
		{
			name: "grammar constraint",
			gd: &GuidedDecoding{
				Grammar: "S -> 'a' | 'b'",
			},
		},
		{
			name: "with whitespace pattern",
			gd: &GuidedDecoding{
				JSON:              map[string]any{},
				WhitespacePattern: `\s+`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotNil(t, tt.gd)
		})
	}
}

func TestModelConfig_WithOutputFormat(t *testing.T) {
	mc := &ModelConfig{
		Provider: ProviderAnthropic,
		Name:     "claude-3-opus",
		OutputFormat: &OutputFormat{
			Format: &OutputFormatSpec{
				Type: ResponseFormatJSONSchema,
				Schema: map[string]any{
					SchemaKeyType: SchemaTypeObject,
				},
			},
		},
	}

	require.NotNil(t, mc.OutputFormat)
	assert.Equal(t, ResponseFormatJSONSchema, mc.OutputFormat.Format.Type)
}

func TestModelConfig_WithGuidedDecoding(t *testing.T) {
	mc := &ModelConfig{
		Provider: ProviderVLLM,
		Name:     "meta-llama/Llama-2-7b-hf",
		GuidedDecoding: &GuidedDecoding{
			Backend: GuidedBackendOutlines,
			JSON: map[string]any{
				SchemaKeyType: SchemaTypeObject,
			},
		},
	}

	require.NotNil(t, mc.GuidedDecoding)
	assert.Equal(t, GuidedBackendOutlines, mc.GuidedDecoding.Backend)
}

func TestInferenceConfig_GetOutputFormat(t *testing.T) {
	of := &OutputFormat{
		Format: &OutputFormatSpec{
			Type: ResponseFormatJSONSchema,
		},
	}

	config := &InferenceConfig{
		Model: &ModelConfig{
			OutputFormat: of,
		},
	}

	result := config.GetOutputFormat()
	assert.Equal(t, of, result)
}

func TestInferenceConfig_GetOutputFormat_Nil(t *testing.T) {
	var config *InferenceConfig
	assert.Nil(t, config.GetOutputFormat())

	config = &InferenceConfig{}
	assert.Nil(t, config.GetOutputFormat())

	config = &InferenceConfig{Model: &ModelConfig{}}
	assert.Nil(t, config.GetOutputFormat())
}

func TestInferenceConfig_GetGuidedDecoding(t *testing.T) {
	gd := &GuidedDecoding{
		Regex: "^test$",
	}

	config := &InferenceConfig{
		Model: &ModelConfig{
			GuidedDecoding: gd,
		},
	}

	result := config.GetGuidedDecoding()
	assert.Equal(t, gd, result)
}

func TestInferenceConfig_GetGuidedDecoding_Nil(t *testing.T) {
	var config *InferenceConfig
	assert.Nil(t, config.GetGuidedDecoding())
}

func TestInferenceConfig_HasOutputFormat(t *testing.T) {
	tests := []struct {
		name     string
		config   *InferenceConfig
		expected bool
	}{
		{"nil config", nil, false},
		{"empty config", &InferenceConfig{}, false},
		{"no model", &InferenceConfig{Model: nil}, false},
		{"no output format", &InferenceConfig{Model: &ModelConfig{}}, false},
		{"with output format", &InferenceConfig{Model: &ModelConfig{OutputFormat: &OutputFormat{}}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.HasOutputFormat())
		})
	}
}

func TestInferenceConfig_HasGuidedDecoding(t *testing.T) {
	tests := []struct {
		name     string
		config   *InferenceConfig
		expected bool
	}{
		{"nil config", nil, false},
		{"empty config", &InferenceConfig{}, false},
		{"no model", &InferenceConfig{Model: nil}, false},
		{"no guided decoding", &InferenceConfig{Model: &ModelConfig{}}, false},
		{"with guided decoding", &InferenceConfig{Model: &ModelConfig{GuidedDecoding: &GuidedDecoding{}}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.HasGuidedDecoding())
		})
	}
}

func TestInferenceConfig_GetEffectiveProvider(t *testing.T) {
	tests := []struct {
		name     string
		config   *InferenceConfig
		expected string
	}{
		{"nil config", nil, ""},
		{"empty config", &InferenceConfig{}, ""},
		{"explicit provider", &InferenceConfig{Model: &ModelConfig{Provider: ProviderOpenAI}}, ProviderOpenAI},
		{"infer from output format", &InferenceConfig{Model: &ModelConfig{OutputFormat: &OutputFormat{}}}, ProviderAnthropic},
		{"infer from guided decoding", &InferenceConfig{Model: &ModelConfig{GuidedDecoding: &GuidedDecoding{}}}, ProviderVLLM},
		{"infer from model name gpt", &InferenceConfig{Model: &ModelConfig{Name: "gpt-4"}}, ProviderOpenAI},
		{"infer from model name claude", &InferenceConfig{Model: &ModelConfig{Name: "claude-3-opus"}}, ProviderAnthropic},
		{"infer from model name gemini", &InferenceConfig{Model: &ModelConfig{Name: "gemini-pro"}}, ProviderGemini},
		{"explicit provider overrides inference", &InferenceConfig{Model: &ModelConfig{Provider: ProviderAzure, Name: "gpt-4"}}, ProviderAzure},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.GetEffectiveProvider())
		})
	}
}

func TestParseYAMLInferenceConfig_WithOutputFormat(t *testing.T) {
	yaml := `
name: test-template
model:
  provider: anthropic
  name: claude-3-opus
  output_format:
    format:
      type: json_schema
      schema:
        type: object
        properties:
          result:
            type: string
`
	config, err := ParseYAMLInferenceConfig(yaml)
	require.NoError(t, err)
	require.NotNil(t, config)
	require.NotNil(t, config.Model)
	require.NotNil(t, config.Model.OutputFormat)
	assert.Equal(t, ResponseFormatJSONSchema, config.Model.OutputFormat.Format.Type)
}

func TestParseYAMLInferenceConfig_WithGuidedDecoding(t *testing.T) {
	yaml := `
name: test-template
model:
  provider: vllm
  name: meta-llama/Llama-2-7b-hf
  guided_decoding:
    backend: xgrammar
    json:
      type: object
      properties:
        output:
          type: string
`
	config, err := ParseYAMLInferenceConfig(yaml)
	require.NoError(t, err)
	require.NotNil(t, config)
	require.NotNil(t, config.Model)
	require.NotNil(t, config.Model.GuidedDecoding)
	assert.Equal(t, GuidedBackendXGrammar, config.Model.GuidedDecoding.Backend)
}

func TestParseYAMLInferenceConfig_WithEnumConstraint(t *testing.T) {
	yaml := `
name: sentiment-analysis
model:
  name: gpt-4o
  response_format:
    type: enum
    enum:
      values:
        - positive
        - negative
        - neutral
      description: Sentiment classification
`
	config, err := ParseYAMLInferenceConfig(yaml)
	require.NoError(t, err)
	require.NotNil(t, config)
	require.NotNil(t, config.Model)
	require.NotNil(t, config.Model.ResponseFormat)
	assert.Equal(t, ResponseFormatEnum, config.Model.ResponseFormat.Type)
	require.NotNil(t, config.Model.ResponseFormat.Enum)
	assert.Len(t, config.Model.ResponseFormat.Enum.Values, 3)
}

func TestParseYAMLInferenceConfig_WithPropertyOrdering(t *testing.T) {
	yaml := `
name: gemini-template
model:
  provider: gemini
  name: gemini-2-5-pro
  response_format:
    type: json_schema
    json_schema:
      name: ordered_data
      schema:
        type: object
        properties:
          first:
            type: string
          second:
            type: number
      propertyOrdering:
        - first
        - second
`
	config, err := ParseYAMLInferenceConfig(yaml)
	require.NoError(t, err)
	require.NotNil(t, config)
	require.NotNil(t, config.Model.ResponseFormat)
	require.NotNil(t, config.Model.ResponseFormat.JSONSchema)
	assert.Equal(t, []string{"first", "second"}, config.Model.ResponseFormat.JSONSchema.PropertyOrdering)
}
