package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"toolbox/internal/config"
	"toolbox/internal/doctor"
	"toolbox/internal/manifest"
	"toolbox/pkg/contract"
)

func TestJSONWritesValidIndentedPayload(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	payload := map[string]any{"hello": "world", "num": 3}
	if err := JSON(buf, payload); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}
	if decoded["hello"] != "world" {
		t.Fatalf("unexpected JSON payload: %v", decoded)
	}
}

func TestListHumanFormatsTasksAndEmptyState(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	ListHuman(buf, nil)
	if got := buf.String(); !strings.Contains(got, "No tasks found.") {
		t.Fatalf("unexpected empty-state output: %q", got)
	}

	buf.Reset()
	ListHuman(buf, []manifest.Task{
		{Name: "task-a", Description: "Task A", Input: manifest.InputSpec{Mode: manifest.InputModeJSON}, Source: manifest.SourceInfo{Path: "/tmp/a.yaml"}},
		{Name: "task-b", Description: "", Input: manifest.InputSpec{Mode: manifest.InputModeNone}, Source: manifest.SourceInfo{Path: "/tmp/b.yaml"}},
	})
	out := buf.String()
	if !strings.Contains(out, "task-a") || !strings.Contains(out, "input_mode: json") {
		t.Fatalf("expected task-a details, got: %s", out)
	}
	if !strings.Contains(out, "description: (none)") {
		t.Fatalf("expected fallback description, got: %s", out)
	}
}

func TestDryRunHumanSortsEnvKeys(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	DryRunHuman(buf, contract.DryRunEnvelope{
		Task:    "dry",
		Command: "/bin/echo",
		Args:    []string{"a", "b"},
		Cwd:     "/tmp",
		Timeout: "1m",
		Env: map[string]string{
			"Z": "last",
			"A": "first",
		},
	})
	out := buf.String()
	if strings.Index(out, "A=first") > strings.Index(out, "Z=last") {
		t.Fatalf("expected sorted env keys in dry run output: %s", out)
	}
}

func TestRunHumanIncludesStreamsAndTruncationMetadata(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	RunHuman(buf, contract.RunEnvelope{
		Task:            "run-task",
		OK:              false,
		ExitCode:        3,
		DurationMS:      12,
		StartedAt:       time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC),
		Stdout:          "out-line",
		Stderr:          "err-line",
		StdoutTruncated: true,
		StderrTruncated: false,
	})
	out := buf.String()
	for _, expected := range []string{
		"Task: run-task",
		"Status: failed",
		"Exit Code: 3",
		"Stdout:",
		"out-line",
		"Stderr:",
		"err-line",
		"Truncated: stdout=true stderr=false",
	} {
		if !strings.Contains(out, expected) {
			t.Fatalf("missing %q in output: %s", expected, out)
		}
	}
}

func TestDoctorHumanFormatsSuccessAndIssues(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	DoctorHuman(buf, doctor.Report{})
	if got := buf.String(); !strings.Contains(got, "doctor: ok") {
		t.Fatalf("unexpected doctor success output: %s", got)
	}

	buf.Reset()
	DoctorHuman(buf, doctor.Report{Issues: []doctor.Issue{{Level: "error", Check: "runtime", Message: "missing", Hint: "install"}}})
	out := buf.String()
	if !strings.Contains(out, "[ERROR] runtime: missing") || !strings.Contains(out, "hint: install") {
		t.Fatalf("unexpected doctor issue output: %s", out)
	}
}

func TestConfigHumanFormatsSourcesAndResolvedConfig(t *testing.T) {
	t.Parallel()
	loaded := config.LoadedConfig{
		Config: config.Config{LogLevel: "debug", Output: config.OutputConfig{CaptureLimitBytes: 10}},
		Sources: config.Sources{
			Precedence:     []string{"flags", "defaults"},
			UserConfig:     "/home/user/.config/toolbox/config.yaml",
			ProjectConfig:  "/repo/.toolbox/config.yaml",
			ExplicitConfig: "/tmp/custom.yaml",
			EnvOverrides:   []string{"TOOLBOX_LOG_LEVEL"},
			FlagOverrides:  []string{"log_level"},
		},
	}
	buf := &bytes.Buffer{}
	if err := ConfigHuman(buf, loaded); err != nil {
		t.Fatalf("ConfigHuman: %v", err)
	}
	out := buf.String()
	for _, expected := range []string{
		"Config precedence (highest to lowest):",
		"- flags",
		"- user: /home/user/.config/toolbox/config.yaml",
		"- project: /repo/.toolbox/config.yaml",
		"- explicit: /tmp/custom.yaml",
		"- env overrides: TOOLBOX_LOG_LEVEL",
		"- flag overrides: log_level",
		"\"log_level\": \"debug\"",
	} {
		if !strings.Contains(out, expected) {
			t.Fatalf("missing %q in output: %s", expected, out)
		}
	}
}
