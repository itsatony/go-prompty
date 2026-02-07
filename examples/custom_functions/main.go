// Example: Custom Functions
//
// This example demonstrates how to register and use custom functions in expressions.
// Run: go run ./examples/custom_functions
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/itsatony/go-prompty/v2"
)

func main() {
	engine := prompty.MustNew()

	// Register a simple greeting function
	engine.MustRegisterFunc(&prompty.Func{
		Name:    "greet",
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			name, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("expected string argument")
			}
			return "Hello, " + name + "!", nil
		},
	})

	// Register a function that extracts initials
	engine.MustRegisterFunc(&prompty.Func{
		Name:    "initials",
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			name, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("expected string argument")
			}
			parts := strings.Fields(name)
			var result string
			for _, p := range parts {
				if len(p) > 0 {
					result += strings.ToUpper(string(p[0]))
				}
			}
			return result, nil
		},
	})

	// Register a math function (variadic)
	engine.MustRegisterFunc(&prompty.Func{
		Name:    "sum",
		MinArgs: 1,
		MaxArgs: -1, // Variadic
		Fn: func(args []any) (any, error) {
			total := 0.0
			for _, arg := range args {
				switch v := arg.(type) {
				case int:
					total += float64(v)
				case float64:
					total += v
				default:
					return nil, fmt.Errorf("expected numeric argument, got %T", arg)
				}
			}
			return total, nil
		},
	})

	// Register a function to calculate percentage
	engine.MustRegisterFunc(&prompty.Func{
		Name:    "percent",
		MinArgs: 2,
		MaxArgs: 2,
		Fn: func(args []any) (any, error) {
			toFloat := func(v any) (float64, error) {
				switch n := v.(type) {
				case int:
					return float64(n), nil
				case float64:
					return n, nil
				default:
					return 0, fmt.Errorf("expected numeric argument")
				}
			}

			value, err := toFloat(args[0])
			if err != nil {
				return nil, err
			}
			total, err := toFloat(args[1])
			if err != nil {
				return nil, err
			}

			if total == 0 {
				return 0.0, nil
			}

			return math.Round(value/total*100*10) / 10, nil
		},
	})

	// Register a word count function
	engine.MustRegisterFunc(&prompty.Func{
		Name:    "wordCount",
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			text, ok := args[0].(string)
			if !ok {
				return nil, fmt.Errorf("expected string argument")
			}
			words := strings.Fields(text)
			return len(words), nil
		},
	})

	// Example 1: Using greet function
	fmt.Println("=== Greet Function ===")
	template1 := `{~prompty.if eval="greet(name) == 'Hello, Alice!'"~}Welcome back, Alice!{~prompty.else~}Hello, stranger!{~/prompty.if~}`

	result, err := engine.Execute(context.Background(), template1, map[string]any{"name": "Alice"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
	fmt.Println()

	// Example 2: Using initials function
	fmt.Println("=== Initials Function ===")
	template2 := `User: {~prompty.var name="fullName" /~}
Initials: {~prompty.if eval="initials(fullName) != ''"~}{~prompty.var name="fullName" /~} ({~prompty.if eval="len(initials(fullName)) > 0"~}with initials{~/prompty.if~}){~/prompty.if~}`

	result, err = engine.Execute(context.Background(), template2, map[string]any{"fullName": "John William Smith"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
	fmt.Println()

	// Example 3: Using sum function (variadic)
	fmt.Println("=== Sum Function (Variadic) ===")
	template3 := `{~prompty.if eval="sum(1, 2, 3, 4, 5) > 10"~}Sum is greater than 10{~prompty.else~}Sum is 10 or less{~/prompty.if~}
Total items: {~prompty.var name="total" /~} (sum of 1+2+3+4+5 = 15)`

	result, err = engine.Execute(context.Background(), template3, map[string]any{"total": 15})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
	fmt.Println()

	// Example 4: Using percent function
	fmt.Println("=== Percent Function ===")
	template4 := `Progress Report:
- Completed: {~prompty.var name="completed" /~} tasks
- Total: {~prompty.var name="total" /~} tasks
{~prompty.if eval="percent(completed, total) >= 80"~}- Status: Excellent progress!{~prompty.elseif eval="percent(completed, total) >= 50"~}- Status: Good progress{~prompty.else~}- Status: Needs attention{~/prompty.if~}`

	testCases := []map[string]any{
		{"completed": 9, "total": 10},
		{"completed": 6, "total": 10},
		{"completed": 3, "total": 10},
	}

	for _, tc := range testCases {
		result, err = engine.Execute(context.Background(), template4, tc)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(result)
		fmt.Println()
	}

	// Example 5: Using wordCount function
	fmt.Println("=== Word Count Function ===")
	template5 := `Text: "{~prompty.var name="text" /~}"
{~prompty.if eval="wordCount(text) > 10"~}Long text ({~prompty.var name="wordCountDisplay" /~} words){~prompty.elseif eval="wordCount(text) > 5"~}Medium text{~prompty.else~}Short text{~/prompty.if~}`

	texts := []struct {
		text string
		wc   int
	}{
		{"Hello world", 2},
		{"The quick brown fox jumps over the lazy dog", 9},
		{"This is a longer piece of text that has more than ten words in total", 14},
	}

	for _, t := range texts {
		result, err = engine.Execute(context.Background(), template5, map[string]any{
			"text":             t.text,
			"wordCountDisplay": t.wc,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(result)
		fmt.Println()
	}

	// List all registered functions
	fmt.Println("=== Registered Functions ===")
	fmt.Printf("Total functions: %d\n", engine.FuncCount())
	fmt.Println("Custom functions: greet, initials, sum, percent, wordCount")
}
