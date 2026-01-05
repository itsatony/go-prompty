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

func TestLint_ValidTemplate(t *testing.T) {
	tmpFile := createTempFile(t, "{~prompty.var name=\"userName\" default=\"Guest\" /~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runLint([]string{"-t", tmpFile}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeSuccess, code)
	assert.Contains(t, stdout.String(), LintTextNoIssues)
}

func TestLint_VariableNamingIssue(t *testing.T) {
	// VAR001: Variable with non-standard naming (uppercase)
	tmpFile := createTempFile(t, "{~prompty.var name=\"UserName\" default=\"Guest\" /~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runLint([]string{"-t", tmpFile}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeSuccess, code) // Warnings don't fail by default
	assert.Contains(t, stdout.String(), LintRuleVAR001)
}

func TestLint_MissingDefaultWarning(t *testing.T) {
	// VAR002: Variable without default
	tmpFile := createTempFile(t, "{~prompty.var name=\"missingVar\" /~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runLint([]string{"-t", tmpFile}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeSuccess, code) // Warnings don't fail by default
	assert.Contains(t, stdout.String(), LintRuleVAR002)
}

func TestLint_LoopWithoutLimit(t *testing.T) {
	// LOOP001: Loop without limit
	tmpFile := createTempFile(t, "{~prompty.for item=\"x\" in=\"items\"~}{~prompty.var name=\"x\" /~}{~/prompty.for~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runLint([]string{"-t", tmpFile}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeSuccess, code)
	assert.Contains(t, stdout.String(), LintRuleLOOP001)
}

func TestLint_LoopWithLimit(t *testing.T) {
	// Loop with limit should not trigger LOOP001
	tmpFile := createTempFile(t, "{~prompty.for item=\"x\" in=\"items\" limit=\"100\"~}{~prompty.var name=\"x\" /~}{~/prompty.for~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runLint([]string{"-t", tmpFile}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeSuccess, code)
	assert.NotContains(t, stdout.String(), LintRuleLOOP001)
}

func TestLint_DeeplyNestedLoops(t *testing.T) {
	// LOOP002: Deeply nested loops (> 2 levels)
	template := `
{~prompty.for item="a" in="level1" limit="10"~}
  {~prompty.for item="b" in="level2" limit="10"~}
    {~prompty.for item="c" in="level3" limit="10"~}
      {~prompty.var name="c" /~}
    {~/prompty.for~}
  {~/prompty.for~}
{~/prompty.for~}`
	tmpFile := createTempFile(t, template)
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runLint([]string{"-t", tmpFile}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeSuccess, code)
	assert.Contains(t, stdout.String(), LintRuleLOOP002)
}

func TestLint_StrictMode(t *testing.T) {
	// With --strict, warnings cause failure
	tmpFile := createTempFile(t, "{~prompty.var name=\"UserName\" default=\"Guest\" /~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runLint([]string{"-t", tmpFile, "--strict"}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeValidationError, code)
}

func TestLint_JSONFormat(t *testing.T) {
	tmpFile := createTempFile(t, "{~prompty.var name=\"userName\" default=\"Guest\" /~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runLint([]string{"-t", tmpFile, "-F", "json"}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeSuccess, code)
	assert.Contains(t, stdout.String(), "\"valid\"")
	assert.Contains(t, stdout.String(), "true")
}

func TestLint_IgnoreRule(t *testing.T) {
	// VAR001 issue but ignored
	tmpFile := createTempFile(t, "{~prompty.var name=\"UserName\" default=\"Guest\" /~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runLint([]string{"-t", tmpFile, "-i", "VAR001"}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeSuccess, code)
	assert.NotContains(t, stdout.String(), LintRuleVAR001)
}

func TestLint_SelectSpecificRules(t *testing.T) {
	// Only check VAR001
	tmpFile := createTempFile(t, "{~prompty.var name=\"missingVar\" /~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runLint([]string{"-t", tmpFile, "-r", "VAR001"}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeSuccess, code)
	assert.NotContains(t, stdout.String(), LintRuleVAR002) // VAR002 not enabled
}

func TestLint_FromStdin(t *testing.T) {
	stdin := strings.NewReader("{~prompty.var name=\"userName\" default=\"Guest\" /~}")
	var stdout, stderr bytes.Buffer
	code := runLint([]string{"-t", "-"}, stdin, &stdout, &stderr)

	assert.Equal(t, ExitCodeSuccess, code)
	assert.Contains(t, stdout.String(), LintTextNoIssues)
}

func TestLint_MissingTemplate(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runLint([]string{}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeUsageError, code)
}

func TestLint_TemplateNotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runLint([]string{"-t", "/nonexistent/template.txt"}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeInputError, code)
}

func TestLint_InvalidFormat(t *testing.T) {
	tmpFile := createTempFile(t, "{~prompty.var name=\"test\" /~}")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	code := runLint([]string{"-t", tmpFile, "-F", "invalid"}, nil, &stdout, &stderr)

	assert.Equal(t, ExitCodeUsageError, code)
}

func TestLint_InvalidTemplateSyntax(t *testing.T) {
	// TAG001: Invalid/unclosed tag
	tmpFile := createTempFile(t, "{~prompty.if eval=\"true\"~}unclosed")
	defer os.Remove(tmpFile)

	var stdout, stderr bytes.Buffer
	_ = runLint([]string{"-t", tmpFile}, nil, &stdout, &stderr)

	// Should report TAG001 error
	assert.Contains(t, stdout.String(), LintRuleTAG001)
}

func TestLintRuleSet(t *testing.T) {
	t.Run("DefaultAllEnabled", func(t *testing.T) {
		rs := newLintRuleSet("", "")
		assert.True(t, rs.isEnabled(LintRuleVAR001))
		assert.True(t, rs.isEnabled(LintRuleVAR002))
		assert.True(t, rs.isEnabled(LintRuleTAG001))
		assert.True(t, rs.isEnabled(LintRuleLOOP001))
		assert.True(t, rs.isEnabled(LintRuleLOOP002))
		assert.True(t, rs.isEnabled(LintRuleEXPR001))
		assert.True(t, rs.isEnabled(LintRuleINC001))
	})

	t.Run("SelectSpecificRules", func(t *testing.T) {
		rs := newLintRuleSet("VAR001,VAR002", "")
		assert.True(t, rs.isEnabled(LintRuleVAR001))
		assert.True(t, rs.isEnabled(LintRuleVAR002))
		assert.False(t, rs.isEnabled(LintRuleTAG001))
		assert.False(t, rs.isEnabled(LintRuleLOOP001))
	})

	t.Run("IgnoreRules", func(t *testing.T) {
		rs := newLintRuleSet("", "VAR001,LOOP001")
		assert.False(t, rs.isEnabled(LintRuleVAR001))
		assert.True(t, rs.isEnabled(LintRuleVAR002))
		assert.False(t, rs.isEnabled(LintRuleLOOP001))
		assert.True(t, rs.isEnabled(LintRuleLOOP002))
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		rs := newLintRuleSet("var001", "")
		assert.True(t, rs.isEnabled(LintRuleVAR001))
	})
}

func TestPositionFromOffset(t *testing.T) {
	source := "line1\nline2\nline3"

	t.Run("FirstLine", func(t *testing.T) {
		line, col := positionFromOffset(source, 0)
		assert.Equal(t, 1, line)
		assert.Equal(t, 1, col)
	})

	t.Run("SecondLine", func(t *testing.T) {
		line, col := positionFromOffset(source, 6) // After "line1\n"
		assert.Equal(t, 2, line)
		assert.Equal(t, 1, col)
	})

	t.Run("ThirdLineMiddle", func(t *testing.T) {
		line, col := positionFromOffset(source, 14) // "li" of "line3"
		assert.Equal(t, 3, line)
		assert.Equal(t, 3, col)
	})

	t.Run("InvalidOffset", func(t *testing.T) {
		line, col := positionFromOffset(source, -1)
		assert.Equal(t, 1, line)
		assert.Equal(t, 1, col)

		line, col = positionFromOffset(source, 100)
		assert.Equal(t, 1, line)
		assert.Equal(t, 1, col)
	})
}

func createTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_template.txt")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)
	return tmpFile
}
