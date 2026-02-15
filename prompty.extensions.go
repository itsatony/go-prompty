package prompty

import (
	"encoding/json"
)

// Extension error messages
const (
	ErrMsgExtensionNotFound   = "extension key not found"
	ErrMsgExtensionCastFailed = "extension type conversion failed"
)

// GetExtensions returns the full extensions map, or nil if empty.
func (p *Prompt) GetExtensions() map[string]any {
	if p == nil {
		return nil
	}
	return p.Extensions
}

// GetExtension returns the value for the given extension key and whether it exists.
func (p *Prompt) GetExtension(key string) (any, bool) {
	if p == nil || p.Extensions == nil {
		return nil, false
	}
	val, ok := p.Extensions[key]
	return val, ok
}

// HasExtension returns true if the given extension key exists.
func (p *Prompt) HasExtension(key string) bool {
	if p == nil || p.Extensions == nil {
		return false
	}
	_, ok := p.Extensions[key]
	return ok
}

// SetExtension sets the given extension key to the given value.
// Initializes the Extensions map if nil.
func (p *Prompt) SetExtension(key string, value any) {
	if p == nil {
		return
	}
	if p.Extensions == nil {
		p.Extensions = make(map[string]any)
	}
	p.Extensions[key] = value
}

// RemoveExtension removes the given extension key.
func (p *Prompt) RemoveExtension(key string) {
	if p == nil || p.Extensions == nil {
		return
	}
	delete(p.Extensions, key)
}

// GetExtensionAs converts the extension value for the given key into the target type T.
// It uses JSON marshal/unmarshal round-trip to convert map[string]any into a typed struct.
// Returns the zero value of T and an error if the key is not found or conversion fails.
func GetExtensionAs[T any](p *Prompt, key string) (T, error) {
	var zero T
	if p == nil || p.Extensions == nil {
		return zero, NewPromptValidationError(ErrMsgExtensionNotFound, key)
	}

	val, ok := p.Extensions[key]
	if !ok {
		return zero, NewPromptValidationError(ErrMsgExtensionNotFound, key)
	}

	// JSON round-trip: marshal the raw value, then unmarshal into T
	data, err := json.Marshal(val)
	if err != nil {
		return zero, NewPromptValidationError(ErrMsgExtensionCastFailed, key)
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return zero, NewPromptValidationError(ErrMsgExtensionCastFailed, key)
	}

	return result, nil
}

// GetStandardFields returns a map of only the Agent Skills genspec standard fields
// that are set on this Prompt.
func (p *Prompt) GetStandardFields() map[string]any {
	if p == nil {
		return nil
	}

	m := make(map[string]any)
	if p.Name != "" {
		m[PromptFieldName] = p.Name
	}
	if p.Description != "" {
		m[PromptFieldDescription] = p.Description
	}
	if p.License != "" {
		m[PromptFieldLicense] = p.License
	}
	if p.Compatibility != "" {
		m[PromptFieldCompatibility] = p.Compatibility
	}
	if p.AllowedTools != "" {
		m[PromptFieldAllowedTools] = p.AllowedTools
	}
	if len(p.Metadata) > 0 {
		m[PromptFieldMetadata] = p.Metadata
	}
	if len(p.Inputs) > 0 {
		m[PromptFieldInputs] = p.Inputs
	}
	if len(p.Outputs) > 0 {
		m[PromptFieldOutputs] = p.Outputs
	}
	if len(p.Sample) > 0 {
		m[PromptFieldSample] = p.Sample
	}

	return m
}

// GetPromptyFields returns a map of only the go-prompty extension fields
// that are set on this Prompt.
func (p *Prompt) GetPromptyFields() map[string]any {
	if p == nil {
		return nil
	}

	m := make(map[string]any)
	if p.Type != "" {
		m[PromptFieldType] = string(p.Type)
	}
	if p.Execution != nil {
		m[PromptFieldExecution] = p.Execution
	}
	if len(p.Skills) > 0 {
		m[PromptFieldSkills] = p.Skills
	}
	if p.Tools != nil {
		m[PromptFieldTools] = p.Tools
	}
	if len(p.Context) > 0 {
		m[PromptFieldContext] = p.Context
	}
	if p.Constraints != nil {
		m[PromptFieldConstraints] = p.Constraints
	}
	if len(p.Messages) > 0 {
		m[PromptFieldMessages] = p.Messages
	}

	return m
}
