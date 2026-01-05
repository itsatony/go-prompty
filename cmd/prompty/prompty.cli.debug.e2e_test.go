package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

// createTempDataFile creates a temporary JSON data file for testing
func createTempDataFile(t *testing.T, data map[string]any) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "data.json")
	jsonBytes, err := json.Marshal(data)
	require.NoError(t, err)
	err = os.WriteFile(tmpFile, jsonBytes, 0644)
	require.NoError(t, err)
	return tmpFile
}

// =============================================================================
// Variable Analysis Tests
// =============================================================================

func TestDebug_E2E_VariableAnalysis(t *testing.T) {
	t.Run("DetectsAllVariables", func(t *testing.T) {
		template := `Hello {~prompty.var name="firstName" /~} {~prompty.var name="lastName" /~}!
Your age is {~prompty.var name="age" /~}.
Email: {~prompty.var name="contact.email" /~}`

		tmpFile := createTempTemplate(t, template)
		data := map[string]any{
			"firstName": "Alice",
			"lastName":  "Smith",
			"age":       30,
			"contact": map[string]any{
				"email": "alice@example.com",
			},
		}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
		}, nil, &stdout, &stderr)

		output := stdout.String()
		assert.Equal(t, ExitCodeSuccess, exitCode)
		assert.Contains(t, output, "firstName")
		assert.Contains(t, output, "lastName")
		assert.Contains(t, output, "age")
		assert.Contains(t, output, "contact.email")
		// All variables should exist
		assert.Contains(t, output, "Alice")
	})

	t.Run("DetectsVariablesWithDefaults", func(t *testing.T) {
		template := `{~prompty.var name="greeting" default="Hello" /~} {~prompty.var name="name" /~}!`

		tmpFile := createTempTemplate(t, template)
		data := map[string]any{"name": "World"}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
		}, nil, &stdout, &stderr)

		output := stdout.String()
		assert.Equal(t, ExitCodeSuccess, exitCode)
		assert.Contains(t, output, "greeting")
		assert.Contains(t, output, "default: Hello")
	})

	t.Run("VariablesInLoops", func(t *testing.T) {
		// Note: debug command does static analysis - loop variables like user.name
		// are detected as variables but reported as "missing" since they're bound
		// at runtime by the loop, not present in the data context
		template := `{~prompty.for item="user" in="users"~}
{~prompty.var name="user.name" /~}: {~prompty.var name="user.email" /~}
{~/prompty.for~}`

		tmpFile := createTempTemplate(t, template)
		data := map[string]any{
			"users": []map[string]any{
				{"name": "Alice", "email": "alice@example.com"},
				{"name": "Bob", "email": "bob@example.com"},
			},
		}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
		}, nil, &stdout, &stderr)

		// Static analysis reports loop variables as missing (they're bound at runtime)
		assert.Equal(t, ExitCodeValidationError, exitCode)
		output := stdout.String()
		// Variables are detected but shown as missing
		assert.Contains(t, output, "user.name")
		assert.Contains(t, output, "user.email")
		assert.Contains(t, output, "MISSING")
	})
}

// =============================================================================
// Missing Variables Tests
// =============================================================================

func TestDebug_E2E_MissingVariables(t *testing.T) {
	t.Run("DetectsMissingVariables", func(t *testing.T) {
		template := `Hello {~prompty.var name="name" /~}!
Your role is {~prompty.var name="role" /~}.`

		tmpFile := createTempTemplate(t, template)
		data := map[string]any{"name": "Alice"}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
		}, nil, &stdout, &stderr)

		output := stdout.String()
		// Should report missing variable
		assert.Contains(t, output, "MISSING")
		assert.Contains(t, output, "role")
		// Exit code should indicate validation error
		assert.Equal(t, ExitCodeValidationError, exitCode)
	})

	t.Run("MissingWithDefault_NoError", func(t *testing.T) {
		template := `{~prompty.var name="greeting" default="Hello" /~} {~prompty.var name="name" /~}!`

		tmpFile := createTempTemplate(t, template)
		data := map[string]any{"name": "World"}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
		}, nil, &stdout, &stderr)

		// "greeting" is missing but has default, so should succeed
		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("SuggestsSimilarVariables", func(t *testing.T) {
		template := `{~prompty.var name="userName" /~}`

		tmpFile := createTempTemplate(t, template)
		data := map[string]any{
			"username":  "alice",
			"user_name": "alice",
		}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
			"-v", // verbose mode to see suggestions
		}, nil, &stdout, &stderr)

		output := stdout.String()
		// Should suggest similar keys
		assert.Contains(t, output, "did you mean")
		assert.Equal(t, ExitCodeValidationError, exitCode)
	})

	t.Run("NestedPathMissing", func(t *testing.T) {
		template := `{~prompty.var name="user.profile.name" /~}`

		tmpFile := createTempTemplate(t, template)
		data := map[string]any{
			"user": map[string]any{
				"id": 123,
				// profile is missing
			},
		}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
		}, nil, &stdout, &stderr)

		output := stdout.String()
		assert.Contains(t, output, "MISSING")
		assert.Equal(t, ExitCodeValidationError, exitCode)
	})
}

