package add

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"toolbox/internal/config"
	"toolbox/internal/manifest"
)

const (
	defaultScope      = "project"
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

type pythonAdder struct {
	lookPath    func(string) (string, error)
	compile     func(string, string) (string, error)
	loadCatalog func(manifest.LoadOptions) manifest.Catalog
}

// AddPython creates a python task manifest and copies the script into toolbox-managed paths.
func AddPython(opts PythonOptions) (PythonResult, error) {
	adder := pythonAdder{
		lookPath:    exec.LookPath,
		compile:     compilePythonScript,
		loadCatalog: manifest.Load,
	}
	return adder.add(opts)
}

func (a pythonAdder) add(opts PythonOptions) (PythonResult, error) {
	cfg, err := a.resolveOptions(opts)
	if err != nil {
		return PythonResult{}, err
	}

	if err := a.checkTaskNameConflicts(cfg); err != nil {
		return PythonResult{}, err
	}

	manifestExists := fileExists(cfg.ManifestPath)
	scriptExists := fileExists(cfg.ScriptDestPath)
	if manifestExists && !cfg.Overwrite {
		return PythonResult{}, fmt.Errorf("task file %q already exists; rerun with --overwrite to replace it", cfg.ManifestPath)
	}
	if scriptExists && !cfg.Overwrite {
		return PythonResult{}, fmt.Errorf("script file %q already exists; rerun with --overwrite to replace it", cfg.ScriptDestPath)
	}

	interpreterPath, err := a.lookPath(cfg.PythonBin)
	if err != nil {
		return PythonResult{}, fmt.Errorf("python interpreter %q was not found in PATH", cfg.PythonBin)
	}

	compileOutput, err := a.compile(interpreterPath, cfg.SourceScript)
	if err != nil {
		trimmed := strings.TrimSpace(compileOutput)
		if trimmed == "" {
			return PythonResult{}, fmt.Errorf("py_compile check failed for %q: %w", cfg.SourceScript, err)
		}
		return PythonResult{}, fmt.Errorf("py_compile check failed for %q: %v (%s)", cfg.SourceScript, err, trimmed)
	}

	task, manifestBytes, err := buildTaskManifest(cfg)
	if err != nil {
		return PythonResult{}, err
	}
	if err := manifest.ValidateTask(task); err != nil {
		return PythonResult{}, fmt.Errorf("generated manifest is invalid: %w", err)
	}

	scriptBytes, scriptPerm, err := readScript(cfg.SourceScript)
	if err != nil {
		return PythonResult{}, err
	}

	overwritten, err := writeScaffoldFiles(cfg, scriptBytes, scriptPerm, manifestBytes)
	if err != nil {
		return PythonResult{}, err
	}

	checks := []Check{
		{Name: "interpreter", Status: "ok"},
		{Name: "py_compile", Status: "ok"},
		{Name: "manifest", Status: "ok"},
	}
	return PythonResult{
		Status:       "created",
		Task:         cfg.Name,
		Scope:        cfg.Scope,
		ManifestPath: cfg.ManifestPath,
		ScriptPath:   cfg.ScriptDestPath,
		Overwritten:  overwritten,
		PythonBin:    cfg.PythonBin,
		Checks:       checks,
		NextCommand:  fmt.Sprintf("toolbox run %s", cfg.Name),
	}, nil
}

func (a pythonAdder) resolveOptions(opts PythonOptions) (resolvedPythonConfig, error) {
	cwd := strings.TrimSpace(opts.CWD)
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return resolvedPythonConfig{}, fmt.Errorf("resolve cwd: %w", err)
		}
	}
	cwdAbs, err := filepath.Abs(cwd)
	if err != nil {
		return resolvedPythonConfig{}, fmt.Errorf("resolve cwd: %w", err)
	}

	home := strings.TrimSpace(opts.HomeDir)
	if home == "" {
		home = os.Getenv("HOME")
	}
	if home == "" {
		home, err = os.UserHomeDir()
		if err != nil {
			return resolvedPythonConfig{}, fmt.Errorf("resolve home directory: %w", err)
		}
	}
	homeAbs, err := filepath.Abs(home)
	if err != nil {
		return resolvedPythonConfig{}, fmt.Errorf("resolve home directory: %w", err)
	}

	var spec *pythonSpec
	if strings.TrimSpace(opts.FromSpec) != "" {
		loaded, err := loadSpec(opts.FromSpec)
		if err != nil {
			return resolvedPythonConfig{}, err
		}
		spec = &loaded
	}

	changed := func(name string) bool {
		return opts.Changed != nil && opts.Changed[name]
	}
	chooseString := func(name, cliValue, specValue, defaultValue string) string {
		if changed(name) {
			return strings.TrimSpace(cliValue)
		}
		if spec != nil && strings.TrimSpace(specValue) != "" {
			return strings.TrimSpace(specValue)
		}
		return defaultValue
	}
	chooseSlice := func(name string, cliValues, specValues []string) []string {
		if changed(name) {
			return append([]string(nil), cliValues...)
		}
		if spec != nil && specValues != nil {
			return append([]string(nil), specValues...)
		}
		return []string{}
	}

	name := chooseString("name", opts.Name, valueFromSpec(spec, func(s *pythonSpec) string { return s.Name }), "")
	if name == "" {
		return resolvedPythonConfig{}, errors.New("--name is required unless provided in --from-spec")
	}
	if !taskNamePattern.MatchString(name) {
		return resolvedPythonConfig{}, fmt.Errorf("task name %q is invalid: use letters, numbers, dots, underscores, and dashes", name)
	}

	scriptInput := chooseString("script", opts.Script, valueFromSpec(spec, func(s *pythonSpec) string { return s.Script }), "")
	if scriptInput == "" {
		return resolvedPythonConfig{}, errors.New("--script is required unless provided in --from-spec")
	}
	sourceScript := scriptInput
	if !filepath.IsAbs(sourceScript) {
		sourceScript = filepath.Join(cwdAbs, sourceScript)
	}
	sourceScript = filepath.Clean(sourceScript)
	info, err := os.Stat(sourceScript)
	if err != nil {
		return resolvedPythonConfig{}, fmt.Errorf("source script %q: %w", sourceScript, err)
	}
	if info.IsDir() {
		return resolvedPythonConfig{}, fmt.Errorf("source script %q is a directory", sourceScript)
	}

	description := chooseString("description", opts.Description, valueFromSpec(spec, func(s *pythonSpec) string { return s.Description }), "")
	args := chooseSlice("arg", opts.Args, sliceFromSpec(spec, func(s *pythonSpec) []string { return s.Args }))
	tags := chooseSlice("tag", opts.Tags, sliceFromSpec(spec, func(s *pythonSpec) []string { return s.Tags }))

	timeout := chooseString("timeout", opts.Timeout, valueFromSpec(spec, func(s *pythonSpec) string { return s.Timeout }), "")
	if timeout != "" {
		duration, err := time.ParseDuration(timeout)
		if err != nil {
			return resolvedPythonConfig{}, fmt.Errorf("invalid timeout %q: %w", timeout, err)
		}
		if duration <= 0 {
			return resolvedPythonConfig{}, errors.New("timeout must be greater than zero")
		}
	}

	taskCWD := chooseString("cwd", opts.TaskCWD, valueFromSpec(spec, func(s *pythonSpec) string { return s.CWD }), "")
	pythonBin := chooseString("python-bin", opts.PythonBin, valueFromSpec(spec, func(s *pythonSpec) string { return s.PythonBin }), defaultPythonBin)
	if pythonBin == "" {
		return resolvedPythonConfig{}, errors.New("python interpreter cannot be empty")
	}
	scope := strings.ToLower(chooseString("scope", opts.Scope, valueFromSpec(spec, func(s *pythonSpec) string { return s.Scope }), defaultScope))
	if scope != "project" && scope != "user" {
		return resolvedPythonConfig{}, fmt.Errorf("invalid scope %q: expected project or user", scope)
	}

	inputModeRaw := strings.ToLower(chooseString("input-mode", opts.InputMode, valueFromSpec(spec, func(s *pythonSpec) string { return s.InputMode }), defaultInputMode))
	inputMode, err := parseInputMode(inputModeRaw)
	if err != nil {
		return resolvedPythonConfig{}, err
	}
	outputModeRaw := strings.ToLower(chooseString("output-mode", opts.OutputMode, valueFromSpec(spec, func(s *pythonSpec) string { return s.OutputMode }), defaultOutputMode))
	outputMode, err := parseOutputMode(outputModeRaw)
	if err != nil {
		return resolvedPythonConfig{}, err
	}

	var env map[string]string
	if changed("env") {
		env, err = parseEnvEntries(opts.Env)
		if err != nil {
			return resolvedPythonConfig{}, err
		}
	} else {
		env = map[string]string{}
		if spec != nil && spec.Env != nil {
			for key, value := range spec.Env {
				trimmedKey := strings.TrimSpace(key)
				if trimmedKey == "" {
					return resolvedPythonConfig{}, errors.New("env keys in --from-spec cannot be empty")
				}
				if strings.Contains(trimmedKey, "=") {
					return resolvedPythonConfig{}, fmt.Errorf("invalid env key %q in --from-spec: keys cannot include '='", key)
				}
				env[trimmedKey] = value
			}
		}
	}

	overwrite := false
	if changed("overwrite") {
		overwrite = opts.Overwrite
	} else if spec != nil && spec.Overwrite != nil {
		overwrite = *spec.Overwrite
	}

	taskDir := config.ProjectTaskDir(cwdAbs)
	scriptDir := config.ProjectScriptDir(cwdAbs)
	if scope == "user" {
		taskDir = config.UserTaskDir(homeAbs)
		scriptDir = config.UserScriptDir(homeAbs)
	}
	manifestPath := filepath.Join(taskDir, name+".yaml")
	scriptDest := filepath.Join(scriptDir, name+".py")

	return resolvedPythonConfig{
		CWD:            cwdAbs,
		HomeDir:        homeAbs,
		Name:           name,
		SourceScript:   sourceScript,
		Description:    description,
		PythonBin:      pythonBin,
		Args:           args,
		Env:            env,
		Timeout:        timeout,
		TaskCWD:        taskCWD,
		Tags:           tags,
		InputMode:      inputMode,
		OutputMode:     outputMode,
		Scope:          scope,
		Overwrite:      overwrite,
		TaskDir:        taskDir,
		ScriptDir:      scriptDir,
		ManifestPath:   manifestPath,
		ScriptDestPath: scriptDest,
	}, nil
}

