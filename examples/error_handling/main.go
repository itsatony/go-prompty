// Example: Error Handling Strategies
//
// This example demonstrates all 5 error strategies in go-prompty:
// - ErrorStrategyThrow (default): Stops execution and returns the error
// - ErrorStrategyDefault: Replaces failed content with a default value
// - ErrorStrategyRemove: Removes the tag entirely from output
// - ErrorStrategyKeepRaw: Keeps the original tag text in output
// - ErrorStrategyLog: Logs the error and continues with empty string
//
// Run: go run ./examples/error_handling
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/itsatony/go-prompty/v2"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Error Handling Strategies in go-prompty ===")
	fmt.Println()

	// 1. ErrorStrategyThrow (default) - stops execution on error
	demonstrateThrowStrategy(ctx)

	// 2. ErrorStrategyDefault - uses default attribute value
	demonstrateDefaultStrategy(ctx)

	// 3. ErrorStrategyRemove - removes failed tag from output
	demonstrateRemoveStrategy(ctx)

	// 4. ErrorStrategyKeepRaw - keeps original tag text
	demonstrateKeepRawStrategy(ctx)

	// 5. ErrorStrategyLog - logs error, continues with empty string
	demonstrateLogStrategy(ctx)

	// 6. Per-tag override with onerror attribute
	demonstratePerTagOverride(ctx)

	// 7. Mixed strategies in single template
	demonstrateMixedStrategies(ctx)
}

func demonstrateThrowStrategy(ctx context.Context) {
	fmt.Println("1. ErrorStrategyThrow (default)")
	fmt.Println("   - Stops execution and returns the error")
	fmt.Println()

	engine := prompty.MustNew() // Default strategy is Throw

	template := `Hello, {~prompty.var name="user" /~}! Your balance is {~prompty.var name="balance" /~}.`

	// Data is missing "balance" - will cause an error
	data := map[string]any{
		"user": "Alice",
	}

	result, err := engine.Execute(ctx, template, data)
	if err != nil {
		fmt.Printf("   Error caught: %v\n", err)
		fmt.Println("   (Execution stopped - no partial output)")
	} else {
		fmt.Println("   Result:", result)
	}
	fmt.Println()
}

func demonstrateDefaultStrategy(ctx context.Context) {
	fmt.Println("2. ErrorStrategyDefault")
	fmt.Println("   - Uses the 'default' attribute value when variable is missing")
	fmt.Println()

	engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyDefault))

	// Template with default values
	template := `Hello, {~prompty.var name="user" default="Guest" /~}! Your balance is {~prompty.var name="balance" default="$0.00" /~}.`

	// Empty data - all variables will use defaults
	data := map[string]any{}

	result, err := engine.Execute(ctx, template, data)
	if err != nil {
		fmt.Println("   Error:", err)
	} else {
		fmt.Println("   Result:", result)
	}
	fmt.Println()
}

func demonstrateRemoveStrategy(ctx context.Context) {
	fmt.Println("3. ErrorStrategyRemove")
	fmt.Println("   - Removes the failed tag entirely from output")
	fmt.Println()

	engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyRemove))

	template := `Hello, {~prompty.var name="user" /~}! You have {~prompty.var name="messages" /~} new messages.`

	// Only "user" is provided, "messages" will be removed
	data := map[string]any{
		"user": "Bob",
	}

	result, err := engine.Execute(ctx, template, data)
	if err != nil {
		fmt.Println("   Error:", err)
	} else {
		fmt.Println("   Result:", result)
		fmt.Println("   (Notice: missing variable tag removed, leaving 'You have  new messages')")
	}
	fmt.Println()
}

func demonstrateKeepRawStrategy(ctx context.Context) {
	fmt.Println("4. ErrorStrategyKeepRaw")
	fmt.Println("   - Keeps the original tag text in output")
	fmt.Println()

	engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyKeepRaw))

	template := `Hello, {~prompty.var name="user" /~}! Order: {~prompty.var name="order_id" /~}`

	// Only "user" is provided
	data := map[string]any{
		"user": "Charlie",
	}

	result, err := engine.Execute(ctx, template, data)
	if err != nil {
		fmt.Println("   Error:", err)
	} else {
		fmt.Println("   Result:", result)
		fmt.Println("   (Notice: missing variable tag kept as raw text)")
	}
	fmt.Println()
}

