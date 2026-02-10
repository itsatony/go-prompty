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

// createTempTemplate creates a temporary template file for testing
func createTempTemplate(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "template.txt")
	err := os.WriteFile(tmpFile, []byte(content), FilePermissions)
	require.NoError(t, err)
	return tmpFile
}

// =============================================================================
// All Rules Tests
// =============================================================================

func TestLint_E2E_AllRules(t *testing.T) {
	t.Run("VAR001_NonStandardCasing", func(t *testing.T) {
		// Template with non-standard variable names (uppercase, weird casing)
		template := `Hello {~prompty.var name="UserName" /~}!
Welcome {~prompty.var name="ALLCAPS" /~}.
Your id is {~prompty.var name="User-ID" /~}.`

		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-r", "VAR001"}, nil, &stdout, &stderr)

		// Should find issues but not error (warnings only)
		assert.Equal(t, ExitCodeSuccess, exitCode)
		output := stdout.String()
		assert.Contains(t, output, "VAR001")
		assert.Contains(t, output, "UserName")
	})

	t.Run("VAR002_NoDefault", func(t *testing.T) {
		// Variables without default values
		template := `Hello {~prompty.var name="user" /~}!
Welcome {~prompty.var name="greeting" default="Hi" /~}.`

		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-r", "VAR002"}, nil, &stdout, &stderr)

		output := stdout.String()
		// "user" has no default, should trigger VAR002
		// "greeting" has default, should not trigger
		assert.Contains(t, output, "VAR002")
		assert.Contains(t, output, "user")
		assert.NotContains(t, output, "greeting")
		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("TAG001_UnknownTag", func(t *testing.T) {
		// Template with syntax error (missing required attribute)
		template := `{~prompty.if~}Missing eval{~/prompty.if~}`

		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-r", "TAG001"}, nil, &stdout, &stderr)

		// Parsing error should be reported as TAG001 error
		output := stdout.String()
		assert.Contains(t, output, "TAG001")
		assert.Equal(t, ExitCodeValidationError, exitCode)
	})

	t.Run("LOOP001_NoLimit", func(t *testing.T) {
		// Loop without limit attribute
		template := `{~prompty.for item="x" in="items"~}
{~prompty.var name="x" /~}
{~/prompty.for~}`

		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-r", "LOOP001"}, nil, &stdout, &stderr)

		output := stdout.String()
		assert.Contains(t, output, "LOOP001")
		assert.Contains(t, output, "limit")
		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("LOOP001_WithLimit", func(t *testing.T) {
		// Loop with limit attribute - should NOT trigger
		template := `{~prompty.for item="x" in="items" limit="100"~}
{~prompty.var name="x" /~}
{~/prompty.for~}`

		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-r", "LOOP001"}, nil, &stdout, &stderr)

		output := stdout.String()
		assert.NotContains(t, output, "LOOP001")
		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("LOOP002_DeeplyNested", func(t *testing.T) {
		// Deeply nested loops (> 2 levels)
		template := `{~prompty.for item="a" in="listA"~}
{~prompty.for item="b" in="listB"~}
{~prompty.for item="c" in="listC"~}
{~prompty.var name="c" /~}
{~/prompty.for~}
{~/prompty.for~}
{~/prompty.for~}`

		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-r", "LOOP002"}, nil, &stdout, &stderr)

		output := stdout.String()
		assert.Contains(t, output, "LOOP002")
		assert.Contains(t, output, "3 levels")
		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("EXPR001_ComplexExpression", func(t *testing.T) {
		// Complex expression with > 3 operators
		template := `{~prompty.if eval="a > 0 && b < 10 || c == 5 && d != 0 && e >= 1"~}
Complex condition
{~/prompty.if~}`

		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-r", "EXPR001"}, nil, &stdout, &stderr)

		output := stdout.String()
		assert.Contains(t, output, "EXPR001")
		assert.Contains(t, output, "operators")
		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("INC001_MissingInclude", func(t *testing.T) {
		// Include references non-existent template
		template := `{~prompty.include template="nonexistent-template" /~}`

		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-r", "INC001"}, nil, &stdout, &stderr)

		output := stdout.String()
		assert.Contains(t, output, "INC001")
		assert.Contains(t, output, "nonexistent-template")
		assert.Equal(t, ExitCodeSuccess, exitCode)
	})
}

// =============================================================================
// Rule Filtering Tests
// =============================================================================

func TestLint_E2E_RuleFiltering(t *testing.T) {
	// Template that triggers multiple rules
	template := `{~prompty.for item="x" in="items"~}
{~prompty.var name="UserName" /~}
{~prompty.var name="missing" /~}
{~/prompty.for~}`

	tmpFile := createTempTemplate(t, template)

	t.Run("EnableSpecificRules", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-r", "VAR001"}, nil, &stdout, &stderr)

		output := stdout.String()
		// Should only show VAR001 issues
		assert.Contains(t, output, "VAR001")
		assert.NotContains(t, output, "VAR002")
		assert.NotContains(t, output, "LOOP001")
		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("EnableMultipleRules", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-r", "VAR001,LOOP001"}, nil, &stdout, &stderr)

		output := stdout.String()
		assert.Contains(t, output, "VAR001")
		assert.Contains(t, output, "LOOP001")
		assert.NotContains(t, output, "VAR002")
		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("IgnoreSpecificRule", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-i", "VAR001"}, nil, &stdout, &stderr)

		output := stdout.String()
		// VAR001 should be ignored
		assert.NotContains(t, output, "VAR001")
		// But other rules should still apply
		assert.Contains(t, output, "LOOP001")
		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("IgnoreMultipleRules", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-i", "VAR001,VAR002,LOOP001"}, nil, &stdout, &stderr)

		output := stdout.String()
		assert.NotContains(t, output, "VAR001")
		assert.NotContains(t, output, "VAR002")
		assert.NotContains(t, output, "LOOP001")
		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("RulesWithWhitespace", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-r", " VAR001 , LOOP001 "}, nil, &stdout, &stderr)

		output := stdout.String()
		assert.Contains(t, output, "VAR001")
		assert.Contains(t, output, "LOOP001")
		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("CaseInsensitiveRules", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-r", "var001,loop001"}, nil, &stdout, &stderr)

		output := stdout.String()
		assert.Contains(t, output, "VAR001")
		assert.Contains(t, output, "LOOP001")
		assert.Equal(t, ExitCodeSuccess, exitCode)
	})
}

// =============================================================================
// Strict Mode Tests
// =============================================================================

func TestLint_E2E_StrictMode(t *testing.T) {
	// Template with warnings only (no errors)
	template := `{~prompty.for item="x" in="items"~}
{~prompty.var name="x" /~}
{~/prompty.for~}`

	tmpFile := createTempTemplate(t, template)

	t.Run("WarningsWithoutStrict", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile}, nil, &stdout, &stderr)

		// Without strict, warnings don't cause failure
		assert.Equal(t, ExitCodeSuccess, exitCode)
		assert.Contains(t, stdout.String(), "LOOP001")
	})

	t.Run("WarningsWithStrict", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "--strict"}, nil, &stdout, &stderr)

		// With strict, warnings cause failure
		assert.Equal(t, ExitCodeValidationError, exitCode)
		assert.Contains(t, stdout.String(), "LOOP001")
	})

	t.Run("NoIssuesWithStrict", func(t *testing.T) {
		// Clean template with no issues
		cleanTemplate := `{~prompty.for item="x" in="items" limit="100"~}
{~prompty.var name="x" default="default" /~}
{~/prompty.for~}`

		tmpFile := createTempTemplate(t, cleanTemplate)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "--strict"}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)
		assert.Contains(t, stdout.String(), "No lint issues found")
	})

	t.Run("ErrorsAlwaysFail", func(t *testing.T) {
		// Template with syntax error
		errorTemplate := `{~prompty.if~}Missing eval{~/prompty.if~}`

		tmpFile := createTempTemplate(t, errorTemplate)

		// Without strict
		var stdout1, stderr1 bytes.Buffer
		exitCode1 := runLint([]string{"-t", tmpFile}, nil, &stdout1, &stderr1)
		assert.Equal(t, ExitCodeValidationError, exitCode1)

		// With strict
		var stdout2, stderr2 bytes.Buffer
		exitCode2 := runLint([]string{"-t", tmpFile, "--strict"}, nil, &stdout2, &stderr2)
		assert.Equal(t, ExitCodeValidationError, exitCode2)
	})
}

// =============================================================================
// JSON Output Tests
// =============================================================================

func TestLint_E2E_JSONOutput(t *testing.T) {
	template := `{~prompty.for item="x" in="items"~}
{~prompty.var name="UserName" /~}
{~/prompty.for~}`

	tmpFile := createTempTemplate(t, template)

	t.Run("ValidJSON", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-F", "json"}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)

		// Should be valid JSON
		var output lintOutput
		err := json.Unmarshal(stdout.Bytes(), &output)
		require.NoError(t, err, "Output should be valid JSON")
	})

	t.Run("JSONStructure", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-F", "json"}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)

		var output lintOutput
		err := json.Unmarshal(stdout.Bytes(), &output)
		require.NoError(t, err)

		// Should have expected fields
		assert.True(t, output.Valid) // No errors, only warnings
		assert.NotEmpty(t, output.Issues)

		// Check issue structure
		for _, issue := range output.Issues {
			assert.NotEmpty(t, issue.RuleID)
			assert.NotEmpty(t, issue.Severity)
			assert.NotEmpty(t, issue.Message)
			assert.Greater(t, issue.Line, 0)
			assert.Greater(t, issue.Column, 0)
		}
	})

	t.Run("JSONWithErrors", func(t *testing.T) {
		// Use template with syntax error to get ERROR severity
		errorTemplate := `{~prompty.if~}Missing eval{~/prompty.if~}`
		tmpFile := createTempTemplate(t, errorTemplate)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-F", "json"}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeValidationError, exitCode)

		var output lintOutput
		err := json.Unmarshal(stdout.Bytes(), &output)
		require.NoError(t, err)

		assert.False(t, output.Valid)
		assert.NotEmpty(t, output.Issues)

		// At least one error
		hasError := false
		for _, issue := range output.Issues {
			if issue.Severity == "ERROR" {
				hasError = true
				break
			}
		}
		assert.True(t, hasError)
	})

	t.Run("JSONStrictMode", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-F", "json", "--strict"}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeValidationError, exitCode)

		var output lintOutput
		err := json.Unmarshal(stdout.Bytes(), &output)
		require.NoError(t, err)

		// With strict mode, warnings make it invalid
		assert.False(t, output.Valid)
	})

	t.Run("JSONNoIssues", func(t *testing.T) {
		cleanTemplate := `Hello {~prompty.var name="user" default="World" /~}!`
		tmpFile := createTempTemplate(t, cleanTemplate)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-F", "json"}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)

		var output lintOutput
		err := json.Unmarshal(stdout.Bytes(), &output)
		require.NoError(t, err)

		assert.True(t, output.Valid)
		assert.Empty(t, output.Issues)
	})
}

// =============================================================================
// Exit Codes Tests
// =============================================================================

func TestLint_E2E_ExitCodes(t *testing.T) {
	t.Run("ExitCode0_NoIssues", func(t *testing.T) {
		template := `Hello {~prompty.var name="user" default="World" /~}!`
		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("ExitCode0_WarningsOnly", func(t *testing.T) {
		template := `{~prompty.for item="x" in="items"~}{~prompty.var name="x" /~}{~/prompty.for~}`
		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile}, nil, &stdout, &stderr)

		// Warnings don't cause failure without strict mode
		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("ExitCode2_UsageError", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{}, nil, &stdout, &stderr)

		// Missing template should be usage error
		assert.Equal(t, ExitCodeUsageError, exitCode)
	})

	t.Run("ExitCode3_ValidationError", func(t *testing.T) {
		// Use a template with actual syntax error (not just warning)
		template := `{~prompty.if~}Missing eval attribute{~/prompty.if~}`
		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeValidationError, exitCode)
	})

	t.Run("ExitCode3_StrictWithWarnings", func(t *testing.T) {
		template := `{~prompty.for item="x" in="items"~}{~prompty.var name="x" /~}{~/prompty.for~}`
		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "--strict"}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeValidationError, exitCode)
	})

	t.Run("ExitCode4_InputError", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", "/nonexistent/file.txt"}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeInputError, exitCode)
	})
}

