// Example: Basic Variable Interpolation
//
// This example demonstrates basic usage of go-prompty for variable interpolation.
// Run: go run ./examples/basic
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/itsatony/go-prompty"
)

func main() {
	// Create an engine with default settings
	engine := prompty.MustNew()

	// Simple template with variable interpolation
	template := `Hello, {~prompty.var name="user" /~}! You have {~prompty.var name="count" /~} messages.`

	// Data to interpolate
	data := map[string]any{
		"user":  "Alice",
		"count": 5,
	}

	// Execute the template
	result, err := engine.Execute(context.Background(), template, data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Basic Variable Interpolation ===")
	fmt.Println(result)
	fmt.Println()

	// Nested path access
	nestedTemplate := `Welcome, {~prompty.var name="user.profile.name" /~}! Your email is {~prompty.var name="user.profile.email" /~}.`
	nestedData := map[string]any{
		"user": map[string]any{
			"profile": map[string]any{
				"name":  "Bob",
				"email": "bob@example.com",
			},
		},
	}

	result, err = engine.Execute(context.Background(), nestedTemplate, nestedData)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Nested Path Access ===")
	fmt.Println(result)
	fmt.Println()

	// Default values
	defaultTemplate := `Hello, {~prompty.var name="name" default="Guest" /~}!`
	result, err = engine.Execute(context.Background(), defaultTemplate, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Default Values ===")
	fmt.Println(result)
}
