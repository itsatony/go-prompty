package internal

// TokenType represents the type of a lexical token
type TokenType string

// Token type constants
const (
	TokenTypeText       TokenType = "TEXT"
	TokenTypeOpenTag    TokenType = "OPEN_TAG"
	TokenTypeCloseTag   TokenType = "CLOSE_TAG"
	TokenTypeSelfClose  TokenType = "SELF_CLOSE"
	TokenTypeBlockClose TokenType = "BLOCK_CLOSE"
	TokenTypeTagName    TokenType = "TAG_NAME"
	TokenTypeAttrName   TokenType = "ATTR_NAME"
	TokenTypeAttrValue  TokenType = "ATTR_VALUE"
	TokenTypeEquals     TokenType = "EQUALS"
	TokenTypeEOF        TokenType = "EOF"
)

// NodeType identifies AST node types
type NodeType int

// Node type constants
const (
	NodeTypeRoot NodeType = iota
	NodeTypeText
	NodeTypeTag
	NodeTypeRaw
	NodeTypeConditional
	NodeTypeFor    // Phase 4: Loop node
	NodeTypeSwitch // Phase 5: Switch/case node
)

// Node type string names for debugging
const (
	NodeTypeNameRoot        = "ROOT"
	NodeTypeNameText        = "TEXT"
	NodeTypeNameTag         = "TAG"
	NodeTypeNameRaw         = "RAW"
	NodeTypeNameConditional = "CONDITIONAL"
	NodeTypeNameFor         = "FOR"
	NodeTypeNameSwitch      = "SWITCH"
)

// String returns the string representation of the node type
func (n NodeType) String() string {
	switch n {
	case NodeTypeRoot:
		return NodeTypeNameRoot
	case NodeTypeText:
		return NodeTypeNameText
	case NodeTypeTag:
		return NodeTypeNameTag
	case NodeTypeRaw:
		return NodeTypeNameRaw
	case NodeTypeConditional:
		return NodeTypeNameConditional
	case NodeTypeFor:
		return NodeTypeNameFor
	case NodeTypeSwitch:
		return NodeTypeNameSwitch
	default:
		return NodeTypeNameRoot
	}
}

// Lexer state constants
const (
	LexStateText  = "TEXT"
	LexStateTag   = "TAG"
	LexStateAttr  = "ATTR"
	LexStateValue = "VALUE"
)

// Character constants
const (
	CharEquals       = '='
	CharDoubleQuote  = '"'
	CharSingleQuote  = '\''
	CharBackslash    = '\\'
	CharSlash        = '/'
	CharNewline      = '\n'
	CharSpace        = ' '
	CharTab          = '\t'
	CharCarriageRet  = '\r'
)

// String constants for delimiter matching
const (
	StrOpenDelim   = "{~"
	StrCloseDelim  = "~}"
	StrSelfClose   = "/~}"
	StrBlockClose  = "{~/"
	StrEscapeOpen  = "\\{~"
)

// Delimiter lengths
const (
	LenOpenDelim   = 2 // {~
	LenCloseDelim  = 2 // ~}
	LenSelfClose   = 3 // /~}
	LenBlockClose  = 3 // {~/
	LenEscapeOpen  = 3 // \{~
)

// Log message constants
const (
	LogMsgLexerCreated     = "lexer created"
	LogMsgTokenizerStart   = "starting tokenization"
	LogMsgTokenizerEnd     = "tokenization complete"
	LogMsgParserCreated    = "parser created"
	LogMsgParserStart      = "starting parse"
	LogMsgParserEnd        = "parse complete"
	LogMsgExecutorCreated  = "executor created"
	LogMsgExecutorStart    = "starting execution"
	LogMsgExecutorEnd      = "execution complete"
	LogMsgResolverInvoked  = "resolver invoked"
	LogMsgResolverComplete = "resolver complete"
	LogMsgRegistryCreated  = "registry created"
	LogMsgResolverRegistered = "resolver registered"
	LogMsgResolverCollision  = "resolver registration collision - first-come-wins"
)

