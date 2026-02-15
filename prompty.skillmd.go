package prompty

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// SKILL.md format constants
const (
	skillMDFrontmatterDelim = "---"
	skillMDMinSections      = 2 // frontmatter + body
)

// Error messages for SKILL.md parsing
const (
	ErrMsgSkillMDInvalidFormat = "invalid SKILL.md format"
	ErrMsgSkillMDMissingFM     = "SKILL.md missing frontmatter"
	ErrMsgSkillMDInvalidFM     = "SKILL.md frontmatter parse failed"
	ErrMsgSkillMDMissingBody   = "SKILL.md missing body content"
)

// SkillMD represents a parsed SKILL.md document.
type SkillMD struct {
	// Prompt is the parsed prompt configuration from frontmatter
	Prompt *Prompt
	// Body is the prompt template content
	Body string
}

// ExportToSkillMD exports the prompt as a SKILL.md formatted string.
// This strips the execution and extension config sections, keeping only
// Agent Skills standard fields.
func (p *Prompt) ExportToSkillMD(body string) (string, error) {
	if p == nil {
		return body, nil
	}

	// Create a stripped version of the prompt for Agent Skills export
	exportPrompt := struct {
		Name          string                `yaml:"name"`
		Description   string                `yaml:"description"`
		License       string                `yaml:"license,omitempty"`
		Compatibility string                `yaml:"compatibility,omitempty"`
		AllowedTools  string                `yaml:"allowed_tools,omitempty"`
		Metadata      map[string]any        `yaml:"metadata,omitempty"`
		Inputs        map[string]*InputDef  `yaml:"inputs,omitempty"`
		Outputs       map[string]*OutputDef `yaml:"outputs,omitempty"`
		Sample        map[string]any        `yaml:"sample,omitempty"`
	}{
		Name:          p.Name,
		Description:   p.Description,
		License:       p.License,
		Compatibility: p.Compatibility,
		AllowedTools:  p.AllowedTools,
		Metadata:      p.Metadata,
		Inputs:        p.Inputs,
		Outputs:       p.Outputs,
		Sample:        p.Sample,
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(exportPrompt)
	if err != nil {
		return "", err
	}

	// Build SKILL.md format
	var sb strings.Builder
	sb.WriteString(skillMDFrontmatterDelim)
	sb.WriteString("\n")
	sb.Write(yamlBytes)
	sb.WriteString(skillMDFrontmatterDelim)
	sb.WriteString("\n")
	sb.WriteString(body)

	return sb.String(), nil
}

// ImportFromSkillMD parses a SKILL.md formatted string into a Prompt and body.
func ImportFromSkillMD(content string) (*SkillMD, error) {
	if content == "" {
		return nil, NewFrontmatterError(ErrMsgSkillMDInvalidFormat, Position{Line: 1, Column: 1}, nil)
	}

	// Trim any leading whitespace/BOM
	content = strings.TrimLeft(content, "\xef\xbb\xbf \t")

	// Check for frontmatter delimiter at start
	if !strings.HasPrefix(content, skillMDFrontmatterDelim) {
		return nil, NewFrontmatterError(ErrMsgSkillMDMissingFM, Position{Line: 1, Column: 1}, nil)
	}

	// Find the closing frontmatter delimiter
	// Skip the opening delimiter and newline
	afterOpening := content[len(skillMDFrontmatterDelim):]
	if len(afterOpening) > 0 && afterOpening[0] == '\n' {
		afterOpening = afterOpening[1:]
	}

	closeIdx := strings.Index(afterOpening, "\n"+skillMDFrontmatterDelim)
	if closeIdx == -1 {
		return nil, NewFrontmatterError(ErrMsgSkillMDMissingFM, Position{Line: 1, Column: 1}, nil)
	}

	// Extract frontmatter YAML
	fmYAML := afterOpening[:closeIdx]

	// Extract body (after closing delimiter and newline)
	bodyStart := closeIdx + len("\n"+skillMDFrontmatterDelim)
	body := ""
	if bodyStart < len(afterOpening) {
		body = afterOpening[bodyStart:]
		// Trim leading newline from body
		if len(body) > 0 && body[0] == '\n' {
			body = body[1:]
		}
	}

	// Parse frontmatter
	prompt, err := ParseYAMLPrompt(fmYAML)
	if err != nil {
		return nil, err
	}

	return &SkillMD{
		Prompt: prompt,
		Body:   body,
	}, nil
}

// ToSource converts SkillMD back to a full template source string.
func (s *SkillMD) ToSource() (string, error) {
	if s == nil {
		return "", nil
	}

	if s.Prompt == nil {
		return s.Body, nil
	}

	return s.Prompt.ExportToSkillMD(s.Body)
}

// WithPrompt returns a new SkillMD with the given prompt.
func (s *SkillMD) WithPrompt(p *Prompt) *SkillMD {
	if s == nil {
		return &SkillMD{Prompt: p}
	}
	return &SkillMD{
		Prompt: p,
		Body:   s.Body,
	}
}

// WithBody returns a new SkillMD with the given body.
func (s *SkillMD) WithBody(body string) *SkillMD {
	if s == nil {
		return &SkillMD{Body: body}
	}
	return &SkillMD{
		Prompt: s.Prompt,
		Body:   body,
	}
}

// Clone creates a deep copy of SkillMD.
func (s *SkillMD) Clone() *SkillMD {
	if s == nil {
		return nil
	}
	return &SkillMD{
		Prompt: s.Prompt.Clone(),
		Body:   s.Body,
	}
}

// MergeExecution merges execution config into the prompt.
// Returns a new Prompt with execution config added (does not modify original).
func (s *SkillMD) MergeExecution(exec *ExecutionConfig) *Prompt {
	if s == nil || s.Prompt == nil {
		return &Prompt{Execution: exec}
	}

	p := s.Prompt.Clone()
	p.Execution = exec
	return p
}

// IsAgentSkillsCompatible returns true if the prompt contains only
// Agent Skills standard fields (no execution, extensions, or agent-specific config).
func (p *Prompt) IsAgentSkillsCompatible() bool {
	if p == nil {
		return true
	}
	return p.Execution == nil && len(p.Extensions) == 0 && p.Type == "" &&
		len(p.Skills) == 0 && p.Tools == nil && p.Constraints == nil && len(p.Messages) == 0
}

// StripExtensions returns a copy of the prompt with execution, extensions, and agent-specific fields removed.
func (p *Prompt) StripExtensions() *Prompt {
	if p == nil {
		return nil
	}

	return &Prompt{
		Name:          p.Name,
		Description:   p.Description,
		License:       p.License,
		Compatibility: p.Compatibility,
		AllowedTools:  p.AllowedTools,
		Metadata:      p.Metadata,
		Inputs:        p.Inputs,
		Outputs:       p.Outputs,
		Sample:        p.Sample,
		// Execution, Extensions, Type, Skills, Tools, Context, Constraints, Messages, Body
		// are intentionally not copied
	}
}
