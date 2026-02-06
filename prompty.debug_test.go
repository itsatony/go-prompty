package prompty

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplate_DryRun_BasicVariable(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`Hello {~prompty.var name="user" /~}!`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), map[string]any{
		"user": "Alice",
	})

	assert.True(t, result.Valid)
	assert.Len(t, result.Variables, 1)
	assert.Equal(t, "user", result.Variables[0].Name)
	assert.True(t, result.Variables[0].InData)
	assert.Empty(t, result.MissingVariables)
	assert.Contains(t, result.Output, "Alice")
}

func TestTemplate_DryRun_MissingVariable(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`Hello {~prompty.var name="user" /~}!`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), map[string]any{})

	assert.True(t, result.Valid) // Still valid, just has missing var
	assert.Len(t, result.Variables, 1)
	assert.Equal(t, "user", result.Variables[0].Name)
	assert.False(t, result.Variables[0].InData)
	assert.Contains(t, result.MissingVariables, "user")
	assert.Contains(t, result.Output, "{{user}}")
}

func TestTemplate_DryRun_VariableWithDefault(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`Hello {~prompty.var name="user" default="Guest" /~}!`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), map[string]any{})

	assert.True(t, result.Valid)
	assert.Len(t, result.Variables, 1)
	assert.Equal(t, "user", result.Variables[0].Name)
	assert.True(t, result.Variables[0].HasDefault)
	assert.Equal(t, "Guest", result.Variables[0].Default)
	assert.Empty(t, result.MissingVariables) // Has default, so not missing
	assert.Contains(t, result.Output, "Guest")
}

func TestTemplate_DryRun_NestedVariable(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`Name: {~prompty.var name="user.profile.name" /~}`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), map[string]any{
		"user": map[string]any{
			"profile": map[string]any{
				"name": "Alice",
			},
		},
	})

	assert.True(t, result.Valid)
	assert.Len(t, result.Variables, 1)
	assert.Equal(t, "user.profile.name", result.Variables[0].Name)
	assert.True(t, result.Variables[0].InData)
	assert.Contains(t, result.Output, "Alice")
}

func TestTemplate_DryRun_VariableSuggestions(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`Hello {~prompty.var name="usre" /~}!`) // typo: usre instead of user
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), map[string]any{
		"user": "Alice",
	})

	assert.Len(t, result.Variables, 1)
	assert.False(t, result.Variables[0].InData)
	assert.Contains(t, result.Variables[0].Suggestions, "user")
}

func TestTemplate_DryRun_UnusedVariables(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`Hello {~prompty.var name="user" /~}!`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), map[string]any{
		"user":   "Alice",
		"unused": "value",
	})

	assert.True(t, result.Valid)
	assert.Contains(t, result.UnusedVariables, "unused")
}

func TestTemplate_DryRun_Include(t *testing.T) {
	engine := MustNew()
	err := engine.RegisterTemplate("header", "Header Content")
	require.NoError(t, err)

	tmpl, err := engine.Parse(`{~prompty.include template="header" /~}`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), nil)

	assert.True(t, result.Valid)
	assert.Len(t, result.Includes, 1)
	assert.Equal(t, "header", result.Includes[0].TemplateName)
	assert.True(t, result.Includes[0].Exists)
	assert.Empty(t, result.Warnings)
}

func TestTemplate_DryRun_IncludeNotFound(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`{~prompty.include template="missing" /~}`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), nil)

	assert.True(t, result.Valid) // Still valid structure
	assert.Len(t, result.Includes, 1)
	assert.Equal(t, "missing", result.Includes[0].TemplateName)
	assert.False(t, result.Includes[0].Exists)
	assert.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "missing")
}

func TestTemplate_DryRun_IncludeIsolated(t *testing.T) {
	engine := MustNew()
	err := engine.RegisterTemplate("header", "Header")
	require.NoError(t, err)

	tmpl, err := engine.Parse(`{~prompty.include template="header" isolate="true" /~}`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), nil)

	assert.Len(t, result.Includes, 1)
	assert.True(t, result.Includes[0].Isolated)
}

func TestTemplate_DryRun_Conditional(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`
{~prompty.if eval="user.isAdmin"~}
Admin
{~prompty.elseif eval="user.isLoggedIn"~}
User
{~prompty.else~}
Guest
{~/prompty.if~}`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), nil)

	assert.True(t, result.Valid)
	assert.Len(t, result.Conditionals, 1)
	assert.Equal(t, "user.isAdmin", result.Conditionals[0].Condition)
	assert.True(t, result.Conditionals[0].HasElseIf)
	assert.True(t, result.Conditionals[0].HasElse)
}

