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
)

// Node type string names for debugging
const (
	NodeTypeNameRoot        = "ROOT"
	NodeTypeNameText        = "TEXT"
	NodeTypeNameTag         = "TAG"
	NodeTypeNameRaw         = "RAW"
	NodeTypeNameConditional = "CONDITIONAL"
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
	TagNameVar     = "prompty.var"
	TagNameRaw     = "prompty.raw"
	TagNameInclude = "prompty.include"
	TagNameIf      = "prompty.if"
	TagNameElseIf  = "prompty.elseif"
	TagNameElse    = "prompty.else"
)

// Attribute name constants
const (
	AttrName     = "name"
	AttrDefault  = "default"
	AttrTemplate = "template"
	AttrWith     = "with"
	AttrIsolate  = "isolate"
	AttrEval     = "eval" // Condition expression for if/elseif
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
