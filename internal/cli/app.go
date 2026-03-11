package cli

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// ExitError allows commands to control process exit codes.
type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("exit code %d", e.Code)
	}
	return e.Message
}

// App wires the CLI runtime.
type App struct {
	version string
	stdout  io.Writer
	stderr  io.Writer
	env     map[string]string
	logger  *slog.Logger
}

// New creates a CLI app instance.
func New(version string, stdout, stderr io.Writer, env map[string]string) *App {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	if env == nil {
		env = envToMap(os.Environ())
	}
	app := &App{version: version, stdout: stdout, stderr: stderr, env: env}
	app.configureLogger("info")
	return app
}

// Execute runs the CLI and returns the process exit code.
func (a *App) Execute() int {
	root := a.rootCommand()
	if err := root.Execute(); err != nil {
		var exitErr *ExitError
		if errors.As(err, &exitErr) {
			if strings.TrimSpace(exitErr.Message) != "" {
				fmt.Fprintln(a.stderr, exitErr.Message)
			}
			return exitErr.Code
		}
		fmt.Fprintln(a.stderr, err)
		return 1
	}
	return 0
}

func (a *App) trace(event string, kv ...string) {
	if len(kv)%2 != 0 {
		kv = append(kv, "")
	}
	attrs := make([]any, 0, len(kv))
	for i := 0; i < len(kv); i += 2 {
		attrs = append(attrs, kv[i], kv[i+1])
	}
	a.logger.Debug(event, attrs...)
}

func (a *App) configureLogger(level string) {
	resolvedLevel := parseLogLevel(level)
	a.logger = slog.New(slog.NewTextHandler(a.stderr, &slog.HandlerOptions{Level: resolvedLevel}))
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
