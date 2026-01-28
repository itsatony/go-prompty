package prompty

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// InferenceConfig represents parsed inference configuration from a template's YAML frontmatter.
// It provides a structured, type-safe way to access model parameters and metadata.
type InferenceConfig struct {
	// Metadata fields
	Name        string   `yaml:"name,omitempty" json:"name,omitempty"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Version     string   `yaml:"version,omitempty" json:"version,omitempty"`
	Authors     []string `yaml:"authors,omitempty" json:"authors,omitempty"`
	Tags        []string `yaml:"tags,omitempty" json:"tags,omitempty"`

	// Model configuration
	Model *ModelConfig `yaml:"model,omitempty" json:"model,omitempty"`

	// Schema definitions
	Inputs  map[string]*InputDef  `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Outputs map[string]*OutputDef `yaml:"outputs,omitempty" json:"outputs,omitempty"`

	// Sample data for testing
	Sample map[string]any `yaml:"sample,omitempty" json:"sample,omitempty"`

	// Retry configuration
	Retry *RetryConfig `yaml:"retry,omitempty" json:"retry,omitempty"`

	// Cache configuration for prompt/inference caching
	Cache *PromptCacheConfig `yaml:"cache,omitempty" json:"cache,omitempty"`
}

// ModelConfig represents the inference model configuration.
type ModelConfig struct {
	// API type: "chat" or "completion"
	API string `yaml:"api,omitempty" json:"api,omitempty"`
	// Provider hint (e.g., "openai", "anthropic", "azure")
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty"`
	// Model identifier (e.g., "gpt-4", "claude-3-opus")
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	// Model parameters
	Parameters *ModelParameters `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	// Response format for structured outputs
	ResponseFormat *ResponseFormat `yaml:"response_format,omitempty" json:"response_format,omitempty"`
	// Tool/function definitions for tool calling
	Tools []*ToolDefinition `yaml:"tools,omitempty" json:"tools,omitempty"`
	// Tool choice strategy: "auto", "none", "required", or specific function
	ToolChoice any `yaml:"tool_choice,omitempty" json:"tool_choice,omitempty"`
	// Streaming configuration
	Streaming *StreamingConfig `yaml:"streaming,omitempty" json:"streaming,omitempty"`
	// Context window size hint (token budget)
	ContextWindow *int `yaml:"context_window,omitempty" json:"context_window,omitempty"`
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

// ResponseFormat configures structured output enforcement.
type ResponseFormat struct {
	// Type: "text", "json_object", or "json_schema"
	Type string `yaml:"type" json:"type"`
	// JSONSchema for structured output validation (when type is "json_schema")
	JSONSchema *JSONSchemaSpec `yaml:"json_schema,omitempty" json:"json_schema,omitempty"`
}

// JSONSchemaSpec defines a JSON schema for structured outputs.
type JSONSchemaSpec struct {
	// Name of the schema (required for API calls)
	Name string `yaml:"name" json:"name"`
	// Description of what the schema represents
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	// Schema is the JSON schema definition
	Schema map[string]any `yaml:"schema" json:"schema"`
	// Strict enables strict schema validation
	Strict bool `yaml:"strict,omitempty" json:"strict,omitempty"`
}

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
	// Strict enables strict parameter validation
	Strict bool `yaml:"strict,omitempty" json:"strict,omitempty"`
}

// StreamingConfig configures streaming behavior.
type StreamingConfig struct {
	// Enabled indicates whether streaming is enabled
	Enabled bool `yaml:"enabled" json:"enabled"`
	// ChunkSize is the optional preferred chunk size
	ChunkSize int `yaml:"chunk_size,omitempty" json:"chunk_size,omitempty"`
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

// Message represents a conversation message for chat APIs.
type Message struct {
	// Role of the message sender: "system", "user", "assistant", or "tool"
	Role string `yaml:"role" json:"role"`
	// Content of the message
	Content string `yaml:"content" json:"content"`
	// Cache indicates whether this message should be cached
	Cache bool `yaml:"cache,omitempty" json:"cache,omitempty"`
}

// InputDef defines an expected input parameter.
type InputDef struct {
	// Type: "string", "number", "boolean", "array", "object"
	Type string `yaml:"type" json:"type"`
	// Description of the input
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	// Required indicates if this input must be provided
	Required bool `yaml:"required,omitempty" json:"required,omitempty"`
	// Default value if not provided
	Default any `yaml:"default,omitempty" json:"default,omitempty"`
}

// OutputDef defines an expected output.
type OutputDef struct {
	// Type: "string", "number", "boolean", "array", "object"
	Type string `yaml:"type" json:"type"`
	// Description of the output
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// ParseYAMLInferenceConfig parses YAML into an InferenceConfig.
// Returns an error if the YAML data exceeds DefaultMaxFrontmatterSize (DoS protection).
func ParseYAMLInferenceConfig(yamlData string) (*InferenceConfig, error) {
	if yamlData == "" {
		return nil, nil
	}

	// Check size limit to prevent DoS via large YAML
	if len(yamlData) > DefaultMaxFrontmatterSize {
		return nil, NewFrontmatterError(ErrMsgFrontmatterTooLarge, Position{Line: 1, Column: 1}, nil)
	}

	var config InferenceConfig
	if err := yaml.Unmarshal([]byte(yamlData), &config); err != nil {
		return nil, NewFrontmatterParseError(err)
	}
	return &config, nil
}

// ParseInferenceConfig parses JSON into an InferenceConfig.
// Deprecated: Use ParseYAMLInferenceConfig for YAML frontmatter.
func ParseInferenceConfig(jsonData string) (*InferenceConfig, error) {
	if jsonData == "" {
		return nil, nil
	}

	var config InferenceConfig
	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		return nil, NewConfigBlockParseError(err)
	}
	return &config, nil
}

// GetTemperature returns the temperature parameter and whether it was set.
func (c *InferenceConfig) GetTemperature() (float64, bool) {
	if c == nil || c.Model == nil || c.Model.Parameters == nil || c.Model.Parameters.Temperature == nil {
		return 0, false
	}
	return *c.Model.Parameters.Temperature, true
}

// GetMaxTokens returns the max_tokens parameter and whether it was set.
func (c *InferenceConfig) GetMaxTokens() (int, bool) {
	if c == nil || c.Model == nil || c.Model.Parameters == nil || c.Model.Parameters.MaxTokens == nil {
		return 0, false
	}
	return *c.Model.Parameters.MaxTokens, true
}

// GetTopP returns the top_p parameter and whether it was set.
func (c *InferenceConfig) GetTopP() (float64, bool) {
	if c == nil || c.Model == nil || c.Model.Parameters == nil || c.Model.Parameters.TopP == nil {
		return 0, false
	}
	return *c.Model.Parameters.TopP, true
}

// GetFrequencyPenalty returns the frequency_penalty parameter and whether it was set.
func (c *InferenceConfig) GetFrequencyPenalty() (float64, bool) {
	if c == nil || c.Model == nil || c.Model.Parameters == nil || c.Model.Parameters.FrequencyPenalty == nil {
		return 0, false
	}
	return *c.Model.Parameters.FrequencyPenalty, true
}

// GetPresencePenalty returns the presence_penalty parameter and whether it was set.
func (c *InferenceConfig) GetPresencePenalty() (float64, bool) {
	if c == nil || c.Model == nil || c.Model.Parameters == nil || c.Model.Parameters.PresencePenalty == nil {
		return 0, false
	}
	return *c.Model.Parameters.PresencePenalty, true
}

// GetStopSequences returns the stop sequences or nil if not set.
func (c *InferenceConfig) GetStopSequences() []string {
	if c == nil || c.Model == nil || c.Model.Parameters == nil {
		return nil
	}
	return c.Model.Parameters.Stop
}

// GetSeed returns the seed parameter and whether it was set.
func (c *InferenceConfig) GetSeed() (int64, bool) {
	if c == nil || c.Model == nil || c.Model.Parameters == nil || c.Model.Parameters.Seed == nil {
		return 0, false
	}
	return *c.Model.Parameters.Seed, true
}

// GetModelName returns the model name or empty string if not set.
func (c *InferenceConfig) GetModelName() string {
	if c == nil || c.Model == nil {
		return ""
	}
	return c.Model.Name
}

// GetAPIType returns the API type ("chat" or "completion") or empty string if not set.
func (c *InferenceConfig) GetAPIType() string {
	if c == nil || c.Model == nil {
		return ""
	}
	return c.Model.API
}

// GetProvider returns the provider hint or empty string if not set.
func (c *InferenceConfig) GetProvider() string {
	if c == nil || c.Model == nil {
		return ""
	}
	return c.Model.Provider
}

// GetSampleData returns the sample data map or nil if not set.
func (c *InferenceConfig) GetSampleData() map[string]any {
	if c == nil {
		return nil
	}
	return c.Sample
}

// GetResponseFormat returns the response format configuration or nil if not set.
func (c *InferenceConfig) GetResponseFormat() *ResponseFormat {
	if c == nil || c.Model == nil {
		return nil
	}
	return c.Model.ResponseFormat
}

// GetTools returns the tool definitions or nil if not set.
func (c *InferenceConfig) GetTools() []*ToolDefinition {
	if c == nil || c.Model == nil {
		return nil
	}
	return c.Model.Tools
}

// GetToolChoice returns the tool choice configuration or nil if not set.
func (c *InferenceConfig) GetToolChoice() any {
	if c == nil || c.Model == nil {
		return nil
	}
	return c.Model.ToolChoice
}

// GetStreaming returns the streaming configuration or nil if not set.
func (c *InferenceConfig) GetStreaming() *StreamingConfig {
	if c == nil || c.Model == nil {
		return nil
	}
	return c.Model.Streaming
}

// GetContextWindow returns the context window size and whether it was set.
func (c *InferenceConfig) GetContextWindow() (int, bool) {
	if c == nil || c.Model == nil || c.Model.ContextWindow == nil {
		return 0, false
	}
	return *c.Model.ContextWindow, true
}

// GetRetry returns the retry configuration or nil if not set.
func (c *InferenceConfig) GetRetry() *RetryConfig {
	if c == nil {
		return nil
	}
	return c.Retry
}

// GetCache returns the prompt cache configuration or nil if not set.
func (c *InferenceConfig) GetCache() *PromptCacheConfig {
	if c == nil {
		return nil
	}
	return c.Cache
}

// HasModel returns true if model configuration is present.
func (c *InferenceConfig) HasModel() bool {
	return c != nil && c.Model != nil
}

// HasInputs returns true if input definitions are present.
func (c *InferenceConfig) HasInputs() bool {
	return c != nil && len(c.Inputs) > 0
}

// HasOutputs returns true if output definitions are present.
func (c *InferenceConfig) HasOutputs() bool {
	return c != nil && len(c.Outputs) > 0
}

// HasSample returns true if sample data is present.
func (c *InferenceConfig) HasSample() bool {
	return c != nil && len(c.Sample) > 0
}

// HasResponseFormat returns true if response format is configured.
func (c *InferenceConfig) HasResponseFormat() bool {
	return c != nil && c.Model != nil && c.Model.ResponseFormat != nil
}

// HasTools returns true if tools are configured.
func (c *InferenceConfig) HasTools() bool {
	return c != nil && c.Model != nil && len(c.Model.Tools) > 0
}

// HasStreaming returns true if streaming is configured.
func (c *InferenceConfig) HasStreaming() bool {
	return c != nil && c.Model != nil && c.Model.Streaming != nil
}

// HasRetry returns true if retry configuration is present.
func (c *InferenceConfig) HasRetry() bool {
	return c != nil && c.Retry != nil
}

// HasCache returns true if cache configuration is present.
func (c *InferenceConfig) HasCache() bool {
	return c != nil && c.Cache != nil
}

// ValidateInputs validates the provided data against the input definitions.
// Returns an error if any required input is missing or has wrong type.
func (c *InferenceConfig) ValidateInputs(data map[string]any) error {
	if c == nil || c.Inputs == nil {
		return nil
	}

	for name, def := range c.Inputs {
		val, exists := data[name]

		// Check required
		if def.Required && !exists {
			return NewRequiredInputMissingError(name)
		}

		// If not required and not present, skip type check
		if !exists {
			continue
		}

		// Type validation
		if err := validateInputType(name, val, def.Type); err != nil {
			return err
		}
	}

	return nil
}

// validateInputType checks if the value matches the expected type.
func validateInputType(name string, val any, expectedType string) error {
	if val == nil {
		return nil // nil is allowed for optional inputs
	}

	var valid bool
	switch expectedType {
	case SchemaTypeString:
		_, valid = val.(string)
	case SchemaTypeNumber:
		switch val.(type) {
		case int, int64, float64, float32:
			valid = true
		}
	case SchemaTypeBoolean:
		_, valid = val.(bool)
	case SchemaTypeArray:
		switch val.(type) {
		case []any, []string, []int, []float64:
			valid = true
		}
	case SchemaTypeObject:
		switch val.(type) {
		case map[string]any, map[string]string:
			valid = true
		}
	default:
		// Unknown type, accept anything
		valid = true
	}

	if !valid {
		reason := fmt.Sprintf(ErrFmtTypeMismatch, expectedType, fmt.Sprintf("%T", val))
		return NewInputValidationError(name, reason)
	}

	return nil
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

// JSON returns the JSON representation of the config.
func (c *InferenceConfig) JSON() (string, error) {
	if c == nil {
		return "", nil
	}
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// JSONPretty returns the pretty-printed JSON representation of the config.
func (c *InferenceConfig) JSONPretty() (string, error) {
	if c == nil {
		return "", nil
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// YAML returns the YAML representation of the config.
func (c *InferenceConfig) YAML() (string, error) {
	if c == nil {
		return "", nil
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
