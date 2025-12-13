// Example: Loops (for)
//
// This example demonstrates for loop iteration with go-prompty.
// Run: go run ./examples/loops
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/itsatony/go-prompty"
)

func main() {
	engine := prompty.MustNew()

	// Simple slice iteration
	simpleTemplate := `Tasks:
{~prompty.for item="task" in="tasks"~}- {~prompty.var name="task" /~}
{~/prompty.for~}`

	data := map[string]any{
		"tasks": []string{"Write code", "Review PR", "Deploy to production"},
	}

	result, err := engine.Execute(context.Background(), simpleTemplate, data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("=== Simple Slice Iteration ===")
	fmt.Println(result)

	// With index
	indexTemplate := `Numbered Tasks:
{~prompty.for item="task" index="i" in="tasks"~}{~prompty.var name="i" /~}. {~prompty.var name="task" /~}
{~/prompty.for~}`

	result, err = engine.Execute(context.Background(), indexTemplate, data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("=== With Index ===")
	fmt.Println(result)

	// Iterating over objects
	objectTemplate := `Users:
{~prompty.for item="user" in="users"~}- {~prompty.var name="user.name" /~} ({~prompty.var name="user.email" /~})
{~/prompty.for~}`

	objectData := map[string]any{
		"users": []map[string]any{
			{"name": "Alice", "email": "alice@example.com"},
			{"name": "Bob", "email": "bob@example.com"},
			{"name": "Charlie", "email": "charlie@example.com"},
		},
	}

	result, err = engine.Execute(context.Background(), objectTemplate, objectData)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("=== Iterating Over Objects ===")
	fmt.Println(result)

	// Map iteration
	mapTemplate := `Configuration:
{~prompty.for item="entry" in="config"~}- {~prompty.var name="entry.key" /~}: {~prompty.var name="entry.value" /~}
{~/prompty.for~}`

	mapData := map[string]any{
		"config": map[string]any{
			"host":    "localhost",
			"port":    8080,
			"timeout": "30s",
		},
	}

	result, err = engine.Execute(context.Background(), mapTemplate, mapData)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("=== Map Iteration ===")
	fmt.Println(result)

	// With limit
	limitTemplate := `Top 3 Items:
{~prompty.for item="item" in="items" limit="3"~}- {~prompty.var name="item" /~}
{~/prompty.for~}`

	limitData := map[string]any{
		"items": []string{"First", "Second", "Third", "Fourth", "Fifth"},
	}

	result, err = engine.Execute(context.Background(), limitTemplate, limitData)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("=== With Limit ===")
	fmt.Println(result)

	// Nested loops
	nestedTemplate := `Categories:
{~prompty.for item="category" in="categories"~}
## {~prompty.var name="category.name" /~}
{~prompty.for item="product" in="category.products"~}  - {~prompty.var name="product" /~}
{~/prompty.for~}{~/prompty.for~}`

	nestedData := map[string]any{
		"categories": []map[string]any{
			{
				"name":     "Electronics",
				"products": []string{"Phone", "Laptop", "Tablet"},
			},
			{
				"name":     "Books",
				"products": []string{"Fiction", "Non-fiction", "Technical"},
			},
		},
	}

	result, err = engine.Execute(context.Background(), nestedTemplate, nestedData)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("=== Nested Loops ===")
	fmt.Println(result)
}
