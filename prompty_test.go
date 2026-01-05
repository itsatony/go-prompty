package prompty_test

import (
	"context"
	"errors"
	"strings"
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
	child := parent.Child(map[string]any{"key": "childValue"}).(*prompty.Context)

	result, err := tmpl.ExecuteWithContext(context.Background(), child)
	require.NoError(t, err)
	assert.Equal(t, "childValue", result)
}

func TestE2E_ContextParentFallback(t *testing.T) {
	engine := prompty.MustNew()

	tmpl, err := engine.Parse("{~prompty.var name=\"parentKey\" /~}")
	require.NoError(t, err)

	parent := prompty.NewContext(map[string]any{"parentKey": "fromParent"})
	child := parent.Child(map[string]any{}).(*prompty.Context)

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

// =============================================================================
// Phase 3 Tests: Error Strategies
// =============================================================================

func TestE2E_ErrorStrategy_Throw(t *testing.T) {
	// Default behavior - errors are thrown
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		"Hello, {~prompty.var name=\"missing\" /~}!",
		nil,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

func TestE2E_ErrorStrategy_Default_GlobalStrategy(t *testing.T) {
	// Using global ErrorStrategyDefault returns default attribute value
	engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyDefault))

	result, err := engine.Execute(context.Background(),
		"Hello, {~prompty.var name=\"missing\" default=\"Guest\" /~}!",
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello, Guest!", result)
}

func TestE2E_ErrorStrategy_Default_NoDefaultAttr(t *testing.T) {
	// ErrorStrategyDefault without a default attribute returns empty string
	engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyDefault))

	result, err := engine.Execute(context.Background(),
		"Hello, {~prompty.var name=\"missing\" /~}!",
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello, !", result)
}

func TestE2E_ErrorStrategy_Remove(t *testing.T) {
	// ErrorStrategyRemove removes the tag entirely
	engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyRemove))

	result, err := engine.Execute(context.Background(),
		"Hello, {~prompty.var name=\"missing\" /~}World!",
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", result)
}

func TestE2E_ErrorStrategy_KeepRaw(t *testing.T) {
	// ErrorStrategyKeepRaw keeps the original tag text
	engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyKeepRaw))

	result, err := engine.Execute(context.Background(),
		"Hello, {~prompty.var name=\"missing\" /~}!",
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello, {~prompty.var name=\"missing\" /~}!", result)
}

func TestE2E_ErrorStrategy_Log(t *testing.T) {
	// ErrorStrategyLog logs the error and continues with empty string
	engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyLog))

	result, err := engine.Execute(context.Background(),
		"Hello, {~prompty.var name=\"missing\" /~}World!",
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", result)
}

func TestE2E_ErrorStrategy_PerTagOverride(t *testing.T) {
	// Per-tag onerror attribute overrides global strategy
	engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyThrow))

	// Global strategy is throw, but tag specifies keepraw
	result, err := engine.Execute(context.Background(),
		"Hello, {~prompty.var name=\"missing\" onerror=\"keepraw\" /~}!",
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello, {~prompty.var name=\"missing\" onerror=\"keepraw\" /~}!", result)
}

func TestE2E_ErrorStrategy_PerTagRemove(t *testing.T) {
	// Per-tag onerror="remove" removes the tag
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		"Hello, {~prompty.var name=\"missing\" onerror=\"remove\" /~}World!",
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", result)
}

func TestE2E_ErrorStrategy_PerTagDefault(t *testing.T) {
	// Per-tag onerror="default" uses default attribute
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		"Hello, {~prompty.var name=\"missing\" default=\"Friend\" onerror=\"default\" /~}!",
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello, Friend!", result)
}

func TestE2E_ErrorStrategy_UnknownTag_WithStrategy(t *testing.T) {
	// Error strategies also apply to unknown tags
	engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyRemove))

	result, err := engine.Execute(context.Background(),
		"Hello, {~unknown.tag /~}World!",
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", result)
}

func TestE2E_ErrorStrategy_UnknownTag_KeepRaw(t *testing.T) {
	// KeepRaw strategy with unknown tag preserves the original
	engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyKeepRaw))

	result, err := engine.Execute(context.Background(),
		"Hello, {~unknown.tag attr=\"value\" /~}!",
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello, {~unknown.tag attr=\"value\" /~}!", result)
}

func TestE2E_ErrorStrategy_MixedTags(t *testing.T) {
	// Some tags succeed, some fail with error strategy
	engine := prompty.MustNew(prompty.WithErrorStrategy(prompty.ErrorStrategyRemove))

	result, err := engine.Execute(context.Background(),
		"Hello, {~prompty.var name=\"user\" /~}! {~prompty.var name=\"missing\" /~}",
		map[string]any{"user": "Alice"},
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello, Alice! ", result)
}

// =============================================================================
// Phase 3 Tests: Comment Tags
// =============================================================================

func TestE2E_Comment_BasicComment(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		"Hello{~prompty.comment~}This is a comment{~/prompty.comment~}World",
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "HelloWorld", result)
}

func TestE2E_Comment_EmptyComment(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		"Hello{~prompty.comment~}{~/prompty.comment~}World",
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "HelloWorld", result)
}

func TestE2E_Comment_MultilineComment(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`Hello{~prompty.comment~}
This is a
multiline
comment
{~/prompty.comment~}World`,
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "HelloWorld", result)
}

func TestE2E_Comment_WithVariables(t *testing.T) {
	// Comments should not affect variable resolution
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		"Hello, {~prompty.var name=\"user\" /~}!{~prompty.comment~}DEBUG: user={~prompty.var name=\"user\" /~}{~/prompty.comment~}",
		map[string]any{"user": "Alice"},
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello, Alice!", result)
}

func TestE2E_Comment_MultipleComments(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		"{~prompty.comment~}Start{~/prompty.comment~}Hello{~prompty.comment~}Middle{~/prompty.comment~}World{~prompty.comment~}End{~/prompty.comment~}",
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "HelloWorld", result)
}