func TestTemplate_DryRun_SimpleConditional(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`{~prompty.if eval="showContent"~}Content{~/prompty.if~}`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), nil)

	assert.Len(t, result.Conditionals, 1)
	assert.Equal(t, "showContent", result.Conditionals[0].Condition)
	assert.False(t, result.Conditionals[0].HasElseIf)
	assert.False(t, result.Conditionals[0].HasElse)
}

func TestTemplate_DryRun_Loop(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`{~prompty.for item="x" index="i" in="items" limit="10"~}{~prompty.var name="x" /~}{~/prompty.for~}`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), map[string]any{
		"items": []string{"a", "b"},
	})

	assert.True(t, result.Valid)
	assert.Len(t, result.Loops, 1)
	assert.Equal(t, "x", result.Loops[0].ItemVar)
	assert.Equal(t, "i", result.Loops[0].IndexVar)
	assert.Equal(t, "items", result.Loops[0].Source)
	assert.Equal(t, 10, result.Loops[0].Limit)
	assert.True(t, result.Loops[0].InData)
}

func TestTemplate_DryRun_LoopSourceNotFound(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`{~prompty.for item="x" in="missing"~}{~prompty.var name="x" /~}{~/prompty.for~}`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), map[string]any{})

	assert.True(t, result.Valid)
	assert.Len(t, result.Loops, 1)
	assert.False(t, result.Loops[0].InData)
	assert.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "missing")
}

func TestTemplate_DryRun_RawContent(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`{~prompty.raw~}This {~prompty.var~} is not parsed{~/prompty.raw~}`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), nil)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Variables) // Raw content should not be parsed
	assert.Contains(t, result.Output, "This {~prompty.var~} is not parsed")
}

func TestTemplate_DryRun_Comment(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`Before{~prompty.comment~}This is removed{~/prompty.comment~}After`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), nil)

	assert.True(t, result.Valid)
	assert.Equal(t, "BeforeAfter", result.Output)
}

func TestTemplate_DryRun_ComplexTemplate(t *testing.T) {
	engine := MustNew()
	err := engine.RegisterTemplate("item", "{~prompty.var name=\"name\" /~}")
	require.NoError(t, err)

	tmpl, err := engine.Parse(`
Welcome {~prompty.var name="user.name" default="Guest" /~}!
{~prompty.if eval="user.isAdmin"~}
Admin Panel
{~prompty.for item="x" in="user.permissions"~}
- {~prompty.var name="x" /~}
{~/prompty.for~}
{~/prompty.if~}
{~prompty.include template="item" name="footer" /~}
`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), map[string]any{
		"user": map[string]any{
			"name":        "Alice",
			"isAdmin":     true,
			"permissions": []string{"read", "write"},
		},
	})

	assert.True(t, result.Valid)
	assert.True(t, len(result.Variables) >= 2)
	assert.Len(t, result.Conditionals, 1)
	assert.Len(t, result.Loops, 1)
	assert.Len(t, result.Includes, 1)
}

func TestTemplate_DryRun_StringOutput(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`Hello {~prompty.var name="missing" /~}!`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), map[string]any{
		"user": "Alice",
	})

	output := result.String()

	assert.Contains(t, output, "Dry Run Result")
	assert.Contains(t, output, "Variables")
	assert.Contains(t, output, "missing")
	assert.Contains(t, output, "MISSING")
	assert.Contains(t, output, "Unused Variables")
	assert.Contains(t, output, "user")
}

func TestTemplate_Explain_Basic(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`Hello {~prompty.var name="user" default="World" /~}!`)
	require.NoError(t, err)

	result := tmpl.Explain(context.Background(), map[string]any{
		"user": "Alice",
	})

	assert.NoError(t, result.Error)
	assert.Equal(t, "Hello Alice!", result.Output)
	assert.NotEmpty(t, result.AST)
	assert.Contains(t, result.AST, "Root")
	assert.Contains(t, result.AST, "Tag: prompty.var")
	assert.True(t, result.Timing.Total > 0)
}

