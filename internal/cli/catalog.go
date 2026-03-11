package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"toolbox/internal/config"
	"toolbox/internal/manifest"
)

func (a *App) loadConfigAndCatalog(ctx context.Context, cmd *cobra.Command, global *globalFlags) (config.LoadedConfig, manifest.Catalog, error) {
	_ = ctx
	levelForResolution := global.LogLevel
	if strings.TrimSpace(levelForResolution) == "" {
		levelForResolution = "info"
	}
	if global.Verbose {
		levelForResolution = "debug"
	}
	a.configureLogger(levelForResolution)

	cwd, err := os.Getwd()
	if err != nil {
		return config.LoadedConfig{}, manifest.Catalog{}, &ExitError{Code: 1, Message: fmt.Sprintf("resolve cwd: %v", err)}
	}
	home := a.env["HOME"]
	flagOverrides := map[string]any{}
	if cmd.Root().PersistentFlags().Changed("log-level") {
		flagOverrides["log_level"] = global.LogLevel
	}
	if global.Verbose {
		a.trace("config resolution", "cwd", cwd, "config_flag", global.ConfigPath)
	}

	loaded, err := config.Load(config.LoadOptions{
		CWD:           cwd,
		ConfigPath:    global.ConfigPath,
		FlagOverrides: flagOverrides,
		Env:           a.env,
		HomeDir:       home,
	})
	if err != nil {
		return config.LoadedConfig{}, manifest.Catalog{}, &ExitError{Code: 1, Message: err.Error()}
	}

	level := loaded.Config.LogLevel
	if cmd.Root().PersistentFlags().Changed("log-level") {
		level = global.LogLevel
	}
	if global.Verbose {
		level = "debug"
	}
	a.configureLogger(level)

	taskSources := config.CatalogTaskSources(cwd, home)
	catalog := manifest.Load(manifest.LoadOptions{Sources: toManifestSources(taskSources)})
	a.warnIfLegacyTaskLayout(cwd)

	if global.Verbose {
		projectDirs := make([]string, 0, 2)
		for _, source := range taskSources {
			if source.Scope == "project" {
				projectDirs = append(projectDirs, source.Dir)
			}
		}
		a.trace(
			"manifest resolution",
			"project_dirs", strings.Join(projectDirs, ","),
			"user_dir", config.UserTaskDir(home),
			"task_count", fmt.Sprintf("%d", len(catalog.Tasks)),
		)
	}
	return loaded, catalog, nil
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

func (a *App) completeTaskNames(cmd *cobra.Command, global *globalFlags, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	_, catalog, err := a.loadConfigAndCatalog(cmd.Context(), cmd, global)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if err := catalog.FatalError(); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	names := make([]string, 0, len(catalog.Tasks))
	for name := range catalog.Tasks {
		if strings.HasPrefix(name, toComplete) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, cobra.ShellCompDirectiveNoFileComp
}

func parseTimeout(value string) (time.Duration, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	duration, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid --timeout value %q: %w", trimmed, err)
	}
	if duration <= 0 {
		return 0, errors.New("--timeout must be greater than zero")
	}
	return duration, nil
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

func (a *App) warnIfLegacyTaskLayout(cwd string) {
	if !config.LegacyTaskLayoutOnly(cwd) {
		return
	}
	fmt.Fprintln(
		a.stderr,
		"[WARN] project uses legacy task layout at .toolbox/tasks; migrate to ./tasks for portable/bundled task workflows.",
	)
}
