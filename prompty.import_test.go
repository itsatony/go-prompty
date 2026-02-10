package prompty

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImport_Markdown(t *testing.T) {
	doc := `---
name: import-test
description: Testing import
type: skill
---
Hello world!`

	result, err := Import([]byte(doc), "SKILL.md")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Prompt)
	assert.Equal(t, "import-test", result.Prompt.Name)
	assert.Equal(t, DocumentTypeSkill, result.Prompt.Type)
	assert.Equal(t, "Hello world!", result.Prompt.Body)
	assert.Empty(t, result.Resources)
}

func TestImport_MarkdownAgent(t *testing.T) {
	doc := `---
name: agent-import
description: Testing agent import
type: agent
---
Agent body.`

	result, err := Import([]byte(doc), "AGENT.md")
	require.NoError(t, err)
	require.NotNil(t, result.Prompt)
	assert.Equal(t, DocumentTypeAgent, result.Prompt.Type)
}

func TestImport_Empty(t *testing.T) {
	_, err := Import([]byte{}, "test.md")
	require.Error(t, err)
}

func TestImport_Zip(t *testing.T) {
	// Create a zip archive in memory
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Add SKILL.md
	doc := `---
name: zipped-skill
description: A zipped skill
type: skill
---
Zipped body content.`
	f, err := w.Create("SKILL.md")
	require.NoError(t, err)
	_, err = f.Write([]byte(doc))
	require.NoError(t, err)

	// Add a resource file
	rf, err := w.Create("resources/data.json")
	require.NoError(t, err)
	_, err = rf.Write([]byte(`{"key": "value"}`))
	require.NoError(t, err)

	require.NoError(t, w.Close())

	// Import
	result, err := Import(buf.Bytes(), "package.zip")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Prompt)
	assert.Equal(t, "zipped-skill", result.Prompt.Name)
	assert.Equal(t, "Zipped body content.", result.Prompt.Body)

	// Check resources
	assert.Contains(t, result.Resources, "resources/data.json")
	assert.Equal(t, `{"key": "value"}`, string(result.Resources["resources/data.json"]))
}

func TestImport_ZipNoDocument(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("readme.txt")
	_, err := f.Write([]byte("not a skill"))
	require.NoError(t, err)
	w.Close()

	_, err = Import(buf.Bytes(), "package.zip")
	require.Error(t, err)
}

func TestImport_UnknownExtension(t *testing.T) {
	// Should default to markdown parsing
	doc := `---
name: unknown-ext
description: Unknown extension
---
body`

	result, err := Import([]byte(doc), "file.txt")
	require.NoError(t, err)
	assert.Equal(t, "unknown-ext", result.Prompt.Name)
}

func TestImportDirectory_WithAgentMD(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	doc := `---
name: zip-agent
description: A zipped agent
type: agent
---
Agent body here.`

	f, _ := w.Create("AGENT.md")
	_, err := f.Write([]byte(doc))
	require.NoError(t, err)
	w.Close()

	result, err := ImportDirectory(buf.Bytes())
	require.NoError(t, err)
	require.NotNil(t, result.Prompt)
	assert.Equal(t, DocumentTypeAgent, result.Prompt.Type)
	assert.Equal(t, "zip-agent", result.Prompt.Name)
}