func TestTemplate_Explain_VariableAccesses(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`{~prompty.var name="first" /~} {~prompty.var name="second" default="default" /~}`)
	require.NoError(t, err)

	result := tmpl.Explain(context.Background(), map[string]any{
		"first": "A",
	})

	assert.Len(t, result.Variables, 2)

	// First variable should be found
	assert.Equal(t, "first", result.Variables[0].Path)
	assert.True(t, result.Variables[0].Found)
	assert.Equal(t, "A", result.Variables[0].Value)

	// Second variable not found but has default
	assert.Equal(t, "second", result.Variables[1].Path)
	assert.False(t, result.Variables[1].Found)
	assert.Equal(t, "default", result.Variables[1].Default)
}

func TestTemplate_Explain_ASTFormatting(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`
{~prompty.if eval="show"~}
Content {~prompty.var name="x" /~}
{~prompty.else~}
Hidden
{~/prompty.if~}
`)
	require.NoError(t, err)

	result := tmpl.Explain(context.Background(), map[string]any{
		"show": true,
		"x":    "value",
	})

	assert.Contains(t, result.AST, "Root")
	assert.Contains(t, result.AST, "Conditional")
	assert.Contains(t, result.AST, "Then:")
	assert.Contains(t, result.AST, "Else:")
}

func TestTemplate_Explain_ForLoop(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`{~prompty.for item="x" index="i" in="items" limit="5"~}{~prompty.var name="x" /~}{~/prompty.for~}`)
	require.NoError(t, err)

	result := tmpl.Explain(context.Background(), map[string]any{
		"items": []string{"a", "b"},
	})

	assert.Contains(t, result.AST, "For: x in items")
	assert.Contains(t, result.AST, "index: i")
	assert.Contains(t, result.AST, "limit: 5")
	assert.Equal(t, "ab", result.Output)
}

func TestTemplate_Explain_StringOutput(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`Hello {~prompty.var name="user" /~}!`)
	require.NoError(t, err)

	result := tmpl.Explain(context.Background(), map[string]any{
		"user": "Alice",
	})

	output := result.String()

	assert.Contains(t, output, "Template Explanation")
	assert.Contains(t, output, "AST Structure")
	assert.Contains(t, output, "Variable Accesses")
	assert.Contains(t, output, "Timing")
	assert.Contains(t, output, "Output")
	assert.Contains(t, output, "Hello Alice!")
}

func TestTemplate_Explain_Error(t *testing.T) {
	engine := MustNew(WithErrorStrategy(ErrorStrategyThrow))
	tmpl, err := engine.Parse(`Hello {~prompty.var name="missing" /~}!`)
	require.NoError(t, err)

	result := tmpl.Explain(context.Background(), map[string]any{})

	assert.Error(t, result.Error)

	output := result.String()
	assert.Contains(t, output, "Error")
}

// Test helper functions

func TestGetPath(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		path     string
		expected any
		found    bool
	}{
		{
			name:     "simple key",
			data:     map[string]any{"user": "Alice"},
			path:     "user",
			expected: "Alice",
			found:    true,
		},
		{
			name: "nested key",
			data: map[string]any{
				"user": map[string]any{
					"name": "Alice",
				},
			},
			path:     "user.name",
			expected: "Alice",
			found:    true,
		},
		{
			name: "deeply nested",
			data: map[string]any{
				"a": map[string]any{
					"b": map[string]any{
						"c": "value",
					},
				},
			},
			path:     "a.b.c",
			expected: "value",
			found:    true,
		},
		{
			name:     "missing key",
			data:     map[string]any{"user": "Alice"},
			path:     "missing",
			expected: nil,
			found:    false,
		},
		{
			name: "missing nested key",
			data: map[string]any{
				"user": map[string]any{
					"name": "Alice",
				},
			},
			path:     "user.email",
			expected: nil,
			found:    false,
		},
		{
			name:     "empty path",
			data:     map[string]any{"user": "Alice"},
			path:     "",
			expected: nil,
			found:    false,
		},
		{
			name:     "nil data",
			data:     nil,
			path:     "user",
			expected: nil,
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, found := getPath(tt.data, tt.path)
			assert.Equal(t, tt.found, found)
			if tt.found {
				assert.Equal(t, tt.expected, val)
			}
		})
	}
}

func TestHasPath(t *testing.T) {
	data := map[string]any{
		"user": map[string]any{
			"name": "Alice",
		},
	}

	assert.True(t, hasPath(data, "user"))
	assert.True(t, hasPath(data, "user.name"))
	assert.False(t, hasPath(data, "user.email"))
	assert.False(t, hasPath(data, "missing"))
}

