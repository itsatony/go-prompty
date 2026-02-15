package prompty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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
				Provider:    ProviderOpenAI,
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
		Provider:      ProviderOpenAI,
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
		Provider:      ProviderOpenAI,
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

	assert.Equal(t, ProviderOpenAI, config.GetProvider())
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
				Provider: ProviderAnthropic,
				Model:    "gpt-4",
			},
			want: ProviderAnthropic,
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

	assert.Equal(t, "gpt-4", result[ParamKeyModel])
	assert.Equal(t, 0.7, result[ParamKeyTemperature])
	assert.Equal(t, 1000, result[ParamKeyMaxTokens])
	assert.NotNil(t, result[ParamKeyResponseFormat])
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

	assert.Equal(t, "claude-3-opus", result[ParamKeyModel])
	assert.Equal(t, 0.7, result[ParamKeyTemperature])
	assert.Equal(t, 1000, result[ParamKeyMaxTokens])
	assert.Equal(t, 40, result[ParamKeyTopK])
	assert.NotNil(t, result[ParamKeyAnthropicThinking])
	assert.NotNil(t, result[ParamKeyAnthropicOutputFormat])
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

	assert.Equal(t, "gemini-pro", result[ParamKeyModel])

	genConfig := result[ParamKeyGenerationConfig].(map[string]any)
	assert.Equal(t, 0.7, genConfig[ParamKeyTemperature])
	assert.Equal(t, 1000, genConfig[ParamKeyGeminiMaxTokens])
	assert.Equal(t, 40, genConfig[ParamKeyGeminiTopK])
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

	assert.Equal(t, "llama-2-7b", result[ParamKeyModel])
	assert.Equal(t, 0.7, result[ParamKeyTemperature])
	assert.Equal(t, 1000, result[ParamKeyMaxTokens])
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
		Provider:    ProviderOpenAI,
		Model:       "gpt-4",
		Temperature: &temp,
	}
	result := a.Merge(b)
	require.NotNil(t, result)
	assert.Equal(t, ProviderOpenAI, result.Provider)
	assert.Equal(t, "gpt-4", result.Model)
	assert.Equal(t, 0.7, *result.Temperature)

	// Verify deep copy
	*result.Temperature = 0.5
	assert.Equal(t, 0.7, *b.Temperature)
}

