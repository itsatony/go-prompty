package prompty

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTemplateRunner_EngineImplements verifies Engine satisfies TemplateRunner at compile time.
func TestTemplateRunner_EngineImplements(t *testing.T) {
	var _ TemplateRunner = (*Engine)(nil)
}

// TestTemplateRunner_StorageEngineImplements verifies StorageEngine satisfies TemplateRunner at compile time.
func TestTemplateRunner_StorageEngineImplements(t *testing.T) {
	var _ TemplateRunner = (*StorageEngine)(nil)
}

// TestTemplateRunner_RegisterResolverViaInterface registers through the interface and verifies HasResolver.
func TestTemplateRunner_RegisterResolverViaInterface(t *testing.T) {
	engine := MustNew()
	var runner TemplateRunner = engine

	resolver := &testTagResolver{tagName: "test.custom"}
	err := runner.RegisterResolver(resolver)
	require.NoError(t, err)
	assert.True(t, runner.HasResolver("test.custom"))
}

// TestTemplateRunner_ListResolversViaInterface lists resolvers through the interface.
func TestTemplateRunner_ListResolversViaInterface(t *testing.T) {
	engine := MustNew()
	var runner TemplateRunner = engine

	resolvers := runner.ListResolvers()
	assert.NotEmpty(t, resolvers)
	assert.Contains(t, resolvers, TagNameVar)
	assert.Contains(t, resolvers, TagNameInclude)
}

// TestTemplateRunner_ResolverCountViaInterface counts resolvers through the interface.
func TestTemplateRunner_ResolverCountViaInterface(t *testing.T) {
	engine := MustNew()
	var runner TemplateRunner = engine

	count := runner.ResolverCount()
	assert.Greater(t, count, 0)

	resolver := &testTagResolver{tagName: "test.countcheck"}
	err := runner.RegisterResolver(resolver)
	require.NoError(t, err)
	assert.Equal(t, count+1, runner.ResolverCount())
}

// TestEngine_RegisterResolver_Alias verifies Engine.RegisterResolver is the same as Engine.Register.
func TestEngine_RegisterResolver_Alias(t *testing.T) {
	engine := MustNew()

	resolver := &testTagResolver{tagName: "test.alias"}
	err := engine.RegisterResolver(resolver)
	require.NoError(t, err)
	assert.True(t, engine.HasResolver("test.alias"))

	// Duplicate registration should fail
	err = engine.RegisterResolver(resolver)
	assert.Error(t, err)
}

// TestStorageEngine_HasResolver_Delegate verifies StorageEngine.HasResolver delegates to engine.
func TestStorageEngine_HasResolver_Delegate(t *testing.T) {
	se := MustNewStorageEngine(StorageEngineConfig{
		Storage: NewMemoryStorage(),
	})
	defer se.Close()

	assert.True(t, se.HasResolver(TagNameVar))
	assert.True(t, se.HasResolver(TagNameInclude))
	assert.False(t, se.HasResolver("nonexistent.tag"))
}

// TestStorageEngine_ListResolvers_Delegate verifies StorageEngine.ListResolvers delegates to engine.
func TestStorageEngine_ListResolvers_Delegate(t *testing.T) {
	se := MustNewStorageEngine(StorageEngineConfig{
		Storage: NewMemoryStorage(),
	})
	defer se.Close()

	resolvers := se.ListResolvers()
	assert.NotEmpty(t, resolvers)
	assert.Contains(t, resolvers, TagNameVar)
}

// testTagResolver is a minimal resolver for testing.
type testTagResolver struct {
	tagName string
}

func (r *testTagResolver) TagName() string {
	return r.tagName
}

func (r *testTagResolver) Resolve(ctx context.Context, execCtx *Context, attrs Attributes) (string, error) {
	return "test", nil
}

func (r *testTagResolver) Validate(attrs Attributes) error {
	return nil
}