func TestE2E_Comment_ContainsPromptyTags(t *testing.T) {
	// Tags inside comments should be stripped, not executed
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		"Hello{~prompty.comment~}{~prompty.var name=\"missing\" /~}{~/prompty.comment~}World",
		nil,
	)

	require.NoError(t, err)
	// The comment is removed entirely, including any tags inside
	assert.Equal(t, "HelloWorld", result)
}

func TestE2E_Comment_PreservesWhitespace(t *testing.T) {
	// Whitespace outside comments should be preserved
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		"Hello {~prompty.comment~}comment{~/prompty.comment~} World",
		nil,
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello  World", result)
}

// =============================================================================
// Phase 3 Tests: Error Strategy Constants
// =============================================================================

func TestErrorStrategy_String(t *testing.T) {
	tests := []struct {
		strategy prompty.ErrorStrategy
		expected string
	}{
		{prompty.ErrorStrategyThrow, "throw"},
		{prompty.ErrorStrategyDefault, "default"},
		{prompty.ErrorStrategyRemove, "remove"},
		{prompty.ErrorStrategyKeepRaw, "keepraw"},
		{prompty.ErrorStrategyLog, "log"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.strategy.String())
		})
	}
}

func TestParseErrorStrategy(t *testing.T) {
	tests := []struct {
		input    string
		expected prompty.ErrorStrategy
	}{
		{"throw", prompty.ErrorStrategyThrow},
		{"default", prompty.ErrorStrategyDefault},
		{"remove", prompty.ErrorStrategyRemove},
		{"keepraw", prompty.ErrorStrategyKeepRaw},
		{"log", prompty.ErrorStrategyLog},
		{"unknown", prompty.ErrorStrategyThrow}, // Unknown defaults to throw
		{"", prompty.ErrorStrategyThrow},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := prompty.ParseErrorStrategy(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidErrorStrategy(t *testing.T) {
	validStrategies := []string{"throw", "default", "remove", "keepraw", "log"}
	invalidStrategies := []string{"unknown", "", "THROW", "Throw", "invalid"}

	for _, s := range validStrategies {
		t.Run(s+"_valid", func(t *testing.T) {
			assert.True(t, prompty.IsValidErrorStrategy(s))
		})
	}

	for _, s := range invalidStrategies {
		t.Run(s+"_invalid", func(t *testing.T) {
			assert.False(t, prompty.IsValidErrorStrategy(s))
		})
	}
}

// =============================================================================
// Phase 3 Tests: Validation API
// =============================================================================

func TestE2E_Validation_ValidTemplate(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Validate("Hello, {~prompty.var name=\"user\" /~}!")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsValid())
	assert.False(t, result.HasErrors())
	assert.False(t, result.HasWarnings())
	assert.Empty(t, result.Issues())
}

func TestE2E_Validation_PlainText(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Validate("Just plain text, no tags")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsValid())
	assert.Empty(t, result.Issues())
}

func TestE2E_Validation_UnknownTag(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Validate("Hello, {~unknown.tag /~}!")

	require.NoError(t, err)
	require.NotNil(t, result)
	// Unknown tag should be a warning, not an error
	assert.True(t, result.IsValid()) // Still valid (just warning)
	assert.False(t, result.HasErrors())
	assert.True(t, result.HasWarnings())
	assert.Len(t, result.Warnings(), 1)
	assert.Contains(t, result.Warnings()[0].Message, "unknown tag")
}

func TestE2E_Validation_InvalidOnErrorAttribute(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Validate(`{~prompty.var name="user" onerror="invalid_value" /~}`)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsValid())
	assert.True(t, result.HasErrors())
	assert.Len(t, result.Errors(), 1)
	assert.Contains(t, result.Errors()[0].Message, "onerror")
}

func TestE2E_Validation_MissingRequiredAttribute(t *testing.T) {
	engine := prompty.MustNew()

	// prompty.var requires a 'name' attribute
	result, err := engine.Validate(`{~prompty.var /~}`)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsValid())
	assert.True(t, result.HasErrors())
	assert.Len(t, result.Errors(), 1)
	assert.Contains(t, result.Errors()[0].Message, "name")
}

func TestE2E_Validation_ParseError(t *testing.T) {
	engine := prompty.MustNew()

	// Unclosed tag should cause a parse error
	result, err := engine.Validate(`{~prompty.var name="user"`)

	require.NoError(t, err) // Validate returns result, not error
	require.NotNil(t, result)
	assert.False(t, result.IsValid())
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors()[0].Message, "parsing")
}

func TestE2E_Validation_MissingIncludeTarget(t *testing.T) {
	engine := prompty.MustNew()

	// Include a template that isn't registered
	result, err := engine.Validate(`{~prompty.include template="nonexistent" /~}`)

	require.NoError(t, err)
	require.NotNil(t, result)
	// Missing template should be a warning (template might be registered later)
	assert.True(t, result.IsValid())
	assert.True(t, result.HasWarnings())
	assert.Len(t, result.Warnings(), 1)
	assert.Contains(t, result.Warnings()[0].Message, "not found")
}

func TestE2E_Validation_RegisteredIncludeTarget(t *testing.T) {
	engine := prompty.MustNew()
	engine.MustRegisterTemplate("footer", "Copyright 2024")

	result, err := engine.Validate(`{~prompty.include template="footer" /~}`)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsValid())
	assert.Empty(t, result.Issues())
}

func TestE2E_Validation_MultipleIssues(t *testing.T) {
	engine := prompty.MustNew()

	// Multiple issues: unknown tag + invalid onerror
	result, err := engine.Validate(`
		{~unknown.tag /~}
		{~prompty.var onerror="bad" /~}
	`)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsValid())
	assert.True(t, result.HasErrors())
	assert.True(t, result.HasWarnings())
	// 1 warning (unknown tag) + 2 errors (invalid onerror + missing name)
	assert.GreaterOrEqual(t, len(result.Issues()), 2)
}

func TestE2E_Validation_NestedConditional(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Validate(`
		{~prompty.if eval="show"~}
			{~prompty.var name="message" /~}
		{~prompty.else~}
			{~prompty.var name="fallback" /~}
		{~/prompty.if~}
	`)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsValid())
	assert.Empty(t, result.Issues())
}

func TestE2E_Validation_CommentBlock(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Validate(`Hello{~prompty.comment~}This is a comment{~/prompty.comment~}World`)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsValid())
	assert.Empty(t, result.Issues())
}

