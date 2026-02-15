package prompty

// ResponseFormat configures structured output enforcement.
type ResponseFormat struct {
	// Type: "text", "json_object", "json_schema", or "enum"
	Type string `yaml:"type" json:"type"`
	// JSONSchema for structured output validation (when type is "json_schema")
	JSONSchema *JSONSchemaSpec `yaml:"json_schema,omitempty" json:"json_schema,omitempty"`
	// Enum constraint for choice-based outputs (when type is "enum")
	Enum *EnumConstraint `yaml:"enum,omitempty" json:"enum,omitempty"`
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
	// AdditionalProperties controls whether extra properties are allowed (all providers require false for strict mode)
	AdditionalProperties *bool `yaml:"additionalProperties,omitempty" json:"additionalProperties,omitempty"`
	// PropertyOrdering specifies the order of properties in output (Gemini 2.5+ only)
	PropertyOrdering []string `yaml:"propertyOrdering,omitempty" json:"propertyOrdering,omitempty"`
}

// EnumConstraint defines enum/choice constraints for outputs.
type EnumConstraint struct {
	// Values contains the allowed enum values
	Values []string `yaml:"values" json:"values"`
	// Description explains the enum choices
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// OutputFormat is Anthropic's alternative to ResponseFormat for structured outputs.
// Use this when targeting Anthropic Claude API specifically.
type OutputFormat struct {
	// Format specifies the output format specification
	Format *OutputFormatSpec `yaml:"format" json:"format"`
}

// OutputFormatSpec defines the output format specification for Anthropic.
type OutputFormatSpec struct {
	// Type: "json_schema"
	Type string `yaml:"type" json:"type"`
	// Schema is the inline JSON schema (no wrapper unlike OpenAI)
	Schema map[string]any `yaml:"schema" json:"schema"`
}

// GuidedDecoding configures vLLM's structured output constraints.
type GuidedDecoding struct {
	// Backend specifies the guided decoding engine: "xgrammar", "outlines", "lm_format_enforcer", "auto"
	Backend string `yaml:"backend,omitempty" json:"backend,omitempty"`
	// JSON is a JSON schema for structured output
	JSON map[string]any `yaml:"json,omitempty" json:"json,omitempty"`
	// Regex is a regex pattern constraint
	Regex string `yaml:"regex,omitempty" json:"regex,omitempty"`
	// Choice is a list of allowed output choices
	Choice []string `yaml:"choice,omitempty" json:"choice,omitempty"`
	// Grammar is a context-free grammar constraint
	Grammar string `yaml:"grammar,omitempty" json:"grammar,omitempty"`
	// WhitespacePattern controls whitespace handling
	WhitespacePattern string `yaml:"whitespace_pattern,omitempty" json:"whitespace_pattern,omitempty"`
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

// Message represents a conversation message for chat APIs.
type Message struct {
	// Role of the message sender: "system", "user", "assistant", or "tool"
	Role string `yaml:"role" json:"role"`
	// Content of the message
	Content string `yaml:"content" json:"content"`
	// Cache indicates whether this message should be cached
	Cache bool `yaml:"cache,omitempty" json:"cache,omitempty"`
}

// ToOpenAI converts the response format to OpenAI API format.
// Returns nil if the response format is not configured.
func (rf *ResponseFormat) ToOpenAI() map[string]any {
	if rf == nil {
		return nil
	}

	result := map[string]any{
		SchemaKeyType: rf.Type,
	}

	if rf.JSONSchema != nil {
		jsonSchema := map[string]any{
			AttrName: rf.JSONSchema.Name,
		}

		if rf.JSONSchema.Description != "" {
			jsonSchema[SchemaKeyDescription] = rf.JSONSchema.Description
		}

		if rf.JSONSchema.Strict {
			jsonSchema[SchemaKeyStrict] = true
		}

		if rf.JSONSchema.Schema != nil {
			// Ensure additionalProperties: false for strict mode
			schema := copySchema(rf.JSONSchema.Schema)
			if rf.JSONSchema.Strict {
				ensureAdditionalPropertiesFalse(schema)
			}
			jsonSchema[SchemaKeySchema] = schema
		}

		result[SchemaKeyJSONSchema] = jsonSchema
	}

	if rf.Enum != nil && len(rf.Enum.Values) > 0 {
		result[SchemaKeyEnum] = rf.Enum.Values
	}

	return result
}

// ToAnthropic converts to Anthropic output_format structure.
// Returns nil if the response format is not configured.
func (rf *ResponseFormat) ToAnthropic() map[string]any {
	if rf == nil || rf.JSONSchema == nil {
		return nil
	}

	schema := copySchema(rf.JSONSchema.Schema)
	ensureAdditionalPropertiesFalse(schema)

	return map[string]any{
		SchemaKeyFormat: map[string]any{
			SchemaKeyType:   ResponseFormatJSONSchema,
			SchemaKeySchema: schema,
		},
	}
}

// ToGemini converts to Google Gemini/Vertex AI format.
// Returns nil if the response format is not configured.
func (rf *ResponseFormat) ToGemini() map[string]any {
	if rf == nil {
		return nil
	}

	result := map[string]any{
		SchemaKeyType: rf.Type,
	}

	if rf.JSONSchema != nil {
		schema := copySchema(rf.JSONSchema.Schema)
		ensureAdditionalPropertiesFalse(schema)

		// Add propertyOrdering for Gemini 2.5+
		if len(rf.JSONSchema.PropertyOrdering) > 0 {
			schema[SchemaKeyPropertyOrdering] = rf.JSONSchema.PropertyOrdering
		}

		jsonSchema := map[string]any{
			AttrName:        rf.JSONSchema.Name,
			SchemaKeySchema: schema,
		}

		if rf.JSONSchema.Description != "" {
			jsonSchema[SchemaKeyDescription] = rf.JSONSchema.Description
		}

		result[SchemaKeyJSONSchema] = jsonSchema
	}

	return result
}

// ToVLLM converts to vLLM guided decoding format.
// Returns nil if guided decoding is not configured.
func (gd *GuidedDecoding) ToVLLM() map[string]any {
	if gd == nil {
		return nil
	}

	result := make(map[string]any)

	if gd.Backend != "" {
		result[GuidedKeyDecodingBackend] = gd.Backend
	}

	if gd.JSON != nil {
		schema := copySchema(gd.JSON)
		ensureAdditionalPropertiesFalse(schema)
		result[GuidedKeyJSON] = schema
	}

	if gd.Regex != "" {
		result[GuidedKeyRegex] = gd.Regex
	}

	if len(gd.Choice) > 0 {
		result[GuidedKeyChoice] = gd.Choice
	}

	if gd.Grammar != "" {
		result[GuidedKeyGrammar] = gd.Grammar
	}

	if gd.WhitespacePattern != "" {
		result[GuidedKeyWhitespacePattern] = gd.WhitespacePattern
	}

	return result
}

// ToAnthropic converts OutputFormat to Anthropic API format.
func (of *OutputFormat) ToAnthropic() map[string]any {
	if of == nil || of.Format == nil {
		return nil
	}

	schema := copySchema(of.Format.Schema)
	ensureAdditionalPropertiesFalse(schema)

	return map[string]any{
		SchemaKeyFormat: map[string]any{
			SchemaKeyType:   of.Format.Type,
			SchemaKeySchema: schema,
		},
	}
}

// copySchema creates a deep copy of a schema map.
func copySchema(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}

	dst := make(map[string]any, len(src))
	for k, v := range src {
		switch val := v.(type) {
		case map[string]any:
			dst[k] = copySchema(val)
		case []any:
			dst[k] = copySlice(val)
		default:
			dst[k] = v
		}
	}
	return dst
}

// copySlice creates a deep copy of a slice.
func copySlice(src []any) []any {
	if src == nil {
		return nil
	}

	dst := make([]any, len(src))
	for i, v := range src {
		switch val := v.(type) {
		case map[string]any:
			dst[i] = copySchema(val)
		case []any:
			dst[i] = copySlice(val)
		default:
			dst[i] = v
		}
	}
	return dst
}

// ensureAdditionalPropertiesFalse recursively ensures all objects have additionalProperties: false.
func ensureAdditionalPropertiesFalse(schema map[string]any) {
	if schema == nil {
		return
	}

	// Check if this is an object type
	if typeVal, ok := schema[SchemaKeyType]; ok && typeVal == SchemaTypeObject {
		// Set additionalProperties: false if not already set
		if _, exists := schema[SchemaKeyAdditionalProperties]; !exists {
			schema[SchemaKeyAdditionalProperties] = false
		}
	}

	// Recursively process properties
	if props, ok := schema[SchemaKeyProperties].(map[string]any); ok {
		for _, propVal := range props {
			if propSchema, ok := propVal.(map[string]any); ok {
				ensureAdditionalPropertiesFalse(propSchema)
			}
		}
	}

	// Recursively process array items
	if items, ok := schema[SchemaKeyItems].(map[string]any); ok {
		ensureAdditionalPropertiesFalse(items)
	}
}

// isOpenAIModel checks if the model name suggests OpenAI.
func isOpenAIModel(name string) bool {
	prefixes := []string{"gpt-", "o1-", "o3-", "text-", "davinci", "curie", "babbage", "ada"}
	for _, prefix := range prefixes {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// isAnthropicModel checks if the model name suggests Anthropic.
func isAnthropicModel(name string) bool {
	prefixes := []string{"claude-", "claude"}
	for _, prefix := range prefixes {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// isGeminiModel checks if the model name suggests Google Gemini.
func isGeminiModel(name string) bool {
	prefixes := []string{"gemini-", "gemini"}
	for _, prefix := range prefixes {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// isMistralModel checks if the model name suggests Mistral AI.
// Recognized prefixes: mistral-, codestral-, pixtral-, ministral-, open-mistral-, open-mixtral-.
// Used by GetEffectiveProvider() for automatic provider detection.
func isMistralModel(name string) bool {
	prefixes := []string{"mistral-", "codestral-", "pixtral-", "ministral-", "open-mistral-", "open-mixtral-"}
	for _, prefix := range prefixes {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// isCohereModel checks if the model name suggests Cohere.
// Recognized prefixes: command-, embed-, rerank-, c4ai-.
// Used by GetEffectiveProvider() for automatic provider detection.
func isCohereModel(name string) bool {
	prefixes := []string{"command-", "embed-", "rerank-", "c4ai-"}
	for _, prefix := range prefixes {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
