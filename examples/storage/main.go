// Package main demonstrates the go-prompty storage layer.
//
// This example shows:
// - Using StorageEngine for template management
// - Template versioning
// - Deployment-aware versioning (labels and status)
// - Caching for performance
// - Querying templates
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/itsatony/go-prompty/v2"
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

	// Deployment-aware versioning: Labels
	fmt.Println("\n4. Deployment labels...")

	// Check if storage supports labels (MemoryStorage does)
	if se.SupportsLabels() {
		// Set "staging" label to v2 (the latest)
		err = se.SetLabel(ctx, "greeting", "staging", 2)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("   Set 'staging' label on version 2")

		// Set "production" label to v1 (the stable version)
		err = se.SetLabel(ctx, "greeting", "production", 1)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("   Set 'production' label on version 1")

		// Execute using labels instead of version numbers
		result, err = se.ExecuteLabeled(ctx, "greeting", "production", map[string]any{"user": "Customer", "app": "Prompty"})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("   Production result: %s\n", result)

		result, err = se.ExecuteLabeled(ctx, "greeting", "staging", map[string]any{"user": "Tester", "app": "Prompty"})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("   Staging result:    %s\n", result)

		// Convenience method for production
		result, err = se.ExecuteProduction(ctx, "greeting", map[string]any{"user": "VIP", "app": "Prompty"})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("   ExecuteProduction: %s\n", result)

		// List labels for template
		labels, err := se.ListLabels(ctx, "greeting")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("   Labels: ")
		for _, l := range labels {
			fmt.Printf("%s->v%d ", l.Label, l.Version)
		}
		fmt.Println()

		// Promote staging to production (move label)
		fmt.Println("\n   Promoting v2 to production...")
		err = se.PromoteToProduction(ctx, "greeting", 2)
		if err != nil {
			log.Fatal(err)
		}

		// Verify production now points to v2
		prodTemplate, err := se.GetProduction(ctx, "greeting")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("   Production is now version %d\n", prodTemplate.Version)
	}

	// Deployment-aware versioning: Status
	fmt.Println("\n5. Deployment status...")

	if se.SupportsStatus() {
		// Deprecate the old v1
		err = se.SetStatus(ctx, "greeting", 1, prompty.DeploymentStatusDeprecated)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("   Deprecated version 1")

		// Get version history to see status
		history, err := se.GetVersionHistory(ctx, "greeting")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("   Version history:")
		for _, v := range history.Versions {
			labels := ""
			if len(v.Labels) > 0 {
				labels = fmt.Sprintf(" [%v]", v.Labels)
			}
			fmt.Printf("      v%d: status=%s%s\n", v.Version, v.Status, labels)
		}
		if history.ProductionVersion > 0 {
			fmt.Printf("   Production version: %d\n", history.ProductionVersion)
		}

		// List templates by status (including all versions to find deprecated ones)
		deprecated, err := se.ListByStatus(ctx, prompty.DeploymentStatusDeprecated, &prompty.TemplateQuery{
			IncludeAllVersions: true,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("   Deprecated versions: %d\n", len(deprecated))
	}

	// Query templates
	fmt.Println("\n7. Querying templates...")

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
	fmt.Println("\n8. Template validation...")
	validResult, err := se.Validate(ctx, "greeting")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   'greeting' is valid: %v\n", validResult.IsValid())

	// Demonstrate caching with CachedStorage
	fmt.Println("\n9. Demonstrating caching...")
	cachedStorage := prompty.NewCachedStorage(prompty.NewMemoryStorage(), prompty.CacheConfig{
		TTL:              1 * time.Hour,
		MaxEntries:       100,
		NegativeCacheTTL: prompty.DefaultNegativeCacheTTL,
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
