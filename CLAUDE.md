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
github.com/itsatony/go-prompty/v2/
├── prompty.go                    # Public API entry point
├── prompty.engine.go             # Engine type (public)
├── prompty.template.go           # Template type (public)
├── prompty.context.go            # Context type (public)
├── prompty.options.go            # Functional options (public)
├── prompty.resolver.go           # Resolver interface (public)
├── prompty.errors.go             # Public error types
├── prompty.constants.go          # Public constants
├── prompty.prompt.go             # v2.1: Prompt type with validation
├── prompty.execution.go          # v2.1: ExecutionConfig with provider serialization
├── prompty.skope.go              # v2.1: SkopeConfig (platform integration)
├── prompty.types.agent.go        # v2.1: SkillRef, ToolsConfig, ConstraintsConfig
├── prompty.types.shared.go       # v2.1: ResponseFormat, GuidedDecoding, InputDef
├── prompty.types.tools.go        # v2.1: FunctionDef, ModelParameters
├── prompty.types.media.go        # v2.5: ImageConfig, AudioConfig, EmbeddingConfig, AsyncConfig
├── prompty.compile.go            # v2.1: CompileAgent, ActivateSkill, Compile, AgentDryRun
├── prompty.catalog.go            # v2.1: Catalog generation (skills, tools)
├── prompty.document.resolver.go  # v2.1: DocumentResolver interface + impls
├── prompty.parse.go              # v2.1: Standalone Parse/ParseFile
├── prompty.runner.go             # v2.1: TemplateRunner interface (Engine + StorageEngine)
├── prompty.agent.executor.go     # v2.1: AgentExecutor convenience wrapper
├── prompty.serialize.go          # v2.1: Serialization with options
├── prompty.import.go             # v2.1: Import from .md/.zip
├── prompty.export.go             # v2.1: Export to .md/.zip
├── prompty.skillmd.go            # v2.1: SKILL.md import/export
├── prompty.storage.go            # Storage interfaces
├── prompty.storage.memory.go     # In-memory storage
├── prompty.storage.postgres.go   # PostgreSQL storage
├── prompty.versioning.go         # Template versioning
└── internal/
    ├── prompty.lexer.go          # Tokenizer
    ├── prompty.lexer.tokens.go   # Token definitions
    ├── prompty.parser.go         # Parser
    ├── prompty.parser.ast.go     # AST nodes
    ├── prompty.executor.go       # Execution engine
    ├── prompty.executor.isolation.go  # Goroutine isolation
    ├── prompty.executor.builtins.go   # Built-in tag resolvers
    ├── prompty.executor.builtins.catalog.go  # v2.1: Catalog resolvers
    ├── prompty.executor.builtins.ref.go      # v2.1: Prompt ref resolver
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

**DocumentResolver** — v2.1 interface for resolving prompts, skills, and agents by slug:
```go
type DocumentResolver interface {
    ResolvePrompt(ctx context.Context, slug string) (*Prompt, error)
    ResolveSkill(ctx context.Context, ref string) (*Prompt, error)
    ResolveAgent(ctx context.Context, slug string) (*Prompt, error)
}

// Implementations: NoopDocumentResolver, MapDocumentResolver, StorageDocumentResolver
```

**TemplateRunner** — Common interface for resolver management shared by Engine and StorageEngine:
```go
type TemplateRunner interface {
    RegisterResolver(r Resolver) error
    HasResolver(tagName string) bool
    ListResolvers() []string
    ResolverCount() int
}

// Both Engine and StorageEngine satisfy this interface.
```

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

### YAML Frontmatter (v2.1 Prompt Configuration)

All YAML frontmatter is parsed as `Prompt` configuration:

```yaml
---
name: my-prompt
description: A v2.1 prompt
type: skill
execution:
  provider: openai
  model: gpt-4
  temperature: 0.7
  max_tokens: 1000
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
model: '{~prompty.env name="MODEL" /~}'

# Wrong - double quotes require escaping which breaks parsing
model: "{~prompty.env name=\"MODEL\" /~}"
```

### Document Types

| Type | Description |
|------|-------------|
| `prompt` | Simple prompt template, no skills/tools/constraints |
| `skill` | Default type. Reusable capability, no sub-skills |
| `agent` | Full agent with skills, tools, constraints, messages |

### Agent Definition (v2.1)

```yaml
---
name: research-agent
description: AI research assistant
type: agent
execution:
  provider: anthropic
  model: claude-sonnet-4-5
  temperature: 0.3
skills:
  - slug: web-search
    injection: system_prompt
  - slug: summarizer
    injection: user_context
tools:
  functions:
    - name: search_web
      description: Search the web
      parameters:
        type: object
        properties:
          query: {type: string}
        required: [query]
context:
  company: Acme Corp
constraints:
  behavioral:
    - Always cite sources
messages:
  - role: system
    content: |
      You are a research assistant for {~prompty.var name="context.company" /~}.
      {~prompty.skills_catalog format="detailed" /~}
  - role: user
    content: '{~prompty.var name="input.query" /~}'
---
{~prompty.include template="self" /~}
```

**Key v2.1 Types:**
- `Prompt`: Full prompt configuration with document type, skills, tools, context, constraints, messages
- `ExecutionConfig`: LLM execution parameters with `Merge()` for 3-layer precedence. Extended in v2.3 with `MinP`, `RepetitionPenalty`, `Seed`, `Logprobs`, `StopTokenIDs`, `LogitBias`
- `SkopeConfig`: Platform integration fields
- `SkillRef`: Skill reference with injection mode and execution overrides
- `ToolsConfig`: Tool definitions with function defs and MCP servers
- `CompiledPrompt`: Result of `CompileAgent()` — messages, execution config, tools, constraints

