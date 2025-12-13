package prompty

import (
	"github.com/itsatony/go-prompty/internal"
)

// ValidationResult contains the results of template validation.
type ValidationResult struct {
	issues []ValidationIssue
}

// ValidationIssue represents a single validation finding.
type ValidationIssue struct {
	Severity ValidationSeverity
	Message  string
	Position Position
	TagName  string
}

// Issues returns all validation issues found.
func (r *ValidationResult) Issues() []ValidationIssue {
	return r.issues
}

// Errors returns only issues with error severity.
func (r *ValidationResult) Errors() []ValidationIssue {
	var errors []ValidationIssue
	for _, issue := range r.issues {
		if issue.Severity == SeverityError {
			errors = append(errors, issue)
		}
	}
	return errors
}

// Warnings returns only issues with warning severity.
func (r *ValidationResult) Warnings() []ValidationIssue {
	var warnings []ValidationIssue
	for _, issue := range r.issues {
		if issue.Severity == SeverityWarning {
			warnings = append(warnings, issue)
		}
	}
	return warnings
}

// HasErrors returns true if there are any error-severity issues.
func (r *ValidationResult) HasErrors() bool {
	for _, issue := range r.issues {
		if issue.Severity == SeverityError {
			return true
		}
	}
	return false
}

// HasWarnings returns true if there are any warning-severity issues.
func (r *ValidationResult) HasWarnings() bool {
	for _, issue := range r.issues {
		if issue.Severity == SeverityWarning {
			return true
		}
	}
	return false
}

// IsValid returns true if there are no error-severity issues.
func (r *ValidationResult) IsValid() bool {
	return !r.HasErrors()
}

// Validate parses and validates a template without executing it.
// It returns validation results containing any issues found.
// Parse errors are returned as validation errors with SeverityError.
func (e *Engine) Validate(source string) (*ValidationResult, error) {
	result := &ValidationResult{
		issues: make([]ValidationIssue, 0),
	}

	// Create lexer with configured delimiters
	lexerConfig := internal.LexerConfig{
		OpenDelim:  e.config.openDelim,
		CloseDelim: e.config.closeDelim,
	}
	lexer := internal.NewLexerWithConfig(source, lexerConfig, e.logger)

	// Tokenize
	tokens, err := lexer.Tokenize()
	if err != nil {
		result.issues = append(result.issues, ValidationIssue{
			Severity: SeverityError,
			Message:  ErrMsgParseFailed + ": " + err.Error(),
			Position: Position{},
		})
		return result, nil
	}

	// Parse with source for validation
	parser := internal.NewParserWithSource(tokens, source, e.logger)
	ast, err := parser.Parse()
	if err != nil {
		result.issues = append(result.issues, ValidationIssue{
			Severity: SeverityError,
			Message:  ErrMsgParseFailed + ": " + err.Error(),
			Position: Position{},
		})
		return result, nil
	}

	// Validate AST nodes
	e.validateNodes(ast.Children, result)

	return result, nil
}

// validateNodes recursively validates a slice of AST nodes.
func (e *Engine) validateNodes(nodes []internal.Node, result *ValidationResult) {
	for _, node := range nodes {
		e.validateNode(node, result)
	}
}

// validateNode validates a single AST node.
func (e *Engine) validateNode(node internal.Node, result *ValidationResult) {
	switch n := node.(type) {
	case *internal.TextNode:
		// Text nodes are always valid
		return

	case *internal.TagNode:
		e.validateTagNode(n, result)

	case *internal.ConditionalNode:
		e.validateConditionalNode(n, result)
	}
}

// validateTagNode validates a tag node.
func (e *Engine) validateTagNode(tag *internal.TagNode, result *ValidationResult) {
	// Skip raw blocks - they're always valid
	if tag.IsRaw() {
		return
	}

	// Check if tag has a registered resolver
	if !e.registry.Has(tag.Name) {
		result.issues = append(result.issues, ValidationIssue{
			Severity: SeverityWarning,
			Message:  ErrMsgUnknownTagInTemplate,
			Position: e.internalPosToPublic(tag.Pos()),
			TagName:  tag.Name,
		})
	} else {
		// Validate using the resolver's Validate method
		resolver, _ := e.registry.Get(tag.Name)
		if err := resolver.Validate(tag.Attributes); err != nil {
			result.issues = append(result.issues, ValidationIssue{
				Severity: SeverityError,
				Message:  err.Error(),
				Position: e.internalPosToPublic(tag.Pos()),
				TagName:  tag.Name,
			})
		}
	}

	// Validate onerror attribute if present
	if onErrorStr, hasOnError := tag.Attributes.Get(AttrOnError); hasOnError {
		if !IsValidErrorStrategy(onErrorStr) {
			result.issues = append(result.issues, ValidationIssue{
				Severity: SeverityError,
				Message:  ErrMsgInvalidOnErrorAttr,
				Position: e.internalPosToPublic(tag.Pos()),
				TagName:  tag.Name,
			})
		}
	}

	// Validate prompty.include references
	if tag.Name == TagNameInclude {
		if templateName, hasTemplate := tag.Attributes.Get(AttrTemplate); hasTemplate {
			if !e.HasTemplate(templateName) {
				result.issues = append(result.issues, ValidationIssue{
					Severity: SeverityWarning,
					Message:  ErrMsgMissingIncludeTarget,
					Position: e.internalPosToPublic(tag.Pos()),
					TagName:  tag.Name,
				})
			}
		}
	}

	// Validate children recursively
	if len(tag.Children) > 0 {
		e.validateNodes(tag.Children, result)
	}
}

// validateConditionalNode validates a conditional node.
func (e *Engine) validateConditionalNode(cond *internal.ConditionalNode, result *ValidationResult) {
	for _, branch := range cond.Branches {
		// Validate branch children recursively
		e.validateNodes(branch.Children, result)
	}
}

// internalPosToPublic converts internal Position to public Position.
func (e *Engine) internalPosToPublic(pos internal.Position) Position {
	return Position{
		Offset: pos.Offset,
		Line:   pos.Line,
		Column: pos.Column,
	}
}
