package main

import (
	"fmt"
	"io"
)

func runHelp(args []string, stdout io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stdout, HelpMainUsage)
		return ExitCodeSuccess
	}

	cmd := args[0]
	switch cmd {
	case CmdNameRender:
		fmt.Fprintln(stdout, HelpRenderUsage)
	case CmdNameValidate:
		fmt.Fprintln(stdout, HelpValidateUsage)
	case CmdNameLint:
		fmt.Fprintln(stdout, HelpLintUsage)
	case CmdNameDebug:
		fmt.Fprintln(stdout, HelpDebugUsage)
	case CmdNameVersion:
		fmt.Fprintln(stdout, HelpVersionUsage)
	case CmdNameHelp:
		fmt.Fprintln(stdout, HelpHelpUsage)
	default:
		fmt.Fprintf(stdout, FmtErrorWithDetail, ErrMsgUnknownCommand, cmd)
		fmt.Fprintln(stdout, HelpMainUsage)
		return ExitCodeUsageError
	}

	return ExitCodeSuccess
}
