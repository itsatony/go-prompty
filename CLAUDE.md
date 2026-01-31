# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

go-prompty is a dynamic LLM prompt templating system for Go with a plugin-based architecture. It provides XML-like tag syntax (`{~...~}`) with built-in conditionals, variable interpolation, a safe expression language with functions, and extensible resolver plugins.

**Core Principles:**
- Excellence. Always. — Production-ready from day one
- Content-Resistant Syntax — Works with any prompt content including code, XML, JSON
- Plugin-First Architecture — Extensible without core modifications
- Fail-Safe by Default — Predictable error handling with configurable strategies
- Isolated Execution — Cancellable, timeout-bounded, panic-recovered resolver execution
- Zero Storage Dependencies — Pure parsing, validation, and execution

## Build & Test Commands

```bash
# Run all tests with race detection
go test -v -race ./...

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run a single test
go test -v -race -run TestName ./...

# Run integration tests (requires Docker)
go test -v -race -tags=integration ./...

# Build
go build ./...

# Lint (use golangci-lint)
golangci-lint run
```

## Architecture

### Package Structure

```
github.com/itsatony/go-prompty/
├── prompty.go                    # Public API entry point
├── prompty.engine.go             # Engine type (public)
├── prompty.template.go           # Template type (public)
├── prompty.context.go            # Context type (public)
├── prompty.options.go            # Functional options (public)
├── prompty.resolver.go           # Resolver interface (public)
├── prompty.errors.go             # Public error types
├── prompty.constants.go          # Public constants
└── internal/
    ├── prompty.lexer.go          # Tokenizer
    ├── prompty.lexer.tokens.go   # Token definitions
    ├── prompty.parser.go         # Parser
    ├── prompty.parser.ast.go     # AST nodes
    ├── prompty.executor.go       # Execution engine
    ├── prompty.executor.isolation.go  # Goroutine isolation
    ├── prompty.executor.builtins.go   # Built-in tag resolvers
    ├── prompty.expr.go           # Expression evaluator
    ├── prompty.expr.operators.go # Comparison operators
    ├── prompty.funcs.go          # Function registry
    ├── prompty.funcs.strings.go  # String functions
    ├── prompty.funcs.math.go     # Math functions
    ├── prompty.funcs.collections.go   # Collection functions
    ├── prompty.funcs.types.go    # Type functions
    ├── prompty.funcs.util.go     # Utility functions
    └── prompty.resolver.registry.go   # Resolver registry
```

**File Naming**: `prompty.{type}.{module}.{variant}.go`
**Package Naming**:
- Root files use `package prompty` — public API
- Files in `internal/` use `package internal` — flat structure

### Data Flow

```
Source String → Lexer → Parser → AST → Executor → Output String
                                          ↓
                         Registry → Resolver (with isolation)
                                          ↓
                              Function Registry → Built-in/Plugin Functions
```

### Key Interfaces

**Resolver** — Plugins implement this to provide dynamic content:
```go
type Resolver interface {
    TagName() string
    Resolve(ctx context.Context, execCtx *prompty.Context, attrs Attributes) (string, error)
    Validate(attrs Attributes) error
}
```

**FuncProvider** — Optional interface for resolvers to register custom functions.

**Message** — Structured message for LLM APIs (extracted from template output):
```go
type Message struct {
    Role    string // "system", "user", "assistant", or "tool"
    Content string // Message content (trimmed)
    Cache   bool   // Cache hint for this message
}

// Template methods for message extraction:
messages, err := tmpl.ExecuteAndExtractMessages(ctx, data)
// Or extract from raw output:
output, _ := tmpl.Execute(ctx, data)
messages := prompty.ExtractMessagesFromOutput(output)
```

## Non-Negotiable Standards

| Rule | Requirement |
|------|-------------|
| No Magic Strings | **EVERY** string literal must be a constant—including error messages |
| Thread Safety | All exported types safe for concurrent access |
| Error Handling | Use `go-cuserr` with constant message strings |
| Static Typing | No `interface{}` without strong justification |
| IDs | Prefixed nanoIDs (e.g., `usr_6ByTSYmGzT2c`), never UUIDs/integers |
| Testing | Unit tests >80% coverage, `go test -race` |

## Error Handling Pattern

Always use `go-cuserr` with constant error messages:

```go
// Constants (REQUIRED)
const (
    ErrMsgMissingAttribute = "required attribute missing"
)

// Creating errors
if !attrs.Has(AttrID) {
    return "", cuserr.NewValidationError(AttrID, ErrMsgMissingAttribute)
}

// Wrapping errors
user, err := service.GetByID(ctx, userID)
if err != nil {
    return "", cuserr.NewInternalError(ErrMsgResolverFailed, err,
        cuserr.WithMetadata("resolver", TagName),
        cuserr.WithMetadata("user_id", userID),
    )
}

// Checking errors
if errors.Is(err, cuserr.ErrNotFound) { /* ... */ }
```

