package unit_test

import (
	"testing"

	"toolbox/internal/runner"
)

func TestResolveTemplateSample(t *testing.T) {
	t.Parallel()
	vars := map[string]string{
		"input.file": "data.json",
	}

	resolved, err := runner.ResolveTemplate("--input={{input.file}}", vars)
	if err != nil {
		t.Fatalf("resolve template: %v", err)
	}
	if resolved != "--input=data.json" {
		t.Fatalf("unexpected template result: %q", resolved)
	}
}
