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
	TagNameVar     = "prompty.var"
	TagNameRaw     = "prompty.raw"
	TagNameInclude = "prompty.include" // Nested template inclusion
	TagNameIf      = "prompty.if"      // Phase 2
	TagNameElseIf  = "prompty.elseif"  // Phase 2
	TagNameElse    = "prompty.else"    // Phase 2
	TagNameFor     = "prompty.for"     // Phase 4
	TagNameComment = "prompty.comment" // Phase 3
	TagNameDefault = "prompty.default" // Phase 3
	TagNameSwitch  = "prompty.switch"  // Phase 5
	TagNameCase    = "prompty.case"    // Phase 5
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
)
