package prompty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateJSONSchema_ValidObject(t *testing.T) {
	schema := map[string]any{
		SchemaKeyType: SchemaTypeObject,
		SchemaKeyProperties: map[string]any{
			"name":  map[string]any{SchemaKeyType: SchemaTypeString},
			"email": map[string]any{SchemaKeyType: SchemaTypeString},
		},
		SchemaKeyRequired:             []any{"name", "email"},
		SchemaKeyAdditionalProperties: false,
	}

	result := ValidateJSONSchema(schema)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidateJSONSchema_MissingType(t *testing.T) {
	schema := map[string]any{
		SchemaKeyProperties: map[string]any{
			"name": map[string]any{SchemaKeyType: SchemaTypeString},
		},
	}

	result := ValidateJSONSchema(schema)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], ErrMsgSchemaMissingType)
}

func TestValidateJSONSchema_InvalidType(t *testing.T) {
	schema := map[string]any{
		SchemaKeyType: 123, // Should be string
	}

	result := ValidateJSONSchema(schema)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], ErrMsgSchemaInvalidType)
}

func TestValidateJSONSchema_NestedObject(t *testing.T) {
	schema := map[string]any{
		SchemaKeyType: SchemaTypeObject,
		SchemaKeyProperties: map[string]any{
			"user": map[string]any{
				SchemaKeyType: SchemaTypeObject,
				SchemaKeyProperties: map[string]any{
					"name": map[string]any{SchemaKeyType: SchemaTypeString},
				},
				SchemaKeyAdditionalProperties: false,
			},
		},
		SchemaKeyAdditionalProperties: false,
	}

	result := ValidateJSONSchema(schema)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidateJSONSchema_ArraySchema(t *testing.T) {
	schema := map[string]any{
		SchemaKeyType: SchemaTypeArray,
		SchemaKeyItems: map[string]any{
			SchemaKeyType: SchemaTypeObject,
			SchemaKeyProperties: map[string]any{
				"id": map[string]any{SchemaKeyType: SchemaTypeNumber},
			},
			SchemaKeyAdditionalProperties: false,
		},
	}

	result := ValidateJSONSchema(schema)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidateJSONSchema_MissingAdditionalProperties(t *testing.T) {
	schema := map[string]any{
		SchemaKeyType: SchemaTypeObject,
		SchemaKeyProperties: map[string]any{
			"name": map[string]any{SchemaKeyType: SchemaTypeString},
		},
		// Missing additionalProperties
	}

	result := ValidateJSONSchema(schema)
	// Should have warning, not error
	assert.True(t, result.Valid)
	assert.NotEmpty(t, result.Warnings)
}

func TestValidateForProvider_OpenAI(t *testing.T) {
	schema := map[string]any{
		SchemaKeyType: SchemaTypeObject,
		SchemaKeyProperties: map[string]any{
			"name": map[string]any{SchemaKeyType: SchemaTypeString},
		},
		// Missing additionalProperties: false - should be error for OpenAI
	}

	result := ValidateForProvider(schema, ProviderOpenAI)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

func TestValidateForProvider_OpenAI_Valid(t *testing.T) {
	schema := map[string]any{
		SchemaKeyType: SchemaTypeObject,
		SchemaKeyProperties: map[string]any{
			"name": map[string]any{SchemaKeyType: SchemaTypeString},
		},
		SchemaKeyAdditionalProperties: false,
	}

	result := ValidateForProvider(schema, ProviderOpenAI)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidateForProvider_Anthropic(t *testing.T) {
	schema := map[string]any{
		SchemaKeyType: SchemaTypeObject,
		SchemaKeyProperties: map[string]any{
			"result": map[string]any{SchemaKeyType: SchemaTypeString},
		},
		SchemaKeyAdditionalProperties: false,
	}

	result := ValidateForProvider(schema, ProviderAnthropic)
	assert.True(t, result.Valid)
}

func TestValidateForProvider_VLLM(t *testing.T) {
	schema := map[string]any{
		SchemaKeyType: SchemaTypeObject,
		SchemaKeyProperties: map[string]any{
			"output": map[string]any{SchemaKeyType: SchemaTypeString},
		},
		// Missing additionalProperties - only warning for vLLM
	}

	result := ValidateForProvider(schema, ProviderVLLM)
	assert.True(t, result.Valid)
	assert.NotEmpty(t, result.Warnings)
}

func TestValidateForProvider_Unknown(t *testing.T) {
	schema := map[string]any{
		SchemaKeyType: SchemaTypeObject,
		SchemaKeyProperties: map[string]any{
			"data": map[string]any{SchemaKeyType: SchemaTypeString},
		},
	}

	result := ValidateForProvider(schema, "unknown-provider")
	assert.True(t, result.Valid)
	assert.NotEmpty(t, result.Warnings) // Warning about unknown provider
}

func TestEnsureAdditionalPropertiesFalse(t *testing.T) {
	schema := map[string]any{
		SchemaKeyType: SchemaTypeObject,
		SchemaKeyProperties: map[string]any{
			"nested": map[string]any{
				SchemaKeyType: SchemaTypeObject,
				SchemaKeyProperties: map[string]any{
					"field": map[string]any{SchemaKeyType: SchemaTypeString},
				},
			},
		},
	}

	result := EnsureAdditionalPropertiesFalse(schema)

	// Original should be unchanged
	_, hasOriginal := schema[SchemaKeyAdditionalProperties]
	assert.False(t, hasOriginal)

	// Result should have additionalProperties: false
	assert.Equal(t, false, result[SchemaKeyAdditionalProperties])

	// Nested object should also have it
	nested := result[SchemaKeyProperties].(map[string]any)["nested"].(map[string]any)
	assert.Equal(t, false, nested[SchemaKeyAdditionalProperties])
}

func TestExtractRequiredFields(t *testing.T) {
	schema := map[string]any{
		SchemaKeyType: SchemaTypeObject,
		SchemaKeyProperties: map[string]any{
			"name":  map[string]any{SchemaKeyType: SchemaTypeString},
			"email": map[string]any{SchemaKeyType: SchemaTypeString},
			"age":   map[string]any{SchemaKeyType: SchemaTypeNumber},
		},
	}

	fields := ExtractRequiredFields(schema)
	assert.Len(t, fields, 3)
	assert.Contains(t, fields, "name")
	assert.Contains(t, fields, "email")
	assert.Contains(t, fields, "age")
}

func TestExtractRequiredFields_Nil(t *testing.T) {
	fields := ExtractRequiredFields(nil)
	assert.Nil(t, fields)
}

func TestExtractRequiredFields_NoProperties(t *testing.T) {
	schema := map[string]any{
		SchemaKeyType: SchemaTypeString,
	}
	fields := ExtractRequiredFields(schema)
	assert.Nil(t, fields)
}

func TestValidateEnumConstraint_Valid(t *testing.T) {
	enum := &EnumConstraint{
		Values:      []string{"positive", "negative", "neutral"},
		Description: "Sentiment classification",
	}

	result := ValidateEnumConstraint(enum)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidateEnumConstraint_Empty(t *testing.T) {
	enum := &EnumConstraint{
		Values: []string{},
	}

	result := ValidateEnumConstraint(enum)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], ErrMsgEnumEmptyValues)
}

func TestValidateEnumConstraint_Nil(t *testing.T) {
	result := ValidateEnumConstraint(nil)
	assert.False(t, result.Valid)
}

func TestValidateGuidedDecoding_Valid(t *testing.T) {
	gd := &GuidedDecoding{
		Backend: GuidedBackendXGrammar,
		JSON: map[string]any{
			SchemaKeyType: SchemaTypeObject,
			SchemaKeyProperties: map[string]any{
				"result": map[string]any{SchemaKeyType: SchemaTypeString},
			},
			SchemaKeyAdditionalProperties: false,
		},
	}

	result := ValidateGuidedDecoding(gd)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidateGuidedDecoding_Conflict(t *testing.T) {
	gd := &GuidedDecoding{
		JSON: map[string]any{
			SchemaKeyType: SchemaTypeObject,
		},
		Regex: "^[a-z]+$", // Conflict - both JSON and Regex set
	}

	result := ValidateGuidedDecoding(gd)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], ErrMsgGuidedDecodingConflict)
}

func TestValidateGuidedDecoding_InvalidBackend(t *testing.T) {
	gd := &GuidedDecoding{
		Backend: "unknown-backend",
		Regex:   "^test$",
	}

	result := ValidateGuidedDecoding(gd)
	assert.True(t, result.Valid) // Warning only
	assert.NotEmpty(t, result.Warnings)
}

func TestValidateGuidedDecoding_Nil(t *testing.T) {
	result := ValidateGuidedDecoding(nil)
	assert.True(t, result.Valid)
}

func TestValidateGuidedDecoding_Choice(t *testing.T) {
	gd := &GuidedDecoding{
		Choice: []string{"yes", "no", "maybe"},
	}

	result := ValidateGuidedDecoding(gd)
	assert.True(t, result.Valid)
}

func TestDetectSchemaProvider(t *testing.T) {
	tests := []struct {
		name     string
		config   *InferenceConfig
		expected string
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: "",
		},
		{
			name: "explicit provider",
			config: &InferenceConfig{
				Model: &ModelConfig{
					Provider: ProviderOpenAI,
				},
			},
			expected: ProviderOpenAI,
		},
		{
			name: "infer from OutputFormat",
			config: &InferenceConfig{
				Model: &ModelConfig{
					OutputFormat: &OutputFormat{},
				},
			},
			expected: ProviderAnthropic,
		},
		{
			name: "infer from GuidedDecoding",
			config: &InferenceConfig{
				Model: &ModelConfig{
					GuidedDecoding: &GuidedDecoding{},
				},
			},
			expected: ProviderVLLM,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectSchemaProvider(tt.config)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsStrictModeRequired(t *testing.T) {
	assert.True(t, IsStrictModeRequired(ProviderOpenAI))
	assert.True(t, IsStrictModeRequired(ProviderAnthropic))
	assert.True(t, IsStrictModeRequired(ProviderGemini))
	assert.True(t, IsStrictModeRequired(ProviderGoogle))
	assert.True(t, IsStrictModeRequired(ProviderVertex))
	assert.True(t, IsStrictModeRequired(ProviderAzure))
	assert.False(t, IsStrictModeRequired(ProviderVLLM))
	assert.False(t, IsStrictModeRequired("unknown"))
}

func TestValidateForProvider_Gemini_PropertyOrdering(t *testing.T) {
	schema := map[string]any{
		SchemaKeyType: SchemaTypeObject,
		SchemaKeyProperties: map[string]any{
			"name":  map[string]any{SchemaKeyType: SchemaTypeString},
			"email": map[string]any{SchemaKeyType: SchemaTypeString},
		},
		SchemaKeyAdditionalProperties: false,
		SchemaKeyPropertyOrdering:     []string{"name", "email"},
	}

	result := ValidateForProvider(schema, ProviderGemini)
	assert.True(t, result.Valid)
}

func TestValidateJSONSchema_PrimitiveTypes(t *testing.T) {
	tests := []struct {
		name     string
		schemaType string
	}{
		{"string", SchemaTypeString},
		{"number", SchemaTypeNumber},
		{"boolean", SchemaTypeBoolean},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := map[string]any{
				SchemaKeyType: tt.schemaType,
			}
			result := ValidateJSONSchema(schema)
			assert.True(t, result.Valid)
		})
	}
}

func TestResponseFormat_ToOpenAI(t *testing.T) {
	rf := &ResponseFormat{
		Type: ResponseFormatJSONSchema,
		JSONSchema: &JSONSchemaSpec{
			Name:        "test_schema",
			Description: "A test schema",
			Strict:      true,
			Schema: map[string]any{
				SchemaKeyType: SchemaTypeObject,
				SchemaKeyProperties: map[string]any{
					"name": map[string]any{SchemaKeyType: SchemaTypeString},
				},
			},
		},
	}

	result := rf.ToOpenAI()
	require.NotNil(t, result)
	assert.Equal(t, ResponseFormatJSONSchema, result[SchemaKeyType])

	jsonSchema := result[SchemaKeyJSONSchema].(map[string]any)
	assert.Equal(t, "test_schema", jsonSchema[AttrName])
	assert.Equal(t, "A test schema", jsonSchema[SchemaKeyDescription])
	assert.True(t, jsonSchema[SchemaKeyStrict].(bool))

	// Schema should have additionalProperties: false added
	schema := jsonSchema[SchemaKeySchema].(map[string]any)
	assert.Equal(t, false, schema[SchemaKeyAdditionalProperties])
}

func TestResponseFormat_ToAnthropic(t *testing.T) {
	rf := &ResponseFormat{
		Type: ResponseFormatJSONSchema,
		JSONSchema: &JSONSchemaSpec{
			Name: "test_schema",
			Schema: map[string]any{
				SchemaKeyType: SchemaTypeObject,
				SchemaKeyProperties: map[string]any{
					"result": map[string]any{SchemaKeyType: SchemaTypeString},
				},
			},
		},
	}

	result := rf.ToAnthropic()
	require.NotNil(t, result)

	format := result[SchemaKeyFormat].(map[string]any)
	assert.Equal(t, ResponseFormatJSONSchema, format[SchemaKeyType])

	schema := format[SchemaKeySchema].(map[string]any)
	assert.Equal(t, false, schema[SchemaKeyAdditionalProperties])
}

func TestResponseFormat_ToGemini(t *testing.T) {
	rf := &ResponseFormat{
		Type: ResponseFormatJSONSchema,
		JSONSchema: &JSONSchemaSpec{
			Name:             "test_schema",
			PropertyOrdering: []string{"first", "second"},
			Schema: map[string]any{
				SchemaKeyType: SchemaTypeObject,
				SchemaKeyProperties: map[string]any{
					"first":  map[string]any{SchemaKeyType: SchemaTypeString},
					"second": map[string]any{SchemaKeyType: SchemaTypeString},
				},
			},
		},
	}

	result := rf.ToGemini()
	require.NotNil(t, result)

	jsonSchema := result[SchemaKeyJSONSchema].(map[string]any)
	schema := jsonSchema[SchemaKeySchema].(map[string]any)

	// Should have propertyOrdering
	ordering := schema[SchemaKeyPropertyOrdering].([]string)
	assert.Equal(t, []string{"first", "second"}, ordering)
}

func TestGuidedDecoding_ToVLLM(t *testing.T) {
	gd := &GuidedDecoding{
		Backend: GuidedBackendXGrammar,
		JSON: map[string]any{
			SchemaKeyType: SchemaTypeObject,
			SchemaKeyProperties: map[string]any{
				"output": map[string]any{SchemaKeyType: SchemaTypeString},
			},
		},
	}

	result := gd.ToVLLM()
	require.NotNil(t, result)

	assert.Equal(t, GuidedBackendXGrammar, result[GuidedKeyDecodingBackend])

	jsonSchema := result[GuidedKeyJSON].(map[string]any)
	assert.Equal(t, false, jsonSchema[SchemaKeyAdditionalProperties])
}

func TestGuidedDecoding_ToVLLM_Regex(t *testing.T) {
	gd := &GuidedDecoding{
		Regex: "^[a-z]+$",
	}

	result := gd.ToVLLM()
	require.NotNil(t, result)
	assert.Equal(t, "^[a-z]+$", result[GuidedKeyRegex])
}

func TestGuidedDecoding_ToVLLM_Choice(t *testing.T) {
	gd := &GuidedDecoding{
		Choice: []string{"yes", "no"},
	}

	result := gd.ToVLLM()
	require.NotNil(t, result)
	assert.Equal(t, []string{"yes", "no"}, result[GuidedKeyChoice])
}

func TestGuidedDecoding_ToVLLM_Grammar(t *testing.T) {
	gd := &GuidedDecoding{
		Grammar: "S -> 'hello' | 'world'",
	}

	result := gd.ToVLLM()
	require.NotNil(t, result)
	assert.Equal(t, "S -> 'hello' | 'world'", result[GuidedKeyGrammar])
}

func TestOutputFormat_ToAnthropic(t *testing.T) {
	of := &OutputFormat{
		Format: &OutputFormatSpec{
			Type: ResponseFormatJSONSchema,
			Schema: map[string]any{
				SchemaKeyType: SchemaTypeObject,
				SchemaKeyProperties: map[string]any{
					"answer": map[string]any{SchemaKeyType: SchemaTypeString},
				},
			},
		},
	}

	result := of.ToAnthropic()
	require.NotNil(t, result)

	format := result[SchemaKeyFormat].(map[string]any)
	assert.Equal(t, ResponseFormatJSONSchema, format[SchemaKeyType])

	schema := format[SchemaKeySchema].(map[string]any)
	assert.Equal(t, false, schema[SchemaKeyAdditionalProperties])
}

func TestInferenceConfig_ProviderFormat_OpenAI(t *testing.T) {
	config := &InferenceConfig{
		Model: &ModelConfig{
			ResponseFormat: &ResponseFormat{
				Type: ResponseFormatJSONSchema,
				JSONSchema: &JSONSchemaSpec{
					Name: "test",
					Schema: map[string]any{
						SchemaKeyType:       SchemaTypeObject,
						SchemaKeyProperties: map[string]any{},
					},
				},
			},
		},
	}

	result, err := config.ProviderFormat(ProviderOpenAI)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ResponseFormatJSONSchema, result[SchemaKeyType])
}

func TestInferenceConfig_ProviderFormat_Anthropic(t *testing.T) {
	config := &InferenceConfig{
		Model: &ModelConfig{
			OutputFormat: &OutputFormat{
				Format: &OutputFormatSpec{
					Type: ResponseFormatJSONSchema,
					Schema: map[string]any{
						SchemaKeyType:       SchemaTypeObject,
						SchemaKeyProperties: map[string]any{},
					},
				},
			},
		},
	}

	result, err := config.ProviderFormat(ProviderAnthropic)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, result, SchemaKeyFormat)
}

