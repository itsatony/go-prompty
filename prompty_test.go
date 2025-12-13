package prompty_test

import (
	"context"
	"errors"
	"testing"

	"github.com/itsatony/go-prompty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2E Integration Tests - Zero Mocks
// These tests exercise the full system from public API through to final output.

func TestE2E_BasicVariableInterpolation(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		"Hello, {~prompty.var name=\"user\" /~}!",
		map[string]any{"user": "Alice"},
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello, Alice!", result)
}

func TestE2E_NestedVariablePath(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		"Welcome {~prompty.var name=\"user.profile.name\" /~}!",
		map[string]any{
			"user": map[string]any{
				"profile": map[string]any{
					"name": "Bob",
				},
			},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "Welcome Bob!", result)
}

func TestE2E_DefaultValue(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		"Hello, {~prompty.var name=\"missing\" default=\"Guest\" /~}!",
		map[string]any{},
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello, Guest!", result)
}

func TestE2E_MissingVariableThrows(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		"Hello, {~prompty.var name=\"missing\" /~}!",
		map[string]any{},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

func TestE2E_RawBlock(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		"Template: {~prompty.raw~}{{ jinja.var }}{~/prompty.raw~}",
		map[string]any{},
	)

	require.NoError(t, err)
	assert.Equal(t, "Template: {{ jinja.var }}", result)
}

func TestE2E_RawBlockPreservesPromptyTags(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.raw~}{~prompty.var name="x" /~}{~/prompty.raw~}`,
		map[string]any{},
	)

	require.NoError(t, err)
	assert.Equal(t, `{~prompty.var name="x" /~}`, result)
}

func TestE2E_EscapeSequence(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`Use \{~ for literal delimiters`,
		map[string]any{},
	)

	require.NoError(t, err)
	assert.Contains(t, result, "{~")
}

func TestE2E_CustomResolver(t *testing.T) {
	engine := prompty.MustNew()

	// Register custom resolver
	engine.MustRegister(&uppercaseResolver{})

	result, err := engine.Execute(context.Background(),
		`{~myapp.uppercase text="hello world" /~}`,
		map[string]any{},
	)

	require.NoError(t, err)
	assert.Equal(t, "HELLO WORLD", result)
}

func TestE2E_UnknownTagThrows(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~unknown.tag /~}`,
		map[string]any{},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown")
}

func TestE2E_ParseOnceExecuteMany(t *testing.T) {
	engine := prompty.MustNew()

	// Parse once
	tmpl, err := engine.Parse("Hello, {~prompty.var name=\"user\" /~}!")
	require.NoError(t, err)

	// Execute multiple times with different data
	users := []string{"Alice", "Bob", "Charlie"}
	for _, user := range users {
		result, err := tmpl.Execute(context.Background(), map[string]any{"user": user})
		require.NoError(t, err)
		assert.Equal(t, "Hello, "+user+"!", result)
	}
}

func TestE2E_ComplexTemplate(t *testing.T) {
	engine := prompty.MustNew()

	template := `System: {~prompty.var name="system_prompt" /~}

User: {~prompty.var name="user.name" default="User" /~}

{~prompty.raw~}<assistant>
Please respond to the following query: {{query}}
</assistant>{~/prompty.raw~}

Context: {~prompty.var name="context" default="No context provided" /~}`

	result, err := engine.Execute(context.Background(), template, map[string]any{
		"system_prompt": "You are a helpful assistant.",
		"user": map[string]any{
			"name": "Alice",
		},
	})

	require.NoError(t, err)
	assert.Contains(t, result, "You are a helpful assistant.")
	assert.Contains(t, result, "Alice")
	assert.Contains(t, result, "{{query}}")
	assert.Contains(t, result, "No context provided")
}

func TestE2E_PlainTextOnly(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		"Just plain text, no tags here.",
		map[string]any{},
	)

	require.NoError(t, err)
	assert.Equal(t, "Just plain text, no tags here.", result)
}

