package prompty

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportSkillDirectory_Skill(t *testing.T) {
	p := &Prompt{
		Name:        "export-skill",
		Description: "A skill to export",
		Type:        DocumentTypeSkill,
		Body:        "Skill body content.",
	}

	resources := map[string][]byte{
		"examples/test.json": []byte(`{"example": true}`),
	}

	data, err := ExportSkillDirectory(p, resources)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Read back the zip
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)

	fileNames := make(map[string]bool)
	for _, f := range reader.File {
		fileNames[f.Name] = true
	}

	assert.True(t, fileNames["SKILL.md"], "expected SKILL.md in archive")
	assert.True(t, fileNames["examples/test.json"], "expected resource in archive")

	// Read SKILL.md and verify content
	for _, f := range reader.File {
		if f.Name == "SKILL.md" {
			rc, err := f.Open()
			require.NoError(t, err)
			content, err := io.ReadAll(rc)
			rc.Close()
			require.NoError(t, err)
			assert.Contains(t, string(content), "export-skill")
			assert.Contains(t, string(content), "Skill body content.")
		}
	}
}

func TestExportSkillDirectory_Agent(t *testing.T) {
	p := &Prompt{
		Name:        "export-agent",
		Description: "An agent to export",
		Type:        DocumentTypeAgent,
		Body:        "Agent body.",
	}

	data, err := ExportSkillDirectory(p, nil)
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)

	found := false
	for _, f := range reader.File {
		if f.Name == "AGENT.md" {
			found = true
		}
	}
	assert.True(t, found, "expected AGENT.md in archive")
}

func TestExportSkillDirectory_Prompt(t *testing.T) {
	p := &Prompt{
		Name:        "export-prompt",
		Description: "A prompt to export",
		Type:        DocumentTypePrompt,
		Body:        "body",
	}

	data, err := ExportSkillDirectory(p, nil)
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)

	found := false
	for _, f := range reader.File {
		if f.Name == "PROMPT.md" {
			found = true
		}
	}
	assert.True(t, found, "expected PROMPT.md in archive")
}

func TestExportSkillDirectory_Nil(t *testing.T) {
	_, err := ExportSkillDirectory(nil, nil)
	require.Error(t, err)
}

func TestExportImportRoundTrip(t *testing.T) {
	original := &Prompt{
		Name:        "roundtrip-skill",
		Description: "Round-trip testing",
		Type:        DocumentTypeSkill,
		Body:        "Original body content.",
	}

	resources := map[string][]byte{
		"config.json": []byte(`{"setting": 42}`),
	}

	// Export
	zipData, err := ExportSkillDirectory(original, resources)
	require.NoError(t, err)

	// Import
	result, err := ImportDirectory(zipData)
	require.NoError(t, err)
	require.NotNil(t, result.Prompt)

	assert.Equal(t, original.Name, result.Prompt.Name)
	assert.Equal(t, original.Description, result.Prompt.Description)
	assert.Equal(t, original.Body, result.Prompt.Body)

	// Check resources
	assert.Contains(t, result.Resources, "config.json")
	assert.Equal(t, `{"setting": 42}`, string(result.Resources["config.json"]))
}
