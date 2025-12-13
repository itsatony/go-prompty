package internal

import (
	"fmt"

	"go.uber.org/zap"
)

// Parser produces an AST from a token stream
type Parser struct {
	tokens     []Token
	source     string // Original source for raw text extraction
	pos        int
	logger     *zap.Logger
	inRawBlock bool // Track if we're inside a raw block
}

// NewParser creates a new parser for the given token stream
func NewParser(tokens []Token, logger *zap.Logger) *Parser {
	return NewParserWithSource(tokens, StringValueEmpty, logger)
}

// NewParserWithSource creates a new parser with source for raw text capture
func NewParserWithSource(tokens []Token, source string, logger *zap.Logger) *Parser {
	if logger == nil {
		logger = zap.NewNop()
	}
	logger.Debug(LogMsgParserCreated, zap.Int(LogFieldTokens, len(tokens)))
	return &Parser{
		tokens:     tokens,
		source:     source,
		pos:        0,
		logger:     logger,
		inRawBlock: false,
	}
}

// extractRawSource extracts the original source text between two positions
func (p *Parser) extractRawSource(startOffset, endOffset int) string {
	if p.source == StringValueEmpty {
		return StringValueEmpty
	}
	if startOffset < 0 || endOffset > len(p.source) || startOffset >= endOffset {
		return StringValueEmpty
	}
	return p.source[startOffset:endOffset]
}

// Parse produces the AST root node from the token stream
func (p *Parser) Parse() (*RootNode, error) {
	p.logger.Debug(LogMsgParserStart)

	nodes, err := p.parseNodes()
	if err != nil {
		return nil, err
	}

	root := &RootNode{Children: nodes}
	p.logger.Debug(LogMsgParserEnd, zap.Int(LogFieldNodes, len(nodes)))
	return root, nil
}