// =============================================================================
// File Operations Tests
// =============================================================================

func TestLint_E2E_RealFiles(t *testing.T) {
	t.Run("ReadFromFile", func(t *testing.T) {
		template := `Hello {~prompty.var name="user" /~}!`
		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("ReadFromStdin", func(t *testing.T) {
		template := `Hello {~prompty.var name="user" /~}!`

		var stdout, stderr bytes.Buffer
		stdin := strings.NewReader(template)
		exitCode := runLint([]string{"-t", "-"}, stdin, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("LargeFile", func(t *testing.T) {
		// Generate a large template
		var builder strings.Builder
		for i := 0; i < 1000; i++ {
			builder.WriteString(`{~prompty.var name="var` + string(rune('a'+i%26)) + `" default="default" /~}` + "\n")
		}

		tmpFile := createTempTemplate(t, builder.String())

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("UnicodeContent", func(t *testing.T) {
		template := `こんにちは {~prompty.var name="user" default="世界" /~}! 🎉`
		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("EmptyFile", func(t *testing.T) {
		tmpFile := createTempTemplate(t, "")

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile}, nil, &stdout, &stderr)

		// Empty file is valid
		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("WhitespaceOnlyFile", func(t *testing.T) {
		tmpFile := createTempTemplate(t, "   \n\t\n   ")

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeSuccess, exitCode)
	})
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestLint_E2E_EdgeCases(t *testing.T) {
	t.Run("MultipleIssuesSameLine", func(t *testing.T) {
		// Multiple issues on same line
		template := `{~prompty.var name="UserName" /~} {~prompty.var name="BadCase" /~}`

		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-r", "VAR001"}, nil, &stdout, &stderr)

		output := stdout.String()
		// Should find multiple VAR001 issues
		assert.Equal(t, strings.Count(output, "VAR001"), 2)
		assert.Equal(t, ExitCodeSuccess, exitCode)
	})

	t.Run("NestedTags", func(t *testing.T) {
		template := `{~prompty.if eval="show"~}
{~prompty.for item="x" in="items"~}
{~prompty.var name="x" /~}
{~/prompty.for~}
{~/prompty.if~}`

		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile}, nil, &stdout, &stderr)

		// Should not cause any errors
		assert.NotEqual(t, ExitCodeInputError, exitCode)
	})

	t.Run("InvalidFormatFlag", func(t *testing.T) {
		template := `Hello World`
		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-F", "invalid"}, nil, &stdout, &stderr)

		assert.Equal(t, ExitCodeUsageError, exitCode)
	})

	t.Run("ShortFlags", func(t *testing.T) {
		template := `{~prompty.for item="x" in="items"~}{~prompty.var name="x" /~}{~/prompty.for~}`
		tmpFile := createTempTemplate(t, template)

		var stdout, stderr bytes.Buffer
		exitCode := runLint([]string{"-t", tmpFile, "-r", "LOOP001", "-i", "VAR001,VAR002"}, nil, &stdout, &stderr)

		output := stdout.String()
		assert.Contains(t, output, "LOOP001")
		assert.NotContains(t, output, "VAR001")
		assert.NotContains(t, output, "VAR002")
		assert.Equal(t, ExitCodeSuccess, exitCode)
	})
}