func (a pythonAdder) checkTaskNameConflicts(cfg resolvedPythonConfig) error {
	catalog := a.loadCatalog(manifest.LoadOptions{
		ProjectDir: config.ProjectTaskDir(cfg.CWD),
		UserDir:    config.UserTaskDir(cfg.HomeDir),
	})

	if sources, ok := catalog.DuplicateNames[cfg.Name]; ok && len(sources) > 0 {
		parts := make([]string, 0, len(sources))
		for _, source := range sources {
			parts = append(parts, fmt.Sprintf("%s (%s)", source.Path, source.Scope))
		}
		return fmt.Errorf("task name %q already exists in multiple manifests: %s", cfg.Name, strings.Join(parts, ", "))
	}

	existingTask, exists := catalog.Tasks[cfg.Name]
	if !exists {
		return nil
	}
	existingPath := filepath.Clean(existingTask.Source.Path)
	targetPath := filepath.Clean(cfg.ManifestPath)
	if existingPath != targetPath {
		return fmt.Errorf("task name %q already exists at %q (%s scope); choose a different --name", cfg.Name, existingTask.Source.Path, existingTask.Source.Scope)
	}
	return nil
}

func buildTaskManifest(cfg resolvedPythonConfig) (manifest.Task, []byte, error) {
	args := make([]string, 0, len(cfg.Args)+1)
	args = append(args, cfg.ScriptDestPath)
	args = append(args, cfg.Args...)

	task := manifest.Task{
		Name:        cfg.Name,
		Description: cfg.Description,
		Command:     cfg.PythonBin,
		Args:        args,
		Input:       manifest.InputSpec{Mode: cfg.InputMode},
		Output:      manifest.OutputSpec{Mode: cfg.OutputMode},
		TimeoutRaw:  cfg.Timeout,
		Env:         cloneMap(cfg.Env),
		Requires:    []string{cfg.PythonBin},
		Tags:        append([]string(nil), cfg.Tags...),
		CWD:         cfg.TaskCWD,
	}

	type taskFile struct {
		Name        string              `yaml:"name"`
		Description string              `yaml:"description,omitempty"`
		Command     string              `yaml:"command"`
		Args        []string            `yaml:"args"`
		Input       manifest.InputSpec  `yaml:"input"`
		Output      manifest.OutputSpec `yaml:"output"`
		Timeout     string              `yaml:"timeout,omitempty"`
		Env         map[string]string   `yaml:"env,omitempty"`
		Requires    []string            `yaml:"requires,omitempty"`
		Tags        []string            `yaml:"tags,omitempty"`
		CWD         string              `yaml:"cwd,omitempty"`
	}

	fileModel := taskFile{
		Name:        task.Name,
		Description: task.Description,
		Command:     task.Command,
		Args:        task.Args,
		Input:       task.Input,
		Output:      task.Output,
		Timeout:     task.TimeoutRaw,
		Env:         cloneMap(task.Env),
		Requires:    append([]string(nil), task.Requires...),
		Tags:        append([]string(nil), task.Tags...),
		CWD:         task.CWD,
	}

	data, err := yaml.Marshal(fileModel)
	if err != nil {
		return manifest.Task{}, nil, fmt.Errorf("marshal task manifest: %w", err)
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	return task, data, nil
}

func parseInputMode(raw string) (manifest.InputMode, error) {
	mode := manifest.InputMode(raw)
	switch mode {
	case manifest.InputModeNone, manifest.InputModeFile, manifest.InputModeJSON:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid input mode %q: expected none, file, or json", raw)
	}
}

func parseOutputMode(raw string) (manifest.OutputMode, error) {
	mode := manifest.OutputMode(raw)
	switch mode {
	case manifest.OutputModeText, manifest.OutputModeJSON:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid output mode %q: expected text or json", raw)
	}
}

func parseEnvEntries(entries []string) (map[string]string, error) {
	env := map[string]string{}
	for _, entry := range entries {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --env value %q: expected KEY=VALUE", entry)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, fmt.Errorf("invalid --env value %q: key cannot be empty", entry)
		}
		env[key] = parts[1]
	}
	return env, nil
}

