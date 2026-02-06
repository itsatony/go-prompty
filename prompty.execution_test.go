package prompty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutionConfig_Validate(t *testing.T) {
	temp := func(v float64) *float64 { return &v }
	maxTokens := func(v int) *int { return &v }

	tests := []struct {
		name    string
		config  *ExecutionConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name:    "empty config",
			config:  &ExecutionConfig{},
			wantErr: false,
		},
		{
			name: "valid config",
			config: &ExecutionConfig{
				Provider:    "openai",
				Model:       "gpt-4",
				Temperature: temp(0.7),
				MaxTokens:   maxTokens(1000),
			},
			wantErr: false,
		},
		{
			name: "temperature too low",
			config: &ExecutionConfig{
				Temperature: temp(-0.1),
			},
			wantErr: true,
		},
		{
			name: "temperature too high",
			config: &ExecutionConfig{
				Temperature: temp(2.1),
			},
			wantErr: true,
		},
		{
			name: "top_p too low",
			config: &ExecutionConfig{
				TopP: temp(-0.1),
			},
			wantErr: true,
		},
		{
			name: "top_p too high",
			config: &ExecutionConfig{
				TopP: temp(1.1),
			},
			wantErr: true,
		},
		{
			name: "max_tokens zero",
			config: &ExecutionConfig{
				MaxTokens: maxTokens(0),
			},
			wantErr: true,
		},
		{
			name: "max_tokens negative",
			config: &ExecutionConfig{
				MaxTokens: maxTokens(-1),
			},
			wantErr: true,
		},
		{
			name: "top_k negative",
			config: &ExecutionConfig{
				TopK: maxTokens(-1),
			},
			wantErr: true,
		},
		{
			name: "thinking budget zero",
			config: &ExecutionConfig{
				Thinking: &ThinkingConfig{
					Enabled:      true,
					BudgetTokens: maxTokens(0),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecutionConfig_Clone(t *testing.T) {
	temp := 0.7
	maxTokens := 1000
	topP := 0.9
	topK := 40
	budgetTokens := 5000

	original := &ExecutionConfig{
		Provider:      "openai",
		Model:         "gpt-4",
		Temperature:   &temp,
		MaxTokens:     &maxTokens,
		TopP:          &topP,
		TopK:          &topK,
		StopSequences: []string{"###", "END"},
		Thinking: &ThinkingConfig{
			Enabled:      true,
			BudgetTokens: &budgetTokens,
		},
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
		},
		GuidedDecoding: &GuidedDecoding{
			Backend: "xgrammar",
		},
		ProviderOptions: map[string]any{
			"custom": "option",
		},
	}

	clone := original.Clone()

	// Verify equality
	assert.Equal(t, original.Provider, clone.Provider)
	assert.Equal(t, original.Model, clone.Model)
	assert.Equal(t, *original.Temperature, *clone.Temperature)
	assert.Equal(t, *original.MaxTokens, *clone.MaxTokens)
	assert.Equal(t, *original.TopP, *clone.TopP)
	assert.Equal(t, *original.TopK, *clone.TopK)
	assert.Equal(t, original.StopSequences, clone.StopSequences)
	assert.Equal(t, original.Thinking.Enabled, clone.Thinking.Enabled)
	assert.Equal(t, *original.Thinking.BudgetTokens, *clone.Thinking.BudgetTokens)

	// Verify deep copy
	*clone.Temperature = 0.5
	assert.NotEqual(t, *original.Temperature, *clone.Temperature)

	clone.StopSequences[0] = "MODIFIED"
	assert.NotEqual(t, original.StopSequences[0], clone.StopSequences[0])
}

func TestExecutionConfig_Getters(t *testing.T) {
	temp := 0.7
	maxTokens := 1000
	topP := 0.9
	topK := 40

	config := &ExecutionConfig{
		Provider:      "openai",
		Model:         "gpt-4",
		Temperature:   &temp,
		MaxTokens:     &maxTokens,
		TopP:          &topP,
		TopK:          &topK,
		StopSequences: []string{"###"},
		Thinking: &ThinkingConfig{
			Enabled: true,
		},
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
		},
		GuidedDecoding: &GuidedDecoding{
			Backend: "xgrammar",
		},
	}

	assert.Equal(t, "openai", config.GetProvider())
	assert.Equal(t, "gpt-4", config.GetModel())

	gotTemp, ok := config.GetTemperature()
	assert.True(t, ok)
	assert.Equal(t, 0.7, gotTemp)

	gotMaxTokens, ok := config.GetMaxTokens()
	assert.True(t, ok)
	assert.Equal(t, 1000, gotMaxTokens)

	gotTopP, ok := config.GetTopP()
	assert.True(t, ok)
	assert.Equal(t, 0.9, gotTopP)

	gotTopK, ok := config.GetTopK()
	assert.True(t, ok)
	assert.Equal(t, 40, gotTopK)

	assert.NotNil(t, config.GetStopSequences())
	assert.NotNil(t, config.GetThinking())
	assert.NotNil(t, config.GetResponseFormat())
	assert.NotNil(t, config.GetGuidedDecoding())

	assert.True(t, config.HasThinking())
	assert.True(t, config.HasResponseFormat())
	assert.True(t, config.HasGuidedDecoding())

	// Test nil config
	var nilConfig *ExecutionConfig
	assert.Empty(t, nilConfig.GetProvider())
	assert.Empty(t, nilConfig.GetModel())

	_, ok = nilConfig.GetTemperature()
	assert.False(t, ok)
}

func TestExecutionConfig_GetEffectiveProvider(t *testing.T) {
	tests := []struct {
		name   string
		config *ExecutionConfig
		want   string
	}{
		{
			name:   "nil config",
			config: nil,
			want:   "",
		},
		{
			name: "explicit provider",
			config: &ExecutionConfig{
				Provider: "anthropic",
				Model:    "gpt-4",
			},
			want: "anthropic",
		},
		{
			name: "infer from guided decoding",
			config: &ExecutionConfig{
				Model:          "llama-2",
				GuidedDecoding: &GuidedDecoding{},
			},
			want: ProviderVLLM,
		},
		{
			name: "infer from thinking config",
			config: &ExecutionConfig{
				Model: "custom-model",
				Thinking: &ThinkingConfig{
					Enabled: true,
				},
			},
			want: ProviderAnthropic,
		},
		{
			name: "infer from OpenAI model name",
			config: &ExecutionConfig{
				Model: "gpt-4",
			},
			want: ProviderOpenAI,
		},
		{
			name: "infer from Anthropic model name",
			config: &ExecutionConfig{
				Model: "claude-3-opus",
			},
			want: ProviderAnthropic,
		},
		{
			name: "infer from Gemini model name",
			config: &ExecutionConfig{
				Model: "gemini-pro",
			},
			want: ProviderGemini,
		},
		{
			name: "unknown model",
			config: &ExecutionConfig{
				Model: "custom-unknown-model",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.config.GetEffectiveProvider())
		})
	}
}

func TestExecutionConfig_ToMap(t *testing.T) {
	temp := 0.7
	maxTokens := 1000
	topP := 0.9

	config := &ExecutionConfig{
		Temperature:   &temp,
		MaxTokens:     &maxTokens,
		TopP:          &topP,
		StopSequences: []string{"###", "END"},
	}

	m := config.ToMap()

	assert.Equal(t, 0.7, m[ParamKeyTemperature])
	assert.Equal(t, 1000, m[ParamKeyMaxTokens])
	assert.Equal(t, 0.9, m[ParamKeyTopP])
	assert.Equal(t, []string{"###", "END"}, m[ParamKeyStop])

	// Test nil config
	var nilConfig *ExecutionConfig
	assert.Nil(t, nilConfig.ToMap())
}

func TestExecutionConfig_ToOpenAI(t *testing.T) {
	temp := 0.7
	maxTokens := 1000

	config := &ExecutionConfig{
		Model:       "gpt-4",
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &JSONSchemaSpec{
				Name:   "test_schema",
				Schema: map[string]any{"type": "object"},
			},
		},
		ProviderOptions: map[string]any{
			"custom": "option",
		},
	}

	result := config.ToOpenAI()

	assert.Equal(t, "gpt-4", result["model"])
	assert.Equal(t, 0.7, result[ParamKeyTemperature])
	assert.Equal(t, 1000, result[ParamKeyMaxTokens])
	assert.NotNil(t, result["response_format"])
	assert.Equal(t, "option", result["custom"])
}

