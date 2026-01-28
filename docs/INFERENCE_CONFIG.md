# Inference Configuration

go-prompty supports embedded inference configuration in templates through YAML frontmatter. This allows templates to be self-describing with model configuration, parameters, input/output schemas, conversation messages, and sample data.

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

## API Usage

### Parsing Templates with Config

```go
engine, _ := prompty.New()

source := `---
name: my-template
model:
  name: gpt-4
---
{~prompty.message role="user"~}
Hello {~prompty.var name="user" /~}!
{~/prompty.message~}`

tmpl, err := engine.Parse(source)
if err != nil {
    log.Fatal(err)
}

// Access config
if tmpl.HasInferenceConfig() {
    config := tmpl.InferenceConfig()
    fmt.Println("Template:", config.Name)
    fmt.Println("Model:", config.GetModelName())
}
```

### Accessing Model Parameters

```go
config := tmpl.InferenceConfig()

// Check for model
if config.HasModel() {
    fmt.Println("API:", config.GetAPIType())
    fmt.Println("Provider:", config.GetProvider())
    fmt.Println("Model:", config.GetModelName())
}

// Get parameters with defaults
if temp, ok := config.GetTemperature(); ok {
    fmt.Println("Temperature:", temp)
}

if maxTokens, ok := config.GetMaxTokens(); ok {
    fmt.Println("Max Tokens:", maxTokens)
}

// Get all parameters as map
if config.Model != nil && config.Model.Parameters != nil {
    params := config.Model.Parameters.ToMap()
    // Use with your LLM client
}
```

### Accessing New v1.4.0 Fields

```go
config := tmpl.InferenceConfig()

// Response format for structured outputs
if config.HasResponseFormat() {
    rf := config.GetResponseFormat()
    fmt.Println("Format type:", rf.Type)
    if rf.JSONSchema != nil {
        fmt.Println("Schema name:", rf.JSONSchema.Name)
    }
}

// Tools for function calling
if config.HasTools() {
    tools := config.GetTools()
    for _, tool := range tools {
        fmt.Println("Tool:", tool.Function.Name)
    }
}

// Streaming config
if config.HasStreaming() {
    streaming := config.GetStreaming()
    fmt.Println("Streaming enabled:", streaming.Enabled)
}

// Retry config
if config.HasRetry() {
    retry := config.GetRetry()
    fmt.Println("Max attempts:", retry.MaxAttempts)
}

// Cache config
if config.HasCache() {
    cache := config.GetCache()
    fmt.Println("Cache system prompt:", cache.SystemPrompt)
}
```

### Input Validation

```go
config := tmpl.InferenceConfig()

// Validate inputs before execution
data := map[string]any{
    "name": "Alice",
    "count": 42,
}

if err := config.ValidateInputs(data); err != nil {
    log.Fatal("Invalid inputs:", err)
}

// Execute template
result, _ := tmpl.Execute(ctx, data)
```

### Using Sample Data

```go
config := tmpl.InferenceConfig()

if config.HasSample() {
    sample := config.GetSampleData()
    // Execute with sample data for testing
    result, _ := tmpl.Execute(ctx, sample)
}
```

### Storage Integration

When using StorageEngine, InferenceConfig is automatically extracted and persisted:

```go
storage, _ := prompty.OpenStorage("memory", "")
engine, _ := prompty.New()
se, _ := prompty.NewStorageEngine(prompty.StorageEngineConfig{
    Storage: storage,
    Engine:  engine,
})

// Save template - InferenceConfig is automatically extracted
tmpl := &prompty.StoredTemplate{
    Name:   "my-template",
    Source: source,
}
se.Save(ctx, tmpl)

// Retrieved templates include InferenceConfig
retrieved, _ := se.Get(ctx, "my-template")
if retrieved.InferenceConfig != nil {
    fmt.Println("Model:", retrieved.InferenceConfig.GetModelName())
}
```

## YAML Serialization

InferenceConfig can be serialized for storage or API responses:

```go
config := tmpl.InferenceConfig()

// YAML output
yamlStr, _ := config.YAML()

// JSON output (compact)
jsonStr, _ := config.JSON()

// JSON output (pretty-printed)
prettyJSON, _ := config.JSONPretty()
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/itsatony/go-prompty"
)

func main() {
    // Set environment variable for model
    os.Setenv("MODEL_NAME", "gpt-4-turbo")

    engine, _ := prompty.New()

    source := `---
name: customer-support
version: 1.0.0
model:
  api: chat
  provider: openai
  name: '{~prompty.env name="MODEL_NAME" default="gpt-4" /~}'
  parameters:
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
    config := tmpl.InferenceConfig()

    // Print configuration
    fmt.Println("Template:", config.Name)
    fmt.Println("Version:", config.Version)
    fmt.Println("Model:", config.GetModelName()) // "gpt-4-turbo" from env

    if temp, ok := config.GetTemperature(); ok {
        fmt.Println("Temperature:", temp)
    }

    // Validate and execute with sample data
    sample := config.GetSampleData()
    if err := config.ValidateInputs(sample); err != nil {
        fmt.Println("Validation error:", err)
        return
    }

    // Execute and extract messages
    messages, _ := tmpl.ExecuteAndExtractMessages(context.Background(), sample)
    for _, msg := range messages {
        fmt.Printf("[%s]: %s\n", msg.Role, msg.Content)
    }
}
```

## Migration from JSON Config Blocks

If you have templates using the legacy JSON `{~prompty.config~}` format, migrate to YAML frontmatter:

**Before (JSON - deprecated):**
```
{~prompty.config~}
{"name": "my-template", "model": {"name": "gpt-4"}}
{~/prompty.config~}
Hello {~prompty.var name="user" /~}
```

**After (YAML frontmatter):**
```yaml
---
name: my-template
model:
  name: gpt-4
---
{~prompty.message role="user"~}
Hello {~prompty.var name="user" /~}
{~/prompty.message~}
```

## Error Handling

Frontmatter errors include position information:

```go
tmpl, err := engine.Parse(source)
if err != nil {
    // Error messages indicate the issue and location
    // "failed to extract YAML frontmatter at line 1, column 1"
    // "failed to parse YAML frontmatter"
    // "legacy JSON config block detected - please migrate to YAML frontmatter"
    fmt.Println(err)
}
```

## Design Notes

- **Store & Expose**: InferenceConfig is parsed and stored but does not make LLM API calls. Use the configuration with your own LLM client.
- **Immutable After Parsing**: InferenceConfig is immutable after template parsing.
- **Coexists with Metadata**: InferenceConfig and StoredTemplate.Metadata are independent and both available.
- **No Frontmatter**: Templates without frontmatter work normally; `HasInferenceConfig()` returns false.
- **Position Requirement**: Frontmatter must be at the start of the template (after optional whitespace/BOM).
- **YAML Single Quotes**: Use single quotes for YAML values containing prompty tags to avoid escaping issues.
