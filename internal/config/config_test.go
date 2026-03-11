package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	loaded, err := Load(LoadOptions{CWD: tempDir, HomeDir: homeDir, Env: map[string]string{}})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if loaded.Config.LogLevel != "info" {
		t.Fatalf("expected default log level info, got %q", loaded.Config.LogLevel)
	}
	if loaded.Config.Execution.DefaultTimeout.String() != "1m0s" {
		t.Fatalf("expected default timeout 1m0s, got %s", loaded.Config.Execution.DefaultTimeout)
	}
	if loaded.Config.Output.CaptureLimitBytes != 1024*1024 {
		t.Fatalf("expected default capture limit, got %d", loaded.Config.Output.CaptureLimitBytes)
	}
}

func TestLoadPrecedence(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	if err := os.MkdirAll(filepath.Join(homeDir, ".config", "toolbox"), 0o755); err != nil {
		t.Fatalf("mkdir user config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeDir, ".config", "toolbox", "config.yaml"), []byte("log_level: warn\noutput:\n  capture_limit_bytes: 100\n"), 0o644); err != nil {
		t.Fatalf("write user config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tempDir, ".toolbox"), 0o755); err != nil {
		t.Fatalf("mkdir project config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, ".toolbox", "config.yaml"), []byte("log_level: error\nexecution:\n  default_timeout: 20s\n"), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	env := map[string]string{
		"TOOLBOX_LOG_LEVEL":                  "debug",
		"TOOLBOX_OUTPUT_CAPTURE_LIMIT_BYTES": "2048",
	}

	loaded, err := Load(LoadOptions{
		CWD:           tempDir,
		HomeDir:       homeDir,
		Env:           env,
		FlagOverrides: map[string]any{"log_level": "info"},
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if loaded.Config.LogLevel != "info" {
		t.Fatalf("expected flag log level info, got %q", loaded.Config.LogLevel)
	}
	if loaded.Config.Output.CaptureLimitBytes != 2048 {
		t.Fatalf("expected env capture limit 2048, got %d", loaded.Config.Output.CaptureLimitBytes)
	}
	if loaded.Config.Execution.DefaultTimeout.String() != "20s" {
		t.Fatalf("expected project timeout 20s, got %s", loaded.Config.Execution.DefaultTimeout)
	}
	if loaded.Sources.UserConfig == "" || loaded.Sources.ProjectConfig == "" {
		t.Fatalf("expected both user and project sources to be set")
	}
}
