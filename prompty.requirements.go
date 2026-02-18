package prompty

// ExecutionRequirements declares metadata about what an execution slot needs.
// This includes the expected modality, provider binding strength, capability
// requirements, and cost/latency hints.
//
// go-prompty does NOT enforce these requirements; they are declarative metadata
// for orchestration layers to use in routing, scheduling, and cost estimation.
type ExecutionRequirements struct {
	// Modality is the expected output modality (e.g., "text", "image", "embedding").
	Modality string `yaml:"modality,omitempty" json:"modality,omitempty"`
	// ProviderBinding indicates how tightly the execution is bound to its provider.
	// Values: "required" (must use this provider), "preferred" (prefer but can substitute),
	// "any" (no preference). Empty string is treated as unset.
	ProviderBinding string `yaml:"provider_binding,omitempty" json:"provider_binding,omitempty"`
	// Capabilities lists required provider capabilities (e.g., ["image_generation", "function_calling"]).
	Capabilities []string `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	// EstimatedCost is a human-readable cost estimate (e.g., "$0.04", "~$0.001/1K tokens").
	EstimatedCost string `yaml:"estimated_cost,omitempty" json:"estimated_cost,omitempty"`
	// EstimatedLatencyMs is the expected latency in milliseconds (must be >= 0).
	EstimatedLatencyMs int `yaml:"estimated_latency_ms,omitempty" json:"estimated_latency_ms,omitempty"`
}

// Validate checks the execution requirements for consistency.
func (r *ExecutionRequirements) Validate() error {
	if r == nil {
		return nil
	}

	if r.ProviderBinding != "" && !isValidProviderBinding(r.ProviderBinding) {
		return NewPromptValidationError(ErrMsgInvalidProviderBinding, "")
	}

	if r.EstimatedLatencyMs < 0 {
		return NewPromptValidationError(ErrMsgEstimatedLatencyNegative, "")
	}

	if r.Modality != "" && !isValidModality(r.Modality) {
		return NewPromptValidationError(ErrMsgInvalidModality, "")
	}

	return nil
}

// Clone creates a deep copy of the ExecutionRequirements.
func (r *ExecutionRequirements) Clone() *ExecutionRequirements {
	if r == nil {
		return nil
	}

	clone := &ExecutionRequirements{
		Modality:           r.Modality,
		ProviderBinding:    r.ProviderBinding,
		EstimatedCost:      r.EstimatedCost,
		EstimatedLatencyMs: r.EstimatedLatencyMs,
	}

	if r.Capabilities != nil {
		clone.Capabilities = make([]string, len(r.Capabilities))
		copy(clone.Capabilities, r.Capabilities)
	}

	return clone
}

// GetModality returns the modality or empty string if not set.
func (r *ExecutionRequirements) GetModality() string {
	if r == nil {
		return ""
	}
	return r.Modality
}

// HasModality returns true if modality is set.
func (r *ExecutionRequirements) HasModality() bool {
	return r != nil && r.Modality != ""
}

// GetProviderBinding returns the provider binding or empty string if not set.
func (r *ExecutionRequirements) GetProviderBinding() string {
	if r == nil {
		return ""
	}
	return r.ProviderBinding
}

// HasProviderBinding returns true if provider binding is set.
func (r *ExecutionRequirements) HasProviderBinding() bool {
	return r != nil && r.ProviderBinding != ""
}

// GetCapabilities returns the capabilities list.
func (r *ExecutionRequirements) GetCapabilities() []string {
	if r == nil {
		return nil
	}
	return r.Capabilities
}

// HasCapabilities returns true if any capabilities are defined.
func (r *ExecutionRequirements) HasCapabilities() bool {
	return r != nil && len(r.Capabilities) > 0
}

// GetEstimatedCost returns the estimated cost string.
func (r *ExecutionRequirements) GetEstimatedCost() string {
	if r == nil {
		return ""
	}
	return r.EstimatedCost
}

// HasEstimatedCost returns true if estimated cost is set.
func (r *ExecutionRequirements) HasEstimatedCost() bool {
	return r != nil && r.EstimatedCost != ""
}

// GetEstimatedLatencyMs returns the estimated latency in milliseconds.
func (r *ExecutionRequirements) GetEstimatedLatencyMs() int {
	if r == nil {
		return 0
	}
	return r.EstimatedLatencyMs
}

// HasEstimatedLatencyMs returns true if estimated latency is set (> 0).
func (r *ExecutionRequirements) HasEstimatedLatencyMs() bool {
	return r != nil && r.EstimatedLatencyMs > 0
}

// isValidProviderBinding checks if the given string is a valid provider binding mode.
func isValidProviderBinding(b string) bool {
	switch b {
	case ProviderBindingRequired, ProviderBindingPreferred, ProviderBindingAny:
		return true
	default:
		return false
	}
}
