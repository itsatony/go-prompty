package prompty

import (
	"archive/zip"
	"bytes"
)

// Export error messages
const (
	ErrMsgExportFailed    = "export failed"
	ErrMsgExportZipFailed = "zip export failed"
)

// ExportSkillDirectory creates a zip archive containing the prompt document
// and optional resources. The main document is named based on its type
// (SKILL.md, AGENT.md, or PROMPT.md).
func ExportSkillDirectory(prompt *Prompt, resources map[string][]byte) ([]byte, error) {
	if prompt == nil {
		return nil, NewCompilationError(ErrMsgExportFailed, nil)
	}

	// Serialize the prompt
	docBytes, err := prompt.ExportFull()
	if err != nil {
		return nil, err
	}

	// Determine filename based on type
	docFilename := documentFilename(prompt.EffectiveType())

	// Create zip
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Write the main document
	f, err := w.Create(docFilename)
	if err != nil {
		return nil, NewCompilationError(ErrMsgExportZipFailed, err)
	}
	if _, err := f.Write(docBytes); err != nil {
		return nil, NewCompilationError(ErrMsgExportZipFailed, err)
	}

	// Write resource files
	for name, content := range resources {
		rf, err := w.Create(name)
		if err != nil {
			return nil, NewCompilationError(ErrMsgExportZipFailed, err)
		}
		if _, err := rf.Write(content); err != nil {
			return nil, NewCompilationError(ErrMsgExportZipFailed, err)
		}
	}

	if err := w.Close(); err != nil {
		return nil, NewCompilationError(ErrMsgExportZipFailed, err)
	}

	return buf.Bytes(), nil
}

// documentFilename returns the appropriate filename for a document type.
func documentFilename(dt DocumentType) string {
	switch dt {
	case DocumentTypeAgent:
		return "AGENT.md"
	case DocumentTypePrompt:
		return "PROMPT.md"
	default:
		return "SKILL.md"
	}
}
