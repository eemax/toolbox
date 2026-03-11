package runner

import "testing"

func BenchmarkResolveTemplate(b *testing.B) {
	value := "--input={{input.file}} --log={{config.log_level}} --token={{env.API_TOKEN}}"
	vars := map[string]string{
		"input.file":       "payload.json",
		"config.log_level": "info",
		"env.API_TOKEN":    "secret-token",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolved, err := ResolveTemplate(value, vars)
		if err != nil {
			b.Fatalf("resolve template: %v", err)
		}
		if len(resolved) == 0 {
			b.Fatal("expected non-empty resolved template")
		}
	}
}

func BenchmarkResolveEnvTemplates(b *testing.B) {
	vars := map[string]string{
		"input.file":       "payload.json",
		"config.log_level": "info",
		"env.API_TOKEN":    "secret-token",
	}
	env := map[string]string{
		"INPUT_PATH":    "{{input.file}}",
		"LOG_LEVEL":     "{{config.log_level}}",
		"SERVICE_TOKEN": "{{env.API_TOKEN}}",
		"STATIC":        "static",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolved, err := resolveEnvTemplates(env, vars)
		if err != nil {
			b.Fatalf("resolve env templates: %v", err)
		}
		if len(resolved) != len(env) {
			b.Fatalf("unexpected env length: got %d want %d", len(resolved), len(env))
		}
	}
}
