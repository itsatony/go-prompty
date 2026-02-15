package prompty

import "encoding/json"

// ToolDefinition defines a tool/function that can be called by the model.
type ToolDefinition struct {
	// Type is always "function" for function calling
	Type string `yaml:"type" json:"type"`
	// Function defines the callable function
	Function *FunctionDef `yaml:"function" json:"function"`
}

// FunctionDef defines a callable function for tool use.
type FunctionDef struct {
	// Name of the function
	Name string `yaml:"name" json:"name"`
	// Description of what the function does
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	// Parameters schema for function arguments
	Parameters map[string]any `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	// Returns schema for function return value (v2.1)
	Returns map[string]any `yaml:"returns,omitempty" json:"returns,omitempty"`
	// Strict enables strict parameter validation
	Strict bool `yaml:"strict,omitempty" json:"strict,omitempty"`
}

// StreamingConfig configures streaming behavior.
// StreamingConfig is safe for concurrent reads. Callers must not modify the config after
// passing it to an ExecutionConfig; use Clone() to create an independent copy if mutation is needed.
type StreamingConfig struct {
	// Enabled indicates whether streaming is enabled
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Method is the streaming transport method: "sse" or "websocket"
	Method string `yaml:"method,omitempty" json:"method,omitempty"`
}

// Validate checks the streaming config for consistency.
func (c *StreamingConfig) Validate() error {
	if c == nil {
		return nil
	}
	if c.Enabled && c.Method != "" && !isValidStreamMethod(c.Method) {
		return NewPromptValidationError(ErrMsgStreamInvalidMethod, "")
	}
	return nil
}

// Clone creates a deep copy of the streaming config.
func (c *StreamingConfig) Clone() *StreamingConfig {
	if c == nil {
		return nil
	}
	return &StreamingConfig{
		Enabled: c.Enabled,
		Method:  c.Method,
	}
}

// ToMap converts the streaming config to a parameter map.
func (c *StreamingConfig) ToMap() map[string]any {
	if c == nil {
		return nil
	}
	result := map[string]any{
		ParamKeyEnabled: c.Enabled,
	}
	if c.Method != "" {
		result[ParamKeyStreamMethod] = c.Method
	}
	return result
}

// RetryConfig configures retry behavior for API calls.
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts
	MaxAttempts int `yaml:"max_attempts,omitempty" json:"max_attempts,omitempty"`
	// Backoff strategy: "linear" or "exponential"
	Backoff string `yaml:"backoff,omitempty" json:"backoff,omitempty"`
}

// PromptCacheConfig configures prompt/inference caching settings.
// This is different from storage.CacheConfig which handles template storage caching.
type PromptCacheConfig struct {
	// SystemPrompt indicates whether to cache the system prompt
	SystemPrompt bool `yaml:"system_prompt,omitempty" json:"system_prompt,omitempty"`
	// TTL is the cache time-to-live in seconds
	TTL int `yaml:"ttl,omitempty" json:"ttl,omitempty"`
}

// ModelParameters holds provider-agnostic inference parameters.
// Pointer types are used to distinguish between unset and zero values.
type ModelParameters struct {
	// Temperature controls randomness (0.0-2.0)
	Temperature *float64 `yaml:"temperature,omitempty" json:"temperature,omitempty"`
	// MaxTokens limits the response length
	MaxTokens *int `yaml:"max_tokens,omitempty" json:"max_tokens,omitempty"`
	// TopP is nucleus sampling probability (0.0-1.0)
	TopP *float64 `yaml:"top_p,omitempty" json:"top_p,omitempty"`
	// FrequencyPenalty reduces repetition (-2.0 to 2.0)
	FrequencyPenalty *float64 `yaml:"frequency_penalty,omitempty" json:"frequency_penalty,omitempty"`
	// PresencePenalty encourages new topics (-2.0 to 2.0)
	PresencePenalty *float64 `yaml:"presence_penalty,omitempty" json:"presence_penalty,omitempty"`
	// Stop sequences that halt generation
	Stop []string `yaml:"stop,omitempty" json:"stop,omitempty"`
	// Seed for deterministic sampling (if supported)
	Seed *int64 `yaml:"seed,omitempty" json:"seed,omitempty"`
}

// ToMap converts ModelParameters to a map for passing to LLM clients.
// Only includes parameters that were explicitly set.
func (p *ModelParameters) ToMap() map[string]any {
	if p == nil {
		return nil
	}

	result := make(map[string]any)

	if p.Temperature != nil {
		result[ParamKeyTemperature] = *p.Temperature
	}
	if p.MaxTokens != nil {
		result[ParamKeyMaxTokens] = *p.MaxTokens
	}
	if p.TopP != nil {
		result[ParamKeyTopP] = *p.TopP
	}
	if p.FrequencyPenalty != nil {
		result[ParamKeyFrequencyPenalty] = *p.FrequencyPenalty
	}
	if p.PresencePenalty != nil {
		result[ParamKeyPresencePenalty] = *p.PresencePenalty
	}
	if len(p.Stop) > 0 {
		result[ParamKeyStop] = p.Stop
	}
	if p.Seed != nil {
		result[ParamKeySeed] = *p.Seed
	}

	return result
}

// ToOpenAITool converts FunctionDef to OpenAI tool calling format.
func (f *FunctionDef) ToOpenAITool() map[string]any {
	if f == nil {
		return nil
	}

	funcDef := map[string]any{
		AttrName: f.Name,
	}
	if f.Description != "" {
		funcDef[SchemaKeyDescription] = f.Description
	}
	if f.Parameters != nil {
		params := copySchema(f.Parameters)
		if f.Strict {
			ensureAdditionalPropertiesFalse(params)
		}
		funcDef["parameters"] = params
	}
	if f.Strict {
		funcDef[SchemaKeyStrict] = true
	}

	return map[string]any{
		SchemaKeyType: "function",
		"function":    funcDef,
	}
}

// ToAnthropicTool converts FunctionDef to Anthropic tool use format.
func (f *FunctionDef) ToAnthropicTool() map[string]any {
	if f == nil {
		return nil
	}

	tool := map[string]any{
		AttrName: f.Name,
	}
	if f.Description != "" {
		tool[SchemaKeyDescription] = f.Description
	}
	if f.Parameters != nil {
		params := copySchema(f.Parameters)
		ensureAdditionalPropertiesFalse(params)
		tool["input_schema"] = params
	}

	return tool
}

// ToJSON returns the JSON representation of the FunctionDef.
func (f *FunctionDef) ToJSON() (string, error) {
	if f == nil {
		return "", nil
	}
	data, err := json.Marshal(f)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
