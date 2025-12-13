package internal

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// ExecutorConfig holds executor configuration options.
type ExecutorConfig struct {
	MaxDepth int // Maximum nesting depth (0 = unlimited)
}

// DefaultExecutorConfig returns the default executor configuration.
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		MaxDepth: DefaultMaxDepth,
	}
}

// Executor traverses an AST and produces output by resolving tags.
type Executor struct {
	registry *Registry
	config   ExecutorConfig
	logger   *zap.Logger
	funcs    *FuncRegistry // Function registry for expression evaluation
}

// NewExecutor creates a new executor with the given registry and configuration.
func NewExecutor(registry *Registry, config ExecutorConfig, logger *zap.Logger) *Executor {
	if logger == nil {
		logger = zap.NewNop()
	}
	logger.Debug(LogMsgExecutorCreated)

	// Create function registry with built-in functions
	funcs := NewFuncRegistry()
	RegisterBuiltinFuncs(funcs)

	return &Executor{
		registry: registry,
		config:   config,
		logger:   logger,
		funcs:    funcs,
	}
}

// Execute processes the AST and returns the rendered output.
func (e *Executor) Execute(ctx context.Context, root *RootNode, execCtx ContextAccessor) (string, error) {
	e.logger.Debug(LogMsgExecutorStart)

	result, err := e.executeNodes(ctx, root.Children, execCtx, 0)
	if err != nil {
		return "", err
	}

	e.logger.Debug(LogMsgExecutorEnd)
	return result, nil
}

// executeNodes processes a slice of nodes and concatenates their output.
func (e *Executor) executeNodes(ctx context.Context, nodes []Node, execCtx ContextAccessor, depth int) (string, error) {
	// Check depth limit
	if e.config.MaxDepth > 0 && depth > e.config.MaxDepth {
		return "", NewExecutorError(ErrMsgMaxDepthExceeded, "", Position{})
	}

	var sb strings.Builder

	for _, node := range nodes {
		output, err := e.executeNode(ctx, node, execCtx, depth)
		if err != nil {
			return "", err
		}
		sb.WriteString(output)
	}

	return sb.String(), nil
}

// executeNode processes a single node and returns its output.
func (e *Executor) executeNode(ctx context.Context, node Node, execCtx ContextAccessor, depth int) (string, error) {
	switch n := node.(type) {
	case *TextNode:
		return n.Content, nil

	case *TagNode:
		return e.executeTag(ctx, n, execCtx, depth)

	case *ConditionalNode:
		return e.executeConditional(ctx, n, execCtx, depth)

	default:
		return "", NewExecutorError(ErrMsgUnknownNodeType, "", node.Pos())
	}
}

// executeConditional processes a conditional node and returns its output.
func (e *Executor) executeConditional(ctx context.Context, cond *ConditionalNode, execCtx ContextAccessor, depth int) (string, error) {
	e.logger.Debug(LogMsgConditionEval, zap.Int("branches", len(cond.Branches)))

	for i, branch := range cond.Branches {
		// else branch - always execute if we reach it
		if branch.IsElse {
			e.logger.Debug(LogMsgBranchSelected, zap.Int(LogFieldBranch, i), zap.Bool("isElse", true))
			return e.executeNodes(ctx, branch.Children, execCtx, depth+1)
		}

		// Evaluate the condition expression
		result, err := e.evaluateCondition(branch.Condition, execCtx)
		if err != nil {
			return "", NewExecutorErrorWithCause(ErrMsgCondExprFailed, TagNameIf, branch.Pos, err)
		}

		if result {
			e.logger.Debug(LogMsgBranchSelected, zap.Int(LogFieldBranch, i), zap.String("condition", branch.Condition))
			return e.executeNodes(ctx, branch.Children, execCtx, depth+1)
		}
	}

	// No branch matched - return empty string
	return "", nil
}

// evaluateCondition parses and evaluates a condition expression.
func (e *Executor) evaluateCondition(expr string, execCtx ContextAccessor) (bool, error) {
	return EvaluateExpressionBool(expr, e.funcs, execCtx)
}

