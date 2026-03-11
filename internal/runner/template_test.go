package runner

import "testing"

func TestResolveTemplate(t *testing.T) {
	t.Parallel()
	vars := map[string]string{"input.file": "payload.json", "config.log_level": "info"}
	value, err := ResolveTemplate("--input={{input.file}} --log={{config.log_level}}", vars)
	if err != nil {
		t.Fatalf("resolve template: %v", err)
	}
	expected := "--input=payload.json --log=info"
	if value != expected {
		t.Fatalf("expected %q, got %q", expected, value)
	}
}

func TestResolveTemplateUnknownVariable(t *testing.T) {
	t.Parallel()
	_, err := ResolveTemplate("{{missing.key}}", map[string]string{})
	if err == nil {
		t.Fatalf("expected unknown variable error")
	}
}
