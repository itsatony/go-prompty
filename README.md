# go-prompty

A dynamic LLM prompt templating system for Go with plugin-based architecture.

[![Go Reference](https://pkg.go.dev/badge/github.com/itsatony/go-prompty.svg)](https://pkg.go.dev/github.com/itsatony/go-prompty)
[![Go Report Card](https://goreportcard.com/badge/github.com/itsatony/go-prompty)](https://goreportcard.com/report/github.com/itsatony/go-prompty)

## Overview

go-prompty provides a powerful templating engine designed specifically for constructing dynamic LLM prompts. It features:

- **Content-Resistant Syntax** - Uses `{~...~}` delimiters that won't conflict with code, XML, JSON, or other prompt content
- **Plugin Architecture** - Extend with custom resolvers for domain-specific functionality
- **Safe Expression Language** - Evaluate conditions with built-in functions and operators
- **Nested Templates** - Register and include reusable template fragments
- **Flexible Error Handling** - Five error strategies from strict to lenient
- **Thread-Safe** - Safe for concurrent use across goroutines
- **CLI Tool** - Render and validate templates from the command line

## Installation

### Go Package

```bash
go get github.com/itsatony/go-prompty
```

### CLI Tool

```bash
go install github.com/itsatony/go-prompty/cmd/prompty@latest
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/itsatony/go-prompty"
)

func main() {
    // Create an engine
    engine := prompty.MustNew()

    // Define a template
    template := `Hello, {~prompty.var name="user" /~}! You have {~prompty.var name="count" /~} messages.`

    // Execute with data
    result, err := engine.Execute(context.Background(), template, map[string]any{
        "user":  "Alice",
        "count": 5,
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(result)
    // Output: Hello, Alice! You have 5 messages.
}
```

### Parse Once, Execute Many

For better performance with repeated executions:

```go
engine := prompty.MustNew()

// Parse the template once
tmpl, err := engine.Parse("Hello, {~prompty.var name=\"user\" /~}!")
if err != nil {
    panic(err)
}

// Execute multiple times with different data
for _, name := range []string{"Alice", "Bob", "Charlie"} {
    result, _ := tmpl.Execute(context.Background(), map[string]any{"user": name})
    fmt.Println(result)
}
```

---

## Template Syntax

### Delimiters

go-prompty uses `{~` and `~}` as delimiters by default. This syntax was chosen to minimize conflicts with common prompt content like code, XML, JSON, and natural language.

**Self-closing tags:**
```
{~tagname attr="value" /~}
```

**Block tags:**
```
{~tagname~}
  content here
{~/tagname~}
```

### Variables (`prompty.var`)

Interpolate values from the execution context.

```
{~prompty.var name="username" /~}
{~prompty.var name="user.profile.email" /~}
{~prompty.var name="greeting" default="Hello" /~}
```

**Attributes:**
| Attribute | Required | Description |
|-----------|----------|-------------|
| `name` | Yes | Dot-notation path to the value (e.g., `user.name`) |
| `default` | No | Fallback value if the path doesn't exist |
| `onerror` | No | Error strategy override for this tag |

**Example:**
```go
template := `Welcome, {~prompty.var name="user.name" default="Guest" /~}!`

data := map[string]any{
    "user": map[string]any{
        "name": "Alice",
        "email": "alice@example.com",
    },
}

result, _ := engine.Execute(ctx, template, data)
// Output: Welcome, Alice!
```

### Conditionals (`prompty.if`, `prompty.elseif`, `prompty.else`)

Conditional rendering based on expression evaluation.

```
{~prompty.if eval="user.isAdmin"~}
  You have admin access.
{~prompty.elseif eval="user.isModerator"~}
  You have moderator access.
{~prompty.else~}
  You have standard access.
{~/prompty.if~}
```

**Attributes:**
| Attribute | Required | Description |
|-----------|----------|-------------|
| `eval` | Yes* | Expression to evaluate (*not allowed on `prompty.else`) |

**Examples with expressions:**
```
{~prompty.if eval="len(items) > 0"~}
  You have {~prompty.var name="items" /~} items.
{~/prompty.if~}

{~prompty.if eval="contains(roles, 'admin') && isActive"~}
  Admin panel available.
{~/prompty.if~}

{~prompty.if eval="!user.isGuest"~}
  Welcome back!
{~/prompty.if~}
```

### Loops (`prompty.for`)

Iterate over slices, arrays, or maps.

```
{~prompty.for item="task" index="i" in="tasks"~}
  {~prompty.var name="i" /~}. {~prompty.var name="task.title" /~}
{~/prompty.for~}
```

**Attributes:**
| Attribute | Required | Description |
|-----------|----------|-------------|
| `item` | Yes | Variable name for the current item |
| `in` | Yes | Dot-notation path to the collection |
| `index` | No | Variable name for the current index (0-based) |
| `limit` | No | Maximum number of iterations |

**Iterating over slices:**
```go
template := `Tasks:
{~prompty.for item="task" index="i" in="tasks"~}
- [{~prompty.var name="i" /~}] {~prompty.var name="task" /~}
{~/prompty.for~}`

data := map[string]any{
    "tasks": []string{"Write code", "Review PR", "Deploy"},
}

// Output:
// Tasks:
// - [0] Write code
// - [1] Review PR
// - [2] Deploy
```

**Iterating over maps:**
```go
template := `Config:
{~prompty.for item="entry" in="config"~}
  {~prompty.var name="entry.key" /~}: {~prompty.var name="entry.value" /~}
{~/prompty.for~}`

data := map[string]any{
    "config": map[string]any{
        "host": "localhost",
        "port": 8080,
    },
}
// Each item is a map with "key" and "value" fields
```

**With limit:**
```
{~prompty.for item="item" in="items" limit="5"~}
  {~prompty.var name="item" /~}
{~/prompty.for~}
```

### Switch/Case (`prompty.switch`, `prompty.case`, `prompty.casedefault`)

Multi-way branching based on value or expression matching.

```
{~prompty.switch eval="status"~}
  {~prompty.case value="active"~}
    Account is active.
  {~/prompty.case~}
  {~prompty.case value="pending"~}
    Account is pending activation.
  {~/prompty.case~}
  {~prompty.casedefault~}
    Unknown status.
  {~/prompty.casedefault~}
{~/prompty.switch~}
```

**Switch Attributes:**
| Attribute | Required | Description |
|-----------|----------|-------------|
| `eval` | Yes | Expression whose result is matched against cases |

**Case Attributes:**
| Attribute | Required | Description |
|-----------|----------|-------------|
| `value` | Yes* | Value to match against (*mutually exclusive with `eval`) |
| `eval` | Yes* | Expression that must evaluate to true (*mutually exclusive with `value`) |

**Using eval in cases:**
```
{~prompty.switch eval="score"~}
  {~prompty.case eval="score >= 90"~}
    Grade: A
  {~/prompty.case~}
  {~prompty.case eval="score >= 80"~}
    Grade: B
  {~/prompty.case~}
  {~prompty.case eval="score >= 70"~}
    Grade: C
  {~/prompty.case~}
  {~prompty.casedefault~}
    Grade: F
  {~/prompty.casedefault~}
{~/prompty.switch~}
```

**Notes:**
- First matching case wins (no fall-through)
- `prompty.casedefault` must be last if present
- Only one default case allowed

### Nested Templates (`prompty.include`)

Register reusable templates and include them in other templates.

```go
engine := prompty.MustNew()

// Register templates
engine.MustRegisterTemplate("header", `=== {~prompty.var name="title" default="Untitled" /~} ===`)
engine.MustRegisterTemplate("footer", `---
Generated by go-prompty`)

// Use in a template
template := `{~prompty.include template="header" title="My Document" /~}

Content goes here...

{~prompty.include template="footer" /~}`
```

**Attributes:**
| Attribute | Required | Description |
|-----------|----------|-------------|
| `template` | Yes | Name of the registered template |
| `with` | No | Context path - use value at path as root context |
| `isolate` | No | Set to "true" to not inherit parent context |
| *other* | No | Any other attributes become context variables in the child template |

**Context inheritance:**
```go
engine.MustRegisterTemplate("greeting", `Hello, {~prompty.var name="name" /~}!`)

// Parent context is inherited
result, _ := engine.Execute(ctx,
    `{~prompty.include template="greeting" /~}`,
    map[string]any{"name": "Alice"})
// Output: Hello, Alice!

// Override with attributes
result, _ := engine.Execute(ctx,
    `{~prompty.include template="greeting" name="Bob" /~}`,
    map[string]any{"name": "Alice"})
// Output: Hello, Bob!
```

**Using `with` attribute:**
```go
engine.MustRegisterTemplate("user-card", `
Name: {~prompty.var name="name" /~}
Email: {~prompty.var name="email" /~}`)

data := map[string]any{
    "currentUser": map[string]any{
        "name": "Alice",
        "email": "alice@example.com",
    },
}

template := `{~prompty.include template="user-card" with="currentUser" /~}`
// The "currentUser" map becomes the root context for the included template
```

**Isolated context:**
```
{~prompty.include template="standalone" isolate="true" customVar="value" /~}
```

**Template name rules:**
- Cannot be empty
- Cannot start with `prompty.` (reserved namespace)
- First registration wins for duplicate names

### Raw Blocks (`prompty.raw`)

Preserve content without parsing. Useful for including code examples or other templating syntax.

```
{~prompty.raw~}
This content is not parsed: {~prompty.var name="ignored" /~}
You can include {{ jinja }} or <%= erb %> syntax here.
{~/prompty.raw~}
```

### Comments (`prompty.comment`)

Comments are completely removed from output.

```
{~prompty.comment~}
This is a comment.
It will not appear in the output.
TODO: Add more examples
{~/prompty.comment~}
```

### Escape Sequences

To output a literal `{~`, use a backslash:

```
Use \{~ to show the delimiter literally.
```

Output: `Use {~ to show the delimiter literally.`

---

## Expression Language

Expressions are used in `eval` attributes for conditionals and switch statements.

### Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `==` | Equal | `status == "active"` |
| `!=` | Not equal | `count != 0` |
| `<` | Less than | `age < 18` |
| `>` | Greater than | `score > 90` |
| `<=` | Less than or equal | `attempts <= 3` |
| `>=` | Greater than or equal | `balance >= 100` |
| `&&` | Logical AND | `isAdmin && isActive` |
| `\|\|` | Logical OR | `isOwner \|\| isModerator` |
| `!` | Logical NOT | `!isGuest` |

### Truthiness

Values are evaluated for truthiness as follows:

| Type | Truthy | Falsy |
|------|--------|-------|
| `bool` | `true` | `false` |
| `string` | non-empty | `""` |
| `int/float` | non-zero | `0` |
| `slice/map` | non-empty | empty |
| `nil` | - | always falsy |

### Built-in Functions

#### String Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `upper` | `upper(s string) string` | Convert to uppercase |
| `lower` | `lower(s string) string` | Convert to lowercase |
| `trim` | `trim(s string) string` | Remove leading/trailing whitespace |
| `trimPrefix` | `trimPrefix(s, prefix string) string` | Remove prefix if present |
| `trimSuffix` | `trimSuffix(s, suffix string) string` | Remove suffix if present |
| `hasPrefix` | `hasPrefix(s, prefix string) bool` | Check if string starts with prefix |
| `hasSuffix` | `hasSuffix(s, suffix string) bool` | Check if string ends with suffix |
| `contains` | `contains(s, substr string) bool` | Check if string contains substring |
| `replace` | `replace(s, old, new string) string` | Replace all occurrences |
| `split` | `split(s, sep string) []string` | Split string by separator |
| `join` | `join(items []string, sep string) string` | Join strings with separator |

#### Collection Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `len` | `len(x any) int` | Length of string, slice, or map |
| `first` | `first(x []any) any` | First element of slice |
| `last` | `last(x []any) any` | Last element of slice |
| `keys` | `keys(m map) []string` | Map keys (sorted) |
| `values` | `values(m map) []any` | Map values |
| `has` | `has(m map, key string) bool` | Check if map has key |
| `contains` | `contains(slice []any, item any) bool` | Check if slice contains item |

#### Type Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `toString` | `toString(x any) string` | Convert to string |
| `toInt` | `toInt(x any) int` | Convert to integer |
| `toFloat` | `toFloat(x any) float64` | Convert to float |
| `toBool` | `toBool(x any) bool` | Convert to boolean |
| `typeOf` | `typeOf(x any) string` | Get type name |
| `isNil` | `isNil(x any) bool` | Check if nil |
| `isEmpty` | `isEmpty(x any) bool` | Check if empty |

#### Utility Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `default` | `default(x any, fallback any) any` | Return fallback if x is nil/empty |
| `coalesce` | `coalesce(args ...any) any` | Return first non-nil, non-empty value |

### Expression Examples

```
// Simple comparison
{~prompty.if eval="age >= 18"~}Adult{~/prompty.if~}

// String operations
{~prompty.if eval="hasPrefix(email, 'admin@')"~}Admin email{~/prompty.if~}

// Collection checks
{~prompty.if eval="len(items) > 0 && contains(items, 'important')"~}
  Has important items
{~/prompty.if~}

// Complex expressions
{~prompty.if eval="(isAdmin || isModerator) && !isBanned"~}
  Access granted
{~/prompty.if~}

// Nested function calls
{~prompty.if eval="len(trim(username)) > 0"~}
  Valid username
{~/prompty.if~}
```

---

## Error Strategies

go-prompty provides five error handling strategies:

| Strategy | Behavior |
|----------|----------|
| `throw` | Stop execution and return error (default) |
| `default` | Use the `default` attribute value, or empty string |
| `remove` | Remove the tag from output |
| `keepraw` | Keep the original tag text in output |
| `log` | Log the error and continue with empty string |

### Global Strategy

Set the default strategy for the entire engine:

```go
engine, _ := prompty.New(
    prompty.WithErrorStrategy(prompty.ErrorStrategyDefault),
)
```

### Per-Tag Override

Override the strategy for individual tags:

```
{~prompty.var name="missing" onerror="remove" /~}
{~prompty.var name="optional" default="N/A" onerror="default" /~}
{~prompty.var name="debug" onerror="keepraw" /~}
```

---

## Custom Resolvers

Extend go-prompty with custom tag handlers by implementing the `Resolver` interface.

### Interface Definition

```go
type Resolver interface {
    // TagName returns the tag name this resolver handles
    TagName() string

    // Resolve executes the tag and returns the output string
    Resolve(ctx context.Context, execCtx *prompty.Context, attrs prompty.Attributes) (string, error)

    // Validate checks if the attributes are valid (called during parsing)
    Validate(attrs prompty.Attributes) error
}
```

### Complete Example

```go
package main

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/itsatony/go-prompty"
)

// DateResolver handles {~myapp.date format="..." /~} tags
type DateResolver struct{}

func (r *DateResolver) TagName() string {
    return "myapp.date"
}

func (r *DateResolver) Resolve(ctx context.Context, execCtx *prompty.Context, attrs prompty.Attributes) (string, error) {
    format := attrs.GetDefault("format", "2006-01-02")

    // Check for a specific date in context
    if dateStr := execCtx.GetString("date"); dateStr != "" {
        t, err := time.Parse(time.RFC3339, dateStr)
        if err != nil {
            return "", fmt.Errorf("invalid date: %w", err)
        }
        return t.Format(format), nil
    }

    // Default to current time
    return time.Now().Format(format), nil
}

func (r *DateResolver) Validate(attrs prompty.Attributes) error {
    // format is optional, no required attributes
    return nil
}

// UppercaseResolver handles {~myapp.upper~}content{~/myapp.upper~} blocks
type UppercaseResolver struct{}

func (r *UppercaseResolver) TagName() string {
    return "myapp.upper"
}

func (r *UppercaseResolver) Resolve(ctx context.Context, execCtx *prompty.Context, attrs prompty.Attributes) (string, error) {
    // For block tags, the content is available via the "content" attribute
    content := attrs.GetDefault("content", "")
    return strings.ToUpper(content), nil
}

func (r *UppercaseResolver) Validate(attrs prompty.Attributes) error {
    return nil
}

func main() {
    engine := prompty.MustNew()

    // Register custom resolvers
    engine.MustRegister(&DateResolver{})
    engine.MustRegister(&UppercaseResolver{})

    template := `
Today is {~myapp.date format="Monday, January 2" /~}.
{~myapp.upper~}this will be uppercase{~/myapp.upper~}
`

    result, _ := engine.Execute(context.Background(), template, nil)
    fmt.Println(result)
}
```

### Using ResolverFunc for Simple Cases

For simple resolvers, use `ResolverFunc`:

```go
engine := prompty.MustNew()

// Create a simple resolver using ResolverFunc
echoResolver := prompty.NewResolverFunc(
    "echo",
    func(ctx context.Context, execCtx *prompty.Context, attrs prompty.Attributes) (string, error) {
        msg, _ := attrs.Get("msg")
        return msg, nil
    },
    func(attrs prompty.Attributes) error {
        if !attrs.Has("msg") {
            return fmt.Errorf("missing required attribute: msg")
        }
        return nil
    },
)

engine.MustRegister(echoResolver)

result, _ := engine.Execute(ctx, `{~echo msg="Hello!" /~}`, nil)
// Output: Hello!
```

### Accessing Context Data

The `Context` object provides access to template data:

```go
func (r *MyResolver) Resolve(ctx context.Context, execCtx *prompty.Context, attrs prompty.Attributes) (string, error) {
    // Get a value by path
    value, exists := execCtx.Get("user.name")

    // Get string with default
    name := execCtx.GetStringDefault("user.name", "Anonymous")

    // Check if path exists
    if execCtx.Has("user.email") {
        // ...
    }

    // Get all data
    data := execCtx.Data()

    return result, nil
}
```

### Registration Behavior

- First registration wins (subsequent registrations for the same tag name are ignored)
- Use `Register()` for error handling or `MustRegister()` to panic on error
- Check registration with `HasResolver(tagName)`
- List all resolvers with `ListResolvers()`

---

## Custom Functions

Register custom functions for use in expressions.

```go
engine := prompty.MustNew()

// Register a custom function
engine.MustRegisterFunc(&prompty.Func{
    Name:    "greet",
    MinArgs: 1,
    MaxArgs: 1,
    Fn: func(args []any) (any, error) {
        name, ok := args[0].(string)
        if !ok {
            return nil, fmt.Errorf("expected string argument")
        }
        return "Hello, " + name + "!", nil
    },
})

// Use in templates
template := `{~prompty.if eval="greet(user) == 'Hello, Admin!'"~}
  Welcome, administrator!
{~/prompty.if~}`
```

### Variadic Functions

Set `MaxArgs` to `-1` for variadic functions:

```go
engine.MustRegisterFunc(&prompty.Func{
    Name:    "sum",
    MinArgs: 1,
    MaxArgs: -1, // Variadic
    Fn: func(args []any) (any, error) {
        total := 0.0
        for _, arg := range args {
            switch v := arg.(type) {
            case int:
                total += float64(v)
            case float64:
                total += v
            default:
                return nil, fmt.Errorf("expected numeric argument")
            }
        }
        return total, nil
    },
})

// Use: {~prompty.if eval="sum(1, 2, 3, 4, 5) > 10"~}...{~/prompty.if~}
```

### Function Management

```go
// Check if function exists
if engine.HasFunc("myFunc") {
    // ...
}

// List all functions (built-in + custom)
funcs := engine.ListFuncs()

// Get function count
count := engine.FuncCount()
```

---

## Validation API

Validate templates without executing them.

```go
engine := prompty.MustNew()

result, err := engine.Validate(templateSource)
if err != nil {
    // Parse error
    log.Fatal(err)
}

// Check for issues
if !result.IsValid() {
    for _, issue := range result.Issues() {
        fmt.Printf("[%s] %s at line %d, column %d\n",
            issue.Severity, issue.Message, issue.Position.Line, issue.Position.Column)
    }
}

// Filter by severity
errors := result.Errors()
warnings := result.Warnings()
```

### Validation Checks

- Unknown tag names (warning)
- Missing required attributes (error)
- Invalid attribute values (error)
- Unbalanced block tags (error)
- Invalid error strategy values (warning)
- Missing include template targets (warning)

---

## CLI Usage

The `prompty` CLI tool provides template rendering and validation.

### Installation

```bash
go install github.com/itsatony/go-prompty/cmd/prompty@latest
```

### Commands

#### render

Execute a template with data.

```bash
# Basic usage
prompty render -t template.txt -d '{"user": "Alice"}'

# From file data
prompty render -t template.txt -f data.json

# From stdin
cat template.txt | prompty render -t - -d '{"user": "Bob"}'

# Output to file
prompty render -t template.txt -d '{}' -o output.txt
```

**Options:**
| Flag | Short | Description |
|------|-------|-------------|
| `--template` | `-t` | Template file (use `-` for stdin) |
| `--data` | `-d` | JSON data string |
| `--data-file` | `-f` | JSON data file |
| `--output` | `-o` | Output file (default: stdout) |
| `--quiet` | `-q` | Suppress non-error output |

#### validate

Check template syntax without executing.

```bash
# Basic validation
prompty validate -t template.txt

# Strict mode (warnings become errors)
prompty validate -t template.txt --strict

# JSON output
prompty validate -t template.txt -F json
```

**Options:**
| Flag | Short | Description |
|------|-------|-------------|
| `--template` | `-t` | Template file (use `-` for stdin) |
| `--format` | `-F` | Output format: `text` or `json` (default: text) |
| `--strict` | | Treat warnings as errors |

#### version

Display version information.

```bash
prompty version
prompty version -F json
```

#### help

Show help for commands.

```bash
prompty help
prompty help render
prompty help validate
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Usage error (invalid arguments) |
| 3 | Validation error |
| 4 | Input error (file not found, invalid JSON) |

---

## Configuration Options

Configure the engine with functional options:

```go
engine, err := prompty.New(
    // Custom delimiters
    prompty.WithDelimiters("<%", "%>"),

    // Global error strategy
    prompty.WithErrorStrategy(prompty.ErrorStrategyDefault),

    // Maximum nesting depth (0 = unlimited)
    prompty.WithMaxDepth(50),

    // Structured logging
    prompty.WithLogger(zapLogger),
)
```

### Default Values

| Option | Default | Description |
|--------|---------|-------------|
| Delimiters | `{~` / `~}` | Tag delimiters |
| Error Strategy | `throw` | How to handle errors |
| Max Depth | `10` | Maximum template nesting depth |
| Logger | `nil` | No logging |

### Resource Limits

Built-in limits protect against resource exhaustion:

| Limit | Default | Description |
|-------|---------|-------------|
| Max Loop Iterations | 10,000 | Per-loop iteration limit |
| Max Output Size | 10 MB | Total output size limit |
| Execution Timeout | 30s | Overall execution timeout |
| Resolver Timeout | 5s | Per-resolver timeout |
| Function Timeout | 1s | Per-function timeout |

---

## Thread Safety

go-prompty is designed for concurrent use:

- **Engine**: Safe for concurrent `Execute()`, `Parse()`, `Register()` calls
- **Template**: Safe for concurrent `Execute()` calls
- **Context**: Safe for concurrent read access; writes should be serialized
- **Registry**: Thread-safe resolver and function registration

```go
engine := prompty.MustNew()
tmpl, _ := engine.Parse(templateSource)

// Safe to execute from multiple goroutines
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        result, _ := tmpl.Execute(ctx, map[string]any{"id": id})
        fmt.Println(result)
    }(i)
}
wg.Wait()
```

---

## API Reference

### Engine

```go
// Creation
func New(opts ...Option) (*Engine, error)
func MustNew(opts ...Option) *Engine

// Template operations
func (e *Engine) Execute(ctx context.Context, source string, data map[string]any) (string, error)
func (e *Engine) Parse(source string) (*Template, error)
func (e *Engine) Validate(source string) (*ValidationResult, error)

// Resolver management
func (e *Engine) Register(resolver Resolver) error
func (e *Engine) MustRegister(resolver Resolver)
func (e *Engine) HasResolver(tagName string) bool
func (e *Engine) ListResolvers() []string
func (e *Engine) ResolverCount() int

// Template management
func (e *Engine) RegisterTemplate(name, source string) error
func (e *Engine) MustRegisterTemplate(name, source string)
func (e *Engine) UnregisterTemplate(name string) bool
func (e *Engine) GetTemplate(name string) (*Template, bool)
func (e *Engine) HasTemplate(name string) bool
func (e *Engine) ListTemplates() []string
func (e *Engine) TemplateCount() int

// Function management
func (e *Engine) RegisterFunc(f *Func) error
func (e *Engine) MustRegisterFunc(f *Func)
func (e *Engine) HasFunc(name string) bool
func (e *Engine) ListFuncs() []string
func (e *Engine) FuncCount() int
```

### Template

```go
func (t *Template) Execute(ctx context.Context, data map[string]any) (string, error)
func (t *Template) ExecuteWithContext(ctx context.Context, execCtx *Context) (string, error)
func (t *Template) Source() string
```

### Context

```go
// Creation
func NewContext(data map[string]any) *Context
func NewContextWithStrategy(data map[string]any, strategy ErrorStrategy) *Context

// Data access
func (c *Context) Get(path string) (any, bool)
func (c *Context) GetString(path string) string
func (c *Context) GetDefault(path string, defaultVal any) any
func (c *Context) GetStringDefault(path, defaultVal string) string
func (c *Context) Has(path string) bool
func (c *Context) Set(key string, value any)
func (c *Context) Data() map[string]any

// Hierarchy
func (c *Context) Child(data map[string]any) interface{}
func (c *Context) Parent() *Context
```

### ValidationResult

```go
func (r *ValidationResult) IsValid() bool
func (r *ValidationResult) HasErrors() bool
func (r *ValidationResult) HasWarnings() bool
func (r *ValidationResult) Issues() []ValidationIssue
func (r *ValidationResult) Errors() []ValidationIssue
func (r *ValidationResult) Warnings() []ValidationIssue
```

---

## Complete Example

A comprehensive example demonstrating multiple features:

```go
package main

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/itsatony/go-prompty"
)

func main() {
    engine := prompty.MustNew()

    // Register custom resolver
    engine.MustRegister(prompty.NewResolverFunc(
        "app.timestamp",
        func(ctx context.Context, execCtx *prompty.Context, attrs prompty.Attributes) (string, error) {
            format := attrs.GetDefault("format", time.RFC3339)
            return time.Now().Format(format), nil
        },
        nil,
    ))

    // Register custom function
    engine.MustRegisterFunc(&prompty.Func{
        Name:    "initials",
        MinArgs: 1,
        MaxArgs: 1,
        Fn: func(args []any) (any, error) {
            name, _ := args[0].(string)
            parts := strings.Fields(name)
            var initials string
            for _, p := range parts {
                if len(p) > 0 {
                    initials += strings.ToUpper(string(p[0]))
                }
            }
            return initials, nil
        },
    })

    // Register reusable templates
    engine.MustRegisterTemplate("user-badge", `[{~prompty.var name="role" default="user" /~}] {~prompty.var name="name" /~}`)

    // Main template
    template := `
{~prompty.comment~}System Prompt for Assistant{~/prompty.comment~}
Generated: {~app.timestamp format="2006-01-02 15:04" /~}

{~prompty.include template="user-badge" name="System" role="SYSTEM" /~}

You are helping {~prompty.var name="user.name" /~} ({~prompty.if eval="initials(user.name) != ''"~}initials: {~prompty.var name="user.name" /~}{~/prompty.if~}).

{~prompty.if eval="user.preferences.verbose"~}
Provide detailed explanations.
{~prompty.else~}
Be concise.
{~/prompty.if~}

User's interests:
{~prompty.for item="interest" index="i" in="user.interests" limit="5"~}
  {~prompty.var name="i" /~}. {~prompty.var name="interest" /~}
{~/prompty.for~}

Response style: {~prompty.switch eval="user.preferences.tone"~}
  {~prompty.case value="formal"~}Use formal language.{~/prompty.case~}
  {~prompty.case value="casual"~}Keep it casual and friendly.{~/prompty.case~}
  {~prompty.casedefault~}Use a balanced, professional tone.{~/prompty.casedefault~}
{~/prompty.switch~}
`

    data := map[string]any{
        "user": map[string]any{
            "name": "Alice Johnson",
            "interests": []string{"AI", "Go programming", "Cloud architecture"},
            "preferences": map[string]any{
                "verbose": false,
                "tone":    "casual",
            },
        },
    }

    result, err := engine.Execute(context.Background(), template, data)
    if err != nil {
        panic(err)
    }

    fmt.Println(result)
}
```

---

## License

MIT License - see [LICENSE](LICENSE) for details.