func TestE2E_EmptyTemplate(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(), "", map[string]any{})

	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestE2E_MultipleVariables(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.var name="greeting" /~}, {~prompty.var name="name" /~}! Today is {~prompty.var name="day" /~}.`,
		map[string]any{
			"greeting": "Hello",
			"name":     "World",
			"day":      "Monday",
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello, World! Today is Monday.", result)
}

func TestE2E_CustomDelimiters(t *testing.T) {
	engine := prompty.MustNew(prompty.WithDelimiters("<%", "%>"))

	result, err := engine.Execute(context.Background(),
		"Hello, <%prompty.var name=\"user\" /%>!",
		map[string]any{"user": "Alice"},
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello, Alice!", result)
}

func TestE2E_NumericValues(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		"Count: {~prompty.var name=\"count\" /~}, Price: ${~prompty.var name=\"price\" /~}",
		map[string]any{
			"count": 42,
			"price": 19.99,
		},
	)

	require.NoError(t, err)
	assert.Contains(t, result, "Count: 42")
	assert.Contains(t, result, "Price: $19.99")
}

func TestE2E_BooleanValues(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		"Active: {~prompty.var name=\"active\" /~}",
		map[string]any{"active": true},
	)

	require.NoError(t, err)
	assert.Equal(t, "Active: true", result)
}

func TestE2E_TemplateSource(t *testing.T) {
	engine := prompty.MustNew()

	source := "Hello, {~prompty.var name=\"user\" /~}!"
	tmpl, err := engine.Parse(source)
	require.NoError(t, err)

	assert.Equal(t, source, tmpl.Source())
}

func TestE2E_ExecuteWithContext(t *testing.T) {
	engine := prompty.MustNew()

	tmpl, err := engine.Parse("{~prompty.var name=\"key\" /~}")
	require.NoError(t, err)

	// Create context with parent-child relationship
	parent := prompty.NewContext(map[string]any{"parentKey": "parentValue"})
	child := parent.Child(map[string]any{"key": "childValue"})

	result, err := tmpl.ExecuteWithContext(context.Background(), child)
	require.NoError(t, err)
	assert.Equal(t, "childValue", result)
}

func TestE2E_ContextParentFallback(t *testing.T) {
	engine := prompty.MustNew()

	tmpl, err := engine.Parse("{~prompty.var name=\"parentKey\" /~}")
	require.NoError(t, err)

	parent := prompty.NewContext(map[string]any{"parentKey": "fromParent"})
	child := parent.Child(map[string]any{})

	result, err := tmpl.ExecuteWithContext(context.Background(), child)
	require.NoError(t, err)
	assert.Equal(t, "fromParent", result)
}

func TestE2E_ResolverValidation(t *testing.T) {
	engine := prompty.MustNew()

	// prompty.var requires 'name' attribute
	_, err := engine.Execute(context.Background(),
		`{~prompty.var /~}`, // Missing name attribute
		map[string]any{},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestE2E_WhitespacePreservation(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		"  Leading and trailing  ",
		map[string]any{},
	)

	require.NoError(t, err)
	assert.Equal(t, "  Leading and trailing  ", result)
}

func TestE2E_NewlinePreservation(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		"Line 1\nLine 2\nLine 3",
		map[string]any{},
	)

	require.NoError(t, err)
	assert.Equal(t, "Line 1\nLine 2\nLine 3", result)
}

func TestE2E_DuplicateResolverRegistration(t *testing.T) {
	engine := prompty.MustNew()

	// First registration should succeed
	err := engine.Register(&uppercaseResolver{})
	require.NoError(t, err)

	// Second registration of same tag name should fail
	err = engine.Register(&uppercaseResolver{})
	require.Error(t, err)
}

func TestE2E_MustNewDoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		prompty.MustNew()
	})
}

func TestE2E_NewReturnsNoError(t *testing.T) {
	engine, err := prompty.New()
	require.NoError(t, err)
	require.NotNil(t, engine)
}

func TestE2E_SingleQuotedAttributes(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.var name='user' /~}`,
		map[string]any{"user": "Alice"},
	)

	require.NoError(t, err)
	assert.Equal(t, "Alice", result)
}

