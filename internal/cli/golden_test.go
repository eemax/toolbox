package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestGoldenOutputs(t *testing.T) {
	repoRoot := projectRoot(t)
	cwd := t.TempDir()
	home := filepath.Join(cwd, "home")
	if err := os.MkdirAll(filepath.Join(cwd, ".toolbox", "tasks"), 0o755); err != nil {
		t.Fatalf("mkdir project tasks: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cwd, ".toolbox"), 0o755); err != nil {
		t.Fatalf("mkdir project config dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".config", "toolbox"), 0o755); err != nil {
		t.Fatalf("mkdir user config dir: %v", err)
	}

	manifest := `name: hello
description: Print hello
command: /bin/echo
args:
  - hello
input:
  mode: none
output:
  mode: text
`
	if err := os.WriteFile(filepath.Join(cwd, ".toolbox", "tasks", "hello.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cwd, ".toolbox", "config.yaml"), []byte("log_level: info\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	cases := []struct {
		name     string
		args     []string
		exitCode int
	}{
		{name: "list_human", args: []string{"list"}, exitCode: 0},
		{name: "list_json", args: []string{"list", "--json"}, exitCode: 0},
		{name: "run_human", args: []string{"run", "hello"}, exitCode: 0},
		{name: "run_json", args: []string{"run", "hello", "--json"}, exitCode: 0},
		{name: "doctor_human", args: []string{"doctor"}, exitCode: 0},
		{name: "doctor_json", args: []string{"doctor", "--json"}, exitCode: 0},
		{name: "config_show_human", args: []string{"config", "show"}, exitCode: 0},
		{name: "config_show_json", args: []string{"config", "show", "--json"}, exitCode: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			env := map[string]string{
				"HOME": home,
				"PATH": os.Getenv("PATH"),
			}
			app := New("test", stdout, stderr, env)
			root := app.rootCommand()
			root.SetArgs(tc.args)
			err := root.Execute()
			actualExit := 0
			if err != nil {
				var exitErr *ExitError
				if errors.As(err, &exitErr) {
					actualExit = exitErr.Code
				} else {
					t.Fatalf("unexpected command error: %v", err)
				}
			}
			if actualExit != tc.exitCode {
				t.Fatalf("expected exit %d got %d (stderr=%s)", tc.exitCode, actualExit, stderr.String())
			}
			output := normalizeOutput(tc.name, stdout.String(), cwd)
			golden := readGolden(t, repoRoot, tc.name)
			if output != golden {
				t.Fatalf("golden mismatch for %s\n--- actual ---\n%s\n--- expected ---\n%s", tc.name, output, golden)
			}
		})
	}
}

func TestGoldenAddOutputs(t *testing.T) {
	repoRoot := projectRoot(t)
	cwd := t.TempDir()
	home := filepath.Join(cwd, "home")
	if err := os.MkdirAll(filepath.Join(home, ".config", "toolbox"), 0o755); err != nil {
		t.Fatalf("mkdir user config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cwd, "source.py"), []byte("print('golden')\n"), 0o644); err != nil {
		t.Fatalf("write source script: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	cases := []struct {
		name     string
		args     []string
		exitCode int
	}{
		{
			name:     "add_python_human",
			args:     []string{"add", "python", "--name", "py-golden", "--script", "./source.py", "--python-bin", "/bin/echo"},
			exitCode: 0,
		},
		{
			name:     "add_python_json",
			args:     []string{"add", "python", "--name", "py-golden-json", "--script", "./source.py", "--python-bin", "/bin/echo", "--json"},
			exitCode: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			env := map[string]string{
				"HOME": home,
				"PATH": os.Getenv("PATH"),
			}
			app := New("test", stdout, stderr, env)
			root := app.rootCommand()
			root.SetArgs(tc.args)
			err := root.Execute()
			actualExit := 0
			if err != nil {
				var exitErr *ExitError
				if errors.As(err, &exitErr) {
					actualExit = exitErr.Code
				} else {
					t.Fatalf("unexpected command error: %v", err)
				}
			}
			if actualExit != tc.exitCode {
				t.Fatalf("expected exit %d got %d (stderr=%s)", tc.exitCode, actualExit, stderr.String())
			}
			output := normalizeOutput(tc.name, stdout.String(), cwd)
			golden := readGolden(t, repoRoot, tc.name)
			if output != golden {
				t.Fatalf("golden mismatch for %s\n--- actual ---\n%s\n--- expected ---\n%s", tc.name, output, golden)
			}
		})
	}
}

func normalizeOutput(name, value, cwd string) string {
	value = strings.ReplaceAll(value, "/private"+cwd, "<TMP>")
	value = strings.ReplaceAll(value, cwd, "<TMP>")
	if strings.Contains(name, "run_human") {
		durationRe := regexp.MustCompile(`Duration: [0-9]+ms`)
		value = durationRe.ReplaceAllString(value, "Duration: <ms>")
		startedRe := regexp.MustCompile(`Started At: .*`)
		value = startedRe.ReplaceAllString(value, "Started At: <ts>")
	}
	if strings.Contains(name, "run_json") {
		durationRe := regexp.MustCompile(`"duration_ms":\s*[0-9]+`)
		value = durationRe.ReplaceAllString(value, `"duration_ms": 0`)
		startedRe := regexp.MustCompile(`"started_at":\s*"[^"]+"`)
		value = startedRe.ReplaceAllString(value, `"started_at": "<ts>"`)
	}
	return value
}

func readGolden(t *testing.T, repoRoot, name string) string {
	t.Helper()
	path := filepath.Join(repoRoot, "testdata", "golden", name+".golden")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", path, err)
	}
	return string(data)
}

func projectRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to resolve caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