func readScript(path string) ([]byte, os.FileMode, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, fmt.Errorf("read source script %q: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, 0, fmt.Errorf("stat source script %q: %w", path, err)
	}
	perm := info.Mode().Perm()
	if perm == 0 {
		perm = 0o644
	}
	return data, perm, nil
}

func writeScaffoldFiles(cfg resolvedPythonConfig, script []byte, scriptPerm os.FileMode, manifestBytes []byte) (bool, error) {
	if err := os.MkdirAll(cfg.ScriptDir, 0o755); err != nil {
		return false, fmt.Errorf("create script directory %q: %w", cfg.ScriptDir, err)
	}
	if err := os.MkdirAll(cfg.TaskDir, 0o755); err != nil {
		return false, fmt.Errorf("create task directory %q: %w", cfg.TaskDir, err)
	}

	previousScript, hadScript, err := snapshotFile(cfg.ScriptDestPath)
	if err != nil {
		return false, err
	}
	_, hadManifest, err := snapshotFile(cfg.ManifestPath)
	if err != nil {
		return false, err
	}

	if err := atomicWriteFile(cfg.ScriptDestPath, script, scriptPerm); err != nil {
		return false, err
	}
	if err := atomicWriteFile(cfg.ManifestPath, manifestBytes, 0o644); err != nil {
		if hadScript {
			_ = atomicWriteFile(cfg.ScriptDestPath, previousScript.Data, previousScript.Perm)
		} else {
			_ = os.Remove(cfg.ScriptDestPath)
		}
		return false, err
	}

	return hadScript || hadManifest, nil
}

