// Example: Debugging Templates with DryRun and Explain
//
// This example demonstrates go-prompty's debugging capabilities:
// - DryRun: Validates template structure without executing resolvers
// - Explain: Provides detailed execution explanation with AST and timing
//
// These features help identify:
// - Missing variables (with suggestions for similar names)
// - Unused data fields
// - Template includes that don't exist
// - Loop sources that are missing
// - Complete AST structure
// - Execution timing breakdown
//
// Run: go run ./examples/debugging
package main

import (
	"context"
	"fmt"

	"github.com/itsatony/go-prompty/v2"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== go-prompty Debugging Features ===")
	fmt.Println()

	// 1. Basic DryRun demonstration
	demonstrateDryRunBasic(ctx)

	// 2. DryRun with missing variables and suggestions
	demonstrateDryRunMissingVars(ctx)

	// 3. DryRun with unused data detection
	demonstrateDryRunUnusedData(ctx)

	// 4. DryRun with loops and conditionals
	demonstrateDryRunComplex(ctx)

	// 5. Explain execution
	demonstrateExplain(ctx)

	// 6. Validation API
	demonstrateValidation()
}

func demonstrateDryRunBasic(ctx context.Context) {
	fmt.Println("1. Basic DryRun")
	fmt.Println("   - Validates template without execution")
	fmt.Println()

	engine := prompty.MustNew()

	template := `Hello, {~prompty.var name="user.name" /~}!
Your email is {~prompty.var name="user.email" /~}.
Account type: {~prompty.var name="account_type" default="free" /~}`

	tmpl, err := engine.Parse(template)
	if err != nil {
		fmt.Println("   Parse error:", err)
		return
	}

	// Complete data
	data := map[string]any{
		"user": map[string]any{
			"name":  "Alice",
			"email": "alice@example.com",
		},
	}

	result := tmpl.DryRun(ctx, data)
	fmt.Printf("   Valid: %v\n", result.Valid)
	fmt.Printf("   Variables found: %d\n", len(result.Variables))
	for _, v := range result.Variables {
		status := "found"
		if !v.InData {
			if v.HasDefault {
				status = fmt.Sprintf("using default: %q", v.Default)
			} else {
				status = "MISSING"
			}
		}
		fmt.Printf("   - %s [line %d]: %s\n", v.Name, v.Line, status)
	}
	fmt.Println()
	fmt.Println("   Placeholder output:")
	fmt.Println("   " + result.Output)
	fmt.Println()
}

func demonstrateDryRunMissingVars(ctx context.Context) {
	fmt.Println("2. DryRun with Missing Variables")
	fmt.Println("   - Detects missing variables and suggests alternatives")
	fmt.Println()

	engine := prompty.MustNew()

	// Template with typos/wrong variable names
	template := `User: {~prompty.var name="usr.name" /~}
Email: {~prompty.var name="user.mail" /~}
Role: {~prompty.var name="role" /~}`

	tmpl, err := engine.Parse(template)
	if err != nil {
		fmt.Println("   Parse error:", err)
		return
	}

	// Data with correct variable names
	data := map[string]any{
		"user": map[string]any{
			"name":  "Bob",
			"email": "bob@example.com",
		},
		"user_role": "admin",
	}

	result := tmpl.DryRun(ctx, data)

	fmt.Println("   Variables Analysis:")
	for _, v := range result.Variables {
		if !v.InData && !v.HasDefault {
			fmt.Printf("   - %s [line %d]: MISSING\n", v.Name, v.Line)
			if len(v.Suggestions) > 0 {
				fmt.Printf("     Did you mean: %v?\n", v.Suggestions)
			}
		}
	}

	if len(result.MissingVariables) > 0 {
		fmt.Printf("\n   Missing variables: %v\n", result.MissingVariables)
	}
	fmt.Println()
}

func demonstrateDryRunUnusedData(ctx context.Context) {
	fmt.Println("3. DryRun with Unused Data Detection")
	fmt.Println("   - Identifies data fields not used in template")
	fmt.Println()

	engine := prompty.MustNew()

	// Simple template using only name
	template := `Hello, {~prompty.var name="name" /~}!`

	tmpl, err := engine.Parse(template)
	if err != nil {
		fmt.Println("   Parse error:", err)
		return
	}

	// Data with many fields, only one used
	data := map[string]any{
		"name":       "Charlie",
		"email":      "charlie@example.com",
		"phone":      "555-1234",
		"department": "Engineering",
		"title":      "Senior Developer",
	}

	result := tmpl.DryRun(ctx, data)

	fmt.Printf("   Variables used: %d\n", len(result.Variables))
	fmt.Printf("   Unused fields: %d\n", len(result.UnusedVariables))
	if len(result.UnusedVariables) > 0 {
		fmt.Println("   Unused data fields:")
		for _, v := range result.UnusedVariables {
			fmt.Printf("   - %s\n", v)
		}
	}
	fmt.Println()
}

