package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkLoadCatalog(b *testing.B) {
	base := b.TempDir()
	legacyDir := filepath.Join(base, ".toolbox", "tasks")
	bundledDir := filepath.Join(base, "tasks")
	userDir := filepath.Join(base, "home", ".config", "toolbox", "tasks")
	for _, dir := range []string{legacyDir, bundledDir, userDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			b.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	for i := 0; i < 120; i++ {
		manifestBody := fmt.Sprintf("name: legacy-%d\ncommand: /bin/echo\n", i)
		if err := os.WriteFile(filepath.Join(legacyDir, fmt.Sprintf("legacy-%d.yaml", i)), []byte(manifestBody), 0o644); err != nil {
			b.Fatalf("write legacy manifest %d: %v", i, err)
		}
	}
	for i := 0; i < 120; i++ {
		manifestBody := fmt.Sprintf("name: bundled-%d\ncommand: /bin/echo\n", i)
		if err := os.WriteFile(filepath.Join(bundledDir, fmt.Sprintf("bundled-%d.yaml", i)), []byte(manifestBody), 0o644); err != nil {
			b.Fatalf("write bundled manifest %d: %v", i, err)
		}
	}
	for i := 0; i < 60; i++ {
		manifestBody := fmt.Sprintf("name: user-%d\ncommand: /bin/echo\n", i)
		if err := os.WriteFile(filepath.Join(userDir, fmt.Sprintf("user-%d.yaml", i)), []byte(manifestBody), 0o644); err != nil {
			b.Fatalf("write user manifest %d: %v", i, err)
		}
	}

	opts := LoadOptions{Sources: []SourceDir{
		{Scope: "user", Category: "user", Dir: userDir},
		{Scope: "project", Category: "project-legacy", Dir: legacyDir},
		{Scope: "project", Category: "project-bundled", Dir: bundledDir},
	}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		catalog := Load(opts)
		if len(catalog.Errors) != 0 {
			b.Fatalf("unexpected parse errors: %v", catalog.Errors)
		}
	}
}
