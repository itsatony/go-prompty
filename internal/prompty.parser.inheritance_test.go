package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockNode(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	children := []Node{NewTextNode("content", pos)}

	block := NewBlockNode("header", children, pos)

	assert.Equal(t, "header", block.Name)
	assert.Len(t, block.Children, 1)
	assert.Equal(t, NodeTypeBlock, block.Type())
	assert.Equal(t, pos, block.Pos())
	assert.Contains(t, block.String(), "header")
}

func TestInheritanceInfo(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}
	info := NewInheritanceInfo("parent", pos)

	assert.Equal(t, "parent", info.ParentTemplate)
	assert.Empty(t, info.Blocks)
	assert.Equal(t, pos, info.ExtendsPos)

	// Add a block
	block := NewBlockNode("content", nil, pos)
	err := info.AddBlock(block)
	require.NoError(t, err)

	// Verify block was added
	assert.True(t, info.HasBlock("content"))
	retrieved, ok := info.GetBlock("content")
	assert.True(t, ok)
	assert.Equal(t, block, retrieved)

	// Duplicate block should fail
	err = info.AddBlock(block)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgBlockDuplicateName)
}

func TestParseBlock(t *testing.T) {
	source := `{~prompty.block name="header"~}Hello World{~/prompty.block~}`

	lexer := NewLexer(source, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParserWithSource(tokens, source, nil)
	root, err := parser.Parse()
	require.NoError(t, err)

	// Should have one BlockNode
	require.Len(t, root.Children, 1)

	block, ok := root.Children[0].(*BlockNode)
	require.True(t, ok, "expected BlockNode")
	assert.Equal(t, "header", block.Name)
	require.Len(t, block.Children, 1)

	textNode, ok := block.Children[0].(*TextNode)
	require.True(t, ok)
	assert.Equal(t, "Hello World", textNode.Content)
}

func TestParseBlock_MissingName(t *testing.T) {
	source := `{~prompty.block~}content{~/prompty.block~}`

	lexer := NewLexer(source, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParserWithSource(tokens, source, nil)
	_, err = parser.Parse()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgBlockMissingName)
}

func TestExtractInheritanceInfo(t *testing.T) {
	t.Run("no inheritance", func(t *testing.T) {
		source := `Hello {~prompty.var name="user" /~}`

		lexer := NewLexer(source, nil)
		tokens, err := lexer.Tokenize()
		require.NoError(t, err)

		parser := NewParserWithSource(tokens, source, nil)
		root, err := parser.Parse()
		require.NoError(t, err)

		info, err := ExtractInheritanceInfo(root)
		require.NoError(t, err)
		assert.Nil(t, info)
	})

	t.Run("with extends", func(t *testing.T) {
		source := `{~prompty.extends template="base" /~}
{~prompty.block name="content"~}Child content{~/prompty.block~}`

		lexer := NewLexer(source, nil)
		tokens, err := lexer.Tokenize()
		require.NoError(t, err)

		parser := NewParserWithSource(tokens, source, nil)
		root, err := parser.Parse()
		require.NoError(t, err)

		info, err := ExtractInheritanceInfo(root)
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.Equal(t, "base", info.ParentTemplate)
		assert.True(t, info.HasBlock("content"))
	})

	t.Run("extends not first", func(t *testing.T) {
		source := `Hello World{~prompty.extends template="base" /~}`

		lexer := NewLexer(source, nil)
		tokens, err := lexer.Tokenize()
		require.NoError(t, err)

		parser := NewParserWithSource(tokens, source, nil)
		root, err := parser.Parse()
		require.NoError(t, err)

		_, err = ExtractInheritanceInfo(root)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgExtendsNotFirst)
	})

	t.Run("multiple extends", func(t *testing.T) {
		source := `{~prompty.extends template="base1" /~}{~prompty.extends template="base2" /~}`

		lexer := NewLexer(source, nil)
		tokens, err := lexer.Tokenize()
		require.NoError(t, err)

		parser := NewParserWithSource(tokens, source, nil)
		root, err := parser.Parse()
		require.NoError(t, err)

		_, err = ExtractInheritanceInfo(root)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgExtendsMultiple)
	})

	t.Run("extends missing template attr", func(t *testing.T) {
		source := `{~prompty.extends /~}`

		lexer := NewLexer(source, nil)
		tokens, err := lexer.Tokenize()
		require.NoError(t, err)

		parser := NewParserWithSource(tokens, source, nil)
		root, err := parser.Parse()
		require.NoError(t, err)

		_, err = ExtractInheritanceInfo(root)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgExtendsMissingTemplate)
	})
}

func TestCollectBlocks(t *testing.T) {
	source := `{~prompty.block name="header"~}Header{~/prompty.block~}
Content
{~prompty.block name="footer"~}Footer{~/prompty.block~}`

	lexer := NewLexer(source, nil)
	tokens, err := lexer.Tokenize()
	require.NoError(t, err)

	parser := NewParserWithSource(tokens, source, nil)
	root, err := parser.Parse()
	require.NoError(t, err)

	blocks := CollectBlocks(root)
	assert.Len(t, blocks, 2)
	assert.Contains(t, blocks, "header")
	assert.Contains(t, blocks, "footer")
}

func TestIsFirstSignificantNode(t *testing.T) {
	pos := Position{Line: 1, Column: 1, Offset: 0}

	t.Run("empty", func(t *testing.T) {
		assert.True(t, isFirstSignificantNode(nil))
		assert.True(t, isFirstSignificantNode([]Node{}))
	})

	t.Run("whitespace only", func(t *testing.T) {
		nodes := []Node{
			NewTextNode("  \n\t  ", pos),
		}
		assert.True(t, isFirstSignificantNode(nodes))
	})

	t.Run("non-whitespace text", func(t *testing.T) {
		nodes := []Node{
			NewTextNode("Hello", pos),
		}
		assert.False(t, isFirstSignificantNode(nodes))
	})

	t.Run("tag node", func(t *testing.T) {
		nodes := []Node{
			NewSelfClosingTag("prompty.var", make(Attributes), pos),
		}
		assert.False(t, isFirstSignificantNode(nodes))
	})
}