func demonstrateLogStrategy(ctx context.Context) {
	fmt.Println("5. ErrorStrategyLog")
	fmt.Println("   - Logs the error and continues with empty string")
	fmt.Println()

	// Create a logger to see the logged errors
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	engine := prompty.MustNew(
		prompty.WithErrorStrategy(prompty.ErrorStrategyLog),
		prompty.WithLogger(logger),
	)

	template := `User: {~prompty.var name="user" /~}, Role: {~prompty.var name="role" /~}`

	// Only "user" is provided
	data := map[string]any{
		"user": "Dave",
	}

	fmt.Println("   (Watch for logged error message below)")
	result, err := engine.Execute(ctx, template, data)
	if err != nil {
		fmt.Println("   Error:", err)
	} else {
		fmt.Println("   Result:", result)
		fmt.Println("   (Missing variable replaced with empty string)")
	}
	fmt.Println()
}

func demonstratePerTagOverride(ctx context.Context) {
	fmt.Println("6. Per-Tag Override with 'onerror' Attribute")
	fmt.Println("   - Override global strategy for specific tags")
	fmt.Println()

	// Global strategy is throw, but we'll override per-tag
	engine := prompty.MustNew() // Default: Throw

	// Each tag has a different error handling strategy
	template := `User: {~prompty.var name="user" /~}
Email: {~prompty.var name="email" onerror="default" default="not provided" /~}
Phone: {~prompty.var name="phone" onerror="remove" /~}
Notes: {~prompty.var name="notes" onerror="keepraw" /~}`

	// Only "user" is provided
	data := map[string]any{
		"user": "Eve",
	}

	result, err := engine.Execute(ctx, template, data)
	if err != nil {
		fmt.Println("   Error:", err)
	} else {
		fmt.Println("   Result:")
		fmt.Println("   " + result)
		fmt.Println()
		fmt.Println("   Explanation:")
		fmt.Println("   - user: resolved normally")
		fmt.Println("   - email: used default value 'not provided' (onerror=default)")
		fmt.Println("   - phone: removed from output (onerror=remove)")
		fmt.Println("   - notes: kept as raw tag text (onerror=keepraw)")
	}
	fmt.Println()
}

func demonstrateMixedStrategies(ctx context.Context) {
	fmt.Println("7. Mixed Strategies - Real World Example")
	fmt.Println("   - Practical example combining different strategies")
	fmt.Println()

	// Set an environment variable for demonstration
	os.Setenv("APP_ENV", "production")
	defer os.Unsetenv("APP_ENV")

	engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyDefault))

	// Real-world template with various error handling needs
	template := `=== User Profile ===
Name: {~prompty.var name="name" default="Unknown User" /~}
Email: {~prompty.var name="email" default="No email on file" /~}
Environment: {~prompty.env name="APP_ENV" default="development" /~}
API Key: {~prompty.env name="SECRET_API_KEY" default="[REDACTED]" /~}

{~prompty.if eval="premium == true"~}
Premium Features: Enabled
{~prompty.else~}
Premium Features: {~prompty.var name="upgrade_message" default="Upgrade to unlock!" /~}
{~/prompty.if~}`

	// Partial data - some fields missing
	data := map[string]any{
		"name":    "Frank",
		"premium": false,
	}

	result, err := engine.Execute(ctx, template, data)
	if err != nil {
		fmt.Println("   Error:", err)
	} else {
		fmt.Println("   Result:")
		for _, line := range splitLines(result) {
			fmt.Println("   " + line)
		}
	}
	fmt.Println()

	fmt.Println("=== Summary ===")
	fmt.Println("Error strategies allow graceful handling of missing data:")
	fmt.Println("  throw   - Fail fast (good for development, required fields)")
	fmt.Println("  default - Use fallback values (good for optional fields)")
	fmt.Println("  remove  - Silently skip (good for conditional content)")
	fmt.Println("  keepraw - Debug visibility (good for template development)")
	fmt.Println("  log     - Silent with audit (good for production monitoring)")
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
