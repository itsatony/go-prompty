package prompty

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Catalog generation error messages
const (
	ErrMsgCatalogResolverFailed = "failed to resolve skill for catalog"
	ErrMsgCatalogInvalidFormat  = "invalid catalog format"
)

// CatalogCompactDescriptionMaxLen is the max description length in compact format.
const CatalogCompactDescriptionMaxLen = 80

// skillCatalogEntry holds resolved info about a skill for catalog generation.
type skillCatalogEntry struct {
	slug        string
	description string
	injection   SkillInjection
	version     string
}

// GenerateSkillsCatalog generates a human-readable catalog of skills in the specified format.
// It resolves each skill reference using the DocumentResolver to get descriptions.
//
// Supported formats:
//   - CatalogFormatDefault (""): Markdown list with bold slugs and descriptions
//   - CatalogFormatDetailed: Markdown with headers, version info, and injection mode
//   - CatalogFormatCompact: Single-line semicolon-separated list
//   - CatalogFormatFunctionCalling: Not supported for skills (returns error)
//
// Resolution failures for individual skills are non-fatal — the skill appears with
// an empty description. Returns empty string for empty skill lists.
//
// Example:
//
//	catalog, err := prompty.GenerateSkillsCatalog(ctx, agent.Skills, resolver, prompty.CatalogFormatDetailed)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(catalog)
func GenerateSkillsCatalog(ctx context.Context, skills []SkillRef, resolver DocumentResolver, format CatalogFormat) (string, error) {
	if len(skills) == 0 {
		return "", nil
	}

	// Collect skill info
	entries := make([]skillCatalogEntry, 0, len(skills))
	for i := range skills {
		ref := &skills[i]
		entry := skillCatalogEntry{
			slug:      ref.GetSlug(),
			injection: ref.Injection,
			version:   ref.GetVersion(),
		}

		// Get description from inline or resolver
		if ref.IsInline() {
			entry.description = ref.Inline.Description
		} else if resolver != nil {
			resolved, err := resolver.ResolveSkill(ctx, ref.Slug)
			if err == nil && resolved != nil {
				entry.description = resolved.Description
			}
			// If resolution fails, use empty description (non-fatal)
		}

		entries = append(entries, entry)
	}

	switch format {
	case CatalogFormatDetailed:
		return generateSkillsCatalogDetailed(entries), nil
	case CatalogFormatCompact:
		return generateSkillsCatalogCompact(entries), nil
	case CatalogFormatFunctionCalling:
		return "", NewCatalogError(ErrMsgCatalogInvalidFormat, fmt.Errorf("%s", ErrMsgCatalogFuncCallingSkills))
	case CatalogFormatDefault:
		return generateSkillsCatalogDefault(entries), nil
	default:
		return "", NewCatalogError(ErrMsgCatalogInvalidFormat, fmt.Errorf("%s: %s", ErrMsgCatalogUnknownFormat, string(format)))
	}
}

