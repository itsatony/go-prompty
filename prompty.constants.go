package prompty

import "time"

// Delimiter constants - the {~ ~} syntax chosen for minimal collision with prompt content
const (
	DefaultOpenDelim   = "{~"
	DefaultCloseDelim  = "~}"
	DefaultSelfClose   = "/~}"
	DefaultBlockClose  = "{~/"
)

// Built-in tag names - all use prompty. namespace prefix
const (
	TagNameVar         = "prompty.var"
	TagNameRaw         = "prompty.raw"
	TagNameInclude     = "prompty.include"     // Nested template inclusion
	TagNameIf          = "prompty.if"          // Phase 2
	TagNameElseIf      = "prompty.elseif"      // Phase 2
	TagNameElse        = "prompty.else"        // Phase 2
	TagNameFor         = "prompty.for"         // Phase 4
	TagNameComment     = "prompty.comment"     // Phase 3
	TagNameDefault     = "prompty.default"     // Phase 3
	TagNameSwitch      = "prompty.switch"      // Phase 5
	TagNameCase        = "prompty.case"        // Phase 5
	TagNameCaseDefault = "prompty.casedefault" // Phase 5 - default case in switch
	TagNameEnv         = "prompty.env"         // Environment variable resolver
	TagNameConfig      = "prompty.config"      // Inference configuration block
)

// Reserved namespace prefix for built-in tags
const (
	ReservedNamespacePrefix = "prompty."
)

// Attribute name constants
const (
	AttrName     = "name"
	AttrDefault  = "default"
	AttrEval     = "eval"
	AttrOnError  = "onerror"
	AttrFormat   = "format"
	AttrEscape   = "escape"
	AttrItem     = "item"
	AttrIndex    = "index"
	AttrIn       = "in"
	AttrLimit    = "limit"
	AttrValue    = "value"
	AttrText     = "text"
	AttrTemplate = "template" // Template name for include
	AttrWith     = "with"     // Context path for include
	AttrIsolate  = "isolate"  // Isolated context flag for include
	AttrRequired = "required" // Required flag for env resolver
)

// Boolean attribute values
const (
	AttrValueTrue  = "true"
	AttrValueFalse = "false"
)

// ErrorStrategy defines how to handle errors during execution
type ErrorStrategy int

const (
	// ErrorStrategyThrow stops execution and returns the error
	ErrorStrategyThrow ErrorStrategy = iota
	// ErrorStrategyDefault replaces failed content with a default value
	ErrorStrategyDefault
	// ErrorStrategyRemove removes the tag entirely from output
	ErrorStrategyRemove
	// ErrorStrategyKeepRaw keeps the original tag text in output
	ErrorStrategyKeepRaw
	// ErrorStrategyLog logs the error and continues with empty string
	ErrorStrategyLog
)

// Error strategy string values for attribute parsing
const (
	ErrorStrategyNameThrow   = "throw"
	ErrorStrategyNameDefault = "default"
	ErrorStrategyNameRemove  = "remove"
	ErrorStrategyNameKeepRaw = "keepraw"
	ErrorStrategyNameLog     = "log"
)

// String returns the string representation of the error strategy
func (s ErrorStrategy) String() string {
	switch s {
	case ErrorStrategyThrow:
		return ErrorStrategyNameThrow
	case ErrorStrategyDefault:
		return ErrorStrategyNameDefault
	case ErrorStrategyRemove:
		return ErrorStrategyNameRemove
	case ErrorStrategyKeepRaw:
		return ErrorStrategyNameKeepRaw
	case ErrorStrategyLog:
		return ErrorStrategyNameLog
	default:
		return ErrorStrategyNameThrow
	}
}

// Default configuration values
const (
	DefaultExecutionTimeout  = 30 * time.Second
	DefaultResolverTimeout   = 5 * time.Second
	DefaultFunctionTimeout   = 1 * time.Second
	DefaultMaxLoopIterations = 10000
	DefaultMaxDepth          = 10
	DefaultMaxOutputSize     = 10 * 1024 * 1024 // 10MB
)

