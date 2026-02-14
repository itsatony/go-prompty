# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.5.0] - 2026-02-14

### Added

#### Media Generation Parameters
- **`Modality`** field on `ExecutionConfig`: Execution intent signal (`text`, `image`, `audio_speech`, `audio_transcription`, `music`, `sound_effects`, `embedding`)
- **`ImageConfig`**: Image generation parameters — width, height, size, quality, style, aspect_ratio, negative_prompt, num_images, guidance_scale, steps, strength
- **`AudioConfig`**: Audio generation (TTS/transcription) parameters — voice, voice_id, speed, output_format, duration, language
- **`EmbeddingConfig`**: Embedding generation parameters — dimensions, format
- **`AsyncConfig`**: Async execution parameters — enabled, poll_interval_seconds, poll_timeout_seconds

#### Execution Mode Configs
- **`StreamingConfig`** updated: Replaced `ChunkSize` with `Method` field (`sse`/`websocket`), added `Validate()`, `Clone()`, `ToMap()` methods
- All new configs include `Validate()`, `Clone()`, `ToMap()` methods

#### Provider Serialization
- **ToOpenAI**: image (size/quality/style/n), audio (voice/speed/response_format), embedding (dimensions/encoding_format), streaming (stream:true)
- **ToAnthropic**: streaming only (stream:true)
- **ToGemini**: image (aspectRatio/numberOfImages in generationConfig), streaming (stream:true)
- **ToVLLM**: streaming only (stream:true)

#### New Accessor Methods
- 12 new getter/checker methods: `GetModality`/`HasModality`, `GetImage`/`HasImage`, `GetAudio`/`HasAudio`, `GetEmbedding`/`HasEmbedding`, `GetStreaming`/`HasStreaming`, `GetAsync`/`HasAsync`

#### Constants & Validation
- ~80 new constants for modalities, stream methods, image quality/style, audio formats, embedding formats, param keys, validation limits
- 18 new error message constants for media validation
- `ParamKeyResponseFormat`, `ParamKeyGeminiNumImages` for provider serialization keys

### Changed
- `StreamingConfig.ChunkSize` replaced with `StreamingConfig.Method`
- `ExecutionConfig.Validate()` now delegates to nested media config validators
- `ExecutionConfig.Clone()` deep-copies all 6 new fields
- `ExecutionConfig.Merge()` supports replacement semantics for all media configs
- `ExecutionConfig.ToMap()` includes media configs as nested maps

## [2.1.0] - 2026-02-06

### Added

#### Agent Definition & Compilation
- **Document Types**: `prompt`, `skill` (default), `agent` with `DocumentType` enum
- **`SkillRef`**: Skill references with slug, version, inline body, injection mode, and execution overrides
- **`ToolsConfig`**: Tool definitions with `FunctionDef`, MCP servers, and provider-specific serialization
- **`ConstraintsConfig`**: Behavioral, safety, and operational constraints for agents
- **`MessageTemplate`**: Typed message templates (role + content) in YAML frontmatter
- **`CompileAgent()`**: Full agent compilation pipeline — resolves skills, generates catalogs, renders body and messages through `{~...~}` engine
- **`ActivateSkill()`**: Activates a skill on a compiled agent with injection modes (none, system_prompt, user_context)
- **`Compile()`**: Simple compilation for prompts/skills (body through engine with context)

#### Catalog Generation
- **`{~prompty.skills_catalog~}`**: Built-in resolver that generates skill catalog in configurable formats (default, detailed, compact)
- **`{~prompty.tools_catalog~}`**: Built-in resolver that generates tool catalog with function_calling JSON support
- **`GenerateSkillsCatalog()`**: Standalone catalog generation API
- **`GenerateToolsCatalog()`**: Standalone catalog generation API
- **Catalog Formats**: default (markdown), detailed, compact, function_calling (JSON schema)

#### ExecutionConfig Enhancements
- **`Merge()`**: 3-layer precedence merge (agent → skill → runtime) with shallow field-level semantics
- **`FunctionDef` Extensions**: `Returns` field, `ToOpenAITool()`, `ToAnthropicTool()` provider-specific serialization

