# go-prompty

**Dynamic LLM prompt templating for Go** - Build complex, maintainable AI prompts with a powerful templating engine designed for production use.

[![Go Reference](https://pkg.go.dev/badge/github.com/itsatony/go-prompty.svg)](https://pkg.go.dev/github.com/itsatony/go-prompty)
[![Go Report Card](https://goreportcard.com/badge/github.com/itsatony/go-prompty)](https://goreportcard.com/report/github.com/itsatony/go-prompty)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Test Coverage](https://img.shields.io/badge/coverage-88%25-brightgreen.svg)](https://github.com/itsatony/go-prompty)

```
{~prompty.if eval="user.tier == 'enterprise'"~}
You are assisting {~prompty.var name="user.company" /~}, an enterprise customer.
{~prompty.else~}
You are assisting {~prompty.var name="user.name" default="a user" /~}.
{~/prompty.if~}
```

## Why go-prompty?

| Challenge | Solution |
|-----------|----------|
| **Prompt sprawl** | Organize prompts as composable, versioned templates |
| **Content conflicts** | `{~...~}` delimiters avoid clashes with code, JSON, XML |
| **Dynamic content** | Variables, conditionals, loops, and expressions |
| **Reusability** | Nested templates with `prompty.include` |
| **Type safety** | Compile-time template validation |
| **Extensibility** | Plugin architecture for custom tags and functions |

## Installation

```bash
go get github.com/itsatony/go-prompty
```

**CLI tool:**
```bash
go install github.com/itsatony/go-prompty/cmd/prompty@latest
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/itsatony/go-prompty"
)

func main() {
    engine := prompty.MustNew()

    prompt := `You are helping {~prompty.var name="user.name" /~}.

{~prompty.if eval="len(context) > 0"~}
Previous context:
{~prompty.for item="msg" in="context"~}
- {~prompty.var name="msg" /~}
{~/prompty.for~}
{~/prompty.if~}

Please respond in {~prompty.var name="language" default="English" /~}.`

    result, _ := engine.Execute(context.Background(), prompt, map[string]any{
        "user":     map[string]any{"name": "Alice", "tier": "pro"},
        "context":  []string{"User asked about pricing", "Interested in API access"},
        "language": "English",
    })

    fmt.Println(result)
}
```

---

## Table of Contents

- [Core Concepts](#core-concepts)
- [Template Syntax](#template-syntax)
- [Built-in Tags](#built-in-tags)
- [Expression Language](#expression-language)
- [Custom Resolvers](#custom-resolvers)
- [Custom Functions](#custom-functions)
- [Production Patterns](#production-patterns)
- [CLI Reference](#cli-reference)
- [Configuration](#configuration)
- [API Reference](#api-reference)
- [Performance](#performance)
- [Troubleshooting](#troubleshooting)

---

## Core Concepts

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        Engine                               │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────┐   ┌─────────┐   ┌──────────┐   ┌───────────┐  │
│  │  Lexer  │ → │ Parser  │ → │ Executor │ → │  Output   │  │
│  └─────────┘   └─────────┘   └──────────┘   └───────────┘  │
│                                    │                        │
│                    ┌───────────────┼───────────────┐        │
│                    ▼               ▼               ▼        │
│              ┌──────────┐   ┌───────────┐   ┌──────────┐   │
│              │ Registry │   │ Functions │   │ Templates│   │
│              │(Resolvers│   │ Registry  │   │ Registry │   │
│              └──────────┘   └───────────┘   └──────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Key Components

| Component | Purpose |
|-----------|---------|
| **Engine** | Central coordinator; thread-safe for concurrent use |
| **Template** | Parsed AST; reusable across executions |
| **Context** | Execution data with dot-notation path access |
| **Resolver** | Plugin handler for custom tags |
| **Func** | Custom function for expressions |

### Parse Once, Execute Many

For production workloads, parse templates once and execute multiple times:

```go
engine := prompty.MustNew()

// Parse at startup or lazily cache
tmpl, err := engine.Parse(templateSource)
if err != nil {
    log.Fatal(err)
}

// Execute many times (thread-safe)
for _, user := range users {
    result, _ := tmpl.Execute(ctx, map[string]any{"user": user})
    // ...
}
```

---

## Template Syntax

### Delimiters

go-prompty uses `{~` and `~}` delimiters, chosen to minimize conflicts with common prompt content:

| Delimiter | Purpose | Example |
|-----------|---------|---------|
| `{~` | Open tag | `{~prompty.var` |
| `~}` | Close tag | `name="x" /~}` |
| `/~}` | Self-close | `{~tag attr="v" /~}` |
| `{~/` | Block close | `{~/prompty.if~}` |
| `\{~` | Escape (literal) | Outputs `{~` |

### Tag Forms

**Self-closing** (no body):
```
{~prompty.var name="user" default="Guest" /~}
```

**Block** (with body):
```
{~prompty.if eval="isAdmin"~}
  Admin content here
{~/prompty.if~}
```

---

## Built-in Tags

### `prompty.var` - Variable Interpolation

Access values from the execution context using dot-notation paths.

```
{~prompty.var name="user.profile.name" /~}
{~prompty.var name="config.timeout" default="30s" /~}
```

| Attribute | Required | Description |
|-----------|----------|-------------|
| `name` | Yes | Dot-notation path (e.g., `user.settings.theme`) |
| `default` | No | Fallback value if path not found |
| `onerror` | No | Error strategy override |

### `prompty.if` / `prompty.elseif` / `prompty.else` - Conditionals

```
{~prompty.if eval="user.role == 'admin'"~}
  Full access granted.
{~prompty.elseif eval="user.role == 'editor'"~}
  Edit access granted.
{~prompty.else~}
  Read-only access.
{~/prompty.if~}
```

Supports complex expressions:
```
{~prompty.if eval="len(items) > 0 && (isAdmin || hasPermission('view'))"~}
  ...
{~/prompty.if~}
```

### `prompty.for` - Loops

Iterate over slices, arrays, or maps.

```
{~prompty.for item="task" index="i" in="tasks" limit="10"~}
  {~prompty.var name="i" /~}. {~prompty.var name="task.title" /~}
{~/prompty.for~}
```

| Attribute | Required | Description |
|-----------|----------|-------------|
| `item` | Yes | Variable name for current element |
| `in` | Yes | Path to collection |
| `index` | No | Variable name for index (0-based) |
| `limit` | No | Maximum iterations |

**Map iteration:**
```
{~prompty.for item="entry" in="config"~}
  {~prompty.var name="entry.key" /~}: {~prompty.var name="entry.value" /~}
{~/prompty.for~}
```

### `prompty.switch` / `prompty.case` / `prompty.casedefault` - Multi-way Branching

```
{~prompty.switch eval="status"~}
  {~prompty.case value="active"~}
    Account is active.
  {~/prompty.case~}
  {~prompty.case value="suspended"~}
    Account is suspended.
  {~/prompty.case~}
  {~prompty.casedefault~}
    Unknown status.
  {~/prompty.casedefault~}
{~/prompty.switch~}
```

Cases can use `value` (exact match) or `eval` (expression):
```
{~prompty.case eval="score >= 90"~}Grade A{~/prompty.case~}
```

### `prompty.include` - Nested Templates

Compose prompts from reusable fragments.

```go
engine.MustRegisterTemplate("system-prefix", `You are {~prompty.var name="assistant_name" /~}, a helpful assistant.`)
engine.MustRegisterTemplate("user-context", `User: {~prompty.var name="name" /~} ({~prompty.var name="tier" /~} tier)`)
```

```
{~prompty.include template="system-prefix" assistant_name="Claude" /~}

{~prompty.include template="user-context" with="user" /~}
```

| Attribute | Required | Description |
|-----------|----------|-------------|
| `template` | Yes | Registered template name |
| `with` | No | Use value at path as context root |
| `isolate` | No | `"true"` to not inherit parent context |
| *(other)* | No | Passed as variables to child template |

### `prompty.raw` - Unprocessed Content

Preserve content without parsing (for code examples, other template syntaxes):

```
{~prompty.raw~}
Example template syntax: {{ variable }} or <%= erb %>
This {~tag~} won't be parsed.
{~/prompty.raw~}
```

### `prompty.comment` - Removed from Output

```
{~prompty.comment~}
TODO: Add more examples
Internal note: This section needs review
{~/prompty.comment~}
```

---

## Expression Language

Expressions are used in `eval` attributes for conditionals and switch/case.

### Operators

| Category | Operators |
|----------|-----------|
| Comparison | `==`, `!=`, `<`, `>`, `<=`, `>=` |
| Logical | `&&`, `\|\|`, `!` |
| Grouping | `(`, `)` |

### Truthiness

| Type | Truthy | Falsy |
|------|--------|-------|
| `bool` | `true` | `false` |
| `string` | non-empty | `""` |
| `int/float` | non-zero | `0` |
| `slice/map` | non-empty | empty |
| `nil` | - | always falsy |

### Built-in Functions

<details>
<summary><strong>String Functions</strong></summary>

| Function | Description |
|----------|-------------|
| `upper(s)` | Uppercase |
| `lower(s)` | Lowercase |
| `trim(s)` | Remove whitespace |
| `trimPrefix(s, prefix)` | Remove prefix |
| `trimSuffix(s, suffix)` | Remove suffix |
| `hasPrefix(s, prefix)` | Check prefix |
| `hasSuffix(s, suffix)` | Check suffix |
| `contains(s, substr)` | Check contains |
| `replace(s, old, new)` | Replace all |
| `split(s, sep)` | Split to slice |
| `join(slice, sep)` | Join with separator |

</details>

<details>
<summary><strong>Collection Functions</strong></summary>

| Function | Description |
|----------|-------------|
| `len(x)` | Length of string/slice/map |
| `first(slice)` | First element |
| `last(slice)` | Last element |
| `keys(map)` | Map keys (sorted) |
| `values(map)` | Map values |
| `has(map, key)` | Check map has key |
| `contains(slice, item)` | Check slice contains item |

</details>

<details>
<summary><strong>Type Functions</strong></summary>

| Function | Description |
|----------|-------------|
| `toString(x)` | Convert to string |
| `toInt(x)` | Convert to integer |
| `toFloat(x)` | Convert to float |
| `toBool(x)` | Convert to boolean |
| `typeOf(x)` | Get type name |
| `isNil(x)` | Check if nil |
| `isEmpty(x)` | Check if empty |

</details>

<details>
<summary><strong>Utility Functions</strong></summary>

| Function | Description |
|----------|-------------|
| `default(x, fallback)` | Return fallback if x is nil/empty |
| `coalesce(a, b, ...)` | First non-nil, non-empty value |

</details>

### Expression Examples

```
// Simple
{~prompty.if eval="age >= 18"~}

// Function calls
{~prompty.if eval="len(trim(input)) > 0"~}

// Complex logic
{~prompty.if eval="(isAdmin || isModerator) && !isBanned && len(permissions) > 0"~}
```

---

## Custom Resolvers

Extend go-prompty with custom tag handlers.

### Interface

```go
type Resolver interface {
    TagName() string
    Resolve(ctx context.Context, execCtx *prompty.Context, attrs prompty.Attributes) (string, error)
    Validate(attrs prompty.Attributes) error
}
```

### Full Example

```go
// TimestampResolver handles {~app.timestamp format="..." /~}
type TimestampResolver struct{}

func (r *TimestampResolver) TagName() string { return "app.timestamp" }

func (r *TimestampResolver) Resolve(ctx context.Context, execCtx *prompty.Context, attrs prompty.Attributes) (string, error) {
    format := attrs.GetDefault("format", time.RFC3339)
    return time.Now().Format(format), nil
}

func (r *TimestampResolver) Validate(attrs prompty.Attributes) error {
    return nil // format is optional
}

// Register
engine.MustRegister(&TimestampResolver{})

// Use
// {~app.timestamp format="2006-01-02" /~}
```

### Quick Resolver with ResolverFunc

```go
engine.MustRegister(prompty.NewResolverFunc(
    "app.uuid",
    func(ctx context.Context, execCtx *prompty.Context, attrs prompty.Attributes) (string, error) {
        return uuid.New().String(), nil
    },
    nil, // no validation
))
```

---

## Custom Functions

Register functions for use in expressions.

```go
engine.MustRegisterFunc(&prompty.Func{
    Name:    "initials",
    MinArgs: 1,
    MaxArgs: 1,
    Fn: func(args []any) (any, error) {
        name, _ := args[0].(string)
        var result strings.Builder
        for _, word := range strings.Fields(name) {
            if len(word) > 0 {
                result.WriteString(strings.ToUpper(word[:1]))
            }
        }
        return result.String(), nil
    },
})

// Use in expression:
// {~prompty.if eval="initials(user.name) == 'JD'"~}
```

**Variadic functions** (set `MaxArgs: -1`):
```go
engine.MustRegisterFunc(&prompty.Func{
    Name:    "sum",
    MinArgs: 1,
    MaxArgs: -1,
    Fn: func(args []any) (any, error) {
        var total float64
        for _, arg := range args {
            // ... add each number
        }
        return total, nil
    },
})
```

---

## Production Patterns

### Prompt Registry Pattern

Organize prompts in a central registry for larger applications:

```go
type PromptRegistry struct {
    engine *prompty.Engine
    mu     sync.RWMutex
    cache  map[string]*prompty.Template
}

func NewPromptRegistry() *PromptRegistry {
    engine := prompty.MustNew(
        prompty.WithErrorStrategy(prompty.ErrorStrategyDefault),
        prompty.WithMaxDepth(20),
    )

    // Register common fragments
    engine.MustRegisterTemplate("system-base", systemBaseTemplate)
    engine.MustRegisterTemplate("user-context", userContextTemplate)
    engine.MustRegisterTemplate("safety-suffix", safetyTemplate)

    return &PromptRegistry{
        engine: engine,
        cache:  make(map[string]*prompty.Template),
    }
}

func (r *PromptRegistry) Execute(name string, data map[string]any) (string, error) {
    r.mu.RLock()
    tmpl, ok := r.cache[name]
    r.mu.RUnlock()

    if !ok {
        return "", fmt.Errorf("unknown prompt: %s", name)
    }

    return tmpl.Execute(context.Background(), data)
}

func (r *PromptRegistry) Register(name, source string) error {
    tmpl, err := r.engine.Parse(source)
    if err != nil {
        return err
    }

    r.mu.Lock()
    r.cache[name] = tmpl
    r.mu.Unlock()
    return nil
}
```

### Version Control for Prompts

Store prompts as versioned files:

```
prompts/
├── v1/
│   ├── chat-system.prompty
│   └── summarize.prompty
└── v2/
    ├── chat-system.prompty  # Updated version
    └── summarize.prompty
```

Load at startup with version selection:

```go
func LoadPrompts(engine *prompty.Engine, version string) error {
    pattern := fmt.Sprintf("prompts/%s/*.prompty", version)
    files, _ := filepath.Glob(pattern)

    for _, file := range files {
        content, _ := os.ReadFile(file)
        name := strings.TrimSuffix(filepath.Base(file), ".prompty")
        if err := engine.RegisterTemplate(name, string(content)); err != nil {
            return err
        }
    }
    return nil
}
```

### Error Strategy Selection

| Scenario | Recommended Strategy |
|----------|---------------------|
| Development | `throw` - Fail fast, see all errors |
| Production (critical) | `throw` - Don't serve malformed prompts |
| Production (graceful) | `default` - Use defaults, log issues |
| User-facing previews | `keepraw` - Show unresolved tags |
| Debug/logging | `log` - Continue but capture issues |

```go
// Per-engine (global)
engine, _ := prompty.New(prompty.WithErrorStrategy(prompty.ErrorStrategyDefault))

// Per-tag override
{~prompty.var name="optional.field" onerror="remove" /~}
```

### Graceful Degradation

```go
func ExecuteWithFallback(engine *prompty.Engine, primary, fallback string, data map[string]any) string {
    result, err := engine.Execute(context.Background(), primary, data)
    if err != nil {
        log.Printf("Primary prompt failed: %v, using fallback", err)
        result, _ = engine.Execute(context.Background(), fallback, data)
    }
    return result
}
```

### Context Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

result, err := engine.Execute(ctx, template, data)
if errors.Is(err, context.DeadlineExceeded) {
    // Handle timeout
}
```

---

## CLI Reference

### render

Execute templates from the command line.

```bash
# Basic
prompty render -t prompt.txt -d '{"user": "Alice"}'

# From file
prompty render -t prompt.txt -f data.json

# From stdin
cat prompt.txt | prompty render -t - -d '{"user": "Bob"}'

# Output to file
prompty render -t prompt.txt -d '{}' -o output.txt
```

### validate

Check template syntax without executing.

```bash
# Basic validation
prompty validate -t prompt.txt

# Strict mode (warnings → errors)
prompty validate -t prompt.txt --strict

# JSON output (for CI/CD)
prompty validate -t prompt.txt -F json
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Usage error |
| 3 | Validation error |
| 4 | Input error |

---

## Configuration

### Engine Options

```go
engine, err := prompty.New(
    prompty.WithDelimiters("<%", "%>"),           // Custom delimiters
    prompty.WithErrorStrategy(prompty.ErrorStrategyDefault),
    prompty.WithMaxDepth(50),                     // Template nesting limit
    prompty.WithLogger(zapLogger),                // Structured logging
)
```

### Default Limits

| Limit | Default | Description |
|-------|---------|-------------|
| Max Depth | 10 | Template nesting |
| Max Loop Iterations | 10,000 | Per loop |
| Max Output Size | 10 MB | Total output |
| Execution Timeout | 30s | Overall |
| Resolver Timeout | 5s | Per resolver |

---

## API Reference

<details>
<summary><strong>Engine</strong></summary>

```go
// Creation
func New(opts ...Option) (*Engine, error)
func MustNew(opts ...Option) *Engine

// Execution
func (e *Engine) Execute(ctx context.Context, source string, data map[string]any) (string, error)
func (e *Engine) Parse(source string) (*Template, error)
func (e *Engine) Validate(source string) (*ValidationResult, error)

// Resolvers
func (e *Engine) Register(resolver Resolver) error
func (e *Engine) MustRegister(resolver Resolver)
func (e *Engine) HasResolver(tagName string) bool
func (e *Engine) ListResolvers() []string

// Templates
func (e *Engine) RegisterTemplate(name, source string) error
func (e *Engine) MustRegisterTemplate(name, source string)
func (e *Engine) UnregisterTemplate(name string) bool
func (e *Engine) GetTemplate(name string) (*Template, bool)
func (e *Engine) HasTemplate(name string) bool
func (e *Engine) ListTemplates() []string

// Functions
func (e *Engine) RegisterFunc(f *Func) error
func (e *Engine) MustRegisterFunc(f *Func)
func (e *Engine) HasFunc(name string) bool
func (e *Engine) ListFuncs() []string
```

</details>

<details>
<summary><strong>Template</strong></summary>

```go
func (t *Template) Execute(ctx context.Context, data map[string]any) (string, error)
func (t *Template) ExecuteWithContext(ctx context.Context, execCtx *Context) (string, error)
func (t *Template) Source() string
```

</details>

<details>
<summary><strong>Context</strong></summary>

```go
func NewContext(data map[string]any) *Context
func NewContextWithStrategy(data map[string]any, strategy ErrorStrategy) *Context

func (c *Context) Get(path string) (any, bool)
func (c *Context) GetString(path string) string
func (c *Context) GetDefault(path string, defaultVal any) any
func (c *Context) GetStringDefault(path, defaultVal string) string
func (c *Context) Has(path string) bool
func (c *Context) Set(key string, value any)
func (c *Context) Data() map[string]any
func (c *Context) Child(data map[string]any) interface{}
func (c *Context) Parent() *Context
```

</details>

<details>
<summary><strong>ValidationResult</strong></summary>

```go
func (r *ValidationResult) IsValid() bool
func (r *ValidationResult) HasErrors() bool
func (r *ValidationResult) HasWarnings() bool
func (r *ValidationResult) Issues() []ValidationIssue
func (r *ValidationResult) Errors() []ValidationIssue
func (r *ValidationResult) Warnings() []ValidationIssue
```

</details>

---

## Performance

### Benchmarks

| Operation | Time | Allocations |
|-----------|------|-------------|
| Parse (small) | ~15μs | ~20 |
| Parse (medium) | ~80μs | ~100 |
| Execute (simple) | ~5μs | ~10 |
| Execute (complex) | ~50μs | ~80 |

### Optimization Tips

1. **Parse once, execute many** - Cache parsed templates
2. **Limit loop iterations** - Use `limit` attribute
3. **Avoid deep nesting** - Keep template depth reasonable
4. **Use simple expressions** - Complex expressions add overhead

---

## Troubleshooting

### Common Issues

<details>
<summary><strong>Tag not recognized</strong></summary>

Ensure the resolver is registered before parsing:
```go
engine.MustRegister(&MyResolver{})
tmpl, _ := engine.Parse(source) // Resolver must be registered first
```

</details>

<details>
<summary><strong>Variable not found</strong></summary>

Check the path and use `default`:
```
{~prompty.var name="user.name" default="Unknown" /~}
```

Or set error strategy:
```
{~prompty.var name="optional" onerror="remove" /~}
```

</details>

<details>
<summary><strong>Infinite loop / max depth</strong></summary>

Templates including themselves cause recursion. Use `WithMaxDepth`:
```go
engine, _ := prompty.New(prompty.WithMaxDepth(5))
```

</details>

<details>
<summary><strong>Delimiter conflicts</strong></summary>

Use custom delimiters:
```go
engine, _ := prompty.New(prompty.WithDelimiters("<%", "%>"))
```

Or escape:
```
Use \{~ for literal delimiters.
```

</details>

---

## Storage & Persistence

go-prompty includes a pluggable storage layer for managing templates with versioning, metadata, and multi-tenant support:

- **Built-in drivers**: Memory (testing) and Filesystem (persistent)
- **Custom backends**: Implement `TemplateStorage` for PostgreSQL, MongoDB, Redis, etc.
- **Caching**: Automatic caching wrapper for any storage backend

```go
// Filesystem storage (persistent)
storage, _ := prompty.NewFilesystemStorage("/path/to/templates")
engine, _ := prompty.NewStorageEngine(prompty.StorageEngineConfig{
    Storage: storage,
})

// Save with versioning
engine.Save(ctx, &prompty.StoredTemplate{
    Name:   "greeting",
    Source: `Hello {~prompty.var name="user" /~}!`,
    Tags:   []string{"production"},
})

// Execute
result, _ := engine.Execute(ctx, "greeting", map[string]any{"user": "Alice"})
```

See [docs/STORAGE.md](docs/STORAGE.md) for complete documentation, and [docs/CUSTOM_STORAGE.md](docs/CUSTOM_STORAGE.md) for implementing custom database backends like PostgreSQL.

---

## Contributing

Contributions welcome! Please read our contributing guidelines and submit PRs.

## License

MIT License - see [LICENSE](LICENSE) for details.

---

<div align="center">

**[Documentation](https://pkg.go.dev/github.com/itsatony/go-prompty)** |
**[Examples](examples/)** |
**[Issues](https://github.com/itsatony/go-prompty/issues)**

</div>
