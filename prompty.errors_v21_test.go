package prompty

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Compilation error constant tests ---

func TestCompileErrors_NotAgent(t *testing.T) {
	p := &Prompt{
		Name:        "skill",
		Description: "A skill",
		Type:        DocumentTypeSkill,
	}
	_, err := p.CompileAgent(context.Background(), nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgCompileNotAgent)
}

func TestCompileErrors_NilPrompt(t *testing.T) {
	var p *Prompt
	_, err := p.Compile(context.Background(), nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgCompilationFailed)

	_, err = p.CompileAgent(context.Background(), nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgCompilationFailed)
}

func TestCompileErrors_BodyFailed(t *testing.T) {
	p := &Prompt{
		Name:        "bad-body",
		Description: "Prompt with bad template",
		Type:        DocumentTypeSkill,
		Body:        "{~prompty.var name=\"nonexistent\" onerror=\"throw\" /~}",
	}
	_, err := p.Compile(context.Background(), nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgCompileBodyFailed)
}

func TestCompileErrors_AgentBodyFailed(t *testing.T) {
	p := &Prompt{
		Name:        "bad-agent",
		Description: "Agent with bad body",
		Type:        DocumentTypeAgent,
		Body:        "{~prompty.var name=\"nonexistent\" onerror=\"throw\" /~}",
	}
	_, err := p.CompileAgent(context.Background(), nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgCompileBodyFailed)
}

func TestCompileErrors_MessageFailed(t *testing.T) {
	p := &Prompt{
		Name:        "msg-agent",
		Description: "Agent with bad message template",
		Type:        DocumentTypeAgent,
		Body:        "body",
		Messages: []MessageTemplate{
			{
				Role:    RoleSystem,
				Content: "{~prompty.var name=\"nonexistent\" onerror=\"throw\" /~}",
			},
		},
	}
	_, err := p.CompileAgent(context.Background(), nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgCompileMessageFailed)
}

func TestCompileErrors_ActivateSkillNotFound(t *testing.T) {
	p := &Prompt{
		Name:        "agent",
		Description: "Agent",
		Type:        DocumentTypeAgent,
		Skills:      []SkillRef{{Slug: "existing"}},
		Body:        "body",
	}
	_, err := p.ActivateSkill(context.Background(), "nonexistent", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgSkillNotFound)
}

func TestValidateForExecution_ErrorConstants(t *testing.T) {
	// No execution config
	p := &Prompt{Name: "test", Description: "test"}
	err := p.ValidateForExecution()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgNoExecutionConfig)

	// No provider
	p.Execution = &ExecutionConfig{Model: "gpt-4"}
	err = p.ValidateForExecution()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgNoProvider)

	// No model
	p.Execution = &ExecutionConfig{Provider: ProviderOpenAI}
	err = p.ValidateForExecution()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgNoModel)
}

// --- Versioning error constructor tests ---

func TestNewVersioningError_WithCause(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := NewVersioningError(ErrMsgVersionSaveRollback, cause)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgVersionSaveRollback)
	assert.True(t, errors.Is(err, cause) || errors.Unwrap(err) != nil)
}

func TestNewVersioningError_NilCause(t *testing.T) {
	err := NewVersioningError(ErrMsgVersionMinimumRequired, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgVersionMinimumRequired)
}

func TestNewVersionGetError(t *testing.T) {
	cause := fmt.Errorf("not found")
	err := NewVersionGetError(42, cause)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgVersionGetFailed)
}

func TestNewVersionTemplateExistsError(t *testing.T) {
	err := NewVersionTemplateExistsError("my-template")
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgVersionTemplateExists)
}

// --- Agent error constructor tests ---

func TestNewAgentError_WithCause(t *testing.T) {
	cause := fmt.Errorf("something failed")
	err := NewAgentError(ErrMsgNotAnAgent, cause)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgNotAnAgent)
}

func TestNewAgentError_NilCause(t *testing.T) {
	err := NewAgentError(ErrMsgNotAnAgent, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgNotAnAgent)
}

func TestNewAgentValidationError(t *testing.T) {
	err := NewAgentValidationError(ErrMsgAgentMessagesInvalid, "my-agent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgAgentMessagesInvalid)
}

func TestNewCompilationError_WithCause(t *testing.T) {
	cause := fmt.Errorf("template error")
	err := NewCompilationError(ErrMsgCompileBodyFailed, cause)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgCompileBodyFailed)
}

func TestNewCompilationError_NilCause(t *testing.T) {
	err := NewCompilationError(ErrMsgCompilationFailed, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgCompilationFailed)
}

func TestNewCatalogError_WithCause(t *testing.T) {
	cause := fmt.Errorf("catalog error")
	err := NewCatalogError(ErrMsgCatalogGenerationFailed, cause)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgCatalogGenerationFailed)
}

func TestNewCatalogError_NilCause(t *testing.T) {
	err := NewCatalogError(ErrMsgCatalogGenerationFailed, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgCatalogGenerationFailed)
}

func TestNewSkillNotFoundError(t *testing.T) {
	err := NewSkillNotFoundError("search-skill")
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgSkillNotFound)
}

func TestNewInvalidDocumentTypeError(t *testing.T) {
	err := NewInvalidDocumentTypeError("bogus")
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgInvalidDocumentType)
}

// --- Template convenience method error tests ---

func TestTemplate_Compile_NoPrompt(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse("Hello world")
	require.NoError(t, err)
	require.False(t, tmpl.HasPrompt())

	_, compileErr := tmpl.Compile(context.Background(), nil, nil)
	require.Error(t, compileErr)
	assert.Contains(t, compileErr.Error(), ErrMsgCompilationFailed)
}

func TestTemplate_CompileAgent_NoPrompt(t *testing.T) {
	engine := MustNew()
	tmpl, err := engine.Parse("Hello world")
	require.NoError(t, err)

	_, compileErr := tmpl.CompileAgent(context.Background(), nil, nil)
	require.Error(t, compileErr)
	assert.Contains(t, compileErr.Error(), ErrMsgCompilationFailed)
}

// --- CompileOptions Engine field test ---

func TestCompileOptions_WithEngine(t *testing.T) {
	engine := MustNew()
	p := &Prompt{
		Name:        "with-engine",
		Description: "Prompt compiled with custom engine",
		Type:        DocumentTypeSkill,
		Body:        "Hello, world!",
	}

	opts := &CompileOptions{Engine: engine}
	result, err := p.Compile(context.Background(), nil, opts)
	require.NoError(t, err)
	assert.Equal(t, "Hello, world!", result)
}

func TestCompileAgent_WithEngine(t *testing.T) {
	engine := MustNew()
	p := &Prompt{
		Name:        "agent-with-engine",
		Description: "Agent compiled with custom engine",
		Type:        DocumentTypeAgent,
		Body:        "System prompt.",
	}

	opts := &CompileOptions{Engine: engine}
	compiled, err := p.CompileAgent(context.Background(), nil, opts)
	require.NoError(t, err)
	require.NotNil(t, compiled)
	require.Len(t, compiled.Messages, 1)
	assert.Equal(t, "System prompt.", compiled.Messages[0].Content)
}
