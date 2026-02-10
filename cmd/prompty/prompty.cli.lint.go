package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/itsatony/go-prompty/v2"
)

// lintConfig holds parsed lint command configuration
type lintConfig struct {
	templatePath string
	format       string
	rules        string
	ignore       string
	strict       bool
}

// lintIssue represents a single lint issue
type lintIssue struct {
	RuleID   string `json:"rule_id"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	TagName  string `json:"tag,omitempty"`
}

// lintOutput represents JSON output for lint
type lintOutput struct {
	Valid  bool        `json:"valid"`
	Issues []lintIssue `json:"issues,omitempty"`
}

// lintRuleSet tracks which rules to apply
type lintRuleSet struct {
	enabledRules map[string]bool
}

func newLintRuleSet(rules, ignore string) *lintRuleSet {
	rs := &lintRuleSet{
		enabledRules: map[string]bool{
			LintRuleVAR001:  true,
			LintRuleVAR002:  true,
			LintRuleTAG001:  true,
			LintRuleLOOP001: true,
			LintRuleLOOP002: true,
			LintRuleEXPR001: true,
			LintRuleINC001:  true,
		},
	}

	// If specific rules requested, only enable those
	if rules != "" {
		for k := range rs.enabledRules {
			rs.enabledRules[k] = false
		}
		for _, r := range strings.Split(rules, ",") {
			r = strings.TrimSpace(strings.ToUpper(r))
			rs.enabledRules[r] = true
		}
	}

	// Disable ignored rules
	if ignore != "" {
		for _, r := range strings.Split(ignore, ",") {
			r = strings.TrimSpace(strings.ToUpper(r))
			rs.enabledRules[r] = false
		}
	}

	return rs
}

func (rs *lintRuleSet) isEnabled(rule string) bool {
	return rs.enabledRules[rule]
}

func runLint(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	cfg, err := parseLintFlags(args)
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

	// Create rule set
	ruleSet := newLintRuleSet(cfg.rules, cfg.ignore)

	// Run lint checks
	issues := lintTemplate(string(templateSource), ruleSet)

	// Output based on format
	if cfg.format == OutputFormatJSON {
		return outputLintJSON(issues, cfg.strict, stdout)
	}
	return outputLintText(issues, cfg.strict, stdout)
}

func parseLintFlags(args []string) (*lintConfig, error) {
	fs := flag.NewFlagSet(CmdNameLint, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	cfg := &lintConfig{}

	fs.StringVar(&cfg.templatePath, FlagTemplate, "", "")
	fs.StringVar(&cfg.templatePath, FlagTemplateShort, "", "")
	fs.StringVar(&cfg.format, FlagFormat, FlagDefaultFormat, "")
	fs.StringVar(&cfg.format, FlagFormatShort, FlagDefaultFormat, "")
	fs.StringVar(&cfg.rules, FlagRules, "", "")
	fs.StringVar(&cfg.rules, FlagRulesShort, "", "")
	fs.StringVar(&cfg.ignore, FlagIgnore, "", "")
	fs.StringVar(&cfg.ignore, FlagIgnoreShort, "", "")
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

// lintTemplate performs lint checks on the template source
func lintTemplate(source string, ruleSet *lintRuleSet) []lintIssue {
	var issues []lintIssue

	// First, parse the template to get validation issues
	engine := prompty.MustNew()
	validationResult, err := engine.Validate(source)
	if err != nil {
		// If parsing fails, report as error
		issues = append(issues, lintIssue{
			RuleID:   LintRuleTAG001,
			Severity: SeverityNameError,
			Message:  err.Error(),
			Line:     1,
			Column:   1,
		})
		return issues
	}

	// Convert validation issues to lint issues
	for _, vi := range validationResult.Issues() {
		if ruleSet.isEnabled(LintRuleTAG001) {
			issues = append(issues, lintIssue{
				RuleID:   LintRuleTAG001,
				Severity: severityToName(vi.Severity),
				Message:  vi.Message,
				Line:     vi.Position.Line,
				Column:   vi.Position.Column,
				TagName:  vi.TagName,
			})
		}
	}

	// Parse template for DryRun analysis
	tmpl, parseErr := engine.Parse(source)
	if parseErr == nil {
		// Run style checks via DryRun analysis
		dryRunResult := tmpl.DryRun(context.Background(), nil)
		issues = append(issues, checkVariableRules(dryRunResult, ruleSet)...)
		issues = append(issues, checkIncludeRules(dryRunResult, ruleSet, engine)...)
	}
	issues = append(issues, checkLoopRules(source, ruleSet)...)
	issues = append(issues, checkExpressionRules(source, ruleSet)...)

	return issues
}

// checkVariableRules checks VAR001 and VAR002 rules
func checkVariableRules(result *prompty.DryRunResult, ruleSet *lintRuleSet) []lintIssue {
	var issues []lintIssue

	// Pattern for standard naming conventions (camelCase, snake_case, dot.notation)
	validNamePattern := regexp.MustCompile(`^[a-z][a-zA-Z0-9]*(\.[a-z][a-zA-Z0-9]*)*$|^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$`)

	for _, v := range result.Variables {
		// VAR001: Check variable naming convention
		if ruleSet.isEnabled(LintRuleVAR001) {
			if !validNamePattern.MatchString(v.Name) {
				issues = append(issues, lintIssue{
					RuleID:   LintRuleVAR001,
					Severity: SeverityNameWarning,
					Message:  fmt.Sprintf("%s: '%s'", LintDescVAR001, v.Name),
					Line:     v.Line,
					Column:   v.Column,
				})
			}
		}

		// VAR002: Check for variables without defaults
		if ruleSet.isEnabled(LintRuleVAR002) {
			if !v.InData && v.Default == "" {
				issues = append(issues, lintIssue{
					RuleID:   LintRuleVAR002,
					Severity: SeverityNameWarning,
					Message:  fmt.Sprintf("%s: '%s'", LintDescVAR002, v.Name),
					Line:     v.Line,
					Column:   v.Column,
				})
			}
		}
	}

	return issues
}

// checkLoopRules checks LOOP001 and LOOP002 rules
func checkLoopRules(source string, ruleSet *lintRuleSet) []lintIssue {
	var issues []lintIssue

	// LOOP001: Check for loops without limit attribute
	if ruleSet.isEnabled(LintRuleLOOP001) {
		// Simple pattern to find for loops
		loopPattern := regexp.MustCompile(`\{~prompty\.for\s+[^}]*~\}`)
		limitPattern := regexp.MustCompile(`limit\s*=`)

		matches := loopPattern.FindAllStringIndex(source, -1)
		for _, match := range matches {
			loopTag := source[match[0]:match[1]]
			if !limitPattern.MatchString(loopTag) {
				line, col := positionFromOffset(source, match[0])
				issues = append(issues, lintIssue{
					RuleID:   LintRuleLOOP001,
					Severity: SeverityNameWarning,
					Message:  LintDescLOOP001,
					Line:     line,
					Column:   col,
				})
			}
		}
	}

	// LOOP002: Check for deeply nested loops (simple heuristic)
	if ruleSet.isEnabled(LintRuleLOOP002) {
		// Count nested loop depth
		depth := 0
		maxDepth := 0
		maxDepthPos := 0

		loopOpenPattern := regexp.MustCompile(`\{~prompty\.for\s+`)
		loopClosePattern := regexp.MustCompile(`\{~/prompty\.for~\}`)

		// Find all loop opens and closes
		opens := loopOpenPattern.FindAllStringIndex(source, -1)
		closes := loopClosePattern.FindAllStringIndex(source, -1)

		// Merge and sort by position
		type event struct {
			pos     int
			isOpen  bool
			origIdx int
		}
		var events []event
		for i, o := range opens {
			events = append(events, event{pos: o[0], isOpen: true, origIdx: i})
		}
		for i, c := range closes {
			events = append(events, event{pos: c[0], isOpen: false, origIdx: i})
		}

		// Sort by position
		for i := 0; i < len(events)-1; i++ {
			for j := i + 1; j < len(events); j++ {
				if events[j].pos < events[i].pos {
					events[i], events[j] = events[j], events[i]
				}
			}
		}

		for _, e := range events {
			if e.isOpen {
				depth++
				if depth > maxDepth {
					maxDepth = depth
					maxDepthPos = e.pos
				}
			} else {
				depth--
			}
		}

		if maxDepth > LintMaxNestedLoopDepth {
			line, col := positionFromOffset(source, maxDepthPos)
			issues = append(issues, lintIssue{
				RuleID:   LintRuleLOOP002,
				Severity: SeverityNameWarning,
				Message:  fmt.Sprintf("%s (%d levels)", LintDescLOOP002, maxDepth),
				Line:     line,
				Column:   col,
			})
		}
	}

	return issues
}

// checkExpressionRules checks EXPR001 rule
func checkExpressionRules(source string, ruleSet *lintRuleSet) []lintIssue {
	var issues []lintIssue

	if !ruleSet.isEnabled(LintRuleEXPR001) {
		return issues
	}

	// Find eval attributes and count operators
	evalPattern := regexp.MustCompile(`eval\s*=\s*"([^"]*)"`)
	operatorPattern := regexp.MustCompile(`(&&|\|\||==|!=|<=|>=|<|>)`)

	matches := evalPattern.FindAllStringSubmatchIndex(source, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			expr := source[match[2]:match[3]]
			operatorCount := len(operatorPattern.FindAllString(expr, -1))
			if operatorCount > LintMaxExpressionOperators {
				line, col := positionFromOffset(source, match[0])
				issues = append(issues, lintIssue{
					RuleID:   LintRuleEXPR001,
					Severity: SeverityNameInfo,
					Message:  fmt.Sprintf("%s (%d operators)", LintDescEXPR001, operatorCount),
					Line:     line,
					Column:   col,
				})
			}
		}
	}

	return issues
}

