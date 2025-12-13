package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"

	"gopkg.in/yaml.v3"
)

// versionConfig holds parsed version command configuration
type versionConfig struct {
	format string
}

// versionOutput represents JSON output for version
type versionOutput struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Branch    string `json:"branch"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
}

// versionInfo holds version information
type versionInfo struct {
	Version   string
	Commit    string
	Branch    string
	BuildTime string
	GoVersion string
}

// versionsYAML represents the versions.yaml file structure
type versionsYAML struct {
	Project struct {
		Name        string `yaml:"name"`
		Version     string `yaml:"version"`
		Description string `yaml:"description"`
	} `yaml:"project"`
	Git struct {
		Commit string `yaml:"commit"`
		Branch string `yaml:"branch"`
		Tag    string `yaml:"tag"`
	} `yaml:"git"`
	Build struct {
		Time      string `yaml:"time"`
		GoVersion string `yaml:"go_version"`
	} `yaml:"build"`
}

func runVersion(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseVersionFlags(args)
	if err != nil {
		fmt.Fprintf(stderr, FmtErrorWithCause, ErrMsgInvalidFormat, err)
		return ExitCodeUsageError
	}

	vInfo := getVersionInfo()

	if cfg.format == OutputFormatJSON {
		return outputVersionJSON(vInfo, stdout)
	}
	return outputVersionText(vInfo, stdout)
}

func parseVersionFlags(args []string) (*versionConfig, error) {
	fs := flag.NewFlagSet(CmdNameVersion, flag.ContinueOnError)

	cfg := &versionConfig{}
	fs.StringVar(&cfg.format, FlagFormat, FlagDefaultFormat, "")
	fs.StringVar(&cfg.format, FlagFormatShort, FlagDefaultFormat, "")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if cfg.format != OutputFormatText && cfg.format != OutputFormatJSON {
		return nil, errors.New(ErrMsgInvalidFormat)
	}

	return cfg, nil
}

func getVersionInfo() *versionInfo {
	// Default version info
	vInfo := &versionInfo{
		Version:   VersionUnknown,
		Commit:    VersionUnknown,
		Branch:    VersionUnknown,
		BuildTime: VersionUnknown,
		GoVersion: runtime.Version(),
	}

	// Try to read versions.yaml from current directory or parent
	paths := []string{"versions.yaml", "../versions.yaml", "../../versions.yaml"}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var vy versionsYAML
		if err := yaml.Unmarshal(data, &vy); err != nil {
			continue
		}

		vInfo.Version = vy.Project.Version
		vInfo.Commit = vy.Git.Commit
		vInfo.Branch = vy.Git.Branch
		if vy.Build.Time != "" {
			vInfo.BuildTime = vy.Build.Time
		}
		if vy.Build.GoVersion != "" {
			vInfo.GoVersion = vy.Build.GoVersion
		}
		break
	}

	return vInfo
}

func outputVersionText(v *versionInfo, stdout io.Writer) int {
	fmt.Fprintf(stdout, VersionTextTemplate+FmtNewline,
		v.Version, v.Commit, v.Branch, v.BuildTime, v.GoVersion)
	return ExitCodeSuccess
}

func outputVersionJSON(v *versionInfo, stdout io.Writer) int {
	output := versionOutput{
		Version:   v.Version,
		Commit:    v.Commit,
		Branch:    v.Branch,
		BuildTime: v.BuildTime,
		GoVersion: v.GoVersion,
	}

	jsonBytes, _ := json.MarshalIndent(output, "", "  ")
	fmt.Fprintln(stdout, string(jsonBytes))
	return ExitCodeSuccess
}