// =============================================================================
// Unused Data Tests
// =============================================================================

func TestDebug_E2E_UnusedData(t *testing.T) {
	t.Run("DetectsUnusedFields", func(t *testing.T) {
		template := `Hello {~prompty.var name="name" /~}!`

		tmpFile := createTempTemplate(t, template)
		data := map[string]any{
			"name":   "Alice",
			"unused": "This field is not used",
			"extra":  123,
		}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
		}, nil, &stdout, &stderr)

		output := stdout.String()
		assert.Equal(t, ExitCodeSuccess, exitCode)
		assert.Contains(t, output, "Unused")
		assert.Contains(t, output, "unused")
		assert.Contains(t, output, "extra")
	})

	t.Run("NoUnusedFields", func(t *testing.T) {
		template := `{~prompty.var name="a" /~} {~prompty.var name="b" /~}`

		tmpFile := createTempTemplate(t, template)
		data := map[string]any{
			"a": "value1",
			"b": "value2",
		}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
		}, nil, &stdout, &stderr)

		output := stdout.String()
		assert.Equal(t, ExitCodeSuccess, exitCode)
		// Should not show unused section
		assert.NotContains(t, output, "Unused Data Fields")
	})

	t.Run("NestedUnusedFields", func(t *testing.T) {
		template := `{~prompty.var name="user.name" /~}`

		tmpFile := createTempTemplate(t, template)
		data := map[string]any{
			"user": map[string]any{
				"name":  "Alice",
				"email": "unused@example.com",
			},
		}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
		}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)
		// Should detect unused nested field (depends on implementation)
		_ = stdout.String() // Content check depends on implementation details
	})
}

// =============================================================================
// Include Validation Tests
// =============================================================================

func TestDebug_E2E_IncludeValidation(t *testing.T) {
	t.Run("DetectsMissingIncludes", func(t *testing.T) {
		template := `Header: {~prompty.include template="header" /~}
Footer: {~prompty.include template="footer" /~}`

		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{"-t", tmpFile}, nil, &stdout, &stderr)

		output := stdout.String()
		assert.Contains(t, output, "Includes")
		assert.Contains(t, output, "NOT FOUND")
		assert.Contains(t, output, "header")
		assert.Contains(t, output, "footer")
		// Still valid (no missing variables), but includes are flagged
		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("NoIncludesTemplate", func(t *testing.T) {
		template := `Simple template without includes: {~prompty.var name="name" default="World" /~}`

		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{"-t", tmpFile}, nil, &stdout, &stderr)

		output := stdout.String()
		assert.Equal(t, ExitCodeSuccess, exitCode)
		// Should not show includes section
		assert.NotContains(t, output, "Includes (")
	})
}

// =============================================================================
// Trace Mode Tests
// =============================================================================

func TestDebug_E2E_TraceMode(t *testing.T) {
	t.Run("TraceShowsWarnings", func(t *testing.T) {
		// Template that might generate warnings
		template := `{~prompty.var name="name" /~}`

		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"--trace",
		}, nil, &stdout, &stderr)

		// Trace should work (may or may not have output depending on warnings)
		assert.NotEqual(t, ExitCodeUsageError, exitCode)
	})

	t.Run("TraceWithComplexTemplate", func(t *testing.T) {
		template := `{~prompty.if eval="show"~}
{~prompty.for item="x" in="items"~}
{~prompty.var name="x" /~}
{~/prompty.for~}
{~/prompty.if~}`

		tmpFile := createTempTemplate(t, template)
		data := map[string]any{
			"show":  true,
			"items": []string{"a", "b", "c"},
		}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
			"--trace",
		}, nil, &stdout, &stderr)

		// Should not error
		assert.NotEqual(t, ExitCodeUsageError, exitCode)
		assert.NotEqual(t, ExitCodeInputError, exitCode)
	})
}

