package prompty

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAgentSource = `---
name: test-agent
description: A test agent for unit testing
type: agent
execution:
  provider: openai
  model: gpt-4
  temperature: 0.7
messages:
  - role: system
    content: You are a helpful assistant.
  - role: user
    content: Hello
---
System prompt body.
`

const testNonAgentSource = `---
name: test-skill
description: A test skill
type: skill
---
Skill body content.
`

const testInvalidSyntaxSource = `---
name: test-agent
description: A test agent
type: agent
execution:
  provider: openai
  model: gpt-4
messages:
  - role: system
    content: Hello
---
{~prompty.var name="unclosed"
`

func TestNewAgentExecutor_Default(t *testing.T) {
	ae := NewAgentExecutor()
	assert.NotNil(t, ae)
	assert.Nil(t, ae.resolver)
	assert.Nil(t, ae.engine)
	assert.Equal(t, CatalogFormat(""), ae.skillsCatalogFormat)
	assert.Equal(t, CatalogFormat(""), ae.toolsCatalogFormat)
}

func TestNewAgentExecutor_WithOptions(t *testing.T) {
	resolver := NewMapDocumentResolver()
	engine := MustNew()

	ae := NewAgentExecutor(
		WithAgentResolver(resolver),
		WithAgentEngine(engine),
		WithAgentSkillsCatalogFormat(CatalogFormatDetailed),
		WithAgentToolsCatalogFormat(CatalogFormatFunctionCalling),
	)

	assert.NotNil(t, ae.resolver)
	assert.NotNil(t, ae.engine)
	assert.Equal(t, CatalogFormatDetailed, ae.skillsCatalogFormat)
	assert.Equal(t, CatalogFormatFunctionCalling, ae.toolsCatalogFormat)
}

func TestAgentExecutor_Execute_ValidAgent(t *testing.T) {
	ae := NewAgentExecutor()
	ctx := context.Background()

	compiled, err := ae.Execute(ctx, testAgentSource, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)
	assert.NotEmpty(t, compiled.Messages)
	assert.Equal(t, RoleSystem, compiled.Messages[0].Role)
}

func TestAgentExecutor_Execute_InvalidSource(t *testing.T) {
	ae := NewAgentExecutor()
	ctx := context.Background()

	// Invalid body syntax is caught during compilation (not parse)
	_, err := ae.Execute(ctx, testInvalidSyntaxSource, nil)
	require.Error(t, err)

	// Completely invalid YAML frontmatter should fail at parse
	_, err = ae.Execute(ctx, "not valid at all {~broken", nil)
	require.Error(t, err)
}

func TestAgentExecutor_Execute_NotAgent(t *testing.T) {
	ae := NewAgentExecutor()
	ctx := context.Background()

	_, err := ae.Execute(ctx, testNonAgentSource, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgNotAnAgent)
}

func TestAgentExecutor_ExecuteFile_Valid(t *testing.T) {
	// Write agent source to a temp file
	dir := t.TempDir()
	path := filepath.Join(dir, "test-agent.md")
	err := os.WriteFile(path, []byte(testAgentSource), FilesystemFilePermissions)
	require.NoError(t, err)

	ae := NewAgentExecutor()
	ctx := context.Background()

	compiled, err := ae.ExecuteFile(ctx, path, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)
	assert.NotEmpty(t, compiled.Messages)
}

func TestAgentExecutor_ExecuteFile_NotFound(t *testing.T) {
	ae := NewAgentExecutor()
	ctx := context.Background()

	_, err := ae.ExecuteFile(ctx, "/nonexistent/path/agent.md", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgAgentExecReadFile)
}

func TestAgentExecutor_ExecutePrompt_Valid(t *testing.T) {
	prompt := MustParse([]byte(testAgentSource))
	ae := NewAgentExecutor()
	ctx := context.Background()

	compiled, err := ae.ExecutePrompt(ctx, prompt, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)
	assert.NotEmpty(t, compiled.Messages)
}

func TestAgentExecutor_ExecutePrompt_NilPrompt(t *testing.T) {
	ae := NewAgentExecutor()
	ctx := context.Background()

	_, err := ae.ExecutePrompt(ctx, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgAgentExecNilPrompt)
}

func TestAgentExecutor_ActivateSkill_Valid(t *testing.T) {
	agentSource := `---
name: skill-agent
description: Agent with skills
type: agent
execution:
  provider: openai
  model: gpt-4
skills:
  - slug: web-search
    injection: system_prompt
messages:
  - role: system
    content: You are a search agent.
  - role: user
    content: Search for something.
---
Agent body.
`

	resolver := NewMapDocumentResolver()
	resolver.AddSkill("web-search", &Prompt{
		Name:        "web-search",
		Description: "Web search skill",
		Type:        DocumentTypeSkill,
		Body:        "Search results here.",
	})

	ae := NewAgentExecutor(WithAgentResolver(resolver))
	ctx := context.Background()

	compiled, err := ae.ActivateSkill(ctx, agentSource, "web-search", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)
	assert.NotEmpty(t, compiled.Messages)
}

func TestAgentExecutor_ActivateSkill_SkillNotFound(t *testing.T) {
	ae := NewAgentExecutor()
	ctx := context.Background()

	agentSource := `---
name: skill-agent
description: Agent with skills
type: agent
execution:
  provider: openai
  model: gpt-4
skills:
  - slug: web-search
    injection: system_prompt
messages:
  - role: system
    content: You are a search agent.
  - role: user
    content: Search for something.
---
Agent body.
`

	_, err := ae.ActivateSkill(ctx, agentSource, "nonexistent-skill", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgSkillNotFound)
}
