// Example: Conditionals (if/elseif/else)
//
// This example demonstrates conditional rendering with go-prompty.
// Run: go run ./examples/conditionals
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/itsatony/go-prompty"
)

func main() {
	engine := prompty.MustNew()

	// Simple boolean condition
	template := `{~prompty.if eval="isAdmin"~}Welcome, Administrator!{~prompty.else~}Welcome, User!{~/prompty.if~}`

	adminData := map[string]any{"isAdmin": true}
	userData := map[string]any{"isAdmin": false}

	result, err := engine.Execute(context.Background(), template, adminData)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("=== Simple Boolean (admin) ===")
	fmt.Println(result)
	fmt.Println()

	result, err = engine.Execute(context.Background(), template, userData)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("=== Simple Boolean (user) ===")
	fmt.Println(result)
	fmt.Println()

	// Comparison operators
	comparisonTemplate := `{~prompty.if eval="score >= 90"~}Grade: A{~prompty.elseif eval="score >= 80"~}Grade: B{~prompty.elseif eval="score >= 70"~}Grade: C{~prompty.else~}Grade: F{~/prompty.if~}`

	scores := []int{95, 85, 75, 50}
	fmt.Println("=== Comparison Operators (grades) ===")
	for _, score := range scores {
		result, err = engine.Execute(context.Background(), comparisonTemplate, map[string]any{"score": score})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Score %d: %s\n", score, result)
	}
	fmt.Println()

	// Logical operators
	logicalTemplate := `{~prompty.if eval="isLoggedIn && hasPermission"~}Access Granted{~prompty.else~}Access Denied{~/prompty.if~}`

	fmt.Println("=== Logical Operators (AND) ===")
	testCases := []map[string]any{
		{"isLoggedIn": true, "hasPermission": true},
		{"isLoggedIn": true, "hasPermission": false},
		{"isLoggedIn": false, "hasPermission": true},
	}
	for _, tc := range testCases {
		result, err = engine.Execute(context.Background(), logicalTemplate, tc)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("LoggedIn=%v, Permission=%v: %s\n", tc["isLoggedIn"], tc["hasPermission"], result)
	}
	fmt.Println()

	// Using functions in conditions
	funcTemplate := `{~prompty.if eval="len(items) > 0"~}You have {~prompty.var name="items" /~} items.{~prompty.else~}No items found.{~/prompty.if~}`

	fmt.Println("=== Using Functions (len) ===")
	result, err = engine.Execute(context.Background(), funcTemplate, map[string]any{"items": []string{"a", "b", "c"}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)

	result, err = engine.Execute(context.Background(), funcTemplate, map[string]any{"items": []string{}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}