func TestExecutionConfig_Merge_NilOther(t *testing.T) {
	temp := 0.8
	a := &ExecutionConfig{
		Provider:    ProviderAnthropic,
		Model:       "claude-3",
		Temperature: &temp,
	}
	result := a.Merge(nil)
	require.NotNil(t, result)
	assert.Equal(t, ProviderAnthropic, result.Provider)
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
		Provider:      ProviderOpenAI,
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
	assert.Equal(t, ProviderOpenAI, result.Provider)
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
		Provider:    ProviderOpenAI,
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
	assert.Equal(t, ProviderOpenAI, result.Provider)
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

// v2.3 extended inference parameter tests

func TestExecutionConfig_Validate_MinP(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	tests := []struct {
		name    string
		config  *ExecutionConfig
		wantErr bool
	}{
		{name: "min_p valid 0.0", config: &ExecutionConfig{MinP: f(0.0)}, wantErr: false},
		{name: "min_p valid 0.5", config: &ExecutionConfig{MinP: f(0.5)}, wantErr: false},
		{name: "min_p valid 1.0", config: &ExecutionConfig{MinP: f(1.0)}, wantErr: false},
		{name: "min_p too low", config: &ExecutionConfig{MinP: f(-0.1)}, wantErr: true},
		{name: "min_p too high", config: &ExecutionConfig{MinP: f(1.1)}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgMinPOutOfRange)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecutionConfig_Validate_RepetitionPenalty(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	tests := []struct {
		name    string
		config  *ExecutionConfig
		wantErr bool
	}{
		{name: "valid 1.0", config: &ExecutionConfig{RepetitionPenalty: f(1.0)}, wantErr: false},
		{name: "valid 2.5", config: &ExecutionConfig{RepetitionPenalty: f(2.5)}, wantErr: false},
		{name: "valid 0.01", config: &ExecutionConfig{RepetitionPenalty: f(0.01)}, wantErr: false},
		{name: "zero", config: &ExecutionConfig{RepetitionPenalty: f(0.0)}, wantErr: true},
		{name: "negative", config: &ExecutionConfig{RepetitionPenalty: f(-1.0)}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgRepetitionPenaltyOutOfRange)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecutionConfig_Validate_Seed(t *testing.T) {
	i := func(v int) *int { return &v }

	// Seed has no range restriction — any int is valid
	tests := []struct {
		name   string
		config *ExecutionConfig
	}{
		{name: "positive", config: &ExecutionConfig{Seed: i(42)}},
		{name: "zero", config: &ExecutionConfig{Seed: i(0)}},
		{name: "negative", config: &ExecutionConfig{Seed: i(-1)}},
		{name: "large", config: &ExecutionConfig{Seed: i(999999999)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, tt.config.Validate())
		})
	}
}

func TestExecutionConfig_Validate_Logprobs(t *testing.T) {
	i := func(v int) *int { return &v }

	tests := []struct {
		name    string
		config  *ExecutionConfig
		wantErr bool
	}{
		{name: "valid 0", config: &ExecutionConfig{Logprobs: i(0)}, wantErr: false},
		{name: "valid 5", config: &ExecutionConfig{Logprobs: i(5)}, wantErr: false},
		{name: "valid 20", config: &ExecutionConfig{Logprobs: i(20)}, wantErr: false},
		{name: "negative", config: &ExecutionConfig{Logprobs: i(-1)}, wantErr: true},
		{name: "too high", config: &ExecutionConfig{Logprobs: i(21)}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgLogprobsOutOfRange)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecutionConfig_Validate_StopTokenIDs(t *testing.T) {
	tests := []struct {
		name    string
		config  *ExecutionConfig
		wantErr bool
	}{
		{name: "valid", config: &ExecutionConfig{StopTokenIDs: []int{0, 1, 50256}}, wantErr: false},
		{name: "empty", config: &ExecutionConfig{StopTokenIDs: []int{}}, wantErr: false},
		{name: "negative", config: &ExecutionConfig{StopTokenIDs: []int{100, -1}}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgStopTokenIDNegative)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecutionConfig_Validate_LogitBias(t *testing.T) {
	tests := []struct {
		name    string
		config  *ExecutionConfig
		wantErr bool
	}{
		{name: "valid", config: &ExecutionConfig{LogitBias: map[string]float64{"100": 5.0, "200": -5.0}}, wantErr: false},
		{name: "boundary low", config: &ExecutionConfig{LogitBias: map[string]float64{"1": -100.0}}, wantErr: false},
		{name: "boundary high", config: &ExecutionConfig{LogitBias: map[string]float64{"1": 100.0}}, wantErr: false},
		{name: "too low", config: &ExecutionConfig{LogitBias: map[string]float64{"1": -100.1}}, wantErr: true},
		{name: "too high", config: &ExecutionConfig{LogitBias: map[string]float64{"1": 100.1}}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgLogitBiasValueOutOfRange)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecutionConfig_Clone_ExtendedParams(t *testing.T) {
	minP := 0.1
	repPen := 1.2
	seed := 42
	logprobs := 5

	original := &ExecutionConfig{
		MinP:              &minP,
		RepetitionPenalty: &repPen,
		Seed:              &seed,
		Logprobs:          &logprobs,
		StopTokenIDs:      []int{50256, 50257},
		LogitBias:         map[string]float64{"100": 5.0, "200": -10.0},
	}

	clone := original.Clone()

	// Verify equality
	assert.Equal(t, *original.MinP, *clone.MinP)
	assert.Equal(t, *original.RepetitionPenalty, *clone.RepetitionPenalty)
	assert.Equal(t, *original.Seed, *clone.Seed)
	assert.Equal(t, *original.Logprobs, *clone.Logprobs)
	assert.Equal(t, original.StopTokenIDs, clone.StopTokenIDs)
	assert.Equal(t, original.LogitBias, clone.LogitBias)

	// Verify deep copy independence
	*clone.MinP = 0.9
	assert.NotEqual(t, *original.MinP, *clone.MinP)

	*clone.Seed = 999
	assert.NotEqual(t, *original.Seed, *clone.Seed)

	clone.StopTokenIDs[0] = 12345
	assert.NotEqual(t, original.StopTokenIDs[0], clone.StopTokenIDs[0])

	clone.LogitBias["100"] = 99.0
	assert.NotEqual(t, original.LogitBias["100"], clone.LogitBias["100"])
}

func TestExecutionConfig_Getters_ExtendedParams(t *testing.T) {
	minP := 0.1
	repPen := 1.5
	seed := 42
	logprobs := 5

	config := &ExecutionConfig{
		MinP:              &minP,
		RepetitionPenalty: &repPen,
		Seed:              &seed,
		Logprobs:          &logprobs,
		StopTokenIDs:      []int{50256},
		LogitBias:         map[string]float64{"100": 5.0},
	}

	gotMinP, ok := config.GetMinP()
	assert.True(t, ok)
	assert.Equal(t, 0.1, gotMinP)
	assert.True(t, config.HasMinP())

	gotRepPen, ok := config.GetRepetitionPenalty()
	assert.True(t, ok)
	assert.Equal(t, 1.5, gotRepPen)
	assert.True(t, config.HasRepetitionPenalty())

	gotSeed, ok := config.GetSeed()
	assert.True(t, ok)
	assert.Equal(t, 42, gotSeed)
	assert.True(t, config.HasSeed())

	gotLogprobs, ok := config.GetLogprobs()
	assert.True(t, ok)
	assert.Equal(t, 5, gotLogprobs)
	assert.True(t, config.HasLogprobs())

	assert.Equal(t, []int{50256}, config.GetStopTokenIDs())
	assert.True(t, config.HasStopTokenIDs())

	assert.Equal(t, map[string]float64{"100": 5.0}, config.GetLogitBias())
	assert.True(t, config.HasLogitBias())

	// nil config
	var nilConfig *ExecutionConfig
	_, ok = nilConfig.GetMinP()
	assert.False(t, ok)
	assert.False(t, nilConfig.HasMinP())
	_, ok = nilConfig.GetRepetitionPenalty()
	assert.False(t, ok)
	_, ok = nilConfig.GetSeed()
	assert.False(t, ok)
	_, ok = nilConfig.GetLogprobs()
	assert.False(t, ok)
	assert.Nil(t, nilConfig.GetStopTokenIDs())
	assert.False(t, nilConfig.HasStopTokenIDs())
	assert.Nil(t, nilConfig.GetLogitBias())
	assert.False(t, nilConfig.HasLogitBias())
}

func TestExecutionConfig_GetEffectiveProvider_VLLMHints(t *testing.T) {
	minP := 0.1
	repPen := 1.2

	tests := []struct {
		name   string
		config *ExecutionConfig
		want   string
	}{
		{
			name:   "min_p hints vllm",
			config: &ExecutionConfig{MinP: &minP},
			want:   ProviderVLLM,
		},
		{
			name:   "repetition_penalty hints vllm",
			config: &ExecutionConfig{RepetitionPenalty: &repPen},
			want:   ProviderVLLM,
		},
		{
			name:   "stop_token_ids hints vllm",
			config: &ExecutionConfig{StopTokenIDs: []int{50256}},
			want:   ProviderVLLM,
		},
		{
			name:   "seed does not hint (cross-provider)",
			config: &ExecutionConfig{Seed: func() *int { v := 42; return &v }()},
			want:   "",
		},
		{
			name:   "logprobs does not hint (cross-provider)",
			config: &ExecutionConfig{Logprobs: func() *int { v := 5; return &v }()},
			want:   "",
		},
		{
			name:   "logit_bias does not hint (cross-provider)",
			config: &ExecutionConfig{LogitBias: map[string]float64{"100": 5.0}},
			want:   "",
		},
		{
			name:   "explicit provider overrides vllm hint",
			config: &ExecutionConfig{Provider: ProviderOpenAI, MinP: &minP},
			want:   ProviderOpenAI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.config.GetEffectiveProvider())
		})
	}
}

func TestExecutionConfig_ToOpenAI_ExtendedParams(t *testing.T) {
	seed := 42
	logprobs := 5
	logprobsZero := 0

	t.Run("seed and logprobs and logit_bias", func(t *testing.T) {
		config := &ExecutionConfig{
			Model:     "gpt-4",
			Seed:      &seed,
			Logprobs:  &logprobs,
			LogitBias: map[string]float64{"100": 5.0, "200": -10.0},
		}

		result := config.ToOpenAI()

		assert.Equal(t, 42, result[ParamKeySeed])
		// OpenAI uses dual fields: logprobs=true + top_logprobs=N
		assert.Equal(t, true, result[ParamKeyLogprobs])
		assert.Equal(t, 5, result[ParamKeyTopLogprobs])
		assert.Equal(t, map[string]float64{"100": 5.0, "200": -10.0}, result[ParamKeyLogitBias])
	})

	t.Run("logprobs zero still emits dual fields", func(t *testing.T) {
		config := &ExecutionConfig{
			Logprobs: &logprobsZero,
		}

		result := config.ToOpenAI()

		assert.Equal(t, true, result[ParamKeyLogprobs])
		assert.Equal(t, 0, result[ParamKeyTopLogprobs])
	})

	t.Run("vllm-only params not emitted", func(t *testing.T) {
		minP := 0.1
		repPen := 1.2
		config := &ExecutionConfig{
			MinP:              &minP,
			RepetitionPenalty: &repPen,
			StopTokenIDs:      []int{50256},
		}

		result := config.ToOpenAI()

		_, hasMinP := result[ParamKeyMinP]
		_, hasRepPen := result[ParamKeyRepetitionPenalty]
		_, hasStopTokenIDs := result[ParamKeyStopTokenIDs]
		assert.False(t, hasMinP)
		assert.False(t, hasRepPen)
		assert.False(t, hasStopTokenIDs)
	})
}

func TestExecutionConfig_ToAnthropic_ExtendedParams(t *testing.T) {
	seed := 42
	minP := 0.1
	logprobs := 5

	config := &ExecutionConfig{
		Model:        "claude-3-opus",
		Seed:         &seed,
		MinP:         &minP,
		Logprobs:     &logprobs,
		LogitBias:    map[string]float64{"100": 5.0},
		StopTokenIDs: []int{50256},
	}

	result := config.ToAnthropic()

	// Seed is supported
	assert.Equal(t, 42, result[ParamKeySeed])

	// Others should not be present
	_, hasMinP := result[ParamKeyMinP]
	_, hasLogprobs := result[ParamKeyLogprobs]
	_, hasLogitBias := result[ParamKeyLogitBias]
	_, hasStopTokenIDs := result[ParamKeyStopTokenIDs]
	assert.False(t, hasMinP)
	assert.False(t, hasLogprobs)
	assert.False(t, hasLogitBias)
	assert.False(t, hasStopTokenIDs)
}

func TestExecutionConfig_ToVLLM_ExtendedParams(t *testing.T) {
	minP := 0.1
	repPen := 1.2
	seed := 42
	logprobs := 5

	config := &ExecutionConfig{
		Model:             "llama-2-7b",
		MinP:              &minP,
		RepetitionPenalty: &repPen,
		Seed:              &seed,
		Logprobs:          &logprobs,
		StopTokenIDs:      []int{50256, 50257},
		LogitBias:         map[string]float64{"100": 5.0},
	}

	result := config.ToVLLM()

	assert.Equal(t, 0.1, result[ParamKeyMinP])
	assert.Equal(t, 1.2, result[ParamKeyRepetitionPenalty])
	assert.Equal(t, 42, result[ParamKeySeed])
	assert.Equal(t, 5, result[ParamKeyLogprobs])
	assert.Equal(t, []int{50256, 50257}, result[ParamKeyStopTokenIDs])
	assert.Equal(t, map[string]float64{"100": 5.0}, result[ParamKeyLogitBias])
}

func TestExecutionConfig_ToMap_ExtendedParams(t *testing.T) {
	minP := 0.1
	repPen := 1.2
	seed := 42
	logprobs := 5

	config := &ExecutionConfig{
		MinP:              &minP,
		RepetitionPenalty: &repPen,
		Seed:              &seed,
		Logprobs:          &logprobs,
		StopTokenIDs:      []int{50256},
		LogitBias:         map[string]float64{"100": 5.0},
	}

	m := config.ToMap()

	assert.Equal(t, 0.1, m[ParamKeyMinP])
	assert.Equal(t, 1.2, m[ParamKeyRepetitionPenalty])
	assert.Equal(t, 42, m[ParamKeySeed])
	assert.Equal(t, 5, m[ParamKeyLogprobs])
	assert.Equal(t, []int{50256}, m[ParamKeyStopTokenIDs])
	assert.Equal(t, map[string]float64{"100": 5.0}, m[ParamKeyLogitBias])
}

func TestExecutionConfig_Merge_ExtendedParams(t *testing.T) {
	t.Run("pointer override", func(t *testing.T) {
		baseMinP := 0.1
		baseSeed := 42
		base := &ExecutionConfig{
			MinP: &baseMinP,
			Seed: &baseSeed,
		}

		overrideMinP := 0.5
		override := &ExecutionConfig{
			MinP: &overrideMinP,
		}

		result := base.Merge(override)
		assert.Equal(t, 0.5, *result.MinP)
		assert.Equal(t, 42, *result.Seed)
	})

	t.Run("stop_token_ids replacement", func(t *testing.T) {
		base := &ExecutionConfig{
			StopTokenIDs: []int{1, 2, 3},
		}
		override := &ExecutionConfig{
			StopTokenIDs: []int{50256},
		}

		result := base.Merge(override)
		assert.Equal(t, []int{50256}, result.StopTokenIDs)

		// Verify deep copy
		result.StopTokenIDs[0] = 99999
		assert.Equal(t, 50256, override.StopTokenIDs[0])
	})

	t.Run("logit_bias full replacement", func(t *testing.T) {
		base := &ExecutionConfig{
			LogitBias: map[string]float64{"100": 5.0, "200": -10.0},
		}
		override := &ExecutionConfig{
			LogitBias: map[string]float64{"300": 50.0},
		}

		result := base.Merge(override)
		// LogitBias is fully replaced, not key-merged
		assert.Equal(t, map[string]float64{"300": 50.0}, result.LogitBias)

		// Verify deep copy
		result.LogitBias["300"] = 99.0
		assert.Equal(t, 50.0, override.LogitBias["300"])
	})

	t.Run("nil override keeps base", func(t *testing.T) {
		baseRepPen := 1.5
		baseLogprobs := 10
		base := &ExecutionConfig{
			RepetitionPenalty: &baseRepPen,
			Logprobs:          &baseLogprobs,
			StopTokenIDs:      []int{1, 2},
			LogitBias:         map[string]float64{"100": 5.0},
		}
		override := &ExecutionConfig{}

		result := base.Merge(override)
		assert.Equal(t, 1.5, *result.RepetitionPenalty)
		assert.Equal(t, 10, *result.Logprobs)
		assert.Equal(t, []int{1, 2}, result.StopTokenIDs)
		assert.Equal(t, map[string]float64{"100": 5.0}, result.LogitBias)
	})
}

func TestExecutionConfig_ToGemini_NoExtendedParams(t *testing.T) {
	seed := 42
	logprobs := 5
	minP := 0.1

	config := &ExecutionConfig{
		Model:    "gemini-pro",
		Seed:     &seed,
		Logprobs: &logprobs,
		MinP:     &minP,
	}

	result := config.ToGemini()

	// None of the extended params should appear in Gemini output
	genConfig, ok := result[ParamKeyGenerationConfig]
	if ok {
		gc := genConfig.(map[string]any)
		_, hasSeed := gc[ParamKeySeed]
		_, hasLogprobs := gc[ParamKeyLogprobs]
		_, hasMinP := gc[ParamKeyMinP]
		assert.False(t, hasSeed)
		assert.False(t, hasLogprobs)
		assert.False(t, hasMinP)
	}
	_, hasSeed := result[ParamKeySeed]
	assert.False(t, hasSeed)
}

func TestExecutionConfig_JSONAndYAML(t *testing.T) {
	temp := 0.7
	config := &ExecutionConfig{
		Provider:    ProviderOpenAI,
		Model:       "gpt-4",
		Temperature: &temp,
	}

	jsonStr, err := config.JSON()
	require.NoError(t, err)
	assert.Contains(t, jsonStr, ProviderOpenAI)
	assert.Contains(t, jsonStr, "gpt-4")

	yamlStr, err := config.YAML()
	require.NoError(t, err)
	assert.Contains(t, yamlStr, ProviderOpenAI)
	assert.Contains(t, yamlStr, "gpt-4")
}

// ==================== v2.5 Media Generation Tests ====================

func TestExecutionConfig_Validate_Modality(t *testing.T) {
	tests := []struct {
		name    string
		config  *ExecutionConfig
		wantErr bool
	}{
		{name: "valid text", config: &ExecutionConfig{Modality: ModalityText}, wantErr: false},
		{name: "valid image", config: &ExecutionConfig{Modality: ModalityImage}, wantErr: false},
		{name: "valid audio_speech", config: &ExecutionConfig{Modality: ModalityAudioSpeech}, wantErr: false},
		{name: "valid embedding", config: &ExecutionConfig{Modality: ModalityEmbedding}, wantErr: false},
		{name: "empty modality", config: &ExecutionConfig{}, wantErr: false},
		{name: "invalid modality", config: &ExecutionConfig{Modality: "video"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ErrMsgInvalidModality)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecutionConfig_Validate_MediaConfigs(t *testing.T) {
	t.Run("invalid image propagates", func(t *testing.T) {
		config := &ExecutionConfig{
			Image: &ImageConfig{Width: func() *int { v := 0; return &v }()},
		}
		err := config.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgImageWidthOutOfRange)
	})

	t.Run("invalid audio propagates", func(t *testing.T) {
		config := &ExecutionConfig{
			Audio: &AudioConfig{Speed: func() *float64 { v := 0.1; return &v }()},
		}
		err := config.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgAudioSpeedOutOfRange)
	})

	t.Run("invalid embedding propagates", func(t *testing.T) {
		config := &ExecutionConfig{
			Embedding: &EmbeddingConfig{Dimensions: func() *int { v := 0; return &v }()},
		}
		err := config.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgEmbeddingDimensionsOutOfRange)
	})

	t.Run("invalid streaming propagates", func(t *testing.T) {
		config := &ExecutionConfig{
			Streaming: &StreamingConfig{Enabled: true, Method: "grpc"},
		}
		err := config.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgStreamInvalidMethod)
	})

	t.Run("invalid async propagates", func(t *testing.T) {
		config := &ExecutionConfig{
			Async: &AsyncConfig{PollIntervalSeconds: func() *float64 { v := -1.0; return &v }()},
		}
		err := config.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgAsyncPollIntervalInvalid)
	})

	t.Run("valid media config", func(t *testing.T) {
		config := &ExecutionConfig{
			Modality: ModalityImage,
			Image: &ImageConfig{
				Width:   func() *int { v := 1024; return &v }(),
				Height:  func() *int { v := 1024; return &v }(),
				Quality: ImageQualityHD,
			},
			Streaming: &StreamingConfig{Enabled: true, Method: StreamMethodSSE},
		}
		assert.NoError(t, config.Validate())
	})
}

func TestExecutionConfig_Clone_MediaFields(t *testing.T) {
	original := &ExecutionConfig{
		Provider: ProviderOpenAI,
		Model:    "dall-e-3",
		Modality: ModalityImage,
		Image: &ImageConfig{
			Width:   func() *int { v := 1024; return &v }(),
			Height:  func() *int { v := 1024; return &v }(),
			Quality: ImageQualityHD,
			Style:   ImageStyleVivid,
		},
		Audio: &AudioConfig{
			Voice:        "alloy",
			Speed:        func() *float64 { v := 1.5; return &v }(),
			OutputFormat: AudioFormatMP3,
		},
		Embedding: &EmbeddingConfig{
			Dimensions: func() *int { v := 1536; return &v }(),
			Format:     EmbeddingFormatFloat,
		},
		Streaming: &StreamingConfig{
			Enabled: true,
			Method:  StreamMethodSSE,
		},
		Async: &AsyncConfig{
			Enabled:             true,
			PollIntervalSeconds: func() *float64 { v := 2.0; return &v }(),
			PollTimeoutSeconds:  func() *float64 { v := 60.0; return &v }(),
		},
	}

	clone := original.Clone()
	require.NotNil(t, clone)

	// Verify equality
	assert.Equal(t, original.Modality, clone.Modality)
	assert.Equal(t, *original.Image.Width, *clone.Image.Width)
	assert.Equal(t, original.Image.Quality, clone.Image.Quality)
	assert.Equal(t, original.Audio.Voice, clone.Audio.Voice)
	assert.Equal(t, *original.Audio.Speed, *clone.Audio.Speed)
	assert.Equal(t, *original.Embedding.Dimensions, *clone.Embedding.Dimensions)
	assert.Equal(t, original.Streaming.Enabled, clone.Streaming.Enabled)
	assert.Equal(t, original.Streaming.Method, clone.Streaming.Method)
	assert.Equal(t, original.Async.Enabled, clone.Async.Enabled)

	// Verify deep copy independence
	*clone.Image.Width = 512
	assert.NotEqual(t, *original.Image.Width, *clone.Image.Width)

	*clone.Audio.Speed = 2.0
	assert.NotEqual(t, *original.Audio.Speed, *clone.Audio.Speed)

	*clone.Embedding.Dimensions = 3072
	assert.NotEqual(t, *original.Embedding.Dimensions, *clone.Embedding.Dimensions)

	*clone.Async.PollIntervalSeconds = 10.0
	assert.NotEqual(t, *original.Async.PollIntervalSeconds, *clone.Async.PollIntervalSeconds)
}

func TestExecutionConfig_Merge_MediaFields(t *testing.T) {
	t.Run("modality override", func(t *testing.T) {
		base := &ExecutionConfig{Modality: ModalityText}
		override := &ExecutionConfig{Modality: ModalityImage}
		result := base.Merge(override)
		assert.Equal(t, ModalityImage, result.Modality)
	})

	t.Run("modality empty keeps base", func(t *testing.T) {
		base := &ExecutionConfig{Modality: ModalityImage}
		override := &ExecutionConfig{}
		result := base.Merge(override)
		assert.Equal(t, ModalityImage, result.Modality)
	})

	t.Run("image override", func(t *testing.T) {
		base := &ExecutionConfig{
			Image: &ImageConfig{Quality: ImageQualityStandard},
		}
		override := &ExecutionConfig{
			Image: &ImageConfig{Quality: ImageQualityHD},
		}
		result := base.Merge(override)
		require.NotNil(t, result.Image)
		assert.Equal(t, ImageQualityHD, result.Image.Quality)
	})

	t.Run("nil override keeps base image", func(t *testing.T) {
		base := &ExecutionConfig{
			Image: &ImageConfig{Quality: ImageQualityHD},
		}
		override := &ExecutionConfig{}
		result := base.Merge(override)
		require.NotNil(t, result.Image)
		assert.Equal(t, ImageQualityHD, result.Image.Quality)
	})

	t.Run("audio override deep copy", func(t *testing.T) {
		overrideSpeed := 2.0
		override := &ExecutionConfig{
			Audio: &AudioConfig{Voice: "echo", Speed: &overrideSpeed},
		}
		result := (&ExecutionConfig{}).Merge(override)
		require.NotNil(t, result.Audio)
		assert.Equal(t, "echo", result.Audio.Voice)

		// Verify deep copy
		*result.Audio.Speed = 3.0
		assert.Equal(t, 2.0, *override.Audio.Speed)
	})

	t.Run("streaming override", func(t *testing.T) {
		base := &ExecutionConfig{
			Streaming: &StreamingConfig{Enabled: false},
		}
		override := &ExecutionConfig{
			Streaming: &StreamingConfig{Enabled: true, Method: StreamMethodSSE},
		}
		result := base.Merge(override)
		require.NotNil(t, result.Streaming)
		assert.True(t, result.Streaming.Enabled)
		assert.Equal(t, StreamMethodSSE, result.Streaming.Method)
	})

	t.Run("async override", func(t *testing.T) {
		interval := 5.0
		timeout := 120.0
		override := &ExecutionConfig{
			Async: &AsyncConfig{Enabled: true, PollIntervalSeconds: &interval, PollTimeoutSeconds: &timeout},
		}
		result := (&ExecutionConfig{}).Merge(override)
		require.NotNil(t, result.Async)
		assert.True(t, result.Async.Enabled)
	})

	t.Run("three-layer merge with media", func(t *testing.T) {
		agent := &ExecutionConfig{
			Provider: ProviderOpenAI,
			Model:    "dall-e-3",
			Modality: ModalityImage,
			Image:    &ImageConfig{Quality: ImageQualityStandard, Style: ImageStyleNatural},
		}
		skill := &ExecutionConfig{
			Image: &ImageConfig{Quality: ImageQualityHD},
		}
		runtime := &ExecutionConfig{
			Streaming: &StreamingConfig{Enabled: true},
		}

		result := agent.Merge(skill).Merge(runtime)
		assert.Equal(t, ProviderOpenAI, result.Provider)
		assert.Equal(t, ModalityImage, result.Modality)
		assert.Equal(t, ImageQualityHD, result.Image.Quality)
		require.NotNil(t, result.Streaming)
		assert.True(t, result.Streaming.Enabled)
	})
}

func TestExecutionConfig_Getters_MediaFields(t *testing.T) {
	config := &ExecutionConfig{
		Modality:  ModalityImage,
		Image:     &ImageConfig{Quality: ImageQualityHD},
		Audio:     &AudioConfig{Voice: "alloy"},
		Embedding: &EmbeddingConfig{Format: EmbeddingFormatFloat},
		Streaming: &StreamingConfig{Enabled: true},
		Async:     &AsyncConfig{Enabled: true},
	}

	assert.Equal(t, ModalityImage, config.GetModality())
	assert.True(t, config.HasModality())
	assert.NotNil(t, config.GetImage())
	assert.True(t, config.HasImage())
	assert.NotNil(t, config.GetAudio())
	assert.True(t, config.HasAudio())
	assert.NotNil(t, config.GetEmbedding())
	assert.True(t, config.HasEmbedding())
	assert.NotNil(t, config.GetStreaming())
	assert.True(t, config.HasStreaming())
	assert.NotNil(t, config.GetAsync())
	assert.True(t, config.HasAsync())

	// Test nil config
	var nilConfig *ExecutionConfig
	assert.Empty(t, nilConfig.GetModality())
	assert.False(t, nilConfig.HasModality())
	assert.Nil(t, nilConfig.GetImage())
	assert.False(t, nilConfig.HasImage())
	assert.Nil(t, nilConfig.GetAudio())
	assert.False(t, nilConfig.HasAudio())
	assert.Nil(t, nilConfig.GetEmbedding())
	assert.False(t, nilConfig.HasEmbedding())
	assert.Nil(t, nilConfig.GetStreaming())
	assert.False(t, nilConfig.HasStreaming())
	assert.Nil(t, nilConfig.GetAsync())
	assert.False(t, nilConfig.HasAsync())
}

func TestExecutionConfig_ToMap_MediaFields(t *testing.T) {
	config := &ExecutionConfig{
		Modality:  ModalityImage,
		Image:     &ImageConfig{Quality: ImageQualityHD},
		Audio:     &AudioConfig{Voice: "alloy"},
		Embedding: &EmbeddingConfig{Format: EmbeddingFormatFloat},
		Streaming: &StreamingConfig{Enabled: true},
		Async:     &AsyncConfig{Enabled: true},
	}

	m := config.ToMap()

	assert.Equal(t, ModalityImage, m[ParamKeyModality])
	assert.NotNil(t, m[ParamKeyImage])
	assert.NotNil(t, m[ParamKeyAudio])
	assert.NotNil(t, m[ParamKeyEmbedding])
	assert.NotNil(t, m[ParamKeyStreaming])
	assert.NotNil(t, m[ParamKeyAsync])
}

func TestExecutionConfig_ToOpenAI_ImageParams(t *testing.T) {
	numImages := 2
	config := &ExecutionConfig{
		Model:    "dall-e-3",
		Modality: ModalityImage,
		Image: &ImageConfig{
			Size:      "1024x1024",
			Quality:   ImageQualityHD,
			Style:     ImageStyleVivid,
			NumImages: &numImages,
		},
	}

	result := config.ToOpenAI()

	assert.Equal(t, "dall-e-3", result[ParamKeyModel])
	assert.Equal(t, "1024x1024", result[ParamKeyImageSize])
	assert.Equal(t, ImageQualityHD, result[ParamKeyImageQuality])
	assert.Equal(t, ImageStyleVivid, result[ParamKeyImageStyle])
	assert.Equal(t, 2, result[ParamKeyImageN])
}

func TestExecutionConfig_ToOpenAI_AudioTTSParams(t *testing.T) {
	speed := 1.5
	config := &ExecutionConfig{
		Model:    "tts-1-hd",
		Modality: ModalityAudioSpeech,
		Audio: &AudioConfig{
			Voice:        "alloy",
			Speed:        &speed,
			OutputFormat: AudioFormatMP3,
		},
	}

	result := config.ToOpenAI()

	assert.Equal(t, "tts-1-hd", result[ParamKeyModel])
	assert.Equal(t, "alloy", result[ParamKeyVoice])
	assert.Equal(t, 1.5, result[ParamKeySpeed])
	assert.Equal(t, AudioFormatMP3, result[ParamKeyResponseFormat])
}

func TestExecutionConfig_ToOpenAI_AudioResponseFormatCollision(t *testing.T) {
	// When both ResponseFormat (structured output) and Audio.OutputFormat are set,
	// the structured output response_format should take precedence because
	// these target different OpenAI endpoints (chat vs TTS).
	speed := 1.0
	config := &ExecutionConfig{
		Model: "gpt-4",
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
		},
		Audio: &AudioConfig{
			Voice:        "alloy",
			Speed:        &speed,
			OutputFormat: AudioFormatMP3,
		},
	}

	result := config.ToOpenAI()

	// Structured output response_format should win
	rf := result[ParamKeyResponseFormat]
	assert.NotEqual(t, AudioFormatMP3, rf, "audio output format should not overwrite structured output response_format")
}

func TestExecutionConfig_ToOpenAI_EmbeddingParams(t *testing.T) {
	dims := 1536
	config := &ExecutionConfig{
		Model:    "text-embedding-3-small",
		Modality: ModalityEmbedding,
		Embedding: &EmbeddingConfig{
			Dimensions: &dims,
			Format:     EmbeddingFormatFloat,
		},
	}

	result := config.ToOpenAI()

	assert.Equal(t, "text-embedding-3-small", result[ParamKeyModel])
	assert.Equal(t, 1536, result[ParamKeyDimensions])
	assert.Equal(t, EmbeddingFormatFloat, result[ParamKeyEncodingFormat])
}

func TestExecutionConfig_ToOpenAI_Streaming(t *testing.T) {
	config := &ExecutionConfig{
		Model:     "gpt-4",
		Streaming: &StreamingConfig{Enabled: true},
	}

	result := config.ToOpenAI()
	assert.Equal(t, true, result[ParamKeyStream])
}

func TestExecutionConfig_ToOpenAI_StreamingDisabled(t *testing.T) {
	config := &ExecutionConfig{
		Model:     "gpt-4",
		Streaming: &StreamingConfig{Enabled: false},
	}

	result := config.ToOpenAI()
	_, hasStream := result[ParamKeyStream]
	assert.False(t, hasStream)
}

func TestExecutionConfig_ToAnthropic_Streaming(t *testing.T) {
	config := &ExecutionConfig{
		Model:     "claude-3-opus",
		Streaming: &StreamingConfig{Enabled: true},
	}

	result := config.ToAnthropic()
	assert.Equal(t, true, result[ParamKeyStream])
}

func TestExecutionConfig_ToAnthropic_NoMediaParams(t *testing.T) {
	numImages := 2
	config := &ExecutionConfig{
		Model: "claude-3-opus",
		Image: &ImageConfig{
			Quality:   ImageQualityHD,
			NumImages: &numImages,
		},
		Audio: &AudioConfig{Voice: "alloy"},
	}

	result := config.ToAnthropic()

	// No image or audio params should appear
	_, hasSize := result[ParamKeyImageSize]
	_, hasQuality := result[ParamKeyImageQuality]
	_, hasVoice := result[ParamKeyVoice]
	assert.False(t, hasSize)
	assert.False(t, hasQuality)
	assert.False(t, hasVoice)
}

func TestExecutionConfig_ToGemini_ImageParams(t *testing.T) {
	numImages := 4
	config := &ExecutionConfig{
		Model:    "gemini-pro",
		Modality: ModalityImage,
		Image: &ImageConfig{
			AspectRatio: "16:9",
			NumImages:   &numImages,
		},
	}

	result := config.ToGemini()

	genConfig, ok := result[ParamKeyGenerationConfig].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "16:9", genConfig[ParamKeyAspectRatio])
	assert.Equal(t, 4, genConfig[ParamKeyGeminiNumImages])
}

func TestExecutionConfig_ToGemini_Streaming(t *testing.T) {
	config := &ExecutionConfig{
		Model:     "gemini-pro",
		Streaming: &StreamingConfig{Enabled: true},
	}

	result := config.ToGemini()
	assert.Equal(t, true, result[ParamKeyStream])
}

func TestExecutionConfig_ToVLLM_NoMediaParams(t *testing.T) {
	config := &ExecutionConfig{
		Model: "llama-2-7b",
		Image: &ImageConfig{Quality: ImageQualityHD},
		Audio: &AudioConfig{Voice: "alloy"},
	}

	result := config.ToVLLM()

	_, hasQuality := result[ParamKeyImageQuality]
	_, hasVoice := result[ParamKeyVoice]
	assert.False(t, hasQuality)
	assert.False(t, hasVoice)
}

func TestExecutionConfig_ToVLLM_Streaming(t *testing.T) {
	config := &ExecutionConfig{
		Model:     "llama-2-7b",
		Streaming: &StreamingConfig{Enabled: true},
	}

	result := config.ToVLLM()
	assert.Equal(t, true, result[ParamKeyStream])
}

func TestExecutionConfig_GetEffectiveProvider_MediaDoesNotHint(t *testing.T) {
	// Media params should NOT influence provider detection
	config := &ExecutionConfig{
		Image:     &ImageConfig{Quality: ImageQualityHD},
		Audio:     &AudioConfig{Voice: "alloy"},
		Embedding: &EmbeddingConfig{Format: EmbeddingFormatFloat},
		Streaming: &StreamingConfig{Enabled: true},
	}

	assert.Equal(t, "", config.GetEffectiveProvider())
}

// --- E2E YAML roundtrip tests ---

func TestE2E_ImageGeneration_YAMLRoundtrip(t *testing.T) {
	yamlStr := `
modality: image
provider: openai
model: dall-e-3
image:
  size: "1024x1024"
  quality: hd
  style: vivid
  num_images: 2
`
	var config ExecutionConfig
	err := parseYAMLConfig(yamlStr, &config)
	require.NoError(t, err)

	assert.Equal(t, ModalityImage, config.Modality)
	assert.Equal(t, ProviderOpenAI, config.Provider)
	assert.Equal(t, "dall-e-3", config.Model)
	require.NotNil(t, config.Image)
	assert.Equal(t, "1024x1024", config.Image.Size)
	assert.Equal(t, ImageQualityHD, config.Image.Quality)
	assert.Equal(t, ImageStyleVivid, config.Image.Style)
	require.NotNil(t, config.Image.NumImages)
	assert.Equal(t, 2, *config.Image.NumImages)

	// Validate
	assert.NoError(t, config.Validate())

	// Clone and verify independence
	clone := config.Clone()
	assert.Equal(t, config.Image.Quality, clone.Image.Quality)

	// ToOpenAI
	openAI := config.ToOpenAI()
	assert.Equal(t, "1024x1024", openAI[ParamKeyImageSize])
	assert.Equal(t, ImageQualityHD, openAI[ParamKeyImageQuality])
	assert.Equal(t, ImageStyleVivid, openAI[ParamKeyImageStyle])
	assert.Equal(t, 2, openAI[ParamKeyImageN])
}

func TestE2E_AudioTTS_YAMLRoundtrip(t *testing.T) {
	yamlStr := `
modality: audio_speech
provider: openai
model: tts-1-hd
audio:
  voice: alloy
  speed: 1.25
  output_format: mp3
`
	var config ExecutionConfig
	err := parseYAMLConfig(yamlStr, &config)
	require.NoError(t, err)

	assert.Equal(t, ModalityAudioSpeech, config.Modality)
	require.NotNil(t, config.Audio)
	assert.Equal(t, "alloy", config.Audio.Voice)
	assert.Equal(t, 1.25, *config.Audio.Speed)
	assert.Equal(t, AudioFormatMP3, config.Audio.OutputFormat)

	assert.NoError(t, config.Validate())

	openAI := config.ToOpenAI()
	assert.Equal(t, "alloy", openAI[ParamKeyVoice])
	assert.Equal(t, 1.25, openAI[ParamKeySpeed])
	assert.Equal(t, AudioFormatMP3, openAI[ParamKeyResponseFormat])
}

func TestE2E_Embedding_YAMLRoundtrip(t *testing.T) {
	yamlStr := `
modality: embedding
provider: openai
model: text-embedding-3-small
embedding:
  dimensions: 1536
  format: float
`
	var config ExecutionConfig
	err := parseYAMLConfig(yamlStr, &config)
	require.NoError(t, err)

	assert.Equal(t, ModalityEmbedding, config.Modality)
	require.NotNil(t, config.Embedding)
	assert.Equal(t, 1536, *config.Embedding.Dimensions)
	assert.Equal(t, EmbeddingFormatFloat, config.Embedding.Format)

	assert.NoError(t, config.Validate())

	openAI := config.ToOpenAI()
	assert.Equal(t, 1536, openAI[ParamKeyDimensions])
	assert.Equal(t, EmbeddingFormatFloat, openAI[ParamKeyEncodingFormat])
}

func TestE2E_StreamingAsync_YAMLRoundtrip(t *testing.T) {
	yamlStr := `
provider: openai
model: gpt-4
streaming:
  enabled: true
  method: sse
async:
  enabled: true
  poll_interval_seconds: 2.0
  poll_timeout_seconds: 120.0
`
	var config ExecutionConfig
	err := parseYAMLConfig(yamlStr, &config)
	require.NoError(t, err)

	require.NotNil(t, config.Streaming)
	assert.True(t, config.Streaming.Enabled)
	assert.Equal(t, StreamMethodSSE, config.Streaming.Method)

	require.NotNil(t, config.Async)
	assert.True(t, config.Async.Enabled)
	assert.Equal(t, 2.0, *config.Async.PollIntervalSeconds)
	assert.Equal(t, 120.0, *config.Async.PollTimeoutSeconds)

	assert.NoError(t, config.Validate())
}

// parseYAMLConfig is a test helper to unmarshal YAML into ExecutionConfig.
func parseYAMLConfig(yamlStr string, config *ExecutionConfig) error {
	return yaml.Unmarshal([]byte(yamlStr), config)
}

// --- v2.7 Mistral/Cohere Provider Tests ---

func TestExecutionConfig_GetEffectiveProvider_Mistral(t *testing.T) {
	tests := []struct {
		name  string
		model string
	}{
		{"mistral-large", "mistral-large-latest"},
		{"mistral-small", "mistral-small-2402"},
		{"codestral", "codestral-latest"},
		{"pixtral", "pixtral-large-latest"},
		{"ministral", "ministral-8b-latest"},
		{"open-mistral", "open-mistral-nemo"},
		{"open-mixtral", "open-mixtral-8x22b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ExecutionConfig{Model: tt.model}
			assert.Equal(t, ProviderMistral, config.GetEffectiveProvider())
		})
	}
}

func TestExecutionConfig_GetEffectiveProvider_Cohere(t *testing.T) {
	tests := []struct {
		name  string
		model string
	}{
		{"command-r", "command-r-plus"},
		{"command-light", "command-light"},
		{"embed", "embed-v4.0"},
		{"embed-english", "embed-english-v3.0"},
		{"rerank", "rerank-v3.5"},
		{"c4ai", "c4ai-aya-expanse-32b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ExecutionConfig{Model: tt.model}
			assert.Equal(t, ProviderCohere, config.GetEffectiveProvider())
		})
	}
}

// --- v2.7 Mistral/Cohere provider serialization tests ---
// Cross-reference: EmbeddingConfig validation/clone/tomap tests are in prompty.types.media_test.go
// Cross-reference: Model detection helpers tested in prompty.types.media_test.go (TestIsMistralModel, TestIsCohereModel)
// Cross-reference: GeminiTaskType and CohereUpperCase mapping tests in prompty.types.media_test.go

func TestExecutionConfig_ToMistral(t *testing.T) {
	floatPtr := func(v float64) *float64 { return &v }
	intPtr := func(v int) *int { return &v }

	t.Run("standard params", func(t *testing.T) {
		config := &ExecutionConfig{
			Model:         "mistral-large-latest",
			Temperature:   floatPtr(0.7),
			MaxTokens:     intPtr(1000),
			TopP:          floatPtr(0.9),
			Seed:          intPtr(42),
			StopSequences: []string{"END"},
		}

		result := config.ToMistral()
		assert.Equal(t, "mistral-large-latest", result[ParamKeyModel])
		assert.Equal(t, 0.7, result[ParamKeyTemperature])
		assert.Equal(t, 1000, result[ParamKeyMaxTokens])
		assert.Equal(t, 0.9, result[ParamKeyTopP])
		assert.Equal(t, 42, result[ParamKeySeed])
		assert.Equal(t, []string{"END"}, result[ParamKeyStop])
	})

	t.Run("embedding params", func(t *testing.T) {
		config := &ExecutionConfig{
			Model: "mistral-embed",
			Embedding: &EmbeddingConfig{
				Dimensions:  intPtr(1024),
				Format:      EmbeddingFormatFloat,
				OutputDtype: EmbeddingDtypeInt8,
			},
		}

		result := config.ToMistral()
		assert.Equal(t, "mistral-embed", result[ParamKeyModel])
		assert.Equal(t, 1024, result[ParamKeyOutputDimension])
		assert.Equal(t, EmbeddingFormatFloat, result[ParamKeyEncodingFormat])
		assert.Equal(t, EmbeddingDtypeInt8, result[ParamKeyOutputDtype])
	})

	t.Run("nil config", func(t *testing.T) {
		var config *ExecutionConfig
		assert.Nil(t, config.ToMistral())
	})

	t.Run("response format", func(t *testing.T) {
		config := &ExecutionConfig{
			Model: "mistral-large-latest",
			ResponseFormat: &ResponseFormat{
				Type: ResponseFormatJSONObject,
			},
		}

		result := config.ToMistral()
		rf := result[ParamKeyResponseFormat].(map[string]any)
		assert.Equal(t, ResponseFormatJSONObject, rf[SchemaKeyType])
	})

	t.Run("streaming", func(t *testing.T) {
		config := &ExecutionConfig{
			Model:     "mistral-large-latest",
			Streaming: &StreamingConfig{Enabled: true},
		}

		result := config.ToMistral()
		assert.Equal(t, true, result[ParamKeyStream])
	})

	t.Run("provider options", func(t *testing.T) {
		config := &ExecutionConfig{
			Model:           "mistral-large-latest",
			ProviderOptions: map[string]any{"safe_prompt": true},
		}

		result := config.ToMistral()
		assert.Equal(t, true, result["safe_prompt"])
	})
}

func TestExecutionConfig_ToCohere(t *testing.T) {
	floatPtr := func(v float64) *float64 { return &v }
	intPtr := func(v int) *int { return &v }

	t.Run("standard params", func(t *testing.T) {
		config := &ExecutionConfig{
			Model:         "command-r-plus",
			Temperature:   floatPtr(0.5),
			MaxTokens:     intPtr(2000),
			TopP:          floatPtr(0.8),
			TopK:          intPtr(50),
			Seed:          intPtr(42),
			StopSequences: []string{"--"},
		}

		result := config.ToCohere()
		assert.Equal(t, "command-r-plus", result[ParamKeyModel])
		assert.Equal(t, 0.5, result[ParamKeyTemperature])
		assert.Equal(t, 2000, result[ParamKeyMaxTokens])
		assert.Equal(t, 0.8, result[ParamKeyCohereTopP])
		assert.Equal(t, 50, result[ParamKeyCohereTopK])
		assert.Equal(t, 42, result[ParamKeySeed])
		assert.Equal(t, []string{"--"}, result[ParamKeyStopSequences])
	})

	t.Run("embedding params", func(t *testing.T) {
		config := &ExecutionConfig{
			Model: "embed-v4.0",
			Embedding: &EmbeddingConfig{
				Dimensions:  intPtr(1024),
				InputType:   EmbeddingInputTypeSearchDocument,
				OutputDtype: EmbeddingDtypeInt8,
				Truncation:  EmbeddingTruncationEnd,
			},
		}

		result := config.ToCohere()
		assert.Equal(t, "embed-v4.0", result[ParamKeyModel])
		assert.Equal(t, 1024, result[ParamKeyOutputDimension])
		assert.Equal(t, EmbeddingInputTypeSearchDocument, result[ParamKeyInputType])
		assert.Equal(t, []string{EmbeddingDtypeInt8}, result[ParamKeyEmbeddingTypes])
		assert.Equal(t, CohereTruncateEnd, result[ParamKeyTruncate])
	})

	t.Run("truncation none", func(t *testing.T) {
		config := &ExecutionConfig{
			Model: "embed-v4.0",
			Embedding: &EmbeddingConfig{
				Truncation: EmbeddingTruncationNone,
			},
		}

		result := config.ToCohere()
		assert.Equal(t, CohereTruncateNone, result[ParamKeyTruncate])
	})

	t.Run("truncation start", func(t *testing.T) {
		config := &ExecutionConfig{
			Model: "embed-v4.0",
			Embedding: &EmbeddingConfig{
				Truncation: EmbeddingTruncationStart,
			},
		}

		result := config.ToCohere()
		assert.Equal(t, CohereTruncateStart, result[ParamKeyTruncate])
	})

	t.Run("nil config", func(t *testing.T) {
		var config *ExecutionConfig
		assert.Nil(t, config.ToCohere())
	})

	t.Run("streaming", func(t *testing.T) {
		config := &ExecutionConfig{
			Model:     "command-r-plus",
			Streaming: &StreamingConfig{Enabled: true},
		}

		result := config.ToCohere()
		assert.Equal(t, true, result[ParamKeyStream])
	})

	t.Run("provider options", func(t *testing.T) {
		config := &ExecutionConfig{
			Model:           "command-r-plus",
			ProviderOptions: map[string]any{"connectors": []string{"web-search"}},
		}

		result := config.ToCohere()
		assert.Equal(t, []string{"web-search"}, result["connectors"])
	})
}

func TestExecutionConfig_ToGemini_EmbeddingParams(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	t.Run("dimensions as output_dimensionality", func(t *testing.T) {
		config := &ExecutionConfig{
			Model: "gemini-embedding-001",
			Embedding: &EmbeddingConfig{
				Dimensions: intPtr(768),
			},
		}

		result := config.ToGemini()
		genConfig := result[ParamKeyGenerationConfig].(map[string]any)
		assert.Equal(t, 768, genConfig[ParamKeyOutputDimensionality])
	})

	t.Run("input type as task_type", func(t *testing.T) {
		config := &ExecutionConfig{
			Model: "gemini-embedding-001",
			Embedding: &EmbeddingConfig{
				InputType: EmbeddingInputTypeSearchQuery,
			},
		}

		result := config.ToGemini()
		genConfig := result[ParamKeyGenerationConfig].(map[string]any)
		assert.Equal(t, GeminiTaskRetrievalQuery, genConfig[ParamKeyTaskType])
	})

	t.Run("all task type mappings", func(t *testing.T) {
		mappings := map[string]string{
			EmbeddingInputTypeSearchQuery:        GeminiTaskRetrievalQuery,
			EmbeddingInputTypeSearchDocument:     GeminiTaskRetrievalDocument,
			EmbeddingInputTypeSemanticSimilarity: GeminiTaskSemanticSimilarity,
			EmbeddingInputTypeClassification:     GeminiTaskClassification,
			EmbeddingInputTypeClustering:         GeminiTaskClustering,
		}

		for inputType, expectedTaskType := range mappings {
			config := &ExecutionConfig{
				Model: "gemini-embedding-001",
				Embedding: &EmbeddingConfig{
					InputType: inputType,
				},
			}

			result := config.ToGemini()
			genConfig := result[ParamKeyGenerationConfig].(map[string]any)
			assert.Equal(t, expectedTaskType, genConfig[ParamKeyTaskType], "mapping for %s", inputType)
		}
	})

	t.Run("combined dimensions and task_type", func(t *testing.T) {
		config := &ExecutionConfig{
			Model: "gemini-embedding-001",
			Embedding: &EmbeddingConfig{
				Dimensions: intPtr(256),
				InputType:  EmbeddingInputTypeClustering,
			},
		}

		result := config.ToGemini()
		genConfig := result[ParamKeyGenerationConfig].(map[string]any)
		assert.Equal(t, 256, genConfig[ParamKeyOutputDimensionality])
		assert.Equal(t, GeminiTaskClustering, genConfig[ParamKeyTaskType])
	})
}

func TestExecutionConfig_ToVLLM_EmbeddingParams(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }

	t.Run("normalize", func(t *testing.T) {
		config := &ExecutionConfig{
			Model: "qwen3-embedding-0.6B",
			Embedding: &EmbeddingConfig{
				Normalize: boolPtr(true),
			},
		}

		result := config.ToVLLM()
		assert.Equal(t, true, result[ParamKeyNormalize])
	})

	t.Run("normalize false", func(t *testing.T) {
		config := &ExecutionConfig{
			Model: "qwen3-embedding-0.6B",
			Embedding: &EmbeddingConfig{
				Normalize: boolPtr(false),
			},
		}

		result := config.ToVLLM()
		assert.Equal(t, false, result[ParamKeyNormalize])
	})

	t.Run("pooling type", func(t *testing.T) {
		config := &ExecutionConfig{
			Model: "qwen3-embedding-0.6B",
			Embedding: &EmbeddingConfig{
				PoolingType: EmbeddingPoolingMean,
			},
		}

		result := config.ToVLLM()
		assert.Equal(t, EmbeddingPoolingMean, result[ParamKeyPoolingType])
	})

	t.Run("combined normalize and pooling", func(t *testing.T) {
		config := &ExecutionConfig{
			Model: "qwen3-embedding-0.6B",
			Embedding: &EmbeddingConfig{
				Normalize:   boolPtr(true),
				PoolingType: EmbeddingPoolingCLS,
			},
		}

		result := config.ToVLLM()
		assert.Equal(t, true, result[ParamKeyNormalize])
		assert.Equal(t, EmbeddingPoolingCLS, result[ParamKeyPoolingType])
	})

	t.Run("no embedding params without config", func(t *testing.T) {
		config := &ExecutionConfig{
			Model: "qwen3-embedding-0.6B",
		}

		result := config.ToVLLM()
		_, hasNormalize := result[ParamKeyNormalize]
		_, hasPooling := result[ParamKeyPoolingType]
		assert.False(t, hasNormalize)
		assert.False(t, hasPooling)
	})
}

func TestExecutionConfig_ProviderFormat_MistralCohere(t *testing.T) {
	t.Run("mistral returns openai format", func(t *testing.T) {
		config := &ExecutionConfig{
			ResponseFormat: &ResponseFormat{
				Type: ResponseFormatJSONObject,
			},
		}

		result, err := config.ProviderFormat(ProviderMistral)
		require.NoError(t, err)
		assert.Equal(t, ResponseFormatJSONObject, result[SchemaKeyType])
	})

	t.Run("mistral nil response format", func(t *testing.T) {
		config := &ExecutionConfig{}

		result, err := config.ProviderFormat(ProviderMistral)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("cohere returns nil", func(t *testing.T) {
		config := &ExecutionConfig{
			ResponseFormat: &ResponseFormat{
				Type: ResponseFormatJSONObject,
			},
		}

		result, err := config.ProviderFormat(ProviderCohere)
		require.NoError(t, err)
		assert.Nil(t, result)
	})
}

// --- v2.7 E2E YAML Roundtrip Tests ---

func TestE2E_MistralEmbedding_YAMLRoundtrip(t *testing.T) {
	yamlStr := `
modality: embedding
provider: mistral
model: mistral-embed
embedding:
  dimensions: 1024
  format: float
  output_dtype: int8
`
	var config ExecutionConfig
	err := parseYAMLConfig(yamlStr, &config)
	require.NoError(t, err)

	assert.Equal(t, ModalityEmbedding, config.Modality)
	assert.Equal(t, ProviderMistral, config.Provider)
	assert.Equal(t, "mistral-embed", config.Model)
	require.NotNil(t, config.Embedding)
	assert.Equal(t, 1024, *config.Embedding.Dimensions)
	assert.Equal(t, EmbeddingFormatFloat, config.Embedding.Format)
	assert.Equal(t, EmbeddingDtypeInt8, config.Embedding.OutputDtype)

	assert.NoError(t, config.Validate())

	mistral := config.ToMistral()
	assert.Equal(t, 1024, mistral[ParamKeyOutputDimension])
	assert.Equal(t, EmbeddingFormatFloat, mistral[ParamKeyEncodingFormat])
	assert.Equal(t, EmbeddingDtypeInt8, mistral[ParamKeyOutputDtype])
}

func TestE2E_CohereEmbedding_YAMLRoundtrip(t *testing.T) {
	yamlStr := `
modality: embedding
provider: cohere
model: embed-v4.0
embedding:
  dimensions: 1024
  input_type: search_document
  output_dtype: int8
  truncation: end
`
	var config ExecutionConfig
	err := parseYAMLConfig(yamlStr, &config)
	require.NoError(t, err)

	assert.Equal(t, ModalityEmbedding, config.Modality)
	assert.Equal(t, ProviderCohere, config.Provider)
	assert.Equal(t, "embed-v4.0", config.Model)
	require.NotNil(t, config.Embedding)
	assert.Equal(t, 1024, *config.Embedding.Dimensions)
	assert.Equal(t, EmbeddingInputTypeSearchDocument, config.Embedding.InputType)
	assert.Equal(t, EmbeddingDtypeInt8, config.Embedding.OutputDtype)
	assert.Equal(t, EmbeddingTruncationEnd, config.Embedding.Truncation)

	assert.NoError(t, config.Validate())

	cohere := config.ToCohere()
	assert.Equal(t, 1024, cohere[ParamKeyOutputDimension])
	assert.Equal(t, EmbeddingInputTypeSearchDocument, cohere[ParamKeyInputType])
	assert.Equal(t, []string{EmbeddingDtypeInt8}, cohere[ParamKeyEmbeddingTypes])
	assert.Equal(t, CohereTruncateEnd, cohere[ParamKeyTruncate])
}

func TestE2E_GeminiEmbedding_YAMLRoundtrip(t *testing.T) {
	yamlStr := `
modality: embedding
provider: gemini
model: gemini-embedding-001
embedding:
  dimensions: 768
  input_type: search_query
`
	var config ExecutionConfig
	err := parseYAMLConfig(yamlStr, &config)
	require.NoError(t, err)

	assert.Equal(t, ModalityEmbedding, config.Modality)
	assert.Equal(t, ProviderGemini, config.Provider)
	require.NotNil(t, config.Embedding)
	assert.Equal(t, 768, *config.Embedding.Dimensions)
	assert.Equal(t, EmbeddingInputTypeSearchQuery, config.Embedding.InputType)

	assert.NoError(t, config.Validate())

	gemini := config.ToGemini()
	genConfig := gemini[ParamKeyGenerationConfig].(map[string]any)
	assert.Equal(t, 768, genConfig[ParamKeyOutputDimensionality])
	assert.Equal(t, GeminiTaskRetrievalQuery, genConfig[ParamKeyTaskType])
}

func TestE2E_VLLMEmbedding_YAMLRoundtrip(t *testing.T) {
	yamlStr := `
modality: embedding
provider: vllm
model: qwen3-embedding-0.6B
embedding:
  normalize: true
  pooling_type: mean
`
	var config ExecutionConfig
	err := parseYAMLConfig(yamlStr, &config)
	require.NoError(t, err)

	assert.Equal(t, ModalityEmbedding, config.Modality)
	assert.Equal(t, ProviderVLLM, config.Provider)
	require.NotNil(t, config.Embedding)
	require.NotNil(t, config.Embedding.Normalize)
	assert.True(t, *config.Embedding.Normalize)
	assert.Equal(t, EmbeddingPoolingMean, config.Embedding.PoolingType)

	assert.NoError(t, config.Validate())

	vllm := config.ToVLLM()
	assert.Equal(t, true, vllm[ParamKeyNormalize])
	assert.Equal(t, EmbeddingPoolingMean, vllm[ParamKeyPoolingType])
}

func TestE2E_FullEmbeddingConfig_YAMLRoundtrip(t *testing.T) {
	yamlStr := `
modality: embedding
provider: cohere
model: embed-v4.0
embedding:
  dimensions: 1024
  format: float
  input_type: search_document
  output_dtype: int8
  truncation: end
  normalize: true
  pooling_type: mean
`
	var config ExecutionConfig
	err := parseYAMLConfig(yamlStr, &config)
	require.NoError(t, err)

	require.NotNil(t, config.Embedding)
	assert.Equal(t, 1024, *config.Embedding.Dimensions)
	assert.Equal(t, EmbeddingFormatFloat, config.Embedding.Format)
	assert.Equal(t, EmbeddingInputTypeSearchDocument, config.Embedding.InputType)
	assert.Equal(t, EmbeddingDtypeInt8, config.Embedding.OutputDtype)
	assert.Equal(t, EmbeddingTruncationEnd, config.Embedding.Truncation)
	assert.True(t, *config.Embedding.Normalize)
	assert.Equal(t, EmbeddingPoolingMean, config.Embedding.PoolingType)

	assert.NoError(t, config.Validate())
}
