package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebug_SimpleTemplate(t *testing.T) {
	tmpFile := createDebugTempFile(t, "{~prompty.var name=\"userName\" default=\"Guest\" /~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runDebug([]string{"-t", tmpFile}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeSuccess, code)
	assert.Contains(t, stdout.String(), "Variables")
	assert.Contains(t, stdout.String(), "userName")
}

func TestDebug_WithData(t *testing.T) {
	tmpFile := createDebugTempFile(t, "{~prompty.var name=\"userName\" /~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runDebug([]string{"-t", tmpFile, "-d", `{"userName":"Alice"}`}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeSuccess, code)
	assert.Contains(t, stdout.String(), "userName")
	assert.Contains(t, stdout.String(), "Alice")
}

func TestDebug_WithDataFile(t *testing.T) {
	tmpTemplate := createDebugTempFile(t, "{~prompty.var name=\"userName\" /~}")
	defer os.Remove(tmpTemplate)

	tmpData := filepath.Join(os.TempDir(), "test_data.json")
	err := os.WriteFile(tmpData, []byte(`{"userName":"Bob"}`), 0644)
	require.NoError(t, err)
	defer os.Remove(tmpData)

	var stdout, stderr bytes.Buffer
	code := runDebug([]string{"-t", tmpTemplate, "-f", tmpData}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeSuccess, code)
	assert.Contains(t, stdout.String(), "userName")
}

func TestDebug_MissingVariable(t *testing.T) {
	tmpFile := createDebugTempFile(t, "{~prompty.var name=\"missingVar\" /~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runDebug([]string{"-t", tmpFile}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeValidationError, code) // Missing vars cause validation error
	assert.Contains(t, stdout.String(), "MISSING")
}

func TestDebug_MissingVariableWithSuggestions(t *testing.T) {
	tmpFile := createDebugTempFile(t, "{~prompty.var name=\"usrName\" /~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	_ = runDebug([]string{"-t", tmpFile, "-d", `{"userName":"Alice"}`, "-v"}, nil, &stdout, &stderr)

	// Should show suggestions for similar variable names
	assert.Contains(t, stdout.String(), "Suggestions")
}

func TestDebug_UnusedData(t *testing.T) {
	tmpFile := createDebugTempFile(t, "{~prompty.var name=\"used\" default=\"default\" /~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runDebug([]string{"-t", tmpFile, "-d", `{"used":"value", "unused":"extra"}`}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeSuccess, code)
	assert.Contains(t, stdout.String(), "Unused")
	assert.Contains(t, stdout.String(), "unused")
}

func TestDebug_JSONFormat(t *testing.T) {
	tmpFile := createDebugTempFile(t, "{~prompty.var name=\"userName\" default=\"Guest\" /~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runDebug([]string{"-t", tmpFile, "-F", "json"}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeSuccess, code)
	assert.Contains(t, stdout.String(), "\"valid\"")
	assert.Contains(t, stdout.String(), "\"variables\"")
}

func TestDebug_VerboseMode(t *testing.T) {
	tmpFile := createDebugTempFile(t, "{~prompty.var name=\"userName\" /~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	_ = runDebug([]string{"-t", tmpFile, "-d", `{"user":"Alice"}`, "-v"}, nil, &stdout, &stderr)

	// Verbose mode should show suggestions for missing variables
	assert.Contains(t, stdout.String(), "Suggestions")
}

func TestDebug_FromStdin(t *testing.T) {
	stdin := strings.NewReader("{~prompty.var name=\"userName\" default=\"Guest\" /~}")
	var stdout, stderr bytes.Buffer
	code := runDebug([]string{"-t", "-"}, stdin, &stdout, &stderr)

	assert.Equal(t, ExitCodeSuccess, code)
	assert.Contains(t, stdout.String(), "Variables")
}

func TestDebug_MissingTemplate(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runDebug([]string{}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeUsageError, code)
}

func TestDebug_TemplateNotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runDebug([]string{"-t", "/nonexistent/template.txt"}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeInputError, code)
}

func TestDebug_InvalidFormat(t *testing.T) {
	tmpFile := createDebugTempFile(t, "{~prompty.var name=\"test\" /~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runDebug([]string{"-t", tmpFile, "-F", "invalid"}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeUsageError, code)
}

func TestDebug_InvalidJSON(t *testing.T) {
	tmpFile := createDebugTempFile(t, "{~prompty.var name=\"test\" /~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runDebug([]string{"-t", tmpFile, "-d", "not-json"}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeInputError, code)
}

func TestDebug_TemplateWithConditionals(t *testing.T) {
	template := `{~prompty.if eval="isAdmin"~}Admin{~/prompty.if~}`
	tmpFile := createDebugTempFile(t, template)
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runDebug([]string{"-t", tmpFile}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeSuccess, code)
	assert.Contains(t, stdout.String(), "Analysis")
}

func TestDebug_TemplateWithLoops(t *testing.T) {
	// Template with loop that references loop variable (x is created at runtime, not in data)
	// DryRun correctly reports x as missing since it's not in input data
	template := `{~prompty.for item="x" in="items" limit="10"~}{~prompty.var name="x" /~}{~/prompty.for~}`
	tmpFile := createDebugTempFile(t, template)
	defer os.Remove(tmpFile)

	// Provide data so loop source exists, but x will still be reported as missing
	// because DryRun checks against provided data, not runtime-created variables
	var stdout, stderr bytes.Buffer
	_ = runDebug([]string{"-t", tmpFile, "-d", `{"items":["a","b","c"]}`}, nil, &stdout, &stderr)

	// Verify it shows the variable info even if it reports as missing
	assert.Contains(t, stdout.String(), "Variables")
}

func TestDebug_TemplateWithResolvers(t *testing.T) {
	// Template with custom resolver that won't be registered but will show in analysis
	template := `Hello {~prompty.var name="name" default="World" /~}`
	tmpFile := createDebugTempFile(t, template)
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runDebug([]string{"-t", tmpFile}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeSuccess, code)
	assert.Contains(t, stdout.String(), "Variables")
}

func TestFindSimilarKeys(t *testing.T) {
	data := map[string]any{
		"userName":  "Alice",
		"userEmail": "alice@example.com",
		"userId":    123,
		"other":     "value",
	}

	t.Run("SimilarName", func(t *testing.T) {
		suggestions := findSimilarKeys("usrName", data)
		assert.Contains(t, suggestions, "userName")
	})

	t.Run("NoMatch", func(t *testing.T) {
		suggestions := findSimilarKeys("completelyDifferent", data)
		// May or may not have suggestions depending on distance - this is expected to return nil/empty
		// when there are no similar matches
		assert.Empty(t, suggestions)
	})

	t.Run("NilData", func(t *testing.T) {
		suggestions := findSimilarKeys("test", nil)
		assert.Nil(t, suggestions)
	})
}

func TestFlattenKeys(t *testing.T) {
	data := map[string]any{
		"user": map[string]any{
			"name":  "Alice",
			"email": "alice@example.com",
		},
		"count": 42,
	}

	keys := flattenKeys(data, "")

	assert.Contains(t, keys, "user")
	assert.Contains(t, keys, "user.name")
	assert.Contains(t, keys, "user.email")
	assert.Contains(t, keys, "count")
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"kitten", "sitting", 3},
	}

	for _, tc := range tests {
		t.Run(tc.a+"_"+tc.b, func(t *testing.T) {
			result := levenshteinDistance(tc.a, tc.b)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetValueFromData(t *testing.T) {
	data := map[string]any{
		"name": "Alice",
		"user": map[string]any{
			"email": "alice@example.com",
		},
	}

	t.Run("TopLevel", func(t *testing.T) {
		val := getValueFromData(data, "name")
		assert.Equal(t, "Alice", val)
	})

	t.Run("Nested", func(t *testing.T) {
		val := getValueFromData(data, "user.email")
		assert.Equal(t, "alice@example.com", val)
	})

	t.Run("NotFound", func(t *testing.T) {
		val := getValueFromData(data, "nonexistent")
		assert.Nil(t, val)
	})

	t.Run("NilData", func(t *testing.T) {
		val := getValueFromData(nil, "name")
		assert.Nil(t, val)
	})

	t.Run("EmptyPath", func(t *testing.T) {
		val := getValueFromData(data, "")
		assert.Nil(t, val)
	})
}

func TestFormatIntSlice(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		assert.Equal(t, "", formatIntSlice(nil))
		assert.Equal(t, "", formatIntSlice([]int{}))
	})

	t.Run("Single", func(t *testing.T) {
		assert.Equal(t, "42", formatIntSlice([]int{42}))
	})

	t.Run("Multiple", func(t *testing.T) {
		assert.Equal(t, "1, 2, 3", formatIntSlice([]int{1, 2, 3}))
	})
}

func createDebugTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_debug_template.txt")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)
	return tmpFile
}