func TestInferenceConfig_ProviderFormat_VLLM(t *testing.T) {
	config := &InferenceConfig{
		Model: &ModelConfig{
			GuidedDecoding: &GuidedDecoding{
				Regex: "^test$",
			},
		},
	}

	result, err := config.ProviderFormat(ProviderVLLM)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "^test$", result[GuidedKeyRegex])
}

func TestInferenceConfig_ProviderFormat_Unknown(t *testing.T) {
	config := &InferenceConfig{
		Model: &ModelConfig{},
	}

	_, err := config.ProviderFormat("unknown-provider")
	assert.Error(t, err)
}

func TestCopySchema_DeepCopy(t *testing.T) {
	original := map[string]any{
		"nested": map[string]any{
			"deep": map[string]any{
				"value": "test",
			},
		},
		"array": []any{
			map[string]any{"item": 1},
			map[string]any{"item": 2},
		},
	}

	copied := copySchema(original)

	// Modify the copy
	copied["nested"].(map[string]any)["deep"].(map[string]any)["value"] = "modified"
	copied["array"].([]any)[0].(map[string]any)["item"] = 999

	// Original should be unchanged
	assert.Equal(t, "test", original["nested"].(map[string]any)["deep"].(map[string]any)["value"])
	assert.Equal(t, 1, original["array"].([]any)[0].(map[string]any)["item"])
}