// generateSkillsCatalogDefault generates a default markdown skills catalog.
func generateSkillsCatalogDefault(entries []skillCatalogEntry) string {
	var b strings.Builder
	b.WriteString("## Available Skills\n\n")
	for _, entry := range entries {
		b.WriteString("- **")
		b.WriteString(entry.slug)
		b.WriteString("**")
		if entry.description != "" {
			b.WriteString(": ")
			b.WriteString(entry.description)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// generateSkillsCatalogDetailed generates a detailed skills catalog.
func generateSkillsCatalogDetailed(entries []skillCatalogEntry) string {
	var b strings.Builder
	b.WriteString("## Available Skills\n\n")
	for _, entry := range entries {
		b.WriteString("### ")
		b.WriteString(entry.slug)
		if entry.version != "" && entry.version != RefVersionLatest {
			b.WriteString(" (v")
			b.WriteString(entry.version)
			b.WriteString(")")
		}
		b.WriteString("\n")
		if entry.description != "" {
			b.WriteString(entry.description)
			b.WriteString("\n")
		}
		if entry.injection != "" {
			b.WriteString("- Injection: ")
			b.WriteString(string(entry.injection))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

// generateSkillsCatalogCompact generates a compact single-line skills catalog.
func generateSkillsCatalogCompact(entries []skillCatalogEntry) string {
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		part := entry.slug
		if entry.description != "" {
			part += " - " + truncateString(entry.description, CatalogCompactDescriptionMaxLen)
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "; ")
}

// GenerateToolsCatalog generates a human-readable catalog of tools in the specified format.
//
// Supported formats:
//   - CatalogFormatDefault (""): Markdown list with tool names and descriptions
//   - CatalogFormatDetailed: Markdown with headers, parameter schemas, and MCP server details
//   - CatalogFormatCompact: Single-line semicolon-separated list
//   - CatalogFormatFunctionCalling: JSON array of OpenAI-compatible function definitions
//
// Returns empty string when tools is nil or has no tools defined.
//
// Example:
//
//	catalog, err := prompty.GenerateToolsCatalog(agent.Tools, prompty.CatalogFormatFunctionCalling)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// catalog is JSON: [{"type":"function","function":{...}}]
func GenerateToolsCatalog(tools *ToolsConfig, format CatalogFormat) (string, error) {
	if tools == nil || !tools.HasTools() {
		return "", nil
	}

	switch format {
	case CatalogFormatFunctionCalling:
		return generateToolsCatalogFunctionCalling(tools)
	case CatalogFormatDetailed:
		return generateToolsCatalogDetailed(tools), nil
	case CatalogFormatCompact:
		return generateToolsCatalogCompact(tools), nil
	case CatalogFormatDefault:
		return generateToolsCatalogDefault(tools), nil
	default:
		return "", NewCatalogError(ErrMsgCatalogInvalidFormat, fmt.Errorf("%s: %s", ErrMsgCatalogUnknownFormat, string(format)))
	}
}

// generateToolsCatalogDefault generates a default markdown tools catalog.
func generateToolsCatalogDefault(tools *ToolsConfig) string {
	var b strings.Builder
	b.WriteString("## Available Tools\n\n")

	for _, fn := range tools.Functions {
		b.WriteString("- **")
		b.WriteString(fn.Name)
		b.WriteString("**")
		if fn.Description != "" {
			b.WriteString(": ")
			b.WriteString(fn.Description)
		}
		b.WriteString("\n")
	}

	for _, srv := range tools.MCPServers {
		b.WriteString("- **[MCP] ")
		b.WriteString(srv.Name)
		b.WriteString("**")
		if len(srv.Tools) > 0 {
			b.WriteString(" (tools: ")
			b.WriteString(strings.Join(srv.Tools, ", "))
			b.WriteString(")")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// generateToolsCatalogDetailed generates a detailed tools catalog.
func generateToolsCatalogDetailed(tools *ToolsConfig) string {
	var b strings.Builder
	b.WriteString("## Available Tools\n\n")

	for _, fn := range tools.Functions {
		b.WriteString("### ")
		b.WriteString(fn.Name)
		b.WriteString("\n")
		if fn.Description != "" {
			b.WriteString(fn.Description)
			b.WriteString("\n")
		}
		if fn.Parameters != nil {
			b.WriteString("**Parameters:**\n")
			b.WriteString("```json\n")
			paramJSON, _ := json.MarshalIndent(fn.Parameters, "", "  ")
			b.Write(paramJSON)
			b.WriteString("\n```\n")
		}
		b.WriteString("\n")
	}

	for _, srv := range tools.MCPServers {
		b.WriteString("### [MCP] ")
		b.WriteString(srv.Name)
		b.WriteString("\n")
		b.WriteString("- URL: ")
		b.WriteString(srv.URL)
		b.WriteString("\n")
		if srv.Transport != "" {
			b.WriteString("- Transport: ")
			b.WriteString(srv.Transport)
			b.WriteString("\n")
		}
		if len(srv.Tools) > 0 {
			b.WriteString("- Tools: ")
			b.WriteString(strings.Join(srv.Tools, ", "))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// generateToolsCatalogCompact generates a compact tools catalog.
func generateToolsCatalogCompact(tools *ToolsConfig) string {
	parts := make([]string, 0, len(tools.Functions)+len(tools.MCPServers))

	for _, fn := range tools.Functions {
		part := fn.Name
		if fn.Description != "" {
			part += " - " + truncateString(fn.Description, CatalogCompactDescriptionMaxLen)
		}
		parts = append(parts, part)
	}

	for _, srv := range tools.MCPServers {
		part := "[MCP] " + srv.Name
		parts = append(parts, part)
	}

	return strings.Join(parts, "; ")
}

// generateToolsCatalogFunctionCalling generates a JSON function-calling schema.
func generateToolsCatalogFunctionCalling(tools *ToolsConfig) (string, error) {
	toolDefs := make([]map[string]any, 0, len(tools.Functions))

	for _, fn := range tools.Functions {
		toolDefs = append(toolDefs, fn.ToOpenAITool())
	}

	data, err := json.MarshalIndent(toolDefs, "", "  ")
	if err != nil {
		return "", NewCatalogError(ErrMsgCatalogGenerationFailed, err)
	}
	return string(data), nil
}

// truncateString truncates a string to the given max length, appending "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
