// Package main demonstrates go-prompty inference configuration.
//
// This example shows how to:
// - Parse templates with embedded config blocks
// - Access model configuration and parameters
// - Validate inputs against schemas
// - Use environment variables in config
// - Work with sample data
package main

import (
	"context"
	"fmt"
	"os"

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

	// Template with full inference configuration
	// Note: In JSON strings within config blocks, use single quotes for prompty tag attributes
	// to avoid JSON escaping issues, or properly escape double quotes.
	source := `{~prompty.config~}
{
  "name": "customer-support-agent",
  "description": "Handles customer inquiries with empathetic responses",
  "version": "1.2.0",
  "authors": ["support-team@example.com"],
  "tags": ["production", "customer-service"],
  "model": {
    "api": "chat",
    "provider": "openai",
    "name": "{~prompty.env name='MODEL_NAME' default='gpt-4' /~}",
    "parameters": {
      "temperature": 0.7,
      "max_tokens": 2048,
      "top_p": 0.9
    }
  },
  "inputs": {
    "customer_name": {"type": "string", "required": true, "description": "Customer's name"},
    "query": {"type": "string", "required": true, "description": "Customer's question"},
    "priority": {"type": "string", "required": false, "description": "Request priority"}
  },
  "outputs": {
    "response": {"type": "string", "description": "Support response"}
  },
  "sample": {
    "customer_name": "Alice",
    "query": "How do I reset my password?",
    "priority": "normal"
  }
}
{~/prompty.config~}
Hello {~prompty.var name="customer_name" /~},

Thank you for reaching out. I understand you need help with:
{~prompty.var name="query" /~}

{~prompty.if eval="priority == 'high'"~}
I'm treating this as a priority request and will ensure quick resolution.
{~prompty.else~}
I'll do my best to help you today.
{~/prompty.if~}

Best regards,
Customer Support Team`

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

		// Execute template
		result, err := tmpl.Execute(context.Background(), sample)
		if err != nil {
			fmt.Println("Execution error:", err)
			return
		}
		fmt.Println("\nRendered output:")
		fmt.Println(result)
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

	result, err := tmpl.Execute(context.Background(), highPriorityData)
	if err != nil {
		fmt.Println("Execution error:", err)
		return
	}
	fmt.Println("Rendered output:")
	fmt.Println(result)

	// Demonstrate JSON serialization
	fmt.Println("\n=== JSON Serialization ===")
	jsonStr, err := config.JSONPretty()
	if err != nil {
		fmt.Println("JSON error:", err)
		return
	}
	fmt.Println("Config as JSON:")
	fmt.Println(jsonStr)
}