// Metadata keys for cuserr.WithMetadata
const (
	MetaKeyLine         = "line"
	MetaKeyColumn       = "column"
	MetaKeyOffset       = "offset"
	MetaKeyTag          = "tag"
	MetaKeyResolver     = "resolver"
	MetaKeyVariable     = "variable"
	MetaKeyAttribute    = "attribute"
	MetaKeyExpected     = "expected"
	MetaKeyActual       = "actual"
	MetaKeyPath         = "path"
	MetaKeyValue        = "value"
	MetaKeyTemplateName = "template_name"
	MetaKeyCurrentDepth = "current_depth"
	MetaKeyMaxDepth     = "max_depth"
	MetaKeyFuncName     = "func_name"
	MetaKeyReason       = "reason"
	MetaKeyFromType     = "from_type"
	MetaKeyToType       = "to_type"
	MetaKeyEnvVar       = "env_var"
	MetaKeyInputName    = "input_name"
)

// Escape sequence constants
const (
	EscapeOpenDelim  = "\\{~"
	LiteralOpenDelim = "{~"
)

// Internal meta keys for nested template data passing
// These are used internally and prefixed with underscore to avoid collision
const (
	MetaKeyParentDepth = "_parentDepth" // Used to pass depth between nested template executions
	MetaKeyValueData   = "_value"       // Used to pass non-map values in with attribute
	MetaKeyRawSource   = "_rawSource"   // Original tag source for keepRaw strategy
	MetaKeyStrategy    = "strategy"     // Applied error strategy for logging
)

// ParseErrorStrategy parses a string into an ErrorStrategy.
// Returns ErrorStrategyThrow for unknown values.
func ParseErrorStrategy(s string) ErrorStrategy {
	switch s {
	case ErrorStrategyNameDefault:
		return ErrorStrategyDefault
	case ErrorStrategyNameRemove:
		return ErrorStrategyRemove
	case ErrorStrategyNameKeepRaw:
		return ErrorStrategyKeepRaw
	case ErrorStrategyNameLog:
		return ErrorStrategyLog
	case ErrorStrategyNameThrow:
		return ErrorStrategyThrow
	default:
		return ErrorStrategyThrow
	}
}

// IsValidErrorStrategy checks if a string is a valid error strategy name.
func IsValidErrorStrategy(s string) bool {
	switch s {
	case ErrorStrategyNameThrow, ErrorStrategyNameDefault,
		ErrorStrategyNameRemove, ErrorStrategyNameKeepRaw, ErrorStrategyNameLog:
		return true
	default:
		return false
	}
}

// ValidationSeverity indicates the severity of a validation issue.
type ValidationSeverity int

const (
	// SeverityError indicates a critical issue that prevents execution
	SeverityError ValidationSeverity = iota
	// SeverityWarning indicates a potential issue that may cause problems
	SeverityWarning
	// SeverityInfo indicates informational feedback
	SeverityInfo
)

// Validation severity string names
const (
	SeverityNameError   = "error"
	SeverityNameWarning = "warning"
	SeverityNameInfo    = "info"
)

// String returns the string representation of the validation severity
func (s ValidationSeverity) String() string {
	switch s {
	case SeverityError:
		return SeverityNameError
	case SeverityWarning:
		return SeverityNameWarning
	case SeverityInfo:
		return SeverityNameInfo
	default:
		return SeverityNameError
	}
}

// ErrorStrategyNotSet is a sentinel value indicating no strategy override
const ErrorStrategyNotSet ErrorStrategy = -1

// Storage ID prefixes
const (
	TemplateIDPrefix = "tmpl_"
)

// Storage driver names
const (
	StorageDriverNameMemory     = "memory"
	StorageDriverNameFilesystem = "filesystem"
)

// Model API types
const (
	ModelAPIChat       = "chat"
	ModelAPICompletion = "completion"
)

// Input/Output schema types
const (
	SchemaTypeString  = "string"
	SchemaTypeNumber  = "number"
	SchemaTypeBoolean = "boolean"
	SchemaTypeArray   = "array"
	SchemaTypeObject  = "object"
)

// Model parameter map keys (for ToMap conversion)
const (
	ParamKeyTemperature      = "temperature"
	ParamKeyMaxTokens        = "max_tokens"
	ParamKeyTopP             = "top_p"
	ParamKeyFrequencyPenalty = "frequency_penalty"
	ParamKeyPresencePenalty  = "presence_penalty"
	ParamKeyStop             = "stop"
	ParamKeySeed             = "seed"
)

// Error format strings for type validation
const (
	ErrFmtTypeMismatch = "expected %s, got %s"
)

// Storage error messages
const (
	ErrMsgCryptoRandFailure    = "cryptographic random number generator failure"
	ErrMsgPathTraversalDetected = "invalid template name: path traversal characters detected"
)

// Access control message formats
const (
	ErrFmtOperationAllowed    = "operation %s is allowed"
	ErrFmtOperationNotAllowed = "operation %s is not allowed"
)
