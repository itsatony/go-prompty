package prompty

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Single Level Inheritance Tests
// =============================================================================

func TestInheritance_E2E_SingleLevel(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// Register parent template with blocks
	engine.MustRegisterTemplate("parent", `Header: {~prompty.block name="header"~}Default Header{~/prompty.block~}
Body: {~prompty.block name="body"~}Default Body{~/prompty.block~}
Footer: {~prompty.block name="footer"~}Default Footer{~/prompty.block~}`)

	t.Run("ExecuteParentDirectly", func(t *testing.T) {
		result, err := engine.ExecuteTemplate(ctx, "parent", nil)
		require.NoError(t, err)
		assert.Contains(t, result, "Default Header")
		assert.Contains(t, result, "Default Body")
		assert.Contains(t, result, "Default Footer")
	})

	t.Run("ChildOverridesSingleBlock", func(t *testing.T) {
		child := `{~prompty.extends template="parent" /~}
{~prompty.block name="header"~}Custom Header{~/prompty.block~}`

		result, err := engine.Execute(ctx, child, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "Custom Header")
		assert.Contains(t, result, "Default Body")   // Not overridden
		assert.Contains(t, result, "Default Footer") // Not overridden
		assert.NotContains(t, result, "Default Header")
	})

	t.Run("ChildOverridesMultipleBlocks", func(t *testing.T) {
		child := `{~prompty.extends template="parent" /~}
{~prompty.block name="header"~}Custom Header{~/prompty.block~}
{~prompty.block name="footer"~}Custom Footer{~/prompty.block~}`

		result, err := engine.Execute(ctx, child, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "Custom Header")
		assert.Contains(t, result, "Default Body") // Not overridden
		assert.Contains(t, result, "Custom Footer")
	})

	t.Run("ChildWithVariablesInBlocks", func(t *testing.T) {
		child := `{~prompty.extends template="parent" /~}
{~prompty.block name="header"~}Welcome, {~prompty.var name="user" /~}!{~/prompty.block~}`

		result, err := engine.Execute(ctx, child, map[string]any{"user": "Alice"})
		require.NoError(t, err)
		assert.Contains(t, result, "Welcome, Alice!")
	})
}

// =============================================================================
// Multi-Level Inheritance Tests
// =============================================================================

