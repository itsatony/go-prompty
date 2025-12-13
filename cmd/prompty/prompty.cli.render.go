package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/itsatony/go-prompty"
)

// renderConfig holds parsed render command configuration
type renderConfig struct {
	templatePath string
	dataJSON     string
	dataFilePath string
	outputPath   string
	quiet        bool
}

func runRender(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	cfg, err := parseRenderFlags(args)
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

	// Parse data
	data, err := loadData(cfg.dataJSON, cfg.dataFilePath)
	if err != nil {
		fmt.Fprintf(stderr, FmtErrorWithCause, ErrMsgInvalidJSON, err)
		return ExitCodeInputError
	}

	// Create engine and execute
	engine := prompty.MustNew()
	result, err := engine.Execute(context.Background(), string(templateSource), data)
	if err != nil {
		fmt.Fprintf(stderr, FmtErrorWithCause, ErrMsgExecuteFailed, err)
		return ExitCodeError
	}

	// Write output
	if err := writeOutput(cfg.outputPath, []byte(result), stdout); err != nil {
		fmt.Fprintf(stderr, FmtErrorWithCause, ErrMsgWriteOutputFailed, err)
		return ExitCodeError
	}

	return ExitCodeSuccess
}

func parseRenderFlags(args []string) (*renderConfig, error) {
	fs := flag.NewFlagSet(CmdNameRender, flag.ContinueOnError)
	fs.SetOutput(io.Discard) // Suppress default error messages

	cfg := &renderConfig{}

	fs.StringVar(&cfg.templatePath, FlagTemplate, "", "")
	fs.StringVar(&cfg.templatePath, FlagTemplateShort, "", "")
	fs.StringVar(&cfg.dataJSON, FlagData, "", "")
	fs.StringVar(&cfg.dataJSON, FlagDataShort, "", "")
	fs.StringVar(&cfg.dataFilePath, FlagDataFile, "", "")
	fs.StringVar(&cfg.dataFilePath, FlagDataFileShort, "", "")
	fs.StringVar(&cfg.outputPath, FlagOutput, FlagDefaultOutput, "")
	fs.StringVar(&cfg.outputPath, FlagOutputShort, FlagDefaultOutput, "")
	fs.BoolVar(&cfg.quiet, FlagQuiet, false, "")
	fs.BoolVar(&cfg.quiet, FlagQuietShort, false, "")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Validation
	if cfg.templatePath == "" {
		return nil, errors.New(ErrMsgMissingTemplate)
	}

	return cfg, nil
}

func loadData(jsonStr, filePath string) (map[string]any, error) {
	var jsonData []byte

	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		jsonData = data
	} else if jsonStr != "" {
		jsonData = []byte(jsonStr)
	} else {
		// No data provided, return empty map
		return make(map[string]any), nil
	}

	var result map[string]any
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, err
	}

	return result, nil
}