**Agent Compilation:**
```go
prompt, _ := prompty.Parse(agentYAML)
compiled, _ := prompt.CompileAgent(ctx, input, &prompty.CompileOptions{
    Resolver: myDocumentResolver,
})
// compiled.Messages, compiled.Execution, compiled.Tools, compiled.Constraints
```

**Catalog Resolvers:**
```
{~prompty.skills_catalog format="detailed" /~}
{~prompty.tools_catalog format="function_calling" /~}
```

**Reference Tag for Prompt Composition:**
```
{~prompty.ref slug="my-prompt" /~}
{~prompty.ref slug="my-prompt" version="v2" /~}
{~prompty.ref slug="my-prompt@v2" /~}
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
| Anthropic | `response_format` | Same field, serialized to Anthropic format |
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
name: entity-extractor
execution:
  provider: openai
  model: gpt-4o
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
name: classifier
execution:
  provider: anthropic
  model: claude-sonnet-4-5
  response_format:
    type: json_schema
    json_schema:
      name: result
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
name: qa-bot
execution:
  provider: vllm
  model: meta-llama/Llama-2-7b-hf
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
name: sentiment-classifier
execution:
  response_format:
    type: enum
    enum:
      values: [positive, negative, neutral]
      description: Sentiment classification
---
```

**Extended Inference Parameters (v2.3):**
```yaml
---
name: vllm-sampler
execution:
  provider: vllm
  model: meta-llama/Llama-2-7b-hf
  temperature: 0.8
  min_p: 0.1
  repetition_penalty: 1.2
  seed: 42
  logprobs: 5
  stop_token_ids: [50256, 50257]
  logit_bias:
    "100": 5.0
    "200": -10.0
---
```

| Parameter | Type | Range | Providers |
|-----------|------|-------|-----------|
| `min_p` | `*float64` | [0.0, 1.0] | vLLM |
| `repetition_penalty` | `*float64` | > 0.0 | vLLM |
| `seed` | `*int` | any int | OpenAI, Anthropic, vLLM |
| `logprobs` | `*int` | [0, 20] | OpenAI (dual-field), vLLM |
| `stop_token_ids` | `[]int` | each >= 0 | vLLM |
| `logit_bias` | `map[string]float64` | values [-100, 100] | OpenAI, vLLM |

### Provider Detection

The `GetEffectiveProvider()` method on `ExecutionConfig` auto-detects the provider from:
1. Explicit `provider` field
2. Presence of `thinking` config or claude model name → Anthropic
3. Presence of `guided_decoding`, `min_p`, `repetition_penalty`, or `stop_token_ids` → vLLM
4. Model name prefix (gpt-, claude-, gemini-)

### Provider Serialization

```go
prompt := tmpl.Prompt()
exec := prompt.Execution

// Get format for specific provider
openAIFormat, _ := exec.ProviderFormat(ProviderOpenAI)
anthropicFormat, _ := exec.ProviderFormat(ProviderAnthropic)
geminiFormat, _ := exec.ProviderFormat(ProviderGemini)
vllmFormat, _ := exec.ProviderFormat(ProviderVLLM)
```

### Media Generation Parameters (v2.5)

ExecutionConfig supports multimodal AI generation via nested config structs:

| Config | Fields | Providers |
|--------|--------|-----------|
| `Modality` | `text`, `image`, `audio_speech`, `audio_transcription`, `music`, `sound_effects`, `embedding` | All (execution intent signal) |
| `Image` | `width`, `height`, `size`, `quality`, `style`, `aspect_ratio`, `negative_prompt`, `num_images`, `guidance_scale`, `steps`, `strength` | OpenAI (size/quality/style/n), Gemini (aspectRatio/numberOfImages) |
| `Audio` | `voice`, `voice_id`, `speed`, `output_format`, `duration`, `language` | OpenAI (voice/speed/response_format) |
| `Embedding` | `dimensions`, `format` | OpenAI (dimensions/encoding_format) |
| `Streaming` | `enabled`, `method` (`sse`/`websocket`) | All (stream: true) |
| `Async` | `enabled`, `poll_interval_seconds`, `poll_timeout_seconds` | Application-level |

**YAML Example (Image Generation):**
```yaml
---
name: image-gen
execution:
  modality: image
  provider: openai
  model: dall-e-3
  image:
    size: "1024x1024"
    quality: hd
    style: vivid
    num_images: 2
---
```

**YAML Example (Audio TTS):**
```yaml
---
name: tts-narrator
execution:
  modality: audio_speech
  provider: openai
  model: tts-1-hd
  audio:
    voice: alloy
    speed: 1.25
    output_format: mp3
---
```

**Provider serialization rules for media params:**
- **ToOpenAI**: image (size/quality/style/n), audio (voice/speed/response_format), embedding (dimensions/encoding_format), streaming (stream:true)
- **ToAnthropic**: streaming only (stream:true). No media generation params
- **ToGemini**: image (aspectRatio/numberOfImages in generationConfig), streaming (stream:true)
- **ToVLLM**: streaming only (stream:true). No media params (text inference only)
- **GetEffectiveProvider**: Media params do NOT hint provider (they span multiple providers)

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