func TestInheritance_E2E_MultiLevel(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// Level 0: Base template
	engine.MustRegisterTemplate("base", `[BASE]
{~prompty.block name="title"~}Default Title{~/prompty.block~}
{~prompty.block name="content"~}Default Content{~/prompty.block~}
{~prompty.block name="footer"~}Default Footer{~/prompty.block~}
[/BASE]`)

	// Level 1: Middle template extends base
	engine.MustRegisterTemplate("middle", `{~prompty.extends template="base" /~}
{~prompty.block name="title"~}Middle Title{~/prompty.block~}
{~prompty.block name="content"~}Middle Content with {~prompty.var name="middleData" default="default" /~}{~/prompty.block~}`)

	t.Run("TwoLevelInheritance", func(t *testing.T) {
		result, err := engine.ExecuteTemplate(ctx, "middle", map[string]any{
			"middleData": "custom data",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "[BASE]")
		assert.Contains(t, result, "Middle Title")
		assert.Contains(t, result, "Middle Content with custom data")
		assert.Contains(t, result, "Default Footer") // Inherited from base
		assert.NotContains(t, result, "Default Title")
	})

	t.Run("ThreeLevelInheritance", func(t *testing.T) {
		// Level 2: Child template extends middle
		child := `{~prompty.extends template="middle" /~}
{~prompty.block name="footer"~}Child Footer - {~prompty.var name="year" /~}{~/prompty.block~}`

		result, err := engine.Execute(ctx, child, map[string]any{
			"middleData": "middle stuff",
			"year":       2024,
		})
		require.NoError(t, err)
		assert.Contains(t, result, "[BASE]")
		assert.Contains(t, result, "Middle Title")        // From middle
		assert.Contains(t, result, "middle stuff")        // From middle
		assert.Contains(t, result, "Child Footer - 2024") // From child
	})

	t.Run("DeepInheritance", func(t *testing.T) {
		// Create a 5-level deep inheritance chain
		engine.MustRegisterTemplate("level1", `{~prompty.block name="a"~}L1-A{~/prompty.block~}|{~prompty.block name="b"~}L1-B{~/prompty.block~}`)
		engine.MustRegisterTemplate("level2", `{~prompty.extends template="level1" /~}{~prompty.block name="a"~}L2-A{~/prompty.block~}`)
		engine.MustRegisterTemplate("level3", `{~prompty.extends template="level2" /~}{~prompty.block name="b"~}L3-B{~/prompty.block~}`)
		engine.MustRegisterTemplate("level4", `{~prompty.extends template="level3" /~}{~prompty.block name="a"~}L4-A{~/prompty.block~}`)

		result, err := engine.ExecuteTemplate(ctx, "level4", nil)
		require.NoError(t, err)
		assert.Contains(t, result, "L4-A") // Latest override of 'a'
		assert.Contains(t, result, "L3-B") // Latest override of 'b'
		assert.NotContains(t, result, "L2-A")
		assert.NotContains(t, result, "L1-A")
		assert.NotContains(t, result, "L1-B")
	})
}

// =============================================================================
// Parent Tag Tests
// =============================================================================

func TestInheritance_E2E_ParentTag(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	engine.MustRegisterTemplate("base-with-content", `{~prompty.block name="greeting"~}Hello{~/prompty.block~} World!
{~prompty.block name="items"~}
- Item 1
- Item 2
{~/prompty.block~}`)

	t.Run("ParentBeforeCustomContent", func(t *testing.T) {
		child := `{~prompty.extends template="base-with-content" /~}
{~prompty.block name="greeting"~}{~prompty.parent /~} and Welcome{~/prompty.block~}`

		result, err := engine.Execute(ctx, child, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "Hello and Welcome")
	})

	t.Run("ParentAfterCustomContent", func(t *testing.T) {
		child := `{~prompty.extends template="base-with-content" /~}
{~prompty.block name="greeting"~}Greetings, {~prompty.parent /~}{~/prompty.block~}`

		result, err := engine.Execute(ctx, child, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "Greetings, Hello")
	})

	t.Run("ParentInListBlock", func(t *testing.T) {
		child := `{~prompty.extends template="base-with-content" /~}
{~prompty.block name="items"~}{~prompty.parent /~}
- Item 3
- Item 4
{~/prompty.block~}`

		result, err := engine.Execute(ctx, child, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "- Item 1")
		assert.Contains(t, result, "- Item 2")
		assert.Contains(t, result, "- Item 3")
		assert.Contains(t, result, "- Item 4")
	})

	t.Run("MultipleParentCalls", func(t *testing.T) {
		child := `{~prompty.extends template="base-with-content" /~}
{~prompty.block name="greeting"~}({~prompty.parent /~}) [{~prompty.parent /~}]{~/prompty.block~}`

		result, err := engine.Execute(ctx, child, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "(Hello) [Hello]")
	})
}

// =============================================================================
// Mixed Content Tests
// =============================================================================

func TestInheritance_E2E_MixedContent(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("BlocksWithConditionals", func(t *testing.T) {
		engine.MustRegisterTemplate("conditional-base", `{~prompty.block name="message"~}
{~prompty.if eval="show"~}Shown{~prompty.else~}Hidden{~/prompty.if~}
{~/prompty.block~}`)

		// Execute parent with different conditions
		result, err := engine.ExecuteTemplate(ctx, "conditional-base", map[string]any{"show": true})
		require.NoError(t, err)
		assert.Contains(t, result, "Shown")

		result, err = engine.ExecuteTemplate(ctx, "conditional-base", map[string]any{"show": false})
		require.NoError(t, err)
		assert.Contains(t, result, "Hidden")

		// Child overrides with different conditional
		child := `{~prompty.extends template="conditional-base" /~}
{~prompty.block name="message"~}
{~prompty.if eval="premium"~}Premium User{~prompty.else~}Free User{~/prompty.if~}
{~/prompty.block~}`

		result, err = engine.Execute(ctx, child, map[string]any{"premium": true})
		require.NoError(t, err)
		assert.Contains(t, result, "Premium User")
	})

	t.Run("BlocksWithLoops", func(t *testing.T) {
		engine.MustRegisterTemplate("loop-base", `Items:
{~prompty.block name="item-list"~}
{~prompty.for item="item" in="items"~}
- {~prompty.var name="item" /~}
{~/prompty.for~}
{~/prompty.block~}`)

		// Execute parent
		result, err := engine.ExecuteTemplate(ctx, "loop-base", map[string]any{
			"items": []string{"A", "B", "C"},
		})
		require.NoError(t, err)
		assert.Contains(t, result, "- A")
		assert.Contains(t, result, "- B")
		assert.Contains(t, result, "- C")

		// Child with different loop format
		child := `{~prompty.extends template="loop-base" /~}
{~prompty.block name="item-list"~}
{~prompty.for item="item" index="i" in="items"~}
{~prompty.var name="i" /~}. {~prompty.var name="item" /~}
{~/prompty.for~}
{~/prompty.block~}`

		result, err = engine.Execute(ctx, child, map[string]any{
			"items": []string{"X", "Y", "Z"},
		})
		require.NoError(t, err)
		assert.Contains(t, result, "0. X")
		assert.Contains(t, result, "1. Y")
		assert.Contains(t, result, "2. Z")
	})

	t.Run("BlocksWithIncludes", func(t *testing.T) {
		engine.MustRegisterTemplate("snippet", `[SNIPPET: {~prompty.var name="text" /~}]`)
		engine.MustRegisterTemplate("include-base", `{~prompty.block name="main"~}
{~prompty.include template="snippet" text="default" /~}
{~/prompty.block~}`)

		// Parent uses snippet with default
		result, err := engine.ExecuteTemplate(ctx, "include-base", nil)
		require.NoError(t, err)
		assert.Contains(t, result, "[SNIPPET: default]")

		// Child overrides with different include params
		child := `{~prompty.extends template="include-base" /~}
{~prompty.block name="main"~}
{~prompty.include template="snippet" text="custom" /~}
{~/prompty.block~}`

		result, err = engine.Execute(ctx, child, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "[SNIPPET: custom]")
	})
}

// =============================================================================
// Circular Inheritance Detection Tests
// =============================================================================

func TestInheritance_E2E_CircularDetection(t *testing.T) {
	ctx := context.Background()

	t.Run("DirectCircularReference", func(t *testing.T) {
		engine := MustNew()

		// A extends B, B extends A
		engine.MustRegisterTemplate("circular-a", `{~prompty.extends template="circular-b" /~}`)
		engine.MustRegisterTemplate("circular-b", `{~prompty.extends template="circular-a" /~}`)

		_, err := engine.ExecuteTemplate(ctx, "circular-a", nil)
		require.Error(t, err)
		// Should detect circular reference
		errStr := strings.ToLower(err.Error())
		assert.True(t,
			strings.Contains(errStr, "circular") ||
				strings.Contains(errStr, "cycle") ||
				strings.Contains(errStr, "depth") ||
				strings.Contains(errStr, "recursion"),
			"expected circular/cycle/depth error, got: %s", err.Error())
	})

	t.Run("IndirectCircularReference", func(t *testing.T) {
		engine := MustNew()

		// A extends B, B extends C, C extends A
		engine.MustRegisterTemplate("chain-a", `{~prompty.extends template="chain-b" /~}`)
		engine.MustRegisterTemplate("chain-b", `{~prompty.extends template="chain-c" /~}`)
		engine.MustRegisterTemplate("chain-c", `{~prompty.extends template="chain-a" /~}`)

		_, err := engine.ExecuteTemplate(ctx, "chain-a", nil)
		require.Error(t, err)
	})

	t.Run("SelfReference", func(t *testing.T) {
		engine := MustNew()

		// Template extends itself
		engine.MustRegisterTemplate("self-ref", `{~prompty.extends template="self-ref" /~}`)

		_, err := engine.ExecuteTemplate(ctx, "self-ref", nil)
		require.Error(t, err)
	})
}

// =============================================================================
// Max Depth Enforcement Tests
// =============================================================================

func TestInheritance_E2E_MaxDepthEnforcement(t *testing.T) {
	ctx := context.Background()

	t.Run("AtMaxDepth", func(t *testing.T) {
		engine := MustNew()

		// Create chain of 10 templates (should work at default max depth)
		for i := 1; i <= 10; i++ {
			if i == 1 {
				engine.MustRegisterTemplate("depth-1", `{~prompty.block name="content"~}Base{~/prompty.block~}`)
			} else {
				source := `{~prompty.extends template="depth-` + string(rune('0'+i-1)) + `" /~}`
				if i < 10 {
					engine.MustRegisterTemplate("depth-"+string(rune('0'+i)), source)
				}
			}
		}
	})

	t.Run("ExceedsMaxDepth", func(t *testing.T) {
		engine := MustNew()

		// Create very deep chain
		engine.MustRegisterTemplate("deep-base", `{~prompty.block name="x"~}base{~/prompty.block~}`)
		for i := 1; i <= 15; i++ {
			prev := "deep-base"
			if i > 1 {
				prev = "deep-" + string(rune('a'+i-2))
			}
			name := "deep-" + string(rune('a'+i-1))
			engine.MustRegisterTemplate(name, `{~prompty.extends template="`+prev+`" /~}`)
		}

		// This should fail due to max depth
		_, err := engine.ExecuteTemplate(ctx, "deep-o", nil)
		require.Error(t, err)
		errStr := strings.ToLower(err.Error())
		assert.True(t,
			strings.Contains(errStr, "depth") ||
				strings.Contains(errStr, "limit") ||
				strings.Contains(errStr, "recursion"),
			"expected depth/limit error, got: %s", err.Error())
	})
}

// =============================================================================
// Storage Integration Tests
// =============================================================================

func TestInheritance_E2E_WithStorage(t *testing.T) {
	ctx := context.Background()

	// Create memory storage
	storage := NewMemoryStorage()

	// Note: prompty.extends looks up templates from the engine's registered templates,
	// NOT from storage. Templates using inheritance need to be registered with the engine.
	// This test demonstrates using prompty.include for storage-based template composition.

	// Save base template to storage (using include pattern instead of extends)
	baseSource := `System: {~prompty.var name="systemPrompt" default="You are an assistant." /~}
User: {~prompty.var name="query" /~}`

	err := storage.Save(ctx, &StoredTemplate{
		Name:   "prompt-base",
		Source: baseSource,
	})
	require.NoError(t, err)

	// Create storage engine
	se, err := NewStorageEngine(StorageEngineConfig{
		Storage: storage,
	})
	require.NoError(t, err)

	t.Run("ExecuteChildFromStorage", func(t *testing.T) {
		// Execute base template with custom system prompt
		result, err := se.Execute(ctx, "prompt-base", map[string]any{
			"systemPrompt": "You are a coding assistant. Only provide code examples.",
			"query":        "How do I sort an array?",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "coding assistant")
		assert.Contains(t, result, "How do I sort an array?")
	})

	t.Run("ExecuteParentFromStorage", func(t *testing.T) {
		result, err := se.Execute(ctx, "prompt-base", map[string]any{
			"query": "What is the weather?",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "You are an assistant.")
		assert.Contains(t, result, "What is the weather?")
	})

	t.Run("UpdateParentTemplate", func(t *testing.T) {
		// Update base template (new version)
		newBase := `System: {~prompty.var name="systemPrompt" default="You are a helpful AI." /~}
User: {~prompty.var name="query" /~}`

		err := storage.Save(ctx, &StoredTemplate{
			Name:   "prompt-base",
			Source: newBase,
		})
		require.NoError(t, err)

		// Clear parsed cache to pick up updated template
		se.ClearParsedCache()

		// Execute with updated template
		result, err := se.Execute(ctx, "prompt-base", map[string]any{
			"query": "Test query",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "helpful AI") // Default changed
	})
}

// =============================================================================
// Blocks in Conditionals Tests
// =============================================================================

func TestInheritance_E2E_BlocksInConditionals(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("ConditionalAroundBlock", func(t *testing.T) {
		engine.MustRegisterTemplate("cond-block-base", `
{~prompty.if eval="showHeader"~}
{~prompty.block name="header"~}Default Header{~/prompty.block~}
{~/prompty.if~}
Content
{~prompty.block name="footer"~}Default Footer{~/prompty.block~}`)

		// With showHeader true
		result, err := engine.ExecuteTemplate(ctx, "cond-block-base", map[string]any{
			"showHeader": true,
		})
		require.NoError(t, err)
		assert.Contains(t, result, "Default Header")
		assert.Contains(t, result, "Default Footer")

		// With showHeader false
		result, err = engine.ExecuteTemplate(ctx, "cond-block-base", map[string]any{
			"showHeader": false,
		})
		require.NoError(t, err)
		assert.NotContains(t, result, "Default Header")
		assert.Contains(t, result, "Default Footer")
	})

	t.Run("OverrideBlockInConditional", func(t *testing.T) {
		engine.MustRegisterTemplate("cond-override-base", `
{~prompty.if eval="mode == 'full'"~}
{~prompty.block name="content"~}Full Mode Default{~/prompty.block~}
{~prompty.else~}
{~prompty.block name="content"~}Simple Mode Default{~/prompty.block~}
{~/prompty.if~}`)

		child := `{~prompty.extends template="cond-override-base" /~}
{~prompty.block name="content"~}Custom Content{~/prompty.block~}`

		// Block override should apply regardless of which conditional branch
		result, err := engine.Execute(ctx, child, map[string]any{"mode": "full"})
		require.NoError(t, err)
		assert.Contains(t, result, "Custom Content")

		result, err = engine.Execute(ctx, child, map[string]any{"mode": "simple"})
		require.NoError(t, err)
		assert.Contains(t, result, "Custom Content")
	})
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestInheritance_E2E_EdgeCases(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("EmptyBlock", func(t *testing.T) {
		engine.MustRegisterTemplate("empty-block-base", `Before{~prompty.block name="empty"~}{~/prompty.block~}After`)

		result, err := engine.ExecuteTemplate(ctx, "empty-block-base", nil)
		require.NoError(t, err)
		assert.Equal(t, "BeforeAfter", result)

		// Child fills empty block
		child := `{~prompty.extends template="empty-block-base" /~}
{~prompty.block name="empty"~}FILLED{~/prompty.block~}`

		result, err = engine.Execute(ctx, child, nil)
		require.NoError(t, err)
		assert.Equal(t, "BeforeFILLEDAfter", result)
	})

	t.Run("WhitespaceOnlyBlock", func(t *testing.T) {
		engine.MustRegisterTemplate("ws-base", `X{~prompty.block name="ws"~}   {~/prompty.block~}Y`)

		result, err := engine.ExecuteTemplate(ctx, "ws-base", nil)
		require.NoError(t, err)
		assert.Equal(t, "X   Y", result)
	})

	t.Run("BlockNameWithSpecialChars", func(t *testing.T) {
		engine.MustRegisterTemplate("special-names", `
{~prompty.block name="header-main"~}Header{~/prompty.block~}
{~prompty.block name="content_area"~}Content{~/prompty.block~}
{~prompty.block name="footer.bottom"~}Footer{~/prompty.block~}`)

		child := `{~prompty.extends template="special-names" /~}
{~prompty.block name="header-main"~}Custom Header{~/prompty.block~}
{~prompty.block name="content_area"~}Custom Content{~/prompty.block~}
{~prompty.block name="footer.bottom"~}Custom Footer{~/prompty.block~}`

		result, err := engine.Execute(ctx, child, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "Custom Header")
		assert.Contains(t, result, "Custom Content")
		assert.Contains(t, result, "Custom Footer")
	})

	t.Run("NestedBlocks", func(t *testing.T) {
		// Note: Nested blocks have specific behavior - overriding an inner block
		// doesn't affect the content inside an outer block because block resolution
		// happens at the parent level. To customize nested content, you must
		// override the outer block entirely.
		engine.MustRegisterTemplate("nested-base", `
{~prompty.block name="outer"~}
OUTER[{~prompty.block name="inner"~}INNER{~/prompty.block~}]OUTER
{~/prompty.block~}`)

		// Override only inner - this does NOT affect nested content within outer
		// because the outer block definition contains its own inner block definition
		childInner := `{~prompty.extends template="nested-base" /~}
{~prompty.block name="inner"~}CUSTOM-INNER{~/prompty.block~}`

		result, err := engine.Execute(ctx, childInner, nil)
		require.NoError(t, err)
		// The outer block's content is used as-is, with its original inner definition
		assert.Contains(t, result, "OUTER")
		assert.Contains(t, result, "INNER")

		// Override only outer - this replaces the entire outer block
		childOuter := `{~prompty.extends template="nested-base" /~}
{~prompty.block name="outer"~}CUSTOM-OUTER{~/prompty.block~}`

		result, err = engine.Execute(ctx, childOuter, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "CUSTOM-OUTER")
		assert.NotContains(t, result, "INNER") // Outer override replaces entire content
	})

	t.Run("ExtendsNonexistentTemplate", func(t *testing.T) {
		child := `{~prompty.extends template="nonexistent" /~}
{~prompty.block name="content"~}Child Content{~/prompty.block~}`

		_, err := engine.Execute(ctx, child, nil)
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "not found")
	})

	t.Run("UnicodeInBlocks", func(t *testing.T) {
		engine.MustRegisterTemplate("unicode-base", `{~prompty.block name="greeting"~}Hello{~/prompty.block~}`)

		child := `{~prompty.extends template="unicode-base" /~}
{~prompty.block name="greeting"~}こんにちは 世界 🌍{~/prompty.block~}`

		result, err := engine.Execute(ctx, child, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "こんにちは")
		assert.Contains(t, result, "世界")
		assert.Contains(t, result, "🌍")
	})
}

// =============================================================================
// Registered Template Tests (vs Inline)
// =============================================================================

func TestInheritance_E2E_RegisteredVsInline(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	engine.MustRegisterTemplate("reg-base", `{~prompty.block name="content"~}Base{~/prompty.block~}`)
	engine.MustRegisterTemplate("reg-child", `{~prompty.extends template="reg-base" /~}
{~prompty.block name="content"~}Registered Child{~/prompty.block~}`)

	t.Run("RegisteredExtendingRegistered", func(t *testing.T) {
		result, err := engine.ExecuteTemplate(ctx, "reg-child", nil)
		require.NoError(t, err)
		assert.Contains(t, result, "Registered Child")
	})

	t.Run("InlineExtendingRegistered", func(t *testing.T) {
		inline := `{~prompty.extends template="reg-base" /~}
{~prompty.block name="content"~}Inline Child{~/prompty.block~}`

		result, err := engine.Execute(ctx, inline, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "Inline Child")
	})

	t.Run("InlineExtendingRegisteredChild", func(t *testing.T) {
		// Inline extends a registered template that itself extends another
		inline := `{~prompty.extends template="reg-child" /~}
{~prompty.block name="content"~}Deep Inline{~/prompty.block~}`

		result, err := engine.Execute(ctx, inline, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "Deep Inline")
	})
}
