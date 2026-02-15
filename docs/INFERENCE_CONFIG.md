# Inference Configuration (Legacy v1)

> **DEPRECATED in v2.1**: The `InferenceConfig` type has been **removed** in go-prompty v2.1. All templates now use the `Prompt` type with `ExecutionConfig` for LLM parameters. The v1 `model:` YAML format is no longer supported. See [v2.1 Prompt Configuration](#v21-prompt-configuration) below for the current format.
>
> **Migration**: Replace `model:` blocks with `execution:` blocks. Replace `tmpl.InferenceConfig()` calls with `tmpl.Prompt()`. See the [Migration Guide](#migration-from-v1-to-v21) at the bottom of this document.

This document is preserved as a reference for the legacy v1 configuration format. The v1 API methods (`HasInferenceConfig()`, `InferenceConfig()`, `ModelConfig`, `ModelParameters`) no longer exist in v2.1.

## YAML Frontmatter Format

Configuration uses standard YAML frontmatter with `---` delimiters, similar to Jekyll, Hugo, and Microsoft Prompty. The frontmatter must appear at the start of the template (after optional whitespace):

```yaml
---
name: customer-support-agent
description: Handles customer inquiries
version: 1.0.0
model:
  api: chat
  provider: openai
  name: gpt-4
  parameters:
    temperature: 0.7
    max_tokens: 2048
---
{~prompty.message role="system"~}
You are a helpful customer support agent.
{~/prompty.message~}

{~prompty.message role="user"~}
{~prompty.var name="query" /~}
{~/prompty.message~}
```

## Configuration Fields

### Metadata Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Template identifier |
| `description` | string | Human-readable description |
| `version` | string | Semantic version (e.g., "1.0.0") |
| `authors` | []string | Author emails or names |
| `tags` | []string | Categorization tags |

### Model Configuration

```yaml
model:
  api: chat
  provider: openai
  name: gpt-4
  parameters:
    temperature: 0.7
    max_tokens: 2048
    top_p: 0.9
    frequency_penalty: 0.0
    presence_penalty: 0.0
    stop:
      - "\n\n"
    seed: 42
  response_format:
    type: json_schema
    json_schema:
      name: response
      strict: true
      schema:
        type: object
        properties:
          answer:
            type: string
        required:
          - answer
  tools:
    - type: function
      function:
        name: get_weather
        description: Get current weather
        parameters:
          type: object
          properties:
            location:
              type: string
          required:
            - location
  tool_choice: auto
  streaming:
    enabled: true
  context_window: 8192
```

| Field | Type | Description |
|-------|------|-------------|
| `api` | string | API type: "chat" or "completion" |
| `provider` | string | Provider hint (e.g., "openai", "anthropic") |
| `name` | string | Model identifier |
| `parameters` | object | Model-specific parameters |
| `response_format` | object | Structured output format (v1.4.0+) |
| `tools` | array | Function/tool calling definitions (v1.4.0+) |
| `tool_choice` | string/object | Tool selection strategy (v1.4.0+) |
| `streaming` | object | Streaming configuration (v1.4.0+) |
| `context_window` | int | Token budget hint (v1.4.0+) |

### Response Format (Structured Outputs)

For structured JSON output enforcement:

```yaml
model:
  response_format:
    type: json_schema  # or "text", "json_object"
    json_schema:
      name: entities
      description: Extracted entities from text
      strict: true
      schema:
        type: object
        properties:
          people:
            type: array
            items:
              type: string
          places:
            type: array
            items:
              type: string
        required:
          - people
          - places
```

### Tool/Function Calling

Define callable functions for the model:

```yaml
model:
  tools:
    - type: function
      function:
        name: search_products
        description: Search product catalog
        parameters:
          type: object
          properties:
            query:
              type: string
              description: Search query
            category:
              type: string
              enum:
                - electronics
                - clothing
                - home
          required:
            - query
        strict: true
  tool_choice: auto  # or "none", "required", or specific tool
```

### Retry Configuration

```yaml
retry:
  max_attempts: 3
  backoff: exponential  # or "linear"
```

### Cache Configuration

```yaml
cache:
  system_prompt: true
  ttl: 3600  # seconds
```

### Input/Output Schemas

Define expected inputs and outputs for validation:

```yaml
inputs:
  name:
    type: string
    required: true
    description: User name
  count:
    type: number
    required: false
    default: 10
outputs:
  response:
    type: string
    description: Generated response
```

Supported types: `string`, `number`, `boolean`, `array`, `object`

### Sample Data

Provide sample data for testing and documentation:

```yaml
sample:
  name: Alice
  query: How do I reset my password?
```

## Message Tags for Conversations

Use `{~prompty.message~}` tags in the template body to define conversation messages:

```yaml
---
name: chat-assistant
model:
  api: chat
  name: gpt-4
---
{~prompty.message role="system"~}
You are a helpful assistant.
{~/prompty.message~}

{~prompty.message role="user"~}
{~prompty.var name="query" /~}
{~/prompty.message~}
```

### Message Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `role` | string | Yes | Message role: "system", "user", "assistant", or "tool" |
| `cache` | string | No | Cache hint: "true" or "false" |

### Dynamic Conversation History

Use loops to inject conversation history:

```yaml
---
name: chat-with-history
---
{~prompty.message role="system"~}
You are a helpful assistant.
{~/prompty.message~}

{~prompty.for item="msg" in="history"~}
{~prompty.message role="{~prompty.var name='msg.role' /~}"~}
{~prompty.var name="msg.content" /~}
{~/prompty.message~}
{~/prompty.for~}

{~prompty.message role="user"~}
{~prompty.var name="query" /~}
{~/prompty.message~}
```

### Extracting Messages

After execution, extract structured messages for LLM API calls:

```go
// Execute and extract messages in one call
messages, err := tmpl.ExecuteAndExtractMessages(ctx, data)
// messages is []prompty.Message{{Role: "system", Content: "..."}, ...}

// Or extract from raw output
output, _ := tmpl.Execute(ctx, data)
messages := prompty.ExtractMessagesFromOutput(output)
```

## Using Variables in Config

go-prompty template tags can be used within config values for dynamic configuration.

### Environment Variables

Use the `prompty.env` tag to reference environment variables:

```yaml
---
name: env-config
description: 'API Key: {~prompty.env name="API_KEY" /~}'
model:
  name: '{~prompty.env name="MODEL_NAME" default="gpt-4" /~}'
---
```

**IMPORTANT: YAML Quoting Rules**

When using prompty tags in YAML values, use **single quotes**:

```yaml
# CORRECT - single quotes preserve literal content
name: '{~prompty.env name="MODEL_NAME" /~}'

# WRONG - double quotes require backslash escaping which breaks prompty parsing
name: "{~prompty.env name=\"MODEL_NAME\" /~}"
```

YAML single-quoted strings preserve all characters literally, while double-quoted strings interpret escape sequences like `\"`.

Environment variable options:
- `name` (required): Environment variable name
- `default`: Default value if not set
- `required="true"`: Error if not set (and no default)

### Variable Substitution

```yaml
model:
  name: '{~prompty.var name="model_name" default="gpt-4" /~}'
```

## v2.1 API Usage

> The examples below show the current v2.1 API. For v1 API patterns, see the git history prior to v2.1.

### Parsing Templates with Config

```go
engine, _ := prompty.New()

source := `---
name: my-template
description: A greeting template
execution:
  model: gpt-4
---
{~prompty.message role="user"~}
Hello {~prompty.var name="user" /~}!
{~/prompty.message~}`

tmpl, err := engine.Parse(source)
if err != nil {
    log.Fatal(err)
}

// Access prompt config
if tmpl.HasPrompt() {
    prompt := tmpl.Prompt()
    fmt.Println("Name:", prompt.Name)
    if prompt.Execution != nil {
        fmt.Println("Model:", prompt.Execution.Model)
    }
}
```

### Accessing Execution Parameters

```go
prompt := tmpl.Prompt()

if prompt.Execution != nil {
    exec := prompt.Execution
    fmt.Println("Provider:", exec.GetEffectiveProvider())
    fmt.Println("Model:", exec.Model)

    // Get parameters (pointer types distinguish unset from zero)
    if temp, ok := exec.GetTemperature(); ok {
        fmt.Println("Temperature:", temp)
    }

    if maxTokens, ok := exec.GetMaxTokens(); ok {
        fmt.Println("Max Tokens:", maxTokens)
    }

    // Provider-specific serialization
    openAIParams := exec.ToOpenAI()
    anthropicParams := exec.ToAnthropic()
    _ = openAIParams
    _ = anthropicParams
}
```

### Structured Outputs

```go
prompt := tmpl.Prompt()

if prompt.Execution != nil && prompt.Execution.ResponseFormat != nil {
    rf := prompt.Execution.ResponseFormat
    fmt.Println("Format type:", rf.Type)
    if rf.JSONSchema != nil {
        fmt.Println("Schema name:", rf.JSONSchema.Name)
    }
}
```

### Input Validation

```go
prompt := tmpl.Prompt()

data := map[string]any{
    "name": "Alice",
    "count": 42,
}

if err := prompt.ValidateInputs(data); err != nil {
    log.Fatal("Invalid inputs:", err)
}

result, _ := tmpl.Execute(ctx, data)
```

### Storage Integration

When using StorageEngine, PromptConfig is automatically extracted and persisted:

```go
se, _ := prompty.NewStorageEngine(prompty.StorageEngineConfig{
    Storage: prompty.NewMemoryStorage(),
})

se.Save(ctx, &prompty.StoredTemplate{
    Name:   "my-template",
    Source: source,
})

// Retrieved templates include PromptConfig
retrieved, _ := se.Get(ctx, "my-template")
if retrieved.PromptConfig != nil && retrieved.PromptConfig.Execution != nil {
    fmt.Println("Model:", retrieved.PromptConfig.Execution.Model)
}
```

## Serialization

Prompt config can be serialized for storage or API responses:

```go
prompt := tmpl.Prompt()

// YAML output
yamlStr, _ := prompt.YAML()

// JSON output (compact)
jsonStr, _ := prompt.JSON()

// JSON output (pretty-printed)
prettyJSON, _ := prompt.JSONPretty()
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/itsatony/go-prompty/v2"
)

func main() {
    os.Setenv("MODEL_NAME", "gpt-4-turbo")

    engine, _ := prompty.New()

    source := `---
name: customer-support
description: Handles customer inquiries
type: skill
execution:
  provider: openai
  model: '{~prompty.env name="MODEL_NAME" default="gpt-4" /~}'
  temperature: 0.7
  max_tokens: 2048
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
{~/prompty.message~}`

    tmpl, _ := engine.Parse(source)
    prompt := tmpl.Prompt()

    fmt.Println("Name:", prompt.Name)
    if prompt.Execution != nil {
        fmt.Println("Model:", prompt.Execution.Model)
        if temp, ok := prompt.Execution.GetTemperature(); ok {
            fmt.Println("Temperature:", temp)
        }
    }

    // Validate and execute with sample data
    if err := prompt.ValidateInputs(prompt.Sample); err != nil {
        fmt.Println("Validation error:", err)
        return
    }

    messages, _ := tmpl.ExecuteAndExtractMessages(context.Background(), prompt.Sample)
    for _, msg := range messages {
        fmt.Printf("[%s]: %s\n", msg.Role, msg.Content)
    }
}
```

## Error Handling

Frontmatter errors include position information:

```go
tmpl, err := engine.Parse(source)
if err != nil {
    // Error messages indicate the issue and location
    // "failed to extract YAML frontmatter at line 1, column 1"
    // "failed to parse YAML frontmatter"
    fmt.Println(err)
}
```

## Design Notes

- **Store & Expose**: Prompt configuration is parsed and stored but does not make LLM API calls. Use the configuration with your own LLM client.
- **Immutable After Parsing**: Prompt configuration is immutable after template parsing.
- **Coexists with Metadata**: PromptConfig and StoredTemplate.Metadata are independent and both available.
- **No Frontmatter**: Templates without frontmatter work normally; `HasPrompt()` returns false.
- **Position Requirement**: Frontmatter must be at the start of the template (after optional whitespace/BOM).
- **YAML Single Quotes**: Use single quotes for YAML values containing prompty tags to avoid escaping issues.

---

## v2.1 Prompt Configuration

go-prompty v2.1 uses the `Prompt` type compatible with the [Agent Skills](https://agentskills.io) specification. This provides better separation of concerns and interoperability with other Agent Skills tools. v2.1 adds document types (prompt, skill, agent), skills, tools, constraints, and agent compilation.

### v2.1 Format

The key difference from v1 is namespaced configuration with `execution` and `skope` sections:

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
  response_format:
    type: json_schema
    json_schema:
      name: response
      schema:
        type: object
        properties:
          answer: {type: string}
        required: [answer]

skope:
  visibility: public
  project_id: proj_helpdesk
  regions: [us-east-1, eu-west-1]
  projects: [support, helpdesk]

inputs:
  query:
    type: string
    required: true
    description: User's support query
---
{~prompty.message role="system"~}
You are a helpful support agent.
{~/prompty.message~}
```

### v2.1 Detection

In v2.1, ALL frontmatter is parsed as `Prompt`. Templates are detected as v2.1 if they have:
- A `type` field (prompt, skill, agent), OR
- An `execution` config block, OR
- A `skope` config block, OR
- A `name` field

The v1 `InferenceConfig` fallback has been completely removed.

### Using v2.1 Prompts

```go
tmpl, _ := engine.Parse(source)

if tmpl.HasPrompt() {
    prompt := tmpl.Prompt()

    // Agent Skills standard fields
    fmt.Println("Name:", prompt.Name)
    fmt.Println("Description:", prompt.Description)
    fmt.Println("License:", prompt.License)

    // Execution config
    if prompt.Execution != nil {
        fmt.Println("Provider:", prompt.Execution.Provider)
        fmt.Println("Model:", prompt.Execution.Model)

        // Provider-specific conversion
        openAIParams := prompt.Execution.ToOpenAI()
        anthropicParams := prompt.Execution.ToAnthropic()
        geminiParams := prompt.Execution.ToGemini()
        vllmParams := prompt.Execution.ToVLLM()

        // Extended thinking (Claude)
        if prompt.Execution.HasThinking() {
            fmt.Println("Thinking enabled:", prompt.Execution.Thinking.Enabled)
        }
    }

    // Skope platform config
    if prompt.Skope != nil {
        fmt.Println("Visibility:", prompt.Skope.Visibility)
        fmt.Println("Projects:", prompt.Skope.Projects)
    }

    // Validate inputs
    if err := prompt.ValidateInputs(data); err != nil {
        log.Fatal("Invalid inputs:", err)
    }
}
```

### Prompt References

The `{~prompty.ref~}` tag enables prompt composition:

```yaml
{~prompty.ref slug="common-instructions" /~}
{~prompty.ref slug="customer-context" version="v2" /~}
{~prompty.ref slug="safety-guidelines@latest" /~}
```

To use references, implement the `PromptResolver` interface and set it on the context:

```go
// Implement PromptResolver
type MyPromptStore struct {
    prompts map[string]PromptWithBody
}

func (s *MyPromptStore) ResolvePrompt(ctx context.Context, slug, version string) (*prompty.Prompt, string, error) {
    p, ok := s.prompts[slug]
    if !ok {
        return nil, "", prompty.NewRefNotFoundError(slug, version)
    }
    return p.Prompt, p.Body, nil
}

// Create adapter and set on context
adapter := prompty.NewPromptResolverAdapter(myStore)
execCtx := prompty.NewContext(data).WithPromptResolver(adapter)

result, err := tmpl.ExecuteWithContext(ctx, execCtx)
```

### SKILL.md Import/Export

Export prompts in Agent Skills SKILL.md format:

```go
prompt := tmpl.Prompt()

// Export to SKILL.md (strips execution/skope for portability)
skillMD, _ := prompt.ExportToSkillMD(tmpl.TemplateBody())

// Import from SKILL.md
parsed, _ := prompty.ImportFromSkillMD(skillMDContent)
prompt := parsed.Prompt
body := parsed.Body

// Check Agent Skills compatibility
if prompt.IsAgentSkillsCompatible() {
    // No go-prompty specific extensions
}

// Strip extensions for export
stripped := prompt.StripExtensions()
```

### Migration from v1 to v2.1

**v1 format (removed in v2.1):**
```yaml
---
name: my-template
model:
  provider: openai
  name: gpt-4
  parameters:
    temperature: 0.7
---
```

**v2.1 format (current):**
```yaml
---
name: my-template
description: A helpful template
type: skill
execution:
  provider: openai
  model: gpt-4
  temperature: 0.7
---
```

**API changes:**
- `tmpl.HasInferenceConfig()` / `tmpl.InferenceConfig()` - **removed**, use `tmpl.HasPrompt()` / `tmpl.Prompt()`
- `InferenceConfig` type - **removed**, use `Prompt` + `ExecutionConfig`
- `ModelConfig` / `ModelParameters` - **removed**, parameters are flat in `ExecutionConfig`
- `StoredTemplate.InferenceConfig` - **removed**, use `StoredTemplate.PromptConfig`

Key differences:
1. `description` is recommended for v2.1
2. `type` field specifies document type: `prompt`, `skill` (default), or `agent`
3. Model config moved to `execution` namespace with flat parameters
4. Optional `skope` config for platform integration
5. Agent type supports `skills`, `tools`, `constraints`, and `messages`

## v2.5 Media Generation Parameters

v2.5 extends `ExecutionConfig` with nested structs for multimodal AI generation:

### Media Config Reference

| Config | Field | Type | Range | Description |
|--------|-------|------|-------|-------------|
| `image` | `width` | `*int` | 1-8192 | Image width in pixels |
| `image` | `height` | `*int` | 1-8192 | Image height in pixels |
| `image` | `size` | `string` | — | Provider-specific size (e.g., "1024x1024") |
| `image` | `quality` | `string` | standard/hd/low/medium/high | Image quality |
| `image` | `style` | `string` | natural/vivid | Image style |
| `image` | `aspect_ratio` | `string` | — | Aspect ratio (e.g., "16:9") |
| `image` | `negative_prompt` | `string` | — | Content to avoid |
| `image` | `num_images` | `*int` | 1-10 | Number of images |
| `image` | `guidance_scale` | `*float64` | 0.0-30.0 | Prompt adherence |
| `image` | `steps` | `*int` | 1-200 | Diffusion steps |
| `image` | `strength` | `*float64` | 0.0-1.0 | Transformation strength |
| `audio` | `voice` | `string` | — | Voice name |
| `audio` | `voice_id` | `string` | — | Provider-specific voice ID |
| `audio` | `speed` | `*float64` | 0.25-4.0 | Playback speed |
| `audio` | `output_format` | `string` | mp3/opus/aac/flac/wav/pcm | Output format |
| `audio` | `duration` | `*float64` | 0-600 | Max duration in seconds |
| `audio` | `language` | `string` | — | Language code |
| `embedding` | `dimensions` | `*int` | 1-65536 | Embedding dimensions |
| `embedding` | `format` | `string` | float/base64 | Wire encoding format (OpenAI) |
| `embedding` | `input_type` | `string` | search_query/search_document/classification/clustering/semantic_similarity | Input classification (Gemini, Cohere) |
| `embedding` | `output_dtype` | `string` | float32/int8/uint8/binary/ubinary | Quantization data type (Mistral, Cohere) |
| `embedding` | `truncation` | `string` | none/start/end | Truncation strategy (Cohere) |
| `embedding` | `normalize` | `*bool` | true/false | L2-normalize embeddings (vLLM) |
| `embedding` | `pooling_type` | `string` | mean/cls/last | Pooling strategy (vLLM) |
| `streaming` | `enabled` | `bool` | — | Enable streaming |
| `streaming` | `method` | `string` | sse/websocket | Transport method |
| `async` | `enabled` | `bool` | — | Enable async execution |
| `async` | `poll_interval_seconds` | `*float64` | >0 | Polling interval |
| `async` | `poll_timeout_seconds` | `*float64` | >0, >=interval | Polling timeout |
| (root) | `modality` | `string` | text/image/audio_speech/audio_transcription/music/sound_effects/embedding | Execution intent |
