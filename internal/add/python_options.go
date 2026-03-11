package add

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"toolbox/internal/config"
	"toolbox/internal/manifest"
)

func (a pythonAdder) resolveOptions(opts PythonOptions) (resolvedPythonConfig, error) {
	cwdAbs, homeAbs, err := resolveBaseDirs(opts)
	if err != nil {
		return resolvedPythonConfig{}, err
	}

	spec, err := loadOptionalSpec(opts.FromSpec)
	if err != nil {
		return resolvedPythonConfig{}, err
	}

	chooser := newOptionChooser(opts, spec)
	name, err := resolveTaskName(chooser)
	if err != nil {
		return resolvedPythonConfig{}, err
	}
	sourceScript, err := resolveSourceScript(cwdAbs, chooser)
	if err != nil {
		return resolvedPythonConfig{}, err
	}
	timeout, err := resolveTimeout(chooser)
	if err != nil {
		return resolvedPythonConfig{}, err
	}
	scope, err := resolveScope(chooser)
	if err != nil {
		return resolvedPythonConfig{}, err
	}
	inputMode, err := resolveInputMode(chooser)
	if err != nil {
		return resolvedPythonConfig{}, err
	}
	outputMode, err := resolveOutputMode(chooser)
	if err != nil {
		return resolvedPythonConfig{}, err
	}
	env, err := resolveEnv(opts, chooser, spec)
	if err != nil {
		return resolvedPythonConfig{}, err
	}
	taskDir, scriptDir := resolveTargetDirs(scope, cwdAbs, homeAbs)
	manifestPath := filepath.Join(taskDir, name+".yaml")
	scriptDest := filepath.Join(scriptDir, name+".py")

	return resolvedPythonConfig{
		CWD:            cwdAbs,
		HomeDir:        homeAbs,
		Name:           name,
		SourceScript:   sourceScript,
		Description:    chooser.string("description", opts.Description, valueFromSpec(spec, func(s *pythonSpec) string { return s.Description }), ""),
		PythonBin:      chooser.string("python-bin", opts.PythonBin, valueFromSpec(spec, func(s *pythonSpec) string { return s.PythonBin }), defaultPythonBin),
		Args:           chooser.slice("arg", opts.Args, sliceFromSpec(spec, func(s *pythonSpec) []string { return s.Args })),
		Env:            env,
		Timeout:        timeout,
		TaskCWD:        chooser.string("cwd", opts.TaskCWD, valueFromSpec(spec, func(s *pythonSpec) string { return s.CWD }), ""),
		Tags:           chooser.slice("tag", opts.Tags, sliceFromSpec(spec, func(s *pythonSpec) []string { return s.Tags })),
		InputMode:      inputMode,
		OutputMode:     outputMode,
		Scope:          scope,
		Overwrite:      resolveOverwrite(opts, chooser, spec),
		TaskDir:        taskDir,
		ScriptDir:      scriptDir,
		ManifestPath:   manifestPath,
		ScriptDestPath: scriptDest,
	}, nil
}

type optionChooser struct {
	opts PythonOptions
	spec *pythonSpec
}

func newOptionChooser(opts PythonOptions, spec *pythonSpec) optionChooser {
	return optionChooser{opts: opts, spec: spec}
}

func (o optionChooser) changed(name string) bool {
	return o.opts.Changed != nil && o.opts.Changed[name]
}

func (o optionChooser) string(name, cliValue, specValue, defaultValue string) string {
	if o.changed(name) {
		return strings.TrimSpace(cliValue)
	}
	if o.spec != nil && strings.TrimSpace(specValue) != "" {
		return strings.TrimSpace(specValue)
	}
	return defaultValue
}

func (o optionChooser) slice(name string, cliValues, specValues []string) []string {
	if o.changed(name) {
		return append([]string(nil), cliValues...)
	}
	if o.spec != nil && specValues != nil {
		return append([]string(nil), specValues...)
	}
	return []string{}
}

func resolveBaseDirs(opts PythonOptions) (string, string, error) {
	cwd := strings.TrimSpace(opts.CWD)
	if cwd == "" {
		resolved, err := os.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("resolve cwd: %w", err)
		}
		cwd = resolved
	}
	cwdAbs, err := filepath.Abs(cwd)
	if err != nil {
		return "", "", fmt.Errorf("resolve cwd: %w", err)
	}

	home := strings.TrimSpace(opts.HomeDir)
	if home == "" {
		home = os.Getenv("HOME")
	}
	if home == "" {
		resolved, err := os.UserHomeDir()
		if err != nil {
			return "", "", fmt.Errorf("resolve home directory: %w", err)
		}
		home = resolved
	}
	homeAbs, err := filepath.Abs(home)
	if err != nil {
		return "", "", fmt.Errorf("resolve home directory: %w", err)
	}
	return cwdAbs, homeAbs, nil
}

