package prompty

import (
	"archive/zip"
	"bytes"
	"io"
	"path/filepath"
	"strings"
)

// Import error messages
const (
	ErrMsgImportFailed      = "import failed"
	ErrMsgImportUnknownType = "unknown import file type"
	ErrMsgImportZipFailed   = "zip import failed"
	ErrMsgImportNoDocument  = "no document found in archive"
	ErrMsgImportReadFailed  = "failed to read import file"
)

// ImportResult holds the result of importing a document.
type ImportResult struct {
	// Prompt is the parsed prompt/skill/agent configuration
	Prompt *Prompt
	// Resources maps resource filenames to their content (e.g., from zip)
	Resources map[string][]byte
}

// Import parses a document from raw data and filename.
// Supported formats: .md (SKILL.md/AGENT.md), .zip (directory archive).
func Import(data []byte, filename string) (*ImportResult, error) {
	if len(data) == 0 {
		return nil, NewFrontmatterError(ErrMsgImportFailed, Position{Line: 1, Column: 1}, nil)
	}

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".md":
		return importMarkdown(data)
	case ".zip":
		return ImportDirectory(data)
	default:
		// Try as markdown
		return importMarkdown(data)
	}
}

// importMarkdown imports from a markdown document with YAML frontmatter.
func importMarkdown(data []byte) (*ImportResult, error) {
	prompt, err := Parse(data)
	if err != nil {
		return nil, err
	}

	return &ImportResult{
		Prompt:    prompt,
		Resources: make(map[string][]byte),
	}, nil
}

// ImportDirectory imports from a zip archive containing a skill/agent directory.
// The archive should contain a SKILL.md or AGENT.md file at the root.
func ImportDirectory(data []byte) (*ImportResult, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, NewFrontmatterError(ErrMsgImportZipFailed, Position{Line: 1, Column: 1}, err)
	}

	var docFile *zip.File
	resources := make(map[string][]byte)

	for _, f := range reader.File {
		name := filepath.Base(f.Name)
		upperName := strings.ToUpper(name)

		// Look for the main document file
		if upperName == "SKILL.MD" || upperName == "AGENT.MD" || upperName == "PROMPT.MD" {
			docFile = f
			continue
		}

		// Skip directories
		if f.FileInfo().IsDir() {
			continue
		}

		// Read resource file
		rc, err := f.Open()
		if err != nil {
			continue
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		resources[f.Name] = content
	}

	if docFile == nil {
		return nil, NewFrontmatterError(ErrMsgImportNoDocument, Position{Line: 1, Column: 1}, nil)
	}

	// Read the document file
	rc, err := docFile.Open()
	if err != nil {
		return nil, NewFrontmatterError(ErrMsgImportReadFailed, Position{Line: 1, Column: 1}, err)
	}
	defer rc.Close()

	docContent, err := io.ReadAll(rc)
	if err != nil {
		return nil, NewFrontmatterError(ErrMsgImportReadFailed, Position{Line: 1, Column: 1}, err)
	}

	// Parse the document
	prompt, err := Parse(docContent)
	if err != nil {
		return nil, err
	}

	return &ImportResult{
		Prompt:    prompt,
		Resources: resources,
	}, nil
}