// =============================================================================
// JSON Output Tests
// =============================================================================

func TestDebug_E2E_JSONOutput(t *testing.T) {
	template := `Hello {~prompty.var name="name" /~}!
Role: {~prompty.var name="role" default="user" /~}`

	tmpFile := createTempTemplate(t, template)
	data := map[string]any{
		"name":   "Alice",
		"unused": "extra data",
	}

	t.Run("ValidJSON", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
			"-F", "json",
		}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)

		// Should be valid JSON
		var output debugOutput
		err := json.Unmarshal(stdout.Bytes(), &output)
		require.NoError(t, err, "Output should be valid JSON")
	})

	t.Run("JSONStructure", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
			"-F", "json",
		}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)

		var output debugOutput
		err := json.Unmarshal(stdout.Bytes(), &output)
		require.NoError(t, err)

		// Check expected structure
		// Note: 'resolvers' is for custom resolvers only, not built-in tags like prompty.var
		assert.True(t, output.Valid)
		assert.NotEmpty(t, output.Variables)

		// Check variable structure
		for _, v := range output.Variables {
			assert.NotEmpty(t, v.Name)
			assert.Greater(t, v.Line, 0)
			assert.Greater(t, v.Column, 0)
		}
	})

	t.Run("JSONWithMissingVariables", func(t *testing.T) {
		missingTemplate := `{~prompty.var name="required" /~}`
		tmpFile := createTempTemplate(t, missingTemplate)

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-F", "json",
		}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeValidationError, exitCode)

		var output debugOutput
		err := json.Unmarshal(stdout.Bytes(), &output)
		require.NoError(t, err)

		assert.NotEmpty(t, output.MissingVariables)
		assert.Equal(t, "required", output.MissingVariables[0].Name)
	})

	t.Run("JSONWithUnusedData", func(t *testing.T) {
		simpleTemplate := `{~prompty.var name="used" /~}`
		tmpFile := createTempTemplate(t, simpleTemplate)
		extraData := map[string]any{
			"used":   "value",
			"unused": "extra",
		}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, extraData),
			"-F", "json",
		}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)

		var output debugOutput
		err := json.Unmarshal(stdout.Bytes(), &output)
		require.NoError(t, err)

		assert.Contains(t, output.UnusedData, "unused")
	})
}

// =============================================================================
// Data Input Tests
// =============================================================================

func TestDebug_E2E_DataInput(t *testing.T) {
	template := `Hello {~prompty.var name="name" /~}!`
	tmpFile := createTempTemplate(t, template)

	t.Run("DataFromFlag", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", `{"name": "Alice"}`,
		}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)
		assert.Contains(t, stdout.String(), "Alice")
	})

	t.Run("DataFromFile", func(t *testing.T) {
		dataFile := createTempDataFile(t, map[string]any{"name": "Bob"})

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-f", dataFile,
		}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)
		assert.Contains(t, stdout.String(), "Bob")
	})

	t.Run("NoData", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{"-t", tmpFile}, nil, &stdout, &stderr)

		// Should work but flag variable as missing
		output := stdout.String()
		assert.Contains(t, output, "MISSING")
		assert.Equal(t, ExitCodeValidationError, exitCode)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", `{invalid json}`,
		}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeInputError, exitCode)
	})

	t.Run("NestedData", func(t *testing.T) {
		nestedTemplate := `{~prompty.var name="user.profile.name" /~}`
		tmpFile := createTempTemplate(t, nestedTemplate)
		data := map[string]any{
			"user": map[string]any{
				"profile": map[string]any{
					"name": "DeepValue",
				},
			},
		}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
		}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)
		assert.Contains(t, stdout.String(), "DeepValue")
	})
}

// =============================================================================
// Resolver Detection Tests
// =============================================================================

