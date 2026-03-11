package add

import (
	"os"
	"regexp"

	"toolbox/internal/manifest"
)

const (
	defaultScope      = "project"
	bundledScope      = "bundled"
	defaultPythonBin  = "python3"
	defaultInputMode  = "none"
	defaultOutputMode = "text"
	specAPIVersion    = "toolbox.add.python/v1"
)

var taskNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// PythonOptions contains CLI-provided options for `toolbox add python`.
type PythonOptions struct {
	CWD         string
	HomeDir     string
	FromSpec    string
	Name        string
	Script      string
	Description string
	Args        []string
	Env         []string
	Timeout     string
	TaskCWD     string
	Tags        []string
	InputMode   string
	OutputMode  string
	PythonBin   string
	Scope       string
	Overwrite   bool
	Changed     map[string]bool
}

// Check records a preflight validation.
type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// PythonResult is returned for successful add operations.
type PythonResult struct {
	Status       string  `json:"status"`
	Task         string  `json:"task"`
	Scope        string  `json:"scope"`
	ManifestPath string  `json:"manifest_path"`
	ScriptPath   string  `json:"script_path"`
	Overwritten  bool    `json:"overwritten"`
	PythonBin    string  `json:"python_bin"`
	Checks       []Check `json:"checks"`
	NextCommand  string  `json:"next_command"`
}

type pythonSpec struct {
	APIVersion  string            `json:"api_version" yaml:"api_version"`
	Name        string            `json:"name" yaml:"name"`
	Script      string            `json:"script" yaml:"script"`
	Description string            `json:"description" yaml:"description"`
	PythonBin   string            `json:"python_bin" yaml:"python_bin"`
	Args        []string          `json:"args" yaml:"args"`
	Env         map[string]string `json:"env" yaml:"env"`
	Timeout     string            `json:"timeout" yaml:"timeout"`
	CWD         string            `json:"cwd" yaml:"cwd"`
	Tags        []string          `json:"tags" yaml:"tags"`
	InputMode   string            `json:"input_mode" yaml:"input_mode"`
	OutputMode  string            `json:"output_mode" yaml:"output_mode"`
	Scope       string            `json:"scope" yaml:"scope"`
	Overwrite   *bool             `json:"overwrite" yaml:"overwrite"`
}

type resolvedPythonConfig struct {
	CWD            string
	HomeDir        string
	Name           string
	SourceScript   string
	Description    string
	PythonBin      string
	Args           []string
	Env            map[string]string
	Timeout        string
	TaskCWD        string
	Tags           []string
	InputMode      manifest.InputMode
	OutputMode     manifest.OutputMode
	Scope          string
	Overwrite      bool
	TaskDir        string
	ScriptDir      string
	ManifestPath   string
	ScriptDestPath string
}

type existingFile struct {
	Data []byte
	Perm os.FileMode
}