// parseNodes parses a sequence of nodes until EOF or a closing tag
func (p *Parser) parseNodes() ([]Node, error) {
	var nodes []Node

	for !p.isAtEnd() && !p.isBlockClose() {
		node, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		if node != nil {
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

// parseNode parses a single node (text or tag)
func (p *Parser) parseNode() (Node, error) {
	tok := p.current()

	switch tok.Type {
	case TokenTypeText:
		return p.parseText()
	case TokenTypeOpenTag:
		return p.parseTag()
	case TokenTypeBlockClose:
		// Block close is handled by parseBlockTag
		return nil, nil
	case TokenTypeEOF:
		return nil, nil
	default:
		return nil, p.newUnexpectedTokenError(tok)
	}
}

// parseText parses a text node
func (p *Parser) parseText() (*TextNode, error) {
	tok := p.advance()
	return NewTextNode(tok.Value, tok.Position), nil
}

// parseTag parses a tag (self-closing or block)
func (p *Parser) parseTag() (Node, error) {
	openTok := p.advance() // consume OPEN_TAG

	// Get tag name
	nameTok := p.current()
	if nameTok.Type != TokenTypeTagName {
		return nil, p.newExpectedTokenError(TokenTypeTagName, nameTok)
	}
	p.advance() // consume TAG_NAME

	tagName := nameTok.Value
	pos := openTok.Position

	// Parse attributes
	attrs, err := p.parseAttributes()
	if err != nil {
		return nil, err
	}

	// Check how the tag ends
	endTok := p.current()

	switch endTok.Type {
	case TokenTypeSelfClose:
		p.advance() // consume SELF_CLOSE
		tag := NewSelfClosingTag(tagName, attrs, pos)
		// Capture raw source for keepRaw error strategy
		endOffset := endTok.Position.Offset + LenSelfClose
		tag.RawSource = p.extractRawSource(pos.Offset, endOffset)
		return tag, nil

	case TokenTypeCloseTag:
		p.advance() // consume CLOSE_TAG
		// This is a block tag - parse content and closing
		return p.parseBlockTag(tagName, attrs, pos, openTok)

	default:
		return nil, p.newUnexpectedTokenError(endTok)
	}
}

// parseBlockTag parses the content and closing of a block tag
func (p *Parser) parseBlockTag(tagName string, attrs Attributes, pos Position, openTok Token) (Node, error) {
	// Special handling for raw blocks
	if tagName == TagNameRaw {
		return p.parseRawBlock(pos, openTok)
	}

	// Special handling for conditionals
	if tagName == TagNameIf {
		return p.parseConditional(attrs, pos)
	}

	// Special handling for comments - discard content entirely
	if tagName == TagNameComment {
		return p.parseCommentBlock(pos, openTok)
	}

	// Parse children
	children, err := p.parseNodes()
	if err != nil {
		return nil, err
	}

	// Expect block close
	if !p.isBlockClose() {
		return nil, p.newMismatchedTagError(tagName, "")
	}

	// Consume BLOCK_CLOSE
	p.advance()

	// Get closing tag name
	closeNameTok := p.current()
	if closeNameTok.Type != TokenTypeTagName {
		return nil, p.newExpectedTokenError(TokenTypeTagName, closeNameTok)
	}
	closeName := closeNameTok.Value
	p.advance() // consume TAG_NAME

	// Verify matching
	if closeName != tagName {
		return nil, p.newMismatchedTagError(tagName, closeName)
	}

	// Consume CLOSE_TAG
	closeTok := p.current()
	if closeTok.Type != TokenTypeCloseTag {
		return nil, p.newExpectedTokenError(TokenTypeCloseTag, closeTok)
	}
	p.advance()

	tag := NewBlockTag(tagName, attrs, children, pos)
	// Capture raw source for keepRaw error strategy (full block from open to close)
	endOffset := closeTok.Position.Offset + LenCloseDelim
	tag.RawSource = p.extractRawSource(pos.Offset, endOffset)
	return tag, nil
}

// parseConditional parses an if/elseif/else conditional block
func (p *Parser) parseConditional(ifAttrs Attributes, pos Position) (*ConditionalNode, error) {
	var branches []ConditionalBranch

	// Get the condition from the if tag
	condition, ok := ifAttrs.Get(AttrEval)
	if !ok {
		return nil, p.newConditionError(ErrMsgCondMissingEval, pos)
	}

	// Parse the first branch (if)
	children, nextTag, nextAttrs, nextPos, err := p.parseConditionalBranch()
	if err != nil {
		return nil, err
	}

	branches = append(branches, NewConditionalBranch(condition, children, false, pos))

	// Process subsequent branches (elseif, else)
	for nextTag != "" {
		switch nextTag {
		case TagNameElseIf:
			// elseif needs an eval attribute
			condition, ok := nextAttrs.Get(AttrEval)
			if !ok {
				return nil, p.newConditionError(ErrMsgCondMissingEval, nextPos)
			}

			children, nextTag, nextAttrs, nextPos, err = p.parseConditionalBranch()
			if err != nil {
				return nil, err
			}

			branches = append(branches, NewConditionalBranch(condition, children, false, nextPos))

		case TagNameElse:
			// else cannot have an eval attribute
			if nextAttrs.Has(AttrEval) {
				return nil, p.newConditionError(ErrMsgCondInvalidElse, nextPos)
			}

			children, nextTag, nextAttrs, nextPos, err = p.parseConditionalBranch()
			if err != nil {
				return nil, err
			}

			// else must be the last branch
			if nextTag != "" {
				return nil, p.newConditionError(ErrMsgCondElseNotLast, nextPos)
			}

			branches = append(branches, NewConditionalBranch("", children, true, nextPos))

		default:
			// Unexpected tag inside conditional
			return nil, p.newConditionError(ErrMsgCondUnexpectedTag, nextPos)
		}
	}

	return NewConditionalNode(branches, pos), nil
}

// parseConditionalBranch parses nodes until we hit elseif, else, or the closing if tag
// Returns: children nodes, next tag name (empty if closing), next tag attrs, next tag position, error
func (p *Parser) parseConditionalBranch() ([]Node, string, Attributes, Position, error) {
	var children []Node

	for !p.isAtEnd() {
		tok := p.current()

		// Check for block close (could be elseif, else, or /if)
		if tok.Type == TokenTypeBlockClose {
			// This is {~/ - check if it's the closing /if
			if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == TokenTypeTagName {
				nextName := p.tokens[p.pos+1].Value
				if nextName == TagNameIf {
					// This is the closing {~/prompty.if~}
					p.advance() // consume BLOCK_CLOSE

					closeNameTok := p.current()
					p.advance() // consume TAG_NAME

					closeTok := p.current()
					if closeTok.Type != TokenTypeCloseTag {
						return nil, "", nil, Position{}, p.newExpectedTokenError(TokenTypeCloseTag, closeTok)
					}
					p.advance() // consume CLOSE_TAG

					return children, "", nil, closeNameTok.Position, nil
				}
			}
		}

		// Check for open tag (could be elseif or else, or a normal nested tag)
		if tok.Type == TokenTypeOpenTag {
			// Peek at the tag name
			if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == TokenTypeTagName {
				nextName := p.tokens[p.pos+1].Value
				if nextName == TagNameElseIf || nextName == TagNameElse {
					// This is a branch boundary
					openPos := tok.Position
					p.advance() // consume OPEN_TAG

					nameTok := p.current()
					p.advance() // consume TAG_NAME

					// Parse attributes
					attrs, err := p.parseAttributes()
					if err != nil {
						return nil, "", nil, Position{}, err
					}

					// Consume CLOSE_TAG
					closeTok := p.current()
					if closeTok.Type != TokenTypeCloseTag {
						return nil, "", nil, Position{}, p.newExpectedTokenError(TokenTypeCloseTag, closeTok)
					}
					p.advance()

					return children, nameTok.Value, attrs, openPos, nil
				}
			}
		}

		// Parse a normal node
		node, err := p.parseNode()
		if err != nil {
			return nil, "", nil, Position{}, err
		}
		if node != nil {
			children = append(children, node)
		}
	}

	// Reached EOF without finding closing tag
	return nil, "", nil, Position{}, p.newConditionError(ErrMsgCondNotClosed, Position{})
}

// newConditionError creates a conditional-specific error
func (p *Parser) newConditionError(message string, pos Position) error {
	return &ParserError{
		Message:  message,
		Position: pos,
	}
}

// parseRawBlock parses a raw block - preserving content literally
func (p *Parser) parseRawBlock(pos Position, openTok Token) (*TagNode, error) {
	// Check for nested raw blocks
	if p.inRawBlock {
		return nil, p.newNestedRawBlockError(pos)
	}

	p.inRawBlock = true
	defer func() { p.inRawBlock = false }()

	// For raw blocks, we need to collect all text and tokens until we see {~/prompty.raw~}
	// The lexer gives us tokens, but for raw content we want the original text
	// We'll reconstruct by collecting text tokens
	var rawContent string

	for !p.isAtEnd() {
		tok := p.current()

		// Check for the closing raw tag
		if tok.Type == TokenTypeBlockClose {
			// Peek at the next token to see if it's prompty.raw
			if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == TokenTypeTagName && p.tokens[p.pos+1].Value == TagNameRaw {
				// This is our closing tag
				break
			}
		}

		// For raw blocks, we only expect text (the lexer doesn't parse inside raw blocks for this to work properly)
		// However, our current lexer does tokenize inside blocks, so we need to handle this
		// For now, collect text tokens and reconstruct any tags we find
		switch tok.Type {
		case TokenTypeText:
			rawContent += tok.Value
			p.advance()
		case TokenTypeOpenTag:
			// Reconstruct the tag as literal text
			rawContent += StrOpenDelim
			p.advance()
			// Get tag content
			tagContent, err := p.collectTagAsText()
			if err != nil {
				return nil, err
			}
			rawContent += tagContent
		case TokenTypeBlockClose:
			rawContent += StrBlockClose
			p.advance()
			// Get tag content
			tagContent, err := p.collectTagCloseAsText()
			if err != nil {
				return nil, err
			}
			rawContent += tagContent
		default:
			return nil, p.newUnexpectedTokenError(tok)
		}
	}

	// Consume the closing sequence: BLOCK_CLOSE, TAG_NAME (prompty.raw), CLOSE_TAG
	if !p.isBlockClose() {
		return nil, p.newMismatchedTagError(TagNameRaw, "")
	}
	p.advance() // BLOCK_CLOSE

	closeNameTok := p.current()
	if closeNameTok.Type != TokenTypeTagName || closeNameTok.Value != TagNameRaw {
		return nil, p.newMismatchedTagError(TagNameRaw, closeNameTok.Value)
	}
	p.advance() // TAG_NAME

	closeTok := p.current()
	if closeTok.Type != TokenTypeCloseTag {
		return nil, p.newExpectedTokenError(TokenTypeCloseTag, closeTok)
	}
	p.advance() // CLOSE_TAG

	tag := NewRawBlockTag(rawContent, pos)
	// Capture raw source for keepRaw error strategy
	endOffset := closeTok.Position.Offset + LenCloseDelim
	tag.RawSource = p.extractRawSource(pos.Offset, endOffset)
	return tag, nil
}

// parseCommentBlock parses a comment block - content is discarded
func (p *Parser) parseCommentBlock(pos Position, openTok Token) (Node, error) {
	// Skip all tokens until we find the closing {~/prompty.comment~}
	for !p.isAtEnd() {
		tok := p.current()

		// Check for the closing comment tag
		if tok.Type == TokenTypeBlockClose {
			// Peek at the next token to see if it's prompty.comment
			if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == TokenTypeTagName && p.tokens[p.pos+1].Value == TagNameComment {
				// This is our closing tag
				break
			}
		}

		// Skip everything - comments are discarded
		p.advance()
	}

	// Consume the closing sequence: BLOCK_CLOSE, TAG_NAME (prompty.comment), CLOSE_TAG
	if !p.isBlockClose() {
		return nil, p.newMismatchedTagError(TagNameComment, "")
	}
	p.advance() // BLOCK_CLOSE

	closeNameTok := p.current()
	if closeNameTok.Type != TokenTypeTagName || closeNameTok.Value != TagNameComment {
		return nil, p.newMismatchedTagError(TagNameComment, closeNameTok.Value)
	}
	p.advance() // TAG_NAME

	closeTok := p.current()
	if closeTok.Type != TokenTypeCloseTag {
		return nil, p.newExpectedTokenError(TokenTypeCloseTag, closeTok)
	}
	p.advance() // CLOSE_TAG

	// Return nil - comment nodes produce no output
	return nil, nil
}

// collectTagAsText collects tag tokens and returns them as literal text (for raw blocks)
func (p *Parser) collectTagAsText() (string, error) {
	var result string

	// Tag name
	if p.current().Type == TokenTypeTagName {
		result += p.current().Value
		p.advance()
	}

	// Attributes and closing
	for !p.isAtEnd() {
		tok := p.current()
		switch tok.Type {
		case TokenTypeAttrName:
			result += " " + tok.Value
			p.advance()
		case TokenTypeEquals:
			result += "="
			p.advance()
		case TokenTypeAttrValue:
			result += "\"" + tok.Value + "\""
			p.advance()
		case TokenTypeSelfClose:
			result += " " + StrSelfClose
			p.advance()
			return result, nil
		case TokenTypeCloseTag:
			result += StrCloseDelim
			p.advance()
			return result, nil
		default:
			return result, nil
		}
	}
	return result, nil
}

// collectTagCloseAsText collects closing tag tokens as literal text (for raw blocks)
func (p *Parser) collectTagCloseAsText() (string, error) {
	var result string

	// Tag name
	if p.current().Type == TokenTypeTagName {
		result += p.current().Value
		p.advance()
	}

	// Close delimiter
	if p.current().Type == TokenTypeCloseTag {
		result += StrCloseDelim
		p.advance()
	}

	return result, nil
}

// parseAttributes parses tag attributes until we hit a closing token
func (p *Parser) parseAttributes() (Attributes, error) {
	attrs := make(Attributes)

	for !p.isAtEnd() {
		tok := p.current()

		// Stop at closing tokens
		if tok.Type == TokenTypeSelfClose || tok.Type == TokenTypeCloseTag {
			break
		}

		// Expect attribute name
		if tok.Type != TokenTypeAttrName {
			return nil, p.newUnexpectedTokenError(tok)
		}
		attrName := tok.Value
		p.advance()

		// Expect equals
		if p.current().Type != TokenTypeEquals {
			return nil, p.newExpectedTokenError(TokenTypeEquals, p.current())
		}
		p.advance()

		// Expect value
		if p.current().Type != TokenTypeAttrValue {
			return nil, p.newExpectedTokenError(TokenTypeAttrValue, p.current())
		}
		attrValue := p.current().Value
		p.advance()

		attrs[attrName] = attrValue
	}

	return attrs, nil
}

// Helper methods

// current returns the current token
func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenTypeEOF}
	}
	return p.tokens[p.pos]
}