#### Document Parsing & Resolution
- **`Parse()`**: Standalone v2.1 document parser (YAML frontmatter + body extraction)
- **`ParseFile()`**: Parse documents from filesystem
- **`DocumentResolver` Interface**: `ResolvePrompt`, `ResolveSkill`, `ResolveAgent` for document lookup
- **`MapDocumentResolver`**: In-memory resolver for testing
- **`StorageDocumentResolver`**: Storage-backed resolver wrapping `TemplateStorage`
- **`NoopDocumentResolver`**: Default resolver returning errors

#### Serialization & Import/Export
- **`Serialize()`**: YAML frontmatter + body serialization with configurable options
- **`ExportAgentSkill()`**: Export agent as portable skill (strips agent-specific fields)
- **`ExportFull()`**: Export with all fields including execution and skope
- **`Import()`**: Import from .md or .zip files
- **`ImportDirectory()`**: Import from zip archives with SKILL.md/AGENT.md/PROMPT.md
- **`ExportSkillDirectory()`**: Export as zip archive with document and resources

### Changed
- **`StoredTemplate`**: `InferenceConfig` field replaced by `PromptConfig *Prompt`
- **`Engine.Parse()`**: All frontmatter now parsed as `Prompt` (v1 InferenceConfig fallback removed)
- **`DetectSchemaProvider()`**: Now accepts `*ExecutionConfig` instead of `*InferenceConfig`
- **`ValidateOptional()`**: Enhanced detection of v2.1 documents (Execution/Skope/Type/Name signals)
- **PostgreSQL Storage**: Column renamed `inference_config` → `prompt_config` (migration 4)
- **Prompt Type**: Extended with `Type`, `Skills`, `Tools`, `Context`, `Constraints`, `Messages`, `Body`

### Removed
- **`InferenceConfig`**: Entire type and all associated methods deleted (replaced by `Prompt` + `ExecutionConfig`)
- **`Template.InferenceConfig()`**: Removed (use `Template.Prompt()` → `Prompt.Execution`)
- **`Template.HasInferenceConfig()`**: Removed (use `Template.HasPrompt()`)
- **`ModelConfig`**: Removed (fields absorbed into `ExecutionConfig`)
- **`examples/inference_config/`**: Removed obsolete example

### Technical Details
- Pre-release: no backward compatibility with v1 InferenceConfig
- Single syntax: all templating uses `{~...~}` engine exclusively
- Zero InferenceConfig references remaining in codebase
- 78.4% test coverage (root), 84.1% (internal), zero race conditions
- New files: 17 source files, comprehensive test coverage
- Postgres migration 4 handles column rename for existing databases

## [2.0.0] - 2026-02-04

### Added

