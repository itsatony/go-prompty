package prompty

import (
	"fmt"
	"strconv"

	"github.com/itsatony/go-cuserr"
)

// Error message constants - ALL error messages must be constants (NO MAGIC STRINGS)
const (
	// Parse errors
	ErrMsgParseFailed       = "template parsing failed"
	ErrMsgInvalidSyntax     = "invalid template syntax"
	ErrMsgUnexpectedChar    = "unexpected character"
	ErrMsgUnterminatedTag   = "unterminated tag"
	ErrMsgUnterminatedStr   = "unterminated string literal"
	ErrMsgInvalidEscape     = "invalid escape sequence"
	ErrMsgUnexpectedEOF     = "unexpected end of input"
	ErrMsgMismatchedTag     = "mismatched closing tag"
	ErrMsgInvalidTagName    = "invalid tag name"
	ErrMsgEmptyTagName      = "tag name cannot be empty"
	ErrMsgNestedRawBlock    = "nested raw blocks are not allowed"

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
	ErrMsgValidationFailed      = "template validation failed"
	ErrMsgUnknownTagInTemplate  = "unknown tag in template"
	ErrMsgInvalidOnErrorAttr    = "invalid onerror attribute value"
	ErrMsgMissingIncludeTarget  = "included template not found"

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
