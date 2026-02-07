// Example: Catalog Generation
//
// This example demonstrates v2.1 catalog generation for skills and tools:
// - Default format (markdown with bullet points)
// - Detailed format (full descriptions, parameters, injection modes)
// - Compact format (single-line, space-efficient)
// - Function calling format (JSON schema for tool use APIs)
// - Using catalogs within agent compilation
//
// Run: go run ./examples/catalog_generation
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/itsatony/go-prompty/v2"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Catalog Generation Example ===")
	fmt.Println()

	// -------------------------------------------------------
	// Set up skills and tools for catalog generation
	// -------------------------------------------------------

	skills := []prompty.SkillRef{
		{Slug: "web-search", Injection: prompty.SkillInjectionSystemPrompt},
		{Slug: "summarizer", Injection: prompty.SkillInjectionUserContext},
		{Slug: "translator@v2", Injection: prompty.SkillInjectionSystemPrompt},
	}

	// Create a resolver so catalogs can pull skill descriptions
	resolver := prompty.NewMapDocumentResolver()
	resolver.AddSkill("web-search", &prompty.Prompt{
		Name:        "web-search",
		Description: "Searches the web using multiple search engines and returns relevant results",
		Type:        prompty.DocumentTypeSkill,
	})
	resolver.AddSkill("summarizer", &prompty.Prompt{
		Name:        "summarizer",
		Description: "Summarizes documents into concise bullet-point summaries",
		Type:        prompty.DocumentTypeSkill,
	})
	resolver.AddSkill("translator", &prompty.Prompt{
		Name:        "translator",
		Description: "Translates text between languages with context awareness",
		Type:        prompty.DocumentTypeSkill,
	})

	tools := &prompty.ToolsConfig{
		Functions: []*prompty.FunctionDef{
			{
				Name:        "search_web",
				Description: "Search the web for information",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{
							"type":        "string",
							"description": "Search query string",
						},
						"max_results": map[string]any{
							"type":        "integer",
							"description": "Maximum number of results to return",
						},
					},
					"required": []any{"query"},
				},
			},
			{
				Name:        "read_file",
				Description: "Read contents of a file",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "File path to read",
						},
					},
					"required": []any{"path"},
				},
			},
		},
		MCPServers: []*prompty.MCPServer{
			{
				Name:      "knowledge-base",
				URL:       "https://kb.example.com/mcp",
				Transport: "sse",
				Tools:     []string{"search_docs", "get_article"},
			},
		},
	}

	// -------------------------------------------------------
	// 1. Skills Catalog — Default Format
	// -------------------------------------------------------

	fmt.Println("1. Skills Catalog — Default Format")
	fmt.Println("   (Markdown bullet list)")
	fmt.Println()

	catalog, err := prompty.GenerateSkillsCatalog(ctx, skills, resolver, prompty.CatalogFormatDefault)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(catalog)

	// -------------------------------------------------------
	// 2. Skills Catalog — Detailed Format
	// -------------------------------------------------------

	fmt.Println("2. Skills Catalog — Detailed Format")
	fmt.Println("   (Full descriptions with injection modes and versions)")
	fmt.Println()

	catalog, err = prompty.GenerateSkillsCatalog(ctx, skills, resolver, prompty.CatalogFormatDetailed)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(catalog)

	// -------------------------------------------------------
	// 3. Skills Catalog — Compact Format
	// -------------------------------------------------------

	fmt.Println("3. Skills Catalog — Compact Format")
	fmt.Println("   (Single-line, semicolon separated)")
	fmt.Println()

	catalog, err = prompty.GenerateSkillsCatalog(ctx, skills, resolver, prompty.CatalogFormatCompact)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   " + catalog)
	fmt.Println()

	// -------------------------------------------------------
	// 4. Tools Catalog — Default Format
	// -------------------------------------------------------

	fmt.Println("4. Tools Catalog — Default Format")
	fmt.Println()

	catalog, err = prompty.GenerateToolsCatalog(tools, prompty.CatalogFormatDefault)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(catalog)

	// -------------------------------------------------------
	// 5. Tools Catalog — Detailed Format
	// -------------------------------------------------------

	fmt.Println("5. Tools Catalog — Detailed Format")
	fmt.Println("   (Full descriptions with parameter schemas)")
	fmt.Println()

	catalog, err = prompty.GenerateToolsCatalog(tools, prompty.CatalogFormatDetailed)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(catalog)

	// -------------------------------------------------------
	// 6. Tools Catalog — Function Calling Format
	// -------------------------------------------------------

	fmt.Println("6. Tools Catalog — Function Calling Format")
	fmt.Println("   (JSON schema for OpenAI-style tool use)")
	fmt.Println()

	catalog, err = prompty.GenerateToolsCatalog(tools, prompty.CatalogFormatFunctionCalling)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(catalog)

	// -------------------------------------------------------
	// 7. Tools Catalog — Compact Format
	// -------------------------------------------------------

	fmt.Println("7. Tools Catalog — Compact Format")
	fmt.Println()

	catalog, err = prompty.GenerateToolsCatalog(tools, prompty.CatalogFormatCompact)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   " + catalog)
	fmt.Println()

	// -------------------------------------------------------
	// 8. Catalogs in agent compilation
	// -------------------------------------------------------

	fmt.Println("8. Catalogs embedded in agent compilation")
	fmt.Println("   (Skills catalog injected via {~prompty.skills_catalog~} tag)")
	fmt.Println()

	agentYAML := `---
name: catalog-demo-agent
description: Demonstrates catalog injection during compilation
type: agent
execution:
  provider: openai
  model: gpt-4
skills:
  - slug: web-search
    injection: system_prompt
  - slug: summarizer
    injection: user_context
messages:
  - role: system
    content: |
      You are a research assistant with the following capabilities:

      {~prompty.skills_catalog format="detailed" /~}
  - role: user
    content: '{~prompty.var name="input.query" default="Hello" /~}'
---
Catalog demo agent ready.
`

	agent, err := prompty.Parse([]byte(agentYAML))
	if err != nil {
		log.Fatal(err)
	}

	compiled, err := agent.CompileAgent(ctx, map[string]any{
		"query": "Summarize recent AI papers",
	}, &prompty.CompileOptions{
		Resolver:            resolver,
		SkillsCatalogFormat: prompty.CatalogFormatDetailed,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   System message with embedded catalog:")
	if len(compiled.Messages) > 0 {
		preview := compiled.Messages[0].Content
		if len(preview) > 300 {
			preview = preview[:300] + "..."
		}
		fmt.Println(preview)
	}

	fmt.Println("\n=== Example Complete ===")
}
