// Example: Agent Compilation
//
// This example demonstrates the v2.1 agent compilation pipeline:
// - Defining an agent with YAML frontmatter (skills, tools, constraints, messages)
// - Parsing the agent definition with prompty.Parse()
// - Setting up a MapDocumentResolver for skill resolution
// - Compiling the agent with CompileAgent()
// - Accessing compiled messages, execution config, tools, and constraints
// - Activating a specific skill with ActivateSkill()
//
// Run: go run ./examples/agent_compilation
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/itsatony/go-prompty"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Agent Compilation Example ===")
	fmt.Println()

	// -------------------------------------------------------
	// 1. Define an agent using YAML frontmatter
	// -------------------------------------------------------

	agentYAML := `---
name: research-agent
description: AI research assistant that helps find and summarize information
type: agent
execution:
  provider: openai
  model: gpt-4
  temperature: 0.3
  max_tokens: 2048
skills:
  - slug: web-search
    injection: system_prompt
  - slug: summarizer
    injection: user_context
tools:
  functions:
    - name: search_web
      description: Search the web for information
      parameters:
        type: object
        properties:
          query:
            type: string
            description: Search query
        required:
          - query
    - name: fetch_url
      description: Fetch content from a URL
      parameters:
        type: object
        properties:
          url:
            type: string
            description: URL to fetch
        required:
          - url
context:
  company: Acme Research Corp
  department: R&D
constraints:
  behavioral:
    - Always cite sources when presenting findings
    - Use formal academic language
  safety:
    - Never fabricate references or citations
messages:
  - role: system
    content: |
      You are a research assistant for {~prompty.var name="context.company" /~}.
      Department: {~prompty.var name="context.department" /~}.

      {~prompty.skills_catalog format="detailed" /~}

      {~prompty.if eval="len(constraints.behavioral) > 0"~}
      Guidelines:
      {~prompty.for item="rule" in="constraints.behavioral"~}
      - {~prompty.var name="rule" /~}
      {~/prompty.for~}
      {~/prompty.if~}
  - role: user
    content: '{~prompty.var name="input.query" /~}'
---
Research agent ready.
`

	// -------------------------------------------------------
	// 2. Parse the agent definition
	// -------------------------------------------------------

	fmt.Println("1. Parsing agent definition...")
	agent, err := prompty.Parse([]byte(agentYAML))
	if err != nil {
		log.Fatal("Failed to parse agent:", err)
	}

	fmt.Printf("   Name: %s\n", agent.Name)
	fmt.Printf("   Type: %s\n", agent.EffectiveType())
	fmt.Printf("   Is Agent: %v\n", agent.IsAgent())
	fmt.Printf("   Skills: %d\n", len(agent.Skills))
	fmt.Printf("   Tools: %d functions\n", len(agent.Tools.Functions))
	fmt.Printf("   Messages: %d templates\n", len(agent.Messages))
	fmt.Println()

	// -------------------------------------------------------
	// 3. Set up a DocumentResolver with skill definitions
	// -------------------------------------------------------

	fmt.Println("2. Setting up document resolver...")
	resolver := prompty.NewMapDocumentResolver()

	// Register skills that the agent references
	resolver.AddSkill("web-search", &prompty.Prompt{
		Name:        "web-search",
		Description: "Searches the web for relevant information using multiple search engines",
		Type:        prompty.DocumentTypeSkill,
		Body:        "Use the search_web tool to find information. Always verify sources.",
	})

	resolver.AddSkill("summarizer", &prompty.Prompt{
		Name:        "summarizer",
		Description: "Summarizes long documents into concise, structured summaries",
		Type:        prompty.DocumentTypeSkill,
		Body:        "Summarize the provided content into key points with citations.",
	})

	fmt.Println("   Registered: web-search, summarizer")
	fmt.Println()

	// -------------------------------------------------------
	// 4. Compile the agent
	// -------------------------------------------------------

	fmt.Println("3. Compiling agent...")
	compiled, err := agent.CompileAgent(ctx, map[string]any{
		"query": "What are the latest advances in quantum computing?",
	}, &prompty.CompileOptions{
		Resolver:            resolver,
		SkillsCatalogFormat: prompty.CatalogFormatDetailed,
	})
	if err != nil {
		log.Fatal("Failed to compile agent:", err)
	}

	fmt.Printf("   Compiled %d messages\n", len(compiled.Messages))
	fmt.Println()

	// -------------------------------------------------------
	// 5. Access compiled messages
	// -------------------------------------------------------

	fmt.Println("4. Compiled messages:")
	for i, msg := range compiled.Messages {
		preview := msg.Content
		if len(preview) > 120 {
			preview = preview[:120] + "..."
		}
		fmt.Printf("   [%d] %s: %s\n", i, msg.Role, preview)
	}
	fmt.Println()

	// -------------------------------------------------------
	// 6. Access execution config
	// -------------------------------------------------------

	fmt.Println("5. Execution config:")
	if compiled.Execution != nil {
		fmt.Printf("   Provider: %s\n", compiled.Execution.Provider)
		fmt.Printf("   Model: %s\n", compiled.Execution.Model)
		if temp, ok := compiled.Execution.GetTemperature(); ok {
			fmt.Printf("   Temperature: %.1f\n", temp)
		}
		if maxTokens, ok := compiled.Execution.GetMaxTokens(); ok {
			fmt.Printf("   Max Tokens: %d\n", maxTokens)
		}
	}
	fmt.Println()

	// -------------------------------------------------------
	// 7. Access tools
	// -------------------------------------------------------

	fmt.Println("6. Tools config:")
	if compiled.Tools != nil && compiled.Tools.HasTools() {
		for _, fn := range compiled.Tools.Functions {
			fmt.Printf("   - %s: %s\n", fn.Name, fn.Description)
		}
	}
	fmt.Println()

	// -------------------------------------------------------
	// 8. Access constraints
	// -------------------------------------------------------

	fmt.Println("7. Constraints:")
	if compiled.Constraints != nil {
		if compiled.Constraints.MaxTurns != nil {
			fmt.Printf("   Max Turns: %d\n", *compiled.Constraints.MaxTurns)
		}
		if len(compiled.Constraints.AllowedDomains) > 0 {
			fmt.Printf("   Allowed Domains: %v\n", compiled.Constraints.AllowedDomains)
		}
	} else {
		fmt.Println("   (no operational constraints defined)")
	}
	fmt.Println()

	// -------------------------------------------------------
	// 9. Activate a specific skill
	// -------------------------------------------------------

	fmt.Println("8. Activating 'web-search' skill...")
	activated, err := agent.ActivateSkill(ctx, "web-search", map[string]any{
		"query": "quantum computing breakthroughs 2025",
	}, &prompty.CompileOptions{
		Resolver:            resolver,
		SkillsCatalogFormat: prompty.CatalogFormatDetailed,
	})
	if err != nil {
		log.Fatal("Failed to activate skill:", err)
	}

	fmt.Printf("   Messages after activation: %d\n", len(activated.Messages))
	// The web-search skill content is injected into the system prompt
	for i, msg := range activated.Messages {
		if msg.Role == "system" {
			preview := msg.Content
			if len(preview) > 120 {
				preview = preview[:120] + "..."
			}
			fmt.Printf("   [%d] %s (with skill): %s\n", i, msg.Role, preview)
		}
	}
	fmt.Println()

	// -------------------------------------------------------
	// 10. Validate for execution
	// -------------------------------------------------------

	fmt.Println("9. Validation:")
	if err := agent.ValidateForExecution(); err != nil {
		fmt.Printf("   Not ready: %v\n", err)
	} else {
		fmt.Println("   Agent is ready for execution")
	}

	fmt.Println("\n=== Example Complete ===")
}
