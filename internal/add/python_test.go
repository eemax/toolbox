package add

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"toolbox/internal/config"
	"toolbox/internal/manifest"
)

func TestLoadSpecRejectsUnknownFieldsYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(`api_version: toolbox.add.python/v1
name: sample
script: ./tools/sample.py
unknown_field: true
`), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	if _, err := loadSpec(specPath); err == nil {
		t.Fatalf("expected strict parsing error")
	}
}

func TestLoadSpecRejectsUnknownFieldsJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.json")
	if err := os.WriteFile(specPath, []byte(`{"api_version":"toolbox.add.python/v1","name":"sample","script":"./tools/sample.py","unknown_field":true}`), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	if _, err := loadSpec(specPath); err == nil {
		t.Fatalf("expected strict parsing error")
	}
}

func TestResolveOptionsFlagsOverrideSpec(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	home := filepath.Join(cwd, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	specScript := filepath.Join(cwd, "spec.py")
	cliScript := filepath.Join(cwd, "cli.py")
	if err := os.WriteFile(specScript, []byte("print('spec')\n"), 0o644); err != nil {
		t.Fatalf("write spec script: %v", err)
	}
	if err := os.WriteFile(cliScript, []byte("print('cli')\n"), 0o644); err != nil {
		t.Fatalf("write cli script: %v", err)
	}

	specPath := filepath.Join(cwd, "addspec.yaml")
	if err := os.WriteFile(specPath, []byte(`api_version: toolbox.add.python/v1
name: spec-name
script: ./spec.py
description: spec-description
python_bin: spec-python
args:
  - --from-spec
env:
  FOO: spec
timeout: 12s
cwd: ./spec-cwd
tags:
  - spec
input_mode: file
output_mode: json
scope: user
overwrite: true
`), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	adder := pythonAdder{}
	cfg, err := adder.resolveOptions(PythonOptions{
		CWD:         cwd,
		HomeDir:     home,
		FromSpec:    specPath,
		Name:        "cli-name",
		Script:      "./cli.py",
		Description: "cli-description",
		Args:        []string{"--from-cli"},
		Env:         []string{"FOO=cli"},
		Timeout:     "45s",
		TaskCWD:     "./cli-cwd",
		Tags:        []string{"cli"},
		InputMode:   "json",
		OutputMode:  "text",
		PythonBin:   "cli-python",
		Scope:       "project",
		Overwrite:   false,
		Changed: changed(
			"name", "script", "description", "arg", "env", "timeout", "cwd",
			"tag", "input-mode", "output-mode", "python-bin", "scope", "overwrite",
		),
	})
	if err != nil {
		t.Fatalf("resolve options: %v", err)
	}

	if cfg.Name != "cli-name" {
		t.Fatalf("name mismatch: %s", cfg.Name)
	}
	if cfg.Description != "cli-description" {
		t.Fatalf("description mismatch: %s", cfg.Description)
	}
	if cfg.PythonBin != "cli-python" {
		t.Fatalf("python mismatch: %s", cfg.PythonBin)
	}
	if cfg.Scope != "project" {
		t.Fatalf("scope mismatch: %s", cfg.Scope)
	}
	if cfg.InputMode != manifest.InputModeJSON {
		t.Fatalf("input mode mismatch: %s", cfg.InputMode)
	}
	if cfg.OutputMode != manifest.OutputModeText {
		t.Fatalf("output mode mismatch: %s", cfg.OutputMode)
	}
	if cfg.Timeout != "45s" {
		t.Fatalf("timeout mismatch: %s", cfg.Timeout)
	}
	if cfg.TaskCWD != "./cli-cwd" {
		t.Fatalf("cwd mismatch: %s", cfg.TaskCWD)
	}
	if cfg.Overwrite {
		t.Fatalf("expected overwrite=false from changed flag")
	}
	if len(cfg.Args) != 1 || cfg.Args[0] != "--from-cli" {
		t.Fatalf("args mismatch: %#v", cfg.Args)
	}
	if len(cfg.Tags) != 1 || cfg.Tags[0] != "cli" {
		t.Fatalf("tags mismatch: %#v", cfg.Tags)
	}
	if len(cfg.Env) != 1 || cfg.Env["FOO"] != "cli" {
		t.Fatalf("env mismatch: %#v", cfg.Env)
	}
	if cfg.SourceScript != filepath.Join(cwd, "cli.py") {
		t.Fatalf("script mismatch: %s", cfg.SourceScript)
	}
	if cfg.TaskDir != config.ProjectTaskDir(cwd) {
		t.Fatalf("task dir mismatch: %s", cfg.TaskDir)
	}
	if cfg.ScriptDir != config.ProjectScriptDir(cwd) {
		t.Fatalf("script dir mismatch: %s", cfg.ScriptDir)
	}
}

func TestAddPythonCreatesFiles(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	home := filepath.Join(cwd, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	source := filepath.Join(cwd, "source.py")
	if err := os.WriteFile(source, []byte("print('hello')\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	adder := pythonAdder{
		lookPath:    func(string) (string, error) { return "/usr/bin/python3", nil },
		compile:     func(string, string) (string, error) { return "", nil },
		loadCatalog: manifest.Load,
	}

	result, err := adder.add(PythonOptions{
		CWD:     cwd,
		HomeDir: home,
		Name:    "hello-task",
		Script:  "./source.py",
		Changed: changed("name", "script"),
	})
	if err != nil {
		t.Fatalf("add python: %v", err)
	}

	if result.Status != "created" {
		t.Fatalf("status mismatch: %s", result.Status)
	}
	if result.Task != "hello-task" {
		t.Fatalf("task mismatch: %s", result.Task)
	}
	if result.Scope != "project" {
		t.Fatalf("scope mismatch: %s", result.Scope)
	}
	if result.Overwritten {
		t.Fatalf("expected overwritten=false")
	}
	if len(result.Checks) != 3 {
		t.Fatalf("checks mismatch: %#v", result.Checks)
	}

	scriptData, err := os.ReadFile(result.ScriptPath)
	if err != nil {
		t.Fatalf("read copied script: %v", err)
	}
	if string(scriptData) != "print('hello')\n" {
		t.Fatalf("copied script mismatch: %q", string(scriptData))
	}

	manifestData, err := os.ReadFile(result.ManifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	content := string(manifestData)
	if !strings.Contains(content, "name: hello-task\n") {
		t.Fatalf("manifest missing task name:\n%s", content)
	}
	if !strings.Contains(content, "command: python3\n") {
		t.Fatalf("manifest missing command:\n%s", content)
	}
	if !strings.Contains(content, "requires:\n    - python3\n") {
		t.Fatalf("manifest missing requires:\n%s", content)
	}
}

func TestResolveOptionsBundledScopeWritesPortablePaths(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	home := filepath.Join(cwd, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	source := filepath.Join(cwd, "portable.py")
	if err := os.WriteFile(source, []byte("print('portable')\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	adder := pythonAdder{}
	cfg, err := adder.resolveOptions(PythonOptions{
		CWD:     cwd,
		HomeDir: home,
		Name:    "portable-task",
		Script:  "./portable.py",
		Scope:   "bundled",
		Changed: changed("name", "script", "scope"),
	})
	if err != nil {
		t.Fatalf("resolve options: %v", err)
	}
	if cfg.Scope != "bundled" {
		t.Fatalf("scope mismatch: %s", cfg.Scope)
	}
	if cfg.TaskDir != config.ProjectBundledTaskDir(cwd) {
		t.Fatalf("task dir mismatch: %s", cfg.TaskDir)
	}
	if cfg.ScriptDir != config.ProjectBundledScriptDir(cwd) {
		t.Fatalf("script dir mismatch: %s", cfg.ScriptDir)
	}
}

func TestAddPythonRejectsInvalidEnv(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	home := filepath.Join(cwd, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	source := filepath.Join(cwd, "source.py")
	if err := os.WriteFile(source, []byte("print('hello')\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	adder := pythonAdder{
		lookPath:    func(string) (string, error) { return "/usr/bin/python3", nil },
		compile:     func(string, string) (string, error) { return "", nil },
		loadCatalog: manifest.Load,
	}
	_, err := adder.add(PythonOptions{
		CWD:     cwd,
		HomeDir: home,
		Name:    "invalid-env",
		Script:  "./source.py",
		Env:     []string{"NOT_VALID"},
		Changed: changed("name", "script", "env"),
	})
	if err == nil {
		t.Fatalf("expected env parsing error")
	}
}

func TestAddPythonDuplicateWithoutOverwrite(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	home := filepath.Join(cwd, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	sourceA := filepath.Join(cwd, "a.py")
	sourceB := filepath.Join(cwd, "b.py")
	if err := os.WriteFile(sourceA, []byte("print('a')\n"), 0o644); err != nil {
		t.Fatalf("write sourceA: %v", err)
	}
	if err := os.WriteFile(sourceB, []byte("print('b')\n"), 0o644); err != nil {
		t.Fatalf("write sourceB: %v", err)
	}

	adder := pythonAdder{
		lookPath:    func(string) (string, error) { return "/usr/bin/python3", nil },
		compile:     func(string, string) (string, error) { return "", nil },
		loadCatalog: manifest.Load,
	}
	if _, err := adder.add(PythonOptions{
		CWD:     cwd,
		HomeDir: home,
		Name:    "dup-task",
		Script:  "./a.py",
		Changed: changed("name", "script"),
	}); err != nil {
		t.Fatalf("first add failed: %v", err)
	}

	_, err := adder.add(PythonOptions{
		CWD:     cwd,
		HomeDir: home,
		Name:    "dup-task",
		Script:  "./b.py",
		Changed: changed("name", "script"),
	})
	if err == nil {
		t.Fatalf("expected duplicate error")
	}
	if !strings.Contains(err.Error(), "--overwrite") {
		t.Fatalf("expected overwrite hint, got: %v", err)
	}
}

func TestAddPythonOverwriteReplacesFiles(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	home := filepath.Join(cwd, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	sourceA := filepath.Join(cwd, "a.py")
	sourceB := filepath.Join(cwd, "b.py")
	if err := os.WriteFile(sourceA, []byte("print('a')\n"), 0o644); err != nil {
		t.Fatalf("write sourceA: %v", err)
	}
	if err := os.WriteFile(sourceB, []byte("print('b')\n"), 0o644); err != nil {
		t.Fatalf("write sourceB: %v", err)
	}

	adder := pythonAdder{
		lookPath:    func(string) (string, error) { return "/usr/bin/python3", nil },
		compile:     func(string, string) (string, error) { return "", nil },
		loadCatalog: manifest.Load,
	}
	if _, err := adder.add(PythonOptions{
		CWD:     cwd,
		HomeDir: home,
		Name:    "overwrite-task",
		Script:  "./a.py",
		Changed: changed("name", "script"),
	}); err != nil {
		t.Fatalf("first add failed: %v", err)
	}

	result, err := adder.add(PythonOptions{
		CWD:       cwd,
		HomeDir:   home,
		Name:      "overwrite-task",
		Script:    "./b.py",
		Overwrite: true,
		Changed:   changed("name", "script", "overwrite"),
	})
	if err != nil {
		t.Fatalf("overwrite add failed: %v", err)
	}
	if !result.Overwritten {
		t.Fatalf("expected overwritten=true")
	}
	data, err := os.ReadFile(result.ScriptPath)
	if err != nil {
		t.Fatalf("read copied script: %v", err)
	}
	if string(data) != "print('b')\n" {
		t.Fatalf("script did not update: %q", string(data))
	}
}

func TestAddPythonCompileFailureLeavesNoFiles(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	home := filepath.Join(cwd, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	source := filepath.Join(cwd, "bad.py")
	if err := os.WriteFile(source, []byte("print('oops'\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	adder := pythonAdder{
		lookPath: func(string) (string, error) { return "/usr/bin/python3", nil },
		compile: func(string, string) (string, error) {
			return "syntax error", errors.New("compile failed")
		},
		loadCatalog: manifest.Load,
	}
	_, err := adder.add(PythonOptions{
		CWD:     cwd,
		HomeDir: home,
		Name:    "bad-task",
		Script:  "./bad.py",
		Changed: changed("name", "script"),
	})
	if err == nil {
		t.Fatalf("expected compile error")
	}
	if !strings.Contains(err.Error(), "py_compile") {
		t.Fatalf("unexpected compile error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cwd, ".toolbox", "tasks", "bad-task.yaml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("manifest should not exist after failure")
	}
	if _, err := os.Stat(filepath.Join(cwd, ".toolbox", "scripts", "bad-task.py")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("script should not exist after failure")
	}
}

func changed(names ...string) map[string]bool {
	result := map[string]bool{}
	for _, name := range names {
		result[name] = true
	}
	return result
}
