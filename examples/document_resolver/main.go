// Example: Document Resolver
//
// This example demonstrates v2.1 DocumentResolver implementations:
// - MapDocumentResolver for in-memory skill/prompt/agent resolution
// - Using resolvers during agent compilation
// - StorageDocumentResolver backed by a template storage engine
// - How resolved documents flow into catalog generation
//
// Run: go run ./examples/document_resolver
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/itsatony/go-prompty/v2"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Document Resolver Example ===")
	fmt.Println()

	// -------------------------------------------------------
	// 1. MapDocumentResolver — in-memory resolver
	// -------------------------------------------------------

	fmt.Println("1. MapDocumentResolver (in-memory)")
	fmt.Println("   Creating resolver and registering documents...")

	resolver := prompty.NewMapDocumentResolver()

	// Register a skill
	resolver.AddSkill("text-formatter", &prompty.Prompt{
		Name:        "text-formatter",
		Description: "Formats text into structured output with headers and bullet points",
		Type:        prompty.DocumentTypeSkill,
		Body:        "Format the input text with clear headers and organized bullet points.",
	})

	// Register another skill
	resolver.AddSkill("sentiment-analyzer", &prompty.Prompt{
		Name:        "sentiment-analyzer",
		Description: "Analyzes text sentiment and returns a classification",
		Type:        prompty.DocumentTypeSkill,
		Body:        "Analyze the sentiment of the provided text. Classify as positive, negative, or neutral.",
		Execution: &prompty.ExecutionConfig{
			Provider:    prompty.ProviderOpenAI,
			Model:       "gpt-4",
			Temperature: floatPtr(0.1),
		},
	})

	// Register a prompt
	resolver.AddPrompt("greeting", &prompty.Prompt{
		Name:        "greeting",
		Description: "A friendly greeting prompt",
		Type:        prompty.DocumentTypePrompt,
		Body:        "Hello! How can I help you today?",
	})

	// Resolve and display
	skill, err := resolver.ResolveSkill(ctx, "text-formatter")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Resolved skill: %s — %s\n", skill.Name, skill.Description)

	prompt, err := resolver.ResolvePrompt(ctx, "greeting")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Resolved prompt: %s — %s\n", prompt.Name, prompt.Description)

	// Demonstrate not-found error
	_, err = resolver.ResolveSkill(ctx, "nonexistent")
	fmt.Printf("   Not found error: %v\n", err)
	fmt.Println()

	// -------------------------------------------------------
	// 2. Using resolver in agent compilation
	// -------------------------------------------------------

	fmt.Println("2. Using resolver in agent compilation")

	agentYAML := `---
name: content-processor
description: Processes and analyzes text content
type: agent
execution:
  provider: openai
  model: gpt-4
  temperature: 0.5
skills:
  - slug: text-formatter
    injection: system_prompt
  - slug: sentiment-analyzer
    injection: user_context
messages:
  - role: system
    content: |
      You are a content processing agent.

      {~prompty.skills_catalog /~}
  - role: user
    content: 'Process: {~prompty.var name="input.text" /~}'
---
Content processing agent ready.
`

	agent, err := prompty.Parse([]byte(agentYAML))
	if err != nil {
		log.Fatal("Failed to parse agent:", err)
	}

	compiled, err := agent.CompileAgent(ctx, map[string]any{
		"text": "This product is amazing! Best purchase I've made all year.",
	}, &prompty.CompileOptions{
		Resolver: resolver,
	})
	if err != nil {
		log.Fatal("Failed to compile:", err)
	}

	fmt.Printf("   Compiled %d messages\n", len(compiled.Messages))
	for i, msg := range compiled.Messages {
		preview := msg.Content
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		fmt.Printf("   [%d] %s: %s\n", i, msg.Role, preview)
	}
	fmt.Println()

	// -------------------------------------------------------
	// 3. StorageDocumentResolver — storage-backed resolver
	// -------------------------------------------------------

	fmt.Println("3. StorageDocumentResolver (storage-backed)")

	storage := prompty.NewMemoryStorage()
	se, err := prompty.NewStorageEngine(prompty.StorageEngineConfig{
		Storage: storage,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer se.Close()

	// Save templates with YAML frontmatter so PromptConfig is extracted
	err = se.Save(ctx, &prompty.StoredTemplate{
		Name: "email-writer",
		Source: `---
name: email-writer
description: Writes professional emails
type: skill
execution:
  provider: openai
  model: gpt-4
  temperature: 0.7
---
Write a professional email based on the provided context.`,
		Tags: []string{"writing", "email"},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Plain template (no frontmatter) — StorageDocumentResolver handles this gracefully
	err = se.Save(ctx, &prompty.StoredTemplate{
		Name:   "simple-helper",
		Source: "You are a helpful assistant. Answer the user's question concisely.",
		Tags:   []string{"general"},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create a StorageDocumentResolver
	storageResolver := prompty.NewStorageDocumentResolver(storage)

	// Resolve a template with PromptConfig
	resolved, err := storageResolver.ResolveSkill(ctx, "email-writer")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Resolved from storage: %s\n", resolved.Name)
	fmt.Printf("   Description: %s\n", resolved.Description)
	if resolved.Execution != nil {
		fmt.Printf("   Model: %s\n", resolved.Execution.Model)
	}

	// Resolve a plain template (returns body from Source)
	plain, err := storageResolver.ResolvePrompt(ctx, "simple-helper")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Plain template body: %s\n", plain.Body)
	fmt.Println()

	// -------------------------------------------------------
	// 4. NoopDocumentResolver — always errors
	// -------------------------------------------------------

	fmt.Println("4. NoopDocumentResolver (always returns errors)")
	noop := &prompty.NoopDocumentResolver{}
	_, err = noop.ResolveSkill(ctx, "anything")
	fmt.Printf("   Result: %v\n", err)

	fmt.Println("\n=== Example Complete ===")
}

func floatPtr(f float64) *float64 {
	return &f
}
