package integration_test

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestVersionCommandSample(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	cmd := exec.Command("go", "run", "./cmd/toolbox", "version")
	cmd.Dir = repoRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run version command: %v\noutput: %s", err, string(output))
	}

	if strings.TrimSpace(string(output)) == "" {
		t.Fatalf("expected non-empty version output")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to resolve caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
