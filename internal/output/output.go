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
		fmt.Fprintln(w, "No tasks found.")
		return
	}
	for _, task := range tasks {
		fmt.Fprintf(w, "%s\n", task.Name)
		fmt.Fprintf(w, "  description: %s\n", fallback(task.Description, "(none)"))
		fmt.Fprintf(w, "  input_mode: %s\n", task.Input.Mode)
		fmt.Fprintf(w, "  source: %s\n", task.Source.Path)
	}
}

// DryRunHuman prints dry-run details.
func DryRunHuman(w io.Writer, dryRun contract.DryRunEnvelope) {
	fmt.Fprintf(w, "Task: %s\n", dryRun.Task)
	fmt.Fprintf(w, "Command: %s\n", dryRun.Command)
	fmt.Fprintf(w, "Args: %s\n", strings.Join(dryRun.Args, " "))
	fmt.Fprintf(w, "CWD: %s\n", dryRun.Cwd)
	fmt.Fprintf(w, "Timeout: %s\n", dryRun.Timeout)
	fmt.Fprintln(w, "Env:")
	keys := make([]string, 0, len(dryRun.Env))
	for key := range dryRun.Env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(w, "  %s=%s\n", key, dryRun.Env[key])
	}
}

// RunHuman prints a concise execution summary.
func RunHuman(w io.Writer, envelope contract.RunEnvelope) {
	status := "ok"
	if !envelope.OK {
		status = "failed"
	}
	fmt.Fprintf(w, "Task: %s\n", envelope.Task)
	fmt.Fprintf(w, "Status: %s\n", status)
	fmt.Fprintf(w, "Exit Code: %d\n", envelope.ExitCode)
	fmt.Fprintf(w, "Duration: %dms\n", envelope.DurationMS)
	fmt.Fprintf(w, "Started At: %s\n", envelope.StartedAt.Format("2006-01-02T15:04:05Z07:00"))
	if envelope.Stdout != "" {
		fmt.Fprintln(w, "Stdout:")
		fmt.Fprintln(w, envelope.Stdout)
	}
	if envelope.Stderr != "" {
		fmt.Fprintln(w, "Stderr:")
		fmt.Fprintln(w, envelope.Stderr)
	}
	if envelope.StdoutTruncated || envelope.StderrTruncated {
		fmt.Fprintf(w, "Truncated: stdout=%t stderr=%t\n", envelope.StdoutTruncated, envelope.StderrTruncated)
	}
}

// DoctorHuman prints doctor diagnostics.
func DoctorHuman(w io.Writer, report doctor.Report) {
	if len(report.Issues) == 0 {
		fmt.Fprintln(w, "doctor: ok")
		return
	}
	for _, issue := range report.Issues {
		fmt.Fprintf(w, "[%s] %s: %s\n", strings.ToUpper(issue.Level), issue.Check, issue.Message)
		if issue.Hint != "" {
			fmt.Fprintf(w, "  hint: %s\n", issue.Hint)
		}
	}
}

// ConfigHuman prints resolved config plus source info.
func ConfigHuman(w io.Writer, loaded config.LoadedConfig) {
	fmt.Fprintln(w, "Config precedence (highest to lowest):")
	for _, item := range loaded.Sources.Precedence {
		fmt.Fprintf(w, "- %s\n", item)
	}
	fmt.Fprintln(w, "\nSources:")
	fmt.Fprintf(w, "- user: %s\n", fallback(loaded.Sources.UserConfig, "(not found)"))
	fmt.Fprintf(w, "- project: %s\n", fallback(loaded.Sources.ProjectConfig, "(not found)"))
	fmt.Fprintf(w, "- explicit: %s\n", fallback(loaded.Sources.ExplicitConfig, "(not set)"))
	fmt.Fprintf(w, "- env overrides: %s\n", fallback(strings.Join(loaded.Sources.EnvOverrides, ", "), "(none)"))
	fmt.Fprintf(w, "- flag overrides: %s\n", fallback(strings.Join(loaded.Sources.FlagOverrides, ", "), "(none)"))
	fmt.Fprintln(w, "\nResolved:")
	_ = JSON(w, loaded.Config)
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}
