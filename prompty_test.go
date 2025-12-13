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

// ============================================================================
// Conditional Tests (prompty.if / prompty.elseif / prompty.else)
// ============================================================================

func TestE2E_Conditional_BasicIf_True(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.if eval="isActive"~}Active{~/prompty.if~}`,
		map[string]any{"isActive": true},
	)

	require.NoError(t, err)
	assert.Equal(t, "Active", result)
}

func TestE2E_Conditional_BasicIf_False(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.if eval="isActive"~}Active{~/prompty.if~}`,
		map[string]any{"isActive": false},
	)

	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestE2E_Conditional_IfElse_True(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.if eval="isAdmin"~}Admin{~prompty.else~}User{~/prompty.if~}`,
		map[string]any{"isAdmin": true},
	)

	require.NoError(t, err)
	assert.Equal(t, "Admin", result)
}

func TestE2E_Conditional_IfElse_False(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.if eval="isAdmin"~}Admin{~prompty.else~}User{~/prompty.if~}`,
		map[string]any{"isAdmin": false},
	)

	require.NoError(t, err)
	assert.Equal(t, "User", result)
}

func TestE2E_Conditional_IfElseIfElse(t *testing.T) {
	engine := prompty.MustNew()

	template := `{~prompty.if eval="role == \"admin\""~}Admin{~prompty.elseif eval="role == \"editor\""~}Editor{~prompty.else~}Viewer{~/prompty.if~}`

	tests := []struct {
		name     string
		role     string
		expected string
	}{
		{"admin", "admin", "Admin"},
		{"editor", "editor", "Editor"},
		{"viewer", "viewer", "Viewer"},
		{"other", "other", "Viewer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Execute(context.Background(), template,
				map[string]any{"role": tt.role},
			)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestE2E_Conditional_MultipleElseIf(t *testing.T) {
	engine := prompty.MustNew()

	template := `{~prompty.if eval="level == 1"~}One{~prompty.elseif eval="level == 2"~}Two{~prompty.elseif eval="level == 3"~}Three{~prompty.else~}Other{~/prompty.if~}`

	tests := []struct {
		level    int
		expected string
	}{
		{1, "One"},
		{2, "Two"},
		{3, "Three"},
		{4, "Other"},
		{0, "Other"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result, err := engine.Execute(context.Background(), template,
				map[string]any{"level": tt.level},
			)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestE2E_Conditional_WithComparison(t *testing.T) {
	engine := prompty.MustNew()

	template := `{~prompty.if eval="count > 0"~}Items: {~prompty.var name="count" /~}{~prompty.else~}No items{~/prompty.if~}`

	t.Run("positive count", func(t *testing.T) {
		result, err := engine.Execute(context.Background(), template,
			map[string]any{"count": 5},
		)
		require.NoError(t, err)
		assert.Equal(t, "Items: 5", result)
	})

	t.Run("zero count", func(t *testing.T) {
		result, err := engine.Execute(context.Background(), template,
			map[string]any{"count": 0},
		)
		require.NoError(t, err)
		assert.Equal(t, "No items", result)
	})
}

func TestE2E_Conditional_WithFunctionCall(t *testing.T) {
	engine := prompty.MustNew()

	template := `{~prompty.if eval="len(items) > 0"~}Has items{~prompty.else~}Empty{~/prompty.if~}`

	t.Run("non-empty array", func(t *testing.T) {
		result, err := engine.Execute(context.Background(), template,
			map[string]any{"items": []any{1, 2, 3}},
		)
		require.NoError(t, err)
		assert.Equal(t, "Has items", result)
	})

	t.Run("empty array", func(t *testing.T) {
		result, err := engine.Execute(context.Background(), template,
			map[string]any{"items": []any{}},
		)
		require.NoError(t, err)
		assert.Equal(t, "Empty", result)
	})
}

func TestE2E_Conditional_WithContainsFunction(t *testing.T) {
	engine := prompty.MustNew()

	template := `{~prompty.if eval="contains(roles, \"admin\")"~}Welcome Admin{~prompty.else~}Welcome User{~/prompty.if~}`

	t.Run("has admin role", func(t *testing.T) {
		result, err := engine.Execute(context.Background(), template,
			map[string]any{"roles": []any{"user", "admin"}},
		)
		require.NoError(t, err)
		assert.Equal(t, "Welcome Admin", result)
	})

	t.Run("no admin role", func(t *testing.T) {
		result, err := engine.Execute(context.Background(), template,
			map[string]any{"roles": []any{"user", "viewer"}},
		)
		require.NoError(t, err)
		assert.Equal(t, "Welcome User", result)
	})
}

func TestE2E_Conditional_LogicalAnd(t *testing.T) {
	engine := prompty.MustNew()

	template := `{~prompty.if eval="isLoggedIn && isVerified"~}Access granted{~prompty.else~}Access denied{~/prompty.if~}`

	tests := []struct {
		name       string
		loggedIn   bool
		verified   bool
		expected   string
	}{
		{"both true", true, true, "Access granted"},
		{"logged in only", true, false, "Access denied"},
		{"verified only", false, true, "Access denied"},
		{"neither", false, false, "Access denied"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Execute(context.Background(), template,
				map[string]any{"isLoggedIn": tt.loggedIn, "isVerified": tt.verified},
			)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestE2E_Conditional_LogicalOr(t *testing.T) {
	engine := prompty.MustNew()

	template := `{~prompty.if eval="isAdmin || isModerator"~}Has privileges{~prompty.else~}No privileges{~/prompty.if~}`

	tests := []struct {
		name      string
		admin     bool
		moderator bool
		expected  string
	}{
		{"both", true, true, "Has privileges"},
		{"admin only", true, false, "Has privileges"},
		{"moderator only", false, true, "Has privileges"},
		{"neither", false, false, "No privileges"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Execute(context.Background(), template,
				map[string]any{"isAdmin": tt.admin, "isModerator": tt.moderator},
			)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestE2E_Conditional_Negation(t *testing.T) {
	engine := prompty.MustNew()

	template := `{~prompty.if eval="!isDisabled"~}Enabled{~prompty.else~}Disabled{~/prompty.if~}`

	t.Run("not disabled", func(t *testing.T) {
		result, err := engine.Execute(context.Background(), template,
			map[string]any{"isDisabled": false},
		)
		require.NoError(t, err)
		assert.Equal(t, "Enabled", result)
	})

	t.Run("disabled", func(t *testing.T) {
		result, err := engine.Execute(context.Background(), template,
			map[string]any{"isDisabled": true},
		)
		require.NoError(t, err)
		assert.Equal(t, "Disabled", result)
	})
}

func TestE2E_Conditional_NestedVariables(t *testing.T) {
	engine := prompty.MustNew()

	template := `{~prompty.if eval="user.isActive"~}Hello, {~prompty.var name="user.name" /~}!{~prompty.else~}User inactive{~/prompty.if~}`

	t.Run("active user", func(t *testing.T) {
		result, err := engine.Execute(context.Background(), template,
			map[string]any{
				"user": map[string]any{
					"name":     "Alice",
					"isActive": true,
				},
			},
		)
		require.NoError(t, err)
		assert.Equal(t, "Hello, Alice!", result)
	})

	t.Run("inactive user", func(t *testing.T) {
		result, err := engine.Execute(context.Background(), template,
			map[string]any{
				"user": map[string]any{
					"name":     "Bob",
					"isActive": false,
				},
			},
		)
		require.NoError(t, err)
		assert.Equal(t, "User inactive", result)
	})
}

func TestE2E_Conditional_NestedConditionals(t *testing.T) {
	engine := prompty.MustNew()

	template := `{~prompty.if eval="hasAccess"~}{~prompty.if eval="isAdmin"~}Admin Panel{~prompty.else~}User Panel{~/prompty.if~}{~prompty.else~}No Access{~/prompty.if~}`

	tests := []struct {
		name      string
		hasAccess bool
		isAdmin   bool
		expected  string
	}{
		{"admin with access", true, true, "Admin Panel"},
		{"user with access", true, false, "User Panel"},
		{"no access", false, false, "No Access"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Execute(context.Background(), template,
				map[string]any{"hasAccess": tt.hasAccess, "isAdmin": tt.isAdmin},
			)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestE2E_Conditional_WithSurroundingText(t *testing.T) {
	engine := prompty.MustNew()

	template := `Start - {~prompty.if eval="show"~}Middle{~/prompty.if~} - End`

	t.Run("condition true", func(t *testing.T) {
		result, err := engine.Execute(context.Background(), template,
			map[string]any{"show": true},
		)
		require.NoError(t, err)
		assert.Equal(t, "Start - Middle - End", result)
	})

	t.Run("condition false", func(t *testing.T) {
		result, err := engine.Execute(context.Background(), template,
			map[string]any{"show": false},
		)
		require.NoError(t, err)
		assert.Equal(t, "Start -  - End", result)
	})
}

func TestE2E_Conditional_StringComparison(t *testing.T) {
	engine := prompty.MustNew()

	template := `{~prompty.if eval="status == \"active\""~}Active{~prompty.elseif eval="status == \"pending\""~}Pending{~prompty.else~}Unknown{~/prompty.if~}`

	tests := []struct {
		status   string
		expected string
	}{
		{"active", "Active"},
		{"pending", "Pending"},
		{"inactive", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result, err := engine.Execute(context.Background(), template,
				map[string]any{"status": tt.status},
			)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestE2E_Conditional_NilCheck(t *testing.T) {
	engine := prompty.MustNew()

	template := `{~prompty.if eval="value != nil"~}Value: {~prompty.var name="value" /~}{~prompty.else~}No value{~/prompty.if~}`

	t.Run("has value", func(t *testing.T) {
		result, err := engine.Execute(context.Background(), template,
			map[string]any{"value": "hello"},
		)
		require.NoError(t, err)
		assert.Equal(t, "Value: hello", result)
	})

	t.Run("nil value", func(t *testing.T) {
		result, err := engine.Execute(context.Background(), template,
			map[string]any{"value": nil},
		)
		require.NoError(t, err)
		assert.Equal(t, "No value", result)
	})
}

func TestE2E_Conditional_Truthiness(t *testing.T) {
	engine := prompty.MustNew()

	template := `{~prompty.if eval="value"~}Truthy{~prompty.else~}Falsy{~/prompty.if~}`

	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{"non-empty string", "hello", "Truthy"},
		{"empty string", "", "Falsy"},
		{"positive number", 1, "Truthy"},
		{"zero", 0, "Falsy"},
		{"true", true, "Truthy"},
		{"false", false, "Falsy"},
		{"nil", nil, "Falsy"},
		{"non-empty array", []any{1}, "Truthy"},
		{"empty array", []any{}, "Falsy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Execute(context.Background(), template,
				map[string]any{"value": tt.value},
			)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestE2E_Conditional_ComplexExpression(t *testing.T) {
	engine := prompty.MustNew()

	template := `{~prompty.if eval="(isAdmin || isModerator) && isActive"~}Authorized{~prompty.else~}Unauthorized{~/prompty.if~}`

	tests := []struct {
		name      string
		admin     bool
		moderator bool
		active    bool
		expected  string
	}{
		{"active admin", true, false, true, "Authorized"},
		{"active moderator", false, true, true, "Authorized"},
		{"inactive admin", true, false, false, "Unauthorized"},
		{"active user", false, false, true, "Unauthorized"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Execute(context.Background(), template,
				map[string]any{"isAdmin": tt.admin, "isModerator": tt.moderator, "isActive": tt.active},
			)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestE2E_Conditional_Error_MissingEvalAttribute(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~prompty.if~}Content{~/prompty.if~}`,
		map[string]any{},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "eval")
}

func TestE2E_Conditional_Error_ElseWithEval(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~prompty.if eval="true"~}A{~prompty.else eval="false"~}B{~/prompty.if~}`,
		map[string]any{},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "else")
}

func TestE2E_Conditional_Error_InvalidExpression(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~prompty.if eval="@@invalid"~}Content{~/prompty.if~}`,
		map[string]any{},
	)

	require.Error(t, err)
}

func TestE2E_Conditional_WithIncludedTemplate(t *testing.T) {
	engine := prompty.MustNew()

	engine.MustRegisterTemplate("greeting", "Hello, {~prompty.var name=\"name\" default=\"Guest\" /~}!")

	template := `{~prompty.if eval="showGreeting"~}{~prompty.include template="greeting" name="Alice" /~}{~prompty.else~}No greeting{~/prompty.if~}`

	t.Run("show greeting", func(t *testing.T) {
		result, err := engine.Execute(context.Background(), template,
			map[string]any{"showGreeting": true},
		)
		require.NoError(t, err)
		assert.Equal(t, "Hello, Alice!", result)
	})

	t.Run("hide greeting", func(t *testing.T) {
		result, err := engine.Execute(context.Background(), template,
			map[string]any{"showGreeting": false},
		)
		require.NoError(t, err)
		assert.Equal(t, "No greeting", result)
	})
}

func TestE2E_Conditional_ParseOnceExecuteMany(t *testing.T) {
	engine := prompty.MustNew()

	tmpl, err := engine.Parse(`{~prompty.if eval="show"~}Visible{~prompty.else~}Hidden{~/prompty.if~}`)
	require.NoError(t, err)

	// Execute multiple times with different data
	result1, err := tmpl.Execute(context.Background(), map[string]any{"show": true})
	require.NoError(t, err)
	assert.Equal(t, "Visible", result1)

	result2, err := tmpl.Execute(context.Background(), map[string]any{"show": false})
	require.NoError(t, err)
	assert.Equal(t, "Hidden", result2)
}
