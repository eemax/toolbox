package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"toolbox/internal/config"
	"toolbox/internal/manifest"
	"toolbox/pkg/contract"
)

const (
	exitCodeCommandNotFound = 127
	exitCodeTimeout         = 124
)

// ExecError carries exit code context for command failures.
type ExecError struct {
	ExitCode int
	Message  string
}

func (e *ExecError) Error() string {
	return e.Message
}

// ExecuteOptions controls task execution behavior.
type ExecuteOptions struct {
	Task            manifest.Task
	InputFile       string
	TimeoutOverride time.Duration
	DryRun          bool
	DryRunFullEnv   bool
	CWD             string
}

// ExecuteResult contains either a run envelope or dry-run data.
type ExecuteResult struct {
	Envelope contract.RunEnvelope
	DryRun   *contract.DryRunEnvelope
}

// Runner executes task manifests.
type Runner struct {
	cfg    config.Config
	rawCfg map[string]any
	env    map[string]string
	now    func() time.Time
}

// New returns a Runner configured with runtime config and environment.
func New(cfg config.Config, rawCfg map[string]any, env map[string]string) *Runner {
	if env == nil {
		env = envToMap(os.Environ())
	}
	if rawCfg == nil {
		rawCfg = map[string]any{}
	}
	return &Runner{cfg: cfg, rawCfg: rawCfg, env: env, now: time.Now}
}

// Execute validates, resolves, and executes a task.
func (r *Runner) Execute(ctx context.Context, opts ExecuteOptions) (ExecuteResult, error) {
	task := opts.Task
	if strings.TrimSpace(task.Name) == "" {
		return ExecuteResult{}, &ExecError{ExitCode: 1, Message: "task name is required"}
	}
	baseEnvelope := contract.RunEnvelope{
		Task:      task.Name,
		OK:        false,
		ExitCode:  1,
		Artifacts: []string{},
		StartedAt: r.now().UTC(),
	}

	timeout := task.Timeout
	if timeout <= 0 {
		timeout = r.cfg.Execution.DefaultTimeout
	}
	if opts.TimeoutOverride > 0 {
		timeout = opts.TimeoutOverride
	}

	vars, err := r.templateVars(task, opts.InputFile)
	if err != nil {
		return preflightFailure(baseEnvelope, 1, err.Error()), &ExecError{ExitCode: 1, Message: err.Error()}
	}

	resolvedArgs, err := ResolveSlice(task.Args, vars)
	if err != nil {
		return preflightFailure(baseEnvelope, 1, err.Error()), &ExecError{ExitCode: 1, Message: err.Error()}
	}

	resolvedEnv, err := resolveEnvTemplates(task.Env, vars)
	if err != nil {
		return preflightFailure(baseEnvelope, 1, err.Error()), &ExecError{ExitCode: 1, Message: err.Error()}
	}

	resolvedCWD := opts.CWD
	if strings.TrimSpace(task.CWD) != "" {
		resolvedCWD = task.CWD
		if !filepath.IsAbs(resolvedCWD) {
			resolvedCWD = filepath.Join(opts.CWD, resolvedCWD)
		}
	}
	if resolvedCWD == "" {
		resolvedCWD, err = os.Getwd()
		if err != nil {
			message := fmt.Sprintf("resolve cwd: %v", err)
			return preflightFailure(baseEnvelope, 1, message), &ExecError{ExitCode: 1, Message: message}
		}
	}

	commandPath, err := resolveCommandPath(task.Command, resolvedCWD)
	if err != nil {
		return preflightFailure(baseEnvelope, exitCodeCommandNotFound, err.Error()), &ExecError{ExitCode: exitCodeCommandNotFound, Message: err.Error()}
	}
	if err := validatePathPolicy(commandPath, r.cfg.Execution.AllowPaths, r.cfg.Execution.DenyPaths); err != nil {
		return preflightFailure(baseEnvelope, 1, err.Error()), &ExecError{ExitCode: 1, Message: err.Error()}
	}

	if err := validateDependencies(task.Requires); err != nil {
		return preflightFailure(baseEnvelope, exitCodeCommandNotFound, err.Error()), &ExecError{ExitCode: exitCodeCommandNotFound, Message: err.Error()}
	}

	if opts.DryRun {
		dryRunEnv := resolvedEnv
		if opts.DryRunFullEnv {
			dryRunEnv = mergeEnv(r.env, resolvedEnv)
		}
		dryRun := contract.DryRunEnvelope{
			Task:    task.Name,
			Command: commandPath,
			Args:    resolvedArgs,
			Cwd:     resolvedCWD,
			Timeout: timeout.String(),
			Env:     redactEnv(dryRunEnv, r.cfg.Execution.RedactKeys),
		}
		return ExecuteResult{DryRun: &dryRun}, nil
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startedAt := r.now().UTC()
	started := r.now()
	cmd := exec.CommandContext(runCtx, commandPath, resolvedArgs...)
	cmd.Dir = resolvedCWD
	cmd.Env = mapToSortedEnv(mergeEnv(r.env, resolvedEnv))

	stdoutBuffer := newCappedBuffer(r.cfg.Output.CaptureLimitBytes)
	stderrBuffer := newCappedBuffer(r.cfg.Output.CaptureLimitBytes)
	cmd.Stdout = stdoutBuffer
	cmd.Stderr = stderrBuffer

	runErr := cmd.Run()
	duration := time.Since(started)

	envelope := contract.RunEnvelope{
		Task:            task.Name,
		DurationMS:      duration.Milliseconds(),
		Stdout:          stdoutBuffer.String(),
		Stderr:          stderrBuffer.String(),
		Artifacts:       []string{},
		StartedAt:       startedAt,
		StdoutTruncated: stdoutBuffer.Truncated(),
		StderrTruncated: stderrBuffer.Truncated(),
		StdoutBytes:     stdoutBuffer.BytesWritten(),
		StderrBytes:     stderrBuffer.BytesWritten(),
	}

	if runErr == nil {
		envelope.OK = true
		envelope.ExitCode = 0
		return ExecuteResult{Envelope: envelope}, nil
	}

	exitCode := 1
	message := runErr.Error()
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		exitCode = exitCodeTimeout
		message = fmt.Sprintf("task %q timed out after %s", task.Name, timeout)
	} else {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
			message = fmt.Sprintf("task %q exited with code %d", task.Name, exitCode)
		}
	}
	envelope.OK = false
	envelope.ExitCode = exitCode
	if strings.TrimSpace(envelope.Stderr) == "" {
		envelope.Stderr = message
	}
	return ExecuteResult{Envelope: envelope}, &ExecError{ExitCode: exitCode, Message: message}
}

