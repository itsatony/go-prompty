# Common Pitfalls

This guide covers common mistakes developers make when using go-prompty and how to avoid them.

## Syntax Errors

### Pitfall: Missing closing tag marker

**Wrong**:
```
{~prompty.var name="user"}
```

**Correct**:
```
{~prompty.var name="user" /~}
```

The closing `/~}` is required for self-closing tags.

---

### Pitfall: Using wrong delimiters

**Wrong** (using standard template syntax):
```
{{prompty.var name="user"}}
{{ .user }}
${user}
```

**Correct**:
```
{~prompty.var name="user" /~}
```

go-prompty uses `{~...~}` delimiters specifically to avoid conflicts with other template systems and code content.

---

### Pitfall: Forgetting quotes around attribute values

**Wrong**:
```
{~prompty.var name=user /~}
{~prompty.if eval=user.isAdmin~}
```

**Correct**:
```
{~prompty.var name="user" /~}
{~prompty.if eval="user.isAdmin"~}
```

All attribute values must be quoted with double quotes.

---

### Pitfall: Using single quotes

**Wrong**:
```
{~prompty.var name='user' /~}
```

**Correct**:
```
{~prompty.var name="user" /~}
```

Only double quotes are supported for attribute values.

---

## Conditional Block Errors

### Pitfall: Incorrect closing tag for conditionals

**Wrong**:
```
{~prompty.if eval="condition"~}
  Content
{~/if~}
```

**Correct**:
```
{~prompty.if eval="condition"~}
  Content
{~/prompty.if~}
```

Closing tags must include the full tag name: `{~/prompty.if~}`.

---

### Pitfall: Missing else/elseif content

**Wrong** (empty branches may cause confusion):
```
{~prompty.if eval="isAdmin"~}
{~prompty.else~}
{~/prompty.if~}
```

**Better**:
```
{~prompty.if eval="isAdmin"~}
  Admin content
{~/prompty.if~}
```

If you don't need an else branch, simply omit it.

---

## Loop Errors

### Pitfall: Incorrect loop variable access

**Wrong**:
```
{~prompty.for item="user" in="users"~}
  {~prompty.var name="user.name" /~}
{~/prompty.for~}
```

When iterating over a slice of strings or simple values, the item itself is the value:

**Correct** (for slice of objects):
```
{~prompty.for item="user" in="users"~}
  {~prompty.var name="user.name" /~}
{~/prompty.for~}
```

**Correct** (for slice of strings):
```
{~prompty.for item="name" in="names"~}
  {~prompty.var name="name" /~}
{~/prompty.for~}
```

---

### Pitfall: Using wrong attribute for loop source

**Wrong**:
```
{~prompty.for item="x" from="items"~}
{~prompty.for item="x" list="items"~}
```

**Correct**:
```
{~prompty.for item="x" in="items"~}
```

The source collection attribute is `in`, not `from` or `list`.

---

## Switch Block Errors

### Pitfall: Using `on` instead of `eval` for switch

**Wrong**:
```
{~prompty.switch on="status"~}
```

**Correct**:
```
{~prompty.switch eval="status"~}
```

---

### Pitfall: Using `default` instead of `casedefault`

**Wrong**:
```
{~prompty.switch eval="status"~}
{~prompty.case value="active"~}Active{~/prompty.case~}
{~prompty.default~}Unknown{~/prompty.default~}
{~/prompty.switch~}
```

**Correct**:
```
{~prompty.switch eval="status"~}
{~prompty.case value="active"~}Active{~/prompty.case~}
{~prompty.casedefault~}Unknown{~/prompty.casedefault~}
{~/prompty.switch~}
```

---

### Pitfall: Adding whitespace/newlines between switch tags

**Wrong** (may cause parse errors):
```
{~prompty.switch eval="status"~}

{~prompty.case value="a"~}A{~/prompty.case~}

{~/prompty.switch~}
```

**Correct**:
```
{~prompty.switch eval="status"~}{~prompty.case value="a"~}A{~/prompty.case~}{~/prompty.switch~}
```

Or with content only inside cases:
```
{~prompty.switch eval="status"~}
{~prompty.case value="a"~}
Content A
{~/prompty.case~}
{~/prompty.switch~}
```

---

## Include Template Errors

### Pitfall: Using reserved namespace for template names

**Wrong**:
```go
engine.RegisterTemplate("prompty.mytemplate", "content")
```

**Correct**:
```go
engine.RegisterTemplate("mytemplate", "content")
engine.RegisterTemplate("my.template", "content")
```

Template names cannot start with `prompty.` as this is reserved for built-in tags.

---

### Pitfall: Circular template includes