func TestE2E_AttributeWithSpecialChars(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.var name="user" default="<default>" /~}`,
		map[string]any{},
	)

	require.NoError(t, err)
	assert.Equal(t, "<default>", result)
}

// Custom resolver for testing
type uppercaseResolver struct{}

func (r *uppercaseResolver) TagName() string {
	return "myapp.uppercase"
}

func (r *uppercaseResolver) Resolve(ctx context.Context, execCtx *prompty.Context, attrs prompty.Attributes) (string, error) {
	text, ok := attrs.Get("text")
	if !ok {
		return "", errors.New("missing text attribute")
	}
	result := ""
	for _, c := range text {
		if c >= 'a' && c <= 'z' {
			result += string(c - 32)
		} else {
			result += string(c)
		}
	}
	return result, nil
}

func (r *uppercaseResolver) Validate(attrs prompty.Attributes) error {
	if !attrs.Has("text") {
		return errors.New("missing text attribute")
	}
	return nil
}

// ============================================================================
// Nested Template Tests
// ============================================================================

func TestE2E_NestedTemplate_RegisterTemplate(t *testing.T) {
	engine := prompty.MustNew()

	err := engine.RegisterTemplate("footer", "Copyright 2024")
	require.NoError(t, err)

	assert.True(t, engine.HasTemplate("footer"))
	assert.Equal(t, 1, engine.TemplateCount())
}

func TestE2E_NestedTemplate_RegisterDuplicate(t *testing.T) {
	engine := prompty.MustNew()

	err := engine.RegisterTemplate("footer", "Copyright 2024")
	require.NoError(t, err)

	// Second registration should fail
	err = engine.RegisterTemplate("footer", "Different content")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestE2E_NestedTemplate_RegisterReservedName(t *testing.T) {
	engine := prompty.MustNew()

	// Names starting with "prompty." are reserved
	err := engine.RegisterTemplate("prompty.custom", "Content")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reserved")
}

func TestE2E_NestedTemplate_RegisterEmptyName(t *testing.T) {
	engine := prompty.MustNew()

	err := engine.RegisterTemplate("", "Content")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestE2E_NestedTemplate_UnregisterTemplate(t *testing.T) {
	engine := prompty.MustNew()

	engine.MustRegisterTemplate("footer", "Copyright 2024")
	assert.True(t, engine.HasTemplate("footer"))

	removed := engine.UnregisterTemplate("footer")
	assert.True(t, removed)
	assert.False(t, engine.HasTemplate("footer"))

	// Second unregister should return false
	removed = engine.UnregisterTemplate("footer")
	assert.False(t, removed)
}

func TestE2E_NestedTemplate_GetTemplate(t *testing.T) {
	engine := prompty.MustNew()

	engine.MustRegisterTemplate("footer", "Copyright 2024")

	tmpl, ok := engine.GetTemplate("footer")
	require.True(t, ok)
	assert.NotNil(t, tmpl)
	assert.Equal(t, "Copyright 2024", tmpl.Source())

	// Non-existent template
	_, ok = engine.GetTemplate("nonexistent")
	assert.False(t, ok)
}

func TestE2E_NestedTemplate_HasTemplate(t *testing.T) {
	engine := prompty.MustNew()

	assert.False(t, engine.HasTemplate("footer"))

	engine.MustRegisterTemplate("footer", "Copyright 2024")
	assert.True(t, engine.HasTemplate("footer"))
}

func TestE2E_NestedTemplate_ListTemplates(t *testing.T) {
	engine := prompty.MustNew()

	assert.Empty(t, engine.ListTemplates())

	engine.MustRegisterTemplate("footer", "Footer")
	engine.MustRegisterTemplate("header", "Header")
	engine.MustRegisterTemplate("sidebar", "Sidebar")

	templates := engine.ListTemplates()
	assert.Equal(t, []string{"footer", "header", "sidebar"}, templates)
}

func TestE2E_NestedTemplate_BasicInclude(t *testing.T) {
	engine := prompty.MustNew()

	engine.MustRegisterTemplate("footer", "Copyright 2024")

	result, err := engine.Execute(context.Background(),
		`Content goes here. {~prompty.include template="footer" /~}`,
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "Content goes here. Copyright 2024", result)
}

func TestE2E_NestedTemplate_WithVariables(t *testing.T) {
	engine := prompty.MustNew()

	engine.MustRegisterTemplate("greeting", "Hello, {~prompty.var name=\"user\" /~}!")

	_, err := engine.Execute(context.Background(),
		`{~prompty.include template="greeting" /~}`,
		map[string]any{"user": "Alice"},
	)

	// Note: Since we're not passing parent context data to child,
	// this should use empty context. We need to pass via attributes.
	require.Error(t, err) // Variable not found in isolated context
}

func TestE2E_NestedTemplate_ContextOverride(t *testing.T) {
	engine := prompty.MustNew()

	engine.MustRegisterTemplate("greeting", "Hello, {~prompty.var name=\"user\" default=\"Guest\" /~}!")

	result, err := engine.Execute(context.Background(),
		`{~prompty.include template="greeting" user="Bob" /~}`,
		map[string]any{"user": "Alice"},
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello, Bob!", result) // Should use override from attribute
}

func TestE2E_NestedTemplate_NotFound(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~prompty.include template="nonexistent" /~}`,
		nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestE2E_NestedTemplate_MissingTemplateAttr(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~prompty.include /~}`,
		nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "template")
}

func TestE2E_NestedTemplate_MaxDepthExceeded(t *testing.T) {
	// Use a small max depth for testing
	engine := prompty.MustNew(prompty.WithMaxDepth(3))

	// Register a template that includes itself
	engine.MustRegisterTemplate("recursive", `X{~prompty.include template="recursive" /~}`)

	_, err := engine.Execute(context.Background(),
		`{~prompty.include template="recursive" /~}`,
		nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "depth")
}

func TestE2E_NestedTemplate_MultiLevelNesting(t *testing.T) {
	engine := prompty.MustNew()

	engine.MustRegisterTemplate("level3", "Level3")
	engine.MustRegisterTemplate("level2", `L2[{~prompty.include template="level3" /~}]`)
	engine.MustRegisterTemplate("level1", `L1[{~prompty.include template="level2" /~}]`)

	result, err := engine.Execute(context.Background(),
		`Start[{~prompty.include template="level1" /~}]End`,
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "Start[L1[L2[Level3]]]End", result)
}

func TestE2E_NestedTemplate_SameTemplateMultipleTimes(t *testing.T) {
	engine := prompty.MustNew()

	engine.MustRegisterTemplate("sep", " | ")

	result, err := engine.Execute(context.Background(),
		`A{~prompty.include template="sep" /~}B{~prompty.include template="sep" /~}C`,
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "A | B | C", result)
}

func TestE2E_NestedTemplate_TemplateWithRawBlocks(t *testing.T) {
	engine := prompty.MustNew()

	engine.MustRegisterTemplate("jinja", "{~prompty.raw~}{{ jinja_var }}{~/prompty.raw~}")

	result, err := engine.Execute(context.Background(),
		`Template: {~prompty.include template="jinja" /~}`,
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "Template: {{ jinja_var }}", result)
}

func TestE2E_NestedTemplate_TemplateWithVariables(t *testing.T) {
	engine := prompty.MustNew()

	engine.MustRegisterTemplate("user-card", `Name: {~prompty.var name="name" default="Unknown" /~}`)

	result, err := engine.Execute(context.Background(),
		`{~prompty.include template="user-card" name="Alice" /~}`,
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "Name: Alice", result)
}

func TestE2E_NestedTemplate_ConcurrentExecution(t *testing.T) {
	engine := prompty.MustNew()

	engine.MustRegisterTemplate("greeting", "Hello!")

	// Execute concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			result, err := engine.Execute(context.Background(),
				`{~prompty.include template="greeting" /~}`,
				nil,
			)
			assert.NoError(t, err)
			assert.Equal(t, "Hello!", result)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestE2E_NestedTemplate_ConcurrentRegistration(t *testing.T) {
	engine := prompty.MustNew()

	// Register templates concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			name := "template" + string(rune('0'+idx))
			err := engine.RegisterTemplate(name, "Content "+name)
			// First registration should succeed, duplicates will fail
			// We just care that it doesn't panic or corrupt state
			_ = err
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify state is consistent
	count := engine.TemplateCount()
	assert.Equal(t, 10, count)
}
