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
)

// Error code constants for categorization
const (
	ErrCodeParse      = "PROMPTY_PARSE"
	ErrCodeExec       = "PROMPTY_EXEC"
	ErrCodeValidation = "PROMPTY_VALIDATION"
	ErrCodeRegistry   = "PROMPTY_REGISTRY"
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
		WithMetadata("reason", reason)
}

// NewResolverError creates an error for resolver failures
func NewResolverError(resolverName string, cause error) error {
	return cuserr.WrapStdError(cause, ErrCodeExec, ErrMsgResolverFailed).
		WithMetadata(MetaKeyResolver, resolverName)
}

// NewTypeConversionError creates a type conversion error
func NewTypeConversionError(fromType, toType string, value interface{}) error {
	return cuserr.NewValidationError(ErrCodeExec, ErrMsgTypeConversion).
		WithMetadata("from_type", fromType).
		WithMetadata("to_type", toType).
		WithMetadata(MetaKeyValue, fmt.Sprintf("%v", value))
}
