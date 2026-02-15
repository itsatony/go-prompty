package prompty

import (
	"fmt"
	"strconv"

	"github.com/itsatony/go-cuserr"
)

// Error message constants - ALL error messages must be constants (NO MAGIC STRINGS)
const (
	// Parse errors
	ErrMsgParseFailed     = "template parsing failed"
	ErrMsgInvalidSyntax   = "invalid template syntax"
	ErrMsgUnexpectedChar  = "unexpected character"
	ErrMsgUnterminatedTag = "unterminated tag"
	ErrMsgUnterminatedStr = "unterminated string literal"
	ErrMsgInvalidEscape   = "invalid escape sequence"
	ErrMsgUnexpectedEOF   = "unexpected end of input"
	ErrMsgMismatchedTag   = "mismatched closing tag"
	ErrMsgInvalidTagName  = "invalid tag name"
	ErrMsgEmptyTagName    = "tag name cannot be empty"
	ErrMsgNestedRawBlock  = "nested raw blocks are not allowed"

	// Execution errors
	ErrMsgUnknownTag       = "unknown tag"
	ErrMsgUnknownResolver  = "no resolver registered for tag"
	ErrMsgResolverFailed   = "resolver execution failed"
	ErrMsgVariableNotFound = "variable not found"
	ErrMsgInvalidPath      = "invalid variable path"
	ErrMsgEmptyPath        = "variable path cannot be empty"
	ErrMsgExecutionFailed  = "template execution failed"

	// Validation errors
	ErrMsgMissingAttribute = "required attribute missing"
	ErrMsgInvalidAttribute = "invalid attribute value"

	// Registry errors
	ErrMsgResolverExists = "resolver already registered"

	// Type conversion errors
	ErrMsgTypeConversion = "type conversion failed"

	// Template errors (nested template inclusion)
	ErrMsgTemplateNotFound      = "template not found"
	ErrMsgTemplateAlreadyExists = "template already registered"
	ErrMsgTemplateDepthExceeded = "template inclusion depth exceeded"
	ErrMsgInvalidTemplateName   = "invalid template name"
	ErrMsgEmptyTemplateName     = "template name cannot be empty"
	ErrMsgMissingTemplateAttr   = "missing required 'template' attribute"
	ErrMsgEngineNotAvailable    = "engine not available for nested template resolution"
	ErrMsgReservedTemplateName  = "template name uses reserved prompty.* namespace"

	// Error strategy messages (Phase 3)
	ErrMsgInvalidErrorStrategy = "invalid error strategy"
	ErrMsgErrorHandledByStrat  = "error handled by strategy"

	// Validation messages (Phase 3)
	ErrMsgValidationFailed     = "template validation failed"
	ErrMsgUnknownTagInTemplate = "unknown tag in template"
	ErrMsgInvalidOnErrorAttr   = "invalid onerror attribute value"
	ErrMsgMissingIncludeTarget = "included template not found"

	// For loop messages (Phase 4)
	ErrMsgForMissingItem    = "missing required 'item' attribute"
	ErrMsgForMissingIn      = "missing required 'in' attribute"
	ErrMsgForInvalidLimit   = "invalid 'limit' attribute value"
	ErrMsgForCollectionPath = "collection path not found"
	ErrMsgForNotIterable    = "value is not iterable"
	ErrMsgForLimitExceeded  = "loop iteration limit exceeded"
	ErrMsgForNotClosed      = "for block not closed"

	// Switch/case messages (Phase 5)
	ErrMsgSwitchMissingEval      = "missing required 'eval' attribute for switch"
	ErrMsgSwitchMissingValue     = "case requires 'value' or 'eval' attribute"
	ErrMsgSwitchNotClosed        = "switch block not closed"
	ErrMsgSwitchCaseNotClosed    = "case block not closed"
	ErrMsgSwitchDefaultNotLast   = "default case must be last in switch"
	ErrMsgSwitchDuplicateDefault = "only one default case allowed in switch"
	ErrMsgSwitchInvalidCaseTag   = "unexpected tag inside switch block"

	// Custom function messages (Phase 5)
	ErrMsgFuncNilFunc       = "function cannot be nil"
	ErrMsgFuncEmptyName     = "function name cannot be empty"
	ErrMsgFuncAlreadyExists = "function already registered"

	// Context messages
	ErrMsgInvalidContextType = "invalid context type"

	// Environment variable messages
	ErrMsgEnvVarNotFound = "environment variable not found"
	ErrMsgEnvVarRequired = "required environment variable not set"

	// Config block messages (legacy JSON - kept for backward compatibility)
	ErrMsgConfigBlockExtract    = "failed to extract config block"
	ErrMsgConfigBlockParse      = "failed to parse config block JSON"
	ErrMsgConfigBlockInvalid    = "invalid config block format"
	ErrMsgConfigBlockUnclosed   = "config block not properly closed"
	ErrMsgInputValidationFailed = "input validation failed"
	ErrMsgRequiredInputMissing  = "required input missing"
	ErrMsgInputTypeMismatch     = "input type mismatch"

	// YAML frontmatter messages
	ErrMsgFrontmatterExtract       = "failed to extract YAML frontmatter"
	ErrMsgFrontmatterParse         = "failed to parse YAML frontmatter"
	ErrMsgFrontmatterInvalid       = "invalid YAML frontmatter format"
	ErrMsgFrontmatterUnclosed      = "YAML frontmatter not properly closed"
	ErrMsgLegacyJSONConfigDetected = "legacy JSON config block detected - please migrate to YAML frontmatter with --- delimiters"

	// Message tag messages
	ErrMsgMessageMissingRole      = "missing required 'role' attribute"
	ErrMsgMessageInvalidRole      = "invalid role - must be system, user, assistant, or tool"
	ErrMsgMessageNestedNotAllowed = "nested message tags are not allowed"

	// YAML frontmatter size limits
	ErrMsgFrontmatterTooLarge = "YAML frontmatter exceeds maximum size limit"

	// Deployment status messages
	ErrMsgInvalidDeploymentStatus = "invalid deployment status"
	ErrMsgStatusTransitionDenied  = "status transition not allowed"
	ErrMsgArchivedVersionReadOnly = "archived versions are read-only"

	// Label messages
	ErrMsgInvalidLabelName   = "invalid label name"
	ErrMsgLabelNotFound      = "label not found"
	ErrMsgLabelNameTooLong   = "label name exceeds maximum length"
	ErrMsgLabelNameEmpty     = "label name cannot be empty"
	ErrMsgLabelVersionError  = "label version mismatch"
	ErrMsgInvalidLabelFormat = "must start with lowercase letter and contain only lowercase letters, digits, underscores, or hyphens"

	// Schema validation messages
	ErrMsgSchemaValidationFailed     = "schema validation failed"
	ErrMsgSchemaInvalidType          = "schema has invalid type"
	ErrMsgSchemaMissingType          = "schema missing required 'type' field"
	ErrMsgSchemaMissingProperties    = "object schema missing 'properties' field"
	ErrMsgSchemaInvalidProperties    = "schema 'properties' field must be an object"
	ErrMsgSchemaInvalidRequired      = "schema 'required' field must be an array"
	ErrMsgSchemaInvalidEnum          = "enum values must be a non-empty array"
	ErrMsgSchemaUnsupportedProvider  = "unsupported provider for schema validation"
	ErrMsgSchemaAdditionalProperties = "strict mode requires additionalProperties: false"
	ErrMsgSchemaPropertyOrdering     = "propertyOrdering requires Gemini 2.5+ provider"
	ErrMsgEnumEmptyValues            = "enum constraint requires at least one value"
	ErrMsgGuidedDecodingConflict     = "only one guided decoding constraint allowed"

	// Core execution parameter validation messages
	ErrMsgTemperatureOutOfRange = "temperature must be between 0.0 and 2.0"
	ErrMsgTopPOutOfRange        = "top_p must be between 0.0 and 1.0"
	ErrMsgMaxTokensInvalid      = "max_tokens must be positive"
	ErrMsgTopKInvalid           = "top_k must be non-negative"
	ErrMsgThinkingBudgetInvalid = "thinking.budget_tokens must be positive"

	// Inference parameter validation messages (v2.3)
	ErrMsgMinPOutOfRange              = "min_p must be between 0.0 and 1.0"
	ErrMsgRepetitionPenaltyOutOfRange = "repetition_penalty must be greater than 0.0"
	ErrMsgLogprobsOutOfRange          = "logprobs must be between 0 and 20"
	ErrMsgStopTokenIDNegative         = "stop_token_ids values must be non-negative"
	ErrMsgLogitBiasValueOutOfRange    = "logit_bias values must be between -100.0 and 100.0"

	// v2.5 Media generation validation messages
	ErrMsgInvalidModality               = "invalid modality value"
	ErrMsgImageWidthOutOfRange          = "image width must be between 1 and 8192"
	ErrMsgImageHeightOutOfRange         = "image height must be between 1 and 8192"
	ErrMsgImageNumImagesOutOfRange      = "num_images must be between 1 and 10"
	ErrMsgImageGuidanceScaleOutOfRange  = "guidance_scale must be between 0.0 and 30.0"
	ErrMsgImageStepsOutOfRange          = "steps must be between 1 and 200"
	ErrMsgImageStrengthOutOfRange       = "strength must be between 0.0 and 1.0"
	ErrMsgImageInvalidQuality           = "invalid image quality value"
	ErrMsgImageInvalidStyle             = "invalid image style value"
	ErrMsgAudioSpeedOutOfRange          = "audio speed must be between 0.25 and 4.0"
	ErrMsgAudioInvalidFormat            = "invalid audio output format"
	ErrMsgAudioDurationOutOfRange       = "audio duration must be between 0.0 and 600.0"
	ErrMsgEmbeddingDimensionsOutOfRange = "embedding dimensions must be between 1 and 65536"
	ErrMsgEmbeddingInvalidFormat        = "invalid embedding format"
	ErrMsgStreamInvalidMethod           = "invalid streaming method"
	ErrMsgAsyncPollIntervalInvalid      = "async poll interval must be positive"
	ErrMsgAsyncPollTimeoutInvalid       = "async poll timeout must be positive"
	ErrMsgAsyncPollTimeoutTooSmall      = "async poll timeout must be greater than or equal to poll interval"

	// Skope validation messages
	ErrMsgInvalidSkopeSlug      = "invalid skope slug format"
	ErrMsgInvalidVisibility     = "invalid visibility value"
	ErrMsgVersionNumberNegative = "version_number cannot be negative"
	ErrMsgInvalidRegion         = "region values must be non-empty strings"

	// v2.0 Prompt validation messages
	ErrMsgPromptNameRequired        = "prompt name is required"
	ErrMsgPromptNameTooLong         = "prompt name exceeds maximum length"
	ErrMsgPromptNameInvalidFormat   = "prompt name must be slug format (lowercase letters, digits, hyphens)"
	ErrMsgPromptDescriptionRequired = "prompt description is required"
	ErrMsgPromptDescriptionTooLong  = "prompt description exceeds maximum length"

	// v2.0 Reference resolution messages
	ErrMsgRefNotFound      = "referenced prompt not found"
	ErrMsgRefCircular      = "circular reference detected"
	ErrMsgRefDepthExceeded = "reference resolution depth exceeded"
	ErrMsgRefMissingSlug   = "prompty.ref requires slug attribute"
	ErrMsgRefInvalidSlug   = "invalid prompt slug format"
)