**Wrong**:
```go
engine.RegisterTemplate("a", `{~prompty.include template="b" /~}`)
engine.RegisterTemplate("b", `{~prompty.include template="a" /~}`)
```

This will cause a stack overflow or max depth error. Design your templates to avoid circular dependencies.

---

### Pitfall: Including non-existent templates

**Problem**:
```
{~prompty.include template="nonexistent" /~}
```

This will fail at execution time. Use dry-run to validate template references:

```go
result := tmpl.DryRun(ctx, data)
for _, inc := range result.Includes {
    if !inc.Exists {
        fmt.Printf("Warning: template %s not found\n", inc.TemplateName)
    }
}
```

---

## Data Access Errors

### Pitfall: Accessing non-existent nested paths

**Problem**:
```
{~prompty.var name="user.profile.settings.theme" /~}
```

If any part of the path is nil or doesn't exist, this fails. Use defaults:

**Better**:
```
{~prompty.var name="user.profile.settings.theme" default="light" /~}
```

Or check with conditionals:
```
{~prompty.if eval="user.profile.settings.theme"~}
  {~prompty.var name="user.profile.settings.theme" /~}
{~prompty.else~}
  light
{~/prompty.if~}
```

---

### Pitfall: Type mismatches in comparisons

**Problem**:
```go
// Go code
data := map[string]any{"count": "5"} // string, not int

// Template
{~prompty.if eval="count > 3"~}  // Comparing string "5" > 3
```

Ensure types match. String "5" is not greater than integer 3 in expected ways.

**Better**:
```go
data := map[string]any{"count": 5} // int
```

---

### Pitfall: Assuming map order

**Problem**:
```
{~prompty.for item="key" in="config"~}
  {~prompty.var name="key" /~}
{~/prompty.for~}
```

Go maps have random iteration order. If order matters, use a slice:

```go
data := map[string]any{
    "orderedKeys": []string{"first", "second", "third"},
}
```

---

## Engine Configuration Errors

### Pitfall: Not reusing parsed templates

**Wrong** (parsing every time):
```go
for _, data := range items {
    result, _ := engine.Execute(ctx, templateString, data)
}
```

**Better** (parse once, execute many):
```go
tmpl, _ := engine.Parse(templateString)
for _, data := range items {
    result, _ := tmpl.Execute(ctx, data)
}
```

Parsing is expensive; templates are thread-safe for concurrent execution.

---

### Pitfall: Creating new engines per request

**Wrong**:
```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    engine := prompty.MustNew()  // Creating engine each request
    // ...
}
```

**Better** (single engine instance):
```go
var engine = prompty.MustNew()

func handleRequest(w http.ResponseWriter, r *http.Request) {
    result, _ := engine.Execute(ctx, template, data)
    // ...
}
```

Engines are thread-safe and should be reused.

---

### Pitfall: Forgetting to close StorageEngine

**Wrong**:
```go
func process() error {
    engine, _ := prompty.NewStorageEngine(config)
    // ... use engine
    return nil  // Storage engine never closed
}
```

**Better**:
```go
func process() error {
    engine, _ := prompty.NewStorageEngine(config)
    defer engine.Close()
    // ... use engine
    return nil
}
```

---

## Expression Errors

### Pitfall: Using Go syntax in expressions

**Wrong**:
```
{~prompty.if eval="user != nil && len(user.Name) > 0"~}
```

**Correct**:
```
{~prompty.if eval="user && len(user.Name) > 0"~}
```

go-prompty expressions use truthy/falsy semantics. `nil` is falsy, non-nil is truthy.

---

### Pitfall: Method calls on objects

**Wrong**:
```
{~prompty.if eval="user.IsAdmin()"~}
```

**Correct**:
Compute in Go and pass as data:
```go
data := map[string]any{
    "user":    user,
    "isAdmin": user.IsAdmin(),
}
```

```
{~prompty.if eval="isAdmin"~}
```

go-prompty expressions cannot call Go methods.

---

### Pitfall: Complex nested expressions

**Problem**:
```
{~prompty.if eval="users[0].roles[getIndex(user.id)].name == 'admin'"~}
```

This is too complex. Simplify in Go:

**Better**:
```go
data := map[string]any{
    "hasAdminRole": checkAdminRole(users, user),
}
```

```
{~prompty.if eval="hasAdminRole"~}
```

---

## Thread Safety Issues

### Pitfall: Modifying context data during execution

**Wrong**:
```go
data := map[string]any{"count": 0}

go func() {
    data["count"] = data["count"].(int) + 1  // Race condition!
}()

result, _ := engine.Execute(ctx, template, data)
```

