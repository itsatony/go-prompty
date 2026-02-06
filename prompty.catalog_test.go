package prompty

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSkillsCatalog_Empty(t *testing.T) {
	result, err := GenerateSkillsCatalog(context.Background(), nil, nil, CatalogFormatDefault)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGenerateSkillsCatalog_Default(t *testing.T) {
	skills := []SkillRef{
		{Slug: "search-skill"},
		{Slug: "calc-skill"},
	}

	resolver := NewMapDocumentResolver()
	resolver.AddSkill("search-skill", &Prompt{
		Name:        "search-skill",
		Description: "Search the web",
		Type:        DocumentTypeSkill,
	})
	resolver.AddSkill("calc-skill", &Prompt{
		Name:        "calc-skill",
		Description: "Calculate things",
		Type:        DocumentTypeSkill,
	})

	result, err := GenerateSkillsCatalog(context.Background(), skills, resolver, CatalogFormatDefault)
	require.NoError(t, err)
	assert.Contains(t, result, "## Available Skills")
	assert.Contains(t, result, "**search-skill**")
	assert.Contains(t, result, "Search the web")
	assert.Contains(t, result, "**calc-skill**")
	assert.Contains(t, result, "Calculate things")
}

func TestGenerateSkillsCatalog_Detailed(t *testing.T) {
	skills := []SkillRef{
		{Slug: "search-skill", Version: "v2", Injection: SkillInjectionSystemPrompt},
	}

	resolver := NewMapDocumentResolver()
	resolver.AddSkill("search-skill", &Prompt{
		Name:        "search-skill",
		Description: "Search the web for answers",
		Type:        DocumentTypeSkill,
	})

	result, err := GenerateSkillsCatalog(context.Background(), skills, resolver, CatalogFormatDetailed)
	require.NoError(t, err)
	assert.Contains(t, result, "### search-skill (vv2)")
	assert.Contains(t, result, "Search the web for answers")
	assert.Contains(t, result, "Injection: system_prompt")
}

func TestGenerateSkillsCatalog_Compact(t *testing.T) {
	skills := []SkillRef{
		{Slug: "skill-a"},
		{Slug: "skill-b"},
	}

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

	result, err := GenerateSkillsCatalog(context.Background(), skills, resolver, CatalogFormatCompact)
	require.NoError(t, err)
	assert.Contains(t, result, "skill-a - First skill")
	assert.Contains(t, result, "skill-b - Second skill")
	assert.Contains(t, result, "; ")
}

func TestGenerateSkillsCatalog_InlineSkill(t *testing.T) {
	skills := []SkillRef{
		{
			Inline: &InlineSkill{
				Slug:        "inline-skill",
				Description: "An inline skill",
				Body:        "Hello",
			},
		},
	}

	result, err := GenerateSkillsCatalog(context.Background(), skills, nil, CatalogFormatDefault)
	require.NoError(t, err)
	assert.Contains(t, result, "**inline-skill**")
	assert.Contains(t, result, "An inline skill")
}

func TestGenerateSkillsCatalog_ResolverError(t *testing.T) {
	skills := []SkillRef{
		{Slug: "nonexistent"},
	}

	// NoopDocumentResolver will fail, but catalog generation is non-fatal
	resolver := &NoopDocumentResolver{}
	result, err := GenerateSkillsCatalog(context.Background(), skills, resolver, CatalogFormatDefault)
	require.NoError(t, err)
	// Should still include the slug even without description
	assert.Contains(t, result, "**nonexistent**")
}

func TestGenerateSkillsCatalog_InvalidFormat(t *testing.T) {
	skills := []SkillRef{{Slug: "s"}}
	_, err := GenerateSkillsCatalog(context.Background(), skills, nil, CatalogFormat("invalid"))
	require.Error(t, err)
}

func TestGenerateToolsCatalog_Empty(t *testing.T) {
	result, err := GenerateToolsCatalog(nil, CatalogFormatDefault)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGenerateToolsCatalog_Default(t *testing.T) {
	tools := &ToolsConfig{
		Functions: []*FunctionDef{
			{Name: "search", Description: "Search the web"},
			{Name: "calculate", Description: "Do math"},
		},
		MCPServers: []*MCPServer{
			{Name: "docs-server", Tools: []string{"read_doc", "list_docs"}},
		},
	}

	result, err := GenerateToolsCatalog(tools, CatalogFormatDefault)
	require.NoError(t, err)
	assert.Contains(t, result, "## Available Tools")
	assert.Contains(t, result, "**search**")
	assert.Contains(t, result, "Search the web")
	assert.Contains(t, result, "**[MCP] docs-server**")
	assert.Contains(t, result, "read_doc, list_docs")
}

func TestGenerateToolsCatalog_Detailed(t *testing.T) {
	tools := &ToolsConfig{
		Functions: []*FunctionDef{
			{
				Name:        "get-user",
				Description: "Get a user by ID",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id": map[string]any{"type": "string"},
					},
				},
			},
		},
		MCPServers: []*MCPServer{
			{Name: "api", URL: "https://api.example.com", Transport: "sse"},
		},
	}

	result, err := GenerateToolsCatalog(tools, CatalogFormatDetailed)
	require.NoError(t, err)
	assert.Contains(t, result, "### get-user")
	assert.Contains(t, result, "Get a user by ID")
	assert.Contains(t, result, "**Parameters:**")
	assert.Contains(t, result, "### [MCP] api")
	assert.Contains(t, result, "https://api.example.com")
	assert.Contains(t, result, "Transport: sse")
}

func TestGenerateToolsCatalog_Compact(t *testing.T) {
	tools := &ToolsConfig{
		Functions: []*FunctionDef{
			{Name: "fn1", Description: "Function one"},
		},
		MCPServers: []*MCPServer{
			{Name: "srv1"},
		},
	}

	result, err := GenerateToolsCatalog(tools, CatalogFormatCompact)
	require.NoError(t, err)
	assert.Contains(t, result, "fn1 - Function one")
	assert.Contains(t, result, "[MCP] srv1")
}

func TestGenerateToolsCatalog_FunctionCalling(t *testing.T) {
	tools := &ToolsConfig{
		Functions: []*FunctionDef{
			{
				Name:        "search",
				Description: "Search the web",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{"type": "string"},
					},
				},
			},
		},
	}

	result, err := GenerateToolsCatalog(tools, CatalogFormatFunctionCalling)
	require.NoError(t, err)
	assert.Contains(t, result, `"type": "function"`)
	assert.Contains(t, result, `"name": "search"`)
	assert.Contains(t, result, `"description": "Search the web"`)
}

func TestGenerateToolsCatalog_InvalidFormat(t *testing.T) {
	tools := &ToolsConfig{
		Functions: []*FunctionDef{{Name: "fn"}},
	}
	_, err := GenerateToolsCatalog(tools, CatalogFormat("bad"))
	require.Error(t, err)
}

func TestTruncateString(t *testing.T) {
	assert.Equal(t, "hello", truncateString("hello", 10))
	assert.Equal(t, "hello", truncateString("hello", 5))
	assert.Equal(t, "he...", truncateString("hello world", 5))
	assert.Equal(t, "hel", truncateString("hello", 3))
	assert.Equal(t, "h", truncateString("hello", 1))
}