// Log field names
const (
	LogFieldSource       = "source_length"
	LogFieldTokens       = "token_count"
	LogFieldNodes        = "node_count"
	LogFieldTag          = "tag"
	LogFieldResolver     = "resolver"
	LogFieldDuration     = "duration"
	LogFieldLine         = "line"
	LogFieldColumn       = "column"
	LogFieldTemplateName = "template_name"
	LogFieldDepth        = "depth"
	LogFieldBranch       = "branch"
	LogFieldExpression   = "expression"
	LogFieldResult       = "result"
)

// Built-in tag names (mirror public constants for internal use)
const (
	TagNameVar         = "prompty.var"
	TagNameRaw         = "prompty.raw"
	TagNameInclude     = "prompty.include"
	TagNameIf          = "prompty.if"
	TagNameElseIf      = "prompty.elseif"
	TagNameElse        = "prompty.else"
	TagNameComment     = "prompty.comment"     // Phase 3
	TagNameFor         = "prompty.for"         // Phase 4
	TagNameSwitch      = "prompty.switch"      // Phase 5
	TagNameCase        = "prompty.case"        // Phase 5
	TagNameCaseDefault = "prompty.casedefault" // Phase 5
)

// Attribute name constants
const (
	AttrName     = "name"
	AttrDefault  = "default"
	AttrTemplate = "template"
	AttrWith     = "with"
	AttrIsolate  = "isolate"
	AttrEval     = "eval"    // Condition expression for if/elseif
	AttrOnError  = "onerror" // Per-tag error strategy override
	AttrItem     = "item"    // Loop variable name (Phase 4)
	AttrIndex    = "index"   // Loop index variable name (Phase 4)
	AttrIn       = "in"      // Loop collection path (Phase 4)
	AttrLimit    = "limit"   // Loop iteration limit (Phase 4)
	AttrValue    = "value"   // Case value for switch/case (Phase 5)
)

// Boolean attribute values
const (
	AttrValueTrue  = "true"
	AttrValueFalse = "false"
)

// Error message constants for include resolver
const (
	ErrMsgMissingTemplateAttr = "missing required 'template' attribute"
	ErrMsgEngineNotAvailable  = "engine not available in context"
	ErrMsgTemplateNotFoundFmt = "template not found: %s"
	ErrMsgDepthExceeded       = "maximum template inclusion depth exceeded"
)

// Meta key constants for internal data passing
const (
	MetaKeyParentDepth = "_parentDepth"
	MetaKeyValue       = "_value"
)

// Log messages for template operations
const (
	LogMsgTemplateRegistered = "template registered"
	LogMsgTemplateIncluded   = "template included"
	LogMsgIncludeDepthCheck  = "checking include depth"
)

// Error messages for conditional resolver
const (
	ErrMsgCondMissingEval   = "missing required 'eval' attribute"
	ErrMsgCondInvalidElse   = "else tag cannot have eval attribute"
	ErrMsgCondUnexpectedTag = "unexpected conditional tag"
	ErrMsgCondNotClosed     = "conditional block not closed"
	ErrMsgCondElseNotLast   = "else must be last in conditional chain"
	ErrMsgCondExprFailed    = "condition expression evaluation failed"
)

// Log messages for conditional operations
const (
	LogMsgConditionEval  = "evaluating condition"
	LogMsgBranchSelected = "branch selected"
)

// Error format string constants (for Error() methods)
const (
	ErrFmtWithPosition       = "%s at %s"
	ErrFmtWithTagAndPosition = "%s [%s] at %s"
	ErrFmtWithCause          = "%s: %v"
	ErrFmtTagMessage         = "%s: %s"
	ErrFmtTypeComparison     = "cannot compare %T and %T"
)