// Error code constants for categorization
const (
	ErrCodeParse      = "PROMPTY_PARSE"
	ErrCodeExec       = "PROMPTY_EXEC"
	ErrCodeValidation = "PROMPTY_VALIDATION"
	ErrCodeRegistry   = "PROMPTY_REGISTRY"
	ErrCodeTemplate   = "PROMPTY_TEMPLATE"
	ErrCodeFunc       = "PROMPTY_FUNC"
	ErrCodeConfig     = "PROMPTY_CONFIG"
	ErrCodeEnv        = "PROMPTY_ENV"
	ErrCodeLabel      = "PROMPTY_LABEL"
	ErrCodeStatus     = "PROMPTY_STATUS"
	ErrCodeSchema     = "PROMPTY_SCHEMA"
	ErrCodePrompt     = "PROMPTY_PROMPT"     // v2.0: Prompt validation errors
	ErrCodeRef        = "PROMPTY_REF"        // v2.0: Reference resolution errors
	ErrCodeVersioning = "PROMPTY_VERSIONING" // Versioning operation errors
)

// Position represents a location in the source template
type Position struct {
	Offset int // Byte offset from start
	Line   int // 1-indexed line number
	Column int // 1-indexed column number
}

// String returns a human-readable position string
func (p Position) String() string {
	return fmt.Sprintf("line %d, column %d", p.Line, p.Column)
}

