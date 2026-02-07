# Migration Guide: v1.x / v2.0 to v2.1

This guide covers migrating from go-prompty v1.x or v2.0 to v2.1.

## Overview of Changes

v2.1 introduces agent definitions, document resolvers, catalog generation, and a compilation pipeline while maintaining full backwards compatibility with existing template functionality.

| Area | v1.x / v2.0 | v2.1 |
|------|-------------|------|
| LLM config | `InferenceConfig` | `ExecutionConfig` (on `Prompt.Execution`) |
| Config access | `tmpl.InferenceConfig()` | `tmpl.Prompt().Execution` |
| Document types | N/A | `prompt`, `skill`, `agent` |
| Skills/tools | N/A | `SkillRef`, `ToolsConfig`, `ConstraintsConfig` |
| Agent compilation | N/A | `CompileAgent()`, `ActivateSkill()`, `Compile()` |
| Document resolution | N/A | `DocumentResolver` interface |
| Catalog generation | N/A | `GenerateSkillsCatalog()`, `GenerateToolsCatalog()` |
| Import/export | N/A | `Import()`, `ExportFull()`, `ExportSkillDirectory()` |
| Storage config field | `InferenceConfig` | `PromptConfig` |

---

## Step 1: InferenceConfig to ExecutionConfig

### Before (v1.x / v2.0)

```go
tmpl, _ := engine.Parse(source)
if tmpl.HasInferenceConfig() {
    ic := tmpl.InferenceConfig()
    fmt.Println(ic.Model.Name)
    fmt.Println(ic.Model.Parameters.Temperature)
}
```

### After (v2.1)

```go
tmpl, _ := engine.Parse(source)
if tmpl.HasPrompt() {
    prompt := tmpl.Prompt()
    if prompt.Execution != nil {
        fmt.Println(prompt.Execution.Model)
        if temp, ok := prompt.Execution.GetTemperature(); ok {
            fmt.Println(temp)
        }
    }
}
```

### YAML Changes

**Before:**
```yaml
---
name: my-prompt
model:
  api: chat
  configuration:
    name: gpt-4
  parameters:
    temperature: 0.7
    max_tokens: 1000
---
```

**After:**
```yaml
---
name: my-prompt
description: My prompt description
execution:
  provider: openai
  model: gpt-4
  temperature: 0.7
  max_tokens: 1000
---
```

---

## Step 2: Storage PromptConfig

### Before (v1.x / v2.0)

```go
tmpl, _ := engine.Get(ctx, "greeting")
if tmpl.InferenceConfig != nil {
    fmt.Println(tmpl.InferenceConfig.Model.Name)
}
```

### After (v2.1)

```go
tmpl, _ := engine.Get(ctx, "greeting")
if tmpl.PromptConfig != nil {
    fmt.Println(tmpl.PromptConfig.Execution.Model)
}
```

**PostgreSQL migration**: If using PostgreSQL storage, migration 4 automatically renames the `inference_config` column to `prompt_config`.

---

## Step 3: Provider-Specific Serialization

### Before (v1.x / v2.0)

```go
ic := tmpl.InferenceConfig()
params := ic.Model.Parameters.ToMap()
```

### After (v2.1)

```go
prompt := tmpl.Prompt()
exec := prompt.Execution

// Auto-detect provider
provider := exec.GetEffectiveProvider()

// Get provider-specific format
openAIParams, _ := exec.ProviderFormat(prompty.ProviderOpenAI)
anthropicParams, _ := exec.ProviderFormat(prompty.ProviderAnthropic)

// Or use convenience methods
openAIMap := exec.ToOpenAI()
anthropicMap := exec.ToAnthropic()
```

---

## Step 4: Adopt Document Types (Optional)

If your templates don't use agent features, no changes are needed. The default document type is `skill`.

To adopt document types, add `type:` to your YAML frontmatter:

```yaml
---
name: my-prompt
description: A simple prompt
type: prompt    # or "skill" (default) or "agent"
---
```

| Type | When to use |
|------|-------------|
| `prompt` | Simple templates with no tools or skills |
| `skill` | Reusable capabilities (default, backwards compatible) |
| `agent` | Full agents with skills, tools, constraints, messages |

---

## Step 5: Agent Compilation (New in v2.1)

For new agent-based workflows:

```go
// Parse an agent definition
agent, _ := prompty.Parse([]byte(agentYAML))

// Set up skill resolution
resolver := prompty.NewMapDocumentResolver()
resolver.AddSkill("my-skill", skillPrompt)

// Compile the agent
compiled, _ := agent.CompileAgent(ctx, input, &prompty.CompileOptions{
    Resolver:            resolver,
    SkillsCatalogFormat: prompty.CatalogFormatDetailed,
})

// Use compiled output
for _, msg := range compiled.Messages {
    // Send to LLM API
}
```

### CompileOptions

```go
type CompileOptions struct {
    Resolver            DocumentResolver  // Resolves skill/prompt/agent references
    SkillsCatalogFormat CatalogFormat     // Format for {~prompty.skills_catalog~} tags
    ToolsCatalogFormat  CatalogFormat     // Format for {~prompty.tools_catalog~} tags
    Engine              *Engine           // Optional pre-configured engine
}
```

### Execution Config Merging

v2.1 supports 3-layer precedence for execution config:

1. **Agent definition** — base execution config from the agent's YAML
2. **Skill override** — execution config from the resolved skill
3. **Runtime input** — execution config from `SkillRef.Execution`

```go
// Merge: other's non-nil fields override receiver's fields
effective := agentExec.Merge(skillExec)
```

---

## Step 6: Document Resolver (New in v2.1)

If you want skills and agents to reference each other by slug:

```go
// In-memory (testing/simple)
resolver := prompty.NewMapDocumentResolver()
resolver.AddSkill("search", searchPrompt)
resolver.AddAgent("assistant", assistantPrompt)

// Storage-backed (production)
resolver := prompty.NewStorageDocumentResolver(storage)

// Use in compilation
compiled, _ := agent.CompileAgent(ctx, input, &prompty.CompileOptions{
    Resolver: resolver,
})
```

---

## Step 7: Import & Export (New in v2.1)

```go
// Export a prompt to markdown
bytes, _ := prompt.ExportFull()

// Export as Agent Skills compatible (no execution/skope)
bytes, _ := prompt.ExportAgentSkill()

// Export as zip with resources
zipBytes, _ := prompty.ExportSkillDirectory(prompt, resources)

// Import from file
result, _ := prompty.Import(data, "agent.md")
prompt := result.Prompt
```

---

## Backwards Compatibility

- All v1.x/v2.0 templates continue to work without changes
- YAML frontmatter without `execution:` is parsed as before
- `Parse()` defaults to `type: skill` when no type is specified
- The `InferenceConfig` type and its JSON tag references have been removed; use `Prompt.Execution` instead
- PostgreSQL storage migrations are automatic with `AutoMigrate: true`

## New Examples

| Example | Description |
|---------|-------------|
| [agent_compilation](../examples/agent_compilation/) | Full agent workflow |
| [document_resolver](../examples/document_resolver/) | Resolver implementations |
| [catalog_generation](../examples/catalog_generation/) | All catalog formats |
| [prompt_import_export](../examples/prompt_import_export/) | Import/export round-trips |
