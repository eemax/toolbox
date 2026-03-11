package add

import (
	"fmt"
	"path/filepath"
	"strings"

	"toolbox/internal/config"
	"toolbox/internal/manifest"
)

func (a pythonAdder) checkTaskNameConflicts(cfg resolvedPythonConfig) error {
	sources := config.CatalogTaskSources(cfg.CWD, cfg.HomeDir)
	catalog := a.loadCatalog(manifest.LoadOptions{Sources: toManifestSources(sources)})

	if sources, ok := catalog.DuplicateNames[cfg.Name]; ok && len(sources) > 0 {
		parts := make([]string, 0, len(sources))
		for _, source := range sources {
			label := source.Scope
			if strings.TrimSpace(source.Category) != "" {
				label = source.Category
			}
			parts = append(parts, fmt.Sprintf("%s (%s)", source.Path, label))
		}
		return fmt.Errorf("task name %q already exists in multiple manifests: %s", cfg.Name, strings.Join(parts, ", "))
	}

	existingTask, exists := catalog.Tasks[cfg.Name]
	if !exists {
		return nil
	}
	existingPath := filepath.Clean(existingTask.Source.Path)
	targetPath := filepath.Clean(cfg.ManifestPath)
	if existingPath != targetPath {
		return fmt.Errorf("task name %q already exists at %q (%s scope); choose a different --name", cfg.Name, existingTask.Source.Path, existingTask.Source.Scope)
	}
	return nil
}

func toManifestSources(sources []config.TaskCatalogSource) []manifest.SourceDir {
	result := make([]manifest.SourceDir, 0, len(sources))
	for _, source := range sources {
		result = append(result, manifest.SourceDir{
			Scope:    source.Scope,
			Category: source.Category,
			Dir:      source.Dir,
		})
	}
	return result
}
