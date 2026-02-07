package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/itsatony/go-prompty/v2"
)

// validateConfig holds parsed validate command configuration
type validateConfig struct {
	templatePath string
	format       string
	strict       bool
}

// validationOutput represents JSON output for validation
type validationOutput struct {
	Valid  bool                    `json:"valid"`
	Issues []validationIssueOutput `json:"issues,omitempty"`
}

type validationIssueOutput struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Tag      string `json:"tag,omitempty"`
}

func runValidate(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	cfg, err := parseValidateFlags(args)
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

	// Create engine and validate
	engine := prompty.MustNew()
	result, err := engine.Validate(string(templateSource))
	if err != nil {
		fmt.Fprintf(stderr, FmtErrorWithCause, ErrMsgParseTemplateFailed, err)
		return ExitCodeError
	}

	// Output based on format
	if cfg.format == OutputFormatJSON {
		return outputValidationJSON(result, cfg.strict, stdout)
	}
	return outputValidationText(result, cfg.strict, stdout)
}

func parseValidateFlags(args []string) (*validateConfig, error) {
	fs := flag.NewFlagSet(CmdNameValidate, flag.ContinueOnError)
	fs.SetOutput(io.Discard) // Suppress default error messages

	cfg := &validateConfig{}

	fs.StringVar(&cfg.templatePath, FlagTemplate, "", "")
	fs.StringVar(&cfg.templatePath, FlagTemplateShort, "", "")
	fs.StringVar(&cfg.format, FlagFormat, FlagDefaultFormat, "")
	fs.StringVar(&cfg.format, FlagFormatShort, FlagDefaultFormat, "")
	fs.BoolVar(&cfg.strict, FlagStrictMode, false, "")

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

func outputValidationText(result *prompty.ValidationResult, strict bool, stdout io.Writer) int {
	issues := result.Issues()
	errors := result.Errors()
	warnings := result.Warnings()

	if len(issues) == 0 {
		fmt.Fprintln(stdout, ValidationTextSuccess)
		return ExitCodeSuccess
	}

	fmt.Fprintln(stdout, ValidationTextIssueHeader)
	for _, issue := range issues {
		severityName := severityToName(issue.Severity)
		fmt.Fprintf(stdout, ValidationTextIssueFormat+FmtNewline,
			severityName, issue.Message, issue.Position.Line, issue.Position.Column)
	}

	fmt.Fprintf(stdout, ValidationTextErrorSummary+FmtNewline, len(errors), len(warnings))

	if len(errors) > 0 || (strict && len(warnings) > 0) {
		return ExitCodeValidationError
	}
	return ExitCodeSuccess
}

func outputValidationJSON(result *prompty.ValidationResult, strict bool, stdout io.Writer) int {
	issues := result.Issues()

	output := validationOutput{
		Valid:  result.IsValid() && (!strict || !result.HasWarnings()),
		Issues: make([]validationIssueOutput, 0, len(issues)),
	}

	for _, issue := range issues {
		output.Issues = append(output.Issues, validationIssueOutput{
			Severity: severityToName(issue.Severity),
			Message:  issue.Message,
			Line:     issue.Position.Line,
			Column:   issue.Position.Column,
			Tag:      issue.TagName,
		})
	}

	jsonBytes, _ := json.MarshalIndent(output, "", "  ")
	fmt.Fprintln(stdout, string(jsonBytes))

	if !output.Valid {
		return ExitCodeValidationError
	}
	return ExitCodeSuccess
}

func severityToName(s prompty.ValidationSeverity) string {
	switch s {
	case prompty.SeverityError:
		return SeverityNameError
	case prompty.SeverityWarning:
		return SeverityNameWarning
	case prompty.SeverityInfo:
		return SeverityNameInfo
	default:
		return SeverityNameError
	}
}