func resolveCommandPath(command, resolvedCWD string) (string, error) {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return "", errors.New("task command is required")
	}

	candidate := trimmed
	if isPathLikeCommand(trimmed) && !filepath.IsAbs(trimmed) {
		candidate = filepath.Join(resolvedCWD, trimmed)
	}

	path, err := exec.LookPath(candidate)
	if err != nil {
		if isPathLikeCommand(trimmed) {
			return "", fmt.Errorf("command path %q not found or not executable", candidate)
		}
		return "", fmt.Errorf("command %q not found in PATH", trimmed)
	}
	return path, nil
}

func isPathLikeCommand(command string) bool {
	return strings.Contains(command, "/") || strings.Contains(command, "\\")
}

func preflightFailure(base contract.RunEnvelope, exitCode int, message string) ExecuteResult {
	base.ExitCode = exitCode
	base.Stderr = message
	return ExecuteResult{Envelope: base}
}

func (r *Runner) templateVars(task manifest.Task, inputPath string) (map[string]string, error) {
	vars := map[string]string{}
	for key, value := range flattenToStrings(r.rawCfg) {
		vars["config."+key] = value
	}
	for key, value := range r.env {
		vars["env."+key] = value
	}

	if strings.TrimSpace(inputPath) != "" {
		vars["input.file"] = inputPath
	}

	switch task.Input.Mode {
	case manifest.InputModeFile:
		if strings.TrimSpace(inputPath) == "" {
			return nil, errors.New("task requires --input <file>")
		}
	case manifest.InputModeJSON:
		if strings.TrimSpace(inputPath) == "" {
			return nil, errors.New("task requires --input <file> for json input mode")
		}
		data, err := os.ReadFile(inputPath)
		if err != nil {
			return nil, fmt.Errorf("read input file %s: %w", inputPath, err)
		}
		vars["input.json"] = string(data)
	case manifest.InputModeNone:
		// No required input.
	default:
		return nil, fmt.Errorf("unsupported input mode %q", task.Input.Mode)
	}

	return vars, nil
}

