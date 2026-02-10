package prompty

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_ParseAndCompileAgent(t *testing.T) {
	doc := `---
name: customer-service-agent
description: A customer service agent
type: agent
execution:
  provider: openai
  model: gpt-4
  temperature: 0.7
context:
  company: Acme Corp
  department: Support
skills:
  - slug: search-skill
    injection: system_prompt
tools:
  functions:
    - name: lookup-customer
      description: Look up customer by ID
constraints:
  behavioral:
    - Be helpful and professional
    - Keep responses concise
  safety:
    - Never share PII
messages:
  - role: system
    content: 'You are {~prompty.var name="meta.name" /~}, working for {~prompty.var name="context.company" /~}.'
  - role: user
    content: '{~prompty.var name="input.message" default="Hello" /~}'
---
You are a customer service agent for {~prompty.var name="company" /~}.
Department: {~prompty.var name="department" /~}.`

	// Parse
	p, err := Parse([]byte(doc))
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Equal(t, DocumentTypeAgent, p.Type)
	assert.True(t, p.IsAgent())
	assert.Equal(t, "customer-service-agent", p.Name)

	// Verify parsed fields
	require.NotNil(t, p.Execution)
	assert.Equal(t, ProviderOpenAI, p.Execution.Provider)

	require.NotNil(t, p.Context)
	assert.Equal(t, "Acme Corp", p.Context["company"])

	require.Len(t, p.Skills, 1)
	assert.Equal(t, "search-skill", p.Skills[0].Slug)

	require.NotNil(t, p.Tools)
	require.Len(t, p.Tools.Functions, 1)

	require.NotNil(t, p.Constraints)
	assert.Len(t, p.Constraints.Behavioral, 2)

	require.Len(t, p.Messages, 2)

	// Body should contain the template
	assert.Contains(t, p.Body, "customer service agent")

	// Compile agent
	input := map[string]any{"message": "I need help with my order."}
	compiled, err := p.CompileAgent(context.Background(), input, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// Should have 2 compiled messages
	require.Len(t, compiled.Messages, 2)

	// System message should have resolved variables
	assert.Contains(t, compiled.Messages[0].Content, "customer-service-agent")
	assert.Contains(t, compiled.Messages[0].Content, "Acme Corp")

	// User message should have the input
	assert.Equal(t, "I need help with my order.", compiled.Messages[1].Content)

	// Execution should be preserved
	require.NotNil(t, compiled.Execution)
	assert.Equal(t, ProviderOpenAI, compiled.Execution.Provider)

	// Tools should be preserved
	require.NotNil(t, compiled.Tools)
	assert.Len(t, compiled.Tools.Functions, 1)
}

func TestE2E_ParseAndCompileSkill(t *testing.T) {
	doc := `---
name: greeting-skill
description: A greeting skill
type: skill
execution:
  provider: anthropic
  model: claude-sonnet-4-5
inputs:
  name:
    type: string
    required: true
---
Hello, {~prompty.var name="name" /~}! Welcome.`

	p, err := Parse([]byte(doc))
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Equal(t, DocumentTypeSkill, p.Type)
	assert.True(t, p.IsSkill())

	result, err := p.Compile(context.Background(), map[string]any{"name": "Alice"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "Hello, Alice! Welcome.", result)
}

func TestE2E_ParseAndCompilePrompt(t *testing.T) {
	doc := `---
name: simple-prompt
description: A simple prompt
type: prompt
---
This is a simple prompt with no dynamic content.`

	p, err := Parse([]byte(doc))
	require.NoError(t, err)

	assert.Equal(t, DocumentTypePrompt, p.Type)
	assert.True(t, p.IsPrompt())

	result, err := p.Compile(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "This is a simple prompt with no dynamic content.", result)
}

func TestE2E_AgentWithSkillActivation(t *testing.T) {
	// Agent document
	agentDoc := `---
name: research-agent
description: A research agent
type: agent
execution:
  provider: openai
  model: gpt-4
skills:
  - slug: search-skill
    injection: system_prompt
  - slug: summarize-skill
    injection: user_context
---
You are a research assistant.`

	agent, err := Parse([]byte(agentDoc))
	require.NoError(t, err)

	// Set up resolver with skills
	resolver := NewMapDocumentResolver()
	resolver.AddSkill("search-skill", &Prompt{
		Name:        "search-skill",
		Description: "Search the web",
		Type:        DocumentTypeSkill,
		Body:        "Use search tools to find relevant information.",
	})
	resolver.AddSkill("summarize-skill", &Prompt{
		Name:        "summarize-skill",
		Description: "Summarize content",
		Type:        DocumentTypeSkill,
		Body:        "Please summarize the following content.",
	})

	opts := &CompileOptions{Resolver: resolver}

	// Activate search skill
	compiled, err := agent.ActivateSkill(context.Background(), "search-skill", nil, opts)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// System message should contain the injected search skill
	systemMsg := compiled.Messages[0]
	assert.Equal(t, RoleSystem, systemMsg.Role)
	assert.Contains(t, systemMsg.Content, "research assistant")
	assert.Contains(t, systemMsg.Content, "Use search tools")

	// Activate summarize skill
	compiled2, err := agent.ActivateSkill(context.Background(), "summarize-skill", nil, opts)
	require.NoError(t, err)

	// Should have a user context message with summarize skill
	hasUserCtx := false
	for _, msg := range compiled2.Messages {
		if msg.Role == RoleUser && msg.Content != "" {
			if containsSubstring(msg.Content, "summarize the following") {
				hasUserCtx = true
			}
		}
	}
	assert.True(t, hasUserCtx, "expected user context message with summarize skill")
}

func TestE2E_AgentConstraintsInBody(t *testing.T) {
	doc := `---
name: constrained-agent
description: Agent with constraints in body
type: agent
constraints:
  behavioral:
    - Be helpful
    - Be concise
---
You must follow these rules:
{~prompty.for item="c" in="constraints.behavioral"~}
- {~prompty.var name="c" /~}
{~/prompty.for~}`

	agent, err := Parse([]byte(doc))
	require.NoError(t, err)

	compiled, err := agent.CompileAgent(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)
	require.Len(t, compiled.Messages, 1)

	content := compiled.Messages[0].Content
	assert.Contains(t, content, "Be helpful")
	assert.Contains(t, content, "Be concise")
}

func TestE2E_ThreeLayerExecutionMerge(t *testing.T) {
	agentTemp := 0.3
	agentMax := 2000
	agentDoc := `---
name: merge-test-agent
description: Agent for testing 3-layer merge
type: agent
execution:
  provider: openai
  model: gpt-4
  temperature: 0.3
  max_tokens: 2000
skills:
  - slug: merge-skill
    execution:
      model: gpt-4-turbo
---
body`

	agent, err := Parse([]byte(agentDoc))
	require.NoError(t, err)

	// Verify parsed values
	require.NotNil(t, agent.Execution)
	assert.Equal(t, agentTemp, *agent.Execution.Temperature)
	assert.Equal(t, agentMax, *agent.Execution.MaxTokens)

	skillTemp := 0.8
	resolver := NewMapDocumentResolver()
	resolver.AddSkill("merge-skill", &Prompt{
		Name:        "merge-skill",
		Description: "Merge skill",
		Type:        DocumentTypeSkill,
		Body:        "skill body",
		Execution: &ExecutionConfig{
			Temperature: &skillTemp,
		},
	})

	opts := &CompileOptions{Resolver: resolver}
	compiled, err := agent.ActivateSkill(context.Background(), "merge-skill", nil, opts)
	require.NoError(t, err)
	require.NotNil(t, compiled.Execution)

	// Provider: from agent (not overridden)
	assert.Equal(t, ProviderOpenAI, compiled.Execution.Provider)
	// Model: from skill ref execution (overrides agent)
	assert.Equal(t, "gpt-4-turbo", compiled.Execution.Model)
	// Temperature: from resolved skill (overrides agent)
	assert.Equal(t, 0.8, *compiled.Execution.Temperature)
	// MaxTokens: from agent (not overridden by skill)
	assert.Equal(t, 2000, *compiled.Execution.MaxTokens)
}

func TestE2E_SkillsCatalogInBody(t *testing.T) {
	doc := `---
name: catalog-agent
description: Agent with catalog in body
type: agent
skills:
  - slug: skill-a
  - slug: skill-b
---
Here are your available skills:
{~prompty.skills_catalog /~}`

	agent, err := Parse([]byte(doc))
	require.NoError(t, err)

	resolver := NewMapDocumentResolver()
	resolver.AddSkill("skill-a", &Prompt{
		Name:        "skill-a",
		Description: "First skill",
		Type:        DocumentTypeSkill,
	})
	resolver.AddSkill("skill-b", &Prompt{
		Name:        "skill-b",
		Description: "Second skill",
		Type:        DocumentTypeSkill,
	})

	opts := &CompileOptions{Resolver: resolver}
	compiled, err := agent.CompileAgent(context.Background(), nil, opts)
	require.NoError(t, err)
	require.NotNil(t, compiled)
	require.Len(t, compiled.Messages, 1)

	content := compiled.Messages[0].Content
	assert.Contains(t, content, "Available Skills")
	assert.Contains(t, content, "skill-a")
	assert.Contains(t, content, "skill-b")
}

func TestE2E_ToolsCatalogInBody(t *testing.T) {
	doc := `---
name: tools-agent
description: Agent with tools catalog
type: agent
tools:
  functions:
    - name: search
      description: Search the web
    - name: calculate
      description: Do math
---
You have the following tools:
{~prompty.tools_catalog /~}`

	agent, err := Parse([]byte(doc))
	require.NoError(t, err)

	compiled, err := agent.CompileAgent(context.Background(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	content := compiled.Messages[0].Content
	assert.Contains(t, content, "Available Tools")
	assert.Contains(t, content, "search")
	assert.Contains(t, content, "calculate")
}
