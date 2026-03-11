package output

import (
	"errors"
	"io"
	"testing"

	"toolbox/internal/config"
)

func TestConfigHumanReturnsWriteError(t *testing.T) {
	t.Parallel()
	loaded := config.LoadedConfig{
		Config: config.Config{LogLevel: "info"},
		Sources: config.Sources{
			Precedence: []string{"flags", "defaults"},
		},
	}

	err := ConfigHuman(failingWriter{}, loaded)
	if err == nil {
		t.Fatalf("expected write error")
	}
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("expected io.ErrClosedPipe, got %v", err)
	}
}

type failingWriter struct{}

func (failingWriter) Write(_ []byte) (int, error) {
	return 0, io.ErrClosedPipe
}
