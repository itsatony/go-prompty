// Package main demonstrates go-prompty inference configuration.
//
// This example shows how to:
// - Parse templates with YAML frontmatter configuration
// - Access model configuration and parameters
// - Validate inputs against schemas
// - Use environment variables in config
// - Work with message tags for conversations
// - Extract messages for LLM API calls
// - Use response_format for structured outputs (v1.4.0+)
// - Use tools/function calling (v1.4.0+)
// - Configure retry and cache behavior (v1.4.0+)
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/itsatony/go-prompty"
)

func main() {
	// Set environment variable for dynamic model selection
	os.Setenv("MODEL_NAME", "gpt-4-turbo")
	defer os.Unsetenv("MODEL_NAME")

	engine, err := prompty.New()
	if err != nil {
		fmt.Println("Failed to create engine:", err)
		return
	}

	// Template with full YAML frontmatter configuration
	// Note: Use single quotes for YAML values containing prompty tags
	// to avoid escaping issues with double-quoted strings
	source := `---
name: customer-support-agent
description: Handles customer inquiries with empathetic responses
version: 1.4.0
authors:
  - support-team@example.com
tags:
  - production
  - customer-service
model:
  api: chat
  provider: openai
  name: '{~prompty.env name="MODEL_NAME" default="gpt-4" /~}'
  parameters:
    temperature: 0.7
    max_tokens: 2048
    top_p: 0.9
inputs:
  customer_name:
    type: string
    required: true
    description: Customer's name
  query:
    type: string
    required: true
    description: Customer's question
  priority:
    type: string
    required: false
    description: Request priority
outputs:
  response:
    type: string
    description: Support response
sample:
  customer_name: Alice
  query: How do I reset my password?
  priority: normal
---
{~prompty.message role="system"~}
You are a helpful customer support agent. Be empathetic and professional.
{~/prompty.message~}

{~prompty.message role="user"~}
Customer: {~prompty.var name="customer_name" /~}
Query: {~prompty.var name="query" /~}

{~prompty.if eval="priority == 'high'"~}
Note: This is a high-priority request requiring immediate attention.
{~/prompty.if~}
{~/prompty.message~}`

	// Parse template
	tmpl, err := engine.Parse(source)
	if err != nil {
		fmt.Println("Failed to parse template:", err)
		return
	}

	// Check for inference config
	if !tmpl.HasInferenceConfig() {
		fmt.Println("Template has no inference config")
		return
	}

	config := tmpl.InferenceConfig()

	// Print template metadata
	fmt.Println("=== Template Metadata ===")
	fmt.Println("Name:", config.Name)
	fmt.Println("Description:", config.Description)
	fmt.Println("Version:", config.Version)
	fmt.Println("Authors:", config.Authors)
	fmt.Println("Tags:", config.Tags)

	// Print model configuration
	fmt.Println("\n=== Model Configuration ===")
	if config.HasModel() {
		fmt.Println("API Type:", config.GetAPIType())
		fmt.Println("Provider:", config.GetProvider())
		fmt.Println("Model Name:", config.GetModelName()) // Will be "gpt-4-turbo" from env

		if temp, ok := config.GetTemperature(); ok {
			fmt.Printf("Temperature: %.1f\n", temp)
		}
		if maxTokens, ok := config.GetMaxTokens(); ok {
			fmt.Println("Max Tokens:", maxTokens)
		}
		if topP, ok := config.GetTopP(); ok {
			fmt.Printf("Top P: %.1f\n", topP)
		}

		// Get all parameters as a map (useful for LLM clients)
		if config.Model.Parameters != nil {
			params := config.Model.Parameters.ToMap()
			fmt.Println("\nAll Parameters:", params)
		}
	}

	// Print input schema
	fmt.Println("\n=== Input Schema ===")
	if config.HasInputs() {
		for name, input := range config.Inputs {
			required := ""
			if input.Required {
				required = " (required)"
			}
			fmt.Printf("- %s: %s%s - %s\n", name, input.Type, required, input.Description)
		}
	}

	// Print output schema
	fmt.Println("\n=== Output Schema ===")
	if config.HasOutputs() {
		for name, output := range config.Outputs {
			fmt.Printf("- %s: %s - %s\n", name, output.Type, output.Description)
		}
	}

	// Validate and execute with sample data
	fmt.Println("\n=== Sample Data Execution ===")
	if config.HasSample() {
		sample := config.GetSampleData()
		fmt.Println("Sample data:", sample)

		// Validate sample data against input schema
		if err := config.ValidateInputs(sample); err != nil {
			fmt.Println("Validation error:", err)
			return
		}
		fmt.Println("Sample data validation: PASSED")

		// Execute and extract messages
		messages, err := tmpl.ExecuteAndExtractMessages(context.Background(), sample)
		if err != nil {
			fmt.Println("Execution error:", err)
			return
		}

		fmt.Println("\nExtracted messages for LLM API:")
		for i, msg := range messages {
			fmt.Printf("  [%d] %s: %s\n", i, msg.Role, truncate(msg.Content, 60))
		}
	}

	// Test with high priority data
	fmt.Println("\n=== High Priority Execution ===")
	highPriorityData := map[string]any{
		"customer_name": "Bob",
		"query":         "My account is locked and I need urgent access!",
		"priority":      "high",
	}

	if err := config.ValidateInputs(highPriorityData); err != nil {
		fmt.Println("Validation error:", err)
		return
	}

	messages, err := tmpl.ExecuteAndExtractMessages(context.Background(), highPriorityData)
	if err != nil {
		fmt.Println("Execution error:", err)
		return
	}

	fmt.Println("Extracted messages:")
	for _, msg := range messages {
		fmt.Printf("  [%s]: %s\n", msg.Role, msg.Content)
	}

	// Demonstrate YAML serialization
	fmt.Println("\n=== YAML Serialization ===")
	yamlStr, err := config.YAML()
	if err != nil {
		fmt.Println("YAML error:", err)
		return
	}
	fmt.Println("Config as YAML:")
	fmt.Println(yamlStr)

	// Run advanced features demo
	demonstrateAdvancedFeatures()
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// demonstrateAdvancedFeatures shows v1.4.0 features like response_format, tools, retry, and cache
func demonstrateAdvancedFeatures() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("=== Advanced v1.4.0 Features Demo ===")
	fmt.Println(strings.Repeat("=", 60))

	engine, _ := prompty.New()

	// Template demonstrating structured outputs and function calling
	advancedSource := `---
name: entity-extractor
description: Extract structured entities with function calling
version: 1.4.0
model:
  api: chat
  provider: openai
  name: gpt-4
  parameters:
    temperature: 0
    max_tokens: 1024
  response_format:
    type: json_schema
    json_schema:
      name: entities
      description: Extracted entities from text
      strict: true
      schema:
        type: object
        properties:
          people:
            type: array
            items:
              type: string
          places:
            type: array
            items:
              type: string
          organizations:
            type: array
            items:
              type: string
        required:
          - people
          - places
          - organizations
  tools:
    - type: function
      function:
        name: search_database
        description: Search for additional entity information
        parameters:
          type: object
          properties:
            entity_name:
              type: string
              description: Name of the entity to search
            entity_type:
              type: string
              enum:
                - person
                - place
                - organization
          required:
            - entity_name
            - entity_type
        strict: true
  tool_choice: auto
  streaming:
    enabled: false
  context_window: 8192
retry:
  max_attempts: 3
  backoff: exponential
cache:
  system_prompt: true
  ttl: 3600
inputs:
  text:
    type: string
    required: true
    description: Text to extract entities from
sample:
  text: John Smith met with Microsoft CEO at New York headquarters.
---
{~prompty.message role="system"~}
You are an entity extraction assistant. Extract all people, places, and organizations.
Respond ONLY with valid JSON matching the schema.
{~/prompty.message~}

{~prompty.message role="user"~}
Extract entities from: {~prompty.var name="text" /~}
{~/prompty.message~}`

	tmpl, err := engine.Parse(advancedSource)
	if err != nil {
		fmt.Println("Failed to parse advanced template:", err)
		return
	}

	config := tmpl.InferenceConfig()

	// Demonstrate response_format access
	fmt.Println("\n--- Response Format ---")
	if config.HasResponseFormat() {
		rf := config.GetResponseFormat()
		fmt.Println("Type:", rf.Type)
		if rf.JSONSchema != nil {
			fmt.Println("Schema Name:", rf.JSONSchema.Name)
			fmt.Println("Schema Strict:", rf.JSONSchema.Strict)
			fmt.Println("Schema Description:", rf.JSONSchema.Description)
		}
	}

	// Demonstrate tools access
	fmt.Println("\n--- Tools (Function Calling) ---")
	if config.HasTools() {
		tools := config.GetTools()
		for i, tool := range tools {
			fmt.Printf("Tool %d: %s\n", i+1, tool.Function.Name)
			fmt.Printf("  Description: %s\n", tool.Function.Description)
			fmt.Printf("  Strict: %v\n", tool.Function.Strict)
		}
	}

	// Demonstrate tool_choice access
	fmt.Println("\n--- Tool Choice ---")
	tc := config.GetToolChoice()
	if tc != nil {
		fmt.Printf("Tool Choice: %v\n", tc)
	}

	// Demonstrate streaming access
	fmt.Println("\n--- Streaming Config ---")
	if config.HasStreaming() {
		streaming := config.GetStreaming()
		fmt.Println("Streaming Enabled:", streaming.Enabled)
	}

	// Demonstrate context_window access
	fmt.Println("\n--- Context Window ---")
	if cw, ok := config.GetContextWindow(); ok {
		fmt.Println("Context Window:", cw)
	}

	// Demonstrate retry config
	fmt.Println("\n--- Retry Config ---")
	if config.HasRetry() {
		retry := config.GetRetry()
		fmt.Println("Max Attempts:", retry.MaxAttempts)
		fmt.Println("Backoff Strategy:", retry.Backoff)
	}

	// Demonstrate cache config
	fmt.Println("\n--- Cache Config ---")
	if config.HasCache() {
		cache := config.GetCache()
		fmt.Println("Cache System Prompt:", cache.SystemPrompt)
		fmt.Println("Cache TTL:", cache.TTL, "seconds")
	}

	// Execute with sample data
	fmt.Println("\n--- Execution with Sample Data ---")
	sample := config.GetSampleData()
	messages, err := tmpl.ExecuteAndExtractMessages(context.Background(), sample)
	if err != nil {
		fmt.Println("Execution error:", err)
		return
	}
	for _, msg := range messages {
		fmt.Printf("[%s]: %s\n", msg.Role, truncate(msg.Content, 80))
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
}
