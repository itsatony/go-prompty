package prompty

import "time"

// Delimiter constants - the {~ ~} syntax chosen for minimal collision with prompt content
const (
	DefaultOpenDelim  = "{~"
	DefaultCloseDelim = "~}"
	DefaultSelfClose  = "/~}"
	DefaultBlockClose = "{~/"
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
	TagNameConfig      = "prompty.config"      // Legacy inference configuration block (JSON)
	TagNameExtends     = "prompty.extends"     // Template inheritance - extends parent
	TagNameBlock       = "prompty.block"       // Template inheritance - overridable block
	TagNameParent      = "prompty.parent"      // Template inheritance - call parent block content
	TagNameMessage     = "prompty.message"     // Conversation message for chat API
	TagNameRef         = "prompty.ref"         // v2.0: Prompt reference resolver
)

// YAML frontmatter constants
const (
	// YAMLFrontmatterDelimiter is the standard YAML frontmatter delimiter
	YAMLFrontmatterDelimiter = "---"
)

// Message role constants for prompty.message tag
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// Message attribute constants
const (
	AttrRole  = "role"
	AttrCache = "cache"
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
	AttrSlug     = "slug"     // v2.0: Prompt slug for reference
	AttrVersion  = "version"  // v2.0: Prompt version for reference
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
	DefaultExecutionTimeout   = 30 * time.Second
	DefaultResolverTimeout    = 5 * time.Second
	DefaultFunctionTimeout    = 1 * time.Second
	DefaultMaxLoopIterations  = 10000
	DefaultMaxDepth           = 10
	DefaultMaxOutputSize      = 10 * 1024 * 1024 // 10MB
	DefaultMaxFrontmatterSize = 64 * 1024        // 64KB - DoS protection for YAML frontmatter
)

// Cache configuration defaults
const (
	DefaultCacheTTL              = 5 * time.Minute
	DefaultCacheMaxEntries       = 1000
	DefaultNegativeCacheTTL      = 30 * time.Second
	DefaultAccessCacheTTL        = 5 * time.Minute
	DefaultAccessCacheMaxEntries = 10000
	DefaultResultCacheTTL        = 5 * time.Minute
	DefaultResultCacheMaxEntries = 1000
	DefaultResultCacheMaxSize    = 1 << 20 // 1MB
)

// Filesystem storage constants
const (
	FilesystemDirPermissions  = 0755
	FilesystemFilePermissions = 0644
	FilesystemVersionPrefix   = "v"
	FilesystemVersionSuffix   = ".json"
)

// Document export/import constants
const (
	DocumentFilenameAgent  = "AGENT.md"
	DocumentFilenamePrompt = "PROMPT.md"
	DocumentFilenameSkill  = "SKILL.md"
	FileExtensionMarkdown  = ".md"
	FileExtensionZip       = ".zip"
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
	MetaKeyPromptSlug   = "prompt_slug"     // v2.0: Prompt slug for reference errors
	MetaKeyPromptName   = "prompt_name"     // v2.0: Prompt name for validation errors
	MetaKeyRefChain     = "reference_chain" // v2.0: Reference chain for circular detection
	MetaKeyLabel        = "label"           // Label name for label operations
	MetaKeyFromStatus   = "from_status"     // Source status in transitions
	MetaKeyToStatus     = "to_status"       // Target status in transitions
	MetaKeyVersion      = "version"         // Version number
	MetaKeyStatus       = "status"          // Deployment status value
	MetaKeyProvider     = "provider"        // LLM provider name
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
	StorageDriverNamePostgres   = "postgres"
)

// PostgreSQL storage driver configuration defaults
const (
	PostgresTablePrefix            = "prompty_"
	PostgresDefaultMaxOpenConns    = 25
	PostgresDefaultMaxIdleConns    = 5
	PostgresDefaultConnMaxLifetime = 5 * time.Minute
	PostgresDefaultConnMaxIdleTime = 5 * time.Minute
	PostgresDefaultQueryTimeout    = 30 * time.Second
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
	ParamKeyTemperature       = "temperature"
	ParamKeyMaxTokens         = "max_tokens"
	ParamKeyTopP              = "top_p"
	ParamKeyFrequencyPenalty  = "frequency_penalty"
	ParamKeyPresencePenalty   = "presence_penalty"
	ParamKeyStop              = "stop"
	ParamKeySeed              = "seed"
	ParamKeyMinP              = "min_p"
	ParamKeyRepetitionPenalty = "repetition_penalty"
	ParamKeyLogprobs          = "logprobs"
	ParamKeyTopLogprobs       = "top_logprobs"
	ParamKeyStopTokenIDs      = "stop_token_ids"
	ParamKeyLogitBias         = "logit_bias"
	ParamKeyModel             = "model"
	ParamKeyTopK              = "top_k"
	ParamKeyStopSequences     = "stop_sequences"
)

