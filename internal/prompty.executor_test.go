package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutor_NewExecutor(t *testing.T) {
	registry := NewRegistry(nil)
	config := DefaultExecutorConfig()

	executor := NewExecutor(registry, config, nil)
	require.NotNil(t, executor)
}

func TestExecutor_ExecutePlainText(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	root := &RootNode{
		Children: []Node{
			NewTextNode("Hello, World!", Position{Line: 1, Column: 1}),
		},
	}

	ctx := newMockContextAccessor(nil)
	result, err := executor.Execute(context.Background(), root, ctx)

	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", result)
}

func TestExecutor_ExecuteMultipleTextNodes(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	root := &RootNode{
		Children: []Node{
			NewTextNode("Hello, ", Position{Line: 1, Column: 1}),
			NewTextNode("World", Position{Line: 1, Column: 8}),
			NewTextNode("!", Position{Line: 1, Column: 13}),
		},
	}

	ctx := newMockContextAccessor(nil)
	result, err := executor.Execute(context.Background(), root, ctx)

	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", result)
}

func TestExecutor_ExecuteVarTag(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	root := &RootNode{
		Children: []Node{
			NewTextNode("Hello, ", Position{Line: 1, Column: 1}),
			NewSelfClosingTag(TagNameVar, Attributes{"name": "user"}, Position{Line: 1, Column: 8}),
			NewTextNode("!", Position{Line: 1, Column: 30}),
		},
	}

	ctx := newMockContextAccessor(map[string]any{
		"user": "Alice",
	})

	result, err := executor.Execute(context.Background(), root, ctx)

	require.NoError(t, err)
	assert.Equal(t, "Hello, Alice!", result)
}

func TestExecutor_ExecuteVarWithDefault(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	root := &RootNode{
		Children: []Node{
			NewSelfClosingTag(TagNameVar, Attributes{
				"name":    "missing",
				"default": "Guest",
			}, Position{Line: 1, Column: 1}),
		},
	}

	ctx := newMockContextAccessor(nil)
	result, err := executor.Execute(context.Background(), root, ctx)

	require.NoError(t, err)
	assert.Equal(t, "Guest", result)
}

func TestExecutor_ExecuteRawBlock(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	// Create a raw block with literal tag syntax inside
	rawTag := NewRawBlockTag(`{~prompty.var name="x" /~}`, Position{Line: 1, Column: 1})

	root := &RootNode{
		Children: []Node{rawTag},
	}

	ctx := newMockContextAccessor(nil)
	result, err := executor.Execute(context.Background(), root, ctx)

	require.NoError(t, err)
	// Raw content should be preserved literally
	assert.Equal(t, `{~prompty.var name="x" /~}`, result)
}

func TestExecutor_ExecuteNestedTags(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry)

	// Register a custom block tag resolver
	registry.MustRegister(&testBlockResolver{name: "wrapper"})

	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	// Create a block tag with children
	blockTag := NewBlockTag("wrapper", Attributes{}, []Node{
		NewTextNode("Inner content", Position{Line: 1, Column: 10}),
	}, Position{Line: 1, Column: 1})

	root := &RootNode{
		Children: []Node{blockTag},
	}

	ctx := newMockContextAccessor(nil)
	result, err := executor.Execute(context.Background(), root, ctx)

	require.NoError(t, err)
	// Block resolver returns "[" and children append
	assert.Equal(t, "[Inner content", result)
}

func TestExecutor_ExecuteUnknownTag(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	root := &RootNode{
		Children: []Node{
			NewSelfClosingTag("unknown.tag", Attributes{}, Position{Line: 1, Column: 1}),
		},
	}

	ctx := newMockContextAccessor(nil)
	_, err := executor.Execute(context.Background(), root, ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgUnknownTag)
	assert.Contains(t, err.Error(), "unknown.tag")
}

func TestExecutor_ExecuteMissingVariable(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	root := &RootNode{
		Children: []Node{
			NewSelfClosingTag(TagNameVar, Attributes{"name": "missing"}, Position{Line: 1, Column: 1}),
		},
	}

	ctx := newMockContextAccessor(nil)
	_, err := executor.Execute(context.Background(), root, ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgResolverFailed)
}