func TestValidationResult_IssueFiltering(t *testing.T) {
	engine := prompty.MustNew()

	// Create a template with both errors and warnings
	result, err := engine.Validate(`
		{~unknown.tag /~}
		{~prompty.var onerror="invalid" /~}
	`)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Test filtering
	allIssues := result.Issues()
	errorsOnly := result.Errors()
	warningsOnly := result.Warnings()

	assert.GreaterOrEqual(t, len(allIssues), 2)
	assert.GreaterOrEqual(t, len(errorsOnly), 1)
	assert.GreaterOrEqual(t, len(warningsOnly), 1)
	assert.Equal(t, len(allIssues), len(errorsOnly)+len(warningsOnly))
}

// =============================================================================
// Phase 4 Tests: For Loops
// =============================================================================

func TestE2E_ForLoop_BasicStringSlice(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="x" in="items"~}{~prompty.var name="x" /~},{~/prompty.for~}`,
		map[string]any{"items": []string{"a", "b", "c"}},
	)

	require.NoError(t, err)
	assert.Equal(t, "a,b,c,", result)
}

func TestE2E_ForLoop_WithIndex(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="x" index="i" in="items"~}[{~prompty.var name="i" /~}:{~prompty.var name="x" /~}]{~/prompty.for~}`,
		map[string]any{"items": []string{"a", "b", "c"}},
	)

	require.NoError(t, err)
	assert.Equal(t, "[0:a][1:b][2:c]", result)
}

func TestE2E_ForLoop_IntSlice(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="n" in="numbers"~}{~prompty.var name="n" /~} {~/prompty.for~}`,
		map[string]any{"numbers": []int{1, 2, 3, 4, 5}},
	)

	require.NoError(t, err)
	assert.Equal(t, "1 2 3 4 5 ", result)
}

func TestE2E_ForLoop_AnySlice(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="x" in="items"~}{~prompty.var name="x" /~}|{~/prompty.for~}`,
		map[string]any{"items": []any{"hello", 42, true}},
	)

	require.NoError(t, err)
	assert.Equal(t, "hello|42|true|", result)
}

func TestE2E_ForLoop_MapSlice(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="user" in="users"~}{~prompty.var name="user.name" /~}({~prompty.var name="user.age" /~}) {~/prompty.for~}`,
		map[string]any{
			"users": []map[string]any{
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 25},
			},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "Alice(30) Bob(25) ", result)
}

func TestE2E_ForLoop_EmptySlice(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`Before{~prompty.for item="x" in="items"~}{~prompty.var name="x" /~}{~/prompty.for~}After`,
		map[string]any{"items": []any{}},
	)

	require.NoError(t, err)
	assert.Equal(t, "BeforeAfter", result)
}

func TestE2E_ForLoop_NilCollection(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="x" in="items"~}{~prompty.var name="x" /~}{~/prompty.for~}`,
		map[string]any{"items": nil},
	)

	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestE2E_ForLoop_SingleItem(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="x" in="items"~}Item: {~prompty.var name="x" /~}{~/prompty.for~}`,
		map[string]any{"items": []string{"only"}},
	)

	require.NoError(t, err)
	assert.Equal(t, "Item: only", result)
}

func TestE2E_ForLoop_NestedPath(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="x" in="data.items"~}{~prompty.var name="x" /~},{~/prompty.for~}`,
		map[string]any{
			"data": map[string]any{
				"items": []string{"a", "b", "c"},
			},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "a,b,c,", result)
}

func TestE2E_ForLoop_WithLimit(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="x" in="items" limit="3"~}{~prompty.var name="x" /~},{~/prompty.for~}`,
		map[string]any{"items": []string{"a", "b", "c", "d", "e"}},
	)

	require.NoError(t, err)
	assert.Equal(t, "a,b,c,", result)
}

func TestE2E_ForLoop_LimitLargerThanCollection(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="x" in="items" limit="100"~}{~prompty.var name="x" /~},{~/prompty.for~}`,
		map[string]any{"items": []string{"a", "b"}},
	)

	require.NoError(t, err)
	assert.Equal(t, "a,b,", result)
}

func TestE2E_ForLoop_IterateOverMap(t *testing.T) {
	engine := prompty.MustNew()

	// Maps iterate over key-value pairs (sorted by key)
	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="entry" in="config"~}{~prompty.var name="entry.key" /~}={~prompty.var name="entry.value" /~};{~/prompty.for~}`,
		map[string]any{
			"config": map[string]any{
				"a": "1",
				"b": "2",
				"c": "3",
			},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "a=1;b=2;c=3;", result)
}

func TestE2E_ForLoop_NestedLoops(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="row" in="matrix"~}[{~prompty.for item="col" in="row"~}{~prompty.var name="col" /~}{~/prompty.for~}]{~/prompty.for~}`,
		map[string]any{
			"matrix": []any{
				[]any{1, 2},
				[]any{3, 4},
			},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "[12][34]", result)
}

func TestE2E_ForLoop_WithConditional(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="n" in="numbers"~}{~prompty.if eval="n > 2"~}*{~/prompty.if~}{~prompty.var name="n" /~} {~/prompty.for~}`,
		map[string]any{"numbers": []int{1, 2, 3, 4}},
	)

	require.NoError(t, err)
	assert.Equal(t, "1 2 *3 *4 ", result)
}

func TestE2E_ForLoop_WithInclude(t *testing.T) {
	engine := prompty.MustNew()
	// Test combining for loops with includes using explicit attributes to pass data
	engine.MustRegisterTemplate("item-display", "Item: {~prompty.var name=\"val\" /~}")

	// Pass the item value explicitly to the included template via attribute
	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="x" in="items"~}[{~prompty.var name="x" /~}] {~/prompty.for~}Included: {~prompty.include template="item-display" val="static" /~}`,
		map[string]any{
			"items": []string{"a", "b", "c"},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "[a] [b] [c] Included: Item: static", result)
}

