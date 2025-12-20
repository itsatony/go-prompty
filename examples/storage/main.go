// Package main demonstrates the go-prompty storage layer.
//
// This example shows:
// - Using StorageEngine for template management
// - Template versioning
// - Caching for performance
// - Querying templates
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/itsatony/go-prompty"
)

func main() {
	ctx := context.Background()

	// Create a storage engine with memory storage
	// In production, use FilesystemStorage or a custom database driver
	storage := prompty.NewMemoryStorage()
	se, err := prompty.NewStorageEngine(prompty.StorageEngineConfig{
		Storage: storage,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer se.Close()

	fmt.Println("=== Storage Layer Example ===")
	fmt.Println()

	// Save templates with metadata
	fmt.Println("1. Saving templates...")
	templates := []prompty.StoredTemplate{
		{
			Name:      "greeting",
			Source:    "Hello, {~prompty.var name=\"user\" default=\"Guest\" /~}! Welcome to {~prompty.var name=\"app\" default=\"our service\" /~}.",
			CreatedBy: "admin",
			TenantID:  "acme-corp",
			Tags:      []string{"public", "welcome"},
			Metadata:  map[string]string{"category": "onboarding"},
		},
		{
			Name:      "farewell",
			Source:    "Goodbye, {~prompty.var name=\"user\" /~}! Thanks for using {~prompty.var name=\"app\" /~}.",
			CreatedBy: "admin",
			TenantID:  "acme-corp",
			Tags:      []string{"public"},
			Metadata:  map[string]string{"category": "offboarding"},
		},
		{
			Name:      "notification-email",
			Source:    "Subject: {~prompty.var name=\"subject\" /~}\n\nDear {~prompty.var name=\"recipient\" /~},\n\n{~prompty.var name=\"body\" /~}\n\nBest regards,\n{~prompty.var name=\"sender\" default=\"The Team\" /~}",
			CreatedBy: "team-notifications",
			TenantID:  "acme-corp",
			Tags:      []string{"email", "notification"},
			Metadata:  map[string]string{"category": "communication", "format": "email"},
		},
	}

	for _, tmpl := range templates {
		t := tmpl // capture for pointer
		if err := se.Save(ctx, &t); err != nil {
			log.Printf("Failed to save %s: %v", tmpl.Name, err)
			continue
		}
		fmt.Printf("   Saved: %s (ID: %s, Version: %d)\n", t.Name, t.ID, t.Version)
	}

	// Execute a template
	fmt.Println("\n2. Executing templates...")
	result, err := se.Execute(ctx, "greeting", map[string]any{
		"user": "Alice",
		"app":  "Prompty",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Result: %s\n", result)

	// Update a template (creates new version)
	fmt.Println("\n3. Updating template (versioning)...")
	err = se.Save(ctx, &prompty.StoredTemplate{
		Name:      "greeting",
		Source:    "Hi {~prompty.var name=\"user\" default=\"friend\" /~}! Great to see you at {~prompty.var name=\"app\" /~}!",
		CreatedBy: "admin",
		TenantID:  "acme-corp",
		Tags:      []string{"public", "welcome", "v2"},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Show version history
	versions, err := se.ListVersions(ctx, "greeting")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Versions available: %v\n", versions)

	// Execute latest version
	result, err = se.Execute(ctx, "greeting", map[string]any{"user": "Bob", "app": "Prompty"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Latest (v2): %s\n", result)

	// Execute specific version
	result, err = se.ExecuteVersion(ctx, "greeting", 1, map[string]any{"user": "Bob", "app": "Prompty"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Version 1:   %s\n", result)

	// Query templates
	fmt.Println("\n4. Querying templates...")

	// List all templates
	all, err := se.List(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Total templates: %d\n", len(all))

	// Query by tag
	publicTemplates, err := se.List(ctx, &prompty.TemplateQuery{Tags: []string{"public"}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Templates with 'public' tag: %d\n", len(publicTemplates))

	// Query by prefix
	notifications, err := se.List(ctx, &prompty.TemplateQuery{NamePrefix: "notification"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Templates starting with 'notification': %d\n", len(notifications))

	// Validation
	fmt.Println("\n5. Template validation...")
	validResult, err := se.Validate(ctx, "greeting")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   'greeting' is valid: %v\n", validResult.IsValid())

	// Demonstrate caching with CachedStorage
	fmt.Println("\n6. Demonstrating caching...")
	cachedStorage := prompty.NewCachedStorage(prompty.NewMemoryStorage(), prompty.CacheConfig{
		TTL:              1 * time.Hour,
		MaxEntries:       100,
		NegativeCacheTTL: 30 * time.Second,
	})
	defer cachedStorage.Close()

	cachedSE, err := prompty.NewStorageEngine(prompty.StorageEngineConfig{
		Storage: cachedStorage,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cachedSE.Close()

	// Save and execute multiple times
	_ = cachedSE.Save(ctx, &prompty.StoredTemplate{
		Name:   "cached-template",
		Source: "This is cached: {~prompty.var name=\"value\" /~}",
	})

	// First execution populates cache
	_, _ = cachedSE.Execute(ctx, "cached-template", map[string]any{"value": "first"})

	// Subsequent executions use cache
	for i := 0; i < 5; i++ {
		_, _ = cachedSE.Execute(ctx, "cached-template", map[string]any{"value": fmt.Sprintf("call %d", i)})
	}

	// Check parsed cache stats
	stats := cachedSE.ParsedCacheStats()
	fmt.Printf("   Parsed template cache entries: %d\n", stats.Entries)

	// Check storage cache stats
	storageStats := cachedStorage.Stats()
	fmt.Printf("   Storage cache entries: %d (valid: %d)\n", storageStats.Entries, storageStats.ValidEntries)

	fmt.Println("\n=== Example Complete ===")
}
