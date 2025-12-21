// Package main demonstrates persistent template storage using FilesystemStorage.
// This example shows how templates persist across application restarts.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/itsatony/go-prompty"
)

func main() {
	ctx := context.Background()

	// Use a persistent directory (in real app, use a proper data directory)
	storageDir := filepath.Join(os.TempDir(), "prompty-example-storage")

	fmt.Println("=== Persistent Template Storage Example ===")
	fmt.Printf("Storage directory: %s\n\n", storageDir)

	// PART 1: Create and save templates
	fmt.Println("--- Part 1: Creating templates ---")
	runPart1(ctx, storageDir)

	// PART 2: Simulate restart - open storage again
	fmt.Println("\n--- Part 2: Simulating restart ---")
	runPart2(ctx, storageDir)

	// Cleanup
	os.RemoveAll(storageDir)
	fmt.Println("\nExample complete. Storage cleaned up.")
}

func runPart1(ctx context.Context, dir string) {
	// Create filesystem storage
	storage, err := prompty.NewFilesystemStorage(dir)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}

	// Create storage engine
	engine, err := prompty.NewStorageEngine(prompty.StorageEngineConfig{
		Storage: storage,
	})
	if err != nil {
		log.Fatalf("Failed to create engine: %v", err)
	}

	// Save a template
	err = engine.Save(ctx, &prompty.StoredTemplate{
		Name:   "greeting",
		Source: `Hello {~prompty.var name="user" /~}! Welcome to {~prompty.var name="app" default="MyApp" /~}.`,
		Tags:   []string{"production", "user-facing"},
		Metadata: map[string]string{
			"author":  "example",
			"version": "1.0",
		},
	})
	if err != nil {
		log.Fatalf("Failed to save template: %v", err)
	}
	fmt.Println("Saved 'greeting' template (version 1)")

	// Execute it
	result, err := engine.Execute(ctx, "greeting", map[string]any{
		"user": "Alice",
		"app":  "DemoApp",
	})
	if err != nil {
		log.Fatalf("Failed to execute: %v", err)
	}
	fmt.Printf("Executed: %s\n", result)

	// Update the template (creates version 2)
	err = engine.Save(ctx, &prompty.StoredTemplate{
		Name:   "greeting",
		Source: `Hi {~prompty.var name="user" /~}! Welcome to {~prompty.var name="app" default="MyApp" /~}.`,
		Tags:   []string{"production", "user-facing", "v2"},
	})
	if err != nil {
		log.Fatalf("Failed to update template: %v", err)
	}
	fmt.Println("Updated 'greeting' template (version 2)")

	// Execute new version
	result, err = engine.Execute(ctx, "greeting", map[string]any{
		"user": "Bob",
	})
	if err != nil {
		log.Fatalf("Failed to execute: %v", err)
	}
	fmt.Printf("Executed (v2): %s\n", result)

	// Close engine (simulating app shutdown)
	engine.Close()
	fmt.Println("Engine closed (simulating shutdown)")
}

func runPart2(ctx context.Context, dir string) {
	// Reopen storage (simulating app restart)
	storage, err := prompty.NewFilesystemStorage(dir)
	if err != nil {
		log.Fatalf("Failed to reopen storage: %v", err)
	}

	engine, err := prompty.NewStorageEngine(prompty.StorageEngineConfig{
		Storage: storage,
	})
	if err != nil {
		log.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	// Verify template persisted
	tmpl, err := engine.Get(ctx, "greeting")
	if err != nil {
		log.Fatalf("Template not found after restart: %v", err)
	}
	fmt.Printf("Template persisted! Version: %d, Tags: %v\n", tmpl.Version, tmpl.Tags)

	// Execute the persisted template
	result, err := engine.Execute(ctx, "greeting", map[string]any{
		"user": "Charlie",
	})
	if err != nil {
		log.Fatalf("Failed to execute: %v", err)
	}
	fmt.Printf("Executed persisted template: %s\n", result)

	// Show version history
	history, err := engine.GetVersionHistory(ctx, "greeting")
	if err != nil {
		log.Fatalf("Failed to get version history: %v", err)
	}
	fmt.Printf("\nVersion History:\n")
	fmt.Printf("  Total versions: %d\n", history.TotalVersions)
	fmt.Printf("  Current version: %d\n", history.CurrentVersion)

	// List all versions
	versions, err := storage.ListVersions(ctx, "greeting")
	if err != nil {
		log.Fatalf("Failed to list versions: %v", err)
	}
	fmt.Printf("  Available versions: %v\n", versions)

	// Show directory structure
	fmt.Println("\n--- Storage Directory Structure ---")
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		if rel == "." {
			return nil
		}
		// Calculate indent based on depth
		depth := len(filepath.SplitList(rel)) - 1
		indent := ""
		for i := 0; i < depth; i++ {
			indent += "  "
		}
		if info.IsDir() {
			fmt.Printf("%s[dir] %s/\n", indent, info.Name())
		} else {
			fmt.Printf("%s[file] %s (%d bytes)\n", indent, info.Name(), info.Size())
		}
		return nil
	})
	if err != nil {
		log.Printf("Error walking directory: %v", err)
	}
}
