package runner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"toolbox/internal/config"
	"toolbox/internal/manifest"
)

func TestDryRunUsesTaskEnvDeltaByDefault(t *testing.T) {
	t.Parallel()
	r := New(config.Config{
		LogLevel: "info",
		Output:   config.OutputConfig{CaptureLimitBytes: 1024},
		Execution: config.ExecutionConfig{
			DefaultTimeout: 60 * time.Second,
			RedactKeys:     []string{"TOKEN", "SECRET", "PASSWORD", "KEY"},
		},
	}, map[string]any{"log_level": "info"}, map[string]string{"API_TOKEN": "super-secret"})

	result, err := r.Execute(context.Background(), ExecuteOptions{
		Task: manifest.Task{
			Name:    "hello",
			Command: "/bin/echo",
			Args:    []string{"ok"},
			Input:   manifest.InputSpec{Mode: manifest.InputModeNone},
			Output:  manifest.OutputSpec{Mode: manifest.OutputModeText},
			Env: map[string]string{
				"MY_SECRET": "x",
			},
		},
		DryRun: true,
		CWD:    "/tmp",
	})
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}
	if result.DryRun == nil {
		t.Fatalf("expected dry run envelope")
	}
	if _, exists := result.DryRun.Env["API_TOKEN"]; exists {
		t.Fatalf("expected base environment to be excluded from dry-run env")
	}
	if got := result.DryRun.Env["MY_SECRET"]; got != "<redacted>" {
		t.Fatalf("expected MY_SECRET redacted, got %q", got)
	}
}

func TestDryRunFullEnvIncludesBaseEnv(t *testing.T) {
	t.Parallel()
	r := New(config.Config{
		LogLevel: "info",
		Output:   config.OutputConfig{CaptureLimitBytes: 1024},
		Execution: config.ExecutionConfig{
			DefaultTimeout: 60 * time.Second,
			RedactKeys:     []string{"TOKEN", "SECRET", "PASSWORD", "KEY"},
		},
	}, map[string]any{"log_level": "info"}, map[string]string{"API_TOKEN": "super-secret"})

	result, err := r.Execute(context.Background(), ExecuteOptions{
		Task: manifest.Task{
			Name:    "hello",
			Command: "/bin/echo",
			Args:    []string{"ok"},
			Input:   manifest.InputSpec{Mode: manifest.InputModeNone},
			Output:  manifest.OutputSpec{Mode: manifest.OutputModeText},
			Env: map[string]string{
				"MY_SECRET": "x",
			},
		},
		DryRun:        true,
		DryRunFullEnv: true,
		CWD:           "/tmp",
	})
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}
	if result.DryRun == nil {
		t.Fatalf("expected dry run envelope")
	}
	if got := result.DryRun.Env["API_TOKEN"]; got != "<redacted>" {
		t.Fatalf("expected API_TOKEN redacted, got %q", got)
	}
	if got := result.DryRun.Env["MY_SECRET"]; got != "<redacted>" {
		t.Fatalf("expected MY_SECRET redacted, got %q", got)
	}
}

func TestExecutePreflightErrorIncludesEnvelopeStderr(t *testing.T) {
	t.Parallel()
	r := New(config.Config{
		LogLevel: "info",
		Output:   config.OutputConfig{CaptureLimitBytes: 1024},
		Execution: config.ExecutionConfig{
			DefaultTimeout: 60 * time.Second,
			RedactKeys:     []string{"TOKEN", "SECRET", "PASSWORD", "KEY"},
		},
	}, map[string]any{"log_level": "info"}, map[string]string{})

	result, err := r.Execute(context.Background(), ExecuteOptions{
		Task: manifest.Task{
			Name:    "missing",
			Command: "toolbox-definitely-missing-cmd",
			Input:   manifest.InputSpec{Mode: manifest.InputModeNone},
			Output:  manifest.OutputSpec{Mode: manifest.OutputModeText},
		},
		CWD: "/tmp",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if result.Envelope.ExitCode != 127 {
		t.Fatalf("expected exit code 127, got %d", result.Envelope.ExitCode)
	}
	if result.Envelope.Stderr == "" {
		t.Fatalf("expected envelope stderr to contain failure reason")
	}
}

func TestExecuteResolvesRelativeCommandFromTaskCWD(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixture is POSIX-only")
	}

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "hello.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/usr/bin/env sh\necho from-script\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	r := New(config.Config{
		LogLevel: "info",
		Output:   config.OutputConfig{CaptureLimitBytes: 1024},
		Execution: config.ExecutionConfig{
			DefaultTimeout: 60 * time.Second,
			RedactKeys:     []string{"TOKEN", "SECRET", "PASSWORD", "KEY"},
		},
	}, map[string]any{"log_level": "info"}, map[string]string{})

	result, err := r.Execute(context.Background(), ExecuteOptions{
		Task: manifest.Task{
			Name:    "relative",
			Command: "./hello.sh",
			Input:   manifest.InputSpec{Mode: manifest.InputModeNone},
			Output:  manifest.OutputSpec{Mode: manifest.OutputModeText},
			CWD:     dir,
		},
		CWD: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("execute relative command: %v", err)
	}
	if !result.Envelope.OK {
		t.Fatalf("expected success envelope, got failure: %+v", result.Envelope)
	}
	if result.Envelope.Stdout != "from-script\n" {
		t.Fatalf("expected script output, got %q", result.Envelope.Stdout)
	}
}
