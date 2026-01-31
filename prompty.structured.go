package prompty

import (
	"slices"
	"strings"
)

// SchemaValidationResult contains validation results for a JSON schema.
type SchemaValidationResult struct {
	// Valid indicates whether the schema passed validation
	Valid bool `json:"valid"`
	// Errors contains critical validation errors
	Errors []string `json:"errors,omitempty"`
	// Warnings contains non-critical issues
	Warnings []string `json:"warnings,omitempty"`
}

// ValidateJSONSchema validates a JSON schema structure.
// Returns a validation result with errors and warnings.
func ValidateJSONSchema(schema map[string]any) *SchemaValidationResult {
	result := &SchemaValidationResult{Valid: true}
	validateSchemaRecursive(schema, "", result)
	return result
}

// validateSchemaRecursive validates a schema recursively.
func validateSchemaRecursive(schema map[string]any, path string, result *SchemaValidationResult) {
	if schema == nil {
		return
	}

	// Check for type field
	typeVal, hasType := schema[SchemaKeyType]
	if !hasType {
		addError(result, path, ErrMsgSchemaMissingType)
		return
	}

	typeStr, ok := typeVal.(string)
	if !ok {
		addError(result, path, ErrMsgSchemaInvalidType)
		return
	}

	switch typeStr {
	case SchemaTypeObject:
		validateObjectSchema(schema, path, result)
	case SchemaTypeArray:
		validateArraySchema(schema, path, result)
	case SchemaTypeString, SchemaTypeNumber, SchemaTypeBoolean:
		// Primitive types are valid
	default:
		addWarning(result, path, "unknown type: "+typeStr)
	}
}

// validateObjectSchema validates an object schema.
func validateObjectSchema(schema map[string]any, path string, result *SchemaValidationResult) {
	// Check for properties
	propsVal, hasProps := schema[SchemaKeyProperties]
	if !hasProps {
		addWarning(result, path, ErrMsgSchemaMissingProperties)
		return
	}

	props, ok := propsVal.(map[string]any)
	if !ok {
		addError(result, path, ErrMsgSchemaInvalidProperties)
		return
	}

	// Check additionalProperties
	if _, hasAdditional := schema[SchemaKeyAdditionalProperties]; !hasAdditional {
		addWarning(result, path, ErrMsgSchemaAdditionalProperties)
	}

	// Check required array
	if reqVal, hasReq := schema[SchemaKeyRequired]; hasReq {
		if _, ok := reqVal.([]any); !ok {
			if _, ok := reqVal.([]string); !ok {
				addError(result, path, ErrMsgSchemaInvalidRequired)
			}
		}
	}

	// Validate nested properties
	for propName, propVal := range props {
		propPath := joinPath(path, propName)
		if propSchema, ok := propVal.(map[string]any); ok {
			validateSchemaRecursive(propSchema, propPath, result)
		}
	}
}

// validateArraySchema validates an array schema.
func validateArraySchema(schema map[string]any, path string, result *SchemaValidationResult) {
	if itemsVal, hasItems := schema[SchemaKeyItems]; hasItems {
		if itemsSchema, ok := itemsVal.(map[string]any); ok {
			validateSchemaRecursive(itemsSchema, joinPath(path, SchemaKeyItems), result)
		}
	}
}

// ValidateForProvider validates schema compatibility with a specific provider.
func ValidateForProvider(schema map[string]any, provider string) *SchemaValidationResult {
	result := ValidateJSONSchema(schema)

	switch provider {
	case ProviderOpenAI, ProviderAnthropic, ProviderGoogle, ProviderGemini, ProviderVertex, ProviderAzure:
		validateStrictModeRequirements(schema, "", result)
	case ProviderVLLM:
		// vLLM is more permissive but still benefits from additionalProperties: false
		validateAdditionalPropertiesRecommended(schema, "", result)
	default:
		addWarning(result, "", ErrMsgSchemaUnsupportedProvider+": "+provider)
	}

	// Gemini-specific validation
	if provider == ProviderGemini || provider == ProviderGoogle {
		validateGeminiSpecific(schema, "", result)
	}

	return result
}

// validateStrictModeRequirements validates requirements for strict mode providers.
func validateStrictModeRequirements(schema map[string]any, path string, result *SchemaValidationResult) {
	if schema == nil {
		return
	}

	if schema[SchemaKeyType] == SchemaTypeObject {
		// Check additionalProperties is explicitly false
		if addProps, hasAddProps := schema[SchemaKeyAdditionalProperties]; hasAddProps {
			if addProps != false {
				addError(result, path, ErrMsgSchemaAdditionalProperties)
			}
		} else {
			addError(result, path, ErrMsgSchemaAdditionalProperties)
		}

		// Validate nested properties
		if props, ok := schema[SchemaKeyProperties].(map[string]any); ok {
			for propName, propVal := range props {
				if propSchema, ok := propVal.(map[string]any); ok {
					validateStrictModeRequirements(propSchema, joinPath(path, propName), result)
				}
			}
		}
	}

	// Validate array items
	if items, ok := schema[SchemaKeyItems].(map[string]any); ok {
		validateStrictModeRequirements(items, joinPath(path, SchemaKeyItems), result)
	}
}

