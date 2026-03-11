package doctor

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"toolbox/internal/config"
	"toolbox/internal/manifest"
)

// Issue is a doctor finding.
type Issue struct {
	Level   string `json:"level"`
	Check   string `json:"check"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

// Report contains all findings.
type Report struct {
	Issues []Issue `json:"issues"`
}

// HasErrors returns true when report contains one or more error-level issues.
func (r Report) HasErrors() bool {
	for _, issue := range r.Issues {
		if strings.EqualFold(issue.Level, "error") {
			return true
		}
	}
	return false
}

// Run builds a full diagnostic report.
func Run(cfg config.LoadedConfig, catalog manifest.Catalog) Report {
	report := Report{Issues: []Issue{}}
	if cfg.Sources.UserConfig == "" && cfg.Sources.ProjectConfig == "" && cfg.Sources.ExplicitConfig == "" {
		report.Issues = append(report.Issues, Issue{
			Level:   "warning",
			Check:   "config",
			Message: "no user/project config file found",
			Hint:    "Create .toolbox/config.yaml or ~/.config/toolbox/config.yaml to customize defaults.",
		})
	}

	for _, err := range catalog.Errors {
		report.Issues = append(report.Issues, Issue{
			Level:   "error",
			Check:   "manifest",
			Message: err.Error(),
			Hint:    "Fix YAML parsing/validation errors and run `toolbox doctor` again.",
		})
	}

	if len(catalog.DuplicateNames) > 0 {
		duplicateNames := make([]string, 0, len(catalog.DuplicateNames))
		for name := range catalog.DuplicateNames {
			duplicateNames = append(duplicateNames, name)
		}
		sort.Strings(duplicateNames)
		for _, name := range duplicateNames {
			sources := catalog.DuplicateNames[name]
			parts := make([]string, 0, len(sources))
			for _, source := range sources {
				label := source.Scope
				if strings.TrimSpace(source.Category) != "" {
					label = source.Category
				}
				parts = append(parts, fmt.Sprintf("%s (%s)", source.Path, label))
			}
			report.Issues = append(report.Issues, Issue{
				Level:   "error",
				Check:   "duplicate_task",
				Message: fmt.Sprintf("task %q is defined more than once: %s", name, strings.Join(parts, ", ")),
				Hint:    "Rename or remove one definition; duplicates are not allowed in v1.",
			})
		}
	}

	for _, task := range sortedTasks(catalog.Tasks) {
		if _, err := exec.LookPath(task.Command); err != nil {
			report.Issues = append(report.Issues, Issue{
				Level:   "error",
				Check:   "runtime",
				Message: fmt.Sprintf("task %q command %q not found in PATH", task.Name, task.Command),
				Hint:    fmt.Sprintf("Install %q or update PATH.", task.Command),
			})
		}
		for _, dependency := range task.Requires {
			if strings.TrimSpace(dependency) == "" {
				continue
			}
			if _, err := exec.LookPath(dependency); err != nil {
				report.Issues = append(report.Issues, Issue{
					Level:   "error",
					Check:   "runtime",
					Message: fmt.Sprintf("task %q missing required binary %q", task.Name, dependency),
					Hint:    fmt.Sprintf("Install %q and rerun doctor.", dependency),
				})
			}
		}
	}

	sort.Slice(report.Issues, func(i, j int) bool {
		if report.Issues[i].Level != report.Issues[j].Level {
			return report.Issues[i].Level < report.Issues[j].Level
		}
		if report.Issues[i].Check != report.Issues[j].Check {
			return report.Issues[i].Check < report.Issues[j].Check
		}
		return report.Issues[i].Message < report.Issues[j].Message
	})

	return report
}

func sortedTasks(tasks map[string]manifest.Task) []manifest.Task {
	names := make([]string, 0, len(tasks))
	for name := range tasks {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]manifest.Task, 0, len(names))
	for _, name := range names {
		result = append(result, tasks[name])
	}
	return result
}
