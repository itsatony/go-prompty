package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/itsatony/go-prompty"
)

// debugConfig holds parsed debug command configuration
type debugConfig struct {
	templatePath string
	dataJSON     string
	dataFilePath string
	format       string
	trace        bool
	verbose      bool
}

// debugOutput represents JSON output for debug
type debugOutput struct {
	Valid            bool              `json:"valid"`
	Variables        []debugVariable   `json:"variables"`
	Resolvers        []debugResolver   `json:"resolvers"`
	Includes         []debugInclude    `json:"includes"`
	MissingVariables []debugMissingVar `json:"missing_variables,omitempty"`
	UnusedData       []string          `json:"unused_data,omitempty"`
	Trace            []string          `json:"trace,omitempty"`
}

type debugVariable struct {
	Name    string `json:"name"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Exists  bool   `json:"exists"`
	Value   any    `json:"value,omitempty"`
	Default string `json:"default,omitempty"`
}

type debugResolver struct {
	Name  string `json:"name"`
	Lines []int  `json:"lines"`
}

type debugInclude struct {
	Name   string `json:"name"`
	Line   int    `json:"line"`
	Exists bool   `json:"exists"`
}

type debugMissingVar struct {
	Name        string   `json:"name"`
	Line        int      `json:"line"`
	Suggestions []string `json:"suggestions,omitempty"`
}

func runDebug(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	cfg, err := parseDebugFlags(args)
	if err != nil {
		fmt.Fprintf(stderr, FmtErrorWithCause, ErrMsgMissingTemplate, err)
		return ExitCodeUsageError
	}

	// Read template
	templateSource, err := readInput(cfg.templatePath, stdin)
	if err != nil {
		fmt.Fprintf(stderr, FmtErrorWithCause, ErrMsgReadFileFailed, err)
		return ExitCodeInputError
	}

	// Parse data if provided
	var data map[string]any
	if cfg.dataJSON != "" {
		if err := json.Unmarshal([]byte(cfg.dataJSON), &data); err != nil {
			fmt.Fprintf(stderr, FmtErrorWithCause, ErrMsgInvalidJSON, err)
			return ExitCodeInputError
		}
	} else if cfg.dataFilePath != "" {
		dataBytes, err := readInput(cfg.dataFilePath, nil)
		if err != nil {
			fmt.Fprintf(stderr, FmtErrorWithCause, ErrMsgReadFileFailed, err)
			return ExitCodeInputError
		}
		if err := json.Unmarshal(dataBytes, &data); err != nil {
			fmt.Fprintf(stderr, FmtErrorWithCause, ErrMsgInvalidJSON, err)
			return ExitCodeInputError
		}
	}

	// Create engine and run DryRun
	engine := prompty.MustNew()
	tmpl, parseErr := engine.Parse(string(templateSource))
	if parseErr != nil {
		fmt.Fprintf(stderr, FmtErrorWithCause, ErrMsgParseTemplateFailed, parseErr)
		return ExitCodeInputError
	}
	result := tmpl.DryRun(context.Background(), data)

	// Build debug output
	output := buildDebugOutput(result, data, engine, cfg)

	// Output based on format
	if cfg.format == OutputFormatJSON {
		return outputDebugJSON(output, stdout)
	}
	return outputDebugText(output, cfg.verbose, stdout)
}

func parseDebugFlags(args []string) (*debugConfig, error) {
	fs := flag.NewFlagSet(CmdNameDebug, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	cfg := &debugConfig{}

	fs.StringVar(&cfg.templatePath, FlagTemplate, "", "")
	fs.StringVar(&cfg.templatePath, FlagTemplateShort, "", "")
	fs.StringVar(&cfg.dataJSON, FlagData, "", "")
	fs.StringVar(&cfg.dataJSON, FlagDataShort, "", "")
	fs.StringVar(&cfg.dataFilePath, FlagDataFile, "", "")
	fs.StringVar(&cfg.dataFilePath, FlagDataFileShort, "", "")
	fs.StringVar(&cfg.format, FlagFormat, FlagDefaultFormat, "")
	fs.StringVar(&cfg.format, FlagFormatShort, FlagDefaultFormat, "")
	fs.BoolVar(&cfg.trace, FlagTrace, false, "")
	fs.BoolVar(&cfg.verbose, FlagVerbose, false, "")
	fs.BoolVar(&cfg.verbose, FlagVerboseShort, false, "")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if cfg.templatePath == "" {
		return nil, errors.New(ErrMsgMissingTemplate)
	}

	if cfg.format != OutputFormatText && cfg.format != OutputFormatJSON {
		return nil, errors.New(ErrMsgInvalidFormat)
	}

	return cfg, nil
}

func buildDebugOutput(result *prompty.DryRunResult, data map[string]any, engine *prompty.Engine, cfg *debugConfig) *debugOutput {
	output := &debugOutput{
		Valid:      result.Valid,
		Variables:  make([]debugVariable, 0, len(result.Variables)),
		Resolvers:  make([]debugResolver, 0),
		Includes:   make([]debugInclude, 0, len(result.Includes)),
		UnusedData: result.UnusedVariables,
	}

	// Process variables
	for _, v := range result.Variables {
		dv := debugVariable{
			Name:    v.Name,
			Line:    v.Line,
			Column:  v.Column,
			Exists:  v.InData,
			Default: v.Default,
		}
		// Get value from data if it exists
		if v.InData {
			dv.Value = getValueFromData(data, v.Name)
		}
		output.Variables = append(output.Variables, dv)

		// Track missing variables
		if !v.InData && v.Default == "" {
			mv := debugMissingVar{
				Name:        v.Name,
				Line:        v.Line,
				Suggestions: findSimilarKeys(v.Name, data),
			}
			output.MissingVariables = append(output.MissingVariables, mv)
		}
	}

	// Aggregate resolvers by name
	resolverLines := make(map[string][]int)
	for _, r := range result.Resolvers {
		resolverLines[r.TagName] = append(resolverLines[r.TagName], r.Line)
	}
	for name, lines := range resolverLines {
		output.Resolvers = append(output.Resolvers, debugResolver{
			Name:  name,
			Lines: lines,
		})
	}
	// Sort resolvers by name
	sort.Slice(output.Resolvers, func(i, j int) bool {
		return output.Resolvers[i].Name < output.Resolvers[j].Name
	})

	// Process includes
	for _, inc := range result.Includes {
		output.Includes = append(output.Includes, debugInclude{
			Name:   inc.TemplateName,
			Line:   inc.Line,
			Exists: engine.HasTemplate(inc.TemplateName),
		})
	}

	// Add trace if requested (DryRunResult doesn't have ExecutionTrace, use Warnings for similar info)
	if cfg.trace && len(result.Warnings) > 0 {
		output.Trace = result.Warnings
	}

	return output
}

// findSimilarKeys finds data keys similar to the given path
func findSimilarKeys(path string, data map[string]any) []string {
	if data == nil {
		return nil
	}

	var suggestions []string
	allKeys := flattenKeys(data, "")

	pathLower := strings.ToLower(path)
	for _, key := range allKeys {
		keyLower := strings.ToLower(key)
		// Simple similarity: contains or prefix match
		if strings.Contains(keyLower, pathLower) || strings.Contains(pathLower, keyLower) {
			suggestions = append(suggestions, key)
		} else if levenshteinDistance(pathLower, keyLower) <= 2 {
			suggestions = append(suggestions, key)
		}
	}

	// Limit suggestions
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}

	return suggestions
}

// flattenKeys extracts all keys from a nested map as dot-notation paths
func flattenKeys(data map[string]any, prefix string) []string {
	var keys []string
	for k, v := range data {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}
		keys = append(keys, fullKey)

		if nested, ok := v.(map[string]any); ok {
			keys = append(keys, flattenKeys(nested, fullKey)...)
		}
	}
	return keys
}

// getValueFromData retrieves a value by dot-notation path from data
func getValueFromData(data map[string]any, path string) any {
	if data == nil || path == "" {
		return nil
	}

	parts := strings.Split(path, ".")
	var current any = data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			val, ok := v[part]
			if !ok {
				return nil
			}
			current = val
		default:
			return nil
		}
	}

	return current
}

// levenshteinDistance calculates edit distance between two strings
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := 0; j <= len(b); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,
				min(matrix[i][j-1]+1, matrix[i-1][j-1]+cost),
			)
		}
	}

	return matrix[len(a)][len(b)]
}

func outputDebugText(output *debugOutput, verbose bool, stdout io.Writer) int {
	fmt.Fprintln(stdout, DebugTextHeader)
	fmt.Fprintln(stdout)

	// Variables
	fmt.Fprintf(stdout, DebugTextVariablesHeader+FmtNewline, len(output.Variables))
	for _, v := range output.Variables {
		if v.Exists {
			valueStr := fmt.Sprintf("%v", v.Value)
			if len(valueStr) > 30 {
				valueStr = valueStr[:27] + "..."
			}
			fmt.Fprintf(stdout, DebugTextVarExists+FmtNewline, v.Name, v.Line, valueStr)
		} else {
			line := fmt.Sprintf(DebugTextVarMissing, v.Name, v.Line)
			if v.Default != "" {
				line += fmt.Sprintf(DebugTextVarDefault, v.Default)
			}
			fmt.Fprintln(stdout, line)
		}
	}
	fmt.Fprintln(stdout)

	// Resolvers
	fmt.Fprintf(stdout, DebugTextResolversHeader+FmtNewline, len(output.Resolvers))
	for _, r := range output.Resolvers {
		linesStr := formatIntSlice(r.Lines)
		fmt.Fprintf(stdout, DebugTextResolverFormat+FmtNewline, r.Name, linesStr)
	}
	fmt.Fprintln(stdout)

	// Includes
	if len(output.Includes) > 0 {
		fmt.Fprintf(stdout, DebugTextIncludesHeader+FmtNewline, len(output.Includes))
		for _, inc := range output.Includes {
			if inc.Exists {
				fmt.Fprintf(stdout, DebugTextIncludeExists+FmtNewline, inc.Name, inc.Line)
			} else {
				fmt.Fprintf(stdout, DebugTextIncludeMissing+FmtNewline, inc.Name, inc.Line)
			}
		}
		fmt.Fprintln(stdout)
	}

	// Unused data
	if len(output.UnusedData) > 0 {
		fmt.Fprintln(stdout, DebugTextUnusedHeader)
		for _, key := range output.UnusedData {
			fmt.Fprintf(stdout, DebugTextUnusedFormat+FmtNewline, key)
		}
		fmt.Fprintln(stdout)
	}

	// Missing variable suggestions (verbose mode)
	if verbose && len(output.MissingVariables) > 0 {
		fmt.Fprintln(stdout, "Missing Variable Suggestions:")
		for _, mv := range output.MissingVariables {
			if len(mv.Suggestions) > 0 {
				fmt.Fprintf(stdout, "  %s: did you mean %s?\n", mv.Name, strings.Join(mv.Suggestions, ", "))
			}
		}
		fmt.Fprintln(stdout)
	}

	// Trace (if available)
	if len(output.Trace) > 0 {
		fmt.Fprintln(stdout, "Execution Trace:")
		for _, t := range output.Trace {
			fmt.Fprintf(stdout, "  %s\n", t)
		}
		fmt.Fprintln(stdout)
	}

	// Summary
	fmt.Fprintf(stdout, DebugTextSummary+FmtNewline, len(output.MissingVariables), len(output.UnusedData))

	if len(output.MissingVariables) > 0 {
		return ExitCodeValidationError
	}
	return ExitCodeSuccess
}

func outputDebugJSON(output *debugOutput, stdout io.Writer) int {
	jsonBytes, _ := json.MarshalIndent(output, "", "  ")
	fmt.Fprintln(stdout, string(jsonBytes))

	if len(output.MissingVariables) > 0 {
		return ExitCodeValidationError
	}
	return ExitCodeSuccess
}

func formatIntSlice(nums []int) string {
	if len(nums) == 0 {
		return ""
	}
	strs := make([]string, len(nums))
	for i, n := range nums {
		strs[i] = fmt.Sprintf("%d", n)
	}
	return strings.Join(strs, ", ")
}
