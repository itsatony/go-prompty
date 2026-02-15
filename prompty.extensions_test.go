package prompty

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPrompt_GetExtension(t *testing.T) {
	p := &Prompt{
		Name:        "test",
		Description: "test",
		Extensions: map[string]any{
			"custom": "value",
			"nested": map[string]any{"key": "val"},
		},
	}

	val, ok := p.GetExtension("custom")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	val, ok = p.GetExtension("nested")
	assert.True(t, ok)
	assert.NotNil(t, val)

	_, ok = p.GetExtension("missing")
	assert.False(t, ok)
}

func TestPrompt_GetExtension_NilSafe(t *testing.T) {
	var nilPrompt *Prompt
	val, ok := nilPrompt.GetExtension("key")
	assert.False(t, ok)
	assert.Nil(t, val)

	emptyPrompt := &Prompt{}
	val, ok = emptyPrompt.GetExtension("key")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestPrompt_HasExtension(t *testing.T) {
	p := &Prompt{
		Extensions: map[string]any{"exists": true},
	}
	assert.True(t, p.HasExtension("exists"))
	assert.False(t, p.HasExtension("missing"))

	var nilPrompt *Prompt
	assert.False(t, nilPrompt.HasExtension("key"))
}

func TestPrompt_SetExtension(t *testing.T) {
	p := &Prompt{Name: "test", Description: "test"}
	assert.Nil(t, p.Extensions)

	p.SetExtension("vendor", "acme")
	assert.Equal(t, "acme", p.Extensions["vendor"])

	p.SetExtension("other", 42)
	assert.Equal(t, 42, p.Extensions["other"])

	// Overwrite
	p.SetExtension("vendor", "newco")
	assert.Equal(t, "newco", p.Extensions["vendor"])
}

func TestPrompt_SetExtension_NilSafe(t *testing.T) {
	var nilPrompt *Prompt
	nilPrompt.SetExtension("key", "val") // should not panic
}

func TestPrompt_RemoveExtension(t *testing.T) {
	p := &Prompt{
		Extensions: map[string]any{"a": 1, "b": 2},
	}

	p.RemoveExtension("a")
	assert.False(t, p.HasExtension("a"))
	assert.True(t, p.HasExtension("b"))

	// Removing non-existent key is a no-op
	p.RemoveExtension("missing")
}

func TestPrompt_RemoveExtension_NilSafe(t *testing.T) {
	var nilPrompt *Prompt
	nilPrompt.RemoveExtension("key") // should not panic

	emptyPrompt := &Prompt{}
	emptyPrompt.RemoveExtension("key") // should not panic
}

func TestPrompt_GetExtensions(t *testing.T) {
	ext := map[string]any{"a": 1}
	p := &Prompt{Extensions: ext}
	assert.Equal(t, ext, p.GetExtensions())

	var nilPrompt *Prompt
	assert.Nil(t, nilPrompt.GetExtensions())
}

func TestGetExtensionAs(t *testing.T) {
	type CustomConfig struct {
		Visibility string   `json:"visibility"`
		Projects   []string `json:"projects"`
	}

	p := &Prompt{
		Extensions: map[string]any{
			"platform": map[string]any{
				"visibility": "public",
				"projects":   []any{"proj1", "proj2"},
			},
			"simple": "just-a-string",
		},
	}

	t.Run("struct conversion", func(t *testing.T) {
		cfg, err := GetExtensionAs[CustomConfig](p, "platform")
		require.NoError(t, err)
		assert.Equal(t, "public", cfg.Visibility)
		assert.Equal(t, []string{"proj1", "proj2"}, cfg.Projects)
	})

	t.Run("string conversion", func(t *testing.T) {
		val, err := GetExtensionAs[string](p, "simple")
		require.NoError(t, err)
		assert.Equal(t, "just-a-string", val)
	})

	t.Run("missing key", func(t *testing.T) {
		_, err := GetExtensionAs[CustomConfig](p, "nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgExtensionNotFound)
	})

	t.Run("nil prompt", func(t *testing.T) {
		_, err := GetExtensionAs[CustomConfig](nil, "key")
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgExtensionNotFound)
	})
}

func TestPrompt_GetStandardFields(t *testing.T) {
	p := &Prompt{
		Name:          "test",
		Description:   "desc",
		License:       "MIT",
		Compatibility: "gpt-4",
		AllowedTools:  "calc",
		Metadata:      map[string]any{"k": "v"},
		Inputs:        map[string]*InputDef{"q": {Type: "string"}},
		Outputs:       map[string]*OutputDef{"r": {Type: "string"}},
		Sample:        map[string]any{"q": "test"},
		// Non-standard fields (should NOT appear)
		Type:      DocumentTypeAgent,
		Execution: &ExecutionConfig{Provider: ProviderOpenAI},
	}

	fields := p.GetStandardFields()
	assert.Contains(t, fields, "name")
	assert.Contains(t, fields, "description")
	assert.Contains(t, fields, "license")
	assert.Contains(t, fields, "compatibility")
	assert.Contains(t, fields, "allowed_tools")
	assert.Contains(t, fields, "metadata")
	assert.Contains(t, fields, "inputs")
	assert.Contains(t, fields, "outputs")
	assert.Contains(t, fields, "sample")

	// Should NOT contain prompty-specific fields
	assert.NotContains(t, fields, "type")
	assert.NotContains(t, fields, "execution")
}

func TestPrompt_GetStandardFields_NilSafe(t *testing.T) {
	var nilPrompt *Prompt
	assert.Nil(t, nilPrompt.GetStandardFields())
}

func TestPrompt_GetPromptyFields(t *testing.T) {
	p := &Prompt{
		Name:        "test",
		Description: "desc",
		Type:        DocumentTypeAgent,
		Execution:   &ExecutionConfig{Provider: ProviderOpenAI},
		Skills:      []SkillRef{{Slug: "skill-a"}},
		Tools:       &ToolsConfig{},
		Context:     map[string]any{"company": "Acme"},
		Constraints: &ConstraintsConfig{},
		Messages:    []MessageTemplate{{Role: RoleSystem, Content: "Hi"}},
	}

	fields := p.GetPromptyFields()
	assert.Contains(t, fields, "type")
	assert.Contains(t, fields, "execution")
	assert.Contains(t, fields, "skills")
	assert.Contains(t, fields, "tools")
	assert.Contains(t, fields, "context")
	assert.Contains(t, fields, "constraints")
	assert.Contains(t, fields, "messages")

	// Should NOT contain standard fields
	assert.NotContains(t, fields, "name")
	assert.NotContains(t, fields, "description")
}

func TestPrompt_GetPromptyFields_NilSafe(t *testing.T) {
	var nilPrompt *Prompt
	assert.Nil(t, nilPrompt.GetPromptyFields())
}

func TestPrompt_Extensions_YAMLRoundTrip(t *testing.T) {
	// Parse YAML with unknown fields
	yamlData := `name: test-prompt
description: A test prompt
custom_vendor:
  setting: enabled
  level: 5
another_ext: hello`

	var p Prompt
	err := yaml.Unmarshal([]byte(yamlData), &p)
	require.NoError(t, err)

	assert.Equal(t, "test-prompt", p.Name)
	assert.True(t, p.HasExtension("custom_vendor"))
	assert.True(t, p.HasExtension("another_ext"))

	// Marshal back to YAML
	out, err := yaml.Marshal(&p)
	require.NoError(t, err)

	content := string(out)
	assert.Contains(t, content, "custom_vendor")
	assert.Contains(t, content, "another_ext")
	assert.Contains(t, content, "test-prompt")
}

func TestPrompt_Extensions_JSONSerialization(t *testing.T) {
	p := &Prompt{
		Name:        "test",
		Description: "test",
		Extensions: map[string]any{
			"vendor": map[string]any{"key": "val"},
		},
	}

	data, err := json.Marshal(p)
	require.NoError(t, err)

	// JSON should put extensions under "extensions" key
	var raw map[string]any
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "extensions")
	ext, ok := raw["extensions"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, ext, "vendor")
}

