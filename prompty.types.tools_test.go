package prompty

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ModelParameters.ToMap ---

func TestModelParametersToMapNil(t *testing.T) {
	var p *ModelParameters
	assert.Nil(t, p.ToMap())
}

func TestModelParametersToMapEmpty(t *testing.T) {
	p := &ModelParameters{}
	result := p.ToMap()
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestModelParametersToMapAllFields(t *testing.T) {
	temp := 0.7
	maxTok := 1024
	topP := 0.9
	freqPen := 0.5
	presPen := 0.3
	seed := int64(42)

	p := &ModelParameters{
		Temperature:      &temp,
		MaxTokens:        &maxTok,
		TopP:             &topP,
		FrequencyPenalty: &freqPen,
		PresencePenalty:  &presPen,
		Stop:             []string{"\n", "END"},
		Seed:             &seed,
	}

	result := p.ToMap()
	assert.Equal(t, 0.7, result[ParamKeyTemperature])
	assert.Equal(t, 1024, result[ParamKeyMaxTokens])
	assert.Equal(t, 0.9, result[ParamKeyTopP])
	assert.Equal(t, 0.5, result[ParamKeyFrequencyPenalty])
	assert.Equal(t, 0.3, result[ParamKeyPresencePenalty])
	assert.Equal(t, []string{"\n", "END"}, result[ParamKeyStop])
	assert.Equal(t, int64(42), result[ParamKeySeed])
}

func TestModelParametersToMapPartial(t *testing.T) {
	temp := 0.5
	p := &ModelParameters{
		Temperature: &temp,
	}

	result := p.ToMap()
	assert.Len(t, result, 1)
	assert.Equal(t, 0.5, result[ParamKeyTemperature])
}

func TestModelParametersToMapZeroValues(t *testing.T) {
	temp := 0.0
	maxTok := 0

	p := &ModelParameters{
		Temperature: &temp,
		MaxTokens:   &maxTok,
	}

	result := p.ToMap()
	assert.Len(t, result, 2)
	assert.Equal(t, 0.0, result[ParamKeyTemperature])
	assert.Equal(t, 0, result[ParamKeyMaxTokens])
}

func TestModelParametersToMapEmptyStop(t *testing.T) {
	p := &ModelParameters{
		Stop: []string{},
	}

	result := p.ToMap()
	assert.Empty(t, result)
}

// --- FunctionDef.ToJSON ---

func TestFunctionDefToJSONNil(t *testing.T) {
	var f *FunctionDef
	result, err := f.ToJSON()
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestFunctionDefToJSONMinimal(t *testing.T) {
	f := &FunctionDef{
		Name: "get_weather",
	}

	result, err := f.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, result, "get_weather")

	// Verify it's valid JSON
	var parsed map[string]any
	err = json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "get_weather", parsed["name"])
}

func TestFunctionDefToJSONFull(t *testing.T) {
	f := &FunctionDef{
		Name:        "search_products",
		Description: "Search the product catalog",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string"},
			},
			"required": []string{"query"},
		},
		Returns: map[string]any{
			"type": "array",
		},
		Strict: true,
	}

	result, err := f.ToJSON()
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "search_products", parsed["name"])
	assert.Equal(t, "Search the product catalog", parsed["description"])
	assert.NotNil(t, parsed["parameters"])
	assert.NotNil(t, parsed["returns"])
	assert.Equal(t, true, parsed["strict"])
}

// --- FunctionDef.ToGenericTool ---

func TestFunctionDefToGenericTool(t *testing.T) {
	f := &FunctionDef{
		Name:        "get_weather",
		Description: "Get weather for a location",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]any{"type": "string"},
			},
			"required": []string{"location"},
		},
	}

	result := f.ToGenericTool()
	assert.Equal(t, "get_weather", result[AttrName])
	assert.Equal(t, "Get weather for a location", result[SchemaKeyDescription])
	assert.NotNil(t, result[ToolKeyParameters])
	assert.Nil(t, result[SchemaKeyStrict])
	// Verify flat format — no "type"/"function" envelope
	assert.Nil(t, result[SchemaKeyType])
	assert.Nil(t, result[ToolKeyFunction])
}

func TestFunctionDefToGenericToolNil(t *testing.T) {
	var f *FunctionDef
	assert.Nil(t, f.ToGenericTool())
}

func TestFunctionDefToGenericToolStrict(t *testing.T) {
	f := &FunctionDef{
		Name:        "strict_func",
		Description: "A strict function",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"x": map[string]any{"type": "string"},
			},
		},
		Strict: true,
	}

	result := f.ToGenericTool()
	assert.Equal(t, true, result[SchemaKeyStrict])
	params := result[ToolKeyParameters].(map[string]any)
	assert.Equal(t, false, params[SchemaKeyAdditionalProperties])
}

func TestFunctionDefToGenericToolMinimalFields(t *testing.T) {
	f := &FunctionDef{
		Name: "minimal_func",
	}

	result := f.ToGenericTool()
	assert.Equal(t, "minimal_func", result[AttrName])
	assert.Nil(t, result[SchemaKeyDescription])
	assert.Nil(t, result[ToolKeyParameters])
	assert.Nil(t, result[SchemaKeyStrict])
}