func TestE2E_ForLoop_PreservesParentContext(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`Prefix: {~prompty.var name="prefix" /~} - {~prompty.for item="x" in="items"~}{~prompty.var name="prefix" /~}:{~prompty.var name="x" /~} {~/prompty.for~}`,
		map[string]any{
			"prefix": "P",
			"items":  []string{"a", "b"},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "Prefix: P - P:a P:b ", result)
}

func TestE2E_ForLoop_ItemShadowsParent(t *testing.T) {
	engine := prompty.MustNew()

	// If the loop variable has the same name as a parent variable, it shadows it
	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="x" in="items"~}{~prompty.var name="x" /~}{~/prompty.for~}`,
		map[string]any{
			"x":     "parent-value",
			"items": []string{"a", "b"},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "ab", result)
}

func TestE2E_ForLoop_Float64Slice(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="x" in="numbers"~}{~prompty.var name="x" /~},{~/prompty.for~}`,
		map[string]any{"numbers": []float64{1.5, 2.5, 3.5}},
	)

	require.NoError(t, err)
	assert.Equal(t, "1.5,2.5,3.5,", result)
}

func TestE2E_ForLoop_BoolSlice(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="x" in="flags"~}{~prompty.var name="x" /~} {~/prompty.for~}`,
		map[string]any{"flags": []bool{true, false, true}},
	)

	require.NoError(t, err)
	assert.Equal(t, "true false true ", result)
}

func TestE2E_ForLoop_Int64Slice(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="x" in="ids"~}{~prompty.var name="x" /~},{~/prompty.for~}`,
		map[string]any{"ids": []int64{100, 200, 300}},
	)

	require.NoError(t, err)
	assert.Equal(t, "100,200,300,", result)
}

func TestE2E_ForLoop_ParseOnceExecuteMany(t *testing.T) {
	engine := prompty.MustNew()

	tmpl, err := engine.Parse(`{~prompty.for item="x" in="items"~}{~prompty.var name="x" /~}{~/prompty.for~}`)
	require.NoError(t, err)

	result1, err := tmpl.Execute(context.Background(), map[string]any{"items": []string{"a", "b"}})
	require.NoError(t, err)
	assert.Equal(t, "ab", result1)

	result2, err := tmpl.Execute(context.Background(), map[string]any{"items": []string{"x", "y", "z"}})
	require.NoError(t, err)
	assert.Equal(t, "xyz", result2)
}

func TestE2E_ForLoop_Error_MissingItemAttr(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~prompty.for in="items"~}content{~/prompty.for~}`,
		map[string]any{"items": []string{"a"}},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "item")
}

func TestE2E_ForLoop_Error_MissingInAttr(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~prompty.for item="x"~}content{~/prompty.for~}`,
		map[string]any{"items": []string{"a"}},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "in")
}

func TestE2E_ForLoop_Error_CollectionNotFound(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~prompty.for item="x" in="missing"~}content{~/prompty.for~}`,
		map[string]any{},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestE2E_ForLoop_Error_NotIterable(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~prompty.for item="x" in="value"~}content{~/prompty.for~}`,
		map[string]any{"value": "not-a-slice"},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "iterable")
}

func TestE2E_ForLoop_Error_InvalidLimit(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~prompty.for item="x" in="items" limit="abc"~}content{~/prompty.for~}`,
		map[string]any{"items": []string{"a"}},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "limit")
}

func TestE2E_ForLoop_Error_NegativeLimit(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~prompty.for item="x" in="items" limit="-1"~}content{~/prompty.for~}`,
		map[string]any{"items": []string{"a"}},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "limit")
}

func TestE2E_ForLoop_Error_UnclosedBlock(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~prompty.for item="x" in="items"~}content`,
		map[string]any{"items": []string{"a"}},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not closed")
}

func TestE2E_ForLoop_Validation_Valid(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Validate(`{~prompty.for item="x" in="items"~}{~prompty.var name="x" /~}{~/prompty.for~}`)

	require.NoError(t, err)
	assert.True(t, result.IsValid())
}

func TestE2E_ForLoop_Validation_NestedContent(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Validate(`{~prompty.for item="x" in="items"~}
		{~prompty.if eval="x > 0"~}
			Positive: {~prompty.var name="x" /~}
		{~/prompty.if~}
	{~/prompty.for~}`)

	require.NoError(t, err)
	assert.True(t, result.IsValid())
}

func TestE2E_ForLoop_ComplexTemplate(t *testing.T) {
	engine := prompty.MustNew()

	// Use template without extra newlines for predictable output
	template := `<ul>{~prompty.for item="user" index="i" in="users"~}<li>{~prompty.var name="i" /~}. {~prompty.var name="user.name" /~} - {~prompty.if eval="user.active"~}Active{~prompty.else~}Inactive{~/prompty.if~}</li>{~/prompty.for~}</ul>`

	result, err := engine.Execute(context.Background(), template, map[string]any{
		"users": []map[string]any{
			{"name": "Alice", "active": true},
			{"name": "Bob", "active": false},
			{"name": "Carol", "active": true},
		},
	})

	require.NoError(t, err)
	expected := `<ul><li>0. Alice - Active</li><li>1. Bob - Inactive</li><li>2. Carol - Active</li></ul>`
	assert.Equal(t, expected, result)
}

// ============================================================================
// Phase 5: Switch/Case Tests
// ============================================================================

func TestE2E_Switch_BasicStringValue(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="status"~}{~prompty.case value="active"~}Active User{~/prompty.case~}{~prompty.case value="inactive"~}Inactive User{~/prompty.case~}{~/prompty.switch~}`,
		map[string]any{"status": "active"},
	)

	require.NoError(t, err)
	assert.Equal(t, "Active User", result)
}

func TestE2E_Switch_SecondCaseMatch(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="status"~}{~prompty.case value="active"~}Active{~/prompty.case~}{~prompty.case value="inactive"~}Inactive{~/prompty.case~}{~/prompty.switch~}`,
		map[string]any{"status": "inactive"},
	)

	require.NoError(t, err)
	assert.Equal(t, "Inactive", result)
}

func TestE2E_Switch_WithDefault(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="status"~}{~prompty.case value="active"~}Active{~/prompty.case~}{~prompty.casedefault~}Unknown{~/prompty.casedefault~}{~/prompty.switch~}`,
		map[string]any{"status": "other"},
	)

	require.NoError(t, err)
	assert.Equal(t, "Unknown", result)
}

