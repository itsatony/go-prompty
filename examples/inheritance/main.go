// Package main demonstrates go-prompty template inheritance.
//
// Template inheritance allows creating base templates with overridable blocks.
// Child templates can extend parents and selectively override blocks while
// optionally preserving parent content using the parent tag.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/itsatony/go-prompty/v2"
)

func main() {
	engine := prompty.MustNew()

	// Register base template with default blocks
	engine.MustRegisterTemplate("base-prompt", `{~prompty.block name="system"~}You are a helpful assistant.{~/prompty.block~}

{~prompty.block name="context"~}{~/prompty.block~}

{~prompty.block name="instructions"~}Please be concise and accurate.{~/prompty.block~}`)

	// Register intermediate template that extends base
	engine.MustRegisterTemplate("support-prompt", `{~prompty.extends template="base-prompt" /~}

{~prompty.block name="system"~}You are a customer support agent for {~prompty.var name="company" default="our company" /~}.{~/prompty.block~}

{~prompty.block name="instructions"~}
{~prompty.parent /~}
Always be empathetic and offer to escalate if the customer is frustrated.
{~/prompty.block~}`)

	ctx := context.Background()

	// Example 1: Execute base template directly
	fmt.Println("=== Example 1: Base Template ===")
	result, err := engine.ExecuteTemplate(ctx, "base-prompt", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
	fmt.Println()

	// Example 2: Execute child template that extends base
	fmt.Println("=== Example 2: Child Template (extends base) ===")
	result, err = engine.ExecuteTemplate(ctx, "support-prompt", map[string]any{
		"company": "Acme Corp",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
	fmt.Println()

	// Example 3: Execute inline template that extends registered template
	fmt.Println("=== Example 3: Inline Template (extends support-prompt) ===")
	inlineTemplate := `{~prompty.extends template="support-prompt" /~}

{~prompty.block name="context"~}
Customer: {~prompty.var name="customer.name" /~}
Issue: {~prompty.var name="issue" /~}
Priority: {~prompty.var name="priority" default="normal" /~}
{~/prompty.block~}`

	result, err = engine.Execute(ctx, inlineTemplate, map[string]any{
		"company": "TechStart Inc",
		"customer": map[string]any{
			"name": "Alice Johnson",
		},
		"issue":    "Unable to login after password reset",
		"priority": "high",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
	fmt.Println()

	// Example 4: Multi-level inheritance (3 levels deep)
	fmt.Println("=== Example 4: Multi-level Inheritance ===")

	// Register templates for 3-level hierarchy
	engine.MustRegisterTemplate("layout", `{~prompty.block name="header"~}[Header]{~/prompty.block~}
{~prompty.block name="body"~}[Body]{~/prompty.block~}
{~prompty.block name="footer"~}[Footer]{~/prompty.block~}`)

	engine.MustRegisterTemplate("page-layout", `{~prompty.extends template="layout" /~}
{~prompty.block name="header"~}== {~prompty.var name="title" default="Untitled" /~} =={~/prompty.block~}
{~prompty.block name="footer"~}--- {~prompty.var name="copyright" default="2024" /~} ---{~/prompty.block~}`)

	thirdLevel := `{~prompty.extends template="page-layout" /~}
{~prompty.block name="body"~}
Welcome to our page!
This content overrides the body block.
{~/prompty.block~}`

	result, err = engine.Execute(ctx, thirdLevel, map[string]any{
		"title":     "Welcome Page",
		"copyright": "2024 Example Inc",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result)
}
