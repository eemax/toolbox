package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// InputMode controls task input expectations.
type InputMode string

const (
	InputModeNone InputMode = "none"
	InputModeFile InputMode = "file"
	InputModeJSON InputMode = "json"
)

// OutputMode controls task output interpretation.
type OutputMode string

const (
	OutputModeText OutputMode = "text"
	OutputModeJSON OutputMode = "json"
)

// Task is a validated task definition with source metadata.
type Task struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Command     string            `json:"command" yaml:"command"`
	Args        []string          `json:"args" yaml:"args"`
	Input       InputSpec         `json:"input" yaml:"input"`
	Output      OutputSpec        `json:"output" yaml:"output"`
	Timeout     time.Duration     `json:"timeout" yaml:"-"`
	TimeoutRaw  string            `json:"timeout_raw,omitempty" yaml:"timeout"`
	Env         map[string]string `json:"env" yaml:"env"`
	Requires    []string          `json:"requires" yaml:"requires"`
	Tags        []string          `json:"tags" yaml:"tags"`
	CWD         string            `json:"cwd,omitempty" yaml:"cwd"`
	Source      SourceInfo        `json:"source" yaml:"-"`
}

// InputSpec defines task input mode.
type InputSpec struct {
	Mode InputMode `json:"mode" yaml:"mode"`
}

// OutputSpec defines task output mode.
type OutputSpec struct {
	Mode OutputMode `json:"mode" yaml:"mode"`
}

// SourceInfo tracks where the task was loaded from.
type SourceInfo struct {
	Scope    string `json:"scope"`
	Category string `json:"category,omitempty"`
	Path     string `json:"path"`
}

// LoadOptions defines where task manifests are loaded from.
type LoadOptions struct {
	ProjectDir  string
	ProjectDirs []string
	UserDir     string
	Sources     []SourceDir
}

// SourceDir defines one scope/category directory to load task manifests from.
type SourceDir struct {
	Scope    string
	Category string
	Dir      string
}

// Catalog contains all parsed tasks and diagnostics.
type Catalog struct {
	Tasks          map[string]Task         `json:"tasks"`
	TaskOrder      []string                `json:"task_order"`
	DuplicateNames map[string][]SourceInfo `json:"duplicate_names"`
	Errors         []error                 `json:"-"`
}

// Load reads and validates manifests from configured task directories.
func Load(opts LoadOptions) Catalog {
	catalog := Catalog{
		Tasks:          map[string]Task{},
		DuplicateNames: map[string][]SourceInfo{},
	}

	loadFromDir := func(scope, category, dir string) {
		if strings.TrimSpace(dir) == "" {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return
			}
			catalog.Errors = append(catalog.Errors, fmt.Errorf("read %s task directory %s: %w", sourceLabel(scope, category), dir, err))
			return
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !isYAMLFile(name) {
				continue
			}
			path := filepath.Join(dir, name)
			task, err := parseTask(path)
			if err != nil {
				catalog.Errors = append(catalog.Errors, err)
				continue
			}
			task.Source = SourceInfo{Scope: scope, Category: category, Path: path}
			if existing, exists := catalog.Tasks[task.Name]; exists {
				catalog.DuplicateNames[task.Name] = appendUniqueSources(catalog.DuplicateNames[task.Name], existing.Source)
				catalog.DuplicateNames[task.Name] = appendUniqueSources(catalog.DuplicateNames[task.Name], task.Source)
				continue
			}
			catalog.Tasks[task.Name] = task
			catalog.TaskOrder = append(catalog.TaskOrder, task.Name)
		}
	}

	for _, source := range normalizeSourceDirs(opts) {
		loadFromDir(source.Scope, source.Category, source.Dir)
	}

	sort.Strings(catalog.TaskOrder)
	for name := range catalog.DuplicateNames {
		delete(catalog.Tasks, name)
	}

	return catalog
}

// FatalError returns a single error suitable for command failure when catalog has fatal issues.
func (c Catalog) FatalError() error {
	if len(c.Errors) == 0 && len(c.DuplicateNames) == 0 {
		return nil
	}
	parts := make([]string, 0, len(c.Errors)+len(c.DuplicateNames))
	for _, err := range c.Errors {
		parts = append(parts, err.Error())
	}
	if len(c.DuplicateNames) > 0 {
		duplicateNames := make([]string, 0, len(c.DuplicateNames))
		for name := range c.DuplicateNames {
			duplicateNames = append(duplicateNames, name)
		}
		sort.Strings(duplicateNames)
		for _, name := range duplicateNames {
			sources := c.DuplicateNames[name]
			sourcePaths := make([]string, 0, len(sources))
			for _, source := range sources {
				sourcePaths = append(sourcePaths, fmt.Sprintf("%s (%s)", source.Path, sourceInfoLabel(source)))
			}
			parts = append(parts, fmt.Sprintf("duplicate task %q found in: %s", name, strings.Join(sourcePaths, ", ")))
		}
	}
	return errors.New(strings.Join(parts, "; "))
}