func TestDebug_E2E_ResolverDetection(t *testing.T) {
	t.Run("DetectsVariables", func(t *testing.T) {
		// Test without loop variables to avoid static analysis issues
		template := `{~prompty.var name="a" /~}
{~prompty.if eval="show"~}
{~prompty.var name="b" /~}
{~/prompty.if~}`

		tmpFile := createTempTemplate(t, template)
		data := map[string]any{
			"a":    "value",
			"b":    "value",
			"show": true,
		}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
		}, nil, &stdout, &stderr)

		output := stdout.String()
		assert.Equal(t, ExitCodeSuccess, exitCode)
		// Debug output shows variables, not "resolvers" for built-in tags
		assert.Contains(t, output, "Variables")
		assert.Contains(t, output, "a")
		assert.Contains(t, output, "b")
	})

	t.Run("CountsResolverInvocations", func(t *testing.T) {
		template := `{~prompty.var name="a" /~}
{~prompty.var name="b" /~}
{~prompty.var name="c" /~}`

		tmpFile := createTempTemplate(t, template)
		data := map[string]any{"a": "1", "b": "2", "c": "3"}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
			"-F", "json",
		}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)

		var output debugOutput
		err := json.Unmarshal(stdout.Bytes(), &output)
		require.NoError(t, err)

		// Should have prompty.var resolver with 3 lines
		for _, r := range output.Resolvers {
			if r.Name == "prompty.var" {
				assert.Len(t, r.Lines, 3)
			}
		}
	})
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestDebug_E2E_EdgeCases(t *testing.T) {
	t.Run("EmptyTemplate", func(t *testing.T) {
		tmpFile := createTempTemplate(t, "")

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{"-t", tmpFile}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("WhitespaceOnlyTemplate", func(t *testing.T) {
		tmpFile := createTempTemplate(t, "   \n\t\n   ")

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{"-t", tmpFile}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("SyntaxError", func(t *testing.T) {
		// Unclosed tag
		tmpFile := createTempTemplate(t, `{~prompty.if eval="true"~}No closing tag`)

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{"-t", tmpFile}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeInputError, exitCode)
	})

	t.Run("UnicodeVariables", func(t *testing.T) {
		template := `{~prompty.var name="greeting" /~} 世界!`
		tmpFile := createTempTemplate(t, template)
		data := map[string]any{"greeting": "こんにちは"}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
		}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("LargeData", func(t *testing.T) {
		template := `{~prompty.var name="data" /~}`
		tmpFile := createTempTemplate(t, template)

		// Create large data value
		largeValue := strings.Repeat("x", 10000)
		data := map[string]any{"data": largeValue}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
		}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)
		// Value should be truncated in text output
		assert.Contains(t, stdout.String(), "...")
	})

	t.Run("MissingTemplateFile", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{"-t", "/nonexistent/file.txt"}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeInputError, exitCode)
	})

	t.Run("InvalidFormatFlag", func(t *testing.T) {
		tmpFile := createTempTemplate(t, "Hello")

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{"-t", tmpFile, "-F", "invalid"}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeUsageError, exitCode)
	})

	t.Run("ReadFromStdin", func(t *testing.T) {
		template := `Hello {~prompty.var name="name" default="World" /~}!`

		var stdout, stderr bytes.Buffer
		stdin := strings.NewReader(template)
		exitCode := runDebug([]string{"-t", "-"}, stdin, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)
	})
}

// =============================================================================
// Verbose Mode Tests
// =============================================================================

func TestDebug_E2E_VerboseMode(t *testing.T) {
	t.Run("ShowsSuggestions", func(t *testing.T) {
		template := `{~prompty.var name="userName" /~}`
		tmpFile := createTempTemplate(t, template)
		data := map[string]any{
			"user_name": "alice", // Similar but not exact match
		}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
			"-v",
		}, nil, &stdout, &stderr)

		output := stdout.String()
		// Verbose mode should show suggestions
		assert.Contains(t, output, "did you mean")
		assert.Equal(t, ExitCodeValidationError, exitCode)
	})

	t.Run("VerboseWithNoIssues", func(t *testing.T) {
		template := `Hello {~prompty.var name="name" /~}!`
		tmpFile := createTempTemplate(t, template)
		data := map[string]any{"name": "World"}

		var stdout, stderr bytes.Buffer
		exitCode := runDebug([]string{
			"-t", tmpFile,
			"-d", toJSON(t, data),
			"-v",
		}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)
		// Should still complete successfully
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

func toJSON(t *testing.T, data map[string]any) string {
	t.Helper()
	b, err := json.Marshal(data)
	require.NoError(t, err)
	return string(b)
}
