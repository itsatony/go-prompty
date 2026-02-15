package prompty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportToSkillMD(t *testing.T) {
	tests := []struct {
		name   string
		prompt *Prompt
		body   string
		want   []string // strings that should be present in output
	}{
		{
			name:   "nil prompt",
			prompt: nil,
			body:   "Hello world",
			want:   []string{"Hello world"},
		},
		{
			name: "minimal prompt",
			prompt: &Prompt{
				Name:        "test-prompt",
				Description: "A test prompt",
			},
			body: "Hello {~prompty.var name=\"user\" /~}",
			want: []string{
				"---",
				"name: test-prompt",
				"description: A test prompt",
				"Hello {~prompty.var name=\"user\" /~}",
			},
		},
		{
			name: "full prompt with extensions stripped",
			prompt: &Prompt{
				Name:          "test-prompt",
				Description:   "A test prompt",
				License:       "MIT",
				Compatibility: "gpt-4",
				AllowedTools:  "calculator",
				Metadata: map[string]any{
					"author": "test",
				},
				Execution: &ExecutionConfig{
					Provider: ProviderOpenAI,
					Model:    "gpt-4",
				},
				Inputs: map[string]*InputDef{
					"query": {Type: "string", Required: true},
				},
				Sample: map[string]any{
					"query": "test query",
				},
			},
			body: "Hello world",
			want: []string{
				"name: test-prompt",
				"description: A test prompt",
				"license: MIT",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.prompt.ExportToSkillMD(tt.body)
			require.NoError(t, err)

			for _, s := range tt.want {
				assert.Contains(t, result, s)
			}

			// Execution should NOT be in output
			if tt.prompt != nil && tt.prompt.Execution != nil {
				assert.NotContains(t, result, "execution:")
			}
		})
	}
}