func parseTask(path string) (Task, error) {
	fileData, err := os.ReadFile(path)
	if err != nil {
		return Task{}, fmt.Errorf("read task file %s: %w", path, err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(fileData))
	decoder.KnownFields(true)
	var task Task
	if err := decoder.Decode(&task); err != nil {
		if !errors.Is(err, io.EOF) {
			return Task{}, fmt.Errorf("parse task file %s: %w", path, err)
		}
	}
	if err := validateTask(task); err != nil {
		return Task{}, fmt.Errorf("validate task file %s: %w", path, err)
	}
	if task.TimeoutRaw != "" {
		duration, err := time.ParseDuration(task.TimeoutRaw)
		if err != nil {
			return Task{}, fmt.Errorf("validate task file %s: timeout must be a valid duration: %w", path, err)
		}
		task.Timeout = duration
	}
	if task.Env == nil {
		task.Env = map[string]string{}
	}
	if task.Input.Mode == "" {
		task.Input.Mode = InputModeNone
	}
	if task.Output.Mode == "" {
		task.Output.Mode = OutputModeText
	}
	return task, nil
}

func validateTask(task Task) error {
	if strings.TrimSpace(task.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(task.Command) == "" {
		return errors.New("command is required")
	}
	if !isValidInputMode(task.Input.Mode) {
		return fmt.Errorf("input.mode must be one of %q, %q, %q", InputModeNone, InputModeFile, InputModeJSON)
	}
	if !isValidOutputMode(task.Output.Mode) {
		return fmt.Errorf("output.mode must be one of %q, %q", OutputModeText, OutputModeJSON)
	}
	return nil
}

// ValidateTask validates a task value without source file context.
func ValidateTask(task Task) error {
	return validateTask(task)
}

func isValidInputMode(mode InputMode) bool {
	switch mode {
	case "", InputModeNone, InputModeFile, InputModeJSON:
		return true
	default:
		return false
	}
}

func isValidOutputMode(mode OutputMode) bool {
	switch mode {
	case "", OutputModeText, OutputModeJSON:
		return true
	default:
		return false
	}
}

func isYAMLFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}

func appendUniqueSources(current []SourceInfo, source SourceInfo) []SourceInfo {
	for _, existing := range current {
		if existing.Path == source.Path && existing.Scope == source.Scope {
			return current
		}
	}
	return append(current, source)
}

func normalizeProjectDirs(primary string, additional []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 1+len(additional))
	add := func(dir string) {
		trimmed := strings.TrimSpace(dir)
		if trimmed == "" {
			return
		}
		clean := filepath.Clean(trimmed)
		if _, exists := seen[clean]; exists {
			return
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	add(primary)
	for _, dir := range additional {
		add(dir)
	}
	return out
}

func normalizeSourceDirs(opts LoadOptions) []SourceDir {
	if len(opts.Sources) > 0 {
		return dedupeSourceDirs(opts.Sources)
	}

	out := []SourceDir{}
	if strings.TrimSpace(opts.UserDir) != "" {
		out = append(out, SourceDir{Scope: "user", Category: "user", Dir: opts.UserDir})
	}
	for _, dir := range normalizeProjectDirs(opts.ProjectDir, opts.ProjectDirs) {
		out = append(out, SourceDir{Scope: "project", Category: "project", Dir: dir})
	}
	return dedupeSourceDirs(out)
}

func dedupeSourceDirs(input []SourceDir) []SourceDir {
	seen := map[string]struct{}{}
	out := make([]SourceDir, 0, len(input))
	for _, source := range input {
		scope := strings.TrimSpace(source.Scope)
		category := strings.TrimSpace(source.Category)
		dir := filepath.Clean(strings.TrimSpace(source.Dir))
		if dir == "." || dir == "" {
			continue
		}
		if scope == "" {
			scope = "project"
		}
		if category == "" {
			category = scope
		}
		key := scope + "|" + category + "|" + dir
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, SourceDir{Scope: scope, Category: category, Dir: dir})
	}
	return out
}

func sourceLabel(scope, category string) string {
	if strings.TrimSpace(category) != "" {
		return category
	}
	return scope
}

func sourceInfoLabel(source SourceInfo) string {
	return sourceLabel(source.Scope, source.Category)
}