// Anthropic-specific parameter keys
const (
	ParamKeyAnthropicThinking     = "thinking"
	ParamKeyAnthropicOutputFormat = "output_format"
	ParamKeyThinkingType          = "type"
	ParamKeyThinkingTypeEnabled   = "enabled"
	ParamKeyBudgetTokens          = "budget_tokens"
)

// Gemini-specific parameter keys
const (
	ParamKeyGenerationConfig     = "generationConfig"
	ParamKeyGeminiMaxTokens      = "maxOutputTokens"
	ParamKeyGeminiTopP           = "topP"
	ParamKeyGeminiTopK           = "topK"
	ParamKeyGeminiStopSeqs       = "stopSequences"
	ParamKeyGeminiResponseMime   = "responseMimeType"
	ParamKeyGeminiResponseSchema = "responseSchema"
	GeminiResponseMimeJSON       = "application/json"
)

// Error format strings for type validation
const (
	ErrFmtTypeMismatch = "expected %s, got %s"
)

// LLM Provider names for structured output handling
const (
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
	ProviderGoogle    = "google"
	ProviderGemini    = "gemini"
	ProviderVertex    = "vertex"
	ProviderVLLM      = "vllm"
	ProviderAzure     = "azure"
	ProviderMistral   = "mistral"
	ProviderCohere    = "cohere"
)

// Response format types for structured outputs
const (
	ResponseFormatText       = "text"
	ResponseFormatJSONObject = "json_object"
	ResponseFormatJSONSchema = "json_schema"
	ResponseFormatEnum       = "enum"
)

// vLLM guided decoding backends
const (
	GuidedBackendXGrammar         = "xgrammar"
	GuidedBackendOutlines         = "outlines"
	GuidedBackendLMFormatEnforcer = "lm_format_enforcer"
	GuidedBackendAuto             = "auto"
)

// JSON Schema property keys
const (
	SchemaKeyType                 = "type"
	SchemaKeyProperties           = "properties"
	SchemaKeyRequired             = "required"
	SchemaKeyAdditionalProperties = "additionalProperties"
	SchemaKeyEnum                 = "enum"
	SchemaKeyItems                = "items"
	SchemaKeyPropertyOrdering     = "propertyOrdering"
	SchemaKeyDescription          = "description"
	SchemaKeySchema               = "schema"
	SchemaKeyStrict               = "strict"
	SchemaKeyFormat               = "format"
	SchemaKeyJSONSchema           = "json_schema"
)

// vLLM guided decoding parameter keys
const (
	GuidedKeyDecodingBackend   = "guided_decoding_backend"
	GuidedKeyJSON              = "guided_json"
	GuidedKeyRegex             = "guided_regex"
	GuidedKeyChoice            = "guided_choice"
	GuidedKeyGrammar           = "guided_grammar"
	GuidedKeyWhitespacePattern = "guided_whitespace_pattern"
)

// Storage error messages
const (
	ErrMsgCryptoRandFailure     = "cryptographic random number generator failure"
	ErrMsgPathTraversalDetected = "invalid template name: path traversal characters detected"
)

// v2.0 Prompt validation constraints (Agent Skills spec)
const (
	// PromptNameMaxLength is the maximum length for prompt names (slug format)
	PromptNameMaxLength = 64
	// PromptDescriptionMaxLength is the maximum length for prompt descriptions
	PromptDescriptionMaxLength = 1024
	// PromptSlugPattern is the regex pattern for valid prompt names/slugs
	// Must start with lowercase letter, followed by lowercase letters, digits, or hyphens
	PromptSlugPattern = `^[a-z][a-z0-9-]*$`
)

