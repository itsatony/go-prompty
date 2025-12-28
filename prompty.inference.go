package prompty

import (
	"encoding/json"
	"fmt"
)

// InferenceConfig represents parsed inference configuration from a template's config block.
// It provides a structured, type-safe way to access model parameters and metadata.
type InferenceConfig struct {
	// Metadata fields
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Version     string   `json:"version,omitempty"`
	Authors     []string `json:"authors,omitempty"`
	Tags        []string `json:"tags,omitempty"`

	// Model configuration
	Model *ModelConfig `json:"model,omitempty"`

	// Schema definitions
	Inputs  map[string]*InputDef  `json:"inputs,omitempty"`
	Outputs map[string]*OutputDef `json:"outputs,omitempty"`

	// Sample data for testing
	Sample map[string]any `json:"sample,omitempty"`
}

// ModelConfig represents the inference model configuration.
type ModelConfig struct {
	// API type: "chat" or "completion"
	API string `json:"api,omitempty"`
	// Provider hint (e.g., "openai", "anthropic", "azure")
	Provider string `json:"provider,omitempty"`
	// Model identifier (e.g., "gpt-4", "claude-3-opus")
	Name string `json:"name,omitempty"`
	// Model parameters
	Parameters *ModelParameters `json:"parameters,omitempty"`
}

// ModelParameters holds provider-agnostic inference parameters.
// Pointer types are used to distinguish between unset and zero values.
type ModelParameters struct {
	// Temperature controls randomness (0.0-2.0)
	Temperature *float64 `json:"temperature,omitempty"`
	// MaxTokens limits the response length
	MaxTokens *int `json:"max_tokens,omitempty"`
	// TopP is nucleus sampling probability (0.0-1.0)
	TopP *float64 `json:"top_p,omitempty"`
	// FrequencyPenalty reduces repetition (-2.0 to 2.0)
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`
	// PresencePenalty encourages new topics (-2.0 to 2.0)
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`
	// Stop sequences that halt generation
	Stop []string `json:"stop,omitempty"`
	// Seed for deterministic sampling (if supported)
	Seed *int64 `json:"seed,omitempty"`
}

// InputDef defines an expected input parameter.
type InputDef struct {
	// Type: "string", "number", "boolean", "array", "object"
	Type string `json:"type"`
	// Description of the input
	Description string `json:"description,omitempty"`
	// Required indicates if this input must be provided
	Required bool `json:"required,omitempty"`
	// Default value if not provided
	Default any `json:"default,omitempty"`
}

// OutputDef defines an expected output.
type OutputDef struct {
	// Type: "string", "number", "boolean", "array", "object"
	Type string `json:"type"`
	// Description of the output
	Description string `json:"description,omitempty"`
}

// ParseInferenceConfig parses JSON into an InferenceConfig.
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