func TestImportFromSkillMD(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		check   func(t *testing.T, result *SkillMD)
	}{
		{
			name:    "empty content",
			content: "",
			wantErr: true,
		},
		{
			name:    "no frontmatter",
			content: "Hello world",
			wantErr: true,
		},
		{
			name:    "unclosed frontmatter",
			content: "---\nname: test\n",
			wantErr: true,
		},
		{
			name: "valid minimal",
			content: `---
name: test-prompt
description: A test prompt
---
Hello world`,
			check: func(t *testing.T, result *SkillMD) {
				require.NotNil(t, result.Prompt)
				assert.Equal(t, "test-prompt", result.Prompt.Name)
				assert.Equal(t, "A test prompt", result.Prompt.Description)
				assert.Equal(t, "Hello world", result.Body)
			},
		},
		{
			name: "valid with inputs",
			content: `---
name: test-prompt
description: A test prompt
inputs:
  query:
    type: string
    required: true
---
Query: {~prompty.var name="query" /~}`,
			check: func(t *testing.T, result *SkillMD) {
				require.NotNil(t, result.Prompt)
				assert.Equal(t, "test-prompt", result.Prompt.Name)
				require.NotNil(t, result.Prompt.Inputs)
				assert.Contains(t, result.Prompt.Inputs, "query")
				assert.Contains(t, result.Body, "prompty.var")
			},
		},
		{
			name:    "with BOM",
			content: "\xef\xbb\xbf---\nname: test\ndescription: test\n---\nbody",
			check: func(t *testing.T, result *SkillMD) {
				require.NotNil(t, result.Prompt)
				assert.Equal(t, "test", result.Prompt.Name)
			},
		},
		{
			name: "empty body",
			content: `---
name: test-prompt
description: A test
---`,
			check: func(t *testing.T, result *SkillMD) {
				require.NotNil(t, result.Prompt)
				assert.Empty(t, result.Body)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ImportFromSkillMD(tt.content)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestSkillMD_RoundTrip(t *testing.T) {
	// Create original prompt
	original := &Prompt{
		Name:          "test-prompt",
		Description:   "A test prompt for round-trip testing",
		License:       "MIT",
		Compatibility: "gpt-4",
		AllowedTools:  "calculator",
		Inputs: map[string]*InputDef{
			"query": {Type: "string", Required: true},
		},
		Outputs: map[string]*OutputDef{
			"result": {Type: "string"},
		},
		Sample: map[string]any{
			"query": "test query",
		},
	}
	originalBody := "Hello {~prompty.var name=\"query\" /~}!"

	// Export to SKILL.md
	exported, err := original.ExportToSkillMD(originalBody)
	require.NoError(t, err)

	// Import back
	imported, err := ImportFromSkillMD(exported)
	require.NoError(t, err)

	// Verify round-trip
	assert.Equal(t, original.Name, imported.Prompt.Name)
	assert.Equal(t, original.Description, imported.Prompt.Description)
	assert.Equal(t, original.License, imported.Prompt.License)
	assert.Equal(t, original.Compatibility, imported.Prompt.Compatibility)
	assert.Equal(t, original.AllowedTools, imported.Prompt.AllowedTools)
	assert.Equal(t, originalBody, imported.Body)

	// Extensions should not be present after round-trip
	assert.Nil(t, imported.Prompt.Execution)
	assert.Nil(t, imported.Prompt.Extensions)
}

func TestSkillMD_ToSource(t *testing.T) {
	skillMD := &SkillMD{
		Prompt: &Prompt{
			Name:        "test",
			Description: "test",
		},
		Body: "Hello world",
	}

	source, err := skillMD.ToSource()
	require.NoError(t, err)
	assert.Contains(t, source, "---")
	assert.Contains(t, source, "name: test")
	assert.Contains(t, source, "Hello world")
}

func TestSkillMD_WithPrompt(t *testing.T) {
	original := &SkillMD{
		Prompt: &Prompt{Name: "old"},
		Body:   "body",
	}

	newPrompt := &Prompt{Name: "new"}
	result := original.WithPrompt(newPrompt)

	assert.Equal(t, "new", result.Prompt.Name)
	assert.Equal(t, "body", result.Body)
	// Original should not be modified
	assert.Equal(t, "old", original.Prompt.Name)
}

func TestSkillMD_WithBody(t *testing.T) {
	original := &SkillMD{
		Prompt: &Prompt{Name: "test"},
		Body:   "old body",
	}

	result := original.WithBody("new body")

	assert.Equal(t, "test", result.Prompt.Name)
	assert.Equal(t, "new body", result.Body)
	// Original should not be modified
	assert.Equal(t, "old body", original.Body)
}

func TestSkillMD_Clone(t *testing.T) {
	original := &SkillMD{
		Prompt: &Prompt{
			Name:        "test",
			Description: "test",
		},
		Body: "body",
	}

	clone := original.Clone()

	assert.Equal(t, original.Prompt.Name, clone.Prompt.Name)
	assert.Equal(t, original.Body, clone.Body)

	// Verify deep copy
	clone.Prompt.Name = "modified"
	assert.NotEqual(t, original.Prompt.Name, clone.Prompt.Name)
}

func TestSkillMD_MergeExecution(t *testing.T) {
	skillMD := &SkillMD{
		Prompt: &Prompt{
			Name:        "test",
			Description: "test",
		},
		Body: "body",
	}

	exec := &ExecutionConfig{
		Provider: ProviderOpenAI,
		Model:    "gpt-4",
	}

	result := skillMD.MergeExecution(exec)

	assert.Equal(t, "test", result.Name)
	assert.NotNil(t, result.Execution)
	assert.Equal(t, ProviderOpenAI, result.Execution.Provider)

	// Original should not be modified
	assert.Nil(t, skillMD.Prompt.Execution)
}

func TestSkillMD_NilHandling(t *testing.T) {
	var nilSkillMD *SkillMD

	source, err := nilSkillMD.ToSource()
	assert.NoError(t, err)
	assert.Empty(t, source)

	result := nilSkillMD.WithPrompt(&Prompt{Name: "test"})
	assert.NotNil(t, result)
	assert.Equal(t, "test", result.Prompt.Name)

	result = nilSkillMD.WithBody("body")
	assert.NotNil(t, result)
	assert.Equal(t, "body", result.Body)

	clone := nilSkillMD.Clone()
	assert.Nil(t, clone)

	exec := nilSkillMD.MergeExecution(&ExecutionConfig{Provider: ProviderOpenAI})
	assert.NotNil(t, exec.Execution)
}