func TestFunctionDefToOpenAIToolWrapsGeneric(t *testing.T) {
	f := &FunctionDef{
		Name:        "test_func",
		Description: "Test function",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"q": map[string]any{"type": "string"},
			},
		},
	}

	openAI := f.ToOpenAITool()
	generic := f.ToGenericTool()

	// OpenAI envelope
	assert.Equal(t, ToolKeyFunction, openAI[SchemaKeyType])
	inner := openAI[ToolKeyFunction].(map[string]any)

	// Inner content matches generic output
	assert.Equal(t, generic[AttrName], inner[AttrName])
	assert.Equal(t, generic[SchemaKeyDescription], inner[SchemaKeyDescription])
	assert.NotNil(t, inner[ToolKeyParameters])
}

func TestFunctionDefToGenericToolDeepCopy(t *testing.T) {
	f := &FunctionDef{
		Name: "deep_copy_func",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"a": map[string]any{"type": "string"},
			},
		},
	}

	result := f.ToGenericTool()
	params := result[ToolKeyParameters].(map[string]any)
	props := params["properties"].(map[string]any)

	// Mutate the returned copy
	props["b"] = map[string]any{"type": "number"}

	// Original should be unaffected
	origProps := f.Parameters["properties"].(map[string]any)
	assert.Nil(t, origProps["b"])
}

// --- FunctionDef.ToOpenAITool ---

func TestFunctionDefToOpenAIToolNil(t *testing.T) {
	var f *FunctionDef
	assert.Nil(t, f.ToOpenAITool())
}

func TestFunctionDefToOpenAIToolMinimal(t *testing.T) {
	f := &FunctionDef{
		Name: "test_func",
	}

	result := f.ToOpenAITool()
	assert.Equal(t, ToolKeyFunction, result[SchemaKeyType])
	funcDef := result[ToolKeyFunction].(map[string]any)
	assert.Equal(t, "test_func", funcDef[AttrName])
}

func TestFunctionDefToOpenAIToolWithStrict(t *testing.T) {
	f := &FunctionDef{
		Name:        "strict_func",
		Description: "A strict function",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"x": map[string]any{"type": "string"},
			},
		},
		Strict: true,
	}

	result := f.ToOpenAITool()
	funcDef := result[ToolKeyFunction].(map[string]any)
	assert.Equal(t, true, funcDef[SchemaKeyStrict])
	assert.NotNil(t, funcDef[ToolKeyParameters])
}

func TestFunctionDefToOpenAIToolWithDescription(t *testing.T) {
	f := &FunctionDef{
		Name:        "described_func",
		Description: "Does something useful",
	}

	result := f.ToOpenAITool()
	funcDef := result[ToolKeyFunction].(map[string]any)
	assert.Equal(t, "Does something useful", funcDef[SchemaKeyDescription])
}

// --- FunctionDef.ToAnthropicTool ---

func TestFunctionDefToAnthropicToolNil(t *testing.T) {
	var f *FunctionDef
	assert.Nil(t, f.ToAnthropicTool())
}

func TestFunctionDefToAnthropicToolMinimal(t *testing.T) {
	f := &FunctionDef{
		Name: "test_func",
	}

	result := f.ToAnthropicTool()
	assert.Equal(t, "test_func", result[AttrName])
	assert.Nil(t, result[ToolKeyInputSchema])
}

func TestFunctionDefToAnthropicToolFull(t *testing.T) {
	f := &FunctionDef{
		Name:        "search",
		Description: "Search for items",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string"},
			},
		},
	}

	result := f.ToAnthropicTool()
	assert.Equal(t, "search", result[AttrName])
	assert.Equal(t, "Search for items", result[SchemaKeyDescription])
	assert.NotNil(t, result[ToolKeyInputSchema])
}

// --- ToolDefinition ---

func TestToolDefinitionStruct(t *testing.T) {
	td := ToolDefinition{
		Type: "function",
		Function: &FunctionDef{
			Name:        "my_tool",
			Description: "A test tool",
		},
	}

	assert.Equal(t, "function", td.Type)
	assert.Equal(t, "my_tool", td.Function.Name)
}

// --- StreamingConfig ---

func TestStreamingConfigDefaults(t *testing.T) {
	sc := StreamingConfig{}
	assert.False(t, sc.Enabled)
	assert.Equal(t, "", sc.Method)
}

// --- RetryConfig ---

func TestRetryConfigDefaults(t *testing.T) {
	rc := RetryConfig{}
	assert.Equal(t, 0, rc.MaxAttempts)
	assert.Equal(t, "", rc.Backoff)
}

// --- PromptCacheConfig ---

func TestPromptCacheConfigDefaults(t *testing.T) {
	cc := PromptCacheConfig{}
	assert.False(t, cc.SystemPrompt)
	assert.Equal(t, 0, cc.TTL)
}
