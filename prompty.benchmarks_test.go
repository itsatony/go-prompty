package prompty

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// =============================================================================
// PARSING BENCHMARKS
// =============================================================================

func BenchmarkParse_Simple(b *testing.B) {
	engine := MustNew()
	source := `Hello {~prompty.var name="user" /~}!`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Parse(source)
	}
}

func BenchmarkParse_Variables(b *testing.B) {
	engine := MustNew()
	source := `Hello {~prompty.var name="user" /~}, welcome to {~prompty.var name="app" /~}!
Your role: {~prompty.var name="role" default="guest" /~}
Email: {~prompty.var name="email" /~}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Parse(source)
	}
}

func BenchmarkParse_Conditionals(b *testing.B) {
	engine := MustNew()
	source := `{~prompty.if eval="isAdmin"~}
Admin Panel
{~prompty.elseif eval="isUser"~}
User Dashboard
{~prompty.else~}
Guest View
{~/prompty.if~}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Parse(source)
	}
}

func BenchmarkParse_Loop(b *testing.B) {
	engine := MustNew()
	source := `Items:
{~prompty.for item="x" in="items"~}
- {~prompty.var name="x.name" /~}: {~prompty.var name="x.value" /~}
{~/prompty.for~}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Parse(source)
	}
}

func BenchmarkParse_Complex(b *testing.B) {
	engine := MustNew()
	source := `# {~prompty.var name="title" /~}

Hello {~prompty.var name="user.name" /~},

{~prompty.if eval="showIntro"~}
Welcome to our system!
{~/prompty.if~}

## Your Items ({~prompty.var name="itemCount" /~} total)

{~prompty.for item="item" index="i" in="items"~}
{~prompty.var name="i" /~}. {~prompty.var name="item.name" /~}
   - Price: {~prompty.var name="item.price" /~}
   - Quantity: {~prompty.var name="item.qty" default="1" /~}
   {~prompty.if eval="item.onSale"~}
   - ON SALE!
   {~/prompty.if~}
{~/prompty.for~}

{~prompty.switch eval="status"~}
{~prompty.case value="active"~}Your account is active.{~/prompty.case~}
{~prompty.case value="pending"~}Your account is pending approval.{~/prompty.case~}
{~prompty.casedefault~}Please contact support.{~/prompty.casedefault~}
{~/prompty.switch~}

Best regards,
{~prompty.var name="signature" default="The Team" /~}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Parse(source)
	}
}

// =============================================================================
// EXECUTION BENCHMARKS
// =============================================================================

func BenchmarkExecute_Simple(b *testing.B) {
	engine := MustNew()
	source := `Hello {~prompty.var name="user" /~}!`
	data := map[string]any{"user": "Alice"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Execute(ctx, source, data)
	}
}

func BenchmarkExecute_PreParsed(b *testing.B) {
	engine := MustNew()
	source := `Hello {~prompty.var name="user" /~}!`
	tmpl, _ := engine.Parse(source)
	data := map[string]any{"user": "Alice"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tmpl.Execute(ctx, data)
	}
}

func BenchmarkExecute_Variables(b *testing.B) {
	engine := MustNew()
	source := `Hello {~prompty.var name="user" /~}, welcome to {~prompty.var name="app" /~}!
Your role: {~prompty.var name="role" default="guest" /~}
Email: {~prompty.var name="email" /~}`
	tmpl, _ := engine.Parse(source)
	data := map[string]any{
		"user":  "Alice",
		"app":   "MyApp",
		"role":  "admin",
		"email": "alice@example.com",
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tmpl.Execute(ctx, data)
	}
}

func BenchmarkExecute_NestedVariables(b *testing.B) {
	engine := MustNew()
	source := `User: {~prompty.var name="user.profile.displayName" /~}
Email: {~prompty.var name="user.contact.email" /~}
Company: {~prompty.var name="user.organization.company.name" /~}`
	tmpl, _ := engine.Parse(source)
	data := map[string]any{
		"user": map[string]any{
			"profile": map[string]any{"displayName": "Alice Smith"},
			"contact": map[string]any{"email": "alice@example.com"},
			"organization": map[string]any{
				"company": map[string]any{"name": "TechCorp"},
			},
		},
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tmpl.Execute(ctx, data)
	}
}

