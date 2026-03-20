package output

import (
	"bytes"
	"io"
	"testing"

	"toolbox/internal/config"
)

func TestConfigHumanDoesNotPanicOnWriteError(t *testing.T) {
	t.Parallel()
	loaded := config.LoadedConfig{
		Config: config.Config{LogLevel: "info"},
		Sources: config.Sources{
			Precedence: []string{"flags", "defaults"},
		},
	}

	// ConfigHuman silently ignores write errors, consistent with other *Human functions.
	ConfigHuman(failingWriter{}, loaded)
}

func TestConfigHumanWritesToBuffer(t *testing.T) {
	t.Parallel()
	loaded := config.LoadedConfig{
		Config: config.Config{LogLevel: "info"},
		Sources: config.Sources{
			Precedence: []string{"flags", "defaults"},
		},
	}
	buf := &bytes.Buffer{}
	ConfigHuman(buf, loaded)
	if buf.Len() == 0 {
		t.Fatal("expected non-empty output")
	}
}

type failingWriter struct{}

func (failingWriter) Write(_ []byte) (int, error) {
	return 0, io.ErrClosedPipe
}
