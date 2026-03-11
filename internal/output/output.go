package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"toolbox/internal/config"
	"toolbox/internal/doctor"
	"toolbox/internal/manifest"
	"toolbox/pkg/contract"
)

// JSON writes pretty JSON to output.
func JSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

// ListHuman prints tasks in a predictable table-like format.
func ListHuman(w io.Writer, tasks []manifest.Task) {
	if len(tasks) == 0 {
		_, _ = fmt.Fprintln(w, "No tasks found.")
		return
	}
	for _, task := range tasks {
		_, _ = fmt.Fprintf(w, "%s\n", task.Name)
		_, _ = fmt.Fprintf(w, "  description: %s\n", fallback(task.Description, "(none)"))
		_, _ = fmt.Fprintf(w, "  input_mode: %s\n", task.Input.Mode)
		_, _ = fmt.Fprintf(w, "  source: %s\n", task.Source.Path)
	}
}

// DryRunHuman prints dry-run details.
func DryRunHuman(w io.Writer, dryRun contract.DryRunEnvelope) {
	_, _ = fmt.Fprintf(w, "Task: %s\n", dryRun.Task)
	_, _ = fmt.Fprintf(w, "Command: %s\n", dryRun.Command)
	_, _ = fmt.Fprintf(w, "Args: %s\n", strings.Join(dryRun.Args, " "))
	_, _ = fmt.Fprintf(w, "CWD: %s\n", dryRun.Cwd)
	_, _ = fmt.Fprintf(w, "Timeout: %s\n", dryRun.Timeout)
	_, _ = fmt.Fprintln(w, "Env:")
	keys := make([]string, 0, len(dryRun.Env))
	for key := range dryRun.Env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		_, _ = fmt.Fprintf(w, "  %s=%s\n", key, dryRun.Env[key])
	}
}

// RunHuman prints a concise execution summary.
func RunHuman(w io.Writer, envelope contract.RunEnvelope) {
	status := "ok"
	if !envelope.OK {
		status = "failed"
	}
	_, _ = fmt.Fprintf(w, "Task: %s\n", envelope.Task)
	_, _ = fmt.Fprintf(w, "Status: %s\n", status)
	_, _ = fmt.Fprintf(w, "Exit Code: %d\n", envelope.ExitCode)
	_, _ = fmt.Fprintf(w, "Duration: %dms\n", envelope.DurationMS)
	_, _ = fmt.Fprintf(w, "Started At: %s\n", envelope.StartedAt.Format("2006-01-02T15:04:05Z07:00"))
	if envelope.Stdout != "" {
		_, _ = fmt.Fprintln(w, "Stdout:")
		_, _ = fmt.Fprintln(w, envelope.Stdout)
	}
	if envelope.Stderr != "" {
		_, _ = fmt.Fprintln(w, "Stderr:")
		_, _ = fmt.Fprintln(w, envelope.Stderr)
	}
	if envelope.StdoutTruncated || envelope.StderrTruncated {
		_, _ = fmt.Fprintf(w, "Truncated: stdout=%t stderr=%t\n", envelope.StdoutTruncated, envelope.StderrTruncated)
	}
}

// DoctorHuman prints doctor diagnostics.
func DoctorHuman(w io.Writer, report doctor.Report) {
	if len(report.Issues) == 0 {
		_, _ = fmt.Fprintln(w, "doctor: ok")
		return
	}
	for _, issue := range report.Issues {
		_, _ = fmt.Fprintf(w, "[%s] %s: %s\n", strings.ToUpper(issue.Level), issue.Check, issue.Message)
		if issue.Hint != "" {
			_, _ = fmt.Fprintf(w, "  hint: %s\n", issue.Hint)
		}
	}
}

// ConfigHuman prints resolved config plus source info.
func ConfigHuman(w io.Writer, loaded config.LoadedConfig) error {
	if _, err := fmt.Fprintln(w, "Config precedence (highest to lowest):"); err != nil {
		return err
	}
	for _, item := range loaded.Sources.Precedence {
		if _, err := fmt.Fprintf(w, "- %s\n", item); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "\nSources:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- user: %s\n", fallback(loaded.Sources.UserConfig, "(not found)")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- project: %s\n", fallback(loaded.Sources.ProjectConfig, "(not found)")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- explicit: %s\n", fallback(loaded.Sources.ExplicitConfig, "(not set)")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- env overrides: %s\n", fallback(strings.Join(loaded.Sources.EnvOverrides, ", "), "(none)")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- flag overrides: %s\n", fallback(strings.Join(loaded.Sources.FlagOverrides, ", "), "(none)")); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "\nResolved:"); err != nil {
		return err
	}
	return JSON(w, loaded.Config)
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}