func TestE2E_Switch_DefaultNotUsedWhenMatch(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="status"~}{~prompty.case value="active"~}Active{~/prompty.case~}{~prompty.casedefault~}Default{~/prompty.casedefault~}{~/prompty.switch~}`,
		map[string]any{"status": "active"},
	)

	require.NoError(t, err)
	assert.Equal(t, "Active", result)
}

func TestE2E_Switch_NoMatchNoDefault(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`Before{~prompty.switch eval="status"~}{~prompty.case value="active"~}Active{~/prompty.case~}{~/prompty.switch~}After`,
		map[string]any{"status": "unknown"},
	)

	require.NoError(t, err)
	assert.Equal(t, "BeforeAfter", result)
}

func TestE2E_Switch_IntegerValue(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="level"~}{~prompty.case value="1"~}Low{~/prompty.case~}{~prompty.case value="2"~}Medium{~/prompty.case~}{~prompty.case value="3"~}High{~/prompty.case~}{~/prompty.switch~}`,
		map[string]any{"level": 2},
	)

	require.NoError(t, err)
	assert.Equal(t, "Medium", result)
}

func TestE2E_Switch_BooleanValue(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="enabled"~}{~prompty.case value="true"~}ON{~/prompty.case~}{~prompty.case value="false"~}OFF{~/prompty.case~}{~/prompty.switch~}`,
		map[string]any{"enabled": true},
	)

	require.NoError(t, err)
	assert.Equal(t, "ON", result)
}

func TestE2E_Switch_NestedPath(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="user.role"~}{~prompty.case value="admin"~}Admin View{~/prompty.case~}{~prompty.case value="user"~}User View{~/prompty.case~}{~/prompty.switch~}`,
		map[string]any{
			"user": map[string]any{"role": "admin"},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "Admin View", result)
}

func TestE2E_Switch_CaseWithEval(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="score"~}{~prompty.case eval="score >= 90"~}A{~/prompty.case~}{~prompty.case eval="score >= 80"~}B{~/prompty.case~}{~prompty.case eval="score >= 70"~}C{~/prompty.case~}{~prompty.casedefault~}F{~/prompty.casedefault~}{~/prompty.switch~}`,
		map[string]any{"score": 85},
	)

	require.NoError(t, err)
	assert.Equal(t, "B", result)
}

func TestE2E_Switch_CaseWithEvalFirstMatch(t *testing.T) {
	engine := prompty.MustNew()

	// First matching case wins (no fall-through)
	result, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="x"~}{~prompty.case eval="x > 0"~}Positive{~/prompty.case~}{~prompty.case eval="x > 5"~}Very Positive{~/prompty.case~}{~/prompty.switch~}`,
		map[string]any{"x": 10},
	)

	require.NoError(t, err)
	assert.Equal(t, "Positive", result)
}

func TestE2E_Switch_WithVariables(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="type"~}{~prompty.case value="greeting"~}Hello, {~prompty.var name="name" /~}!{~/prompty.case~}{~prompty.case value="farewell"~}Goodbye, {~prompty.var name="name" /~}!{~/prompty.case~}{~/prompty.switch~}`,
		map[string]any{"type": "greeting", "name": "Alice"},
	)

	require.NoError(t, err)
	assert.Equal(t, "Hello, Alice!", result)
}

func TestE2E_Switch_WithConditional(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="status"~}{~prompty.case value="active"~}{~prompty.if eval="premium"~}Premium Active{~prompty.else~}Basic Active{~/prompty.if~}{~/prompty.case~}{~/prompty.switch~}`,
		map[string]any{"status": "active", "premium": true},
	)

	require.NoError(t, err)
	assert.Equal(t, "Premium Active", result)
}

func TestE2E_Switch_MultipleCases(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="day"~}{~prompty.case value="mon"~}Monday{~/prompty.case~}{~prompty.case value="tue"~}Tuesday{~/prompty.case~}{~prompty.case value="wed"~}Wednesday{~/prompty.case~}{~prompty.case value="thu"~}Thursday{~/prompty.case~}{~prompty.case value="fri"~}Friday{~/prompty.case~}{~prompty.casedefault~}Weekend{~/prompty.casedefault~}{~/prompty.switch~}`,
		map[string]any{"day": "wed"},
	)

	require.NoError(t, err)
	assert.Equal(t, "Wednesday", result)
}

func TestE2E_Switch_WithForLoop(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.for item="n" in="numbers"~}{~prompty.switch eval="n"~}{~prompty.case value="1"~}one{~/prompty.case~}{~prompty.case value="2"~}two{~/prompty.case~}{~prompty.casedefault~}other{~/prompty.casedefault~}{~/prompty.switch~} {~/prompty.for~}`,
		map[string]any{"numbers": []int{1, 2, 3}},
	)

	require.NoError(t, err)
	assert.Equal(t, "one two other ", result)
}

func TestE2E_Switch_NestedSwitch(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="category"~}{~prompty.case value="color"~}{~prompty.switch eval="value"~}{~prompty.case value="red"~}Red Color{~/prompty.case~}{~prompty.case value="blue"~}Blue Color{~/prompty.case~}{~/prompty.switch~}{~/prompty.case~}{~prompty.casedefault~}Unknown{~/prompty.casedefault~}{~/prompty.switch~}`,
		map[string]any{"category": "color", "value": "blue"},
	)

	require.NoError(t, err)
	assert.Equal(t, "Blue Color", result)
}

func TestE2E_Switch_ExpressionEvaluation(t *testing.T) {
	engine := prompty.MustNew()

	// The switch expression can be any expression, not just a variable
	result, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="len(items)"~}{~prompty.case value="0"~}Empty{~/prompty.case~}{~prompty.case value="1"~}Single{~/prompty.case~}{~prompty.casedefault~}Multiple{~/prompty.casedefault~}{~/prompty.switch~}`,
		map[string]any{"items": []string{"a", "b", "c"}},
	)

	require.NoError(t, err)
	assert.Equal(t, "Multiple", result)
}

