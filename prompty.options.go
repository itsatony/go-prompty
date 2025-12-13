package prompty

import (
	"go.uber.org/zap"
)

// Option is a functional option for configuring the Engine.
type Option func(*engineConfig)

// engineConfig holds the internal configuration for an Engine.
type engineConfig struct {
	openDelim     string
	closeDelim    string
	errorStrategy ErrorStrategy
	maxDepth      int
	logger        *zap.Logger
}

// defaultEngineConfig returns the default engine configuration.
func defaultEngineConfig() *engineConfig {
	return &engineConfig{
		openDelim:     DefaultOpenDelim,
		closeDelim:    DefaultCloseDelim,
		errorStrategy: ErrorStrategyThrow,
		maxDepth:      DefaultMaxDepth,
		logger:        nil,
	}
}

// WithDelimiters sets custom delimiters for template tags.
// Default: "{~" and "~}"
func WithDelimiters(open, close string) Option {
	return func(c *engineConfig) {
		if open != "" {
			c.openDelim = open
		}
		if close != "" {
			c.closeDelim = close
		}
	}
}

// WithErrorStrategy sets the error handling strategy.
// Default: ErrorStrategyThrow
func WithErrorStrategy(strategy ErrorStrategy) Option {
	return func(c *engineConfig) {
		c.errorStrategy = strategy
	}
}

// WithMaxDepth sets the maximum nesting depth for templates.
// Use 0 for unlimited depth.
// Default: 100
func WithMaxDepth(depth int) Option {
	return func(c *engineConfig) {
		c.maxDepth = depth
	}
}

// WithLogger sets the logger for the engine.
// Default: nil (no logging)
func WithLogger(logger *zap.Logger) Option {
	return func(c *engineConfig) {
		c.logger = logger
	}
}