func TestCollectAllKeys(t *testing.T) {
	data := map[string]any{
		"name": "test",
		"user": map[string]any{
			"id": "123",
			"profile": map[string]any{
				"email": "test@example.com",
			},
		},
		"items": []string{"a", "b"},
	}

	keys := collectAllKeys(data, "")

	assert.Contains(t, keys, "name")
	assert.Contains(t, keys, "user")
	assert.Contains(t, keys, "user.id")
	assert.Contains(t, keys, "user.profile")
	assert.Contains(t, keys, "user.profile.email")
	assert.Contains(t, keys, "items")
}

func TestMarkKeyUsed(t *testing.T) {
	usedKeys := make(map[string]bool)

	markKeyUsed(usedKeys, "user.profile.name")

	assert.True(t, usedKeys["user.profile.name"])
	assert.True(t, usedKeys["user.profile"])
	assert.True(t, usedKeys["user"])
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"abc", "abc", 0},
		{"abc", "ab", 1},
		{"abc", "abcd", 1},
		{"abc", "abd", 1},
		{"kitten", "sitting", 3},
		{"user", "usre", 2},
		{"name", "nmae", 2},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			result := levenshteinDistance(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindSimilarStrings(t *testing.T) {
	candidates := []string{"user", "username", "email", "name", "profile"}

	// typo of "user"
	suggestions := findSimilarStrings("usre", candidates, 3)
	assert.Contains(t, suggestions, "user")

	// typo of "name"
	suggestions = findSimilarStrings("nmae", candidates, 3)
	assert.Contains(t, suggestions, "name")

	// completely different string
	suggestions = findSimilarStrings("zzzzzzzzz", candidates, 3)
	assert.Empty(t, suggestions)
}

func TestMinOfThree(t *testing.T) {
	assert.Equal(t, 1, minOfThree(1, 2, 3))
	assert.Equal(t, 1, minOfThree(3, 1, 2))
	assert.Equal(t, 1, minOfThree(2, 3, 1))
	assert.Equal(t, 5, minOfThree(5, 5, 5))
}

func TestDryRunResult_String_AllSections(t *testing.T) {
	result := &DryRunResult{
		Valid:  true,
		Output: "test output",
		Variables: []VariableReference{
			{Name: "found", Line: 1, InData: true},
			{Name: "missing", Line: 2, InData: false, Suggestions: []string{"missed"}},
			{Name: "defaulted", Line: 3, InData: false, HasDefault: true, Default: "def"},
		},
		Resolvers: []ResolverReference{
			{TagName: "CustomTag", Line: 4},
		},
		Includes: []IncludeReference{
			{TemplateName: "header", Line: 5, Exists: true},
			{TemplateName: "missing", Line: 6, Exists: false},
		},
		Conditionals: []ConditionalReference{
			{Condition: "isTrue", Line: 7, HasElseIf: true, HasElse: true},
		},
		Loops: []LoopReference{
			{ItemVar: "x", Source: "items", Line: 8, InData: true},
			{ItemVar: "y", Source: "missing", Line: 9, InData: false},
		},
		MissingVariables: []string{"missing"},
		UnusedVariables:  []string{"extra"},
		Errors:           []string{"some error"},
		Warnings:         []string{"some warning"},
	}

	output := result.String()

	// Check all sections are present
	assert.Contains(t, output, "Valid: true")
	assert.Contains(t, output, "Variables (3)")
	assert.Contains(t, output, "found [line 1]: found")
	assert.Contains(t, output, "missing [line 2]: MISSING")
	assert.Contains(t, output, "Did you mean: missed?")
	assert.Contains(t, output, "defaulted [line 3]: not found (default: \"def\")")
	assert.Contains(t, output, "Resolvers (1)")
	assert.Contains(t, output, "CustomTag [line 4]")
	assert.Contains(t, output, "Includes (2)")
	assert.Contains(t, output, "header [line 5]: found")
	assert.Contains(t, output, "missing [line 6]: NOT FOUND")
	assert.Contains(t, output, "Conditionals (1)")
	assert.Contains(t, output, "isTrue [line 7]")
	assert.Contains(t, output, "Loops (2)")
	assert.Contains(t, output, "for x in items [line 8]: source found")
	assert.Contains(t, output, "for y in missing [line 9]: source NOT FOUND")
	assert.Contains(t, output, "Missing Variables (1)")
	assert.Contains(t, output, "Unused Variables (1)")
	assert.Contains(t, output, "Errors (1)")
	assert.Contains(t, output, "some error")
	assert.Contains(t, output, "Warnings (1)")
	assert.Contains(t, output, "some warning")
	assert.Contains(t, output, "Placeholder Output")
	assert.Contains(t, output, "test output")
}

func TestExplainResult_String_WithError(t *testing.T) {
	result := &ExplainResult{
		AST:    "Root\n  Text: \"test\"",
		Output: "test",
		Error:  assert.AnError,
		Variables: []VariableAccess{
			{Path: "user", Value: "Alice", Found: true, Line: 1},
			{Path: "missing", Found: false, Default: "default", Line: 2},
			{Path: "nodefault", Found: false, Line: 3},
		},
		Timing: ExecutionTiming{
			Total:     100,
			Execution: 50,
		},
	}

	output := result.String()

	assert.Contains(t, output, "Template Explanation")
	assert.Contains(t, output, "AST Structure")
	assert.Contains(t, output, "Root")
	assert.Contains(t, output, "Variable Accesses")
	assert.Contains(t, output, "[line 1] user: = Alice")
	assert.Contains(t, output, "[line 2] missing: not found, using default: \"default\"")
	assert.Contains(t, output, "[line 3] nodefault: NOT FOUND")
	assert.Contains(t, output, "Timing")
	assert.Contains(t, output, "Error")
	assert.Contains(t, output, "Output")
}

func TestTemplate_DryRun_Switch(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`{~prompty.switch eval="status"~}{~prompty.case value="active"~}Active{~/prompty.case~}{~prompty.case value="pending"~}Pending{~/prompty.case~}{~prompty.casedefault~}Unknown{~/prompty.casedefault~}{~/prompty.switch~}`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), map[string]any{
		"status": "active",
	})

	assert.True(t, result.Valid)
	// Switch nodes are processed but we don't have a specific Switches field
	// The output should contain placeholders
	assert.Contains(t, result.Output, "{{switch:")
}

