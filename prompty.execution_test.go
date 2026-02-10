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
		Model:    "claude-3-opus",
		Seed:     &seed,
		MinP:     &minP,
		Logprobs: &logprobs,
		LogitBias: map[string]float64{"100": 5.0},
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
	genConfig, ok := result["generationConfig"]
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
