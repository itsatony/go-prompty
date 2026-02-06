package prompty

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parse parses a v2.1 document (YAML frontmatter + body) into a Prompt.
// The document must start with --- and have a closing --- delimiter.
// The body (everything after the closing ---) is set on the Prompt.Body field.
// If the document has no frontmatter, the entire content is treated as body.
func Parse(data []byte) (*Prompt, error) {
	if len(data) == 0 {
		return nil, NewFrontmatterError(ErrMsgFrontmatterInvalid, Position{Line: 1, Column: 1}, nil)
	}

	content := string(data)

	// Trim BOM and leading whitespace
	content = strings.TrimLeft(content, "\xef\xbb\xbf \t")

	// Check for frontmatter
	if !strings.HasPrefix(content, YAMLFrontmatterDelimiter) {
		// No frontmatter — treat entire content as body of a default skill prompt
		return &Prompt{
			Type: DocumentTypeSkill,
			Body: content,
		}, nil
	}

	// Skip opening delimiter and newline
	afterOpening := content[len(YAMLFrontmatterDelimiter):]
	if len(afterOpening) > 0 && afterOpening[0] == '\n' {
		afterOpening = afterOpening[1:]
	} else if len(afterOpening) > 1 && afterOpening[0] == '\r' && afterOpening[1] == '\n' {
		afterOpening = afterOpening[2:]
	}

	// Find closing delimiter
	closeIdx := strings.Index(afterOpening, "\n"+YAMLFrontmatterDelimiter)
	if closeIdx == -1 {
		return nil, NewFrontmatterError(ErrMsgFrontmatterUnclosed, Position{Line: 1, Column: 1}, nil)
	}

	// Extract frontmatter YAML
	fmYAML := afterOpening[:closeIdx]

	// Check size limit
	if len(fmYAML) > DefaultMaxFrontmatterSize {
		return nil, NewFrontmatterError(ErrMsgFrontmatterTooLarge, Position{Line: 1, Column: 1}, nil)
	}

	// Extract body (after closing delimiter and newline)
	bodyStart := closeIdx + len("\n"+YAMLFrontmatterDelimiter)
	body := ""
	if bodyStart < len(afterOpening) {
		body = afterOpening[bodyStart:]
		if len(body) > 0 && body[0] == '\n' {
			body = body[1:]
		} else if len(body) > 1 && body[0] == '\r' && body[1] == '\n' {
			body = body[2:]
		}
	}

	// Parse YAML frontmatter into Prompt
	var prompt Prompt
	if err := yaml.Unmarshal([]byte(fmYAML), &prompt); err != nil {
		return nil, NewFrontmatterParseError(err)
	}

	// Set the body
	prompt.Body = body

	// Set default type if not specified
	if prompt.Type == "" {
		prompt.Type = DocumentTypeSkill
	}

	// Validate
	if err := prompt.Validate(); err != nil {
		return nil, err
	}

	return &prompt, nil
}

// ParseFile reads a file and parses it as a v2.1 document.
func ParseFile(path string) (*Prompt, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, NewFrontmatterError(ErrMsgFrontmatterExtract, Position{Line: 1, Column: 1}, err)
	}
	return Parse(data)
}

// MustParse parses a v2.1 document and panics on error.
func MustParse(data []byte) *Prompt {
	p, err := Parse(data)
	if err != nil {
		panic(err)
	}
	return p
}
