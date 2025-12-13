package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_ParsePlainText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple text",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "text with newlines",
			input:    "Line 1\nLine 2\nLine 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   \t\n   ",
			expected: "   \t\n   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)

			parser := NewParser(tokens, nil)
			ast, err := parser.Parse()
			require.NoError(t, err)
			require.NotNil(t, ast)

			if tt.expected == "" {
				assert.Empty(t, ast.Children)
			} else {
				require.Len(t, ast.Children, 1)
				textNode, ok := ast.Children[0].(*TextNode)
				require.True(t, ok, "expected TextNode")
				assert.Equal(t, tt.expected, textNode.Content)
			}
		})
	}
}

func TestParser_ParseSelfClosingTag(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedTag   string
		expectedAttrs map[string]string
	}{
		{
			name:        "simple self-closing tag",
			input:       `{~prompty.var name="user" /~}`,
			expectedTag: TagNameVar,
			expectedAttrs: map[string]string{
				"name": "user",
			},
		},
		{
			name:        "tag with multiple attributes",
			input:       `{~prompty.var name="user" default="Guest" /~}`,
			expectedTag: TagNameVar,
			expectedAttrs: map[string]string{
				"name":    "user",
				"default": "Guest",
			},
		},
		{
			name:          "tag without attributes",
			input:         `{~MyTag /~}`,
			expectedTag:   "MyTag",
			expectedAttrs: map[string]string{},
		},
		{
			name:        "tag with hyphenated name",
			input:       `{~my-custom-tag attr="value" /~}`,
			expectedTag: "my-custom-tag",
			expectedAttrs: map[string]string{
				"attr": "value",
			},
		},
		{
			name:        "tag with dotted name",
			input:       `{~my.namespaced.tag attr="value" /~}`,
			expectedTag: "my.namespaced.tag",
			expectedAttrs: map[string]string{
				"attr": "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)

			parser := NewParser(tokens, nil)
			ast, err := parser.Parse()
			require.NoError(t, err)
			require.NotNil(t, ast)

			require.Len(t, ast.Children, 1)
			tagNode, ok := ast.Children[0].(*TagNode)
			require.True(t, ok, "expected TagNode")

			assert.Equal(t, tt.expectedTag, tagNode.Name)
			assert.True(t, tagNode.SelfClose)
			assert.Nil(t, tagNode.Children)

			for key, expectedVal := range tt.expectedAttrs {
				actualVal, exists := tagNode.Attributes.Get(key)
				assert.True(t, exists, "attribute %s should exist", key)
				assert.Equal(t, expectedVal, actualVal)
			}
		})
	}
}

func TestParser_ParseBlockTag(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedTag      string
		expectedChildren int
	}{
		{
			name:             "simple block tag",
			input:            `{~section~}Content{~/section~}`,
			expectedTag:      "section",
			expectedChildren: 1, // TextNode
		},
		{
			name:             "empty block tag",
			input:            `{~section~}{~/section~}`,
			expectedTag:      "section",
			expectedChildren: 0,
		},
		{
			name:             "block tag with multiline content",
			input:            "{~section~}\nLine 1\nLine 2\n{~/section~}",
			expectedTag:      "section",
			expectedChildren: 1, // TextNode with newlines
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)

			parser := NewParser(tokens, nil)
			ast, err := parser.Parse()
			require.NoError(t, err)
			require.NotNil(t, ast)

			require.Len(t, ast.Children, 1)
			tagNode, ok := ast.Children[0].(*TagNode)
			require.True(t, ok, "expected TagNode")

			assert.Equal(t, tt.expectedTag, tagNode.Name)
			assert.False(t, tagNode.SelfClose)
			assert.Len(t, tagNode.Children, tt.expectedChildren)
		})
	}
}

