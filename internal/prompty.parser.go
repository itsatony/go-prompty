package internal

import (
	"go.uber.org/zap"
)

// Parser produces an AST from a token stream
type Parser struct {
	tokens     []Token
	pos        int
	logger     *zap.Logger
	inRawBlock bool // Track if we're inside a raw block
}

// NewParser creates a new parser for the given token stream
func NewParser(tokens []Token, logger *zap.Logger) *Parser {
	if logger == nil {
		logger = zap.NewNop()
	}
	logger.Debug(LogMsgParserCreated, zap.Int(LogFieldTokens, len(tokens)))
	return &Parser{
		tokens:     tokens,
		pos:        0,
		logger:     logger,
		inRawBlock: false,
	}
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
func (p *Parser) parseTag() (*TagNode, error) {
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
		return NewSelfClosingTag(tagName, attrs, pos), nil

	case TokenTypeCloseTag:
		p.advance() // consume CLOSE_TAG
		// This is a block tag - parse content and closing
		return p.parseBlockTag(tagName, attrs, pos)

	default:
		return nil, p.newUnexpectedTokenError(endTok)
	}
}

// parseBlockTag parses the content and closing of a block tag
func (p *Parser) parseBlockTag(tagName string, attrs Attributes, pos Position) (*TagNode, error) {
	// Special handling for raw blocks
	if tagName == TagNameRaw {
		return p.parseRawBlock(pos)
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

	return NewBlockTag(tagName, attrs, children, pos), nil
}

// parseRawBlock parses a raw block - preserving content literally
func (p *Parser) parseRawBlock(pos Position) (*TagNode, error) {
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

	return NewRawBlockTag(rawContent, pos), nil
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
	return e.Message + " at " + e.Position.String()
}

// Parser error message constants
const (
	ErrMsgUnexpectedToken = "unexpected token"
	ErrMsgExpectedToken   = "expected token"
	ErrMsgMismatchedTag   = "mismatched closing tag"
	ErrMsgNestedRawBlock  = "nested raw blocks are not allowed"
)