func TestPrompt_Extensions_CloneDeepCopy(t *testing.T) {
	original := &Prompt{
		Name:        "test",
		Description: "test",
		Extensions: map[string]any{
			"vendor": map[string]any{
				"key": "original-value",
			},
		},
	}

	clone := original.Clone()
	require.NotNil(t, clone.Extensions)
	assert.True(t, clone.HasExtension("vendor"))

	// Modify clone's nested extension map — should NOT affect original
	cloneVendor, _ := clone.GetExtension("vendor")
	cloneMap, ok := cloneVendor.(map[string]any)
	require.True(t, ok)
	cloneMap["key"] = "modified"

	origVendor, _ := original.GetExtension("vendor")
	origMap, ok := origVendor.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "original-value", origMap["key"])
}

func TestPrompt_Extensions_DontInterfereWithKnownFields(t *testing.T) {
	yamlData := `name: my-prompt
description: My description
execution:
  provider: openai
  model: gpt-4
custom_field: custom_value`

	var p Prompt
	err := yaml.Unmarshal([]byte(yamlData), &p)
	require.NoError(t, err)

	// Known fields parsed normally
	assert.Equal(t, "my-prompt", p.Name)
	assert.Equal(t, "My description", p.Description)
	require.NotNil(t, p.Execution)
	assert.Equal(t, ProviderOpenAI, p.Execution.Provider)

	// Unknown field captured in Extensions
	assert.True(t, p.HasExtension("custom_field"))
	val, ok := p.GetExtension("custom_field")
	require.True(t, ok)
	assert.Equal(t, "custom_value", val)

	// Known fields should NOT appear in Extensions
	assert.False(t, p.HasExtension("name"))
	assert.False(t, p.HasExtension("description"))
	assert.False(t, p.HasExtension("execution"))
}

func TestPrompt_Extensions_EmptyMapOmitted(t *testing.T) {
	p := &Prompt{
		Name:        "test",
		Description: "test",
	}

	data, err := json.Marshal(p)
	require.NoError(t, err)

	// Extensions should be omitted from JSON when nil/empty
	assert.NotContains(t, string(data), "extensions")
}