func BenchmarkExecute_Conditionals(b *testing.B) {
	engine := MustNew()
	source := `{~prompty.if eval="isAdmin"~}
Admin content here
{~prompty.elseif eval="isUser"~}
User content here
{~prompty.else~}
Guest content here
{~/prompty.if~}`
	tmpl, _ := engine.Parse(source)
	data := map[string]any{"isAdmin": true}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tmpl.Execute(ctx, data)
	}
}

func BenchmarkExecute_Loop_10Items(b *testing.B) {
	benchmarkLoop(b, 10)
}

func BenchmarkExecute_Loop_100Items(b *testing.B) {
	benchmarkLoop(b, 100)
}

func BenchmarkExecute_Loop_1000Items(b *testing.B) {
	benchmarkLoop(b, 1000)
}

func benchmarkLoop(b *testing.B, itemCount int) {
	engine := MustNew()
	source := `{~prompty.for item="x" in="items"~}{~prompty.var name="x" /~},{~/prompty.for~}`
	tmpl, _ := engine.Parse(source)

	items := make([]string, itemCount)
	for i := 0; i < itemCount; i++ {
		items[i] = fmt.Sprintf("item%d", i)
	}
	data := map[string]any{"items": items}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tmpl.Execute(ctx, data)
	}
}

func BenchmarkExecute_Switch(b *testing.B) {
	engine := MustNew()
	source := `{~prompty.switch eval="status"~}
{~prompty.case value="active"~}Active{~/prompty.case~}
{~prompty.case value="pending"~}Pending{~/prompty.case~}
{~prompty.case value="inactive"~}Inactive{~/prompty.case~}
{~prompty.casedefault~}Unknown{~/prompty.casedefault~}
{~/prompty.switch~}`
	tmpl, _ := engine.Parse(source)
	data := map[string]any{"status": "pending"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tmpl.Execute(ctx, data)
	}
}

func BenchmarkExecute_Expressions(b *testing.B) {
	engine := MustNew()
	source := `{~prompty.if eval="len(items) > 0 && contains(roles, 'admin')"~}
Content for admin with items
{~/prompty.if~}`
	tmpl, _ := engine.Parse(source)
	data := map[string]any{
		"items": []string{"a", "b", "c"},
		"roles": []string{"user", "admin"},
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tmpl.Execute(ctx, data)
	}
}

func BenchmarkExecute_Functions(b *testing.B) {
	engine := MustNew()
	source := `Name: {~prompty.var name="upper(trim(user.name))" /~}
Length: {~prompty.var name="len(items)" /~}
First: {~prompty.var name="first(items)" /~}`
	tmpl, _ := engine.Parse(source)
	data := map[string]any{
		"user":  map[string]any{"name": "  alice  "},
		"items": []string{"first", "second", "third"},
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tmpl.Execute(ctx, data)
	}
}

func BenchmarkExecute_Complex(b *testing.B) {
	engine := MustNew()
	source := `# Welcome {~prompty.var name="user.name" /~}

{~prompty.if eval="showIntro"~}
Thank you for being a member since {~prompty.var name="user.memberSince" /~}.
{~/prompty.if~}

## Your Items ({~prompty.var name="len(items)" /~})

{~prompty.for item="item" index="i" in="items"~}
{~prompty.var name="i" /~}. {~prompty.var name="item.name" /~} - ${~prompty.var name="item.price" /~}
{~/prompty.for~}

{~prompty.switch eval="user.tier"~}
{~prompty.case value="gold"~}You get 20% discount!{~/prompty.case~}
{~prompty.case value="silver"~}You get 10% discount!{~/prompty.case~}
{~prompty.casedefault~}Upgrade for discounts!{~/prompty.casedefault~}
{~/prompty.switch~}

Best regards`
	tmpl, _ := engine.Parse(source)

	items := make([]map[string]any, 10)
	for i := 0; i < 10; i++ {
		items[i] = map[string]any{"name": fmt.Sprintf("Product %d", i), "price": 9.99}
	}
	data := map[string]any{
		"user": map[string]any{
			"name":        "Alice",
			"memberSince": "2023",
			"tier":        "gold",
		},
		"showIntro": true,
		"items":     items,
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tmpl.Execute(ctx, data)
	}
}