// Prompt field name constants for YAML/JSON serialization keys.
// Used in buildSerializeMap, GetStandardFields, GetPromptyFields, and extension key filtering.
const (
	// Agent Skills genspec standard fields
	PromptFieldName          = "name"
	PromptFieldDescription   = "description"
	PromptFieldLicense       = "license"
	PromptFieldCompatibility = "compatibility"
	PromptFieldAllowedTools  = "allowed_tools"
	PromptFieldMetadata      = "metadata"
	PromptFieldInputs        = "inputs"
	PromptFieldOutputs       = "outputs"
	PromptFieldSample        = "sample"

	// go-prompty extension fields
	PromptFieldType        = "type"
	PromptFieldExecution   = "execution"
	PromptFieldExtensions  = "extensions"
	PromptFieldSkills      = "skills"
	PromptFieldTools       = "tools"
	PromptFieldContext     = "context"
	PromptFieldConstraints = "constraints"
	PromptFieldMessages    = "messages"
)

// v2.0 Reference resolution constants
const (
	// RefMaxDepth is the maximum depth for nested prompt references
	RefMaxDepth = 10
	// RefVersionLatest is the default version when not specified
	RefVersionLatest = "latest"
)

// DeploymentStatus represents the lifecycle status of a template version.
type DeploymentStatus string

// Deployment status values - lifecycle of a template version.
const (
	// DeploymentStatusDraft is the initial status for new versions not yet ready for use.
	DeploymentStatusDraft DeploymentStatus = "draft"
	// DeploymentStatusActive indicates the version is ready for production use.
	DeploymentStatusActive DeploymentStatus = "active"
	// DeploymentStatusDeprecated marks the version as still functional but discouraged.
	DeploymentStatusDeprecated DeploymentStatus = "deprecated"
	// DeploymentStatusArchived is a terminal state - version is read-only and preserved for history.
	DeploymentStatusArchived DeploymentStatus = "archived"
)

// Reserved label names - commonly used deployment targets.
const (
	LabelProduction = "production"
	LabelStaging    = "staging"
	LabelCanary     = "canary"
)

// Label validation constraints.
const (
	// LabelMaxLength is the maximum length of a label name.
	LabelMaxLength = 64
	// LabelNamePattern is the regex pattern for valid label names.
	// Must start with lowercase letter, followed by lowercase letters, digits, underscores, or hyphens.
	LabelNamePattern = `^[a-z][a-z0-9_-]*$`
)

// Metadata keys for deployment audit trail.
const (
	// MetaKeyLabelPrefix prefixes label-related metadata entries.
	MetaKeyLabelPrefix = "label:"
	// MetaKeyStatusChangedAt records when status was last changed.
	MetaKeyStatusChangedAt = "status_changed_at"
	// MetaKeyStatusChangedBy records who changed the status.
	MetaKeyStatusChangedBy = "status_changed_by"
	// MetaKeyLabelAssignedAt records when a label was assigned.
	MetaKeyLabelAssignedAt = "label_assigned_at"
	// MetaKeyLabelAssignedBy records who assigned a label.
	MetaKeyLabelAssignedBy = "label_assigned_by"
	// MetaKeyRollbackFromVersion records the version that was rolled back to.
	MetaKeyRollbackFromVersion = "rollback_from_version"
	// MetaKeyClonedFrom records the template name that was cloned from.
	MetaKeyClonedFrom = "cloned_from"
	// MetaKeyClonedFromVersion records the version that was cloned from.
	MetaKeyClonedFromVersion = "cloned_from_version"
)

// PostgreSQL storage error messages
const (
	ErrMsgPostgresConnectionFailed      = "failed to connect to PostgreSQL"
	ErrMsgPostgresQueryFailed           = "PostgreSQL query failed"
	ErrMsgPostgresTransactionFailed     = "PostgreSQL transaction failed"
	ErrMsgPostgresScanFailed            = "failed to scan PostgreSQL result"
	ErrMsgPostgresMarshalFailed         = "failed to marshal data for PostgreSQL"
	ErrMsgPostgresUnmarshalFailed       = "failed to unmarshal PostgreSQL data"
	ErrMsgPostgresUnmarshalMetadata     = "failed to unmarshal PostgreSQL metadata"
	ErrMsgPostgresUnmarshalPromptConfig = "failed to unmarshal PostgreSQL prompt config"
	ErrMsgPostgresUnmarshalTags         = "failed to unmarshal PostgreSQL tags"
	ErrMsgPostgresMigrationFailed       = "PostgreSQL migration failed"
	ErrMsgPostgresCloseAfterError       = "PostgreSQL close failed after error"
	ErrMsgPostgresEmptyConnString       = "PostgreSQL connection string is empty"
	ErrMsgPostgresAlreadyClosed         = "PostgreSQL storage is already closed"
)