// NewParseError creates a parse error with position context
func NewParseError(msg string, pos Position, cause error) error {
	var err *cuserr.CustomError
	if cause != nil {
		err = cuserr.WrapStdError(cause, ErrCodeParse, msg)
	} else {
		err = cuserr.NewValidationError(ErrCodeParse, msg)
	}
	return err.
		WithMetadata(MetaKeyLine, strconv.Itoa(pos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(pos.Column)).
		WithMetadata(MetaKeyOffset, strconv.Itoa(pos.Offset))
}

// NewUnterminatedTagError creates an error for unterminated tags
func NewUnterminatedTagError(pos Position) error {
	return cuserr.NewValidationError(ErrCodeParse, ErrMsgUnterminatedTag).
		WithMetadata(MetaKeyLine, strconv.Itoa(pos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(pos.Column)).
		WithMetadata(MetaKeyOffset, strconv.Itoa(pos.Offset))
}

// NewUnterminatedStrError creates an error for unterminated string literals
func NewUnterminatedStrError(pos Position) error {
	return cuserr.NewValidationError(ErrCodeParse, ErrMsgUnterminatedStr).
		WithMetadata(MetaKeyLine, strconv.Itoa(pos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(pos.Column)).
		WithMetadata(MetaKeyOffset, strconv.Itoa(pos.Offset))
}

// NewMismatchedTagError creates an error for mismatched closing tags
func NewMismatchedTagError(expected, actual string, pos Position) error {
	return cuserr.NewValidationError(ErrCodeParse, ErrMsgMismatchedTag).
		WithMetadata(MetaKeyLine, strconv.Itoa(pos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(pos.Column)).
		WithMetadata(MetaKeyExpected, expected).
		WithMetadata(MetaKeyActual, actual)
}

// NewNestedRawBlockError creates an error for nested raw blocks
func NewNestedRawBlockError(pos Position) error {
	return cuserr.NewValidationError(ErrCodeParse, ErrMsgNestedRawBlock).
		WithMetadata(MetaKeyLine, strconv.Itoa(pos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(pos.Column)).
		WithMetadata(MetaKeyOffset, strconv.Itoa(pos.Offset))
}

// NewExecutionError creates an execution error with tag context
func NewExecutionError(msg string, tagName string, pos Position, cause error) error {
	var err *cuserr.CustomError
	if cause != nil {
		err = cuserr.WrapStdError(cause, ErrCodeExec, msg)
	} else {
		err = cuserr.NewInternalError(ErrCodeExec, nil)
	}
	return err.
		WithMetadata(MetaKeyTag, tagName).
		WithMetadata(MetaKeyLine, strconv.Itoa(pos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(pos.Column))
}

// NewVariableNotFoundError creates a variable not found error
func NewVariableNotFoundError(path string) error {
	return cuserr.NewNotFoundError(MetaKeyVariable, ErrMsgVariableNotFound).
		WithMetadata(MetaKeyPath, path)
}

// NewUnknownTagError creates an unknown tag error
func NewUnknownTagError(tagName string, pos Position) error {
	return cuserr.NewNotFoundError(MetaKeyResolver, ErrMsgUnknownResolver).
		WithMetadata(MetaKeyTag, tagName).
		WithMetadata(MetaKeyLine, strconv.Itoa(pos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(pos.Column))
}

// NewResolverExistsError creates a resolver collision error
func NewResolverExistsError(tagName string) error {
	return cuserr.NewValidationError(ErrCodeRegistry, ErrMsgResolverExists).
		WithMetadata(MetaKeyTag, tagName)
}

// NewMissingAttributeError creates a missing required attribute error
func NewMissingAttributeError(attrName string, tagName string) error {
	return cuserr.NewValidationError(ErrCodeValidation, ErrMsgMissingAttribute).
		WithMetadata(MetaKeyAttribute, attrName).
		WithMetadata(MetaKeyTag, tagName)
}

// NewInvalidAttributeError creates an invalid attribute value error
func NewInvalidAttributeError(attrName string, value string, reason string) error {
	return cuserr.NewValidationError(ErrCodeValidation, ErrMsgInvalidAttribute).
		WithMetadata(MetaKeyAttribute, attrName).
		WithMetadata(MetaKeyValue, value).
		WithMetadata(MetaKeyReason, reason)
}

// NewResolverError creates an error for resolver failures
func NewResolverError(resolverName string, cause error) error {
	return cuserr.WrapStdError(cause, ErrCodeExec, ErrMsgResolverFailed).
		WithMetadata(MetaKeyResolver, resolverName)
}

// NewTypeConversionError creates a type conversion error
func NewTypeConversionError(fromType, toType string, value any) error {
	return cuserr.NewValidationError(ErrCodeExec, ErrMsgTypeConversion).
		WithMetadata(MetaKeyFromType, fromType).
		WithMetadata(MetaKeyToType, toType).
		WithMetadata(MetaKeyValue, fmt.Sprintf("%v", value))
}

// NewTemplateNotFoundError creates an error for missing templates
func NewTemplateNotFoundError(name string) error {
	return cuserr.NewNotFoundError(MetaKeyTemplateName, ErrMsgTemplateNotFound).
		WithMetadata(MetaKeyTemplateName, name)
}

// NewTemplateExistsError creates an error for duplicate template registration
func NewTemplateExistsError(name string) error {
	return cuserr.NewValidationError(ErrCodeTemplate, ErrMsgTemplateAlreadyExists).
		WithMetadata(MetaKeyTemplateName, name)
}

// NewTemplateDepthError creates an error when max inclusion depth is exceeded
func NewTemplateDepthError(depth, max int) error {
	return cuserr.NewValidationError(ErrCodeTemplate, ErrMsgTemplateDepthExceeded).
		WithMetadata(MetaKeyCurrentDepth, strconv.Itoa(depth)).
		WithMetadata(MetaKeyMaxDepth, strconv.Itoa(max))
}

// NewReservedTemplateNameError creates an error for reserved namespace usage
func NewReservedTemplateNameError(name string) error {
	return cuserr.NewValidationError(ErrCodeTemplate, ErrMsgReservedTemplateName).
		WithMetadata(MetaKeyTemplateName, name)
}

// NewEmptyTemplateNameError creates an error for empty template names
func NewEmptyTemplateNameError() error {
	return cuserr.NewValidationError(ErrCodeTemplate, ErrMsgEmptyTemplateName)
}

// NewEngineNotAvailableError creates an error when engine is not in context
func NewEngineNotAvailableError() error {
	return cuserr.NewInternalError(ErrCodeTemplate, nil).
		WithMetadata(MetaKeyTag, TagNameInclude)
}

// NewFuncRegistrationError creates an error for function registration failures
func NewFuncRegistrationError(msg, funcName string) error {
	err := cuserr.NewValidationError(ErrCodeFunc, msg)
	if funcName != "" {
		err = err.WithMetadata(MetaKeyFuncName, funcName)
	}
	return err
}

// NewEnvVarNotFoundError creates an error for environment variable not found
func NewEnvVarNotFoundError(varName string) error {
	return cuserr.NewNotFoundError(ErrCodeEnv, ErrMsgEnvVarNotFound).
		WithMetadata(MetaKeyEnvVar, varName)
}

// NewEnvVarRequiredError creates an error for required environment variable not set
func NewEnvVarRequiredError(varName string) error {
	return cuserr.NewValidationError(ErrCodeEnv, ErrMsgEnvVarRequired).
		WithMetadata(MetaKeyEnvVar, varName)
}

// NewConfigBlockError creates an error for config block parsing failures
func NewConfigBlockError(msg string, pos Position, cause error) error {
	var err *cuserr.CustomError
	if cause != nil {
		err = cuserr.WrapStdError(cause, ErrCodeConfig, msg)
	} else {
		err = cuserr.NewValidationError(ErrCodeConfig, msg)
	}
	return err.
		WithMetadata(MetaKeyLine, strconv.Itoa(pos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(pos.Column)).
		WithMetadata(MetaKeyOffset, strconv.Itoa(pos.Offset))
}

// NewConfigBlockParseError creates an error for config block JSON parsing failures
func NewConfigBlockParseError(cause error) error {
	return cuserr.WrapStdError(cause, ErrCodeConfig, ErrMsgConfigBlockParse)
}

// NewInputValidationError creates an error for input validation failures
func NewInputValidationError(inputName, reason string) error {
	return cuserr.NewValidationError(ErrCodeConfig, ErrMsgInputValidationFailed).
		WithMetadata(MetaKeyInputName, inputName).
		WithMetadata(MetaKeyReason, reason)
}

// NewRequiredInputMissingError creates an error for missing required input
func NewRequiredInputMissingError(inputName string) error {
	return cuserr.NewValidationError(ErrCodeConfig, ErrMsgRequiredInputMissing).
		WithMetadata(MetaKeyInputName, inputName)
}

// NewFrontmatterError creates an error for YAML frontmatter extraction failures
func NewFrontmatterError(msg string, pos Position, cause error) error {
	var err *cuserr.CustomError
	if cause != nil {
		err = cuserr.WrapStdError(cause, ErrCodeConfig, msg)
	} else {
		err = cuserr.NewValidationError(ErrCodeConfig, msg)
	}
	return err.
		WithMetadata(MetaKeyLine, strconv.Itoa(pos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(pos.Column)).
		WithMetadata(MetaKeyOffset, strconv.Itoa(pos.Offset))
}

// NewFrontmatterParseError creates an error for YAML frontmatter parsing failures
func NewFrontmatterParseError(cause error) error {
	return cuserr.WrapStdError(cause, ErrCodeConfig, ErrMsgFrontmatterParse)
}

// NewMessageTagError creates an error for message tag validation failures
func NewMessageTagError(msg string, tagPos Position) error {
	return cuserr.NewValidationError(ErrCodeValidation, msg).
		WithMetadata(MetaKeyTag, TagNameMessage).
		WithMetadata(MetaKeyLine, strconv.Itoa(tagPos.Line)).
		WithMetadata(MetaKeyColumn, strconv.Itoa(tagPos.Column))
}

// NewLabelNotFoundError creates an error for label not found.
func NewLabelNotFoundError(templateName, label string) error {
	return cuserr.NewNotFoundError(ErrCodeLabel, ErrMsgLabelNotFound).
		WithMetadata(MetaKeyTemplateName, templateName).
		WithMetadata(MetaKeyLabel, label)
}

// NewInvalidLabelNameError creates an error for invalid label name.
func NewInvalidLabelNameError(label, reason string) error {
	return cuserr.NewValidationError(ErrCodeLabel, ErrMsgInvalidLabelName).
		WithMetadata(MetaKeyLabel, label).
		WithMetadata(MetaKeyReason, reason)
}

// NewInvalidStatusTransitionError creates an error for invalid status transition.
func NewInvalidStatusTransitionError(from, to DeploymentStatus) error {
	return cuserr.NewValidationError(ErrCodeStatus, ErrMsgStatusTransitionDenied).
		WithMetadata(MetaKeyFromStatus, string(from)).
		WithMetadata(MetaKeyToStatus, string(to))
}

// NewArchivedVersionError creates an error for operations on archived versions.
func NewArchivedVersionError(templateName string, version int) error {
	return cuserr.NewValidationError(ErrCodeStatus, ErrMsgArchivedVersionReadOnly).
		WithMetadata(MetaKeyTemplateName, templateName).
		WithMetadata(MetaKeyVersion, strconv.Itoa(version))
}

// NewInvalidDeploymentStatusError creates an error for invalid deployment status value.
func NewInvalidDeploymentStatusError(status string) error {
	return cuserr.NewValidationError(ErrCodeStatus, ErrMsgInvalidDeploymentStatus).
		WithMetadata(MetaKeyStatus, status)
}

// NewSchemaValidationError creates an error for schema validation failures.
func NewSchemaValidationError(msg, path string) error {
	return cuserr.NewValidationError(ErrCodeSchema, msg).
		WithMetadata(MetaKeyPath, path)
}

// NewSchemaProviderError creates an error for provider-specific schema issues.
func NewSchemaProviderError(msg, provider string) error {
	return cuserr.NewValidationError(ErrCodeSchema, msg).
		WithMetadata(MetaKeyProvider, provider)
}

// NewPromptValidationError creates an error for prompt validation failures.
func NewPromptValidationError(msg, promptName string) error {
	return cuserr.NewValidationError(ErrCodePrompt, msg).
		WithMetadata(MetaKeyPromptName, promptName)
}

// NewPromptNameRequiredError creates an error for missing prompt name.
func NewPromptNameRequiredError() error {
	return cuserr.NewValidationError(ErrCodePrompt, ErrMsgPromptNameRequired)
}

// NewPromptNameTooLongError creates an error for prompt name exceeding max length.
func NewPromptNameTooLongError(name string, maxLen int) error {
	return cuserr.NewValidationError(ErrCodePrompt, ErrMsgPromptNameTooLong).
		WithMetadata(MetaKeyPromptName, name).
		WithMetadata(MetaKeyMaxDepth, strconv.Itoa(maxLen))
}

// NewPromptNameInvalidFormatError creates an error for invalid prompt name format.
func NewPromptNameInvalidFormatError(name string) error {
	return cuserr.NewValidationError(ErrCodePrompt, ErrMsgPromptNameInvalidFormat).
		WithMetadata(MetaKeyPromptName, name)
}

// NewPromptDescriptionRequiredError creates an error for missing prompt description.
func NewPromptDescriptionRequiredError() error {
	return cuserr.NewValidationError(ErrCodePrompt, ErrMsgPromptDescriptionRequired)
}

// NewPromptDescriptionTooLongError creates an error for prompt description exceeding max length.
func NewPromptDescriptionTooLongError(maxLen int) error {
	return cuserr.NewValidationError(ErrCodePrompt, ErrMsgPromptDescriptionTooLong).
		WithMetadata(MetaKeyMaxDepth, strconv.Itoa(maxLen))
}

// NewRefNotFoundError creates an error for referenced prompt not found.
func NewRefNotFoundError(slug, version string) error {
	return cuserr.NewNotFoundError(ErrCodeRef, ErrMsgRefNotFound).
		WithMetadata(MetaKeyPromptSlug, slug).
		WithMetadata(AttrVersion, version)
}

// NewRefCircularError creates an error for circular reference detection.
func NewRefCircularError(slug string, chain []string) error {
	chainStr := ""
	for i, s := range chain {
		if i > 0 {
			chainStr += " -> "
		}
		chainStr += s
	}
	return cuserr.NewValidationError(ErrCodeRef, ErrMsgRefCircular).
		WithMetadata(MetaKeyPromptSlug, slug).
		WithMetadata(MetaKeyRefChain, chainStr)
}

// NewRefDepthExceededError creates an error for reference resolution depth exceeded.
func NewRefDepthExceededError(depth, maxDepth int) error {
	return cuserr.NewValidationError(ErrCodeRef, ErrMsgRefDepthExceeded).
		WithMetadata(MetaKeyCurrentDepth, strconv.Itoa(depth)).
		WithMetadata(MetaKeyMaxDepth, strconv.Itoa(maxDepth))
}

// NewRefMissingSlugError creates an error for missing slug attribute in prompty.ref.
func NewRefMissingSlugError() error {
	return cuserr.NewValidationError(ErrCodeRef, ErrMsgRefMissingSlug).
		WithMetadata(MetaKeyTag, TagNameRef)
}

// NewRefInvalidSlugError creates an error for invalid slug format in prompty.ref.
func NewRefInvalidSlugError(slug string) error {
	return cuserr.NewValidationError(ErrCodeRef, ErrMsgRefInvalidSlug).
		WithMetadata(MetaKeyPromptSlug, slug)
}

// v2.1 Agent error constructors

// NewAgentError creates an error for agent-related failures.
func NewAgentError(msg string, cause error) error {
	if cause != nil {
		return cuserr.WrapStdError(cause, ErrCodeAgent, msg)
	}
	return cuserr.NewValidationError(ErrCodeAgent, msg)
}

// NewAgentValidationError creates an error for agent validation failures.
func NewAgentValidationError(msg, promptName string) error {
	return cuserr.NewValidationError(ErrCodeAgent, msg).
		WithMetadata(MetaKeyPromptName, promptName)
}

// NewCompilationError creates an error for agent compilation failures.
func NewCompilationError(msg string, cause error) error {
	if cause != nil {
		return cuserr.WrapStdError(cause, ErrCodeCompile, msg)
	}
	return cuserr.NewValidationError(ErrCodeCompile, msg)
}

// NewCatalogError creates an error for catalog generation failures.
func NewCatalogError(msg string, cause error) error {
	if cause != nil {
		return cuserr.WrapStdError(cause, ErrCodeCatalog, msg)
	}
	return cuserr.NewValidationError(ErrCodeCatalog, msg)
}

// NewSkillNotFoundError creates an error for a skill reference that could not be resolved.
func NewSkillNotFoundError(slug string) error {
	return cuserr.NewNotFoundError(ErrCodeAgent, ErrMsgSkillNotFound).
		WithMetadata(MetaKeySkillSlug, slug)
}

// NewCompileMessageError creates an error for message compilation failures with index context.
func NewCompileMessageError(messageIndex int, role string, cause error) error {
	return cuserr.WrapStdError(cause, ErrCodeCompile, ErrMsgCompileMessageFailed).
		WithMetadata(MetaKeyMessageIndex, strconv.Itoa(messageIndex)).
		WithMetadata(MetaKeyMessageRole, role).
		WithMetadata(MetaKeyCompileStage, "messages")
}

// NewCompileSkillError creates an error for skill compilation failures with slug context.
func NewCompileSkillError(skillSlug string, cause error) error {
	return cuserr.WrapStdError(cause, ErrCodeCompile, ErrMsgCompileSkillFailed).
		WithMetadata(MetaKeySkillSlug, skillSlug).
		WithMetadata(MetaKeyCompileStage, "skill_activation")
}

// NewCompileBodyError creates an error for body compilation failures with stage context.
func NewCompileBodyError(cause error) error {
	return cuserr.WrapStdError(cause, ErrCodeCompile, ErrMsgCompileBodyFailed).
		WithMetadata(MetaKeyCompileStage, "body")
}

// NewProviderMessageError creates an error for unsupported provider in message serialization.
func NewProviderMessageError(provider string) error {
	return cuserr.NewValidationError(ErrCodeCompile, ErrMsgUnsupportedMsgProvider).
		WithMetadata(MetaKeyProvider, provider)
}

// NewInvalidDocumentTypeError creates an error for invalid document type.
func NewInvalidDocumentTypeError(docType string) error {
	return cuserr.NewValidationError(ErrCodeAgent, ErrMsgInvalidDocumentType).
		WithMetadata(MetaKeyDocumentType, docType)
}

// Versioning error constructors

// NewVersioningError creates an error for versioning operation failures.
func NewVersioningError(msg string, cause error) error {
	if cause != nil {
		return cuserr.WrapStdError(cause, ErrCodeVersioning, msg)
	}
	return cuserr.NewValidationError(ErrCodeVersioning, msg)
}

// NewVersionGetError creates an error for version retrieval failures.
func NewVersionGetError(version int, cause error) error {
	return cuserr.WrapStdError(cause, ErrCodeVersioning, ErrMsgVersionGetFailed).
		WithMetadata(AttrVersion, strconv.Itoa(version))
}

// NewVersionTemplateExistsError creates an error when a template already exists.
func NewVersionTemplateExistsError(name string) error {
	return cuserr.NewValidationError(ErrCodeVersioning, ErrMsgVersionTemplateExists).
		WithMetadata(MetaKeyTemplateName, name)
}
