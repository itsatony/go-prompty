# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