func demonstrateDryRunComplex(ctx context.Context) {
	fmt.Println("4. DryRun with Complex Templates")
	fmt.Println("   - Analyzes loops, conditionals, and includes")
	fmt.Println()

	engine := prompty.MustNew()

	// Register a template for include analysis
	engine.MustRegisterTemplate("header", "=== Header ===")

	template := `{~prompty.include template="header" /~}

{~prompty.if eval="user.premium"~}
Premium User: {~prompty.var name="user.name" /~}
{~prompty.else~}
Free User: {~prompty.var name="user.name" /~}
{~/prompty.if~}

Your items:
{~prompty.for item="item" index="i" in="items" limit="10"~}
  {~prompty.var name="i" /~}. {~prompty.var name="item.name" /~}
{~/prompty.for~}

{~prompty.include template="footer" /~}`

	tmpl, err := engine.Parse(template)
	if err != nil {
		fmt.Println("   Parse error:", err)
		return
	}

	data := map[string]any{
		"user": map[string]any{
			"name":    "Dave",
			"premium": true,
		},
		"items": []map[string]any{
			{"name": "Item 1"},
			{"name": "Item 2"},
		},
	}

	result := tmpl.DryRun(ctx, data)

	fmt.Println("   Template Analysis:")
	fmt.Printf("   - Valid: %v\n", result.Valid)
	fmt.Printf("   - Variables: %d\n", len(result.Variables))
	fmt.Printf("   - Conditionals: %d\n", len(result.Conditionals))
	fmt.Printf("   - Loops: %d\n", len(result.Loops))
	fmt.Printf("   - Includes: %d\n", len(result.Includes))

	if len(result.Conditionals) > 0 {
		fmt.Println("\n   Conditionals:")
		for _, c := range result.Conditionals {
			fmt.Printf("   - [line %d] if %s (has else: %v)\n", c.Line, c.Condition, c.HasElse)
		}
	}

	if len(result.Loops) > 0 {
		fmt.Println("\n   Loops:")
		for _, l := range result.Loops {
			status := "source found"
			if !l.InData {
				status = "source NOT FOUND"
			}
			fmt.Printf("   - [line %d] for %s in %s (limit: %d) - %s\n", l.Line, l.ItemVar, l.Source, l.Limit, status)
		}
	}

	if len(result.Includes) > 0 {
		fmt.Println("\n   Template Includes:")
		for _, inc := range result.Includes {
			status := "found"
			if !inc.Exists {
				status = "NOT FOUND"
			}
			fmt.Printf("   - [line %d] %s - %s\n", inc.Line, inc.TemplateName, status)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("\n   Warnings:")
		for _, w := range result.Warnings {
			fmt.Printf("   - %s\n", w)
		}
	}
	fmt.Println()
}

func demonstrateExplain(ctx context.Context) {
	fmt.Println("5. Explain Execution")
	fmt.Println("   - Shows AST structure and execution details")
	fmt.Println()

	engine := prompty.MustNew()

	template := `{~prompty.if eval="premium"~}Welcome Premium {~prompty.var name="name" /~}!{~prompty.else~}Hello {~prompty.var name="name" default="Guest" /~}!{~/prompty.if~}`

	tmpl, err := engine.Parse(template)
	if err != nil {
		fmt.Println("   Parse error:", err)
		return
	}

	data := map[string]any{
		"name":    "Eve",
		"premium": false,
	}

	result := tmpl.Explain(ctx, data)

	fmt.Println("   AST Structure:")
	lines := splitLines(result.AST)
	for _, line := range lines {
		if line != "" {
			fmt.Println("   " + line)
		}
	}

	fmt.Println("\n   Variable Accesses:")
	for _, v := range result.Variables {
		var status string
		if v.Found {
			status = fmt.Sprintf("= %v", v.Value)
		} else if v.Default != "" {
			status = fmt.Sprintf("using default: %q", v.Default)
		} else {
			status = "NOT FOUND"
		}
		fmt.Printf("   - [line %d] %s: %s\n", v.Line, v.Path, status)
	}

	fmt.Println("\n   Timing:")
	fmt.Printf("   - Total: %v\n", result.Timing.Total)
	fmt.Printf("   - Execution: %v\n", result.Timing.Execution)

	if result.Error != nil {
		fmt.Printf("\n   Error: %v\n", result.Error)
	}

	fmt.Println("\n   Output:")
	fmt.Println("   " + result.Output)
	fmt.Println()
}

func demonstrateValidation() {
	fmt.Println("6. Validation API")
	fmt.Println("   - Validates template syntax without data")
	fmt.Println()

	engine := prompty.MustNew()

	// Valid template
	validTemplate := `Hello, {~prompty.var name="user" /~}!`
	result, _ := engine.Validate(validTemplate)
	fmt.Printf("   Valid template: IsValid=%v, Errors=%d, Warnings=%d\n",
		result.IsValid(), len(result.Errors()), len(result.Warnings()))

	// Template with unknown tag (warning)
	unknownTagTemplate := `Hello, {~unknown.tag name="test" /~}!`
	result, _ = engine.Validate(unknownTagTemplate)
	fmt.Printf("   Unknown tag: IsValid=%v, Errors=%d, Warnings=%d\n",
		result.IsValid(), len(result.Errors()), len(result.Warnings()))
	if result.HasWarnings() {
		fmt.Println("   Warnings:")
		for _, w := range result.Warnings() {
			fmt.Printf("   - [line %d] %s: %s\n", w.Position.Line, w.TagName, w.Message)
		}
	}

	// Template with invalid for loop (error)
	invalidForTemplate := `{~prompty.for in="items"~}missing item attr{~/prompty.for~}`
	result, _ = engine.Validate(invalidForTemplate)
	fmt.Printf("   Invalid for loop: IsValid=%v, Errors=%d, Warnings=%d\n",
		result.IsValid(), len(result.Errors()), len(result.Warnings()))
	if result.HasErrors() {
		fmt.Println("   Errors:")
		for _, e := range result.Errors() {
			fmt.Printf("   - [line %d] %s: %s\n", e.Position.Line, e.TagName, e.Message)
		}
	}
	fmt.Println()

	fmt.Println("=== Summary ===")
	fmt.Println("Debugging tools help you:")
	fmt.Println("  - DryRun: Find missing variables, unused data, broken includes")
	fmt.Println("  - Explain: Understand AST structure and execution flow")
	fmt.Println("  - Validate: Check syntax before providing data")
}

// splitLines splits a string into lines for formatted output
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
