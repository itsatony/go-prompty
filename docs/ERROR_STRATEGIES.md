# Error Strategy Decision Guide

go-prompty provides five error handling strategies for template execution. This guide helps you choose the right strategy for your use case.

## Quick Decision Flowchart

```
Is this development/testing?
├── Yes → Use `throw` (fail fast, see errors immediately)
└── No → Is the field optional?
    ├── Yes → Does it need a fallback value?
    │   ├── Yes → Use `default`
    │   └── No → Use `remove`
    └── No → Is this a preview/debug context?
        ├── Yes → Use `keepraw`
        └── No → Need to track errors for analytics?
            ├── Yes → Use `log`
            └── No → Use `throw`
```

## Strategy Overview

| Strategy | Output on Error | Best For |
|----------|----------------|----------|
| `throw` | Execution stops | Development, CI/CD, required fields |
| `default` | Uses `default` attribute | Optional fields with sensible fallbacks |
| `remove` | Empty string | Conditionally shown content |
| `keepraw` | Original tag text | Template previews, debugging |
| `log` | Empty string + log | Production analytics, monitoring |

## Detailed Strategy Guide

### `throw` (Default)

**Behavior**: Stops execution immediately and returns the error.

**Use when**:
- During development to catch errors early
- In CI/CD pipelines where failures should break the build
- For required fields that must have values
- When data integrity is critical

**Example**:
```go
engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyThrow))

// This will return an error because 'name' is not provided
result, err := engine.Execute(ctx,
    `Hello {~prompty.var name="name" /~}!`,
    map[string]any{})

if err != nil {
    // Handle error: "variable not found: name"
}
```

**Advantages**:
- Immediate feedback on issues
- Prevents silent failures
- Ensures data completeness

**Disadvantages**:
- Can be disruptive in production
- May require more defensive coding

---

### `default`

**Behavior**: Uses the `default` attribute value when the variable is not found.

**Use when**:
- Fields are optional but should show something
- You have sensible fallback values
- User experience matters more than strict data requirements

**Example**:
```go
engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyDefault))

// Uses "Guest" when user.name is not found
result, _ := engine.Execute(ctx,
    `Welcome, {~prompty.var name="user.name" default="Guest" /~}!`,
    map[string]any{})
// Result: "Welcome, Guest!"
```

**Per-tag override**:
```
{~prompty.var name="optional.field" default="N/A" onerror="default" /~}
```

**Advantages**:
- Graceful degradation
- User-friendly output
- Clear fallback semantics

**Disadvantages**:
- Silent substitution may hide data issues
- Requires `default` attribute on each tag

---

### `remove`

**Behavior**: Removes the entire tag from output, leaving nothing.

**Use when**:
- Content should only appear when data exists
- Building conditional sections without explicit conditionals
- Cleaning up optional decorative elements

**Example**:
```go
engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyRemove))

// Missing subtitle is simply removed
result, _ := engine.Execute(ctx,
    `Title: {~prompty.var name="title" /~}
{~prompty.var name="subtitle" /~}
Content here...`,
    map[string]any{"title": "Welcome"})
// Result: "Title: Welcome\n\nContent here..."
```

**Per-tag override**:
```
{~prompty.var name="optional.badge" onerror="remove" /~}
```

**Advantages**:
- Clean output without placeholders
- Simple conditional visibility
- No residual markers

**Disadvantages**:
- May produce unexpected whitespace
- Hard to debug missing content

---

### `keepraw`

**Behavior**: Keeps the original tag text in the output.

**Use when**:
- Previewing templates with placeholder data
- Debugging template structure
- Showing users what variables are expected
- Creating template documentation

**Example**:
```go
engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyKeepRaw))

// Shows the original tag when data is missing
result, _ := engine.Execute(ctx,
    `Hello {~prompty.var name="user" /~}!`,
    map[string]any{})
// Result: `Hello {~prompty.var name="user" /~}!`
```

**Per-tag override**:
```
{~prompty.var name="debug.value" onerror="keepraw" /~}
```

**Advantages**:
- Visible placeholder locations
- Easy template debugging
- Self-documenting output

**Disadvantages**:
- Raw tags may confuse end users
- Not suitable for production output

---

### `log`

**Behavior**: Logs the error, then continues with an empty string.

**Use when**:
- Monitoring production template execution
- Tracking error rates and patterns
- Building analytics on template usage
- Graceful degradation with observability