func TestExecutor_MaxDepthExceeded(t *testing.T) {
	registry := NewRegistry(nil)
	registry.MustRegister(&testBlockResolver{name: "block"})

	config := ExecutorConfig{MaxDepth: 2}
	executor := NewExecutor(registry, config, nil)

	// Create deeply nested structure
	innermost := NewBlockTag("block", Attributes{}, []Node{
		NewTextNode("deep", Position{Line: 1, Column: 1}),
	}, Position{Line: 1, Column: 1})

	middle := NewBlockTag("block", Attributes{}, []Node{innermost}, Position{Line: 1, Column: 1})
	outer := NewBlockTag("block", Attributes{}, []Node{middle}, Position{Line: 1, Column: 1})

	root := &RootNode{Children: []Node{outer}}

	ctx := newMockContextAccessor(nil)
	_, err := executor.Execute(context.Background(), root, ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgMaxDepthExceeded)
}

func TestExecutor_NoDepthLimitWhenZero(t *testing.T) {
	registry := NewRegistry(nil)
	registry.MustRegister(&testBlockResolver{name: "block"})

	config := ExecutorConfig{MaxDepth: 0} // Unlimited
	executor := NewExecutor(registry, config, nil)

	// Create nested structure
	inner := NewBlockTag("block", Attributes{}, []Node{
		NewTextNode("deep", Position{Line: 1, Column: 1}),
	}, Position{Line: 1, Column: 1})

	outer := NewBlockTag("block", Attributes{}, []Node{inner}, Position{Line: 1, Column: 1})
	root := &RootNode{Children: []Node{outer}}

	ctx := newMockContextAccessor(nil)
	result, err := executor.Execute(context.Background(), root, ctx)

	require.NoError(t, err)
	assert.Equal(t, "[[deep", result) // Two "[" from the two block resolvers
}

func TestExecutor_ResolverError(t *testing.T) {
	registry := NewRegistry(nil)
	registry.MustRegister(&testErrorResolver{})

	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	root := &RootNode{
		Children: []Node{
			NewSelfClosingTag("error.tag", Attributes{}, Position{Line: 5, Column: 10}),
		},
	}

	ctx := newMockContextAccessor(nil)
	_, err := executor.Execute(context.Background(), root, ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgResolverFailed)
}

func TestExecutor_ComplexTemplate(t *testing.T) {
	registry := NewRegistry(nil)
	RegisterBuiltins(registry)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	// Simulate a parsed template like:
	// Hello, {~prompty.var name="user" /~}!
	// Your items: {~prompty.raw~}{{ items }}{~/prompty.raw~}
	root := &RootNode{
		Children: []Node{
			NewTextNode("Hello, ", Position{Line: 1, Column: 1}),
			NewSelfClosingTag(TagNameVar, Attributes{"name": "user"}, Position{Line: 1, Column: 8}),
			NewTextNode("!\nYour items: ", Position{Line: 1, Column: 35}),
			NewRawBlockTag("{{ items }}", Position{Line: 2, Column: 13}),
		},
	}

	ctx := newMockContextAccessor(map[string]any{
		"user": "Bob",
	})

	result, err := executor.Execute(context.Background(), root, ctx)

	require.NoError(t, err)
	assert.Equal(t, "Hello, Bob!\nYour items: {{ items }}", result)
}

