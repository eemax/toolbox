package runner

import (
	"context"
	"testing"
	"time"

	"toolbox/internal/config"
	"toolbox/internal/manifest"
)

func TestDryRunRedactsSensitiveEnv(t *testing.T) {
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
	if got := result.DryRun.Env["API_TOKEN"]; got != "<redacted>" {
		t.Fatalf("expected API_TOKEN redacted, got %q", got)
	}
	if got := result.DryRun.Env["MY_SECRET"]; got != "<redacted>" {
		t.Fatalf("expected MY_SECRET redacted, got %q", got)
	}
}