func TestE2E_Switch_ParseOnceExecuteMany(t *testing.T) {
	engine := prompty.MustNew()

	tmpl, err := engine.Parse(`{~prompty.switch eval="status"~}{~prompty.case value="a"~}A{~/prompty.case~}{~prompty.case value="b"~}B{~/prompty.case~}{~/prompty.switch~}`)
	require.NoError(t, err)

	result1, err := tmpl.Execute(context.Background(), map[string]any{"status": "a"})
	require.NoError(t, err)
	assert.Equal(t, "A", result1)

	result2, err := tmpl.Execute(context.Background(), map[string]any{"status": "b"})
	require.NoError(t, err)
	assert.Equal(t, "B", result2)
}

func TestE2E_Switch_Error_MissingEval(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~prompty.switch~}{~prompty.case value="x"~}X{~/prompty.case~}{~/prompty.switch~}`,
		map[string]any{},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "eval")
}

func TestE2E_Switch_Error_CaseMissingValueAndEval(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="status"~}{~prompty.case~}Content{~/prompty.case~}{~/prompty.switch~}`,
		map[string]any{"status": "x"},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "value")
}

func TestE2E_Switch_Error_UnclosedSwitch(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="status"~}{~prompty.case value="x"~}X{~/prompty.case~}`,
		map[string]any{"status": "x"},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not closed")
}

func TestE2E_Switch_Error_UnclosedCase(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="status"~}{~prompty.case value="x"~}X{~/prompty.switch~}`,
		map[string]any{"status": "x"},
	)

	require.Error(t, err)
}

func TestE2E_Switch_Error_DefaultNotLast(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="status"~}{~prompty.casedefault~}Default{~/prompty.casedefault~}{~prompty.case value="x"~}X{~/prompty.case~}{~/prompty.switch~}`,
		map[string]any{"status": "x"},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "last")
}

func TestE2E_Switch_Error_DuplicateDefault(t *testing.T) {
	engine := prompty.MustNew()

	_, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="status"~}{~prompty.casedefault~}Default1{~/prompty.casedefault~}{~prompty.casedefault~}Default2{~/prompty.casedefault~}{~/prompty.switch~}`,
		map[string]any{"status": "x"},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "one default")
}

func TestE2E_Switch_Validation_Valid(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Validate(`{~prompty.switch eval="status"~}{~prompty.case value="a"~}A{~/prompty.case~}{~prompty.casedefault~}Default{~/prompty.casedefault~}{~/prompty.switch~}`)

	require.NoError(t, err)
	assert.True(t, result.IsValid())
}

func TestE2E_Switch_Validation_NestedContent(t *testing.T) {
	engine := prompty.MustNew()

	result, err := engine.Validate(`{~prompty.switch eval="type"~}
		{~prompty.case value="user"~}
			{~prompty.var name="user.name" /~}
		{~/prompty.case~}
		{~prompty.casedefault~}
			Unknown
		{~/prompty.casedefault~}
	{~/prompty.switch~}`)

	require.NoError(t, err)
	assert.True(t, result.IsValid())
}

func TestE2E_Switch_ComplexTemplate(t *testing.T) {
	engine := prompty.MustNew()

	template := `User: {~prompty.var name="user.name" /~}
Status: {~prompty.switch eval="user.status"~}{~prompty.case value="active"~}Active{~/prompty.case~}{~prompty.case value="pending"~}Pending Approval{~/prompty.case~}{~prompty.case value="suspended"~}Suspended{~/prompty.case~}{~prompty.casedefault~}Unknown{~/prompty.casedefault~}{~/prompty.switch~}
Role: {~prompty.switch eval="user.role"~}{~prompty.case value="admin"~}Administrator{~/prompty.case~}{~prompty.case value="mod"~}Moderator{~/prompty.case~}{~prompty.casedefault~}Member{~/prompty.casedefault~}{~/prompty.switch~}`

	result, err := engine.Execute(context.Background(), template, map[string]any{
		"user": map[string]any{
			"name":   "Alice",
			"status": "active",
			"role":   "admin",
		},
	})

	require.NoError(t, err)
	expected := `User: Alice
Status: Active
Role: Administrator`
	assert.Equal(t, expected, result)
}

// ============================================================================
// Phase 5: Custom Function Tests
// ============================================================================

func TestE2E_CustomFunc_Register(t *testing.T) {
	engine := prompty.MustNew()

	// Register a simple double function
	err := engine.RegisterFunc(&prompty.Func{
		Name:    "double",
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			if n, ok := args[0].(int); ok {
				return n * 2, nil
			}
			if n, ok := args[0].(float64); ok {
				return n * 2, nil
			}
			return nil, errors.New("expected numeric argument")
		},
	})

	require.NoError(t, err)
	assert.True(t, engine.HasFunc("double"))
}

func TestE2E_CustomFunc_UseInConditional(t *testing.T) {
	engine := prompty.MustNew()

	// Register double function
	engine.MustRegisterFunc(&prompty.Func{
		Name:    "double",
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			if n, ok := args[0].(int); ok {
				return n * 2, nil
			}
			return nil, errors.New("expected int")
		},
	})

	result, err := engine.Execute(context.Background(),
		`{~prompty.if eval="double(x) > 10"~}Big{~prompty.else~}Small{~/prompty.if~}`,
		map[string]any{"x": 6},
	)

	require.NoError(t, err)
	assert.Equal(t, "Big", result)
}

func TestE2E_CustomFunc_UseInSwitch(t *testing.T) {
	engine := prompty.MustNew()

	// Register a grading function
	engine.MustRegisterFunc(&prompty.Func{
		Name:    "grade",
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			var score int
			switch v := args[0].(type) {
			case int:
				score = v
			case float64:
				score = int(v)
			default:
				return "F", nil
			}
			if score >= 90 {
				return "A", nil
			} else if score >= 80 {
				return "B", nil
			} else if score >= 70 {
				return "C", nil
			}
			return "F", nil
		},
	})

	result, err := engine.Execute(context.Background(),
		`{~prompty.switch eval="grade(score)"~}{~prompty.case value="A"~}Excellent{~/prompty.case~}{~prompty.case value="B"~}Good{~/prompty.case~}{~prompty.casedefault~}Needs Improvement{~/prompty.casedefault~}{~/prompty.switch~}`,
		map[string]any{"score": 85},
	)

	require.NoError(t, err)
	assert.Equal(t, "Good", result)
}