// =============================================================================
// CACHED ENGINE BENCHMARKS
// =============================================================================

func BenchmarkCachedEngine_Miss(b *testing.B) {
	engine := MustNew()
	cached := NewCachedEngine(engine, DefaultResultCacheConfig())
	ctx := context.Background()

	// Different data each time = cache miss
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data := map[string]any{"user": fmt.Sprintf("User%d", i)}
		_, _ = cached.Execute(ctx, `Hello {~prompty.var name="user" /~}!`, data)
	}
}

func BenchmarkCachedEngine_Hit(b *testing.B) {
	engine := MustNew()
	cached := NewCachedEngine(engine, DefaultResultCacheConfig())
	source := `Hello {~prompty.var name="user" /~}!`
	data := map[string]any{"user": "Alice"}
	ctx := context.Background()

	// Warm cache
	_, _ = cached.Execute(ctx, source, data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cached.Execute(ctx, source, data)
	}
}

func BenchmarkCachedEngine_vs_Direct(b *testing.B) {
	engine := MustNew()
	source := `Hello {~prompty.var name="user" /~}, welcome to {~prompty.var name="app" /~}!`
	tmpl, _ := engine.Parse(source)
	data := map[string]any{"user": "Alice", "app": "MyApp"}
	ctx := context.Background()

	b.Run("Direct", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = tmpl.Execute(ctx, data)
		}
	})

	cached := NewCachedEngine(engine, DefaultResultCacheConfig())
	_, _ = cached.Execute(ctx, source, data) // Warm

	b.Run("Cached", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = cached.Execute(ctx, source, data)
		}
	})
}

// =============================================================================
// STORAGE BENCHMARKS
// =============================================================================

func BenchmarkMemoryStorage_Save(b *testing.B) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storage.Save(ctx, &StoredTemplate{
			Name:   fmt.Sprintf("template%d", i%100),
			Source: "Template content here",
		})
	}
}

func BenchmarkMemoryStorage_Get(b *testing.B) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 100; i++ {
		_ = storage.Save(ctx, &StoredTemplate{
			Name:   fmt.Sprintf("template%d", i),
			Source: "Template content here",
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.Get(ctx, fmt.Sprintf("template%d", i%100))
	}
}

func BenchmarkStorageEngine_Execute(b *testing.B) {
	storage := NewMemoryStorage()
	engine, _ := NewStorageEngine(StorageEngineConfig{Storage: storage})
	defer engine.Close()
	ctx := context.Background()

	_ = engine.Save(ctx, &StoredTemplate{
		Name:   "greeting",
		Source: `Hello {~prompty.var name="user" /~}!`,
	})
	data := map[string]any{"user": "Alice"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Execute(ctx, "greeting", data)
	}
}

func BenchmarkCachedStorageEngine_Execute(b *testing.B) {
	storage := NewMemoryStorage()
	engine, _ := NewStorageEngine(StorageEngineConfig{Storage: storage})
	defer engine.Close()
	cached := NewCachedStorageEngine(engine, DefaultResultCacheConfig())
	ctx := context.Background()

	_ = cached.Save(ctx, &StoredTemplate{
		Name:   "greeting",
		Source: `Hello {~prompty.var name="user" /~}!`,
	})
	data := map[string]any{"user": "Alice"}

	// Warm cache
	_, _ = cached.Execute(ctx, "greeting", data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cached.Execute(ctx, "greeting", data)
	}
}

// =============================================================================
// CONCURRENT ACCESS BENCHMARKS
// =============================================================================

func BenchmarkExecute_Concurrent(b *testing.B) {
	engine := MustNew()
	source := `Hello {~prompty.var name="user" /~}! Count: {~prompty.var name="count" /~}`
	tmpl, _ := engine.Parse(source)
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			data := map[string]any{"user": "Alice", "count": i}
			_, _ = tmpl.Execute(ctx, data)
			i++
		}
	})
}

