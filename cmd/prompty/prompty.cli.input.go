package main

import (
	"io"
	"os"
)

// readInput reads content from a file or stdin
func readInput(path string, stdin io.Reader) ([]byte, error) {
	if path == InputSourceStdin {
		return io.ReadAll(stdin)
	}

	return os.ReadFile(path)
}

// writeOutput writes content to a file or stdout
func writeOutput(path string, data []byte, stdout io.Writer) error {
	if path == FlagDefaultOutput {
		_, err := stdout.Write(data)
		return err
	}

	return os.WriteFile(path, data, FilePermissions)
}