func TestTemplate_Explain_Switch(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`{~prompty.switch eval="status"~}{~prompty.case value="active"~}Active{~/prompty.case~}{~prompty.case value="pending"~}Pending{~/prompty.case~}{~prompty.casedefault~}Unknown{~/prompty.casedefault~}{~/prompty.switch~}`)
	require.NoError(t, err)

	result := tmpl.Explain(context.Background(), map[string]any{
		"status": "active",
	})

	assert.NoError(t, result.Error)
	assert.Contains(t, result.AST, "Switch:")
	assert.Contains(t, result.AST, "Case:")
	assert.Contains(t, result.AST, "Default:")
	assert.Contains(t, strings.TrimSpace(result.Output), "Active")
}

func TestTemplate_DryRun_MultipleVariableSameKey(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`{~prompty.var name="user" /~} - {~prompty.var name="user" /~}`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), map[string]any{
		"user": "Alice",
	})

	assert.True(t, result.Valid)
	assert.Len(t, result.Variables, 2)
	assert.Empty(t, result.MissingVariables)
	assert.Equal(t, "Alice - Alice", result.Output)
}

func TestTemplate_DryRun_EmptyData(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`Plain text only`)
	require.NoError(t, err)

	result := tmpl.DryRun(context.Background(), nil)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Variables)
	assert.Empty(t, result.MissingVariables)
	assert.Empty(t, result.UnusedVariables)
	assert.Equal(t, "Plain text only", result.Output)
}

func TestTemplate_Explain_EmptyTemplate(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse(`Plain text only`)
	require.NoError(t, err)

	result := tmpl.Explain(context.Background(), nil)

	assert.NoError(t, result.Error)
	assert.Equal(t, "Plain text only", result.Output)
	assert.Empty(t, result.Variables)
	assert.Contains(t, result.AST, "Root")
	assert.Contains(t, result.AST, "Text:")
}

func TestGetPath_MapStringString(t *testing.T) {
	// Test with map[string]string type
	data := map[string]any{
		"headers": map[string]string{
			"Content-Type": "application/json",
		},
	}

	val, found := getPath(data, "headers.Content-Type")
	assert.True(t, found)
	assert.Equal(t, "application/json", val)
}