func BenchmarkCachedEngine_Concurrent(b *testing.B) {
	engine := MustNew()
	cached := NewCachedEngine(engine, DefaultResultCacheConfig())
	source := `Hello {~prompty.var name="user" /~}!`
	data := map[string]any{"user": "Alice"}
	ctx := context.Background()

	// Warm
	_, _ = cached.Execute(ctx, source, data)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = cached.Execute(ctx, source, data)
		}
	})
}

func BenchmarkMemoryStorage_Concurrent(b *testing.B) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 100; i++ {
		_ = storage.Save(ctx, &StoredTemplate{
			Name:   fmt.Sprintf("template%d", i),
			Source: "content",
		})
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = storage.Get(ctx, fmt.Sprintf("template%d", i%100))
			i++
		}
	})
}

// =============================================================================
// TEMPLATE SIZE BENCHMARKS
// =============================================================================

func BenchmarkExecute_SmallTemplate(b *testing.B) {
	benchmarkTemplateSize(b, 100)
}

func BenchmarkExecute_MediumTemplate(b *testing.B) {
	benchmarkTemplateSize(b, 1000)
}

func BenchmarkExecute_LargeTemplate(b *testing.B) {
	benchmarkTemplateSize(b, 10000)
}

func benchmarkTemplateSize(b *testing.B, size int) {
	engine := MustNew()

	// Build template with roughly the target size
	var sb strings.Builder
	sb.WriteString("Start: {~prompty.var name=\"user\" /~}\n")
	for sb.Len() < size {
		sb.WriteString("Line of content with {~prompty.var name=\"x\" /~} variable.\n")
	}
	sb.WriteString("End: {~prompty.var name=\"count\" /~}")

	source := sb.String()
	tmpl, _ := engine.Parse(source)
	data := map[string]any{"user": "Alice", "x": "value", "count": 42}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tmpl.Execute(ctx, data)
	}
}

// =============================================================================
// MEMORY ALLOCATION BENCHMARKS
// =============================================================================

func BenchmarkParse_Allocs(b *testing.B) {
	engine := MustNew()
	source := `Hello {~prompty.var name="user" /~}! Welcome to {~prompty.var name="app" /~}.`

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Parse(source)
	}
}

func BenchmarkExecute_Allocs(b *testing.B) {
	engine := MustNew()
	source := `Hello {~prompty.var name="user" /~}! Welcome to {~prompty.var name="app" /~}.`
	tmpl, _ := engine.Parse(source)
	data := map[string]any{"user": "Alice", "app": "MyApp"}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tmpl.Execute(ctx, data)
	}
}

func BenchmarkCacheHit_Allocs(b *testing.B) {
	engine := MustNew()
	cached := NewCachedEngine(engine, DefaultResultCacheConfig())
	source := `Hello {~prompty.var name="user" /~}!`
	data := map[string]any{"user": "Alice"}
	ctx := context.Background()

	_, _ = cached.Execute(ctx, source, data) // Warm

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cached.Execute(ctx, source, data)
	}
}

// =============================================================================
// INCLUDE/NESTED TEMPLATE BENCHMARKS
// =============================================================================

func BenchmarkExecute_Include_Single(b *testing.B) {
	engine := MustNew()
	engine.MustRegisterTemplate("header", "=== Header ===")
	source := `{~prompty.include template="header" /~}
Content here`
	tmpl, _ := engine.Parse(source)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tmpl.Execute(ctx, nil)
	}
}

func BenchmarkExecute_Include_Nested(b *testing.B) {
	engine := MustNew()
	engine.MustRegisterTemplate("inner", "Inner: {~prompty.var name=\"x\" /~}")
	engine.MustRegisterTemplate("middle", `Middle: {~prompty.include template="inner" x="value" /~}`)
	engine.MustRegisterTemplate("outer", `Outer: {~prompty.include template="middle" /~}`)
	source := `{~prompty.include template="outer" /~}`
	tmpl, _ := engine.Parse(source)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tmpl.Execute(ctx, nil)
	}
}