#### Agent Skills Specification Support (v2.0)
- **New `Prompt` Type**: Comprehensive v2.0 prompt configuration compatible with [Agent Skills](https://agentskills.io) specification
  - `name`: Prompt identifier in slug format (max 64 chars, lowercase letters/digits/hyphens)
  - `description`: Prompt description (max 1024 chars)
  - `license`: License identifier (MIT, Apache-2.0, etc.)
  - `compatibility`: Compatible models/providers list
  - `allowed_tools`: Tools the prompt is designed to work with
  - `metadata`: Arbitrary key-value metadata
  - `inputs`: Input schema definitions with type validation
  - `outputs`: Output schema definitions
  - `sample`: Sample data for testing
- **Namespaced `execution` Config**: LLM execution parameters separated from prompt metadata
  - `provider`: LLM provider (openai, anthropic, google, vllm, azure)
  - `model`: Model name
  - `temperature`, `max_tokens`, `top_p`, `top_k`: Model parameters
  - `stop_sequences`: Stop sequences
  - `thinking`: Claude extended thinking configuration (enabled, budget_tokens)
  - `response_format`: Structured output format
  - `guided_decoding`: vLLM guided decoding configuration
  - `provider_options`: Provider-specific options
- **Namespaced `skope` Config**: Skope platform integration
  - `slug`: Platform-specific slug
  - `visibility`: Visibility level (public, private, team)
  - `version_number`: Version tracking
  - `projects`: Associated projects
  - `references`: Prompt references
  - Audit fields: `created_at`, `created_by`, `updated_at`, `updated_by`, `forked_from`

#### Prompt Reference Syntax (`prompty.ref`)
- **New `{~prompty.ref~}` Tag**: Reference and compose prompts from a registry
  - `slug` attribute (required): Prompt slug identifier
  - `version` attribute (optional): Specific version, defaults to "latest"
  - Supports `slug@version` syntax: `{~prompty.ref slug="greeting@v2" /~}`
- **Circular Reference Detection**: Prevents infinite loops with clear error messages
- **Depth Limiting**: Maximum 10 levels of nested references
- **PromptResolver Interface**: Implement to provide prompt lookup functionality
- **PromptResolverAdapter**: Convenience adapter for integrating custom resolvers

#### SKILL.md Import/Export
- **`ExportToSkillMD()`**: Export prompts in Agent Skills SKILL.md format (strips execution/skope)
- **`ImportFromSkillMD()`**: Parse SKILL.md files into Prompt + body
- **`SkillMD` Type**: Represents parsed SKILL.md with Prompt and body sections
- **`IsAgentSkillsCompatible()`**: Check if prompt uses only standard Agent Skills fields
- **`StripExtensions()`**: Remove go-prompty specific extensions for portability

#### Context Enhancements for References
- **`WithPromptResolver()`**: Set prompt resolver for reference resolution
- **`WithRefDepth()`**: Track reference resolution depth
- **`WithRefChain()`**: Track reference chain for circular detection
- **`PromptResolver()`**, **`RefDepth()`**, **`RefChain()`**: Accessor methods

#### New Types
- **`ExecutionConfig`**: LLM execution configuration with provider-specific methods
  - `ToOpenAI()`, `ToAnthropic()`, `ToGemini()`, `ToVLLM()`: Provider format conversion
  - `ProviderFormat(provider)`: Universal format conversion
  - `GetEffectiveProvider()`: Auto-detect provider from config
- **`SkopeConfig`**: Skope platform configuration
- **`ThinkingConfig`**: Claude extended thinking configuration
- **`PromptBodyResolver`**: Interface for prompt body lookup
- **`PromptResolverAdapter`**: Wraps PromptResolver to PromptBodyResolver

### Changed
- **Template Detection**: Templates with `execution` or `skope` config are treated as v2.0 Prompts
- **v1 Backward Compatibility**: Templates without v2-specific config still parse as v1 InferenceConfig
- **`Template.Prompt()`**: New method to access v2.0 Prompt configuration
- **`Template.HasPrompt()`**: Check if template has v2.0 Prompt

### Deprecated
- **`Template.InferenceConfig()`**: Use `Template.Prompt().Execution` instead
- **`Template.HasInferenceConfig()`**: Use `Template.HasPrompt()` instead

### Technical Details
- Agent Skills specification compatibility for prompt interoperability
- v2 detection based on presence of namespaced configs (execution/skope), not just name+description
- Full test coverage for v2.0 types and reference resolution
- Zero magic strings - all constants in prompty.constants.go
- Thread-safe prompt resolution with proper context propagation

## [1.6.0] - 2026-01-31

### Added

#### Enhanced Structured Output Support
- **Provider-Specific Formats**: Full alignment with OpenAI, Anthropic, Google Gemini, and vLLM APIs
- **JSONSchemaSpec Extensions**:
  - `AdditionalProperties`: Control extra properties in schema (all providers require `false` for strict mode)
  - `PropertyOrdering`: Specify property output order (Gemini 2.5+ feature)
- **EnumConstraint**: First-class enum/choice constraint support with values and description
- **Anthropic OutputFormat**: Native support for Anthropic's `output_format` (alternative to `response_format`)
- **vLLM GuidedDecoding**: Full guided decoding support
  - `json`: JSON schema constraints
  - `regex`: Regex pattern constraints
  - `choice`: Choice list constraints
  - `grammar`: Context-free grammar constraints
  - `backend`: Backend selection (xgrammar, outlines, lm_format_enforcer, auto)

#### Provider Serialization
- `ToOpenAI()`: Convert ResponseFormat to OpenAI API format
- `ToAnthropic()`: Convert to Anthropic output_format structure
- `ToGemini()`: Convert to Google Gemini/Vertex AI format with propertyOrdering support
- `ToVLLM()`: Convert GuidedDecoding to vLLM format
- `ProviderFormat()`: Universal method to get format for any supported provider

#### Schema Validation Utilities
- `ValidateJSONSchema()`: Validate JSON schema structure
- `ValidateForProvider()`: Validate schema compatibility with specific provider
- `ValidateEnumConstraint()`: Validate enum constraint configuration
- `ValidateGuidedDecoding()`: Validate vLLM guided decoding configuration
- `EnsureAdditionalPropertiesFalse()`: Recursively add additionalProperties: false to schemas
- `ExtractRequiredFields()`: Extract property names for required array generation

#### Provider Detection
- `GetEffectiveProvider()`: Auto-detect provider from configuration
  - Explicit `provider` field takes precedence
  - Presence of `output_format` → Anthropic
  - Presence of `guided_decoding` → vLLM
  - Model name prefix inference (gpt-, claude-, gemini-)

#### New Constants
- Provider names: `ProviderOpenAI`, `ProviderAnthropic`, `ProviderGoogle`, `ProviderGemini`, `ProviderVertex`, `ProviderVLLM`, `ProviderAzure`
- Response format types: `ResponseFormatText`, `ResponseFormatJSONObject`, `ResponseFormatJSONSchema`, `ResponseFormatEnum`
- Guided decoding backends: `GuidedBackendXGrammar`, `GuidedBackendOutlines`, `GuidedBackendLMFormatEnforcer`, `GuidedBackendAuto`
- Schema property keys: `SchemaKeyType`, `SchemaKeyProperties`, `SchemaKeyRequired`, `SchemaKeyAdditionalProperties`, `SchemaKeyEnum`, `SchemaKeyItems`, `SchemaKeyPropertyOrdering`

### Technical Details
- All provider serializers automatically add `additionalProperties: false` for strict mode compliance
- Deep copy of schemas during serialization (original schema not modified)
- Comprehensive test coverage for all provider formats
- Zero magic strings - all constants in prompty.constants.go

## [1.5.1] - 2026-01-29

### Fixed
- Replaced magic strings in `prompty.versioning.go` with constants (`MetaKeyRollbackFromVersion`, `MetaKeyClonedFrom`, `MetaKeyClonedFromVersion`)
- Added missing tests for convenience methods: `PromoteToStaging`, `ExecuteStaging`, `GetActiveTemplates`, `ArchiveVersion`, `DeprecateVersion`, `ActivateVersion`

### Technical Details
- Zero magic strings - all metadata keys are now constants in `prompty.constants.go`
- 100% test coverage for deployment convenience methods

## [1.5.0] - 2026-01-29

### Added

#### Deployment-Aware Versioning
- **Labels**: Named pointers to specific template versions for deployment workflows
  - Reserved labels: `production`, `staging`, `canary`
  - Custom labels with validation (lowercase, alphanumeric with hyphens/underscores)
  - Label assignment tracking (AssignedAt, AssignedBy metadata)
- **Deployment Status**: Template version lifecycle management
  - Status values: `draft` → `active` → `deprecated` → `archived`
  - Status transition validation (archived is terminal state)
  - Status filtering in queries
- **Convenience Methods** for common workflows:
  - `ExecuteLabeled()` - Execute template by label
  - `ExecuteProduction()` - Execute the "production" labeled version
  - `PromoteToProduction()` - Move production label to specific version
  - `GetProduction()` - Get template with production label
  - `GetByLabel()` - Get template by any label
  - `ListByStatus()` - Query templates by deployment status
- **Storage Interface Extensions**:
  - `LabelStorage` interface for label operations
  - `StatusStorage` interface for status management
  - `ExtendedTemplateStorage` combining all storage interfaces
- **Version History Enhancement**:
  - Labels displayed in version history
  - Production version tracking
  - Status tracking per version

#### Storage Updates
- Memory, Filesystem, and PostgreSQL storage all implement `ExtendedTemplateStorage`
- PostgreSQL Migration V2: Added `status` column and `prompty_template_labels` table
- PostgreSQL Migration V3: Added trigger for automatic label cleanup on template deletion
- Label cleanup on template deletion (prevents orphaned labels)
- AssignedAt and AssignedBy tracking for label audit trails

### Changed
- `RollbackToVersion()` now creates rolled-back version with `draft` status (requires review before activation)
- `CloneVersion()` now creates cloned template with `draft` status (requires customization before activation)
- Default status for new templates is `active` (not `draft`) to maintain backward compatibility

### Technical Details
- All storage backends implement LabelStorage and StatusStorage interfaces
- Thread-safe label operations with proper locking
- Label validation with regex pattern `^[a-z][a-z0-9_-]*$`
- Maximum label length: 64 characters
- Comprehensive E2E tests for all storage backends
- Zero magic strings - all constants in prompty.constants.go

## [1.4.0] - Unreleased

### Added

#### YAML Frontmatter Configuration (replaces JSON config blocks)
- Standard YAML frontmatter with `---` delimiters for inference configuration
- Industry-standard format (compatible with Jekyll, Hugo, Microsoft Prompty)
- Native comment support in configuration
- Cleaner multiline values with YAML block scalars
- Environment variable substitution using `{~prompty.env~}` tags in YAML values

#### Conversation Message Support
- New `prompty.message` tag for LLM conversation messages
- Supported roles: `system`, `user`, `assistant`, `tool`
- Optional `cache` attribute for message-level caching hints
- Full template support inside message blocks (conditionals, loops, variables)
- `ExecuteAndExtractMessages()` method for structured message extraction
- Dynamic conversation history via loops around message tags

#### Extended Model Configuration
- `response_format` - Structured output enforcement with JSON Schema
- `tools` - Function/tool calling definitions
- `tool_choice` - Tool selection strategy ("auto", "none", "required")
- `streaming` - Streaming configuration with enabled flag
- `context_window` - Token budget hints

#### New Top-Level Configuration
- `retry` - Retry behavior with max_attempts and backoff strategy
- `cache` - Caching configuration with system_prompt and TTL settings

### Changed
- **BREAKING**: JSON `{~prompty.config~}` blocks deprecated in favor of YAML `---` frontmatter
- Message role validation is case-insensitive (accepts "USER", "System", etc.)
- Message roles are normalized to lowercase in storage for consistency

### Security
- Added null byte sanitization to prevent marker injection attacks in message content
- Added YAML frontmatter size limit (64KB) to prevent DoS attacks
- Message extraction gracefully handles malformed markers

## [1.3.0] - 2025-01-05

### Added

#### Template Inheritance System
- Base template support with `prompty.extends` tag for template inheritance
- Named block definitions with `prompty.block` for overridable sections
- `prompty.parent` tag for including parent block content in overrides
- Multi-level inheritance support (A extends B extends C)
- Circular inheritance detection and prevention
- Block-level override validation

#### Date/Time Expression Functions
- `now()` - Returns current timestamp
- `formatDate(time, layout)` - Format time using Go layout strings
- `parseDate(string, layout)` - Parse string to time
- `addDays(t, n)`, `addHours(t, n)`, `addMinutes(t, n)` - Time arithmetic
- `diffDays(t1, t2)` - Calculate days between times
- `year(t)`, `month(t)`, `day(t)` - Extract date components
- `weekday(t)` - Get day name
- `isAfter(t1, t2)`, `isBefore(t1, t2)` - Time comparisons
- Common format constants: `DateFormatISO`, `DateFormatUS`, `DateFormatEU`, etc.

#### CLI Enhancements
- New `prompty lint` command for template quality checks:
  - `VAR001`: Variable name uses non-standard casing
  - `VAR002`: Variable without default might be missing
  - `TAG001`: Unknown or unregistered tag
  - `LOOP001`: Loop without limit attribute
  - `LOOP002`: Deeply nested loops (> 2 levels)
  - `EXPR001`: Complex expression (> 3 operators)
  - `INC001`: Include references non-existent template
  - Supports `--rules` and `--ignore` flags for rule selection
  - Supports `--strict` mode (warnings fail validation)
  - JSON and text output formats
- New `prompty debug` command for dry-run analysis:
  - Shows all variables referenced with existence check
  - Shows all resolvers invoked
  - Shows template includes and their resolution
  - Highlights missing variables with suggestions
  - Shows unused data fields
  - Supports `--trace` and `--verbose` modes
  - JSON and text output formats

#### PostgreSQL Storage Driver
- Built-in PostgreSQL storage driver with full `TemplateStorage` interface implementation
- Automatic schema migrations with version tracking
- Connection pooling with configurable limits (MaxOpenConns, MaxIdleConns, ConnMaxLifetime)
- JSONB storage for metadata, inference config, and tags with GIN indexes
- SERIALIZABLE transaction isolation for safe version management
- Context-aware query timeouts
- Driver registration via `OpenStorage("postgres", connectionString)` or `NewPostgresStorage(config)`

#### Examples
- `examples/error_handling/main.go` - Demonstrates all 5 error strategies
- `examples/debugging/main.go` - Demonstrates dry-run analysis usage
- `examples/inheritance/main.go` - Demonstrates template inheritance

### Technical Details

- 13 new date/time functions with comprehensive tests
- Full test coverage for CLI lint and debug commands
- Template inheritance with depth limiting (max 10 levels)
- Production-ready PostgreSQL storage with migrations and connection pooling
- All string literals use constants (no magic strings)
- Thread-safe for concurrent access
- Backward compatible with existing templates

## [1.2.0] - 2024-12-28

### Added

#### Inference Configuration (Config Blocks)
- JSON-based config blocks (`{~prompty.config~}...{~/prompty.config~}`) for self-describing templates
- `InferenceConfig` type with typed model configuration, parameters, input/output schemas, and sample data
- Model configuration fields: `api`, `provider`, `name`, `parameters` (temperature, max_tokens, top_p, etc.)
- Input/output schema definitions with type validation
- Sample data for testing and documentation
- `prompty.env` resolver for environment variable access in templates and config blocks
- Automatic `InferenceConfig` extraction and storage persistence
- Template accessor methods: `InferenceConfig()`, `HasInferenceConfig()`, `TemplateBody()`
- JSON serialization: `InferenceConfig.JSON()`, `InferenceConfig.JSONPretty()`
- Input validation: `InferenceConfig.ValidateInputs(data)`
- Parameter helpers: `GetTemperature()`, `GetMaxTokens()`, `GetTopP()`, `ToMap()`, etc.

#### Environment Variable Resolver
- New `prompty.env` tag for environment variable interpolation
- Attributes: `name` (required), `default` (optional), `required` (optional)
- Works in both templates and config blocks

### Changed

- `StoredTemplate` now includes `InferenceConfig` field for storage persistence
- `StorageEngine.Save()` automatically extracts `InferenceConfig` from templates
- Memory and filesystem storage implementations preserve `InferenceConfig`

### Documentation

- New `docs/INFERENCE_CONFIG.md` with comprehensive usage guide
- New `examples/inference_config/main.go` demonstrating all features

### Technical Details

- 19 new E2E tests for inference configuration
- All string literals use constants (no magic strings)
- Thread-safe for concurrent access
- Compatible with existing templates (backward compatible)

## [1.1.0] - 2024-12-19

### Added

- Comprehensive unit tests for parser internals (conditionals, loops, switch/case, comments)
- Comprehensive unit tests for executor internals (conditional execution, for loops, switch statements)
- Complete error constructor test coverage (100% on all error creation functions)
- Production-focused README with architecture overview, best practices, and troubleshooting guide

### Changed

- Improved error handling with metadata support for all internal error types
- Replaced magic format strings with structured error constructors using metadata
- Enhanced test coverage from 64.9% to 88.3% across library code

### Removed

- Unused Token helper methods (IsEOF, IsText, IsOpenTag, IsSelfClose, IsBlockClose, IsCloseTag, NewToken)
- Obsolete format string constants (ErrMsgVariableNotFoundFmt, ErrMsgTemplateNotFoundFmt, ErrFmtTypeComparison)

### Technical Details

- 88%+ test coverage with race detection
- Zero magic string violations in error handling
- All tests pass with `-race` flag
- Clean golangci-lint output

## [1.0.0] - 2024-12-13

### Added

#### Phase 1: Core Templating Engine
- Lexer with configurable delimiters (`{~...~}`)
- Parser producing AST with support for self-closing and block tags
- Executor with resolver dispatch and raw block handling
- Registry with first-come-wins semantics for custom resolvers
- Built-in resolvers: `prompty.var` (variable interpolation), `prompty.raw` (literal blocks)
- Thread-safe Context with dot-notation path resolution and parent-child scoping
- Functional options API (`WithDelimiters`, `WithMaxDepth`, `WithLogger`)
- Error handling with position tracking using go-cuserr

#### Phase 2: Expression Evaluator and Conditionals
- Safe expression language with comparison operators (`==`, `!=`, `<`, `>`, `<=`, `>=`)
- Logical operators (`&&`, `||`, `!`)
- Built-in functions (`len`, `upper`, `lower`, `trim`, `contains`, `hasPrefix`, `hasSuffix`, etc.)
- Conditional tags: `prompty.if`, `prompty.elseif`, `prompty.else`
- Truthiness evaluation for various types

#### Phase 2.5: Nested Templates
- Template registration API (`RegisterTemplate`, `MustRegisterTemplate`)
- `prompty.include` resolver for nested template execution
- Context inheritance and override capabilities
- `with` attribute for context path selection
- `isolate` attribute for isolated context execution
- Maximum depth protection against infinite recursion

#### Phase 3: Error Strategies, Comments, and Validation
- Five error strategies: `throw`, `default`, `remove`, `keepraw`, `log`
- Per-tag error strategy override via `onerror` attribute
- Comment blocks (`prompty.comment`) removed from output
- Validation API for template syntax checking
- ValidationResult with severity levels (Error, Warning, Info)

#### Phase 4: Loops
- `prompty.for` loop construct for iterating over collections
- Support for slices, arrays, and maps
- `item` and `index` loop variables
- `limit` attribute for iteration bounds
- Nested loop support
- Context isolation per iteration

#### Phase 5: Switch/Case, Custom Functions, and CLI
- Switch/case statements: `prompty.switch`, `prompty.case`, `prompty.casedefault`
- First-match-wins semantics (no fall-through)
- Case matching by value or expression evaluation
- Custom function registration API (`RegisterFunc`, `MustRegisterFunc`)
- CLI tool (`prompty`) with four commands:
  - `render`: Execute templates with JSON data
  - `validate`: Check template syntax and structure
  - `version`: Display version and build information
  - `help`: Show command usage

### Technical Details
- 82%+ test coverage with race detection
- All string literals use constants (no magic strings)
- Thread-safe for concurrent access
- Configurable timeouts and resource limits
- Structured logging with zap

## [0.5.0] - 2024-12-13

### Added
- Switch/case statements (`prompty.switch`, `prompty.case`, `prompty.casedefault`)
- Custom function registration API
- CLI tool with render, validate, version, help commands

## [0.4.0] - 2024-12-12

### Added
- For loops (`prompty.for`) with item, index, in, and limit attributes
- Map iteration support
- Nested loop support

## [0.3.0] - 2024-12-12

### Added
- Error strategies (throw, default, remove, keepraw, log)
- Comment blocks (`prompty.comment`)
- Validation API with severity levels

## [0.2.0] - 2024-12-12

### Added
- Expression evaluator with comparison and logical operators
- Built-in functions for strings, collections, and types
- Conditional tags (if/elseif/else)
- Nested template inclusion (`prompty.include`)

## [0.1.0] - 2024-12-11

### Added
- Initial release with core templating engine
- Lexer, parser, and executor
- Variable interpolation (`prompty.var`)
- Raw blocks (`prompty.raw`)
- Custom resolver registration
