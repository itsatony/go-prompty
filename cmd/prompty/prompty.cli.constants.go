package main

// Command names
const (
	CmdNameRender   = "render"
	CmdNameValidate = "validate"
	CmdNameLint     = "lint"
	CmdNameDebug    = "debug"
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
	FlagRules      = "rules"
	FlagIgnore     = "ignore"
	FlagTrace      = "trace"
	FlagVerbose    = "verbose"
)

// Flag names - short form
const (
	FlagTemplateShort = "t"
	FlagDataShort     = "d"
	FlagDataFileShort = "f"
	FlagOutputShort   = "o"
	FlagQuietShort    = "q"
	FlagFormatShort   = "F"
	FlagRulesShort    = "r"
	FlagIgnoreShort   = "i"
	FlagVerboseShort  = "v"
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
    lint        Check template for style issues and best practices
    debug       Analyze template without executing (dry-run)
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
    lint        Show help for lint command
    debug       Show help for debug command
    version     Show help for version command`

	HelpLintUsage = `Check template for style issues and best practices

Usage:
    prompty lint [options]

Options:
    -t, --template <file>   Template file (use "-" for stdin)
    -F, --format <format>   Output format: text, json (default: text)
    -r, --rules <rules>     Comma-separated list of rules to check
    -i, --ignore <rules>    Comma-separated list of rules to ignore
    --strict                Treat warnings as errors

Lint Rules:
    VAR001    Variable name uses non-standard casing
    VAR002    Variable without default might be missing
    TAG001    Unknown or unregistered tag
    LOOP001   Loop without limit attribute
    LOOP002   Deeply nested loops (> 2 levels)
    EXPR001   Complex expression (> 3 operators)
    INC001    Include references non-existent template

Examples:
    prompty lint -t template.txt
    prompty lint -t template.txt --strict
    prompty lint -t template.txt --ignore VAR002,LOOP001
    prompty lint -t template.txt -F json`

	HelpDebugUsage = `Analyze template without executing (dry-run)

Usage:
    prompty debug [options]

Options:
    -t, --template <file>   Template file (use "-" for stdin)
    -d, --data <json>       JSON data string
    -f, --data-file <file>  JSON data file
    -F, --format <format>   Output format: text, json (default: text)
    --trace                 Show execution trace
    -v, --verbose           Show detailed analysis

Output includes:
    - Variables referenced (with existence check)
    - Resolvers invoked
    - Template includes and their resolution
    - Missing variables with suggestions
    - Unused data fields

Examples:
    prompty debug -t template.txt -d '{"name": "Alice"}'
    prompty debug -t template.txt -f data.json
    prompty debug -t template.txt -f data.json --trace
    prompty debug -t template.txt -f data.json -F json`
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

// Lint rule IDs
const (
	LintRuleVAR001  = "VAR001"  // Variable name non-standard casing
	LintRuleVAR002  = "VAR002"  // Variable without default
	LintRuleTAG001  = "TAG001"  // Unknown tag
	LintRuleLOOP001 = "LOOP001" // Loop without limit
	LintRuleLOOP002 = "LOOP002" // Deeply nested loops
	LintRuleEXPR001 = "EXPR001" // Complex expression
	LintRuleINC001  = "INC001"  // Include non-existent template
)

// Lint output format templates
const (
	LintTextNoIssues     = "No lint issues found"
	LintTextIssueHeader  = "Lint issues:"
	LintTextIssueFormat  = "  [%s] %s: %s (line %d, column %d)"
	LintTextIssueSummary = "%d issue(s) found"
)

// Lint rule descriptions
const (
	LintDescVAR001  = "Variable name should use camelCase or snake_case"
	LintDescVAR002  = "Variable without default value might cause errors"
	LintDescTAG001  = "Unknown or unregistered tag"
	LintDescLOOP001 = "Loop without limit attribute may cause performance issues"
	LintDescLOOP002 = "Deeply nested loops (> 2 levels) reduce readability"
	LintDescEXPR001 = "Complex expression with many operators may be hard to maintain"
	LintDescINC001  = "Include references template that may not exist"
)

// Debug output format templates
const (
	DebugTextHeader          = "=== Template Debug Analysis ==="
	DebugTextVariablesHeader = "Variables (%d found):"
	DebugTextVarExists       = "  ✓ %-20s [line %d]  = %v"
	DebugTextVarMissing      = "  ✗ %-20s [line %d]  MISSING"
	DebugTextVarDefault      = " (default: %s)"
	DebugTextResolversHeader = "Resolvers (%d invocations):"
	DebugTextResolverFormat  = "  %-20s [line %s]"
	DebugTextIncludesHeader  = "Includes (%d found):"
	DebugTextIncludeExists   = "  ✓ %-20s [line %d]  exists"
	DebugTextIncludeMissing  = "  ✗ %-20s [line %d]  NOT FOUND"
	DebugTextUnusedHeader    = "Unused Data Fields:"
	DebugTextUnusedFormat    = "  - %s"
	DebugTextSummary         = "Issues: %d missing variable(s), %d unused field(s)"
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

// Lint rule thresholds
const (
	LintMaxNestedLoopDepth     = 2
	LintMaxExpressionOperators = 3
)

// Debug command thresholds
const (
	DebugMaxLevenshteinDistance = 2
	DebugMaxSuggestions         = 3
	DebugMaxValuePreviewLength  = 30
	DebugTruncationSuffix       = "..."
)

// Format string constants
const (
	FmtErrorWithDetail = "%s: %s\n"
	FmtErrorWithCause  = "%s: %v\n"
	FmtNewline         = "\n"
)