func TestE2E_CustomFunc_Variadic(t *testing.T) {
	engine := prompty.MustNew()

	// Register a sum function with variadic args
	engine.MustRegisterFunc(&prompty.Func{
		Name:    "sum",
		MinArgs: 1,
		MaxArgs: -1, // Variadic
		Fn: func(args []any) (any, error) {
			total := 0
			for _, arg := range args {
				if n, ok := arg.(int); ok {
					total += n
				}
			}
			return total, nil
		},
	})

	result, err := engine.Execute(context.Background(),
		`{~prompty.if eval="sum(a, b, c) > 10"~}Large{~prompty.else~}Small{~/prompty.if~}`,
		map[string]any{"a": 3, "b": 4, "c": 5},
	)

	require.NoError(t, err)
	assert.Equal(t, "Large", result)
}

func TestE2E_CustomFunc_ListFuncs(t *testing.T) {
	engine := prompty.MustNew()

	// Built-in functions should already be registered
	initialCount := engine.FuncCount()
	assert.Greater(t, initialCount, 0)

	// Register a custom function
	engine.MustRegisterFunc(&prompty.Func{
		Name:    "myCustomFunc",
		MinArgs: 0,
		MaxArgs: 0,
		Fn:      func(args []any) (any, error) { return "custom", nil },
	})

	// Count should increase
	assert.Equal(t, initialCount+1, engine.FuncCount())

	// Function should be in list
	funcs := engine.ListFuncs()
	found := false
	for _, name := range funcs {
		if name == "myCustomFunc" {
			found = true
			break
		}
	}
	assert.True(t, found, "custom function should be in list")
}

