package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractConfigBlock_NoConfig(t *testing.T) {
	source := `Hello {~prompty.var name="user" /~}!`

	result, err := ExtractConfigBlock(source, DefaultLexerConfig())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.HasConfig)
	assert.Empty(t, result.ConfigJSON)
	assert.Equal(t, source, result.TemplateBody)
}

func TestExtractConfigBlock_BasicConfig(t *testing.T) {
	source := `{~prompty.config~}
{"name": "test"}
{~/prompty.config~}
Hello World!`

	result, err := ExtractConfigBlock(source, DefaultLexerConfig())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.HasConfig)
	assert.Equal(t, `{"name": "test"}`, result.ConfigJSON)
	assert.Equal(t, "Hello World!", result.TemplateBody)
}

func TestExtractConfigBlock_WithLeadingWhitespace(t *testing.T) {
	source := `
  {~prompty.config~}
{"name": "test"}
{~/prompty.config~}
Template body`

	result, err := ExtractConfigBlock(source, DefaultLexerConfig())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.HasConfig)
	assert.Equal(t, `{"name": "test"}`, result.ConfigJSON)
	assert.Equal(t, "Template body", result.TemplateBody)
}

func TestExtractConfigBlock_ComplexJSON(t *testing.T) {
	source := `{~prompty.config~}
{
  "name": "customer-support",
  "description": "A support agent",
  "model": {
    "api": "chat",
    "provider": "openai",
    "name": "gpt-4",
    "parameters": {
      "temperature": 0.7,
      "max_tokens": 2048
    }
  },
  "inputs": {
    "query": {"type": "string", "required": true}
  }
}
{~/prompty.config~}
Hello {~prompty.var name="user" /~}!`

	result, err := ExtractConfigBlock(source, DefaultLexerConfig())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.HasConfig)
	assert.Contains(t, result.ConfigJSON, `"name": "customer-support"`)
	assert.Contains(t, result.ConfigJSON, `"temperature": 0.7`)
	assert.Equal(t, `Hello {~prompty.var name="user" /~}!`, result.TemplateBody)
}

func TestExtractConfigBlock_UnclosedError(t *testing.T) {
	source := `{~prompty.config~}
{"name": "test"}
Missing close tag
Hello World!`

	result, err := ExtractConfigBlock(source, DefaultLexerConfig())
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), ErrMsgConfigBlockUnclosed)
}

func TestExtractConfigBlock_EmptyConfig(t *testing.T) {
	source := `{~prompty.config~}
{}
{~/prompty.config~}
Template`

	result, err := ExtractConfigBlock(source, DefaultLexerConfig())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.HasConfig)
	assert.Equal(t, "{}", result.ConfigJSON)
	assert.Equal(t, "Template", result.TemplateBody)
}

func TestExtractConfigBlock_NoTemplateBody(t *testing.T) {
	source := `{~prompty.config~}
{"name": "test"}
{~/prompty.config~}`

	result, err := ExtractConfigBlock(source, DefaultLexerConfig())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.HasConfig)
	assert.Equal(t, `{"name": "test"}`, result.ConfigJSON)
	assert.Empty(t, result.TemplateBody)
}

func TestExtractConfigBlock_ConfigInMiddle(t *testing.T) {
	// Config block that appears after template content should NOT be extracted
	source := `Template start
{~prompty.config~}
{"name": "test"}
{~/prompty.config~}
Template end`

	result, err := ExtractConfigBlock(source, DefaultLexerConfig())
	require.NoError(t, err)
	require.NotNil(t, result)

	// Config block in middle is not extracted
	assert.False(t, result.HasConfig)
	assert.Empty(t, result.ConfigJSON)
	assert.Equal(t, source, result.TemplateBody)
}

func TestExtractConfigBlock_EmptySource(t *testing.T) {
	result, err := ExtractConfigBlock("", DefaultLexerConfig())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.HasConfig)
	assert.Empty(t, result.ConfigJSON)
	assert.Empty(t, result.TemplateBody)
}

func TestExtractConfigBlock_OnlyWhitespace(t *testing.T) {
	result, err := ExtractConfigBlock("   \n\t  ", DefaultLexerConfig())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.HasConfig)
}

func TestExtractConfigBlock_WithTemplateVarsInJSON(t *testing.T) {
	// This tests that prompty tags inside the config JSON are preserved
	source := `{~prompty.config~}
{
  "model": {
    "name": "{~prompty.var name=\"model_name\" default=\"gpt-4\" /~}"
  }
}
{~/prompty.config~}
Template`

	result, err := ExtractConfigBlock(source, DefaultLexerConfig())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.HasConfig)
	assert.Contains(t, result.ConfigJSON, `{~prompty.var name=\"model_name\" default=\"gpt-4\" /~}`)
}

func TestExtractConfigBlock_Position(t *testing.T) {
	source := `{~prompty.config~}
{}
{~/prompty.config~}
Body`

	result, err := ExtractConfigBlock(source, DefaultLexerConfig())
	require.NoError(t, err)
	require.NotNil(t, result)

	// Config starts at line 1, column 1
	assert.Equal(t, 1, result.ConfigPosition.Line)
	assert.Equal(t, 1, result.ConfigPosition.Column)
	assert.Equal(t, 0, result.ConfigPosition.Offset)
}

func TestExtractConfigBlock_PositionWithLeadingNewlines(t *testing.T) {
	source := "\n\n{~prompty.config~}\n{}\n{~/prompty.config~}\nBody"

	result, err := ExtractConfigBlock(source, DefaultLexerConfig())
	require.NoError(t, err)
	require.NotNil(t, result)

	// Config starts after 2 newlines
	assert.Equal(t, 3, result.ConfigPosition.Line)
	assert.Equal(t, 1, result.ConfigPosition.Column)
}

func TestExtractConfigBlock_CarriageReturnNewline(t *testing.T) {
	source := "{~prompty.config~}\r\n{}\r\n{~/prompty.config~}\r\nTemplate"

	result, err := ExtractConfigBlock(source, DefaultLexerConfig())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.HasConfig)
	assert.Equal(t, "{}", result.ConfigJSON)
	assert.Equal(t, "Template", result.TemplateBody)
}

func TestCalculatePosition(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		want   Position
	}{
		{
			name:   "empty",
			prefix: "",
			want:   Position{Offset: 0, Line: 1, Column: 1},
		},
		{
			name:   "single line",
			prefix: "hello",
			want:   Position{Offset: 5, Line: 1, Column: 6},
		},
		{
			name:   "two lines",
			prefix: "hello\nworld",
			want:   Position{Offset: 11, Line: 2, Column: 6},
		},
		{
			name:   "three lines",
			prefix: "a\nb\nc",
			want:   Position{Offset: 5, Line: 3, Column: 2},
		},
		{
			name:   "trailing newline",
			prefix: "hello\n",
			want:   Position{Offset: 6, Line: 2, Column: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculatePosition(tt.prefix)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfigError_Error(t *testing.T) {
	err := NewConfigError(ErrMsgConfigBlockUnclosed, Position{Line: 5, Column: 10}, nil)
	assert.Contains(t, err.Error(), ErrMsgConfigBlockUnclosed)
	assert.Contains(t, err.Error(), "line 5")
	assert.Contains(t, err.Error(), "column 10")
}

func TestConfigError_Unwrap(t *testing.T) {
	cause := &ConfigError{Message: "inner error"}
	err := NewConfigError("outer", Position{}, cause)
	assert.Equal(t, cause, err.Unwrap())
}
