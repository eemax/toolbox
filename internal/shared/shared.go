package shared

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"toolbox/internal/manifest"
)

// EnvToMap converts os.Environ()-style entries ("KEY=VALUE") to a map.
func EnvToMap(entries []string) map[string]string {
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

// SortedTasks returns tasks sorted alphabetically by name.
func SortedTasks(tasks map[string]manifest.Task) []manifest.Task {
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

// FileExists reports whether a regular file exists at path.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// SourceLabel returns the display label for a task source.
func SourceLabel(scope, category string) string {
	if strings.TrimSpace(category) != "" {
		return category
	}
	return scope
}

// FormatDuplicateSources formats duplicate task source info for error messages.
func FormatDuplicateSources(sources []manifest.SourceInfo) []string {
	parts := make([]string, 0, len(sources))
	for _, source := range sources {
		parts = append(parts, fmt.Sprintf("%s (%s)", source.Path, SourceLabel(source.Scope, source.Category)))
	}
	return parts
}