func TestExecutionConfig_ToAnthropic(t *testing.T) {
	temp := 0.7
	maxTokens := 1000
	topK := 40
	budgetTokens := 5000

	config := &ExecutionConfig{
		Model:         "claude-3-opus",
		Temperature:   &temp,
		MaxTokens:     &maxTokens,
		TopK:          &topK,
		StopSequences: []string{"###"},
		Thinking: &ThinkingConfig{
			Enabled:      true,
			BudgetTokens: &budgetTokens,
		},
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &JSONSchemaSpec{
				Name:   "test_schema",
				Schema: map[string]any{"type": "object"},
			},
		},
	}

	result := config.ToAnthropic()

	assert.Equal(t, "claude-3-opus", result["model"])
	assert.Equal(t, 0.7, result[ParamKeyTemperature])
	assert.Equal(t, 1000, result["max_tokens"])
	assert.Equal(t, 40, result["top_k"])
	assert.NotNil(t, result["thinking"])
	assert.NotNil(t, result["output_format"])
}

func TestExecutionConfig_ToGemini(t *testing.T) {
	temp := 0.7
	maxTokens := 1000
	topK := 40

	config := &ExecutionConfig{
		Model:         "gemini-pro",
		Temperature:   &temp,
		MaxTokens:     &maxTokens,
		TopK:          &topK,
		StopSequences: []string{"###"},
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
		},
	}

	result := config.ToGemini()

	assert.Equal(t, "gemini-pro", result["model"])

	genConfig := result["generationConfig"].(map[string]any)
	assert.Equal(t, 0.7, genConfig[ParamKeyTemperature])
	assert.Equal(t, 1000, genConfig["maxOutputTokens"])
	assert.Equal(t, 40, genConfig["topK"])
}