// validateAdditionalPropertiesRecommended adds warnings (not errors) for missing additionalProperties.
func validateAdditionalPropertiesRecommended(schema map[string]any, path string, result *SchemaValidationResult) {
	if schema == nil {
		return
	}

	if schema[SchemaKeyType] == SchemaTypeObject {
		if _, hasAddProps := schema[SchemaKeyAdditionalProperties]; !hasAddProps {
			addWarning(result, path, "additionalProperties: false recommended")
		}

		if props, ok := schema[SchemaKeyProperties].(map[string]any); ok {
			for propName, propVal := range props {
				if propSchema, ok := propVal.(map[string]any); ok {
					validateAdditionalPropertiesRecommended(propSchema, joinPath(path, propName), result)
				}
			}
		}
	}

	if items, ok := schema[SchemaKeyItems].(map[string]any); ok {
		validateAdditionalPropertiesRecommended(items, joinPath(path, SchemaKeyItems), result)
	}
}

// validateGeminiSpecific validates Gemini-specific features.
func validateGeminiSpecific(schema map[string]any, path string, result *SchemaValidationResult) {
	// propertyOrdering is valid for Gemini 2.5+
	if ordering, hasOrdering := schema[SchemaKeyPropertyOrdering]; hasOrdering {
		if _, ok := ordering.([]any); !ok {
			if _, ok := ordering.([]string); !ok {
				addWarning(result, path, "propertyOrdering should be an array of strings")
			}
		}
	}
}

// EnsureAdditionalPropertiesFalse recursively ensures all objects have additionalProperties: false.
// Returns a new schema map (does not modify the original).
func EnsureAdditionalPropertiesFalse(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}

	result := copySchema(schema)
	ensureAdditionalPropertiesFalse(result)
	return result
}

// ExtractRequiredFields extracts all property names from an object schema.
// This is useful for generating a required array that includes all properties.
func ExtractRequiredFields(schema map[string]any) []string {
	if schema == nil {
		return nil
	}

	props, ok := schema[SchemaKeyProperties].(map[string]any)
	if !ok {
		return nil
	}

	fields := make([]string, 0, len(props))
	for name := range props {
		fields = append(fields, name)
	}
	return fields
}

// ValidateEnumConstraint validates an enum constraint.
func ValidateEnumConstraint(enum *EnumConstraint) *SchemaValidationResult {
	result := &SchemaValidationResult{Valid: true}

	if enum == nil {
		addError(result, "", "enum constraint is nil")
		return result
	}

	if len(enum.Values) == 0 {
		addError(result, "", ErrMsgEnumEmptyValues)
	}

	return result
}

// ValidateGuidedDecoding validates a guided decoding configuration.
func ValidateGuidedDecoding(gd *GuidedDecoding) *SchemaValidationResult {
	result := &SchemaValidationResult{Valid: true}

	if gd == nil {
		return result
	}

	// Count how many constraints are set
	constraintCount := 0
	if gd.JSON != nil {
		constraintCount++
	}
	if gd.Regex != "" {
		constraintCount++
	}
	if len(gd.Choice) > 0 {
		constraintCount++
	}
	if gd.Grammar != "" {
		constraintCount++
	}

	if constraintCount > 1 {
		addError(result, "", ErrMsgGuidedDecodingConflict)
	}

	// Validate backend if specified
	if gd.Backend != "" {
		validBackends := []string{GuidedBackendXGrammar, GuidedBackendOutlines, GuidedBackendLMFormatEnforcer, GuidedBackendAuto}
		if !slices.Contains(validBackends, gd.Backend) {
			addWarning(result, "", "unknown guided decoding backend: "+gd.Backend)
		}
	}

	// Validate JSON schema if present
	if gd.JSON != nil {
		jsonResult := ValidateJSONSchema(gd.JSON)
		result.Errors = append(result.Errors, jsonResult.Errors...)
		result.Warnings = append(result.Warnings, jsonResult.Warnings...)
		if !jsonResult.Valid {
			result.Valid = false
		}
	}

	return result
}

// addError adds an error to the validation result.
func addError(result *SchemaValidationResult, path, msg string) {
	result.Valid = false
	if path != "" {
		msg = path + ": " + msg
	}
	result.Errors = append(result.Errors, msg)
}

// addWarning adds a warning to the validation result.
func addWarning(result *SchemaValidationResult, path, msg string) {
	if path != "" {
		msg = path + ": " + msg
	}
	result.Warnings = append(result.Warnings, msg)
}

// joinPath joins path segments.
func joinPath(base, segment string) string {
	if base == "" {
		return segment
	}
	return base + "." + segment
}

// DetectSchemaProvider attempts to detect the provider from schema configuration.
func DetectSchemaProvider(config *InferenceConfig) string {
	if config == nil {
		return ""
	}
	return config.GetEffectiveProvider()
}

// IsStrictModeRequired returns true if the provider requires strict mode.
func IsStrictModeRequired(provider string) bool {
	switch strings.ToLower(provider) {
	case ProviderOpenAI, ProviderAnthropic, ProviderGoogle, ProviderGemini, ProviderVertex, ProviderAzure:
		return true
	default:
		return false
	}
}
