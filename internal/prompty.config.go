package internal

import (
	"strings"
)

// Config block tag constants
const (
	// ConfigBlockOpen is the opening tag for config blocks
	ConfigBlockOpen = "{~prompty.config~}"
	// ConfigBlockClose is the closing tag for config blocks
	ConfigBlockClose = "{~/prompty.config~}"
)

// ConfigBlockResult holds the result of extracting a config block from source.
type ConfigBlockResult struct {
	// ConfigJSON contains the JSON string from inside the config block.
	// Empty if no config block was found.
	ConfigJSON string
	// TemplateBody contains the template source after the config block.
	// If no config block, this is the original source.
	TemplateBody string
	// ConfigPosition is the position where the config block started.
	// Zero position if no config block was found.
	ConfigPosition Position
	// HasConfig indicates whether a config block was found.
	HasConfig bool
}

// ExtractConfigBlock finds and extracts the config block from source.
// The config block must appear at the start of the source (after optional leading whitespace).
//
// Format:
//
//	{~prompty.config~}
//	{ "json": "content" }
//	{~/prompty.config~}
//	Template body here...
//
// Returns:
//   - ConfigBlockResult containing the extracted config JSON and remaining template body
//   - error if the config block is malformed (e.g., unclosed)
func ExtractConfigBlock(source string, config LexerConfig) (*ConfigBlockResult, error) {
	result := &ConfigBlockResult{
		TemplateBody: source,
	}

	// Skip leading whitespace
	trimmedStart := 0
	for trimmedStart < len(source) {
		ch := source[trimmedStart]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			trimmedStart++
		} else {
			break
		}
	}

	// Check for config block open tag
	remaining := source[trimmedStart:]
	if !strings.HasPrefix(remaining, ConfigBlockOpen) {
		// No config block
		return result, nil
	}

	// Calculate position of config block
	result.ConfigPosition = calculatePosition(source[:trimmedStart])
	result.HasConfig = true

	// Find the end of the config block
	contentStart := trimmedStart + len(ConfigBlockOpen)
	closeIdx := strings.Index(source[contentStart:], ConfigBlockClose)
	if closeIdx == -1 {
		// Config block not closed
		pos := calculatePosition(source[:contentStart])
		return nil, &ConfigError{
			Message:  ErrMsgConfigBlockUnclosed,
			Position: pos,
		}
	}

	// Extract config JSON (content between open and close tags)
	result.ConfigJSON = strings.TrimSpace(source[contentStart : contentStart+closeIdx])

	// Template body is everything after the close tag
	bodyStart := contentStart + closeIdx + len(ConfigBlockClose)
	result.TemplateBody = source[bodyStart:]

	// Trim leading newline from template body if present
	// This makes the template body cleaner when config block ends with newline
	if len(result.TemplateBody) > 0 && result.TemplateBody[0] == '\n' {
		result.TemplateBody = result.TemplateBody[1:]
	} else if len(result.TemplateBody) > 1 && result.TemplateBody[0] == '\r' && result.TemplateBody[1] == '\n' {
		result.TemplateBody = result.TemplateBody[2:]
	}

	return result, nil
}

// calculatePosition calculates the Position (line, column, offset) for a given prefix string.
func calculatePosition(prefix string) Position {
	pos := Position{
		Offset: len(prefix),
		Line:   1,
		Column: 1,
	}

	for i := 0; i < len(prefix); i++ {
		if prefix[i] == '\n' {
			pos.Line++
			pos.Column = 1
		} else {
			pos.Column++
		}
	}

	return pos
}

// ConfigError represents an error during config block extraction.
type ConfigError struct {
	Message  string
	Position Position
	Cause    error
}

// Error implements the error interface.
func (e *ConfigError) Error() string {
	if e.Position.Line > 0 {
		return e.Message + " at " + e.Position.String()
	}
	return e.Message
}

// Unwrap returns the underlying cause.
func (e *ConfigError) Unwrap() error {
	return e.Cause
}

// NewConfigError creates a new config error with position.
func NewConfigError(message string, pos Position, cause error) *ConfigError {
	return &ConfigError{
		Message:  message,
		Position: pos,
		Cause:    cause,
	}
}