**Example**:
```go
engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyLog))

// Logs the error but produces output
result, _ := engine.Execute(ctx,
    `User: {~prompty.var name="user" /~}`,
    map[string]any{})
// Result: "User: "
// Log output: WARN: variable not found: user
```

**Per-tag override**:
```
{~prompty.var name="analytics.field" onerror="log" /~}
```

**Advantages**:
- Non-blocking execution
- Error tracking/metrics
- Production-safe

**Disadvantages**:
- Requires log monitoring
- Silent failures in output

---

## Per-Tag Override

Any tag can override the global strategy using the `onerror` attribute:

```
{~prompty.var name="required.field" /~}                          {-- Uses global strategy --}
{~prompty.var name="optional.field" onerror="remove" /~}         {-- Always removes on error --}
{~prompty.var name="debug.field" onerror="keepraw" /~}           {-- Always keeps raw --}
{~prompty.var name="fallback.field" default="N/A" onerror="default" /~}  {-- Uses default --}
```

This allows mixing strategies within a single template:

```go
engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyThrow))

result, err := engine.Execute(ctx, `
User: {~prompty.var name="user.name" /~}
Bio: {~prompty.var name="user.bio" onerror="remove" /~}
Avatar: {~prompty.var name="user.avatar" default="/default.png" onerror="default" /~}
`, data)
```

In this example:
- `user.name` - Required, throws on error (global strategy)
- `user.bio` - Optional, silently removed if missing
- `user.avatar` - Optional, uses default image path if missing

---

## Environment-Based Configuration

A common pattern is using different strategies per environment:

```go
func getEngine(env string) *prompty.Engine {
    var strategy prompty.ErrorStrategy

    switch env {
    case "development", "test":
        strategy = prompty.ErrorStrategyThrow
    case "staging":
        strategy = prompty.ErrorStrategyLog
    case "production":
        strategy = prompty.ErrorStrategyDefault
    default:
        strategy = prompty.ErrorStrategyThrow
    }

    return prompty.MustNew(prompty.WithErrorStrategy(strategy))
}
```

---

## Strategy Comparison Table

| Scenario | throw | default | remove | keepraw | log |
|----------|-------|---------|--------|---------|-----|
| Missing required field | Error returned | Uses default | Empty output | Shows tag | Empty + log |
| Development feedback | Immediate | Delayed/hidden | Hidden | Visible | Requires monitoring |
| Production safety | Risky | Safe | Safe | Unsafe | Safe |
| Data integrity | Strict | Relaxed | Relaxed | N/A | Relaxed + tracked |
| User experience | Poor on error | Good | Good | Confusing | Good |
| Debugging ease | High | Medium | Low | High | Medium |

---

## Recommendations by Use Case

### API Response Generation
```go
// Use throw - API responses should fail clearly
engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyThrow))
```

### Email Templates
```go
// Use default - emails should render with fallbacks
engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyDefault))
```

### LLM Prompt Generation
```go
// Use throw or log - prompts need reliable data
engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyThrow))
// Or for production:
engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyLog))
```

### Template Preview/Editor
```go
// Use keepraw - show users what variables exist
engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyKeepRaw))
```

### Notification Messages
```go
// Use remove - missing optional parts should be clean
engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyRemove))
```

### CI/CD Validation
```go
// Use throw - validation should fail fast
engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyThrow))
```

---

## Dry-Run for Safe Validation

Before executing templates with real data, use dry-run to validate without side effects:

```go
tmpl, _ := engine.Parse(templateSource)

// Dry-run reports all issues without executing
result := tmpl.DryRun(ctx, sampleData)

if !result.Valid {
    fmt.Println("Template issues:")
    for _, err := range result.Errors {
        fmt.Println("  -", err)
    }
}

if len(result.MissingVariables) > 0 {
    fmt.Println("Missing variables:", result.MissingVariables)
}

// Only execute if validation passes
if result.Valid && len(result.MissingVariables) == 0 {
    output, _ := tmpl.Execute(ctx, realData)
}
```

This combines well with `throw` strategy - validate with dry-run first, then execute with confidence.

---

## Summary

1. **Development**: Use `throw` to catch issues early
2. **Production with fallbacks**: Use `default` for graceful degradation
3. **Clean conditional content**: Use `remove` for optional sections
4. **Template debugging**: Use `keepraw` to see what's missing
5. **Production monitoring**: Use `log` to track errors without disruption

Choose the strategy that best matches your error tolerance and observability needs. When in doubt, start with `throw` during development, then relax to `default` or `log` for production.