func TestE2E_CustomFunc_Error_NilFunc(t *testing.T) {
	engine := prompty.MustNew()

	err := engine.RegisterFunc(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestE2E_CustomFunc_Error_EmptyName(t *testing.T) {
	engine := prompty.MustNew()

	err := engine.RegisterFunc(&prompty.Func{
		Name:    "",
		MinArgs: 0,
		MaxArgs: 0,
		Fn:      func(args []any) (any, error) { return nil, nil },
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestE2E_CustomFunc_Error_Duplicate(t *testing.T) {
	engine := prompty.MustNew()

	f := &prompty.Func{
		Name:    "myFunc",
		MinArgs: 0,
		MaxArgs: 0,
		Fn:      func(args []any) (any, error) { return nil, nil },
	}

	// First registration should succeed
	err := engine.RegisterFunc(f)
	require.NoError(t, err)

	// Second registration with same name should fail
	err = engine.RegisterFunc(f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already")
}

func TestE2E_CustomFunc_CombinedWithBuiltin(t *testing.T) {
	engine := prompty.MustNew()

	// Register a custom function
	engine.MustRegisterFunc(&prompty.Func{
		Name:    "triple",
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			if n, ok := args[0].(int); ok {
				return n * 3, nil
			}
			return nil, errors.New("expected int")
		},
	})

	// Use both built-in (len) and custom (triple) in same expression
	result, err := engine.Execute(context.Background(),
		`{~prompty.if eval="len(items) > 0 && triple(x) > 10"~}Yes{~prompty.else~}No{~/prompty.if~}`,
		map[string]any{"items": []string{"a", "b"}, "x": 5},
	)

	require.NoError(t, err)
	assert.Equal(t, "Yes", result)
}

// ============================================================================
// Resolver Introspection Tests (HasResolver / ListResolvers / ResolverCount)
// ============================================================================

func TestE2E_HasResolver_BuiltIn(t *testing.T) {
	engine := prompty.MustNew()

	// Test built-in resolvers exist
	// Note: prompty.if/for/switch are handled by the executor's block processing,
	// not registered as separate resolvers. Only prompty.var, prompty.raw, prompty.include
	// are registered as traditional resolvers.
	assert.True(t, engine.HasResolver("prompty.var"))
	assert.True(t, engine.HasResolver("prompty.raw"))
	assert.True(t, engine.HasResolver("prompty.include"))

	// Test nonexistent resolver
	assert.False(t, engine.HasResolver("nonexistent"))
	assert.False(t, engine.HasResolver("custom.tag"))
}

func TestE2E_HasResolver_CustomResolver(t *testing.T) {
	engine := prompty.MustNew()

	// Register custom resolver (uppercaseResolver uses "myapp.uppercase")
	engine.MustRegister(&uppercaseResolver{})

	// Verify it's registered
	assert.True(t, engine.HasResolver("myapp.uppercase"))
	assert.False(t, engine.HasResolver("myapp.lowercase"))
}

func TestE2E_ListResolvers(t *testing.T) {
	engine := prompty.MustNew()

	resolvers := engine.ListResolvers()

	// Should contain built-in resolvers
	assert.Contains(t, resolvers, "prompty.var")
	assert.Contains(t, resolvers, "prompty.raw")
	assert.Contains(t, resolvers, "prompty.include")

	// List should be sorted
	sorted := true
	for i := 1; i < len(resolvers); i++ {
		if resolvers[i-1] > resolvers[i] {
			sorted = false
			break
		}
	}
	assert.True(t, sorted, "ListResolvers should return sorted list")
}

func TestE2E_ListResolvers_IncludesCustom(t *testing.T) {
	engine := prompty.MustNew()

	// Register custom resolver (uppercaseResolver uses "myapp.uppercase")
	engine.MustRegister(&uppercaseResolver{})

	resolvers := engine.ListResolvers()

	// Should contain custom resolver
	assert.Contains(t, resolvers, "myapp.uppercase")
}

func TestE2E_ResolverCount(t *testing.T) {
	engine := prompty.MustNew()

	// Get initial count (built-in resolvers)
	initialCount := engine.ResolverCount()
	assert.Greater(t, initialCount, 0)

	// Register custom resolver
	engine.MustRegister(&uppercaseResolver{})

	// Count should increase by 1
	assert.Equal(t, initialCount+1, engine.ResolverCount())
}

func TestE2E_ResolverIntrospection_Consistency(t *testing.T) {
	engine := prompty.MustNew()

	// ResolverCount should match length of ListResolvers
	assert.Equal(t, engine.ResolverCount(), len(engine.ListResolvers()))

	// Register custom resolver
	engine.MustRegister(&uppercaseResolver{})

	// Still consistent after registration
	assert.Equal(t, engine.ResolverCount(), len(engine.ListResolvers()))
}

// Template Inheritance Tests

func TestE2E_Inheritance_BasicExtends(t *testing.T) {
	engine := prompty.MustNew()

	// Register base template
	engine.MustRegisterTemplate("base", `{~prompty.block name="content"~}Default Content{~/prompty.block~}`)

	// Child template extends base and overrides block
	child := `{~prompty.extends template="base" /~}
{~prompty.block name="content"~}Overridden Content{~/prompty.block~}`

	result, err := engine.Execute(context.Background(), child, nil)
	require.NoError(t, err)
	assert.Equal(t, "Overridden Content", strings.TrimSpace(result))
}

func TestE2E_Inheritance_BlockNotOverridden(t *testing.T) {
	engine := prompty.MustNew()

	// Register base template with two blocks
	engine.MustRegisterTemplate("base", `{~prompty.block name="header"~}Header{~/prompty.block~}
{~prompty.block name="footer"~}Footer{~/prompty.block~}`)

	// Child only overrides header
	child := `{~prompty.extends template="base" /~}
{~prompty.block name="header"~}Custom Header{~/prompty.block~}`

	result, err := engine.Execute(context.Background(), child, nil)
	require.NoError(t, err)
	assert.Contains(t, result, "Custom Header")
	assert.Contains(t, result, "Footer")
}

func TestE2E_Inheritance_ParentCall(t *testing.T) {
	engine := prompty.MustNew()

	// Register base template
	engine.MustRegisterTemplate("base", `{~prompty.block name="content"~}Base Content{~/prompty.block~}`)

	// Child extends and calls parent
	child := `{~prompty.extends template="base" /~}
{~prompty.block name="content"~}Before - {~prompty.parent /~} - After{~/prompty.block~}`

	result, err := engine.Execute(context.Background(), child, nil)
	require.NoError(t, err)
	assert.Equal(t, "Before - Base Content - After", strings.TrimSpace(result))
}

func TestE2E_Inheritance_WithVariables(t *testing.T) {
	engine := prompty.MustNew()

	// Register base template with variable
	engine.MustRegisterTemplate("base", `{~prompty.block name="greeting"~}Hello, {~prompty.var name="name" default="World" /~}!{~/prompty.block~}`)

	// Child overrides greeting
	child := `{~prompty.extends template="base" /~}
{~prompty.block name="greeting"~}Welcome, {~prompty.var name="name" /~}!{~/prompty.block~}`

	result, err := engine.Execute(context.Background(), child, map[string]any{
		"name": "Alice",
	})
	require.NoError(t, err)
	assert.Equal(t, "Welcome, Alice!", strings.TrimSpace(result))
}

func TestE2E_Inheritance_MultiLevel(t *testing.T) {
	engine := prompty.MustNew()

	// Level 1: Base layout
	engine.MustRegisterTemplate("layout", `{~prompty.block name="body"~}Default Body{~/prompty.block~}`)

	// Level 2: Page layout extends layout
	engine.MustRegisterTemplate("page-layout", `{~prompty.extends template="layout" /~}
{~prompty.block name="body"~}Page: {~prompty.parent /~}{~/prompty.block~}`)

	// Level 3: Actual page extends page-layout
	page := `{~prompty.extends template="page-layout" /~}
{~prompty.block name="body"~}Custom: {~prompty.parent /~}{~/prompty.block~}`

	result, err := engine.Execute(context.Background(), page, nil)
	require.NoError(t, err)
	assert.Equal(t, "Custom: Page: Default Body", strings.TrimSpace(result))
}

func TestE2E_Inheritance_ExecuteRegisteredTemplate(t *testing.T) {
	engine := prompty.MustNew()

	// Register base and child templates
	engine.MustRegisterTemplate("base", `{~prompty.block name="msg"~}Hello{~/prompty.block~}`)
	engine.MustRegisterTemplate("child", `{~prompty.extends template="base" /~}
{~prompty.block name="msg"~}Goodbye{~/prompty.block~}`)

	// Execute child template by name
	result, err := engine.ExecuteTemplate(context.Background(), "child", nil)
	require.NoError(t, err)
	assert.Equal(t, "Goodbye", strings.TrimSpace(result))
}

func TestE2E_Inheritance_EmptyBlock(t *testing.T) {
	engine := prompty.MustNew()

	// Base with empty block
	engine.MustRegisterTemplate("base", `Start{~prompty.block name="middle"~}{~/prompty.block~}End`)

	// Child fills the empty block
	child := `{~prompty.extends template="base" /~}
{~prompty.block name="middle"~} CONTENT {~/prompty.block~}`

	result, err := engine.Execute(context.Background(), child, nil)
	require.NoError(t, err)
	assert.Equal(t, "Start CONTENT End", strings.TrimSpace(result))
}

func TestE2E_Inheritance_ParentNotFound(t *testing.T) {
	engine := prompty.MustNew()

	// Try to extend non-existent template
	child := `{~prompty.extends template="nonexistent" /~}
{~prompty.block name="content"~}Content{~/prompty.block~}`

	_, err := engine.Execute(context.Background(), child, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template not found")
}

func TestE2E_Inheritance_MultipleBlocks(t *testing.T) {
	engine := prompty.MustNew()

	// Base with three blocks
	engine.MustRegisterTemplate("base", `{~prompty.block name="header"~}H{~/prompty.block~}|{~prompty.block name="body"~}B{~/prompty.block~}|{~prompty.block name="footer"~}F{~/prompty.block~}`)

	// Child overrides two blocks
	child := `{~prompty.extends template="base" /~}
{~prompty.block name="header"~}HEADER{~/prompty.block~}
{~prompty.block name="footer"~}FOOTER{~/prompty.block~}`

	result, err := engine.Execute(context.Background(), child, nil)
	require.NoError(t, err)
	assert.Equal(t, "HEADER|B|FOOTER", strings.TrimSpace(result))
}