// executeTag processes a tag node and returns its output.
func (e *Executor) executeTag(ctx context.Context, tag *TagNode, execCtx ContextAccessor, depth int) (string, error) {
	e.logger.Debug(LogMsgResolverInvoked, zap.String(LogFieldTag, tag.Name))

	// Handle raw blocks specially
	if tag.IsRaw() {
		return tag.RawContent, nil
	}

	// Look up resolver
	resolver, ok := e.registry.Get(tag.Name)
	if !ok {
		return e.handleTagError(tag, execCtx, NewExecutorError(ErrMsgUnknownTag, tag.Name, tag.Pos()))
	}

	// Execute resolver
	result, err := resolver.Resolve(ctx, execCtx, tag.Attributes)
	if err != nil {
		return e.handleTagError(tag, execCtx, NewExecutorErrorWithCause(ErrMsgResolverFailed, tag.Name, tag.Pos(), err))
	}

	// For block tags with children, process children
	if !tag.SelfClose && len(tag.Children) > 0 {
		childResult, err := e.executeNodes(ctx, tag.Children, execCtx, depth+1)
		if err != nil {
			return "", err
		}
		// Combine resolver result with children (resolver result comes first)
		return result + childResult, nil
	}

	e.logger.Debug(LogMsgResolverComplete, zap.String(LogFieldTag, tag.Name))
	return result, nil
}

// handleTagError applies the appropriate error strategy for a tag execution failure.
func (e *Executor) handleTagError(tag *TagNode, execCtx ContextAccessor, err error) (string, error) {
	// Determine the error strategy to use
	strategy := e.getErrorStrategy(tag, execCtx)

	e.logger.Debug(LogMsgErrorStrategyApplied,
		zap.String(LogFieldTag, tag.Name),
		zap.String(LogFieldStrategy, ErrorStrategy(strategy).String()),
		zap.String(LogFieldErrorMsg, err.Error()))

	switch ErrorStrategy(strategy) {
	case ErrorStrategyThrow:
		// Default behavior - propagate the error
		return "", err

	case ErrorStrategyDefault:
		// Use the default attribute value if available
		if defaultVal, hasDefault := tag.Attributes.Get(AttrDefault); hasDefault {
			return defaultVal, nil
		}
		// No default specified - return empty string
		return "", nil

	case ErrorStrategyRemove:
		// Remove the tag entirely - return empty string
		return "", nil

	case ErrorStrategyKeepRaw:
		// Keep the original tag source
		if tag.RawSource != "" {
			return tag.RawSource, nil
		}
		// Fallback to empty if no raw source captured
		return "", nil

	case ErrorStrategyLog:
		// Log the error and continue with empty string
		e.logger.Warn(LogMsgErrorLogged,
			zap.String(LogFieldTag, tag.Name),
			zap.Error(err))
		return "", nil

	default:
		// Unknown strategy - fall back to throw
		return "", err
	}
}

// getErrorStrategy determines which error strategy to use for a tag.
// Priority: per-tag onerror attribute > context default > throw
func (e *Executor) getErrorStrategy(tag *TagNode, execCtx ContextAccessor) int {
	// Check for per-tag onerror attribute
	if onErrorStr, hasOnError := tag.Attributes.Get(AttrOnError); hasOnError {
		return int(ParseErrorStrategy(onErrorStr))
	}

	// Check if context provides error strategy
	if stratCtx, ok := execCtx.(ErrorStrategyAccessor); ok {
		return stratCtx.ErrorStrategy()
	}

	// Default to throw
	return int(ErrorStrategyThrow)
}

// ExecutorError represents an executor error with context.
type ExecutorError struct {
	Message  string
	TagName  string
	Position Position
	Cause    error
}

// NewExecutorError creates a new executor error.
func NewExecutorError(message, tagName string, pos Position) *ExecutorError {
	return &ExecutorError{
		Message:  message,
		TagName:  tagName,
		Position: pos,
	}
}

// NewExecutorErrorWithCause creates a new executor error with a cause.
func NewExecutorErrorWithCause(message, tagName string, pos Position, cause error) *ExecutorError {
	return &ExecutorError{
		Message:  message,
		TagName:  tagName,
		Position: pos,
		Cause:    cause,
	}
}

// Error implements the error interface.
func (e *ExecutorError) Error() string {
	var result string
	if e.TagName != StringValueEmpty {
		result = fmt.Sprintf(ErrFmtWithTagAndPosition, e.Message, e.TagName, e.Position.String())
	} else {
		result = fmt.Sprintf(ErrFmtWithPosition, e.Message, e.Position.String())
	}
	if e.Cause != nil {
		result = fmt.Sprintf(ErrFmtWithCause, result, e.Cause)
	}
	return result
}

// Unwrap returns the underlying cause error.
func (e *ExecutorError) Unwrap() error {
	return e.Cause
}

// Executor error message constants
const (
	ErrMsgMaxDepthExceeded = "maximum nesting depth exceeded"
	ErrMsgUnknownNodeType  = "unknown node type"
	ErrMsgUnknownTag       = "unknown tag"
	ErrMsgResolverFailed   = "resolver failed"
)

// Default configuration values
const (
	DefaultMaxDepth = 100
)
