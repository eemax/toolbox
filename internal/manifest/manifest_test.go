package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadParsesValidManifest(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	file := filepath.Join(projectDir, "hello.yaml")
	content := []byte(`name: hello
command: /bin/echo
args:
  - hi
input:
  mode: none
output:
  mode: text
timeout: 15s
`)
	if err := os.WriteFile(file, content, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	catalog := Load(LoadOptions{ProjectDir: projectDir})
	if err := catalog.FatalError(); err != nil {
		t.Fatalf("expected valid catalog, got error: %v", err)
	}
	task, ok := catalog.Tasks["hello"]
	if !ok {
		t.Fatalf("task not found")
	}
	if task.Timeout.String() != "15s" {
		t.Fatalf("expected timeout 15s, got %s", task.Timeout)
	}
}

func TestLoadInvalidMode(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	file := filepath.Join(projectDir, "bad.yaml")
	content := []byte(`name: bad
command: /bin/echo
input:
  mode: weird
`)
	if err := os.WriteFile(file, content, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	catalog := Load(LoadOptions{ProjectDir: projectDir})
	if len(catalog.Errors) == 0 {
		t.Fatalf("expected parse/validation errors")
	}
}

func TestLoadDuplicateNames(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	userDir := filepath.Join(tempDir, "user")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatalf("mkdir user dir: %v", err)
	}
	content := []byte("name: same\ncommand: /bin/echo\n")
	if err := os.WriteFile(filepath.Join(projectDir, "a.yaml"), content, 0o644); err != nil {
		t.Fatalf("write project manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "b.yaml"), content, 0o644); err != nil {
		t.Fatalf("write user manifest: %v", err)
	}

	catalog := Load(LoadOptions{ProjectDir: projectDir, UserDir: userDir})
	if len(catalog.DuplicateNames) != 1 {
		t.Fatalf("expected one duplicate, got %d", len(catalog.DuplicateNames))
	}
	if _, ok := catalog.Tasks["same"]; ok {
		t.Fatalf("duplicate task should not be runnable")
	}
}

func TestLoadFromMultipleProjectDirs(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	projectPrimary := filepath.Join(tempDir, "project-primary")
	projectBundled := filepath.Join(tempDir, "project-bundled")
	if err := os.MkdirAll(projectPrimary, 0o755); err != nil {
		t.Fatalf("mkdir primary project dir: %v", err)
	}
	if err := os.MkdirAll(projectBundled, 0o755); err != nil {
		t.Fatalf("mkdir bundled project dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectPrimary, "legacy.yaml"), []byte("name: legacy\ncommand: /bin/echo\n"), 0o644); err != nil {
		t.Fatalf("write primary manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectBundled, "portable.yaml"), []byte("name: portable\ncommand: /bin/echo\n"), 0o644); err != nil {
		t.Fatalf("write bundled manifest: %v", err)
	}

	catalog := Load(LoadOptions{
		ProjectDirs: []string{projectPrimary, projectBundled},
	})
	if err := catalog.FatalError(); err != nil {
		t.Fatalf("expected valid catalog, got error: %v", err)
	}
	if _, ok := catalog.Tasks["legacy"]; !ok {
		t.Fatalf("expected legacy task from primary project dir")
	}
	if _, ok := catalog.Tasks["portable"]; !ok {
		t.Fatalf("expected portable task from bundled project dir")
	}
}