**Better**:
Create separate data for each execution:
```go
for i := 0; i < n; i++ {
    data := map[string]any{"count": i}  // Fresh map each time
    go func(d map[string]any) {
        result, _ := engine.Execute(ctx, template, d)
    }(data)
}
```

---

### Pitfall: Sharing context across goroutines

**Wrong**:
```go
ctx := prompty.NewContext(data)
go func() {
    ctx.Set("key", "value1")  // Race!
}()
go func() {
    ctx.Get("key")  // Race!
}()
```

**Better**:
Use `WithValue` for derived contexts (creates copies):
```go
baseCtx := prompty.NewContext(data)
go func() {
    ctx := baseCtx.WithValue("key", "value1")  // Safe copy
    // use ctx
}()
```

---

## Validation Errors

### Pitfall: Not validating templates in CI/CD

**Problem**:
```go
// Template string with typo goes to production
template := `Hello {~prompty.varr name="user" /~}`  // "varr" not "var"
```

**Better**:
Add validation to your CI/CD pipeline:
```go
func TestTemplateValidity(t *testing.T) {
    engine := prompty.MustNew()

    templates := []string{
        greetingTemplate,
        emailTemplate,
        notificationTemplate,
    }

    for _, tmpl := range templates {
        result, err := engine.Validate(tmpl)
        require.NoError(t, err)
        assert.True(t, result.IsValid(), result.String())
    }
}
```

---

### Pitfall: Ignoring validation warnings

**Problem**:
```go
result, _ := engine.Validate(template)
if result.IsValid() {
    // Good to go!  ...but warnings ignored
}
```

**Better**:
```go
result, _ := engine.Validate(template)
if !result.IsValid() {
    return fmt.Errorf("template invalid: %s", result.String())
}
if len(result.Warnings) > 0 {
    log.Warn("template warnings", zap.Strings("warnings", result.Warnings))
}
```

---

## Access Control Pitfalls

### Pitfall: Not setting TenantID on templates

**Problem**:
```go
engine.SaveSecure(ctx, &prompty.StoredTemplate{
    Name:   "my-template",
    Source: "...",
    // TenantID not set!
}, subject)
```

The template may be accessible across tenants.

**Better**:
```go
engine.SaveSecure(ctx, &prompty.StoredTemplate{
    Name:     "my-template",
    Source:   "...",
    TenantID: subject.TenantID,  // Explicit tenant assignment
}, subject)
```

Or use a hook to auto-tag:
```go
engine.RegisterHook(prompty.HookBeforeSave, func(ctx context.Context, point prompty.HookPoint, data *prompty.HookData) error {
    if data.Template != nil && data.Subject != nil {
        if data.Template.TenantID == "" {
            data.Template.TenantID = data.Subject.TenantID
        }
    }
    return nil
})
```

---

### Pitfall: Assuming access checks happen automatically

**Problem**:
```go
// Using non-secure methods bypasses access control!
engine.Execute(ctx, "template", data)  // No access check!
```

**Correct**:
```go
// Use secure methods with subject
engine.ExecuteSecure(ctx, "template", data, subject)
```

---

## Performance Pitfalls

### Pitfall: Large template strings in loops

**Problem**:
```go
var result string
for _, item := range items {
    output, _ := engine.Execute(ctx, hugeTemplate, item)
    result += output  // String concatenation in loop
}
```

**Better**:
```go
var sb strings.Builder
tmpl, _ := engine.Parse(hugeTemplate)  // Parse once
for _, item := range items {
    output, _ := tmpl.Execute(ctx, item)
    sb.WriteString(output)
}
result := sb.String()
```

---

### Pitfall: Not using caching for storage

**Problem**:
```go
// Every execute hits storage
for i := 0; i < 1000; i++ {
    result, _ := storageEngine.Execute(ctx, "template", data)
}
```

**Better**:
```go
// Use cached storage
cachedStorage := prompty.NewCachedStorage(storage, prompty.CachedStorageConfig{
    TTL:        5 * time.Minute,
    MaxEntries: 1000,
})
```

---

## Debugging Tips

### Use DryRun for validation
```go
result := tmpl.DryRun(ctx, data)
fmt.Println(result.String())
```

### Use Explain for debugging
```go
result := tmpl.Explain(ctx, data)
fmt.Println(result.String())
```

### Enable debug logging
```go
engine := prompty.MustNew(
    prompty.WithLogger(zap.NewDevelopment()),
)
```

### Validate templates in tests
```go
func TestTemplates(t *testing.T) {
    engine := prompty.MustNew()

    result := tmpl.DryRun(context.Background(), testData)

    assert.Empty(t, result.MissingVariables, "missing vars: %v", result.MissingVariables)
    assert.Empty(t, result.Errors, "errors: %v", result.Errors)
}
```