// advance consumes and returns the current token
func (p *Parser) advance() Token {
	tok := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

// isAtEnd returns true if we've reached EOF
func (p *Parser) isAtEnd() bool {
	return p.current().Type == TokenTypeEOF
}

// isBlockClose returns true if current token is BLOCK_CLOSE
func (p *Parser) isBlockClose() bool {
	return p.current().Type == TokenTypeBlockClose
}

// Error helpers

func (p *Parser) newUnexpectedTokenError(tok Token) error {
	return &ParserError{
		Message:  ErrMsgUnexpectedToken,
		Position: tok.Position,
		Token:    tok,
	}
}

func (p *Parser) newExpectedTokenError(expected TokenType, actual Token) error {
	return &ParserError{
		Message:  ErrMsgExpectedToken,
		Position: actual.Position,
		Expected: expected,
		Token:    actual,
	}
}

func (p *Parser) newMismatchedTagError(expected, actual string) error {
	return &ParserError{
		Message:       ErrMsgMismatchedTag,
		Position:      p.current().Position,
		ExpectedTag:   expected,
		ActualTag:     actual,
	}
}

func (p *Parser) newNestedRawBlockError(pos Position) error {
	return &ParserError{
		Message:  ErrMsgNestedRawBlock,
		Position: pos,
	}
}

// ParserError represents a parser error with context
type ParserError struct {
	Message     string
	Position    Position
	Token       Token
	Expected    TokenType
	ExpectedTag string
	ActualTag   string
}

func (e *ParserError) Error() string {
	return fmt.Sprintf(ErrFmtWithPosition, e.Message, e.Position.String())
}

// Parser error message constants
const (
	ErrMsgUnexpectedToken = "unexpected token"
	ErrMsgExpectedToken   = "expected token"
	ErrMsgMismatchedTag   = "mismatched closing tag"
	ErrMsgNestedRawBlock  = "nested raw blocks are not allowed"
)
