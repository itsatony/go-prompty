# Inference Configuration

go-prompty supports embedded inference configuration in templates through JSON-based config blocks. This allows templates to be self-describing with model configuration, parameters, input/output schemas, and sample data.

## Config Block Format

Config blocks use go-prompty delimiters and must appear at the start of the template (after optional whitespace):

```
{~prompty.config~}
{
  "name": "customer-support-agent",
  "description": "Handles customer inquiries",
  "version": "1.0.0",
  "model": {
    "api": "chat",
    "provider": "openai",
    "name": "gpt-4",
    "parameters": {
      "temperature": 0.7,
      "max_tokens": 2048
    }
  }
}
{~/prompty.config~}

Hello {~prompty.var name="user" /~}!
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

```json
{
  "model": {
    "api": "chat",
    "provider": "openai",
    "name": "gpt-4",
    "parameters": {
      "temperature": 0.7,
      "max_tokens": 2048,
      "top_p": 0.9,
      "frequency_penalty": 0.0,
      "presence_penalty": 0.0,
      "stop": ["\n\n"],
      "seed": 42
    }
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `api` | string | API type: "chat" or "completion" |
| `provider` | string | Provider hint (e.g., "openai", "anthropic") |
| `name` | string | Model identifier |
| `parameters` | object | Model-specific parameters |

### Input/Output Schemas

Define expected inputs and outputs for validation:

```json
{
  "inputs": {
    "name": {"type": "string", "required": true, "description": "User name"},
    "count": {"type": "number", "required": false, "default": 10}
  },
  "outputs": {
    "response": {"type": "string", "description": "Generated response"}
  }
}
```

Supported types: `string`, `number`, `boolean`, `array`, `object`

### Sample Data

Provide sample data for testing and documentation:

```json
{
  "sample": {
    "name": "Alice",
    "query": "How do I reset my password?"
  }
}
```

## Using Variables in Config

go-prompty template tags can be used within config blocks for dynamic configuration:

### Variable Substitution

```json
{
  "model": {
    "name": "{~prompty.var name=\"model_name\" default=\"gpt-4\" /~}"
  }
}
```

### Environment Variables

Use the `prompty.env` tag to reference environment variables:

```json
{
  "description": "API Key: {~prompty.env name='API_KEY' /~}",
  "model": {
    "name": "{~prompty.env name='MODEL_NAME' default='gpt-4' /~}"
  }
}
```

**Note on Quoting**: When using prompty tags inside JSON strings, use single quotes for tag attributes to avoid JSON escaping issues:

```json
// Recommended: single quotes for tag attributes
"name": "{~prompty.env name='MODEL_NAME' /~}"

// Also works with proper JSON escaping
"name": "{~prompty.env name=\"MODEL_NAME\" /~}"
```

Environment variable options:
- `name` (required): Environment variable name
- `default`: Default value if not set
- `required="true"`: Error if not set (and no default)

## API Usage

### Parsing Templates with Config

```go
engine, _ := prompty.New()

source := `{~prompty.config~}
{"name": "my-template", "model": {"name": "gpt-4"}}
{~/prompty.config~}
Hello {~prompty.var name="user" /~}!`

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

## JSON Serialization

InferenceConfig can be serialized for storage or API responses:

```go
config := tmpl.InferenceConfig()

// Compact JSON
jsonStr, _ := config.JSON()

// Pretty-printed JSON
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

    source := `{~prompty.config~}
{
  "name": "customer-support",
  "version": "1.0.0",
  "model": {
    "api": "chat",
    "provider": "openai",
    "name": "{~prompty.env name=\"MODEL_NAME\" default=\"gpt-4\" /~}",
    "parameters": {
      "temperature": 0.7,
      "max_tokens": 2048
    }
  },
  "inputs": {
    "customer_name": {"type": "string", "required": true},
    "query": {"type": "string", "required": true}
  },
  "sample": {
    "customer_name": "Alice",
    "query": "How do I reset my password?"
  }
}
{~/prompty.config~}
Hello {~prompty.var name="customer_name" /~},

Thank you for reaching out. I understand you need help with:
{~prompty.var name="query" /~}

Best regards,
Support Team`

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

    result, _ := tmpl.Execute(context.Background(), sample)
    fmt.Println("\nRendered template:")
    fmt.Println(result)
}
```

## Error Handling

Config block errors include position information:

```go
tmpl, err := engine.Parse(source)
if err != nil {
    // Error messages indicate the issue and location
    // "failed to extract config block at line 1, column 1"
    // "failed to parse config block JSON"
    fmt.Println(err)
}
```

## Design Notes

- **Store & Expose**: InferenceConfig is parsed and stored but does not make LLM API calls. Use the configuration with your own LLM client.
- **Immutable After Parsing**: InferenceConfig is immutable after template parsing.
- **Coexists with Metadata**: InferenceConfig and StoredTemplate.Metadata are independent and both available.
- **No Config Block**: Templates without config blocks work normally; `HasInferenceConfig()` returns false.
- **Position Requirement**: Config block must be at the start of the template (after optional whitespace).