func loadOptionalSpec(path string) (*pythonSpec, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	loaded, err := loadSpec(path)
	if err != nil {
		return nil, err
	}
	return &loaded, nil
}

func resolveTaskName(chooser optionChooser) (string, error) {
	name := chooser.string("name", chooser.opts.Name, valueFromSpec(chooser.spec, func(s *pythonSpec) string { return s.Name }), "")
	if name == "" {
		return "", errors.New("--name is required unless provided in --from-spec")
	}
	if !taskNamePattern.MatchString(name) {
		return "", fmt.Errorf("task name %q is invalid: use letters, numbers, dots, underscores, and dashes", name)
	}
	return name, nil
}

func resolveSourceScript(cwdAbs string, chooser optionChooser) (string, error) {
	scriptInput := chooser.string("script", chooser.opts.Script, valueFromSpec(chooser.spec, func(s *pythonSpec) string { return s.Script }), "")
	if scriptInput == "" {
		return "", errors.New("--script is required unless provided in --from-spec")
	}
	sourceScript := scriptInput
	if !filepath.IsAbs(sourceScript) {
		sourceScript = filepath.Join(cwdAbs, sourceScript)
	}
	sourceScript = filepath.Clean(sourceScript)
	info, err := os.Stat(sourceScript)
	if err != nil {
		return "", fmt.Errorf("source script %q: %w", sourceScript, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("source script %q is a directory", sourceScript)
	}
	return sourceScript, nil
}

func resolveTimeout(chooser optionChooser) (string, error) {
	timeout := chooser.string("timeout", chooser.opts.Timeout, valueFromSpec(chooser.spec, func(s *pythonSpec) string { return s.Timeout }), "")
	if timeout == "" {
		return "", nil
	}
	duration, err := time.ParseDuration(timeout)
	if err != nil {
		return "", fmt.Errorf("invalid timeout %q: %w", timeout, err)
	}
	if duration <= 0 {
		return "", errors.New("timeout must be greater than zero")
	}
	return timeout, nil
}

func resolveScope(chooser optionChooser) (string, error) {
	scope := strings.ToLower(chooser.string("scope", chooser.opts.Scope, valueFromSpec(chooser.spec, func(s *pythonSpec) string { return s.Scope }), defaultScope))
	if scope != "project" && scope != "user" && scope != bundledScope {
		return "", fmt.Errorf("invalid scope %q: expected project, user, or bundled", scope)
	}
	return scope, nil
}

func resolveInputMode(chooser optionChooser) (manifest.InputMode, error) {
	raw := strings.ToLower(chooser.string("input-mode", chooser.opts.InputMode, valueFromSpec(chooser.spec, func(s *pythonSpec) string { return s.InputMode }), defaultInputMode))
	return parseInputMode(raw)
}

func resolveOutputMode(chooser optionChooser) (manifest.OutputMode, error) {
	raw := strings.ToLower(chooser.string("output-mode", chooser.opts.OutputMode, valueFromSpec(chooser.spec, func(s *pythonSpec) string { return s.OutputMode }), defaultOutputMode))
	return parseOutputMode(raw)
}

func resolveEnv(opts PythonOptions, chooser optionChooser, spec *pythonSpec) (map[string]string, error) {
	if chooser.changed("env") {
		return parseEnvEntries(opts.Env)
	}
	env := map[string]string{}
	if spec == nil || spec.Env == nil {
		return env, nil
	}
	for key, value := range spec.Env {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return nil, errors.New("env keys in --from-spec cannot be empty")
		}
		if strings.Contains(trimmedKey, "=") {
			return nil, fmt.Errorf("invalid env key %q in --from-spec: keys cannot include '='", key)
		}
		env[trimmedKey] = value
	}
	return env, nil
}

func resolveOverwrite(opts PythonOptions, chooser optionChooser, spec *pythonSpec) bool {
	if chooser.changed("overwrite") {
		return opts.Overwrite
	}
	if spec != nil && spec.Overwrite != nil {
		return *spec.Overwrite
	}
	return false
}

func resolveTargetDirs(scope, cwdAbs, homeAbs string) (string, string) {
	taskDir := config.ProjectTaskDir(cwdAbs)
	scriptDir := config.ProjectScriptDir(cwdAbs)
	switch scope {
	case "user":
		taskDir = config.UserTaskDir(homeAbs)
		scriptDir = config.UserScriptDir(homeAbs)
	case bundledScope:
		taskDir = config.ProjectBundledTaskDir(cwdAbs)
		scriptDir = config.ProjectBundledScriptDir(cwdAbs)
	}
	return taskDir, scriptDir
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
