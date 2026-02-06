package prompty

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoopDocumentResolver_ResolvePrompt(t *testing.T) {
	r := &NoopDocumentResolver{}
	_, err := r.ResolvePrompt(context.Background(), "test-slug")
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgRefNotFound)
}

func TestNoopDocumentResolver_ResolveSkill(t *testing.T) {
	r := &NoopDocumentResolver{}
	_, err := r.ResolveSkill(context.Background(), "test-slug")
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgRefNotFound)
}

func TestNoopDocumentResolver_ResolveAgent(t *testing.T) {
	r := &NoopDocumentResolver{}
	_, err := r.ResolveAgent(context.Background(), "test-slug")
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgRefNotFound)
}

func TestMapDocumentResolver_AddAndResolvePrompt(t *testing.T) {
	r := NewMapDocumentResolver()
	p := &Prompt{
		Name:        "test-prompt",
		Description: "A test prompt",
		Type:        DocumentTypePrompt,
		Body:        "Hello!",
	}
	r.AddPrompt("test-prompt", p)

	resolved, err := r.ResolvePrompt(context.Background(), "test-prompt")
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "test-prompt", resolved.Name)
	assert.Equal(t, DocumentTypePrompt, resolved.Type)

	// Verify clone (not same pointer)
	assert.NotSame(t, p, resolved)
}

func TestMapDocumentResolver_ResolvePromptFallsBackToSkills(t *testing.T) {
	r := NewMapDocumentResolver()
	p := &Prompt{
		Name:        "my-skill",
		Description: "A skill",
		Type:        DocumentTypeSkill,
		Body:        "Skill body",
	}
	r.AddSkill("my-skill", p)

	// ResolvePrompt should fall back to skills map
	resolved, err := r.ResolvePrompt(context.Background(), "my-skill")
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "my-skill", resolved.Name)
}

func TestMapDocumentResolver_ResolvePromptNotFound(t *testing.T) {
	r := NewMapDocumentResolver()
	_, err := r.ResolvePrompt(context.Background(), "nonexistent")
	require.Error(t, err)
}

func TestMapDocumentResolver_AddAndResolveSkill(t *testing.T) {
	r := NewMapDocumentResolver()
	p := &Prompt{
		Name:        "my-skill",
		Description: "A skill",
		Type:        DocumentTypeSkill,
		Body:        "{~prompty.var name=\"query\" /~}",
	}
	r.AddSkill("my-skill", p)

	resolved, err := r.ResolveSkill(context.Background(), "my-skill")
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "my-skill", resolved.Name)
	assert.NotSame(t, p, resolved)
}

func TestMapDocumentResolver_ResolveSkillWithVersion(t *testing.T) {
	r := NewMapDocumentResolver()
	p := &Prompt{
		Name:        "my-skill",
		Description: "A skill",
		Type:        DocumentTypeSkill,
	}
	r.AddSkill("my-skill", p)

	// Resolve with slug@version syntax — should strip version and find by slug
	resolved, err := r.ResolveSkill(context.Background(), "my-skill@v2")
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "my-skill", resolved.Name)
}

func TestMapDocumentResolver_ResolveSkillFallsBackToPrompts(t *testing.T) {
	r := NewMapDocumentResolver()
	p := &Prompt{
		Name:        "shared",
		Description: "A prompt",
		Type:        DocumentTypePrompt,
	}
	r.AddPrompt("shared", p)

	// ResolveSkill should fall back to prompts map
	resolved, err := r.ResolveSkill(context.Background(), "shared")
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "shared", resolved.Name)
}

func TestMapDocumentResolver_ResolveSkillNotFound(t *testing.T) {
	r := NewMapDocumentResolver()
	_, err := r.ResolveSkill(context.Background(), "nonexistent")
	require.Error(t, err)
}

func TestMapDocumentResolver_AddAndResolveAgent(t *testing.T) {
	r := NewMapDocumentResolver()
	p := &Prompt{
		Name:        "my-agent",
		Description: "An agent",
		Type:        DocumentTypeAgent,
	}
	r.AddAgent("my-agent", p)

	resolved, err := r.ResolveAgent(context.Background(), "my-agent")
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "my-agent", resolved.Name)
	assert.NotSame(t, p, resolved)
}

func TestMapDocumentResolver_ResolveAgentNotFound(t *testing.T) {
	r := NewMapDocumentResolver()
	_, err := r.ResolveAgent(context.Background(), "nonexistent")
	require.Error(t, err)
}

func TestMapDocumentResolver_ReturnsClones(t *testing.T) {
	r := NewMapDocumentResolver()
	original := &Prompt{
		Name:        "original",
		Description: "Original prompt",
		Type:        DocumentTypeSkill,
		Body:        "Original body",
	}
	r.AddSkill("original", original)

	// Resolve twice
	clone1, err := r.ResolveSkill(context.Background(), "original")
	require.NoError(t, err)
	clone2, err := r.ResolveSkill(context.Background(), "original")
	require.NoError(t, err)

	// Modify clone1
	clone1.Name = "modified"
	clone1.Body = "Modified body"

	// clone2 should be unaffected
	assert.Equal(t, "original", clone2.Name)
	// Original should be unaffected
	assert.Equal(t, "original", original.Name)
}

func TestMapDocumentResolver_ImplementsInterface(t *testing.T) {
	var _ DocumentResolver = &MapDocumentResolver{}
	var _ DocumentResolver = &NoopDocumentResolver{}
}
