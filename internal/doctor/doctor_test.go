package doctor

import (
	"errors"
	"strings"
	"testing"

	"toolbox/internal/config"
	"toolbox/internal/manifest"
)

func TestHasErrors(t *testing.T) {
	t.Parallel()
	report := Report{Issues: []Issue{{Level: "warning"}, {Level: "ERROR"}}}
	if !report.HasErrors() {
		t.Fatalf("expected HasErrors to detect error-level issue")
	}
}

func TestRunReportsMissingConfigWarning(t *testing.T) {
	t.Parallel()
	report := Run(config.LoadedConfig{}, manifest.Catalog{Tasks: map[string]manifest.Task{}})
	if len(report.Issues) != 1 {
		t.Fatalf("expected one warning issue, got %d", len(report.Issues))
	}
	issue := report.Issues[0]
	if issue.Check != "config" || issue.Level != "warning" {
		t.Fatalf("unexpected warning issue: %+v", issue)
	}
}

func TestRunIncludesManifestAndDuplicateErrorsWithCategories(t *testing.T) {
	t.Parallel()
	catalog := manifest.Catalog{
		Tasks: map[string]manifest.Task{},
		Errors: []error{
			errors.New("parse task file /tmp/bad.yaml: invalid"),
		},
		DuplicateNames: map[string][]manifest.SourceInfo{
			"dup-task": {
				{Path: "/tmp/project/.toolbox/tasks/dup.yaml", Scope: "project", Category: "project-legacy"},
				{Path: "/tmp/project/tasks/dup.yaml", Scope: "project", Category: "project-bundled"},
				{Path: "/tmp/home/.config/toolbox/tasks/dup.yaml", Scope: "user", Category: "user"},
			},
		},
	}
	cfg := config.LoadedConfig{Sources: config.Sources{ProjectConfig: "/tmp/project/.toolbox/config.yaml"}}

	report := Run(cfg, catalog)
	if !report.HasErrors() {
		t.Fatalf("expected error issues")
	}
	if !hasIssue(report.Issues, "manifest", "parse task file /tmp/bad.yaml") {
		t.Fatalf("expected manifest parse issue in report: %+v", report.Issues)
	}
	if !hasIssue(report.Issues, "duplicate_task", "project-legacy") {
		t.Fatalf("expected project-legacy category in duplicate message: %+v", report.Issues)
	}
	if !hasIssue(report.Issues, "duplicate_task", "project-bundled") {
		t.Fatalf("expected project-bundled category in duplicate message: %+v", report.Issues)
	}
	if !hasIssue(report.Issues, "duplicate_task", "user") {
		t.Fatalf("expected user category in duplicate message: %+v", report.Issues)
	}
}

func TestRunReportsMissingRuntimeDependencies(t *testing.T) {
	t.Parallel()
	catalog := manifest.Catalog{
		Tasks: map[string]manifest.Task{
			"missing-runtime": {
				Name:    "missing-runtime",
				Command: "toolbox-missing-command-for-test",
				Requires: []string{
					"toolbox-missing-required-binary",
					"",
				},
			},
		},
	}
	cfg := config.LoadedConfig{Sources: config.Sources{ProjectConfig: "/tmp/project/.toolbox/config.yaml"}}

	report := Run(cfg, catalog)
	if !report.HasErrors() {
		t.Fatalf("expected runtime errors")
	}
	if !hasIssue(report.Issues, "runtime", "command \"toolbox-missing-command-for-test\" not found") {
		t.Fatalf("expected missing command issue: %+v", report.Issues)
	}
	if !hasIssue(report.Issues, "runtime", "missing required binary \"toolbox-missing-required-binary\"") {
		t.Fatalf("expected missing required dependency issue: %+v", report.Issues)
	}
}

func hasIssue(issues []Issue, check string, messageContains string) bool {
	for _, issue := range issues {
		if issue.Check == check && strings.Contains(issue.Message, messageContains) {
			return true
		}
	}
	return false
}
