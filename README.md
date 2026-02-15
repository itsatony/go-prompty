# go-prompty

**Dynamic LLM prompt templating for Go** - Build complex, maintainable AI prompts with a powerful templating engine designed for production use.

[![Go Reference](https://pkg.go.dev/badge/github.com/itsatony/go-prompty/v2.svg)](https://pkg.go.dev/github.com/itsatony/go-prompty/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/itsatony/go-prompty)](https://goreportcard.com/report/github.com/itsatony/go-prompty)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Test Coverage](https://img.shields.io/badge/coverage-88%25-brightgreen.svg)](https://github.com/itsatony/go-prompty)
[![Version](https://img.shields.io/badge/version-2.2.0-blue.svg)](https://github.com/itsatony/go-prompty/releases/tag/v2.2.0)

```yaml
---
name: enterprise-assistant
description: Context-aware enterprise support agent
type: agent
execution:
  provider: openai
  model: '{~prompty.env name="MODEL" default="gpt-4" /~}'
  temperature: 0.7
---
{~prompty.message role="system"~}
{~prompty.if eval="user.tier == 'enterprise'"~}
You are assisting {~prompty.var name="user.company" /~}, an enterprise customer.
{~prompty.else~}
You are assisting {~prompty.var name="user.name" default="a user" /~}.
{~/prompty.if~}
{~/prompty.message~}
```

## Why go-prompty?

| Challenge | Solution |
|-----------|----------|
| **Prompt sprawl** | Organize prompts as composable, versioned templates |
| **Content conflicts** | `{~...~}` delimiters avoid clashes with code, JSON, XML |
| **Dynamic content** | Variables, conditionals, loops, and expressions |
| **Reusability** | Nested templates with `prompty.include` |
| **Template inheritance** | Base templates with overridable blocks (`extends`/`block`/`parent`) |
| **Type safety** | Compile-time template validation |
| **Extensibility** | Plugin architecture for custom tags and functions |
| **Self-describing** | Embed model configuration with YAML frontmatter |
| **Environment aware** | Access env vars with `prompty.env` |
| **Conversation support** | Message tags for chat/LLM API integration |
| **Agent definitions** | Skills, tools, constraints, and catalog generation |
| **Actionable errors** | Error messages include solution hints (e.g., "use default=") |
| **Production ready** | Access control, multi-tenancy, audit logging |

## Installation

```bash
go get github.com/itsatony/go-prompty/v2
```

**CLI tool:**
```bash
go install github.com/itsatony/go-prompty/v2/cmd/prompty@latest
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/itsatony/go-prompty/v2"
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
- [Template Inheritance](#promptyextends--promptyblock--promptyparent---template-inheritance)
- [Expression Language](#expression-language)
- [Prompt Configuration](#prompt-configuration)
- [Agent Definitions & Compilation](#agent-definitions--compilation)
- [Custom Resolvers](#custom-resolvers)
- [Custom Functions](#custom-functions)
- [Storage & Persistence](#storage--persistence)
- [Deployment-Aware Versioning](#deployment-aware-versioning)
- [Access Control](#access-control)
- [Hooks System](#hooks-system)
- [Production Patterns](#production-patterns)
- [CLI Reference](#cli-reference)
- [Configuration](#configuration)
- [API Reference](#api-reference)
- [Performance](#performance)
- [Troubleshooting](#troubleshooting)
- [Examples](#examples)
- [Documentation](#documentation)

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
| **Prompt** | v2.1 prompt config with document types, skills, tools, constraints |
| **ExecutionConfig** | LLM execution parameters with provider serialization |
| **CompiledPrompt** | v2.1 agent compilation result (messages, execution, tools) |
| **DocumentResolver** | v2.1 interface for resolving prompts/skills/agents by slug |

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

### `prompty.env` - Environment Variables

Access environment variables with optional defaults.

```
{~prompty.env name="API_KEY" /~}
{~prompty.env name="MODEL" default="gpt-4" /~}
{~prompty.env name="SECRET" required="true" /~}
```

| Attribute | Required | Description |
|-----------|----------|-------------|
| `name` | Yes | Environment variable name |
| `default` | No | Fallback value if not set |
| `required` | No | Error if not set (and no default) |

### YAML Frontmatter - Prompt Configuration

Embed prompt configuration at the start of templates using YAML frontmatter. See [Prompt Configuration](#prompt-configuration) for full details.

```yaml
---
name: customer-support
description: Handles customer inquiries
type: skill
execution:
  provider: openai
  model: gpt-4
  temperature: 0.7
  max_tokens: 2048
---
```

### `prompty.message` - Conversation Messages

Define messages for chat/LLM API calls:

```
{~prompty.message role="system"~}
You are a helpful assistant.
{~/prompty.message~}

{~prompty.message role="user"~}
{~prompty.var name="query" /~}
{~/prompty.message~}
```

| Attribute | Required | Description |
|-----------|----------|-------------|
| `role` | Yes | Message role: "system", "user", "assistant", "tool" |
| `cache` | No | Cache hint for this message |

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

### `prompty.ref` - Prompt References (v2.0)

Reference and compose prompts from a registry. Enables modular prompt composition.

```
{~prompty.ref slug="greeting-prompt" /~}
{~prompty.ref slug="customer-support" version="v2" /~}
{~prompty.ref slug="my-prompt@latest" /~}
```

| Attribute | Required | Description |
|-----------|----------|-------------|
| `slug` | Yes | Prompt slug identifier (lowercase, letters/digits/hyphens) |
| `version` | No | Specific version, defaults to "latest" |

**Slug@version syntax**: `{~prompty.ref slug="greeting@v2" /~}` is equivalent to `version="v2"`.

**Requires**: A `PromptResolver` must be set on the context via `WithPromptResolver()`.

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

### `prompty.extends` / `prompty.block` / `prompty.parent` - Template Inheritance

Create reusable base templates with overridable sections. Child templates can extend parents and selectively override blocks while optionally preserving parent content.

**Base Template** (`base-prompt.prompty`):
```
{~prompty.block name="system"~}You are a helpful assistant.{~/prompty.block~}

{~prompty.block name="context"~}{~/prompty.block~}

{~prompty.block name="instructions"~}Please be concise and accurate.{~/prompty.block~}
```

**Child Template** (extends base):
```
{~prompty.extends template="base-prompt" /~}

{~prompty.block name="system"~}You are a customer support agent for {~prompty.var name="company" /~}.{~/prompty.block~}

{~prompty.block name="context"~}
Customer: {~prompty.var name="customer.name" /~}
Issue: {~prompty.var name="issue" /~}
{~/prompty.block~}

{~prompty.block name="instructions"~}
{~prompty.parent /~}
Additionally, always offer to escalate if the customer is frustrated.
{~/prompty.block~}
```

**Result** (when executed):
```
You are a customer support agent for Acme Corp.

Customer: Alice
Issue: Billing question

Please be concise and accurate.
Additionally, always offer to escalate if the customer is frustrated.
```

| Tag | Description |
|-----|-------------|
| `prompty.extends` | Inherit from a parent template (must be first tag) |
| `prompty.block` | Define an overridable named section |
| `prompty.parent` | Insert parent's block content (call super) |

| Attribute | Tag | Required | Description |
|-----------|-----|----------|-------------|
| `template` | extends | Yes | Name of registered parent template |
| `name` | block | Yes | Unique block identifier |

**Multi-level inheritance** is supported (A extends B extends C), with blocks resolved from most-derived to base.

```go
engine := prompty.MustNew()

// Register base templates
engine.MustRegisterTemplate("base-layout", `
{~prompty.block name="header"~}Default Header{~/prompty.block~}
{~prompty.block name="body"~}{~/prompty.block~}
{~prompty.block name="footer"~}Default Footer{~/prompty.block~}
`)

engine.MustRegisterTemplate("page-layout", `
{~prompty.extends template="base-layout" /~}
{~prompty.block name="header"~}== {~prompty.var name="title" /~} =={~/prompty.block~}
`)

// Execute child that extends page-layout
result, _ := engine.Execute(ctx, `
{~prompty.extends template="page-layout" /~}
{~prompty.block name="body"~}Page content here{~/prompty.block~}
`, map[string]any{"title": "My Page"})
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

### Built-in Functions (37 total)

<details>
<summary><strong>String Functions (11)</strong></summary>

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
<summary><strong>Collection Functions (6)</strong></summary>

| Function | Description |
|----------|-------------|
| `len(x)` | Length of string/slice/map |
| `first(slice)` | First element |
| `last(slice)` | Last element |
| `keys(map)` | Map keys (sorted) |
| `values(map)` | Map values |
| `has(map, key)` | Check map has key |

</details>

<details>
<summary><strong>Type Functions (5)</strong></summary>

| Function | Description |
|----------|-------------|
| `toString(x)` | Convert to string |
| `toInt(x)` | Convert to integer |
| `toFloat(x)` | Convert to float |
| `toBool(x)` | Convert to boolean |
| `typeOf(x)` | Get type name |

</details>

<details>
<summary><strong>Utility Functions (2)</strong></summary>

| Function | Description |
|----------|-------------|
| `default(x, fallback)` | Return fallback if x is nil/empty |
| `coalesce(a, b, ...)` | First non-nil, non-empty value |

</details>

<details>
<summary><strong>Date/Time Functions (13)</strong></summary>

| Function | Description |
|----------|-------------|
| `now()` | Current timestamp |
| `formatDate(t, layout)` | Format time using Go layout |
| `parseDate(s, [layout])` | Parse string to time (auto-detects format if no layout) |
| `addDays(t, n)` | Add n days to time |
| `addHours(t, n)` | Add n hours to time |
| `addMinutes(t, n)` | Add n minutes to time |
| `diffDays(t1, t2)` | Days between two times (t2 - t1) |
| `year(t)` | Extract year |
| `month(t)` | Extract month (1-12) |
| `day(t)` | Extract day of month (1-31) |
| `weekday(t)` | Day name (Monday, Tuesday, etc.) |
| `isAfter(t1, t2)` | True if t1 is after t2 |
| `isBefore(t1, t2)` | True if t1 is before t2 |

**Common date format layouts:**
- `2006-01-02` - ISO date (YYYY-MM-DD)
- `01/02/2006` - US format (MM/DD/YYYY)
- `02/01/2006` - EU format (DD/MM/YYYY)
- `2006-01-02T15:04:05Z07:00` - ISO datetime with timezone
- `15:04:05` - 24-hour time
- `Jan 2, 2006` - Human-readable

</details>

### Expression Examples

```
// Simple comparison
{~prompty.if eval="age >= 18"~}

// Function calls
{~prompty.if eval="len(trim(input)) > 0"~}

// Complex logic
{~prompty.if eval="(isAdmin || isModerator) && !isBanned && len(permissions) > 0"~}

// Date/time operations
{~prompty.var name="today" default="{~prompty.var name=\"formatDate(now(), '2006-01-02')\" /~}" /~}
{~prompty.if eval="isAfter(expiryDate, now())"~}Still valid{~/prompty.if~}
{~prompty.if eval="diffDays(startDate, now()) > 30"~}Over a month{~/prompty.if~}
```

---

## Prompt Configuration

Templates can embed prompt configuration using YAML frontmatter with `---` delimiters. This makes templates self-describing with execution parameters, input/output schemas, and sample data.

### YAML Frontmatter Format

```yaml
---
name: customer-support-agent
description: Handles customer inquiries
type: skill
execution:
  provider: openai
  model: '{~prompty.env name="MODEL_NAME" default="gpt-4" /~}'
  temperature: 0.7
  max_tokens: 2048
  top_p: 0.9
inputs:
  customer_name:
    type: string
    required: true
  query:
    type: string
    required: true
sample:
  customer_name: Alice
  query: How do I reset my password?
---
{~prompty.message role="system"~}
You are a helpful customer support agent.
{~/prompty.message~}

{~prompty.message role="user"~}
Customer: {~prompty.var name="customer_name" /~}
Query: {~prompty.var name="query" /~}
{~/prompty.message~}
```

**IMPORTANT:** Use YAML single quotes for values containing prompty tags to avoid escaping issues.

### Accessing Configuration

```go
tmpl, _ := engine.Parse(source)

if tmpl.HasPrompt() {
    prompt := tmpl.Prompt()

    fmt.Println("Name:", prompt.Name)
    fmt.Println("Description:", prompt.Description)

    // Access execution config
    if prompt.Execution != nil {
        fmt.Println("Provider:", prompt.Execution.Provider)
        fmt.Println("Model:", prompt.Execution.Model)
    }

    // Validate inputs before execution
    if err := prompt.ValidateInputs(data); err != nil {
        log.Fatal("Invalid inputs:", err)
    }
}
```

### Extracting Messages

Execute and extract structured messages for LLM API calls:

```go
messages, err := tmpl.ExecuteAndExtractMessages(ctx, data)
// messages is []prompty.Message{{Role: "system", Content: "..."}, ...}

// Use with your LLM client
for _, msg := range messages {
    fmt.Printf("[%s]: %s\n", msg.Role, msg.Content)
}
```

### Execution Parameters

Parameters are set directly in the `execution:` block:

| Parameter | Type | Range | Providers | Description |
|-----------|------|-------|-----------|-------------|
| `provider` | string | — | all | Provider: openai, anthropic, google, vllm, azure, mistral, cohere |
| `model` | string | — | all | Model name (e.g., gpt-4, claude-3-sonnet) |
| `temperature` | float | [0.0, 2.0] | all | Sampling temperature |
| `max_tokens` | int | > 0 | all | Maximum tokens to generate |
| `top_p` | float | [0.0, 1.0] | all | Nucleus sampling |
| `top_k` | int | >= 0 | all | Top-k sampling |
| `stop_sequences` | []string | — | all | Stop sequences |
| `min_p` | float | [0.0, 1.0] | vLLM | Minimum probability sampling |
| `repetition_penalty` | float | > 0.0 | vLLM | Repetition penalty multiplier |
| `seed` | int | any | OpenAI, Anthropic, vLLM | Deterministic sampling seed |
| `logprobs` | int | [0, 20] | OpenAI, vLLM | Number of log probabilities to return |
| `stop_token_ids` | []int | each >= 0 | vLLM | Token IDs that trigger stop |
| `logit_bias` | map | values [-100, 100] | OpenAI, vLLM | Token logit bias adjustments |
| `response_format` | object | — | all | Structured output format |
| `thinking` | object | — | Anthropic | Extended thinking configuration |
| `guided_decoding` | object | — | vLLM | Guided decoding constraints |

**Legacy Reference:** See [docs/INFERENCE_CONFIG.md](docs/INFERENCE_CONFIG.md) for v1 configuration documentation (deprecated in v2.1).

### v2.1 Prompt Configuration (Agent Skills)

v2.1 extends the `Prompt` type compatible with [Agent Skills](https://agentskills.io) specification, adding agent definitions with skills, tools, constraints, and compilation:

```yaml
---
name: customer-support-agent
description: Handles customer inquiries with context awareness
license: MIT
compatibility: gpt-4,claude-3

execution:
  provider: openai
  model: gpt-4
  temperature: 0.7
  max_tokens: 2048
  thinking:
    enabled: true
    budget_tokens: 5000

skope:
  visibility: public
  project_id: proj_helpdesk
  regions: [us-east-1, eu-west-1]
  projects: [support, helpdesk]

inputs:
  query:
    type: string
    required: true
---
{~prompty.message role="system"~}
You are a helpful support agent.
{~/prompty.message~}
```

**v2.1 Detection**: Templates with `execution`, `skope`, `type`, or `name` config are v2.1 Prompts.

```go
tmpl, _ := engine.Parse(source)

if tmpl.HasPrompt() {
    prompt := tmpl.Prompt()

    fmt.Println("Name:", prompt.Name)
    fmt.Println("Description:", prompt.Description)

    // Access execution config
    if prompt.Execution != nil {
        fmt.Println("Provider:", prompt.Execution.Provider)

        // Provider-specific format conversion
        openAIParams := prompt.Execution.ToOpenAI()
        anthropicParams := prompt.Execution.ToAnthropic()
    }

    // Validate inputs
    if err := prompt.ValidateInputs(data); err != nil {
        log.Fatal("Invalid inputs:", err)
    }
}
```

**Key v2.1 Types**:
- `Prompt`: Full prompt config with document type (prompt/skill/agent), skills, tools, constraints, messages
- `ExecutionConfig`: LLM parameters with provider-specific conversion and `Merge()` for 3-layer precedence
- `SkopeConfig`: Platform integration (visibility, projects, project_id, regions, versioning)
- `SkillRef`, `ToolsConfig`, `ConstraintsConfig`: Agent-specific configuration
- `CompiledPrompt`: Result of `CompileAgent()` — messages, execution config, tools, constraints
- `DocumentResolver`: Interface for resolving prompts/skills/agents by slug

---

## Agent Definitions & Compilation

v2.1 introduces full agent support: define agents with skills, tools, constraints, and message templates, then compile them into structured output ready for LLM APIs.

### Document Types

| Type | Description | Skills | Tools | Constraints |
|------|-------------|--------|-------|-------------|
| `prompt` | Simple prompt template | No | No | No |
| `skill` | Reusable capability (default) | No | Yes | Yes |
| `agent` | Full agent definition | Yes | Yes | Yes |

### Defining an Agent

```yaml
---
name: research-agent
description: AI research assistant
type: agent
execution:
  provider: openai
  model: gpt-4
  temperature: 0.3
skills:
  - slug: web-search
    injection: system_prompt
  - slug: summarizer
    injection: user_context
tools:
  functions:
    - name: search_web
      description: Search the web for information
      parameters:
        type: object
        properties:
          query: {type: string}
        required: [query]
constraints:
  behavioral:
    - Always cite sources
  safety:
    - Never fabricate references
context:
  company: Acme Corp
messages:
  - role: system
    content: |
      You are a research assistant for {~prompty.var name="context.company" /~}.
      {~prompty.skills_catalog format="detailed" /~}
  - role: user
    content: '{~prompty.var name="input.query" /~}'
---
```

### Compiling an Agent

```go
// Parse the agent definition
agent, _ := prompty.Parse([]byte(agentYAML))

// Set up a resolver for skill references
resolver := prompty.NewMapDocumentResolver()
resolver.AddSkill("web-search", &prompty.Prompt{
    Name:        "web-search",
    Description: "Searches the web for information",
    Type:        prompty.DocumentTypeSkill,
    Body:        "Use search tools to find relevant information.",
})

// Compile the agent
compiled, _ := agent.CompileAgent(ctx, map[string]any{
    "query": "Latest quantum computing advances",
}, &prompty.CompileOptions{
    Resolver:            resolver,
    SkillsCatalogFormat: prompty.CatalogFormatDetailed,
})

// Access compiled output
for _, msg := range compiled.Messages {
    fmt.Printf("[%s]: %s\n", msg.Role, msg.Content)
}
fmt.Println("Model:", compiled.Execution.Model)
```

### DocumentResolver

`DocumentResolver` resolves prompts, skills, and agents by slug during compilation:

```go
type DocumentResolver interface {
    ResolvePrompt(ctx context.Context, slug string) (*Prompt, error)
    ResolveSkill(ctx context.Context, ref string) (*Prompt, error)
    ResolveAgent(ctx context.Context, slug string) (*Prompt, error)
}
```

**Built-in implementations:**

| Resolver | Description |
|----------|-------------|
| `MapDocumentResolver` | In-memory map for testing and simple cases (thread-safe) |
| `StorageDocumentResolver` | Backed by any `TemplateStorage` (memory, filesystem, PostgreSQL) |
| `NoopDocumentResolver` | Always returns errors (default when no resolver configured) |

### Skill Activation

Activate a specific skill within a compiled agent. The skill body is resolved, compiled, and injected into messages based on the injection mode:

```go
compiled, _ := agent.ActivateSkill(ctx, "web-search", input, &prompty.CompileOptions{
    Resolver: resolver,
})
// Skill content is injected into the system prompt (or user context)
```

**Injection modes:**
- `system_prompt` — Appends skill content to the system message
- `user_context` — Adds skill content as a user message
- `none` — No automatic injection

### Catalog Generation

Generate catalogs of available skills and tools in multiple formats:

```go
// Skills catalog
catalog, _ := prompty.GenerateSkillsCatalog(ctx, agent.Skills, resolver, prompty.CatalogFormatDetailed)

// Tools catalog (JSON schema for function calling)
toolsCatalog, _ := prompty.GenerateToolsCatalog(agent.Tools, prompty.CatalogFormatFunctionCalling)
```

**Catalog formats:**

| Format | Description |
|--------|-------------|
| `""` (default) | Markdown bullet list |
| `"detailed"` | Full descriptions, parameters, injection modes |
| `"compact"` | Single-line, semicolon-separated |
| `"function_calling"` | JSON schema for OpenAI-style tool use (tools only) |

Use catalog tags inside agent message templates:
```
{~prompty.skills_catalog format="detailed" /~}
{~prompty.tools_catalog format="function_calling" /~}
```

### Execution Config Merging

`CompileOptions` supports 3-layer precedence for execution config: agent definition → skill override → runtime input. Use `ExecutionConfig.Merge()` for manual merging:

```go
// Base config from agent
base := agent.Execution

// Skill-specific overrides
skillOverride := &prompty.ExecutionConfig{Temperature: floatPtr(0.1)}

// Merged: skill values override base where set
effective := base.Merge(skillOverride)
```

### Provider Message Serialization

Convert compiled messages to LLM provider-specific formats for direct API submission:

```go
compiled, _ := agent.CompileAgent(ctx, input, opts)

// OpenAI: []map[string]any with role/content
openAIMsgs := compiled.ToOpenAIMessages()

// Anthropic: {system: "...", messages: [...]} (system extracted)
anthropicPayload := compiled.ToAnthropicMessages()

// Gemini: {system_instruction: {...}, contents: [...]} (assistant → model)
geminiPayload := compiled.ToGeminiContents()

// Or auto-dispatch by provider name
msgs, _ := compiled.ToProviderMessages("openai")
```

### Agent Validation

Use `ValidateAsAgent()` before compilation to catch configuration issues early:

```go
if err := agent.ValidateAsAgent(); err != nil {
    log.Fatal("Agent config invalid:", err)
}
```

This checks: agent type, execution config with provider/model, and body or messages present.

**Deep Dive:** See [examples/agent_compilation](examples/agent_compilation/), [examples/document_resolver](examples/document_resolver/), and [examples/catalog_generation](examples/catalog_generation/).

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

## Storage & Persistence

go-prompty includes a pluggable storage layer for managing templates with versioning, metadata, and multi-tenant support:

- **Built-in drivers**: Memory (testing), Filesystem (persistent), and PostgreSQL (production)
- **Custom backends**: Implement `TemplateStorage` for MongoDB, Redis, or other databases
- **Caching**: Automatic caching wrapper for any storage backend
- **PromptConfig persistence**: Prompt configuration is automatically extracted and stored

```go
// PostgreSQL storage (production-ready with migrations)
storage, _ := prompty.OpenStorage("postgres",
    "postgres://user:pass@localhost/prompty?sslmode=disable")

// Or with full configuration
storage, _ := prompty.NewPostgresStorage(prompty.PostgresConfig{
    ConnectionString: os.Getenv("DATABASE_URL"),
    AutoMigrate:      true,
})

engine, _ := prompty.NewStorageEngine(prompty.StorageEngineConfig{
    Storage: storage,
})

// Save with versioning - PromptConfig automatically extracted
engine.Save(ctx, &prompty.StoredTemplate{
    Name:   "greeting",
    Source: "---\nname: greeting\nexecution:\n  model: gpt-4\n---\nHello {~prompty.var name=\"user\" /~}!",
    Tags:   []string{"production"},
})

// Execute
result, _ := engine.Execute(ctx, "greeting", map[string]any{"user": "Alice"})

// Retrieved templates include PromptConfig
tmpl, _ := engine.Get(ctx, "greeting")
if tmpl.PromptConfig != nil && tmpl.PromptConfig.Execution != nil {
    fmt.Println("Model:", tmpl.PromptConfig.Execution.Model)
}
```

**Deep Dive:** See [docs/STORAGE.md](docs/STORAGE.md) for architecture (including PostgreSQL) and [docs/CUSTOM_STORAGE.md](docs/CUSTOM_STORAGE.md) for implementing custom backends.

---

## Deployment-Aware Versioning

go-prompty supports deployment-aware versioning with **labels** and **status** tracking for production workflows.

### Labels

Named pointers to specific template versions:

```go
// Set labels for deployment stages
engine.SetLabel(ctx, "greeting", "staging", 2)
engine.SetLabel(ctx, "greeting", "production", 1)

// Execute by label instead of version number
result, _ := engine.ExecuteLabeled(ctx, "greeting", "production", data)

// Convenience method for production
result, _ := engine.ExecuteProduction(ctx, "greeting", data)

// Promote staging to production
engine.PromoteToProduction(ctx, "greeting", 2)

// List all labels for a template
labels, _ := engine.ListLabels(ctx, "greeting")
// []*TemplateLabel{{Label: "production", Version: 2}, {Label: "staging", Version: 2}}
```

**Reserved labels**: `production`, `staging`, `canary`

**Custom labels**: lowercase alphanumeric with hyphens/underscores (e.g., `beta-test`, `a_b_testing`)

### Deployment Status

Lifecycle states for template versions:

| Status | Description | Transitions To |
|--------|-------------|----------------|
| `draft` | Not yet active, needs review | `active`, `archived` |
| `active` | In use (default for new templates) | `deprecated`, `archived` |
| `deprecated` | Scheduled for removal | `active`, `archived` |
| `archived` | Read-only, terminal state | (none) |

```go
// Set status
engine.SetStatus(ctx, "greeting", 1, prompty.DeploymentStatusDeprecated)

// Query by status
deprecated, _ := engine.ListByStatus(ctx, prompty.DeploymentStatusDeprecated, nil)

// Get version history with labels and status
history, _ := engine.GetVersionHistory(ctx, "greeting")
for _, v := range history.Versions {
    fmt.Printf("v%d: status=%s labels=%v\n", v.Version, v.Status, v.Labels)
}
```

### Rollback and Clone Behavior

- `RollbackToVersion()` creates a new version with `draft` status (requires review before activation)
- `CloneVersion()` creates a new template with `draft` status

**Deep Dive:** See [docs/STORAGE.md](docs/STORAGE.md#deployment-aware-versioning) for complete documentation.

---

## Access Control

go-prompty provides a flexible, unopinionated access control system with RBAC support, multi-tenant isolation, and audit logging.

### SecureStorageEngine

```go
// Create secure engine with access control
engine, _ := prompty.NewSecureStorageEngine(prompty.SecureStorageEngineConfig{
    StorageEngineConfig: prompty.StorageEngineConfig{
        Storage: storage,
    },
    AccessChecker: &MyRBACChecker{},
    Auditor:       prompty.NewMemoryAuditor(1000),
})

// Create subject from authenticated user
subject := prompty.NewAccessSubject("usr_123").
    WithTenant("org_456").
    WithRoles("editor", "viewer")

// All operations require subject
result, err := engine.ExecuteSecure(ctx, "greeting", data, subject)
tmpl, err := engine.GetSecure(ctx, "greeting", subject)
err := engine.SaveSecure(ctx, tmpl, subject)
```

### Built-in Checkers

| Checker | Description |
|---------|-------------|
| `AllowAllChecker` | Allows all access (development) |
| `DenyAllChecker` | Denies all access (maintenance) |
| `TenantChecker` | Enforces tenant isolation |
| `RoleChecker` | Requires specific roles |
| `ChainedChecker` | AND logic (all must allow) |
| `AnyOfChecker` | OR logic (any can allow) |
| `CachedChecker` | Caches decisions for performance |

### RBAC Example

```go
type RBACChecker struct {
    rolePermissions map[string][]string
}

func (c *RBACChecker) Check(ctx context.Context, req *prompty.AccessRequest) (*prompty.AccessDecision, error) {
    for _, role := range req.Subject.Roles {
        if perms := c.rolePermissions[role]; contains(perms, string(req.Operation)) {
            return prompty.Allow("granted by role " + role), nil
        }
    }
    return prompty.Deny("no permission"), nil
}
```

**Deep Dive:** See [docs/ACCESS_CONTROL.md](docs/ACCESS_CONTROL.md) for complete documentation.

---

## Hooks System

Hooks provide extension points at every operation stage for logging, metrics, validation, and custom logic.

### Hook Points

| Hook Point | When Called |
|------------|-------------|
| `HookBeforeLoad` | Before loading a template |
| `HookAfterLoad` | After loading a template |
| `HookBeforeExecute` | Before executing a template |
| `HookAfterExecute` | After executing a template |
| `HookBeforeSave` | Before saving a template |
| `HookAfterSave` | After saving a template |
| `HookBeforeDelete` | Before deleting a template |
| `HookAfterDelete` | After deleting a template |
| `HookBeforeValidate` | Before validating a template |
| `HookAfterValidate` | After validating a template |

### Registering Hooks

```go
// Register a logging hook
engine.RegisterHook(prompty.HookBeforeExecute, func(ctx context.Context, point HookPoint, data *HookData) error {
    log.Printf("Executing: %s by %s", data.TemplateName, data.Subject.ID)
    return nil
})

// Before hooks can abort operations by returning an error
engine.RegisterHook(prompty.HookBeforeSave, func(ctx context.Context, point HookPoint, data *HookData) error {
    if len(data.Template.Source) > 100000 {
        return errors.New("template too large")
    }
    return nil
})
```

### Built-in Hooks

```go
// Logging hook
hook := prompty.LoggingHook(logFunc)

// Timing hook
hook, getElapsed := prompty.TimingHook()

// Access check hook
hook := prompty.AccessCheckHook(checker)

// Audit hook
hook := prompty.AuditHook(auditor)
```

**Deep Dive:** See [docs/ACCESS_CONTROL.md](docs/ACCESS_CONTROL.md) for hooks documentation.

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

**Deep Dive:** See [docs/ERROR_STRATEGIES.md](docs/ERROR_STRATEGIES.md) for detailed examples.

### Debugging Templates

Use DryRun and Explain for template debugging:

```go
tmpl, _ := engine.Parse(source)

// DryRun - validate without execution
result := tmpl.DryRun(ctx, data)
fmt.Println(result.MissingVariables)  // Variables not in data
fmt.Println(result.UnusedVariables)   // Data not used in template
fmt.Println(result.Warnings)          // Potential issues

// Explain - detailed execution analysis
explain := tmpl.Explain(ctx, data)
fmt.Println(explain.AST)              // AST structure
fmt.Println(explain.Variables)        // All variable accesses
fmt.Println(explain.Timing)           // Execution timing
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

# Strict mode (warnings become errors)
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

## Media Generation (v2.5)

ExecutionConfig supports multimodal AI generation via nested config structs for image, audio, embedding, and more:

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

```yaml
---
name: embedder
execution:
  modality: embedding
  provider: openai
  model: text-embedding-3-small
  embedding:
    dimensions: 1536
    format: float
  streaming:
    enabled: true
    method: sse
---
```

Supported modalities: `text`, `image`, `audio_speech`, `audio_transcription`, `music`, `sound_effects`, `embedding`.

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
func (t *Template) TemplateBody() string                    // Source without config block
func (t *Template) HasPrompt() bool                         // Check for Prompt config
func (t *Template) Prompt() *Prompt                         // Get Prompt config
func (t *Template) DryRun(ctx context.Context, data map[string]any) *DryRunResult
func (t *Template) Explain(ctx context.Context, data map[string]any) *ExplainResult
```

</details>

<details>
<summary><strong>Prompt (v2.1)</strong></summary>

```go
// Prompt configuration (Agent Skills compatible)
type Prompt struct {
    Name          string                // Required: slug format
    Description   string                // Required: max 1024 chars
    Type          DocumentType          // v2.1: "prompt", "skill" (default), "agent"
    License       string                // Optional: MIT, Apache-2.0, etc.
    Compatibility string                // Optional: compatible models
    Metadata      map[string]any        // Optional: custom metadata
    Inputs        map[string]*InputDef  // Optional: input schema
    Outputs       map[string]*OutputDef // Optional: output schema
    Sample        map[string]any        // Optional: sample data
    Execution     *ExecutionConfig      // LLM execution config
    Skope         *SkopeConfig          // Platform config
    Skills        []SkillRef            // v2.1: Agent skill references
    Tools         *ToolsConfig          // v2.1: Tool definitions
    Context       map[string]any        // v2.1: Agent context data
    Constraints   *ConstraintsConfig    // v2.1: Agent constraints
    Messages      []MessageTemplate     // v2.1: Message templates
    Body          string                // Template body (after frontmatter)
}

func (p *Prompt) Validate() error
func (p *Prompt) ValidateInputs(data map[string]any) error
func (p *Prompt) GetSlug() string
func (p *Prompt) Clone() *Prompt
func (p *Prompt) IsAgent() bool
func (p *Prompt) IsSkill() bool
func (p *Prompt) IsPrompt() bool
func (p *Prompt) EffectiveType() DocumentType
func (p *Prompt) Compile(ctx context.Context, input map[string]any, opts CompileOptions) (string, error)
func (p *Prompt) CompileAgent(ctx context.Context, input map[string]any, opts CompileOptions) (*CompiledPrompt, error)
func (p *Prompt) ActivateSkill(ctx context.Context, skillSlug string, input map[string]any, opts CompileOptions) (*CompiledPrompt, error)
func (p *Prompt) ValidateForExecution() error
func (p *Prompt) ValidateAsAgent() error
func (p *Prompt) AgentDryRun(ctx context.Context, opts *CompileOptions) *AgentDryRunResult
func (p *Prompt) IsAgentSkillsCompatible() bool
func (p *Prompt) StripExtensions() *Prompt
func (p *Prompt) ExportToSkillMD(body string) (string, error)
```

</details>

<details>
<summary><strong>ExecutionConfig (v2.7)</strong></summary>

```go
type ExecutionConfig struct {
    Provider          string              // openai, anthropic, google, vllm, azure, mistral, cohere
    Model             string              // Model name
    Temperature       *float64            // 0.0-2.0
    MaxTokens         *int                // Max output tokens
    TopP              *float64            // Nucleus sampling
    TopK              *int                // Top-k sampling
    StopSequences     []string            // Stop sequences
    MinP              *float64            // v2.3: Min-p sampling [0.0, 1.0] (vLLM)
    RepetitionPenalty *float64            // v2.3: Repetition penalty > 0.0 (vLLM)
    Seed              *int                // v2.3: Deterministic seed (OpenAI, Anthropic, vLLM)
    Logprobs          *int                // v2.3: Log probabilities [0, 20] (OpenAI, vLLM)
    StopTokenIDs      []int               // v2.3: Stop token IDs (vLLM)
    LogitBias         map[string]float64  // v2.3: Logit bias [-100, 100] (OpenAI, vLLM)
    Thinking          *ThinkingConfig     // Claude extended thinking
    ResponseFormat    *ResponseFormat     // Structured output
    GuidedDecoding    *GuidedDecoding     // vLLM guided decoding
    Embedding         *EmbeddingConfig    // v2.7: Extended embedding params
    ProviderOptions   map[string]any      // Provider-specific options
}

func (c *ExecutionConfig) Validate() error
func (c *ExecutionConfig) Clone() *ExecutionConfig
func (c *ExecutionConfig) Merge(other *ExecutionConfig) *ExecutionConfig  // v2.1: 3-layer precedence merge
func (c *ExecutionConfig) GetTemperature() (float64, bool)
func (c *ExecutionConfig) GetMaxTokens() (int, bool)
func (c *ExecutionConfig) GetMinP() (float64, bool)                      // v2.3
func (c *ExecutionConfig) GetRepetitionPenalty() (float64, bool)         // v2.3
func (c *ExecutionConfig) GetSeed() (int, bool)                         // v2.3
func (c *ExecutionConfig) GetLogprobs() (int, bool)                     // v2.3
func (c *ExecutionConfig) GetStopTokenIDs() []int                       // v2.3
func (c *ExecutionConfig) GetLogitBias() map[string]float64             // v2.3
func (c *ExecutionConfig) HasThinking() bool
func (c *ExecutionConfig) ToOpenAI() map[string]any
func (c *ExecutionConfig) ToAnthropic() map[string]any
func (c *ExecutionConfig) ToGemini() map[string]any
func (c *ExecutionConfig) ToVLLM() map[string]any
func (c *ExecutionConfig) ToMistral() map[string]any                     // v2.7
func (c *ExecutionConfig) ToCohere() map[string]any                      // v2.7
func (c *ExecutionConfig) ProviderFormat(provider string) (map[string]any, error)
func (c *ExecutionConfig) GetEffectiveProvider() string
```

</details>

<details>
<summary><strong>DocumentResolver (v2.1)</strong></summary>

```go
type DocumentResolver interface {
    ResolvePrompt(ctx context.Context, slug string) (*Prompt, error)
    ResolveSkill(ctx context.Context, ref string) (*Prompt, error)
    ResolveAgent(ctx context.Context, slug string) (*Prompt, error)
}

func NewMapDocumentResolver() *MapDocumentResolver
func NewStorageDocumentResolver(storage TemplateStorage) *StorageDocumentResolver
```

</details>

<details>
<summary><strong>CompiledPrompt (v2.1)</strong></summary>

```go
type CompiledPrompt struct {
    Messages    []CompiledMessage
    Execution   *ExecutionConfig
    Tools       *ToolsConfig
    Constraints *OperationalConstraints
}

type CompiledMessage struct {
    Role    string
    Content string
    Cache   bool
}

// Provider message serialization
func (cp *CompiledPrompt) ToOpenAIMessages() []map[string]any
func (cp *CompiledPrompt) ToAnthropicMessages() map[string]any
func (cp *CompiledPrompt) ToGeminiContents() map[string]any
func (cp *CompiledPrompt) ToProviderMessages(provider string) (any, error)

// Functional options
func NewCompileOptions(options ...CompileOption) *CompileOptions
func WithResolver(r DocumentResolver) CompileOption
func WithCompileEngine(e *Engine) CompileOption
func WithSkillsCatalogFormat(f CatalogFormat) CompileOption
func WithToolsCatalogFormat(f CatalogFormat) CompileOption
```

</details>

<details>
<summary><strong>TemplateRunner Interface</strong></summary>

```go
// Common interface for resolver management shared by Engine and StorageEngine
type TemplateRunner interface {
    RegisterResolver(r Resolver) error
    HasResolver(tagName string) bool
    ListResolvers() []string
    ResolverCount() int
}

// Both Engine and StorageEngine satisfy TemplateRunner,
// allowing generic code to work with either:
func configureRunner(runner prompty.TemplateRunner) {
    runner.RegisterResolver(myCustomResolver)
}
```

</details>

<details>
<summary><strong>AgentExecutor</strong></summary>

```go
// High-level wrapper: parse → validate → compile in one call
func NewAgentExecutor(options ...AgentExecutorOption) *AgentExecutor

// Functional options
func WithAgentResolver(r DocumentResolver) AgentExecutorOption
func WithAgentEngine(e *Engine) AgentExecutorOption
func WithAgentSkillsCatalogFormat(f CatalogFormat) AgentExecutorOption
func WithAgentToolsCatalogFormat(f CatalogFormat) AgentExecutorOption

// Methods
func (ae *AgentExecutor) Execute(ctx context.Context, source string, input map[string]any) (*CompiledPrompt, error)
func (ae *AgentExecutor) ExecuteFile(ctx context.Context, path string, input map[string]any) (*CompiledPrompt, error)
func (ae *AgentExecutor) ExecutePrompt(ctx context.Context, prompt *Prompt, input map[string]any) (*CompiledPrompt, error)
func (ae *AgentExecutor) ActivateSkill(ctx context.Context, source string, skillSlug string, input map[string]any, runtimeExec *ExecutionConfig) (*CompiledPrompt, error)
```

Example:
```go
executor := prompty.NewAgentExecutor(
    prompty.WithAgentResolver(myResolver),
    prompty.WithAgentSkillsCatalogFormat(prompty.CatalogFormatDetailed),
)
compiled, err := executor.Execute(ctx, agentYAML, map[string]any{"query": "hello"})
```

</details>

<details>
<summary><strong>Agent Dry Run</strong></summary>

```go
// Validate all refs and templates without producing output
func (p *Prompt) AgentDryRun(ctx context.Context, opts *CompileOptions) *AgentDryRunResult

type AgentDryRunResult struct {
    Issues         []AgentDryRunIssue
    SkillsResolved int
    ToolsDefined   int
    MessageCount   int
}

func (r *AgentDryRunResult) OK() bool
func (r *AgentDryRunResult) HasErrors() bool
func (r *AgentDryRunResult) String() string
```

Example:
```go
prompt, _ := prompty.Parse(agentYAML)
result := prompt.AgentDryRun(ctx, &prompty.CompileOptions{Resolver: myResolver})
if result.HasErrors() {
    fmt.Println(result.String()) // Shows all issues
}
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

// v2.0: Prompt reference support
func (c *Context) WithPromptResolver(resolver PromptBodyResolver) *Context
func (c *Context) WithRefDepth(depth int) *Context
func (c *Context) WithRefChain(chain []string) *Context
func (c *Context) PromptResolver() interface{}
func (c *Context) RefDepth() int
func (c *Context) RefChain() []string
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
| Parse (small) | ~15us | ~20 |
| Parse (medium) | ~80us | ~100 |
| Execute (simple) | ~5us | ~10 |
| Execute (complex) | ~50us | ~80 |

### Optimization Tips

1. **Parse once, execute many** - Cache parsed templates
2. **Limit loop iterations** - Use `limit` attribute
3. **Avoid deep nesting** - Keep template depth reasonable
4. **Use simple expressions** - Complex expressions add overhead

**Deep Dive:** See [docs/PERFORMANCE.md](docs/PERFORMANCE.md) for detailed benchmarks.

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

**Deep Dive:** See [docs/COMMON_PITFALLS.md](docs/COMMON_PITFALLS.md) for more troubleshooting tips.

---

## Examples

| Example | Description |
|---------|-------------|
| [basic](examples/basic/) | Variable interpolation and defaults |
| [conditionals](examples/conditionals/) | If/else branching |
| [loops](examples/loops/) | For loop iteration |
| [inheritance](examples/inheritance/) | Template inheritance with extends/block/parent |
| [custom_functions](examples/custom_functions/) | Registering custom expression functions |
| [custom_resolver](examples/custom_resolver/) | Creating custom tag handlers |
| [agent_compilation](examples/agent_compilation/) | v2.1 agent definition, compilation, and skill activation |
| [document_resolver](examples/document_resolver/) | v2.1 DocumentResolver implementations (Map, Storage, Noop) |
| [catalog_generation](examples/catalog_generation/) | v2.1 catalog generation in all formats (default, detailed, compact, function_calling) |
| [prompt_import_export](examples/prompt_import_export/) | v2.1 prompt serialization, import/export, zip archives |
| [storage](examples/storage/) | Template storage and versioning |
| [storage_persistence](examples/storage_persistence/) | Filesystem storage |
| [access_rbac](examples/access_rbac/) | RBAC access control |
| [access_tenant](examples/access_tenant/) | Multi-tenant isolation |
| [error_handling](examples/error_handling/) | All 5 error strategies |
| [debugging](examples/debugging/) | DryRun and Explain features |
| [custom_storage_postgres](examples/custom_storage_postgres/) | PostgreSQL backend (now built-in) |

---

## Documentation

| Guide | Description |
|-------|-------------|
| [MIGRATION_V2.1.md](docs/MIGRATION_V2.1.md) | Migration guide from v1.x/v2.0 to v2.1 |
| [INFERENCE_CONFIG.md](docs/INFERENCE_CONFIG.md) | Legacy v1 model configuration reference (deprecated in v2.1) |
| [STORAGE.md](docs/STORAGE.md) | Storage architecture, versioning, PostgreSQL |
| [CUSTOM_STORAGE.md](docs/CUSTOM_STORAGE.md) | Implementing custom backends |
| [ACCESS_CONTROL.md](docs/ACCESS_CONTROL.md) | RBAC, multi-tenancy, audit logging |
| [ERROR_STRATEGIES.md](docs/ERROR_STRATEGIES.md) | Error handling patterns |
| [PERFORMANCE.md](docs/PERFORMANCE.md) | Benchmarks and optimization |
| [THREAD_SAFETY.md](docs/THREAD_SAFETY.md) | Concurrency patterns |
| [TESTING_PATTERNS.md](docs/TESTING_PATTERNS.md) | Testing best practices |
| [COMMON_PITFALLS.md](docs/COMMON_PITFALLS.md) | Troubleshooting guide |

---

## Contributing

Contributions welcome! Please read our contributing guidelines and submit PRs.

## License

MIT License - see [LICENSE](LICENSE) for details.

---

<div align="center">

**[Documentation](https://pkg.go.dev/github.com/itsatony/go-prompty/v2)** |
**[Examples](examples/)** |
**[Issues](https://github.com/itsatony/go-prompty/issues)**

</div>
