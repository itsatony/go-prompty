// Example: Custom Resolvers
//
// This example demonstrates how to create and register custom tag resolvers.
// Run: go run ./examples/custom_resolver
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/itsatony/go-prompty/v2"
)

// DateResolver handles {~myapp.date format="..." /~} tags
// This is a self-closing tag resolver that returns formatted dates.
type DateResolver struct{}

func (r *DateResolver) TagName() string {
	return "myapp.date"
}

func (r *DateResolver) Resolve(ctx context.Context, execCtx *prompty.Context, attrs prompty.Attributes) (string, error) {
	format := attrs.GetDefault("format", "2006-01-02")

	// Check for a specific date in context
	if dateStr := execCtx.GetString("date"); dateStr != "" {
		t, err := time.Parse(time.RFC3339, dateStr)
		if err != nil {
			return "", fmt.Errorf("invalid date: %w", err)
		}
		return t.Format(format), nil
	}

	// Default to current time
	return time.Now().Format(format), nil
}

func (r *DateResolver) Validate(attrs prompty.Attributes) error {
	// format is optional, no required attributes
	return nil
}

// UUIDResolver handles {~myapp.uuid /~} tags
// Generates a random hex ID (simplified UUID-like).
type UUIDResolver struct{}

func (r *UUIDResolver) TagName() string {
	return "myapp.uuid"
}

