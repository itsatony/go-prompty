package prompty

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// Serialization error messages
const (
	ErrMsgSerializeFailed = "serialization failed"
	ErrMsgSerializeYAML   = "YAML marshaling failed"
)

// SerializeOptions configures prompt serialization.
type SerializeOptions struct {
	// IncludeExecution includes the execution config in output
	IncludeExecution bool
	// IncludeExtensions includes extension fields (non-standard YAML keys) in output
	IncludeExtensions bool
	// IncludeAgentFields includes agent-specific fields (type, skills, tools, constraints, messages)
	IncludeAgentFields bool
	// IncludeContext includes the context map in output
	IncludeContext bool
}

// DefaultSerializeOptions returns the default serialization options (all included).
func DefaultSerializeOptions() *SerializeOptions {
	return &SerializeOptions{
		IncludeExecution:   true,
		IncludeExtensions:  true,
		IncludeAgentFields: true,
		IncludeContext:     true,
	}
}

// AgentSkillsExportOptions returns options for Agent Skills compatible export.
// This strips all non-standard fields.
func AgentSkillsExportOptions() *SerializeOptions {
	return &SerializeOptions{
		IncludeExecution:   false,
		IncludeExtensions:  false,
		IncludeAgentFields: false,
		IncludeContext:     false,
	}
}

// Serialize outputs the Prompt as a YAML frontmatter + body document.
// If opts is nil, DefaultSerializeOptions is used.
func (p *Prompt) Serialize(opts *SerializeOptions) ([]byte, error) {
	if p == nil {
		return nil, nil
	}

	if opts == nil {
		opts = DefaultSerializeOptions()
	}

	// Build a serializable struct based on options
	exportData := p.buildSerializeMap(opts)

	yamlBytes, err := yaml.Marshal(exportData)
	if err != nil {
		return nil, NewCompilationError(ErrMsgSerializeYAML, err)
	}

	var sb strings.Builder
	sb.WriteString(YAMLFrontmatterDelimiter)
	sb.WriteString("\n")
	sb.Write(yamlBytes)
	sb.WriteString(YAMLFrontmatterDelimiter)
	sb.WriteString("\n")
	if p.Body != "" {
		sb.WriteString(p.Body)
	}

	return []byte(sb.String()), nil
}

// ExportAgentSkill serializes the prompt with only Agent Skills compatible fields.
func (p *Prompt) ExportAgentSkill() ([]byte, error) {
	return p.Serialize(AgentSkillsExportOptions())
}

// ExportFull serializes the prompt with all fields included.
func (p *Prompt) ExportFull() ([]byte, error) {
	return p.Serialize(DefaultSerializeOptions())
}

// knownPromptFields is the set of all known Prompt struct YAML field names.
// Extensions with these keys are skipped during serialization to prevent overwriting.
var knownPromptFields = map[string]bool{
	PromptFieldName:          true,
	PromptFieldDescription:   true,
	PromptFieldLicense:       true,
	PromptFieldCompatibility: true,
	PromptFieldAllowedTools:  true,
	PromptFieldMetadata:      true,
	PromptFieldInputs:        true,
	PromptFieldOutputs:       true,
	PromptFieldSample:        true,
	PromptFieldType:          true,
	PromptFieldExecution:     true,
	PromptFieldExtensions:    true,
	PromptFieldSkills:        true,
	PromptFieldTools:         true,
	PromptFieldContext:       true,
	PromptFieldConstraints:   true,
	PromptFieldMessages:      true,
}

// buildSerializeMap creates an ordered map for YAML serialization.
func (p *Prompt) buildSerializeMap(opts *SerializeOptions) map[string]any {
	m := make(map[string]any)

	// Standard fields (always included)
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

	// Type (include if agent fields are included or if type is explicitly set)
	if opts.IncludeAgentFields && p.Type != "" {
		m[PromptFieldType] = string(p.Type)
	}

	// Execution config
	if opts.IncludeExecution && p.Execution != nil {
		m[PromptFieldExecution] = p.Execution
	}

	// Extensions (non-standard YAML keys, written as top-level keys)
	// Skip keys that match known Prompt fields to prevent overwriting.
	if opts.IncludeExtensions && len(p.Extensions) > 0 {
		for k, v := range p.Extensions {
			if !knownPromptFields[k] {
				m[k] = v
			}
		}
	}

	// Inputs/Outputs (always included)
	if len(p.Inputs) > 0 {
		m[PromptFieldInputs] = p.Inputs
	}
	if len(p.Outputs) > 0 {
		m[PromptFieldOutputs] = p.Outputs
	}

	// Sample (always included)
	if len(p.Sample) > 0 {
		m[PromptFieldSample] = p.Sample
	}

	// Metadata (always included)
	if len(p.Metadata) > 0 {
		m[PromptFieldMetadata] = p.Metadata
	}

	// Agent-specific fields
	if opts.IncludeAgentFields {
		if len(p.Skills) > 0 {
			m[PromptFieldSkills] = p.Skills
		}
		if p.Tools != nil && p.Tools.HasTools() {
			m[PromptFieldTools] = p.Tools
		}
		if p.Constraints != nil {
			m[PromptFieldConstraints] = p.Constraints
		}
		if len(p.Messages) > 0 {
			m[PromptFieldMessages] = p.Messages
		}
	}

	// Context
	if opts.IncludeContext && len(p.Context) > 0 {
		m[PromptFieldContext] = p.Context
	}

	return m
}