func TestExecutor_EmptyRoot(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	root := &RootNode{Children: []Node{}}

	ctx := newMockContextAccessor(nil)
	result, err := executor.Execute(context.Background(), root, ctx)

	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestExecutorError_Error(t *testing.T) {
	t.Run("with tag name", func(t *testing.T) {
		err := NewExecutorError("test error", "my.tag", Position{Line: 5, Column: 10})
		errStr := err.Error()
		assert.Contains(t, errStr, "test error")
		assert.Contains(t, errStr, "5")
	})

	t.Run("without tag name", func(t *testing.T) {
		err := NewExecutorError("test error", "", Position{Line: 1, Column: 1})
		errStr := err.Error()
		assert.Contains(t, errStr, "test error")
	})
}

func TestDefaultExecutorConfig(t *testing.T) {
	config := DefaultExecutorConfig()
	assert.Equal(t, DefaultMaxDepth, config.MaxDepth)
}

// Test helpers

// testBlockResolver is a test resolver that returns "[" for block tags
type testBlockResolver struct {
	name string
}

func (r *testBlockResolver) TagName() string { return r.name }

func (r *testBlockResolver) Resolve(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error) {
	return "[", nil
}

func (r *testBlockResolver) Validate(attrs Attributes) error {
	return nil
}

// testErrorResolver is a test resolver that always returns an error
type testErrorResolver struct{}

func (r *testErrorResolver) TagName() string { return "error.tag" }

func (r *testErrorResolver) Resolve(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error) {
	return "", errors.New("intentional error")
}

func (r *testErrorResolver) Validate(attrs Attributes) error {
	return nil
}

// Tests for Executor function wrappers
func TestExecutor_RegisterFunc(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	fn := &Func{
		Name:    "testfunc",
		MinArgs: 0,
		MaxArgs: 0,
		Fn:      func(args []any) (any, error) { return "test", nil },
	}

	err := executor.RegisterFunc(fn)
	require.NoError(t, err)
	assert.True(t, executor.HasFunc("testfunc"))
}

func TestExecutor_RegisterFunc_Duplicate(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	fn := &Func{
		Name:    "testfunc",
		MinArgs: 0,
		MaxArgs: 0,
		Fn:      func(args []any) (any, error) { return "test", nil },
	}

	err := executor.RegisterFunc(fn)
	require.NoError(t, err)

	// Second registration should fail
	err = executor.RegisterFunc(fn)
	require.Error(t, err)
}

func TestExecutor_MustRegisterFunc(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	fn := &Func{
		Name:    "testfunc",
		MinArgs: 0,
		MaxArgs: 0,
		Fn:      func(args []any) (any, error) { return "test", nil },
	}

	assert.NotPanics(t, func() {
		executor.MustRegisterFunc(fn)
	})
	assert.True(t, executor.HasFunc("testfunc"))
}

func TestExecutor_MustRegisterFunc_Panic(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	fn := &Func{
		Name:    "testfunc",
		MinArgs: 0,
		MaxArgs: 0,
		Fn:      func(args []any) (any, error) { return "test", nil },
	}

	executor.MustRegisterFunc(fn)

	assert.Panics(t, func() {
		executor.MustRegisterFunc(fn) // duplicate
	})
}

func TestExecutor_HasFunc(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	assert.False(t, executor.HasFunc("nonexistent"))

	fn := &Func{
		Name:    "exists",
		MinArgs: 0,
		MaxArgs: 0,
		Fn:      func(args []any) (any, error) { return nil, nil },
	}
	executor.MustRegisterFunc(fn)

	assert.True(t, executor.HasFunc("exists"))
}

func TestExecutor_ListFuncs(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	executor.MustRegisterFunc(&Func{Name: "customfunc1", MinArgs: 0, MaxArgs: 0, Fn: func(args []any) (any, error) { return nil, nil }})
	executor.MustRegisterFunc(&Func{Name: "customfunc2", MinArgs: 0, MaxArgs: 0, Fn: func(args []any) (any, error) { return nil, nil }})

	funcs := executor.ListFuncs()
	// Should have built-in functions plus custom ones
	assert.True(t, len(funcs) > 2, "should have multiple functions")
	assert.Contains(t, funcs, "customfunc1")
	assert.Contains(t, funcs, "customfunc2")
	// Verify some built-in functions are present
	assert.Contains(t, funcs, "len")
	assert.Contains(t, funcs, "upper")
}

func TestExecutor_FuncCount(t *testing.T) {
	registry := NewRegistry(nil)
	executor := NewExecutor(registry, DefaultExecutorConfig(), nil)

	// Executor starts with built-in functions registered
	initialCount := executor.FuncCount()
	assert.True(t, initialCount > 0, "should have built-in functions")

	executor.MustRegisterFunc(&Func{Name: "customfunc1", MinArgs: 0, MaxArgs: 0, Fn: func(args []any) (any, error) { return nil, nil }})
	assert.Equal(t, initialCount+1, executor.FuncCount())

	executor.MustRegisterFunc(&Func{Name: "customfunc2", MinArgs: 0, MaxArgs: 0, Fn: func(args []any) (any, error) { return nil, nil }})
	assert.Equal(t, initialCount+2, executor.FuncCount())
}

func TestExecutorError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	execErr := &ExecutorError{
		Message:  "execution failed",
		TagName:  "test.tag",
		Position: Position{Line: 1, Column: 1},
		Cause:    cause,
	}

	assert.Equal(t, cause, execErr.Unwrap())
}

func TestExecutorError_UnwrapNil(t *testing.T) {
	execErr := &ExecutorError{
		Message:  "execution failed",
		TagName:  "test.tag",
		Position: Position{Line: 1, Column: 1},
		Cause:    nil,
	}

	assert.Nil(t, execErr.Unwrap())
}
