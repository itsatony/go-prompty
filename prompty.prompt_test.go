package prompty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrompt_Validate(t *testing.T) {
	tests := []struct {
		name    string
		prompt  *Prompt
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil prompt",
			prompt:  nil,
			wantErr: true,
			errMsg:  ErrMsgPromptNameRequired,
		},
		{
			name: "empty name",
			prompt: &Prompt{
				Description: "A description",
			},
			wantErr: true,
			errMsg:  ErrMsgPromptNameRequired,
		},
		{
			name: "name too long",
			prompt: &Prompt{
				Name:        "this-is-a-very-long-name-that-exceeds-the-maximum-length-allowed-for-prompts",
				Description: "A description",
			},
			wantErr: true,
			errMsg:  ErrMsgPromptNameTooLong,
		},
		{
			name: "invalid name format - uppercase",
			prompt: &Prompt{
				Name:        "MyPrompt",
				Description: "A description",
			},
			wantErr: true,
			errMsg:  ErrMsgPromptNameInvalidFormat,
		},
		{
			name: "invalid name format - starts with number",
			prompt: &Prompt{
				Name:        "1prompt",
				Description: "A description",
			},
			wantErr: true,
			errMsg:  ErrMsgPromptNameInvalidFormat,
		},
		{
			name: "invalid name format - underscore",
			prompt: &Prompt{
				Name:        "my_prompt",
				Description: "A description",
			},
			wantErr: true,
			errMsg:  ErrMsgPromptNameInvalidFormat,
		},
		{
			name: "empty description",
			prompt: &Prompt{
				Name: "my-prompt",
			},
			wantErr: true,
			errMsg:  ErrMsgPromptDescriptionRequired,
		},
		{
			name: "description too long",
			prompt: &Prompt{
				Name:        "my-prompt",
				Description: string(make([]byte, PromptDescriptionMaxLength+1)),
			},
			wantErr: true,
			errMsg:  ErrMsgPromptDescriptionTooLong,
		},
		{
			name: "valid prompt - minimal",
			prompt: &Prompt{
				Name:        "my-prompt",
				Description: "A valid description",
			},
			wantErr: false,
		},
		{
			name: "valid prompt - with hyphens",
			prompt: &Prompt{
				Name:        "my-complex-prompt-name",
				Description: "A valid description",
			},
			wantErr: false,
		},
		{
			name: "valid prompt - with numbers",
			prompt: &Prompt{
				Name:        "prompt123",
				Description: "A valid description",
			},
			wantErr: false,
		},
		{
			name: "valid prompt - full",
			prompt: &Prompt{
				Name:          "my-prompt",
				Description:   "A valid description",
				License:       "MIT",
				Compatibility: "gpt-4,claude-3",
				AllowedTools:  "calculator,search",
				Metadata: map[string]any{
					"author": "test",
				},
				Execution: &ExecutionConfig{
					Provider: "openai",
					Model:    "gpt-4",
				},
				Skope: &SkopeConfig{
					Visibility: SkopeVisibilityPublic,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.prompt.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPrompt_ValidateOptional(t *testing.T) {
	tests := []struct {
		name    string
		prompt  *Prompt
		wantErr bool
	}{
		{
			name:    "nil prompt",
			prompt:  nil,
			wantErr: false,
		},
		{
			name: "empty name - skipped",
			prompt: &Prompt{
				Description: "A description",
			},
			wantErr: false,
		},
		{
			name: "has name - validates",
			prompt: &Prompt{
				Name:        "my-prompt",
				Description: "A description",
			},
			wantErr: false,
		},
		{
			name: "has name but invalid",
			prompt: &Prompt{
				Name:        "MyPrompt",
				Description: "A description",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.prompt.ValidateOptional()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseYAMLPrompt(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    *Prompt
		wantErr bool
	}{
		{
			name: "empty yaml",
			yaml: "",
			want: nil,
		},
		{
			name: "minimal prompt",
			yaml: `name: my-prompt
description: A test prompt`,
			want: &Prompt{
				Name:        "my-prompt",
				Description: "A test prompt",
			},
		},
		{
			name: "full prompt",
			yaml: `name: my-prompt
description: A test prompt
license: MIT
compatibility: gpt-4
allowed_tools: calculator
metadata:
  author: test
execution:
  provider: openai
  model: gpt-4
  temperature: 0.7
skope:
  visibility: public
inputs:
  query:
    type: string
    required: true
outputs:
  result:
    type: string
sample:
  query: "test query"`,
			want: &Prompt{
				Name:          "my-prompt",
				Description:   "A test prompt",
				License:       "MIT",
				Compatibility: "gpt-4",
				AllowedTools:  "calculator",
				Metadata: map[string]any{
					"author": "test",
				},
				Execution: &ExecutionConfig{
					Provider: "openai",
					Model:    "gpt-4",
				},
				Skope: &SkopeConfig{
					Visibility: "public",
				},
				Inputs: map[string]*InputDef{
					"query": {Type: "string", Required: true},
				},
				Outputs: map[string]*OutputDef{
					"result": {Type: "string"},
				},
				Sample: map[string]any{
					"query": "test query",
				},
			},
		},
		{
			name:    "invalid yaml",
			yaml:    "invalid: yaml: content:",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseYAMLPrompt(tt.yaml)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.want == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.want.Name, got.Name)
			assert.Equal(t, tt.want.Description, got.Description)
			assert.Equal(t, tt.want.License, got.License)
			assert.Equal(t, tt.want.Compatibility, got.Compatibility)
			assert.Equal(t, tt.want.AllowedTools, got.AllowedTools)
		})
	}
}

func TestPrompt_Clone(t *testing.T) {
	original := &Prompt{
		Name:          "test-prompt",
		Description:   "Test description",
		License:       "MIT",
		Compatibility: "gpt-4",
		AllowedTools:  "calculator",
		Metadata: map[string]any{
			"author": "test",
		},
		Execution: &ExecutionConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Skope: &SkopeConfig{
			Visibility: SkopeVisibilityPublic,
		},
		Inputs: map[string]*InputDef{
			"query": {Type: "string", Required: true},
		},
		Outputs: map[string]*OutputDef{
			"result": {Type: "string"},
		},
		Sample: map[string]any{
			"query": "test",
		},
	}

	clone := original.Clone()

	// Verify clone is equal
	assert.Equal(t, original.Name, clone.Name)
	assert.Equal(t, original.Description, clone.Description)
	assert.Equal(t, original.License, clone.License)

	// Verify deep copy - modify clone shouldn't affect original
	clone.Name = "modified"
	assert.NotEqual(t, original.Name, clone.Name)

	clone.Metadata["author"] = "modified"
	assert.NotEqual(t, original.Metadata["author"], clone.Metadata["author"])
}

func TestPrompt_Getters(t *testing.T) {
	prompt := &Prompt{
		Name:          "test-prompt",
		Description:   "Test description",
		License:       "MIT",
		Compatibility: "gpt-4",
		AllowedTools:  "calculator",
		Metadata: map[string]any{
			"author": "test",
		},
		Execution: &ExecutionConfig{
			Provider: "openai",
		},
		Skope: &SkopeConfig{
			Slug: "test-slug",
		},
		Sample: map[string]any{
			"query": "test",
		},
	}

	assert.Equal(t, "test-prompt", prompt.GetName())
	assert.Equal(t, "Test description", prompt.GetDescription())
	assert.Equal(t, "MIT", prompt.GetLicense())
	assert.Equal(t, "gpt-4", prompt.GetCompatibility())
	assert.Equal(t, "calculator", prompt.GetAllowedTools())
	assert.NotNil(t, prompt.GetMetadata())
	assert.NotNil(t, prompt.GetExecution())
	assert.NotNil(t, prompt.GetSkope())
	assert.NotNil(t, prompt.GetSampleData())
	assert.Equal(t, "test-slug", prompt.GetSlug())

	// Test nil prompt
	var nilPrompt *Prompt
	assert.Empty(t, nilPrompt.GetName())
	assert.Empty(t, nilPrompt.GetDescription())
	assert.Nil(t, nilPrompt.GetMetadata())
	assert.Nil(t, nilPrompt.GetExecution())
	assert.Nil(t, nilPrompt.GetSkope())
}

func TestPrompt_Has(t *testing.T) {
	prompt := &Prompt{
		Name:        "test",
		Description: "test",
		Execution:   &ExecutionConfig{},
		Skope:       &SkopeConfig{},
		Inputs:      map[string]*InputDef{"query": {}},
		Outputs:     map[string]*OutputDef{"result": {}},
		Sample:      map[string]any{"key": "value"},
		Metadata:    map[string]any{"key": "value"},
	}

	assert.True(t, prompt.HasExecution())
	assert.True(t, prompt.HasSkope())
	assert.True(t, prompt.HasInputs())
	assert.True(t, prompt.HasOutputs())
	assert.True(t, prompt.HasSample())
	assert.True(t, prompt.HasMetadata())

	emptyPrompt := &Prompt{}
	assert.False(t, emptyPrompt.HasExecution())
	assert.False(t, emptyPrompt.HasSkope())
	assert.False(t, emptyPrompt.HasInputs())
	assert.False(t, emptyPrompt.HasOutputs())
	assert.False(t, emptyPrompt.HasSample())
	assert.False(t, emptyPrompt.HasMetadata())
}

func TestPrompt_ValidateInputs(t *testing.T) {
	prompt := &Prompt{
		Name:        "test",
		Description: "test",
		Inputs: map[string]*InputDef{
			"query":    {Type: "string", Required: true},
			"limit":    {Type: "number", Required: false},
			"active":   {Type: "boolean", Required: false},
			"tags":     {Type: "array", Required: false},
			"metadata": {Type: "object", Required: false},
		},
	}

	tests := []struct {
		name    string
		data    map[string]any
		wantErr bool
	}{
		{
			name:    "missing required input",
			data:    map[string]any{},
			wantErr: true,
		},
		{
			name: "valid required input",
			data: map[string]any{
				"query": "test query",
			},
			wantErr: false,
		},
		{
			name: "wrong type for string",
			data: map[string]any{
				"query": 123,
			},
			wantErr: true,
		},
		{
			name: "valid number",
			data: map[string]any{
				"query": "test",
				"limit": 10,
			},
			wantErr: false,
		},
		{
			name: "valid boolean",
			data: map[string]any{
				"query":  "test",
				"active": true,
			},
			wantErr: false,
		},
		{
			name: "valid array",
			data: map[string]any{
				"query": "test",
				"tags":  []string{"tag1", "tag2"},
			},
			wantErr: false,
		},
		{
			name: "valid object",
			data: map[string]any{
				"query": "test",
				"metadata": map[string]any{
					"key": "value",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := prompt.ValidateInputs(tt.data)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPrompt_JSONAndYAML(t *testing.T) {
	prompt := &Prompt{
		Name:        "test-prompt",
		Description: "Test description",
		License:     "MIT",
	}

	// Test JSON
	jsonStr, err := prompt.JSON()
	require.NoError(t, err)
	assert.Contains(t, jsonStr, "test-prompt")
	assert.Contains(t, jsonStr, "Test description")

	// Test JSON Pretty
	jsonPretty, err := prompt.JSONPretty()
	require.NoError(t, err)
	assert.Contains(t, jsonPretty, "test-prompt")
	assert.Contains(t, jsonPretty, "\n")

	// Test YAML
	yamlStr, err := prompt.YAML()
	require.NoError(t, err)
	assert.Contains(t, yamlStr, "test-prompt")
	assert.Contains(t, yamlStr, "Test description")
}

func TestPrompt_IsAgentSkillsCompatible(t *testing.T) {
	tests := []struct {
		name   string
		prompt *Prompt
		want   bool
	}{
		{
			name:   "nil prompt",
			prompt: nil,
			want:   true,
		},
		{
			name: "no extensions",
			prompt: &Prompt{
				Name:        "test",
				Description: "test",
			},
			want: true,
		},
		{
			name: "has execution",
			prompt: &Prompt{
				Name:        "test",
				Description: "test",
				Execution:   &ExecutionConfig{},
			},
			want: false,
		},
		{
			name: "has skope",
			prompt: &Prompt{
				Name:        "test",
				Description: "test",
				Skope:       &SkopeConfig{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.prompt.IsAgentSkillsCompatible())
		})
	}
}

func TestPrompt_StripExtensions(t *testing.T) {
	original := &Prompt{
		Name:        "test",
		Description: "test",
		License:     "MIT",
		Execution:   &ExecutionConfig{Provider: "openai"},
		Skope:       &SkopeConfig{Visibility: "public"},
	}

	stripped := original.StripExtensions()

	assert.Equal(t, original.Name, stripped.Name)
	assert.Equal(t, original.Description, stripped.Description)
	assert.Equal(t, original.License, stripped.License)
	assert.Nil(t, stripped.Execution)
	assert.Nil(t, stripped.Skope)
}