func TestExecutionConfig_ToVLLM(t *testing.T) {
	temp := 0.7
	maxTokens := 1000

	config := &ExecutionConfig{
		Model:       "llama-2-7b",
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		GuidedDecoding: &GuidedDecoding{
			Backend: "xgrammar",
			JSON:    map[string]any{"type": "object"},
		},
	}

	result := config.ToVLLM()

	assert.Equal(t, "llama-2-7b", result["model"])
	assert.Equal(t, 0.7, result[ParamKeyTemperature])
	assert.Equal(t, 1000, result["max_tokens"])
	assert.Equal(t, "xgrammar", result[GuidedKeyDecodingBackend])
}

func TestExecutionConfig_ProviderFormat(t *testing.T) {
	config := &ExecutionConfig{
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &JSONSchemaSpec{
				Name:   "test",
				Schema: map[string]any{"type": "object"},
			},
		},
	}

	tests := []struct {
		name     string
		provider string
		wantErr  bool
	}{
		{name: "openai", provider: ProviderOpenAI},
		{name: "azure", provider: ProviderAzure},
		{name: "anthropic", provider: ProviderAnthropic},
		{name: "gemini", provider: ProviderGemini},
		{name: "google", provider: ProviderGoogle},
		{name: "vllm", provider: ProviderVLLM},
		{name: "unknown", provider: "unknown", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.ProviderFormat(tt.provider)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecutionConfig_Merge_BothNil(t *testing.T) {
	var a, b *ExecutionConfig
	result := a.Merge(b)
	assert.Nil(t, result)
}

func TestExecutionConfig_Merge_NilReceiver(t *testing.T) {
	var a *ExecutionConfig
	temp := 0.7
	b := &ExecutionConfig{
		Provider:    "openai",
		Model:       "gpt-4",
		Temperature: &temp,
	}
	result := a.Merge(b)
	require.NotNil(t, result)
	assert.Equal(t, "openai", result.Provider)
	assert.Equal(t, "gpt-4", result.Model)
	assert.Equal(t, 0.7, *result.Temperature)

	// Verify deep copy
	*result.Temperature = 0.5
	assert.Equal(t, 0.7, *b.Temperature)
}

func TestExecutionConfig_Merge_NilOther(t *testing.T) {
	temp := 0.8
	a := &ExecutionConfig{
		Provider:    "anthropic",
		Model:       "claude-3",
		Temperature: &temp,
	}
	result := a.Merge(nil)
	require.NotNil(t, result)
	assert.Equal(t, "anthropic", result.Provider)
	assert.Equal(t, "claude-3", result.Model)
	assert.Equal(t, 0.8, *result.Temperature)

	// Verify deep copy
	*result.Temperature = 0.1
	assert.Equal(t, 0.8, *a.Temperature)
}

func TestExecutionConfig_Merge_PartialOverride(t *testing.T) {
	baseTemp := 0.5
	baseMaxTokens := 1000
	baseTopP := 0.9
	base := &ExecutionConfig{
		Provider:      "openai",
		Model:         "gpt-3.5-turbo",
		Temperature:   &baseTemp,
		MaxTokens:     &baseMaxTokens,
		TopP:          &baseTopP,
		StopSequences: []string{"###"},
		ProviderOptions: map[string]any{
			"seed": 42,
		},
	}

	overrideTemp := 0.8
	overrideTopK := 50
	override := &ExecutionConfig{
		Model:       "gpt-4",
		Temperature: &overrideTemp,
		TopK:        &overrideTopK,
		ProviderOptions: map[string]any{
			"frequency_penalty": 0.5,
		},
	}

	result := base.Merge(override)
	require.NotNil(t, result)

	// Provider kept from base (override was empty)
	assert.Equal(t, "openai", result.Provider)
	// Model overridden
	assert.Equal(t, "gpt-4", result.Model)
	// Temperature overridden
	assert.Equal(t, 0.8, *result.Temperature)
	// MaxTokens kept from base
	assert.Equal(t, 1000, *result.MaxTokens)
	// TopP kept from base (override was nil)
	assert.Equal(t, 0.9, *result.TopP)
	// TopK set from override (base was nil)
	assert.Equal(t, 50, *result.TopK)
	// StopSequences kept from base (override was empty)
	assert.Equal(t, []string{"###"}, result.StopSequences)
	// ProviderOptions merged (both keys present)
	assert.Equal(t, 42, result.ProviderOptions["seed"])
	assert.Equal(t, 0.5, result.ProviderOptions["frequency_penalty"])
}

func TestExecutionConfig_Merge_ThreeLayerChain(t *testing.T) {
	// Simulate: agent → skill → runtime merge
	agentTemp := 0.3
	agentMaxTokens := 2000
	agent := &ExecutionConfig{
		Provider:    "openai",
		Model:       "gpt-4",
		Temperature: &agentTemp,
		MaxTokens:   &agentMaxTokens,
	}

	skillTemp := 0.7
	skill := &ExecutionConfig{
		Temperature: &skillTemp,
	}

	runtimeMaxTokens := 500
	runtime := &ExecutionConfig{
		MaxTokens: &runtimeMaxTokens,
	}

	// agent.Merge(skill).Merge(runtime)
	result := agent.Merge(skill).Merge(runtime)
	require.NotNil(t, result)

	// Provider from agent (not overridden)
	assert.Equal(t, "openai", result.Provider)
	// Model from agent (not overridden)
	assert.Equal(t, "gpt-4", result.Model)
	// Temperature from skill (overrides agent)
	assert.Equal(t, 0.7, *result.Temperature)
	// MaxTokens from runtime (overrides agent)
	assert.Equal(t, 500, *result.MaxTokens)
}

func TestExecutionConfig_Merge_ProviderOptionsOverride(t *testing.T) {
	base := &ExecutionConfig{
		ProviderOptions: map[string]any{
			"key1": "base-val1",
			"key2": "base-val2",
		},
	}
	override := &ExecutionConfig{
		ProviderOptions: map[string]any{
			"key2": "override-val2",
			"key3": "override-val3",
		},
	}

	result := base.Merge(override)
	require.NotNil(t, result)
	assert.Equal(t, "base-val1", result.ProviderOptions["key1"])
	assert.Equal(t, "override-val2", result.ProviderOptions["key2"])
	assert.Equal(t, "override-val3", result.ProviderOptions["key3"])
}

func TestExecutionConfig_Merge_ThinkingOverride(t *testing.T) {
	baseBudget := 5000
	base := &ExecutionConfig{
		Thinking: &ThinkingConfig{
			Enabled:      true,
			BudgetTokens: &baseBudget,
		},
	}

	overrideBudget := 10000
	override := &ExecutionConfig{
		Thinking: &ThinkingConfig{
			Enabled:      false,
			BudgetTokens: &overrideBudget,
		},
	}

	result := base.Merge(override)
	require.NotNil(t, result)
	require.NotNil(t, result.Thinking)
	assert.False(t, result.Thinking.Enabled)
	assert.Equal(t, 10000, *result.Thinking.BudgetTokens)

	// Verify deep copy
	*result.Thinking.BudgetTokens = 999
	assert.Equal(t, 5000, *base.Thinking.BudgetTokens)
	assert.Equal(t, 10000, *override.Thinking.BudgetTokens)
}

func TestExecutionConfig_Merge_StopSequencesOverride(t *testing.T) {
	base := &ExecutionConfig{
		StopSequences: []string{"###", "END"},
	}
	override := &ExecutionConfig{
		StopSequences: []string{"STOP"},
	}

	result := base.Merge(override)
	assert.Equal(t, []string{"STOP"}, result.StopSequences)

	// Verify deep copy
	result.StopSequences[0] = "MODIFIED"
	assert.Equal(t, "STOP", override.StopSequences[0])
}

func TestExecutionConfig_Merge_ResponseFormatOverride(t *testing.T) {
	base := &ExecutionConfig{
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &JSONSchemaSpec{
				Name:   "base_schema",
				Schema: map[string]any{"type": "object"},
			},
		},
	}
	override := &ExecutionConfig{
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &JSONSchemaSpec{
				Name:   "override_schema",
				Schema: map[string]any{"type": "array"},
			},
		},
	}

	result := base.Merge(override)
	require.NotNil(t, result.ResponseFormat)
	require.NotNil(t, result.ResponseFormat.JSONSchema)
	assert.Equal(t, "override_schema", result.ResponseFormat.JSONSchema.Name)
}

func TestExecutionConfig_JSONAndYAML(t *testing.T) {
	temp := 0.7
	config := &ExecutionConfig{
		Provider:    "openai",
		Model:       "gpt-4",
		Temperature: &temp,
	}

	jsonStr, err := config.JSON()
	require.NoError(t, err)
	assert.Contains(t, jsonStr, "openai")
	assert.Contains(t, jsonStr, "gpt-4")

	yamlStr, err := config.YAML()
	require.NoError(t, err)
	assert.Contains(t, yamlStr, "openai")
	assert.Contains(t, yamlStr, "gpt-4")
}
