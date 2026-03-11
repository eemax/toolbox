package runner

import (
	"context"
	"os"
	"testing"
	"time"

	"toolbox/internal/config"
	"toolbox/internal/manifest"
)

func BenchmarkExecuteDryRun(b *testing.B) {
	if _, err := os.Stat("/bin/echo"); err != nil {
		b.Skip("/bin/echo not available on this platform")
	}

	r := New(config.Config{
		LogLevel: "info",
		Output:   config.OutputConfig{CaptureLimitBytes: 1024},
		Execution: config.ExecutionConfig{
			DefaultTimeout: 60 * time.Second,
			RedactKeys:     []string{"TOKEN", "SECRET", "PASSWORD", "KEY"},
		},
	}, map[string]any{"log_level": "info"}, map[string]string{"API_TOKEN": "secret"})

	task := manifest.Task{
		Name:    "bench-dry-run",
		Command: "/bin/echo",
		Args:    []string{"{{config.log_level}}", "{{env.API_TOKEN}}"},
		Input:   manifest.InputSpec{Mode: manifest.InputModeNone},
		Output:  manifest.OutputSpec{Mode: manifest.OutputModeText},
		Env: map[string]string{
			"LOCAL_SECRET": "{{env.API_TOKEN}}",
		},
	}

	ctx := context.Background()
	cwd := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := r.Execute(ctx, ExecuteOptions{Task: task, DryRun: true, CWD: cwd})
		if err != nil {
			b.Fatalf("execute dry run: %v", err)
		}
		if result.DryRun == nil {
			b.Fatal("expected dry-run envelope")
		}
	}
}