func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".toolbox-add-*")
	if err != nil {
		return fmt.Errorf("create temporary file in %q: %w", dir, err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary file for %q: %w", path, err)
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temporary file for %q: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary file for %q: %w", path, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace %q: %w", path, err)
	}
	return nil
}

func snapshotFile(path string) (existingFile, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return existingFile{}, false, nil
		}
		return existingFile{}, false, fmt.Errorf("read existing file %q: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return existingFile{}, false, fmt.Errorf("stat existing file %q: %w", path, err)
	}
	return existingFile{Data: data, Perm: info.Mode().Perm()}, true, nil
}

func compilePythonScript(interpreterPath, scriptPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, interpreterPath, "-m", "py_compile", scriptPath)
	out, err := cmd.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return string(out), errors.New("timed out")
	}
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

func loadSpec(path string) (pythonSpec, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return pythonSpec{}, fmt.Errorf("resolve --from-spec path: %w", err)
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return pythonSpec{}, fmt.Errorf("read --from-spec file %q: %w", absPath, err)
	}

	var spec pythonSpec
	switch strings.ToLower(filepath.Ext(absPath)) {
	case ".json":
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&spec); err != nil {
			return pythonSpec{}, fmt.Errorf("parse --from-spec JSON %q: %w", absPath, err)
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			if err == nil {
				return pythonSpec{}, fmt.Errorf("parse --from-spec JSON %q: multiple documents are not allowed", absPath)
			}
			return pythonSpec{}, fmt.Errorf("parse --from-spec JSON %q: %w", absPath, err)
		}
	default:
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		if err := decoder.Decode(&spec); err != nil {
			return pythonSpec{}, fmt.Errorf("parse --from-spec YAML %q: %w", absPath, err)
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			if err == nil {
				return pythonSpec{}, fmt.Errorf("parse --from-spec YAML %q: multiple documents are not allowed", absPath)
			}
			return pythonSpec{}, fmt.Errorf("parse --from-spec YAML %q: %w", absPath, err)
		}
	}

	if strings.TrimSpace(spec.APIVersion) != specAPIVersion {
		return pythonSpec{}, fmt.Errorf("--from-spec api_version must be %q", specAPIVersion)
	}
	return spec, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func valueFromSpec(spec *pythonSpec, getter func(*pythonSpec) string) string {
	if spec == nil {
		return ""
	}
	return getter(spec)
}

func sliceFromSpec(spec *pythonSpec, getter func(*pythonSpec) []string) []string {
	if spec == nil {
		return nil
	}
	return getter(spec)
}

func cloneMap(value map[string]string) map[string]string {
	if len(value) == 0 {
		return nil
	}
	out := make(map[string]string, len(value))
	for key, val := range value {
		out[key] = val
	}
	return out
}