func (r *UUIDResolver) Resolve(ctx context.Context, execCtx *prompty.Context, attrs prompty.Attributes) (string, error) {
	length := 16 // default 16 bytes = 32 hex chars
	if attrs.Has("short") {
		length = 8 // 8 bytes = 16 hex chars
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate ID: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func (r *UUIDResolver) Validate(attrs prompty.Attributes) error {
	return nil
}

// SectionResolver handles {~myapp.section title="..."~}content{~/myapp.section~}
// This demonstrates block tags: resolver returns a prefix, children are appended.
// NOTE: In go-prompty, block tag resolvers return a PREFIX, then children are
// rendered and appended. The resolver does NOT receive children content.
type SectionResolver struct{}

func (r *SectionResolver) TagName() string {
	return "myapp.section"
}

func (r *SectionResolver) Resolve(ctx context.Context, execCtx *prompty.Context, attrs prompty.Attributes) (string, error) {
	title := attrs.GetDefault("title", "Section")
	level := attrs.GetDefault("level", "2")

	// Return markdown header as prefix - children will be appended after this
	prefix := ""
	switch level {
	case "1":
		prefix = fmt.Sprintf("# %s\n\n", title)
	case "2":
		prefix = fmt.Sprintf("## %s\n\n", title)
	case "3":
		prefix = fmt.Sprintf("### %s\n\n", title)
	default:
		prefix = fmt.Sprintf("**%s**\n\n", title)
	}
	return prefix, nil
}

func (r *SectionResolver) Validate(attrs prompty.Attributes) error {
	return nil
}

// DividerResolver handles {~myapp.divider /~} and {~myapp.divider char="-" count="40" /~}
// Returns a horizontal divider line.
type DividerResolver struct{}

func (r *DividerResolver) TagName() string {
	return "myapp.divider"
}

func (r *DividerResolver) Resolve(ctx context.Context, execCtx *prompty.Context, attrs prompty.Attributes) (string, error) {
	char := attrs.GetDefault("char", "-")
	countStr := attrs.GetDefault("count", "40")

	var count int
	if _, err := fmt.Sscanf(countStr, "%d", &count); err != nil || count <= 0 {
		count = 40
	}

	result := ""
	for i := 0; i < count; i++ {
		result += char
	}
	return result, nil
}

func (r *DividerResolver) Validate(attrs prompty.Attributes) error {
	return nil
}

// TimestampResolver handles {~myapp.timestamp /~}
// Returns current Unix timestamp.
type TimestampResolver struct{}

func (r *TimestampResolver) TagName() string {
	return "myapp.timestamp"
}

func (r *TimestampResolver) Resolve(ctx context.Context, execCtx *prompty.Context, attrs prompty.Attributes) (string, error) {
	if attrs.Has("millis") {
		return fmt.Sprintf("%d", time.Now().UnixMilli()), nil
	}
	return fmt.Sprintf("%d", time.Now().Unix()), nil
}

func (r *TimestampResolver) Validate(attrs prompty.Attributes) error {
	return nil
}

func main() {
	engine := prompty.MustNew()

	// Register custom resolvers
	engine.MustRegister(&DateResolver{})
	engine.MustRegister(&UUIDResolver{})
	engine.MustRegister(&SectionResolver{})
	engine.MustRegister(&DividerResolver{})
	engine.MustRegister(&TimestampResolver{})

	// Self-closing tag examples
	fmt.Println("=== Self-Closing Tags ===")
	fmt.Println()

	// Date resolver
	fmt.Println("--- Date Resolver ---")
	dateTemplate := `Today is {~myapp.date format="Monday, January 2, 2006" /~}.
ISO format: {~myapp.date format="2006-01-02T15:04:05Z07:00" /~}
Short format: {~myapp.date format="01/02/06" /~}`

	result, err := engine.Execute(context.Background(), dateTemplate, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
	fmt.Println()

	// UUID resolver
	fmt.Println("--- UUID Resolver ---")
	uuidTemplate := `Full ID: {~myapp.uuid /~}
Short ID: {~myapp.uuid short="true" /~}`

	result, err = engine.Execute(context.Background(), uuidTemplate, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
	fmt.Println()

	// Timestamp resolver
	fmt.Println("--- Timestamp Resolver ---")
	tsTemplate := `Unix timestamp: {~myapp.timestamp /~}
Milliseconds: {~myapp.timestamp millis="true" /~}`

	result, err = engine.Execute(context.Background(), tsTemplate, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
	fmt.Println()

	// Divider resolver
	fmt.Println("--- Divider Resolver ---")
	dividerTemplate := `{~myapp.divider /~}
{~myapp.divider char="=" count="30" /~}
{~myapp.divider char="*" count="20" /~}`

	result, err = engine.Execute(context.Background(), dividerTemplate, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
	fmt.Println()

	// Block tag example (section resolver)
	fmt.Println("=== Block Tags ===")
	fmt.Println()
	fmt.Println("--- Section Resolver ---")
	fmt.Println("(Block tags: resolver returns PREFIX, children are appended)")
	fmt.Println()

	sectionTemplate := `{~myapp.section title="Introduction" level="1"~}This is the introduction section content.{~/myapp.section~}
{~myapp.section title="Details" level="2"~}Here are the details.{~/myapp.section~}
{~myapp.section title="Notes" level="3"~}Some additional notes.{~/myapp.section~}`

	result, err = engine.Execute(context.Background(), sectionTemplate, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
	fmt.Println()

	// Using ResolverFunc for simple cases
	fmt.Println("=== ResolverFunc (Simple Resolver) ===")
	echoResolver := prompty.NewResolverFunc(
		"myapp.echo",
		func(ctx context.Context, execCtx *prompty.Context, attrs prompty.Attributes) (string, error) {
			msg, _ := attrs.Get("msg")
			return msg, nil
		},
		func(attrs prompty.Attributes) error {
			if !attrs.Has("msg") {
				return fmt.Errorf("missing required attribute: msg")
			}
			return nil
		},
	)

	engine.MustRegister(echoResolver)

	echoTemplate := `Message: {~myapp.echo msg="Hello from ResolverFunc!" /~}`
	result, err = engine.Execute(context.Background(), echoTemplate, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
	fmt.Println()

	// Combined example: Report generation
	fmt.Println("=== Combined Example: Report Generation ===")
	reportTemplate := `{~myapp.section title="Daily Report" level="1"~}Generated: {~myapp.date format="2006-01-02 15:04:05" /~}
Report ID: {~myapp.uuid short="true" /~}
{~/myapp.section~}
{~myapp.divider char="=" count="50" /~}

{~myapp.section title="Overview" level="2"~}Hello, {~prompty.var name="user" /~}!

Here is your daily summary.
{~/myapp.section~}
{~myapp.section title="Tasks" level="2"~}{~prompty.for item="task" in="tasks"~}- {~prompty.var name="task" /~}
{~/prompty.for~}{~/myapp.section~}
{~myapp.divider /~}
Timestamp: {~myapp.timestamp /~}`

	reportData := map[string]any{
		"user":  "Alice",
		"tasks": []string{"Review code changes", "Update documentation", "Deploy to staging"},
	}

	result, err = engine.Execute(context.Background(), reportTemplate, reportData)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}