// Access control message formats
const (
	ErrFmtOperationAllowed    = "operation %s is allowed"
	ErrFmtOperationNotAllowed = "operation %s is not allowed"
)

// v2.1 Document type constants
// DocumentType identifies the kind of document (prompt, skill, agent).
type DocumentType string

const (
	// DocumentTypePrompt is a simple prompt template (no skills/tools/constraints)
	DocumentTypePrompt DocumentType = "prompt"
	// DocumentTypeSkill is a reusable skill document (default type)
	DocumentTypeSkill DocumentType = "skill"
	// DocumentTypeAgent is a full agent definition with skills, tools, and constraints
	DocumentTypeAgent DocumentType = "agent"
)

// SkillInjection defines how a skill is injected into an agent's context.
type SkillInjection string

const (
	// SkillInjectionNone does not inject skill content into the agent
	SkillInjectionNone SkillInjection = "none"
	// SkillInjectionSystemPrompt appends skill content to the system prompt
	SkillInjectionSystemPrompt SkillInjection = "system_prompt"
	// SkillInjectionUserContext injects skill content into user context
	SkillInjectionUserContext SkillInjection = "user_context"
)

// CatalogFormat defines the output format for catalog generation.
type CatalogFormat string

const (
	// CatalogFormatDefault uses markdown format
	CatalogFormatDefault CatalogFormat = ""
	// CatalogFormatDetailed includes full descriptions and parameters
	CatalogFormatDetailed CatalogFormat = "detailed"
	// CatalogFormatCompact uses minimal single-line format
	CatalogFormatCompact CatalogFormat = "compact"
	// CatalogFormatFunctionCalling generates JSON schema for function calling
	CatalogFormatFunctionCalling CatalogFormat = "function_calling"
)

// v2.1 Catalog resolver tag names
const (
	TagNameSkillsCatalog = "prompty.skills_catalog"
	TagNameToolsCatalog  = "prompty.tools_catalog"
)

// v2.1 Field constraints
const (
	MaxLicenseLength       = 100
	MaxCompatibilityLength = 500
)

// v2.1 Error message constants
const (
	ErrMsgNotAnAgent              = "document is not an agent type"
	ErrMsgSkillNotFound           = "skill not found"
	ErrMsgSkillRefEmpty           = "skill reference slug is empty"
	ErrMsgSkillRefAmbiguous       = "skill reference is ambiguous"
	ErrMsgNoExecutionConfig       = "execution configuration is required"
	ErrMsgNoProvider              = "provider is required in execution config"
	ErrMsgNoModel                 = "model is required in execution config"
	ErrMsgCompilationFailed       = "agent compilation failed"
	ErrMsgInvalidDocumentType     = "invalid document type"
	ErrMsgPromptNoSkillsAllowed   = "prompt type does not support skills"
	ErrMsgPromptNoToolsAllowed    = "prompt type does not support tools"
	ErrMsgPromptNoConstraints     = "prompt type does not support constraints"
	ErrMsgAgentMessagesInvalid    = "agent messages must include system or user role"
	ErrMsgSkillRefInvalidVersion  = "invalid skill reference version"
	ErrMsgCatalogGenerationFailed = "catalog generation failed"
	ErrMsgSkillNoSkillsAllowed    = "skill type does not support nested skills"
	ErrMsgInvalidSkillInjection   = "invalid skill injection mode"
	ErrMsgMCPServerNameEmpty      = "MCP server name is empty"
	ErrMsgMCPServerURLEmpty       = "MCP server URL is empty"
	ErrMsgMessageTemplateNoRole   = "message template requires a role"
	ErrMsgMessageTemplateNoBody   = "message template requires content"
	ErrMsgInlineSkillNoSlug       = "inline skill requires a slug"
	ErrMsgInlineSkillNoBody       = "inline skill requires a body"
	ErrMsgAgentNoBodyOrMessages   = "agent requires body or messages"
	ErrMsgUnsupportedMsgProvider  = "unsupported provider for message serialization"
	ErrMsgNoDocumentResolver      = "no document resolver configured"
)

