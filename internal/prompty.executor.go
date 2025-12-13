package internal

import (
	"context"
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
}

// NewExecutor creates a new executor with the given registry and configuration.
func NewExecutor(registry *Registry, config ExecutorConfig, logger *zap.Logger) *Executor {
	if logger == nil {
		logger = zap.NewNop()
	}
	logger.Debug(LogMsgExecutorCreated)
	return &Executor{
		registry: registry,
		config:   config,
		logger:   logger,
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

	default:
		return "", NewExecutorError(ErrMsgUnknownNodeType, "", node.Pos())
	}
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
		return "", NewExecutorError(ErrMsgUnknownTag+": "+tag.Name, tag.Name, tag.Pos())
	}

	// Execute resolver
	result, err := resolver.Resolve(ctx, execCtx, tag.Attributes)
	if err != nil {
		return "", NewExecutorError(ErrMsgResolverFailed+": "+err.Error(), tag.Name, tag.Pos())
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

// ExecutorError represents an executor error with context.
type ExecutorError struct {
	Message  string
	TagName  string
	Position Position
}

// NewExecutorError creates a new executor error.
func NewExecutorError(message, tagName string, pos Position) *ExecutorError {
	return &ExecutorError{
		Message:  message,
		TagName:  tagName,
		Position: pos,
	}
}

// Error implements the error interface.
func (e *ExecutorError) Error() string {
	if e.TagName != "" {
		return e.Message + " at " + e.Position.String()
	}
	return e.Message + " at " + e.Position.String()
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
