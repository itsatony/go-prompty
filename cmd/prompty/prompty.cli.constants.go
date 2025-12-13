package main

// Command names
const (
	CmdNameRender   = "render"
	CmdNameValidate = "validate"
	CmdNameVersion  = "version"
	CmdNameHelp     = "help"
)

// Flag names - long form
const (
	FlagTemplate   = "template"
	FlagData       = "data"
	FlagDataFile   = "data-file"
	FlagOutput     = "output"
	FlagQuiet      = "quiet"
	FlagFormat     = "format"
	FlagStrictMode = "strict"
)

// Flag names - short form
const (
	FlagTemplateShort = "t"
	FlagDataShort     = "d"
	FlagDataFileShort = "f"
	FlagOutputShort   = "o"
	FlagQuietShort    = "q"
	FlagFormatShort   = "F"
)

// Flag default values
const (
	FlagDefaultOutput = "-" // stdout
	FlagDefaultFormat = "text"
)

// Output formats
const (
	OutputFormatText = "text"
	OutputFormatJSON = "json"
)

// Exit codes
const (
	ExitCodeSuccess         = 0
	ExitCodeError           = 1
	ExitCodeUsageError      = 2
	ExitCodeValidationError = 3
	ExitCodeInputError      = 4
)

// Input source indicators
const (
	InputSourceStdin = "-"
)

// Error messages - ALL must be constants
const (
	ErrMsgNoCommand           = "no command specified"
	ErrMsgUnknownCommand      = "unknown command"
	ErrMsgMissingTemplate     = "template source required"
	ErrMsgInvalidJSON         = "invalid JSON data"
	ErrMsgReadFileFailed      = "failed to read file"
	ErrMsgReadStdinFailed     = "failed to read from stdin"
	ErrMsgWriteOutputFailed   = "failed to write output"
	ErrMsgParseTemplateFailed = "template parsing failed"
	ErrMsgExecuteFailed       = "template execution failed"
	ErrMsgInvalidFormat       = "invalid output format"
	ErrMsgOpenFileFailed      = "failed to open file"
	ErrMsgCreateFileFailed    = "failed to create output file"
	ErrMsgJSONMarshalFailed   = "failed to marshal JSON"
	ErrMsgJSONUnmarshalFailed = "failed to unmarshal JSON"
)

// Help text templates
const (
	HelpMainUsage = `go-prompty - Dynamic LLM prompt templating CLI

Usage:
    prompty <command> [options]

Commands:
    render      Render a template with data
    validate    Validate a template without executing
    version     Show version information
    help        Show help for a command

Use "prompty help <command>" for more information about a command.`

	HelpRenderUsage = `Render a template with data

Usage:
    prompty render [options]

Options:
    -t, --template <file>   Template file (use "-" for stdin)
    -d, --data <json>       JSON data string
    -f, --data-file <file>  JSON data file
    -o, --output <file>     Output file (default: stdout)
    -q, --quiet             Suppress non-error output

Examples:
    prompty render -t template.txt -d '{"name": "Alice"}'
    prompty render -t template.txt -f data.json
    cat template.txt | prompty render -t - -d '{"name": "Bob"}'
    prompty render -t template.txt -f data.json -o output.txt`

	HelpValidateUsage = `Validate a template without executing

Usage:
    prompty validate [options]

Options:
    -t, --template <file>   Template file (use "-" for stdin)
    -F, --format <format>   Output format: text, json (default: text)
    --strict                Treat warnings as errors

Examples:
    prompty validate -t template.txt
    prompty validate -t template.txt --strict
    cat template.txt | prompty validate -t -`

	HelpVersionUsage = `Show version information

Usage:
    prompty version [options]

Options:
    -F, --format <format>   Output format: text, json (default: text)`

	HelpHelpUsage = `Show help for a command

Usage:
    prompty help [command]

Commands:
    render      Show help for render command
    validate    Show help for validate command
    version     Show help for version command`
)

// Version output format templates
const (
	VersionTextTemplate = "go-prompty version %s\nCommit: %s\nBranch: %s\nBuilt: %s\nGo: %s"
	VersionUnknown      = "unknown"
)

// Validation output format templates
const (
	ValidationTextSuccess      = "Template is valid"
	ValidationTextIssueHeader  = "Validation issues:"
	ValidationTextIssueFormat  = "  [%s] %s at line %d, column %d"
	ValidationTextErrorSummary = "%d error(s), %d warning(s)"
)

// Severity names for output
const (
	SeverityNameError   = "ERROR"
	SeverityNameWarning = "WARNING"
	SeverityNameInfo    = "INFO"
)

// CLI metadata
const (
	CLIName        = "prompty"
	CLIDescription = "Dynamic LLM prompt templating CLI"
)

// File permission constant
const (
	FilePermissions = 0644
)

// Format string constants
const (
	FmtErrorWithDetail = "%s: %s\n"
	FmtErrorWithCause  = "%s: %v\n"
	FmtNewline         = "\n"
)