func TestParser_ParseNestedTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "nested block tags",
			input: `{~outer~}{~inner~}Content{~/inner~}{~/outer~}`,
		},
		{
			name:  "deeply nested tags",
			input: `{~a~}{~b~}{~c~}Deep{~/c~}{~/b~}{~/a~}`,
		},
		{
			name:  "nested with text between",
			input: `{~outer~}Before{~inner~}Inside{~/inner~}After{~/outer~}`,
		},
		{
			name:  "self-closing inside block",
			input: `{~outer~}{~prompty.var name="x" /~}{~/outer~}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)

			parser := NewParser(tokens, nil)
			ast, err := parser.Parse()
			require.NoError(t, err)
			require.NotNil(t, ast)

			// Just verify it parses without error
			assert.NotEmpty(t, ast.Children)
		})
	}
}

func TestParser_ParseRawBlock(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedContent string
	}{
		{
			name:            "raw block with plain text",
			input:           `{~prompty.raw~}This is raw content{~/prompty.raw~}`,
			expectedContent: "This is raw content",
		},
		{
			name:            "raw block preserves inner tags",
			input:           `{~prompty.raw~}{~prompty.var name="x" /~}{~/prompty.raw~}`,
			expectedContent: `{~prompty.var name="x" /~}`,
		},
		{
			name:            "raw block with newlines",
			input:           "{~prompty.raw~}\nLine 1\nLine 2\n{~/prompty.raw~}",
			expectedContent: "\nLine 1\nLine 2\n",
		},
		{
			name:            "empty raw block",
			input:           `{~prompty.raw~}{~/prompty.raw~}`,
			expectedContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)

			parser := NewParser(tokens, nil)
			ast, err := parser.Parse()
			require.NoError(t, err)
			require.NotNil(t, ast)

			require.Len(t, ast.Children, 1)
			tagNode, ok := ast.Children[0].(*TagNode)
			require.True(t, ok, "expected TagNode")

			assert.Equal(t, TagNameRaw, tagNode.Name)
			assert.True(t, tagNode.IsRaw())
			assert.Equal(t, tt.expectedContent, tagNode.RawContent)
		})
	}
}

func TestParser_ParseMixedContent(t *testing.T) {
	input := `Hello {~prompty.var name="user" /~}, welcome to {~section~}the app{~/section~}!`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Should have: Text, Tag, Text, Tag, Text
	require.Len(t, ast.Children, 5)

	// First: "Hello "
	text0, ok := ast.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Hello ", text0.Content)

	// Second: prompty.var tag
	tag1, ok := ast.Children[1].(*TagNode)
	require.True(t, ok)
	assert.Equal(t, TagNameVar, tag1.Name)
	assert.True(t, tag1.SelfClose)

	// Third: ", welcome to "
	text2, ok := ast.Children[2].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, ", welcome to ", text2.Content)

	// Fourth: section block tag
	tag3, ok := ast.Children[3].(*TagNode)
	require.True(t, ok)
	assert.Equal(t, "section", tag3.Name)
	assert.False(t, tag3.SelfClose)
	require.Len(t, tag3.Children, 1)

	// Fifth: "!"
	text4, ok := ast.Children[4].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "!", text4.Content)
}

func TestParser_ParseEscapedDelimiter(t *testing.T) {
	input := `Use \{~ for literal delimiters`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Should produce: "Use " + "{~" + " for literal delimiters"
	// But the lexer handles this, so parser just sees text nodes
	require.NotEmpty(t, ast.Children)
}

func TestParser_Errors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		errContains string
	}{
		{
			name:        "mismatched closing tag",
			input:       `{~outer~}Content{~/inner~}`,
			errContains: ErrMsgMismatchedTag,
		},
		{
			name:        "unclosed block tag",
			input:       `{~outer~}Content`,
			errContains: ErrMsgMismatchedTag,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input, nil)
			tokens, err := lexer.Tokenize()
			require.NoError(t, err)

			parser := NewParser(tokens, nil)
			_, err = parser.Parse()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestParser_PositionTracking(t *testing.T) {
	input := "Text\n{~tag /~}"

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, ast)

	require.Len(t, ast.Children, 2)

	// Text node at line 1
	textNode := ast.Children[0].(*TextNode)
	assert.Equal(t, 1, textNode.Pos().Line)
	assert.Equal(t, 1, textNode.Pos().Column)

	// Tag node at line 2
	tagNode := ast.Children[1].(*TagNode)
	assert.Equal(t, 2, tagNode.Pos().Line)
	assert.Equal(t, 1, tagNode.Pos().Column)
}

func TestParser_AttributeTypes(t *testing.T) {
	input := `{~tag name="value" count="42" flag="true" /~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, ast)

	require.Len(t, ast.Children, 1)
	tagNode := ast.Children[0].(*TagNode)

	// All attributes stored as strings
	name, ok := tagNode.Attributes.Get("name")
	assert.True(t, ok)
	assert.Equal(t, "value", name)

	count, ok := tagNode.Attributes.Get("count")
	assert.True(t, ok)
	assert.Equal(t, "42", count)

	flag, ok := tagNode.Attributes.Get("flag")
	assert.True(t, ok)
	assert.Equal(t, "true", flag)
}

func TestParser_AttributeMethods(t *testing.T) {
	input := `{~tag name="value" /~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	tagNode := ast.Children[0].(*TagNode)
	attrs := tagNode.Attributes

	// Test Get
	val, ok := attrs.Get("name")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	// Test Get with non-existent key
	_, ok = attrs.Get("nonexistent")
	assert.False(t, ok)

	// Test GetDefault
	assert.Equal(t, "value", attrs.GetDefault("name", "default"))
	assert.Equal(t, "default", attrs.GetDefault("nonexistent", "default"))

	// Test Has
	assert.True(t, attrs.Has("name"))
	assert.False(t, attrs.Has("nonexistent"))

	// Test Keys
	keys := attrs.Keys()
	assert.Contains(t, keys, "name")
}

func TestParser_ComplexTemplate(t *testing.T) {
	input := `{~prompty.raw~}
<system>You are a helpful assistant.</system>
{~/prompty.raw~}

<user>Hello, {~prompty.var name="user.name" default="Guest" /~}!</user>

{~section role="assistant"~}
I can help you with {~prompty.var name="task" /~}.
{~/section~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Should parse without error - complex templates work
	assert.NotEmpty(t, ast.Children)
}

func TestParser_RootNodeMethods(t *testing.T) {
	input := "Hello"

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	assert.Equal(t, NodeTypeRoot, ast.Type())
	assert.Equal(t, 1, ast.Pos().Line)
	assert.Contains(t, ast.String(), "RootNode")
}

func TestParser_TagNodeMethods(t *testing.T) {
	input := `{~prompty.var name="x" /~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	tagNode := ast.Children[0].(*TagNode)

	assert.Equal(t, NodeTypeTag, tagNode.Type())
	assert.True(t, tagNode.IsBuiltin())
	assert.False(t, tagNode.IsRaw())
	assert.Contains(t, tagNode.String(), TagNameVar)
}

func TestParser_TextNodeMethods(t *testing.T) {
	input := "Hello, World!"

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	textNode := ast.Children[0].(*TextNode)

	assert.Equal(t, NodeTypeText, textNode.Type())
	assert.Contains(t, textNode.String(), "Hello")
}

func TestParser_LongTextTruncation(t *testing.T) {
	// Test that TextNode.String() truncates long content
	longContent := "This is a very long string that should be truncated when displayed in the String() method because it exceeds fifty characters"

	node := NewTextNode(longContent, Position{Line: 1, Column: 1})
	str := node.String()

	// Should contain "..." for truncation
	assert.Contains(t, str, "...")
	// Should not contain the full string
	assert.Less(t, len(str), len(longContent)+50) // Allow for formatting overhead
}

func TestParser_NilAttributesMethods(t *testing.T) {
	var attrs Attributes = nil

	// Get returns empty and false
	val, ok := attrs.Get("key")
	assert.Equal(t, "", val)
	assert.False(t, ok)

	// GetDefault returns default
	assert.Equal(t, "default", attrs.GetDefault("key", "default"))

	// Has returns false
	assert.False(t, attrs.Has("key"))

	// Keys returns nil
	assert.Nil(t, attrs.Keys())

	// Map returns empty map
	m := attrs.Map()
	assert.NotNil(t, m)
	assert.Empty(t, m)

	// String returns empty braces
	assert.Equal(t, "{}", attrs.String())
}

func TestParser_AttributesMap(t *testing.T) {
	attrs := Attributes{
		"name":  "value",
		"count": "42",
	}

	m := attrs.Map()
	assert.Equal(t, "value", m["name"])
	assert.Equal(t, "42", m["count"])

	// Modifying the copy shouldn't affect original
	m["name"] = "modified"
	origVal, _ := attrs.Get("name")
	assert.Equal(t, "value", origVal)
}

func TestParser_ConsecutiveTags(t *testing.T) {
	input := `{~a /~}{~b /~}{~c /~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	require.Len(t, ast.Children, 3)

	names := []string{"a", "b", "c"}
	for i, name := range names {
		tagNode, ok := ast.Children[i].(*TagNode)
		require.True(t, ok)
		assert.Equal(t, name, tagNode.Name)
	}
}

func TestParser_BlockTagWithAttributes(t *testing.T) {
	input := `{~section role="system" priority="high"~}Content here{~/section~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	require.Len(t, ast.Children, 1)
	tagNode := ast.Children[0].(*TagNode)

	assert.Equal(t, "section", tagNode.Name)
	assert.False(t, tagNode.SelfClose)

	role, ok := tagNode.Attributes.Get("role")
	assert.True(t, ok)
	assert.Equal(t, "system", role)

	priority, ok := tagNode.Attributes.Get("priority")
	assert.True(t, ok)
	assert.Equal(t, "high", priority)
}

func TestParser_SingleQuoteAttributes(t *testing.T) {
	input := `{~tag name='single quoted' /~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	tagNode := ast.Children[0].(*TagNode)
	name, ok := tagNode.Attributes.Get("name")
	assert.True(t, ok)
	assert.Equal(t, "single quoted", name)
}

func TestParser_EscapedQuotesInAttributes(t *testing.T) {
	input := `{~tag name="value with \"quotes\"" /~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	tagNode := ast.Children[0].(*TagNode)
	name, ok := tagNode.Attributes.Get("name")
	assert.True(t, ok)
	assert.Equal(t, `value with "quotes"`, name)
}

func TestParser_NestedRawBlockError(t *testing.T) {
	// Nested raw blocks are disallowed - this tests that error path
	// However, since the lexer tokenizes everything, and the parser tracks inRawBlock,
	// we need to construct a scenario where a raw block is detected inside another.
	// Currently, the raw block parser collects tokens until {~/prompty.raw~},
	// so inner {~prompty.raw~} would be collected as text.
	// The inRawBlock flag protects against recursive parseRawBlock calls.

	// This test verifies the raw block parsing logic works correctly
	input := `{~prompty.raw~}Outer content{~/prompty.raw~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, ast)

	tagNode := ast.Children[0].(*TagNode)
	assert.Equal(t, TagNameRaw, tagNode.Name)
	assert.Equal(t, "Outer content", tagNode.RawContent)
}

func TestParser_RawBlockWithNestedTags(t *testing.T) {
	// Raw block preserves inner tag syntax as literal text
	input := `{~prompty.raw~}{~inner~}content{~/inner~}{~/prompty.raw~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)
	require.NotNil(t, ast)

	tagNode := ast.Children[0].(*TagNode)
	assert.Equal(t, TagNameRaw, tagNode.Name)
	// The inner tags are preserved as literal text
	assert.Contains(t, tagNode.RawContent, "{~inner~}")
	assert.Contains(t, tagNode.RawContent, "{~/inner~}")
}

func TestParser_RawBlockWithSelfClosingInside(t *testing.T) {
	// Raw block preserves self-closing tags as text
	input := `{~prompty.raw~}Before {~tag attr="val" /~} After{~/prompty.raw~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	tagNode := ast.Children[0].(*TagNode)
	assert.Contains(t, tagNode.RawContent, "{~tag")
	assert.Contains(t, tagNode.RawContent, "/~}")
}

func TestParser_EmptyAttributes(t *testing.T) {
	attrs := Attributes{}

	// Empty attributes
	assert.Equal(t, "{}", attrs.String())
	keys := attrs.Keys()
	assert.NotNil(t, keys)
	assert.Empty(t, keys)
}

func TestParser_AttributesKeysOrder(t *testing.T) {
	attrs := Attributes{
		"zebra":  "z",
		"apple":  "a",
		"middle": "m",
	}

	keys := attrs.Keys()
	// Keys should be sorted alphabetically
	assert.Equal(t, []string{"apple", "middle", "zebra"}, keys)
}

func TestParser_AttributesStringFormat(t *testing.T) {
	attrs := Attributes{
		"name": "value",
	}

	str := attrs.String()
	assert.Contains(t, str, "name=")
	assert.Contains(t, str, "\"value\"")
}

func TestParser_TagNodeTypeRaw(t *testing.T) {
	input := `{~prompty.raw~}content{~/prompty.raw~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	tagNode := ast.Children[0].(*TagNode)
	// Raw blocks should return NodeTypeRaw
	assert.Equal(t, NodeTypeRaw, tagNode.Type())
}

func TestParser_TagNodeStringBlockForm(t *testing.T) {
	input := `{~section~}content{~/section~}`

	lexer := NewLexer(input, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParser(tokens, nil)
	ast, err := parser.Parse()
	require.NoError(t, err)

	tagNode := ast.Children[0].(*TagNode)
	str := tagNode.String()
	assert.Contains(t, str, "block")
	assert.Contains(t, str, "children=1")
}

func TestParser_ParserErrorString(t *testing.T) {
	err := &ParserError{
		Message: ErrMsgMismatchedTag,
		Position: Position{
			Line:   5,
			Column: 10,
		},
	}

	errStr := err.Error()
	assert.Contains(t, errStr, ErrMsgMismatchedTag)
	assert.Contains(t, errStr, "5")
}
