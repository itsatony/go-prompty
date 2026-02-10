package prompty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_AgentDocument(t *testing.T) {
	doc := `---
name: my-agent
description: A test agent
type: agent
execution:
  provider: openai
  model: gpt-4
skills:
  - slug: search-skill
    injection: system_prompt
tools:
  functions:
    - name: search
      description: Search the web
constraints:
  behavioral:
    - Be helpful
messages:
  - role: system
    content: You are a helpful assistant.
---
This is the body content.`

	p, err := Parse([]byte(doc))
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Equal(t, "my-agent", p.Name)
	assert.Equal(t, DocumentTypeAgent, p.Type)
	assert.Equal(t, "This is the body content.", p.Body)
	assert.True(t, p.IsAgent())

	// Execution
	require.NotNil(t, p.Execution)
	assert.Equal(t, ProviderOpenAI, p.Execution.Provider)
	assert.Equal(t, "gpt-4", p.Execution.Model)

	// Skills
	require.Len(t, p.Skills, 1)
	assert.Equal(t, "search-skill", p.Skills[0].Slug)
	assert.Equal(t, SkillInjectionSystemPrompt, p.Skills[0].Injection)

	// Tools
	require.NotNil(t, p.Tools)
	require.Len(t, p.Tools.Functions, 1)
	assert.Equal(t, "search", p.Tools.Functions[0].Name)

	// Constraints
	require.NotNil(t, p.Constraints)
	assert.Equal(t, []string{"Be helpful"}, p.Constraints.Behavioral)

	// Messages
	require.Len(t, p.Messages, 1)
	assert.Equal(t, RoleSystem, p.Messages[0].Role)
}

func TestParse_SkillDocument(t *testing.T) {
	doc := `---
name: my-skill
description: A test skill
type: skill
execution:
  provider: anthropic
  model: claude-sonnet-4-5
inputs:
  query:
    type: string
    required: true
---
{~prompty.var name="query" /~}`

	p, err := Parse([]byte(doc))
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Equal(t, "my-skill", p.Name)
	assert.Equal(t, DocumentTypeSkill, p.Type)
	assert.True(t, p.IsSkill())
	assert.Contains(t, p.Body, "prompty.var")
}

func TestParse_PromptDocument(t *testing.T) {
	doc := `---
name: simple-prompt
description: A simple prompt
type: prompt
---
Hello world!`

	p, err := Parse([]byte(doc))
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Equal(t, DocumentTypePrompt, p.Type)
	assert.True(t, p.IsPrompt())
	assert.Equal(t, "Hello world!", p.Body)
}

func TestParse_DefaultTypeIsSkill(t *testing.T) {
	doc := `---
name: no-type
description: No explicit type
---
body here`

	p, err := Parse([]byte(doc))
	require.NoError(t, err)
	require.NotNil(t, p)

	// Default type should be skill
	assert.Equal(t, DocumentTypeSkill, p.Type)
	assert.True(t, p.IsSkill())
}

func TestParse_NoFrontmatter(t *testing.T) {
	doc := `Just plain content, no frontmatter.`

	p, err := Parse([]byte(doc))
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Equal(t, DocumentTypeSkill, p.Type)
	assert.Equal(t, doc, p.Body)
	assert.Empty(t, p.Name)
}

func TestParse_EmptyInput(t *testing.T) {
	_, err := Parse([]byte{})
	require.Error(t, err)
}

func TestParse_UnclosedFrontmatter(t *testing.T) {
	doc := `---
name: broken
description: No closing delimiter`

	_, err := Parse([]byte(doc))
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgFrontmatterUnclosed)
}

func TestParse_ValidationError(t *testing.T) {
	// Name too long
	doc := `---
name: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
description: too long name
---
body`

	_, err := Parse([]byte(doc))
	require.Error(t, err)
}

func TestParse_PromptTypeWithSkillsFails(t *testing.T) {
	doc := `---
name: bad-prompt
description: A prompt with skills
type: prompt
skills:
  - slug: some-skill
---
body`

	_, err := Parse([]byte(doc))
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgPromptNoSkillsAllowed)
}

func TestParse_AgentWithContext(t *testing.T) {
	doc := `---
name: agent-with-context
description: Agent with context
type: agent
context:
  company: Acme Corp
  department: Engineering
---
Context: {~prompty.var name="context.company" /~}`

	p, err := Parse([]byte(doc))
	require.NoError(t, err)
	require.NotNil(t, p.Context)
	assert.Equal(t, "Acme Corp", p.Context["company"])
	assert.Equal(t, "Engineering", p.Context["department"])
}
