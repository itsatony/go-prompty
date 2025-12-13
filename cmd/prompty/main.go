package main

import (
	"io"
	"os"
)

func main() {
	exitCode := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(exitCode)
}

// run is the main entry point for the CLI, separated for testing
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return runHelp(nil, stdout)
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case CmdNameRender:
		return runRender(cmdArgs, stdin, stdout, stderr)
	case CmdNameValidate:
		return runValidate(cmdArgs, stdin, stdout, stderr)
	case CmdNameVersion:
		return runVersion(cmdArgs, stdout, stderr)
	case CmdNameHelp:
		return runHelp(cmdArgs, stdout)
	default:
		// Unknown command - show error and help
		return runHelp([]string{cmd}, stdout)
	}
}
