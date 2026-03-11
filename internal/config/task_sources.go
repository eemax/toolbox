package config

import (
	"os"
	"path/filepath"
)

const (
	SourceCategoryUser           = "user"
	SourceCategoryProjectLegacy  = "project-legacy"
	SourceCategoryProjectBundled = "project-bundled"
)

// TaskCatalogSource describes one task manifest directory for catalog resolution.
type TaskCatalogSource struct {
	Scope    string
	Category string
	Dir      string
}

// ProjectTaskSources returns all project task directories in deterministic order.
func ProjectTaskSources(cwd string) []TaskCatalogSource {
	return []TaskCatalogSource{
		{
			Scope:    "project",
			Category: SourceCategoryProjectLegacy,
			Dir:      ProjectTaskDir(cwd),
		},
		{
			Scope:    "project",
			Category: SourceCategoryProjectBundled,
			Dir:      ProjectBundledTaskDir(cwd),
		},
	}
}

// CatalogTaskSources returns user + project task directories in deterministic order.
func CatalogTaskSources(cwd, home string) []TaskCatalogSource {
	sources := []TaskCatalogSource{
		{
			Scope:    "user",
			Category: SourceCategoryUser,
			Dir:      UserTaskDir(home),
		},
	}
	return append(sources, ProjectTaskSources(cwd)...)
}

// LegacyTaskLayoutOnly reports whether the project uses only .toolbox/tasks without ./tasks.
func LegacyTaskLayoutOnly(cwd string) bool {
	legacy := ProjectTaskDir(cwd)
	bundled := ProjectBundledTaskDir(cwd)
	return dirExists(legacy) && !dirExists(bundled)
}

func dirExists(path string) bool {
	info, err := os.Stat(filepath.Clean(path))
	if err != nil {
		return false
	}
	return info.IsDir()
}