// v2.1 Error code constants
const (
	ErrCodeAgent   = "PROMPTY_AGENT"
	ErrCodeCompile = "PROMPTY_COMPILE"
	ErrCodeCatalog = "PROMPTY_CATALOG"
)

// v2.1 Metadata keys for agent context
const (
	MetaKeyDocumentType  = "document_type"
	MetaKeySkillSlug     = "skill_slug"
	MetaKeyInjectionMode = "injection_mode"
	MetaKeySkillVersion  = "skill_version"
	MetaKeyMessageIndex  = "message_index"
	MetaKeyMessageRole   = "message_role"
	MetaKeyCompileStage  = "compile_stage"
)

// v2.1 Special template name for self-reference
const (
	TemplateNameSelf = "self"
)

// v2.1 Context keys used during agent compilation
const (
	ContextKeyInput       = "input"
	ContextKeyMeta        = "meta"
	ContextKeyContext     = "context"
	ContextKeyConstraints = "constraints"
	ContextKeySkills      = "skills"
	ContextKeyTools       = "tools"
	ContextKeySelfBody    = "_selfBody"
)

// v2.1 Skill injection markers
const (
	SkillInjectionMarkerStart = "<!-- SKILL_START:"
	SkillInjectionMarkerEnd   = "<!-- SKILL_END:"
	SkillInjectionMarkerClose = " -->"
)

// Versioning error messages
const (
	ErrMsgVersionGetFailed       = "failed to get version"
	ErrMsgVersionSaveRollback    = "failed to save rollback"
	ErrMsgVersionGetSource       = "failed to get source version"
	ErrMsgVersionTemplateExists  = "template already exists"
	ErrMsgVersionSaveClone       = "failed to save clone"
	ErrMsgVersionMinimumRequired = "must keep at least 1 version"
	ErrMsgVersionNoPrevious      = "no previous version for version 1"
)

// Catalog-specific error detail messages
const (
	ErrMsgCatalogFuncCallingSkills = "function_calling not supported for skills catalog"
	ErrMsgCatalogUnknownFormat     = "unknown catalog format"
)

// v2.5 Modality constants — execution intent signal
const (
	ModalityText               = "text"
	ModalityImage              = "image"
	ModalityAudioSpeech        = "audio_speech"
	ModalityAudioTranscription = "audio_transcription"
	ModalityMusic              = "music"
	ModalitySoundEffects       = "sound_effects"
	ModalityEmbedding          = "embedding"
)

// v2.5 Streaming method constants
const (
	StreamMethodSSE       = "sse"
	StreamMethodWebSocket = "websocket"
)

// v2.5 Image quality constants
const (
	ImageQualityStandard = "standard"
	ImageQualityHD       = "hd"
	ImageQualityLow      = "low"
	ImageQualityMedium   = "medium"
	ImageQualityHigh     = "high"
)

// v2.5 Image style constants
const (
	ImageStyleNatural = "natural"
	ImageStyleVivid   = "vivid"
)

// v2.5 Audio format constants
const (
	AudioFormatMP3  = "mp3"
	AudioFormatOpus = "opus"
	AudioFormatAAC  = "aac"
	AudioFormatFLAC = "flac"
	AudioFormatWAV  = "wav"
	AudioFormatPCM  = "pcm"
)

// v2.5 Embedding format constants (wire encoding)
const (
	EmbeddingFormatFloat  = "float"
	EmbeddingFormatBase64 = "base64"
)

// v2.7 Embedding input type constants
const (
	EmbeddingInputTypeSearchQuery        = "search_query"
	EmbeddingInputTypeSearchDocument     = "search_document"
	EmbeddingInputTypeClassification     = "classification"
	EmbeddingInputTypeClustering         = "clustering"
	EmbeddingInputTypeSemanticSimilarity = "semantic_similarity"
)

