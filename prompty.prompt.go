package prompty

import (
	"encoding/json"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Prompt represents a v2.0/v2.1 prompt configuration parsed from YAML frontmatter.
// It follows the Agent Skills specification (agentskills.io) with namespaced extensions.
type Prompt struct {
	// Agent Skills Standard fields (required)
	Name        string `yaml:"name" json:"name"`               // max 64 chars, slug format
	Description string `yaml:"description" json:"description"` // max 1024 chars

	// Agent Skills Standard fields (optional)
	License       string         `yaml:"license,omitempty" json:"license,omitempty"`
	Compatibility string         `yaml:"compatibility,omitempty" json:"compatibility,omitempty"`
	AllowedTools  string         `yaml:"allowed_tools,omitempty" json:"allowed_tools,omitempty"`
	Metadata      map[string]any `yaml:"metadata,omitempty" json:"metadata,omitempty"`

	// v2.1 Document type: "prompt", "skill" (default), "agent"
	Type DocumentType `yaml:"type,omitempty" json:"type,omitempty"`

	// v2.0 Namespaced configuration
	Execution *ExecutionConfig `yaml:"execution,omitempty" json:"execution,omitempty"`
	Skope     *SkopeConfig     `yaml:"skope,omitempty" json:"skope,omitempty"`

	// Schema definitions (preserved from v1)
	Inputs  map[string]*InputDef  `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Outputs map[string]*OutputDef `yaml:"outputs,omitempty" json:"outputs,omitempty"`

	// Sample data for testing
	Sample map[string]any `yaml:"sample,omitempty" json:"sample,omitempty"`

	// v2.1 Agent-specific fields
	Skills      []SkillRef         `yaml:"skills,omitempty" json:"skills,omitempty"`
	Tools       *ToolsConfig       `yaml:"tools,omitempty" json:"tools,omitempty"`
	Context     map[string]any     `yaml:"context,omitempty" json:"context,omitempty"`
	Constraints *ConstraintsConfig `yaml:"constraints,omitempty" json:"constraints,omitempty"`
	Messages    []MessageTemplate  `yaml:"messages,omitempty" json:"messages,omitempty"`

	// Body is the template content (not serialized to YAML, extracted separately)
	Body string `yaml:"-" json:"-"`
}

// slugRegex is the compiled regex for slug validation
var slugRegex = regexp.MustCompile(PromptSlugPattern)

// Validate checks the prompt configuration for required fields and constraints.
// Returns an error if validation fails, nil if valid.
func (p *Prompt) Validate() error {
	if p == nil {
		return NewPromptNameRequiredError()
	}

	// Validate name (required, max length, slug format)
	if p.Name == "" {
		return NewPromptNameRequiredError()
	}
	if len(p.Name) > PromptNameMaxLength {
		return NewPromptNameTooLongError(p.Name, PromptNameMaxLength)
	}
	if !slugRegex.MatchString(p.Name) {
		return NewPromptNameInvalidFormatError(p.Name)
	}

	// Validate description (required, max length)
	if p.Description == "" {
		return NewPromptDescriptionRequiredError()
	}
	if len(p.Description) > PromptDescriptionMaxLength {
		return NewPromptDescriptionTooLongError(PromptDescriptionMaxLength)
	}

	// Validate document type if set
	if p.Type != "" && !isValidDocumentType(p.Type) {
		return NewInvalidDocumentTypeError(string(p.Type))
	}

	// Type-specific validation
	effectiveType := p.EffectiveType()
	switch effectiveType {
	case DocumentTypePrompt:
		if len(p.Skills) > 0 {
			return NewAgentValidationError(ErrMsgPromptNoSkillsAllowed, p.Name)
		}
		if p.Tools != nil && p.Tools.HasTools() {
			return NewAgentValidationError(ErrMsgPromptNoToolsAllowed, p.Name)
		}
		if p.Constraints != nil {
			return NewAgentValidationError(ErrMsgPromptNoConstraints, p.Name)
		}

	case DocumentTypeSkill:
		if len(p.Skills) > 0 {
			return NewAgentValidationError(ErrMsgSkillNoSkillsAllowed, p.Name)
		}

	case DocumentTypeAgent:
		// Validate skill refs
		for i := range p.Skills {
			if err := p.Skills[i].Validate(); err != nil {
				return err
			}
		}
		// Validate messages if defined
		for i := range p.Messages {
			if err := p.Messages[i].Validate(); err != nil {
				return err
			}
		}
	}

	// Validate tools config if present
	if p.Tools != nil {
		if err := p.Tools.Validate(); err != nil {
			return err
		}
	}

	// Validate execution config if present
	if p.Execution != nil {
		if err := p.Execution.Validate(); err != nil {
			return err
		}
	}

	// Validate skope config if present
	if p.Skope != nil {
		if err := p.Skope.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// ValidateOptional performs validation only if the prompt has enough
// fields to indicate it is a well-formed v2.1 document. Prompts with
// Execution, Skope, Type, or a Name are validated; bare frontmatter
// with none of these fields is silently accepted.
func (p *Prompt) ValidateOptional() error {
	if p == nil {
		return nil
	}
	// If the prompt has v2.1-specific fields, require full validation
	if p.Execution != nil || p.Skope != nil || p.Type != "" || p.Name != "" {
		return p.Validate()
	}
	return nil
}

// ParseYAMLPrompt parses YAML data into a Prompt.
// Returns an error if the YAML data exceeds DefaultMaxFrontmatterSize (DoS protection).
func ParseYAMLPrompt(yamlData string) (*Prompt, error) {
	if yamlData == "" {
		return nil, nil
	}

	// Check size limit to prevent DoS via large YAML
	if len(yamlData) > DefaultMaxFrontmatterSize {
		return nil, NewFrontmatterError(ErrMsgFrontmatterTooLarge, Position{Line: 1, Column: 1}, nil)
	}

	var prompt Prompt
	if err := yaml.Unmarshal([]byte(yamlData), &prompt); err != nil {
		return nil, NewFrontmatterParseError(err)
	}
	return &prompt, nil
}

// GetName returns the prompt name or empty string if not set.
func (p *Prompt) GetName() string {
	if p == nil {
		return ""
	}
	return p.Name
}

// GetDescription returns the prompt description or empty string if not set.
func (p *Prompt) GetDescription() string {
	if p == nil {
		return ""
	}
	return p.Description
}

// GetLicense returns the license or empty string if not set.
func (p *Prompt) GetLicense() string {
	if p == nil {
		return ""
	}
	return p.License
}

// GetCompatibility returns the compatibility or empty string if not set.
func (p *Prompt) GetCompatibility() string {
	if p == nil {
		return ""
	}
	return p.Compatibility
}

// GetAllowedTools returns the allowed_tools or empty string if not set.
func (p *Prompt) GetAllowedTools() string {
	if p == nil {
		return ""
	}
	return p.AllowedTools
}

// GetMetadata returns the metadata map or nil if not set.
func (p *Prompt) GetMetadata() map[string]any {
	if p == nil {
		return nil
	}
	return p.Metadata
}

// GetExecution returns the execution config or nil if not set.
func (p *Prompt) GetExecution() *ExecutionConfig {
	if p == nil {
		return nil
	}
	return p.Execution
}

// GetSkope returns the skope config or nil if not set.
func (p *Prompt) GetSkope() *SkopeConfig {
	if p == nil {
		return nil
	}
	return p.Skope
}

// GetSampleData returns the sample data map or nil if not set.
func (p *Prompt) GetSampleData() map[string]any {
	if p == nil {
		return nil
	}
	return p.Sample
}

// HasExecution returns true if execution config is present.
func (p *Prompt) HasExecution() bool {
	return p != nil && p.Execution != nil
}

// HasSkope returns true if skope config is present.
func (p *Prompt) HasSkope() bool {
	return p != nil && p.Skope != nil
}

// HasInputs returns true if input definitions are present.
func (p *Prompt) HasInputs() bool {
	return p != nil && len(p.Inputs) > 0
}

// HasOutputs returns true if output definitions are present.
func (p *Prompt) HasOutputs() bool {
	return p != nil && len(p.Outputs) > 0
}

// HasSample returns true if sample data is present.
func (p *Prompt) HasSample() bool {
	return p != nil && len(p.Sample) > 0
}

// HasMetadata returns true if metadata is present.
func (p *Prompt) HasMetadata() bool {
	return p != nil && len(p.Metadata) > 0
}

// ValidateInputs validates the provided data against the input definitions.
// Returns an error if any required input is missing or has wrong type.
func (p *Prompt) ValidateInputs(data map[string]any) error {
	if p == nil || p.Inputs == nil {
		return nil
	}

	for name, def := range p.Inputs {
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
		if err := validatePromptInputType(name, val, def.Type); err != nil {
			return err
		}
	}

	return nil
}

// validatePromptInputType checks if the value matches the expected type.
func validatePromptInputType(name string, val any, expectedType string) error {
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
		reason := expectedType + " expected"
		return NewInputValidationError(name, reason)
	}

	return nil
}

// JSON returns the JSON representation of the prompt.
func (p *Prompt) JSON() (string, error) {
	if p == nil {
		return "", nil
	}
	data, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// JSONPretty returns the pretty-printed JSON representation of the prompt.
func (p *Prompt) JSONPretty() (string, error) {
	if p == nil {
		return "", nil
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// YAML returns the YAML representation of the prompt.
func (p *Prompt) YAML() (string, error) {
	if p == nil {
		return "", nil
	}
	data, err := yaml.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Clone creates a deep copy of the prompt.
func (p *Prompt) Clone() *Prompt {
	if p == nil {
		return nil
	}

	clone := &Prompt{
		Name:          p.Name,
		Description:   p.Description,
		License:       p.License,
		Compatibility: p.Compatibility,
		AllowedTools:  p.AllowedTools,
		Type:          p.Type,
		Body:          p.Body,
	}

	// Clone metadata
	if p.Metadata != nil {
		clone.Metadata = make(map[string]any, len(p.Metadata))
		for k, v := range p.Metadata {
			clone.Metadata[k] = v
		}
	}

	// Clone execution
	if p.Execution != nil {
		clone.Execution = p.Execution.Clone()
	}

	// Clone skope
	if p.Skope != nil {
		clone.Skope = p.Skope.Clone()
	}

	// Clone inputs
	if p.Inputs != nil {
		clone.Inputs = make(map[string]*InputDef, len(p.Inputs))
		for k, v := range p.Inputs {
			inputClone := *v
			clone.Inputs[k] = &inputClone
		}
	}

	// Clone outputs
	if p.Outputs != nil {
		clone.Outputs = make(map[string]*OutputDef, len(p.Outputs))
		for k, v := range p.Outputs {
			outputClone := *v
			clone.Outputs[k] = &outputClone
		}
	}

	// Clone sample
	if p.Sample != nil {
		clone.Sample = make(map[string]any, len(p.Sample))
		for k, v := range p.Sample {
			clone.Sample[k] = v
		}
	}

	// Clone v2.1 agent fields
	if p.Skills != nil {
		clone.Skills = make([]SkillRef, len(p.Skills))
		for i := range p.Skills {
			if cloned := p.Skills[i].Clone(); cloned != nil {
				clone.Skills[i] = *cloned
			}
		}
	}

	if p.Tools != nil {
		clone.Tools = p.Tools.Clone()
	}

	if p.Context != nil {
		clone.Context = make(map[string]any, len(p.Context))
		for k, v := range p.Context {
			clone.Context[k] = v
		}
	}

	if p.Constraints != nil {
		clone.Constraints = p.Constraints.Clone()
	}

	if p.Messages != nil {
		clone.Messages = make([]MessageTemplate, len(p.Messages))
		copy(clone.Messages, p.Messages)
	}

	return clone
}

// GetSlug returns the slug from skope config, or derives from name.
func (p *Prompt) GetSlug() string {
	if p == nil {
		return ""
	}
	if p.Skope != nil && p.Skope.Slug != "" {
		return p.Skope.Slug
	}
	return p.Name
}

// EffectiveType returns the document type, defaulting to "skill" if not set.
func (p *Prompt) EffectiveType() DocumentType {
	if p == nil || p.Type == "" {
		return DocumentTypeSkill
	}
	return p.Type
}

// IsAgent returns true if this is an agent document.
func (p *Prompt) IsAgent() bool {
	return p != nil && p.EffectiveType() == DocumentTypeAgent
}

// IsSkill returns true if this is a skill document (default type).
func (p *Prompt) IsSkill() bool {
	return p != nil && p.EffectiveType() == DocumentTypeSkill
}

// IsPrompt returns true if this is a simple prompt document.
func (p *Prompt) IsPrompt() bool {
	return p != nil && p.EffectiveType() == DocumentTypePrompt
}

// ValidateAsAgent performs stricter validation for agent documents.
// In addition to standard Validate() checks, it verifies that:
//   - The document type is "agent"
//   - Execution config with provider and model is present
//   - Body or messages are defined (at least one is required)
//
// Use this before CompileAgent() to catch configuration issues early.
func (p *Prompt) ValidateAsAgent() error {
	if err := p.Validate(); err != nil {
		return err
	}

	if !p.IsAgent() {
		return NewAgentValidationError(ErrMsgNotAnAgent, p.Name)
	}

	if p.Execution == nil {
		return NewCompilationError(ErrMsgNoExecutionConfig, nil)
	}
	if p.Execution.Provider == "" {
		return NewCompilationError(ErrMsgNoProvider, nil)
	}
	if p.Execution.Model == "" {
		return NewCompilationError(ErrMsgNoModel, nil)
	}

	if p.Body == "" && len(p.Messages) == 0 {
		return NewAgentValidationError(ErrMsgAgentNoBodyOrMessages, p.Name)
	}

	return nil
}