// String format constants for AST String() methods
const (
	FmtOpenBrace    = "{"
	FmtCloseBrace   = "}"
	FmtCommaSep     = ", "
	FmtKeyValueSep  = "="
	FmtEmptyBraces  = "{}"
)

// ErrorStrategy mirrors prompty.ErrorStrategy for internal use
type ErrorStrategy int

// Error strategy constants (mirrors prompty.ErrorStrategy)
const (
	ErrorStrategyThrow ErrorStrategy = iota
	ErrorStrategyDefault
	ErrorStrategyRemove
	ErrorStrategyKeepRaw
	ErrorStrategyLog
)

// ErrorStrategyNotSet is a sentinel value indicating no strategy override
const ErrorStrategyNotSet ErrorStrategy = -1

// Error strategy name constants for parsing
const (
	ErrorStrategyNameThrow   = "throw"
	ErrorStrategyNameDefault = "default"
	ErrorStrategyNameRemove  = "remove"
	ErrorStrategyNameKeepRaw = "keepraw"
	ErrorStrategyNameLog     = "log"
)

// ParseErrorStrategy parses a string into an ErrorStrategy.
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

// String returns the string representation of the error strategy.
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

// Log messages for error strategy handling
const (
	LogMsgErrorStrategyApplied = "error strategy applied"
	LogMsgErrorLogged          = "error logged and execution continued"
	LogFieldStrategy           = "strategy"
	LogFieldErrorMsg           = "error_message"
)

// Error messages for for loop (Phase 4)
const (
	ErrMsgForMissingItem    = "missing required 'item' attribute"
	ErrMsgForMissingIn      = "missing required 'in' attribute"
	ErrMsgForInvalidLimit   = "invalid 'limit' attribute value"
	ErrMsgForCollectionPath = "collection path not found"
	ErrMsgForNotIterable    = "value is not iterable"
	ErrMsgForLimitExceeded  = "loop iteration limit exceeded"
	ErrMsgForNotClosed      = "for block not closed"
	ErrMsgForContextNoChild = "context does not support child creation"
	ErrMsgForTypeNotIterable = "type %s is not iterable"
)

// Log messages for for loop operations (Phase 4)
const (
	LogMsgForStart        = "starting for loop"
	LogMsgForIteration    = "for loop iteration"
	LogMsgForEnd          = "for loop complete"
	LogMsgForLimitApplied = "loop limit applied"
)

// Log field names for for loop (Phase 4)
const (
	LogFieldIteration  = "iteration"
	LogFieldCollection = "collection"
	LogFieldItemVar    = "item_var"
	LogFieldIndexVar   = "index_var"
)

// Default values for for loop (Phase 4)
const (
	DefaultMaxLoopIterations = 10000
)

// Map iteration field names (Phase 4)
// When iterating over a map, each item is a map with these keys.
const (
	ForMapKeyField   = "key"
	ForMapValueField = "value"
)

// Error messages for switch/case (Phase 5)
const (
	ErrMsgSwitchMissingEval      = "missing required 'eval' attribute for switch"
	ErrMsgSwitchMissingValue     = "case requires 'value' or 'eval' attribute"
	ErrMsgSwitchNotClosed        = "switch block not closed"
	ErrMsgSwitchCaseNotClosed    = "case block not closed"
	ErrMsgSwitchDefaultNotLast   = "default case must be last in switch"
	ErrMsgSwitchDuplicateDefault = "only one default case allowed in switch"
	ErrMsgSwitchInvalidCaseTag   = "unexpected tag inside switch block"
)

// Log messages for switch/case operations (Phase 5)
const (
	LogMsgSwitchEval    = "evaluating switch expression"
	LogMsgSwitchCase    = "evaluating switch case"
	LogMsgCaseMatch     = "switch case matched"
	LogMsgCaseDefault   = "switch default case selected"
	LogMsgSwitchNoMatch = "no switch case matched"
)

// Log field names for switch/case (Phase 5)
const (
	LogFieldCaseValue = "case_value"
	LogFieldCaseEval  = "case_eval"
)
