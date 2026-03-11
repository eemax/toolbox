package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogTaskSourcesOrder(t *testing.T) {
	t.Parallel()
	cwd := "/tmp/project"
	home := "/tmp/home"
	sources := CatalogTaskSources(cwd, home)
	if len(sources) != 3 {
		t.Fatalf("expected three task sources, got %d", len(sources))
	}
	if sources[0].Category != SourceCategoryUser {
		t.Fatalf("expected user source first, got %q", sources[0].Category)
	}
	if sources[1].Category != SourceCategoryProjectLegacy {
		t.Fatalf("expected legacy project source second, got %q", sources[1].Category)
	}
	if sources[2].Category != SourceCategoryProjectBundled {
		t.Fatalf("expected bundled project source third, got %q", sources[2].Category)
	}
}

func TestLegacyTaskLayoutOnly(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	legacyDir := filepath.Join(root, ".toolbox", "tasks")
	bundledDir := filepath.Join(root, "tasks")

	if LegacyTaskLayoutOnly(root) {
		t.Fatalf("expected false when no task dirs exist")
	}
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatalf("mkdir legacy dir: %v", err)
	}
	if !LegacyTaskLayoutOnly(root) {
		t.Fatalf("expected true for legacy-only layout")
	}
	if err := os.MkdirAll(bundledDir, 0o755); err != nil {
		t.Fatalf("mkdir bundled dir: %v", err)
	}
	if LegacyTaskLayoutOnly(root) {
		t.Fatalf("expected false once bundled layout exists")
	}
}