// v2.7 Embedding output dtype constants (quantization data type)
const (
	EmbeddingDtypeFloat32 = "float32"
	EmbeddingDtypeInt8    = "int8"
	EmbeddingDtypeUint8   = "uint8"
	EmbeddingDtypeBinary  = "binary"
	EmbeddingDtypeUbinary = "ubinary"
)

// v2.7 Embedding truncation strategy constants
const (
	EmbeddingTruncationNone  = "none"
	EmbeddingTruncationStart = "start"
	EmbeddingTruncationEnd   = "end"
)

// v2.7 Embedding pooling type constants (vLLM)
const (
	EmbeddingPoolingMean = "mean"
	EmbeddingPoolingCLS  = "cls"
	EmbeddingPoolingLast = "last"
)

// v2.7 Gemini task type mappings (UPPER_CASE values for Gemini API)
const (
	GeminiTaskRetrievalQuery     = "RETRIEVAL_QUERY"
	GeminiTaskRetrievalDocument  = "RETRIEVAL_DOCUMENT"
	GeminiTaskSemanticSimilarity = "SEMANTIC_SIMILARITY"
	GeminiTaskClassification     = "CLASSIFICATION"
	GeminiTaskClustering         = "CLUSTERING"
)

// v2.5 Media parameter map keys (for serialization)
const (
	ParamKeyModality        = "modality"
	ParamKeyImage           = "image"
	ParamKeyAudio           = "audio"
	ParamKeyEmbedding       = "embedding"
	ParamKeyStreaming       = "streaming"
	ParamKeyAsync           = "async"
	ParamKeyStream          = "stream"
	ParamKeyImageSize       = "size"
	ParamKeyImageQuality    = "quality"
	ParamKeyImageStyle      = "style"
	ParamKeyImageN          = "n"
	ParamKeyVoice           = "voice"
	ParamKeySpeed           = "speed"
	ParamKeyDimensions      = "dimensions"
	ParamKeyEncodingFormat  = "encoding_format"
	ParamKeyAspectRatio     = "aspect_ratio"
	ParamKeyNegativePrompt  = "negative_prompt"
	ParamKeyNumImages       = "num_images"
	ParamKeyGuidanceScale   = "guidance_scale"
	ParamKeySteps           = "steps"
	ParamKeyStrength        = "strength"
	ParamKeyVoiceID         = "voice_id"
	ParamKeyOutputFormat    = "output_format"
	ParamKeyDuration        = "duration"
	ParamKeyLanguage        = "language"
	ParamKeyPollInterval    = "poll_interval_seconds"
	ParamKeyPollTimeout     = "poll_timeout_seconds"
	ParamKeyStreamMethod    = "method"
	ParamKeyWidth           = "width"
	ParamKeyHeight          = "height"
	ParamKeyEnabled         = "enabled"
	ParamKeyResponseFormat  = "response_format"
	ParamKeyGeminiNumImages = "numberOfImages"

	// v2.7 Embedding parameter keys
	ParamKeyInputType   = "input_type"
	ParamKeyOutputDtype = "output_dtype"
	ParamKeyTruncation  = "truncation"
	ParamKeyNormalize   = "normalize"
	ParamKeyPoolingType = "pooling_type"

	// v2.7 Provider-specific embedding parameter keys
	ParamKeyOutputDimension      = "output_dimension"
	ParamKeyOutputDimensionality = "output_dimensionality"
	ParamKeyTaskType             = "task_type"
	ParamKeyEmbeddingTypes       = "embedding_types"
	ParamKeyTruncate             = "truncate"

	// v2.7 Cohere-specific parameter keys
	ParamKeyCohereTopP          = "p"
	ParamKeyCohereTopK          = "k"
	ParamKeyCohereStopSequences = "stop_sequences"
)

// v2.7 Cohere truncation UPPER_CASE constants
const (
	CohereTruncateNone  = "NONE"
	CohereTruncateStart = "START"
	CohereTruncateEnd   = "END"
)

// v2.5 Media validation limits
const (
	ImageMaxWidth          = 8192
	ImageMaxHeight         = 8192
	ImageMaxNumImages      = 10
	ImageMaxGuidanceScale  = 30.0
	ImageMaxSteps          = 200
	AudioMinSpeed          = 0.25
	AudioMaxSpeed          = 4.0
	AudioMaxDuration       = 600.0
	EmbeddingMaxDimensions = 65536
)