// checkIncludeRules checks INC001 rule
func checkIncludeRules(result *prompty.DryRunResult, ruleSet *lintRuleSet, engine *prompty.Engine) []lintIssue {
	var issues []lintIssue

	if !ruleSet.isEnabled(LintRuleINC001) {
		return issues
	}

	for _, inc := range result.Includes {
		if !engine.HasTemplate(inc.TemplateName) {
			issues = append(issues, lintIssue{
				RuleID:   LintRuleINC001,
				Severity: SeverityNameWarning,
				Message:  fmt.Sprintf("%s: '%s'", LintDescINC001, inc.TemplateName),
				Line:     inc.Line,
				Column:   inc.Column,
			})
		}
	}

	return issues
}

// positionFromOffset converts a byte offset to line and column numbers
func positionFromOffset(source string, offset int) (int, int) {
	if offset < 0 || offset >= len(source) {
		return 1, 1
	}

	line := 1
	col := 1
	for i := 0; i < offset; i++ {
		if source[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}

func outputLintText(issues []lintIssue, strict bool, stdout io.Writer) int {
	if len(issues) == 0 {
		fmt.Fprintln(stdout, LintTextNoIssues)
		return ExitCodeSuccess
	}

	fmt.Fprintln(stdout, LintTextIssueHeader)
	hasErrors := false
	hasWarnings := false

	for _, issue := range issues {
		fmt.Fprintf(stdout, LintTextIssueFormat+FmtNewline,
			issue.RuleID, issue.Severity, issue.Message, issue.Line, issue.Column)
		if issue.Severity == SeverityNameError {
			hasErrors = true
		}
		if issue.Severity == SeverityNameWarning {
			hasWarnings = true
		}
	}

	fmt.Fprintf(stdout, LintTextIssueSummary+FmtNewline, len(issues))

	if hasErrors || (strict && hasWarnings) {
		return ExitCodeValidationError
	}
	return ExitCodeSuccess
}

func outputLintJSON(issues []lintIssue, strict bool, stdout io.Writer) int {
	hasErrors := false
	hasWarnings := false

	for _, issue := range issues {
		if issue.Severity == SeverityNameError {
			hasErrors = true
		}
		if issue.Severity == SeverityNameWarning {
			hasWarnings = true
		}
	}

	output := lintOutput{
		Valid:  len(issues) == 0 || (!hasErrors && (!strict || !hasWarnings)),
		Issues: issues,
	}

	jsonBytes, _ := json.MarshalIndent(output, "", "  ")
	fmt.Fprintln(stdout, string(jsonBytes))

	if hasErrors || (strict && hasWarnings) {
		return ExitCodeValidationError
	}
	return ExitCodeSuccess
}