func validateDependencies(requires []string) error {
	missing := make([]string, 0)
	for _, required := range requires {
		if strings.TrimSpace(required) == "" {
			continue
		}
		if _, err := exec.LookPath(required); err != nil {
			missing = append(missing, required)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("missing required binaries: %s", strings.Join(missing, ", "))
	}
	return nil
}

func validatePathPolicy(commandPath string, allowPaths, denyPaths []string) error {
	cleanPath := filepath.Clean(commandPath)
	if matchesPolicy(cleanPath, denyPaths) {
		return fmt.Errorf("command path %q denied by execution policy", cleanPath)
	}
	if len(allowPaths) > 0 && !matchesPolicy(cleanPath, allowPaths) {
		return fmt.Errorf("command path %q is not in allowed execution paths", cleanPath)
	}
	return nil
}

func matchesPolicy(commandPath string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	base := filepath.Base(commandPath)
	for _, pattern := range patterns {
		trimmed := strings.TrimSpace(pattern)
		if trimmed == "" {
			continue
		}
		cleanPattern := filepath.Clean(trimmed)
		if base == cleanPattern || commandPath == cleanPattern || strings.HasPrefix(commandPath, cleanPattern+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func resolveEnvTemplates(taskEnv map[string]string, vars map[string]string) (map[string]string, error) {
	if len(taskEnv) == 0 {
		return map[string]string{}, nil
	}
	resolved := make(map[string]string, len(taskEnv))
	for key, value := range taskEnv {
		item, err := ResolveTemplate(value, vars)
		if err != nil {
			return nil, fmt.Errorf("resolve env %q: %w", key, err)
		}
		resolved[key] = item
	}
	return resolved, nil
}

func mergeEnv(base map[string]string, overrides map[string]string) map[string]string {
	result := make(map[string]string, len(base)+len(overrides))
	for key, value := range base {
		result[key] = value
	}
	for key, value := range overrides {
		result[key] = value
	}
	return result
}

func mapToSortedEnv(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, fmt.Sprintf("%s=%s", key, values[key]))
	}
	return out
}

func flattenToStrings(values map[string]any) map[string]string {
	result := map[string]string{}
	flatten("", values, result)
	return result
}

func flatten(prefix string, value any, out map[string]string) {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			next := key
			if prefix != "" {
				next = prefix + "." + key
			}
			flatten(next, typed[key], out)
		}
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, fmt.Sprint(item))
		}
		out[prefix] = strings.Join(items, ",")
	case []string:
		out[prefix] = strings.Join(typed, ",")
	default:
		out[prefix] = fmt.Sprint(value)
	}
}

func redactEnv(values map[string]string, keys []string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	result := make(map[string]string, len(values))
	patterns := make([]string, 0, len(keys))
	for _, key := range keys {
		if trimmed := strings.TrimSpace(key); trimmed != "" {
			patterns = append(patterns, strings.ToUpper(trimmed))
		}
	}
	for key, value := range values {
		upper := strings.ToUpper(key)
		redact := false
		for _, pattern := range patterns {
			if strings.Contains(upper, pattern) {
				redact = true
				break
			}
		}
		if redact {
			result[key] = "<redacted>"
			continue
		}
		result[key] = value
	}
	return result
}

type cappedBuffer struct {
	cap       int64
	total     int64
	truncated bool
	buffer    bytes.Buffer
}

func newCappedBuffer(capacity int64) *cappedBuffer {
	if capacity <= 0 {
		capacity = 1
	}
	return &cappedBuffer{cap: capacity}
}

func (b *cappedBuffer) Write(data []byte) (int, error) {
	written := len(data)
	b.total += int64(written)
	remaining := b.cap - int64(b.buffer.Len())
	if remaining > 0 {
		toWrite := int64(written)
		if toWrite > remaining {
			toWrite = remaining
		}
		_, _ = b.buffer.Write(data[:toWrite])
	}
	if int64(written) > remaining {
		b.truncated = true
	}
	return written, nil
}

func (b *cappedBuffer) String() string {
	return b.buffer.String()
}

func (b *cappedBuffer) BytesWritten() int64 {
	return b.total
}

func (b *cappedBuffer) Truncated() bool {
	return b.truncated
}

func envToMap(entries []string) map[string]string {
	result := make(map[string]string, len(entries))
	for _, entry := range entries {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		result[parts[0]] = parts[1]
	}
	return result
}