## Template Syntax Reference

**Delimiter**: `{~...~}` (tilde chosen for minimal collision with prompt content)

```
VARIABLE:
{~prompty.var name="user.name" default="Guest" /~}

INCLUDE (nested templates):
{~prompty.include template="header" /~}
{~prompty.include template="greeting" user="Alice" /~}
{~prompty.include template="item" with="currentItem" /~}
{~prompty.include template="footer" isolate="true" /~}

CONDITIONAL:
{~prompty.if eval="user.isAdmin"~}
  Admin content
{~prompty.elseif eval="user.isLoggedIn"~}
  User content
{~prompty.else~}
  Guest content
{~/prompty.if~}

LOOP:
{~prompty.for item="x" index="i" in="items" limit="100"~}
  {~prompty.var name="i" /~}: {~prompty.var name="x.name" /~}
{~/prompty.for~}

RAW (unparsed):
{~prompty.raw~}content not parsed{~/prompty.raw~}

COMMENT (removed):
{~prompty.comment~}removed from output{~/prompty.comment~}

MESSAGE (conversation message for LLM APIs):
{~prompty.message role="system"~}
You are a helpful assistant.
{~/prompty.message~}

{~prompty.message role="user"~}
{~prompty.var name="query" /~}
{~/prompty.message~}

Roles: "system", "user", "assistant", "tool"
Optional cache attribute: cache="true" (for cache hints)

CUSTOM PLUGIN TAG:
{~UserProfile id="123" fields="name,avatar" /~}

ESCAPE:
\{~ produces literal {~

EXPRESSIONS:
len(items) > 0 && contains(roles, "admin")
upper(trim(user.name))
```

### YAML Frontmatter (Inference Configuration)

Templates can include YAML frontmatter for inference configuration:

```yaml
---
name: my-template
model:
  api: chat
  name: gpt-4
  parameters:
    temperature: 0.7
inputs:
  query:
    type: string
    required: true
---
{~prompty.message role="user"~}
{~prompty.var name="query" /~}
{~/prompty.message~}
```

**Important**: When using prompty tags in YAML values, use single quotes:
```yaml
# Correct - single quotes preserve literal content
name: '{~prompty.env name="MODEL" /~}'

# Wrong - double quotes require escaping which breaks parsing
name: "{~prompty.env name=\"MODEL\" /~}"
```

### Nested Templates

Templates can reference other registered templates via `prompty.include`. Templates are registered with the engine:

```go
engine := prompty.MustNew()

// Register reusable templates
engine.MustRegisterTemplate("header", "Welcome to {~prompty.var name=\"siteName\" default=\"MyApp\" /~}")
engine.MustRegisterTemplate("footer", "Copyright 2024")

// Use in templates
result, _ := engine.Execute(ctx, `
{~prompty.include template="header" siteName="My App" /~}
Content here...
{~prompty.include template="footer" /~}
`, nil)
```

**Include Attributes:**
- `template` (required): Name of the registered template
- `with`: Context path - use value at path as root context
- `isolate`: "true" to not inherit parent context
- Other attributes become context variables in child template

**Template Name Rules:**
- Cannot be empty
- Cannot start with `prompty.` (reserved namespace)
- First-come-wins for duplicate registrations

## Execution Safety

All resolver and plugin function execution uses isolated goroutines with:
- Configurable timeout per resolver (default: 5s) and function (default: 1s)
- Overall execution timeout (default: 30s)
- Panic recovery
- Context cancellation propagation
- Resource limits: max loop iterations (10000), max output size (10MB), max depth (10)

## Structured Output Support

go-prompty supports structured outputs for all major LLM providers with provider-specific serialization.

### Provider-Specific Formats

| Provider | Configuration Field | Notes |
|----------|---------------------|-------|
| OpenAI/Azure | `response_format` | `json_schema` with `strict: true` |
| Anthropic | `output_format` | Alternative format for Claude API |
| Gemini | `response_format` | Supports `propertyOrdering` for Gemini 2.5+ |
| vLLM | `guided_decoding` | `json`, `regex`, `choice`, `grammar` constraints |

### Schema Requirements

All providers require `additionalProperties: false` for strict mode:
- Schemas are automatically augmented with `additionalProperties: false` when serializing
- Use `EnsureAdditionalPropertiesFalse()` for manual validation
- Use `ValidateForProvider()` to check provider compatibility

### Example Configurations

