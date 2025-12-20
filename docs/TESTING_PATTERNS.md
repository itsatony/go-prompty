# Template Testing Patterns

This guide covers best practices for testing go-prompty templates in your application.

## Table of Contents

1. [Basic Testing Setup](#basic-testing-setup)
2. [Validation Testing](#validation-testing)
3. [Execution Testing](#execution-testing)
4. [Dry-Run Testing](#dry-run-testing)
5. [Snapshot Testing](#snapshot-testing)
6. [Edge Case Testing](#edge-case-testing)
7. [Performance Testing](#performance-testing)
8. [CI/CD Integration](#cicd-integration)

---

## Basic Testing Setup

### Create a Test Engine

```go
package templates_test

import (
    "context"
    "testing"

    prompty "github.com/itsatony/go-prompty"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func newTestEngine(t *testing.T) *prompty.Engine {
    t.Helper()
    engine := prompty.MustNew(
        prompty.WithErrorStrategy(prompty.ErrorStrategyThrow),
    )
    return engine
}
```

### Test Helper for Templates

```go
func testTemplate(t *testing.T, template string, data map[string]any, expected string) {
    t.Helper()
    engine := newTestEngine(t)

    result, err := engine.Execute(context.Background(), template, data)
    require.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

---

## Validation Testing

### Test Template Syntax

```go
func TestTemplateValidation(t *testing.T) {
    engine := newTestEngine(t)

    tests := []struct {
        name     string
        template string
        valid    bool
    }{
        {
            name:     "valid variable",
            template: `Hello {~prompty.var name="user" /~}!`,
            valid:    true,
        },
        {
            name:     "valid conditional",
            template: `{~prompty.if eval="show"~}visible{~/prompty.if~}`,
            valid:    true,
        },
        {
            name:     "invalid - unclosed tag",
            template: `{~prompty.if eval="show"~}visible`,
            valid:    false,
        },
        {
            name:     "invalid - missing attribute",
            template: `{~prompty.var /~}`,
            valid:    false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := engine.Validate(tt.template)
            if tt.valid {
                require.NoError(t, err)
                assert.True(t, result.IsValid(), result.String())
            } else {
                if err == nil {
                    assert.False(t, result.IsValid())
                }
            }
        })
    }
}
```

### Test All Production Templates

```go
func TestAllProductionTemplates(t *testing.T) {
    engine := newTestEngine(t)

    // Load all templates from your template directory
    templates := map[string]string{
        "greeting":     GreetingTemplate,
        "notification": NotificationTemplate,
        "email":        EmailTemplate,
    }

    for name, tmpl := range templates {
        t.Run(name, func(t *testing.T) {
            result, err := engine.Validate(tmpl)
            require.NoError(t, err, "template %s failed to parse", name)
            assert.True(t, result.IsValid(), "template %s invalid: %s", name, result.String())
        })
    }
}
```

---

## Execution Testing

### Basic Execution Test

```go
func TestGreetingTemplate(t *testing.T) {
    engine := newTestEngine(t)

    template := `Hello {~prompty.var name="user" /~}!`

    tests := []struct {
        name     string
        data     map[string]any
        expected string
        wantErr  bool
    }{
        {
            name:     "with user",
            data:     map[string]any{"user": "Alice"},
            expected: "Hello Alice!",
        },
        {
            name:    "missing user throws",
            data:    map[string]any{},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := engine.Execute(context.Background(), template, tt.data)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                require.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

### Test Conditional Logic

```go
func TestConditionalTemplate(t *testing.T) {
    engine := newTestEngine(t)

    template := `
{~prompty.if eval="user.isAdmin"~}
Welcome, Admin {~prompty.var name="user.name" /~}!
{~prompty.elseif eval="user.isLoggedIn"~}
Welcome back, {~prompty.var name="user.name" /~}!
{~prompty.else~}
Please log in.
{~/prompty.if~}`

    tests := []struct {
        name     string
        data     map[string]any
        contains string
    }{
        {
            name: "admin user",
            data: map[string]any{
                "user": map[string]any{
                    "name":       "Alice",
                    "isAdmin":    true,
                    "isLoggedIn": true,
                },
            },
            contains: "Admin Alice",
        },
        {
            name: "logged in user",
            data: map[string]any{
                "user": map[string]any{
                    "name":       "Bob",
                    "isAdmin":    false,
                    "isLoggedIn": true,
                },
            },
            contains: "Welcome back, Bob",
        },
        {
            name: "guest",
            data: map[string]any{
                "user": map[string]any{
                    "isAdmin":    false,
                    "isLoggedIn": false,
                },
            },
            contains: "Please log in",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := engine.Execute(context.Background(), template, tt.data)
            require.NoError(t, err)
            assert.Contains(t, result, tt.contains)
        })
    }
}
```

### Test Loop Rendering

```go
func TestLoopTemplate(t *testing.T) {
    engine := newTestEngine(t)

    template := `Items:{~prompty.for item="item" index="i" in="items"~}
- {~prompty.var name="i" /~}: {~prompty.var name="item" /~}{~/prompty.for~}`

    result, err := engine.Execute(context.Background(), template, map[string]any{
        "items": []string{"apple", "banana", "cherry"},
    })

    require.NoError(t, err)
    assert.Contains(t, result, "0: apple")
    assert.Contains(t, result, "1: banana")
    assert.Contains(t, result, "2: cherry")
}
```

---

## Dry-Run Testing

### Validate Data Requirements

```go
func TestTemplateDryRun(t *testing.T) {
    engine := newTestEngine(t)

    template := `
Hello {~prompty.var name="user.name" /~}!
Your email: {~prompty.var name="user.email" default="not provided" /~}
{~prompty.if eval="user.premium"~}
Premium features: {~prompty.var name="features" /~}
{~/prompty.if~}`

    tmpl, err := engine.Parse(template)
    require.NoError(t, err)

    // Test with complete data
    t.Run("complete data", func(t *testing.T) {
        result := tmpl.DryRun(context.Background(), map[string]any{
            "user": map[string]any{
                "name":    "Alice",
                "email":   "alice@example.com",
                "premium": true,
            },
            "features": "unlimited storage",
        })

        assert.True(t, result.Valid)
        assert.Empty(t, result.MissingVariables)
        assert.Empty(t, result.Errors)
    })

    // Test with minimal data
    t.Run("minimal data shows missing", func(t *testing.T) {
        result := tmpl.DryRun(context.Background(), map[string]any{})

        assert.True(t, result.Valid) // Structure is valid
        assert.Contains(t, result.MissingVariables, "user.name")
        // user.email has default, so not in missing
        // features is in conditional, may or may not be flagged
    })
}
```

### Test Variable Suggestions

```go
func TestVariableSuggestions(t *testing.T) {
    engine := newTestEngine(t)

    // Template with typo
    template := `Hello {~prompty.var name="usre" /~}!` // typo: usre

    tmpl, err := engine.Parse(template)
    require.NoError(t, err)

    result := tmpl.DryRun(context.Background(), map[string]any{
        "user": "Alice",
    })

    // Should suggest "user" for "usre"
    require.Len(t, result.Variables, 1)
    assert.Contains(t, result.Variables[0].Suggestions, "user")
}
```

---

## Snapshot Testing

### Golden File Testing

```go
func TestTemplateSnapshot(t *testing.T) {
    engine := newTestEngine(t)

    template := `... your complex template ...`

    data := map[string]any{
        "user":  "Alice",
        "items": []string{"a", "b", "c"},
    }

    result, err := engine.Execute(context.Background(), template, data)
    require.NoError(t, err)

    // Compare with golden file
    golden := "testdata/template_expected.txt"

    if *update {
        os.WriteFile(golden, []byte(result), 0644)
    }

    expected, _ := os.ReadFile(golden)
    assert.Equal(t, string(expected), result)
}
```

### Using Cupaloy for Snapshots

```go
import "github.com/bradleyjkemp/cupaloy"

func TestTemplateSnapshots(t *testing.T) {
    engine := newTestEngine(t)

    template := `...`

    result, err := engine.Execute(context.Background(), template, testData)
    require.NoError(t, err)

    cupaloy.SnapshotT(t, result)
}
```

---

## Edge Case Testing

### Test Empty Inputs

```go
func TestEdgeCases(t *testing.T) {
    engine := newTestEngine(t)

    tests := []struct {
        name     string
        template string
        data     map[string]any
        expected string
    }{
        {
            name:     "empty data",
            template: `Hello World`,
            data:     nil,
            expected: "Hello World",
        },
        {
            name:     "empty string value",
            template: `Value: {~prompty.var name="val" /~}`,
            data:     map[string]any{"val": ""},
            expected: "Value: ",
        },
        {
            name:     "empty slice",
            template: `{~prompty.for item="x" in="items"~}X{~/prompty.for~}`,
            data:     map[string]any{"items": []string{}},
            expected: "",
        },
        {
            name:     "nil slice",
            template: `{~prompty.for item="x" in="items"~}X{~/prompty.for~}`,
            data:     map[string]any{"items": nil},
            expected: "",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := engine.Execute(context.Background(), tt.template, tt.data)
            require.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Test Special Characters

```go
func TestSpecialCharacters(t *testing.T) {
    engine := newTestEngine(t)

    tests := []struct {
        name     string
        data     map[string]any
        expected string
    }{
        {
            name:     "html characters",
            data:     map[string]any{"val": "<script>alert('xss')</script>"},
            expected: "<script>alert('xss')</script>",
        },
        {
            name:     "newlines",
            data:     map[string]any{"val": "line1\nline2"},
            expected: "line1\nline2",
        },
        {
            name:     "unicode",
            data:     map[string]any{"val": "Hello 世界 🌍"},
            expected: "Hello 世界 🌍",
        },
        {
            name:     "quotes",
            data:     map[string]any{"val": `He said "Hello"`},
            expected: `He said "Hello"`,
        },
    }

    template := `{~prompty.var name="val" /~}`

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := engine.Execute(context.Background(), template, tt.data)
            require.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Test Escaped Delimiters

```go
func TestEscapedDelimiters(t *testing.T) {
    engine := newTestEngine(t)

    // Escaped delimiter should produce literal {~
    template := `Syntax: \{~prompty.var~} is a variable tag`

    result, err := engine.Execute(context.Background(), template, nil)
    require.NoError(t, err)
    assert.Contains(t, result, "{~prompty.var~}")
}
```

---

## Performance Testing

### Benchmark Parsing

```go
func BenchmarkParsing(b *testing.B) {
    engine := prompty.MustNew()
    template := `Hello {~prompty.var name="user" /~}! You have {~prompty.var name="count" /~} messages.`

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = engine.Parse(template)
    }
}
```

### Benchmark Execution

```go
func BenchmarkExecution(b *testing.B) {
    engine := prompty.MustNew()
    template := `Hello {~prompty.var name="user" /~}! You have {~prompty.var name="count" /~} messages.`
    tmpl, _ := engine.Parse(template)
    data := map[string]any{"user": "Alice", "count": 5}
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = tmpl.Execute(ctx, data)
    }
}
```

### Benchmark Concurrent Execution

```go
func BenchmarkConcurrentExecution(b *testing.B) {
    engine := prompty.MustNew()
    template := `Hello {~prompty.var name="user" /~}!`
    tmpl, _ := engine.Parse(template)
    ctx := context.Background()

    b.RunParallel(func(pb *testing.PB) {
        data := map[string]any{"user": "Alice"}
        for pb.Next() {
            _, _ = tmpl.Execute(ctx, data)
        }
    })
}
```

---

## CI/CD Integration

### GitHub Actions Workflow

```yaml
name: Template Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run template tests
        run: go test -v -race ./templates/...

      - name: Run template validation
        run: go run ./cmd/validate-templates

      - name: Check template coverage
        run: |
          go test -coverprofile=coverage.out ./templates/...
          go tool cover -func=coverage.out | grep total
```

### Pre-commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

# Validate all templates before commit
go test -run "TestTemplate" ./templates/...
if [ $? -ne 0 ]; then
    echo "Template tests failed. Commit aborted."
    exit 1
fi
```

### Template Validation Script

```go
// cmd/validate-templates/main.go
package main

import (
    "fmt"
    "os"
    "path/filepath"

    prompty "github.com/itsatony/go-prompty"
)

func main() {
    engine := prompty.MustNew()

    err := filepath.Walk("templates", func(path string, info os.FileInfo, err error) error {
        if err != nil || info.IsDir() || filepath.Ext(path) != ".tmpl" {
            return nil
        }

        content, _ := os.ReadFile(path)
        result, err := engine.Validate(string(content))
        if err != nil {
            return fmt.Errorf("%s: %w", path, err)
        }
        if !result.IsValid() {
            return fmt.Errorf("%s: %s", path, result.String())
        }

        fmt.Printf("✓ %s\n", path)
        return nil
    })

    if err != nil {
        fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("All templates valid!")
}
```

---

## Test Organization

### Recommended Directory Structure

```
project/
├── templates/
│   ├── greeting.tmpl
│   ├── notification.tmpl
│   └── email.tmpl
├── templates_test.go          # Template execution tests
├── templates_validation_test.go   # Validation tests
├── templates_benchmark_test.go    # Performance tests
└── testdata/
    └── snapshots/             # Golden files
        ├── greeting_expected.txt
        └── notification_expected.txt
```

### Test Categories

```go
// templates_test.go - execution tests
func TestExecution_Greeting(t *testing.T) { ... }
func TestExecution_Notification(t *testing.T) { ... }

// templates_validation_test.go - validation tests
func TestValidation_AllTemplates(t *testing.T) { ... }
func TestValidation_DryRun(t *testing.T) { ... }

// templates_benchmark_test.go - performance tests
func BenchmarkGreeting(b *testing.B) { ... }
func BenchmarkNotification(b *testing.B) { ... }
```

---

## Summary

1. **Always validate** templates during CI/CD
2. **Use dry-run** to check data requirements before execution
3. **Test edge cases**: empty data, special characters, nil values
4. **Benchmark** critical templates for performance
5. **Use table-driven tests** for comprehensive coverage
6. **Consider snapshot testing** for complex outputs
7. **Test error strategies** explicitly
8. **Run with race detector**: `go test -race`
