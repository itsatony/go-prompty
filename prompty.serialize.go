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
	// IncludeSkope includes the skope config in output
	IncludeSkope bool
	// IncludeAgentFields includes agent-specific fields (type, skills, tools, constraints, messages)
	IncludeAgentFields bool
	// IncludeContext includes the context map in output
	IncludeContext bool
}

// DefaultSerializeOptions returns the default serialization options (all included).
func DefaultSerializeOptions() *SerializeOptions {
	return &SerializeOptions{
		IncludeExecution:   true,
		IncludeSkope:       true,
		IncludeAgentFields: true,
		IncludeContext:     true,
	}
}

// AgentSkillsExportOptions returns options for Agent Skills compatible export.
// This strips all non-standard fields.
func AgentSkillsExportOptions() *SerializeOptions {
	return &SerializeOptions{
		IncludeExecution:   false,
		IncludeSkope:       false,
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

// buildSerializeMap creates an ordered map for YAML serialization.
func (p *Prompt) buildSerializeMap(opts *SerializeOptions) map[string]any {
	m := make(map[string]any)

	// Standard fields (always included)
	if p.Name != "" {
		m["name"] = p.Name
	}
	if p.Description != "" {
		m["description"] = p.Description
	}
	if p.License != "" {
		m["license"] = p.License
	}
	if p.Compatibility != "" {
		m["compatibility"] = p.Compatibility
	}
	if p.AllowedTools != "" {
		m["allowed_tools"] = p.AllowedTools
	}

	// Type (include if agent fields are included or if type is explicitly set)
	if opts.IncludeAgentFields && p.Type != "" {
		m["type"] = string(p.Type)
	}

	// Execution config
	if opts.IncludeExecution && p.Execution != nil {
		m["execution"] = p.Execution
	}

	// Skope config
	if opts.IncludeSkope && p.Skope != nil {
		m["skope"] = p.Skope
	}

	// Inputs/Outputs (always included)
	if len(p.Inputs) > 0 {
		m["inputs"] = p.Inputs
	}
	if len(p.Outputs) > 0 {
		m["outputs"] = p.Outputs
	}

	// Sample (always included)
	if len(p.Sample) > 0 {
		m["sample"] = p.Sample
	}

	// Metadata (always included)
	if len(p.Metadata) > 0 {
		m["metadata"] = p.Metadata
	}

	// Agent-specific fields
	if opts.IncludeAgentFields {
		if len(p.Skills) > 0 {
			m["skills"] = p.Skills
		}
		if p.Tools != nil && p.Tools.HasTools() {
			m["tools"] = p.Tools
		}
		if p.Constraints != nil {
			m["constraints"] = p.Constraints
		}
		if len(p.Messages) > 0 {
			m["messages"] = p.Messages
		}
	}

	// Context
	if opts.IncludeContext && len(p.Context) > 0 {
		m["context"] = p.Context
	}

	return m
}
