// Example: Prompt Import & Export
//
// This example demonstrates v2.1 prompt serialization, import, and export:
// - ExportFull() — serialize a prompt with all fields to YAML frontmatter + body
// - ExportAgentSkill() — serialize with only Agent Skills compatible fields
// - ExportToSkillMD() — export as SKILL.md format
// - Import() — import from .md files
// - ImportFromSkillMD() — parse SKILL.md format
// - ExportSkillDirectory() — create a zip archive with resources
// - Serialize() with custom SerializeOptions
//
// Run: go run ./examples/prompt_import_export
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/itsatony/go-prompty/v2"
)

func main() {
	ctx := context.Background()
	_ = ctx

	fmt.Println("=== Prompt Import & Export Example ===")
	fmt.Println()

	// -------------------------------------------------------
	// 1. Create a prompt programmatically
	// -------------------------------------------------------

	fmt.Println("1. Creating a prompt programmatically...")

	prompt := &prompty.Prompt{
		Name:        "code-reviewer",
		Description: "Reviews code for bugs, style issues, and security vulnerabilities",
		Type:        prompty.DocumentTypeSkill,
		License:     "MIT",
		Execution: &prompty.ExecutionConfig{
			Provider:    prompty.ProviderOpenAI,
			Model:       "gpt-4",
			Temperature: floatPtr(0.2),
			MaxTokens:   intPtr(4096),
		},
		Inputs: map[string]*prompty.InputDef{
			"code": {
				Type:        "string",
				Required:    true,
				Description: "The code to review",
			},
			"language": {
				Type:        "string",
				Required:    false,
				Description: "Programming language",
				Default:     "auto-detect",
			},
		},
		Sample: map[string]any{
			"code":     "func main() { fmt.Println(\"hello\") }",
			"language": "go",
		},
		Body: `{~prompty.message role="system"~}
You are an expert code reviewer. Review the following code for:
- Bugs and logical errors
- Style and readability issues
- Security vulnerabilities
- Performance concerns
{~/prompty.message~}

{~prompty.message role="user"~}
Language: {~prompty.var name="language" default="auto-detect" /~}

Code:
{~prompty.var name="code" /~}
{~/prompty.message~}`,
	}

	fmt.Printf("   Name: %s\n", prompt.Name)
	fmt.Printf("   Type: %s\n", prompt.EffectiveType())
	fmt.Println()

	// -------------------------------------------------------
	// 2. Export with all fields (ExportFull)
	// -------------------------------------------------------

	fmt.Println("2. ExportFull() — all fields included")

	fullBytes, err := prompt.ExportFull()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   Output:")
	printIndented(string(fullBytes), 3)
	fmt.Println()

	// -------------------------------------------------------
	// 3. Export as Agent Skills compatible (ExportAgentSkill)
	// -------------------------------------------------------

	fmt.Println("3. ExportAgentSkill() — stripped to Agent Skills standard fields")

	agentSkillBytes, err := prompt.ExportAgentSkill()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   Output (no execution/skope/agent fields):")
	printIndented(string(agentSkillBytes), 3)
	fmt.Println()

	// -------------------------------------------------------
	// 4. Export as SKILL.md format
	// -------------------------------------------------------

	fmt.Println("4. ExportToSkillMD() — SKILL.md format")

	skillMD, err := prompt.ExportToSkillMD(prompt.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   Output:")
	printIndented(skillMD, 3)
	fmt.Println()

	// -------------------------------------------------------
	// 5. Import from SKILL.md format (round-trip)
	// -------------------------------------------------------

	fmt.Println("5. ImportFromSkillMD() — round-trip from SKILL.md")

	parsed, err := prompty.ImportFromSkillMD(skillMD)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Imported name: %s\n", parsed.Prompt.Name)
	fmt.Printf("   Imported description: %s\n", parsed.Prompt.Description)
	fmt.Printf("   Body length: %d chars\n", len(parsed.Body))
	fmt.Println()

	// -------------------------------------------------------
	// 6. Import from .md file data
	// -------------------------------------------------------

	fmt.Println("6. Import() — from markdown document")

	mdSource := `---
name: summarizer
description: Summarizes text into key points
type: skill
execution:
  provider: anthropic
  model: claude-sonnet-4-5
  temperature: 0.3
---
Summarize the following text into 3-5 bullet points:

{~prompty.var name="text" /~}
`

	result, err := prompty.Import([]byte(mdSource), "summarizer.md")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Imported: %s (%s)\n", result.Prompt.Name, result.Prompt.EffectiveType())
	fmt.Printf("   Provider: %s\n", result.Prompt.Execution.Provider)
	fmt.Printf("   Model: %s\n", result.Prompt.Execution.Model)
	fmt.Printf("   Body: %s...\n", result.Prompt.Body[:50])
	fmt.Println()

	// -------------------------------------------------------
	// 7. Export as zip archive with resources
	// -------------------------------------------------------

	fmt.Println("7. ExportSkillDirectory() — zip archive with resources")

	resources := map[string][]byte{
		"examples/good_code.go":  []byte("package main\n\nfunc main() {}\n"),
		"examples/bad_code.go":   []byte("package main\n\nfunc main() { panic(\"oops\") }\n"),
		"config/review_rules.md": []byte("# Review Rules\n- Check for nil pointers\n- Validate error handling\n"),
	}

	zipBytes, err := prompty.ExportSkillDirectory(prompt, resources)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Created zip archive: %d bytes\n", len(zipBytes))
	fmt.Printf("   Contains: SKILL.md + %d resource files\n", len(resources))
	fmt.Println()

	// -------------------------------------------------------
	// 8. Import from zip archive (round-trip)
	// -------------------------------------------------------

	fmt.Println("8. ImportDirectory() — round-trip from zip")

	zipResult, err := prompty.ImportDirectory(zipBytes)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Imported prompt: %s\n", zipResult.Prompt.Name)
	fmt.Printf("   Resources found: %d\n", len(zipResult.Resources))
	for name, content := range zipResult.Resources {
		fmt.Printf("   - %s (%d bytes)\n", name, len(content))
	}
	fmt.Println()

	// -------------------------------------------------------
	// 9. Custom serialization options
	// -------------------------------------------------------

	fmt.Println("9. Serialize() — custom options")

	// Serialize with only execution config (useful for API config extraction)
	execOnlyBytes, err := prompt.Serialize(&prompty.SerializeOptions{
		IncludeExecution:   true,
		IncludeSkope:       false,
		IncludeAgentFields: false,
		IncludeContext:     false,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("   Execution-only export:")
	printIndented(string(execOnlyBytes), 3)
	fmt.Println()

	// -------------------------------------------------------
	// 10. Check Agent Skills compatibility
	// -------------------------------------------------------

	fmt.Println("10. Agent Skills compatibility check")

	fmt.Printf("   Full prompt is AS-compatible: %v\n", prompt.IsAgentSkillsCompatible())

	stripped := prompt.StripExtensions()
	fmt.Printf("   Stripped prompt is AS-compatible: %v\n", stripped.IsAgentSkillsCompatible())
	fmt.Printf("   Stripped name: %s\n", stripped.Name)

	fmt.Println("\n=== Example Complete ===")
}

func floatPtr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}

func printIndented(s string, spaces int) {
	prefix := strings.Repeat(" ", spaces)
	for _, line := range splitLines(s) {
		fmt.Printf("%s%s\n", prefix, line)
	}
}

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