// =============================================================================
// DRY RUN AND VALIDATION BENCHMARKS
// =============================================================================

func BenchmarkDryRun(b *testing.B) {
	engine := MustNew()
	source := `Hello {~prompty.var name="user" /~}! Count: {~prompty.var name="count" /~}`
	tmpl, _ := engine.Parse(source)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tmpl.DryRun(ctx, nil)
	}
}

func BenchmarkExplain(b *testing.B) {
	engine := MustNew()
	source := `Hello {~prompty.var name="user.name" /~}! Email: {~prompty.var name="user.email" /~}`
	tmpl, _ := engine.Parse(source)
	data := map[string]any{
		"user": map[string]any{
			"name":  "Alice",
			"email": "alice@example.com",
		},
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tmpl.Explain(ctx, data)
	}
}

// =============================================================================
// TEMPLATE REGISTRATION BENCHMARKS
// =============================================================================

func BenchmarkRegisterTemplate(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine := MustNew()
		for j := 0; j < 100; j++ {
			_ = engine.RegisterTemplate(fmt.Sprintf("template%d", j), "content")
		}
	}
}

func BenchmarkExecuteRegistered(b *testing.B) {
	engine := MustNew()
	for i := 0; i < 100; i++ {
		engine.MustRegisterTemplate(fmt.Sprintf("template%d", i), `Hello {~prompty.var name="user" /~}!`)
	}
	data := map[string]any{"user": "Alice"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.ExecuteTemplate(ctx, fmt.Sprintf("template%d", i%100), data)
	}
}

// =============================================================================
// CONTEXT OPERATIONS BENCHMARKS
// =============================================================================

func BenchmarkContext_Get(b *testing.B) {
	ctx := NewContext(map[string]any{
		"user": map[string]any{
			"profile": map[string]any{
				"name":  "Alice",
				"email": "alice@example.com",
			},
		},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ctx.Get("user.profile.name")
	}
}

func BenchmarkContext_Set(b *testing.B) {
	ctx := NewContext(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Set("key", "value")
	}
}

func BenchmarkContext_Child(b *testing.B) {
	ctx := NewContext(map[string]any{"existing": "data"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ctx.Child(map[string]any{"key": "value"})
	}
}

// =============================================================================
// COMPREHENSIVE COMPARISON
// =============================================================================

func BenchmarkComparison_ParseVsExecute(b *testing.B) {
	engine := MustNew()
	source := `Hello {~prompty.var name="user" /~}, welcome to {~prompty.var name="app" /~}!`
	data := map[string]any{"user": "Alice", "app": "MyApp"}
	ctx := context.Background()

	b.Run("ParseOnly", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = engine.Parse(source)
		}
	})

	tmpl, _ := engine.Parse(source)
	b.Run("ExecuteOnly", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = tmpl.Execute(ctx, data)
		}
	})

	b.Run("ParseAndExecute", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = engine.Execute(ctx, source, data)
		}
	})
}

// =============================================================================
// PARALLEL SCALING BENCHMARKS
// =============================================================================

func BenchmarkParallelScaling(b *testing.B) {
	engine := MustNew()
	source := `Hello {~prompty.var name="user" /~}! ID: {~prompty.var name="id" /~}`
	tmpl, _ := engine.Parse(source)
	ctx := context.Background()

	for _, goroutines := range []int{1, 2, 4, 8, 16} {
		b.Run(fmt.Sprintf("Goroutines-%d", goroutines), func(b *testing.B) {
			var wg sync.WaitGroup
			iterations := b.N / goroutines
			if iterations == 0 {
				iterations = 1
			}

			b.ResetTimer()
			for g := 0; g < goroutines; g++ {
				wg.Add(1)
				go func(gid int) {
					defer wg.Done()
					for i := 0; i < iterations; i++ {
						data := map[string]any{"user": "Alice", "id": gid*iterations + i}
						_, _ = tmpl.Execute(ctx, data)
					}
				}(g)
			}
			wg.Wait()
		})
	}
}
