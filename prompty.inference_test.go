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