**OpenAI Style:**
```yaml
---
model:
  provider: openai
  name: gpt-4o
  response_format:
    type: json_schema
    json_schema:
      name: extracted_data
      strict: true
      schema:
        type: object
        properties:
          name: {type: string}
          email: {type: string}
        required: [name, email]
---
```

**Anthropic Style:**
```yaml
---
model:
  provider: anthropic
  name: claude-sonnet-4-5
  output_format:
    format:
      type: json_schema
      schema:
        type: object
        properties:
          result: {type: string}
        required: [result]
---
```

**vLLM Guided Decoding:**
```yaml
---
model:
  provider: vllm
  name: meta-llama/Llama-2-7b-hf
  guided_decoding:
    backend: xgrammar
    json:
      type: object
      properties:
        answer: {type: string}
---
```

**Enum Constraint:**
```yaml
---
model:
  response_format:
    type: enum
    enum:
      values: [positive, negative, neutral]
      description: Sentiment classification
---
```

### Provider Detection

The `GetEffectiveProvider()` method auto-detects the provider from:
1. Explicit `provider` field
2. Presence of `output_format` → Anthropic
3. Presence of `guided_decoding` → vLLM
4. Model name prefix (gpt-, claude-, gemini-)

### Provider Serialization

```go
config := tmpl.InferenceConfig()

// Get format for specific provider
openAIFormat, _ := config.ProviderFormat(ProviderOpenAI)
anthropicFormat, _ := config.ProviderFormat(ProviderAnthropic)
geminiFormat, _ := config.ProviderFormat(ProviderGemini)
vllmFormat, _ := config.ProviderFormat(ProviderVLLM)
```

## Deployment-Aware Versioning

Templates support deployment status and named labels for production workflows.

### Deployment Status

Templates have a lifecycle status that tracks their deployment readiness:

| Status | Description |
|--------|-------------|
| `draft` | Initial state for new versions not yet ready for use |
| `active` | Ready for production use (default for new templates) |
| `deprecated` | Still functional but discouraged |
| `archived` | Terminal state - read-only, preserved for history |

### Status Transitions

```
draft ────────────────→ active ────────────────→ deprecated
   │                       │                          │
   │                       │                          │
   └───────────┬───────────┴───────────┬──────────────┘
               │                       │
               ↓                       ↓
           archived ←─────────────────-┘
```

| From | Allowed To |
|------|------------|
| draft | active, archived |
| active | deprecated, archived |
| deprecated | active, archived |
| archived | (terminal - no transitions) |

### Labels

Labels are named pointers to specific versions (e.g., "production" → v42):
- `production` - The version currently running in production
- `staging` - The version being tested before production
- `canary` - The version for gradual rollout testing
- Custom labels (lowercase, alphanumeric with underscores/hyphens)

### Usage

```go
// Create engine with storage
engine := prompty.MustNewStorageEngine(prompty.StorageEngineConfig{
    Storage: prompty.NewMemoryStorage(),
})

// Save template (defaults to "active" status)
engine.Save(ctx, &prompty.StoredTemplate{
    Name:   "greeting",
    Source: "Hello {~prompty.var name=\"name\" /~}!",
})

// Label operations
engine.SetLabel(ctx, "greeting", "production", 1)    // Assign label
engine.ExecuteLabeled(ctx, "greeting", "production", data)  // Execute by label
engine.ExecuteProduction(ctx, "greeting", data)      // Convenience method
engine.PromoteToProduction(ctx, "greeting", 2)       // Promote new version

// Status operations
engine.SetStatus(ctx, "greeting", 1, prompty.DeploymentStatusDeprecated)
engine.ArchiveVersion(ctx, "greeting", 1)  // Convenience method
engine.GetActiveTemplates(ctx, nil)        // List active templates

// Check feature support (storage backend dependent)
if engine.SupportsLabels() {
    labels, _ := engine.ListLabels(ctx, "greeting")
}
```

### Storage Backend Support

All built-in storage backends (Memory, Filesystem, PostgreSQL) support labels and status.

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/itsatony/go-cuserr` | Error handling |
| `go.uber.org/zap` | Structured logging |
| `github.com/stretchr/testify` | Testing |

## Implementation Phases

- Phase 1 (v0.1.0): Lexer, Parser, Basic Executor, Registry, `prompty.var`, `prompty.raw`
- Phase 2 (v0.2.0): Expression evaluator, Conditionals (`prompty.if/elseif/else`)
- Phase 3 (v0.3.0): All error strategies, Comments, Validation API
- Phase 4 (v0.4.0): Loops (`prompty.for`)
- Phase 5 (v1.0.0): Switch/case, Custom functions, CLI tool
