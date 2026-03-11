package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"toolbox/internal/add"
	"toolbox/internal/config"
	"toolbox/internal/doctor"
	"toolbox/internal/manifest"
	"toolbox/internal/output"
	"toolbox/internal/runner"
)

// ExitError allows commands to control process exit codes.
type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("exit code %d", e.Code)
	}
	return e.Message
}

// App wires the CLI runtime.
type App struct {
	version string
	stdout  io.Writer
	stderr  io.Writer
	env     map[string]string
	logger  *slog.Logger
}

// New creates a CLI app instance.
func New(version string, stdout, stderr io.Writer, env map[string]string) *App {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	if env == nil {
		env = envToMap(os.Environ())
	}
	app := &App{version: version, stdout: stdout, stderr: stderr, env: env}
	app.configureLogger("info")
	return app
}

// Execute runs the CLI and returns the process exit code.
func (a *App) Execute() int {
	root := a.rootCommand()
	if err := root.Execute(); err != nil {
		var exitErr *ExitError
		if errors.As(err, &exitErr) {
			if strings.TrimSpace(exitErr.Message) != "" {
				fmt.Fprintln(a.stderr, exitErr.Message)
			}
			return exitErr.Code
		}
		fmt.Fprintln(a.stderr, err)
		return 1
	}
	return 0
}

type globalFlags struct {
	ConfigPath string
	Verbose    bool
	LogLevel   string
	JSON       bool
}

type runFlags struct {
	InputFile     string
	DryRun        bool
	DryRunFullEnv bool
	Timeout       string
}

type addPythonFlags struct {
	FromSpec    string
	Name        string
	Script      string
	Description string
	Args        []string
	Env         []string
	Timeout     string
	CWD         string
	Tags        []string
	InputMode   string
	OutputMode  string
	PythonBin   string
	Scope       string
	Overwrite   bool
}

func (a *App) rootCommand() *cobra.Command {
	global := &globalFlags{}

	root := &cobra.Command{
		Use:           "toolbox",
		Short:         "Run local tasks through a consistent CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&global.ConfigPath, "config", "", "Path to config file")
	root.PersistentFlags().BoolVar(&global.Verbose, "verbose", false, "Enable verbose execution tracing")
	root.PersistentFlags().StringVar(&global.LogLevel, "log-level", "", "Set log level (debug|info|warn|error)")
	root.PersistentFlags().BoolVar(&global.JSON, "json", false, "Emit machine-readable JSON output")

	root.AddCommand(
		a.newListCommand(global),
		a.newRunCommand(global),
		a.newAddCommand(global),
		a.newDoctorCommand(global),
		a.newConfigCommand(global),
		a.newVersionCommand(),
	)
	return root
}

func (a *App) newListCommand(global *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available tasks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			loaded, catalog, err := a.loadConfigAndCatalog(ctx, cmd, global)
			if err != nil {
				return err
			}
			if err := catalog.FatalError(); err != nil {
				return &ExitError{Code: 1, Message: err.Error()}
			}
			tasks := sortedTasks(catalog.Tasks)
			if global.JSON {
				type listItem struct {
					Name        string `json:"name"`
					Description string `json:"description"`
					InputMode   string `json:"input_mode"`
					Source      string `json:"source"`
				}
				items := make([]listItem, 0, len(tasks))
				for _, task := range tasks {
					items = append(items, listItem{
						Name:        task.Name,
						Description: task.Description,
						InputMode:   string(task.Input.Mode),
						Source:      task.Source.Path,
					})
				}
				payload := map[string]any{
					"tasks":      items,
					"task_count": len(tasks),
					"config": map[string]any{
						"log_level": loaded.Config.LogLevel,
					},
				}
				return output.JSON(a.stdout, payload)
			}
			output.ListHuman(a.stdout, tasks)
			return nil
		},
	}
}

func (a *App) newRunCommand(global *globalFlags) *cobra.Command {
	runOpts := &runFlags{}
	cmd := &cobra.Command{
		Use:   "run <task>",
		Short: "Run a task by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			loaded, catalog, err := a.loadConfigAndCatalog(ctx, cmd, global)
			if err != nil {
				return err
			}
			if err := catalog.FatalError(); err != nil {
				return &ExitError{Code: 1, Message: err.Error()}
			}

			taskName := args[0]
			task, ok := catalog.Tasks[taskName]
			if !ok {
				return &ExitError{Code: 1, Message: fmt.Sprintf("task %q not found", taskName)}
			}

			timeout, err := parseTimeout(runOpts.Timeout)
			if err != nil {
				return &ExitError{Code: 1, Message: err.Error()}
			}

			cwd, err := os.Getwd()
			if err != nil {
				return &ExitError{Code: 1, Message: fmt.Sprintf("resolve cwd: %v", err)}
			}

			r := runner.New(loaded.Config, loaded.Raw, a.env)
			if global.Verbose {
				a.trace("task resolution", "task", task.Name, "source", task.Source.Path)
				a.trace("dependency check", "requires", strings.Join(task.Requires, ","))
			}

			result, execErr := r.Execute(ctx, runner.ExecuteOptions{
				Task:            task,
				InputFile:       runOpts.InputFile,
				TimeoutOverride: timeout,
				DryRun:          runOpts.DryRun,
				DryRunFullEnv:   runOpts.DryRunFullEnv,
				CWD:             cwd,
			})

			if result.DryRun != nil {
				if global.JSON {
					if err := output.JSON(a.stdout, result.DryRun); err != nil {
						return err
					}
				} else {
					output.DryRunHuman(a.stdout, *result.DryRun)
				}
				if execErr != nil {
					var runErr *runner.ExecError
					if errors.As(execErr, &runErr) {
						msg := runErr.Message
						if global.JSON {
							msg = ""
						}
						return &ExitError{Code: runErr.ExitCode, Message: msg}
					}
					return execErr
				}
				return nil
			}

			if global.JSON {
				if err := output.JSON(a.stdout, result.Envelope); err != nil {
					return err
				}
			} else {
				output.RunHuman(a.stdout, result.Envelope)
			}

			if execErr != nil {
				var runErr *runner.ExecError
				if errors.As(execErr, &runErr) {
					msg := runErr.Message
					if global.JSON {
						msg = ""
					}
					return &ExitError{Code: runErr.ExitCode, Message: msg}
				}
				return execErr
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&runOpts.InputFile, "input", "", "Input file path")
	cmd.Flags().BoolVar(&runOpts.DryRun, "dry-run", false, "Print resolved execution plan without running")
	cmd.Flags().BoolVar(&runOpts.DryRunFullEnv, "dry-run-full-env", false, "Include inherited environment in dry-run output (redacted)")
	cmd.Flags().StringVar(&runOpts.Timeout, "timeout", "", "Override timeout for this run (e.g. 30s)")
	return cmd
}

func (a *App) newAddCommand(global *globalFlags) *cobra.Command {
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add new tasks",
	}
	addCmd.AddCommand(a.newAddPythonCommand(global))
	return addCmd
}

func (a *App) newAddPythonCommand(global *globalFlags) *cobra.Command {
	opts := &addPythonFlags{}
	cmd := &cobra.Command{
		Use:   "python",
		Short: "Add a Python script task",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return &ExitError{Code: 1, Message: fmt.Sprintf("resolve cwd: %v", err)}
			}
			home := strings.TrimSpace(a.env["HOME"])
			if home == "" {
				home, err = os.UserHomeDir()
				if err != nil {
					return &ExitError{Code: 1, Message: fmt.Sprintf("resolve home dir: %v", err)}
				}
			}

			flagNames := []string{
				"from-spec", "name", "script", "description", "arg", "env", "timeout",
				"cwd", "tag", "input-mode", "output-mode", "python-bin", "scope", "overwrite",
			}
			changed := make(map[string]bool, len(flagNames))
			for _, flagName := range flagNames {
				changed[flagName] = cmd.Flags().Changed(flagName)
			}

			result, err := add.AddPython(add.PythonOptions{
				CWD:         cwd,
				HomeDir:     home,
				FromSpec:    opts.FromSpec,
				Name:        opts.Name,
				Script:      opts.Script,
				Description: opts.Description,
				Args:        opts.Args,
				Env:         opts.Env,
				Timeout:     opts.Timeout,
				TaskCWD:     opts.CWD,
				Tags:        opts.Tags,
				InputMode:   opts.InputMode,
				OutputMode:  opts.OutputMode,
				PythonBin:   opts.PythonBin,
				Scope:       opts.Scope,
				Overwrite:   opts.Overwrite,
				Changed:     changed,
			})
			if err != nil {
				return &ExitError{Code: 1, Message: err.Error()}
			}

			if global.JSON {
				return output.JSON(a.stdout, result)
			}

			fmt.Fprintf(a.stdout, "Created task %q\n", result.Task)
			fmt.Fprintf(a.stdout, "Scope: %s\n", result.Scope)
			fmt.Fprintf(a.stdout, "Manifest: %s\n", result.ManifestPath)
			fmt.Fprintf(a.stdout, "Script: %s\n", result.ScriptPath)
			fmt.Fprintf(a.stdout, "Python: %s\n", result.PythonBin)
			fmt.Fprintln(a.stdout, "Checks:")
			for _, check := range result.Checks {
				fmt.Fprintf(a.stdout, "- %s: %s\n", check.Name, check.Status)
			}
			fmt.Fprintf(a.stdout, "Next: %s\n", result.NextCommand)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.FromSpec, "from-spec", "", "Path to YAML/JSON add specification")
	cmd.Flags().StringVar(&opts.Name, "name", "", "Task name")
	cmd.Flags().StringVar(&opts.Script, "script", "", "Path to source python script")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Task description")
	cmd.Flags().StringArrayVar(&opts.Args, "arg", nil, "Task argument (repeatable)")
	cmd.Flags().StringArrayVar(&opts.Env, "env", nil, "Task environment variable KEY=VALUE (repeatable)")
	cmd.Flags().StringVar(&opts.Timeout, "timeout", "", "Task timeout duration (e.g. 30s)")
	cmd.Flags().StringVar(&opts.CWD, "cwd", "", "Task working directory")
	cmd.Flags().StringArrayVar(&opts.Tags, "tag", nil, "Task tag (repeatable)")
	cmd.Flags().StringVar(&opts.InputMode, "input-mode", "", "Task input mode (none|file|json)")
	cmd.Flags().StringVar(&opts.OutputMode, "output-mode", "", "Task output mode (text|json)")
	cmd.Flags().StringVar(&opts.PythonBin, "python-bin", "", "Python interpreter binary")
	cmd.Flags().StringVar(&opts.Scope, "scope", "", "Target scope (project|user)")
	cmd.Flags().BoolVar(&opts.Overwrite, "overwrite", false, "Overwrite existing generated files")
	return cmd
}

func (a *App) newDoctorCommand(global *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Validate config, manifests, and dependencies",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			loaded, catalog, err := a.loadConfigAndCatalog(ctx, cmd, global)
			if err != nil {
				return err
			}
			report := doctor.Run(loaded, catalog)
			if global.JSON {
				if err := output.JSON(a.stdout, report); err != nil {
					return err
				}
			} else {
				output.DoctorHuman(a.stdout, report)
			}
			if report.HasErrors() {
				return &ExitError{Code: 1}
			}
			return nil
		},
	}
}

func (a *App) newConfigCommand(global *globalFlags) *cobra.Command {
	configCmd := &cobra.Command{Use: "config", Short: "Inspect resolved configuration"}
	configCmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show resolved config and precedence",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			loaded, _, err := a.loadConfigAndCatalog(ctx, cmd, global)
			if err != nil {
				return err
			}
			if global.JSON {
				payload := map[string]any{
					"config":  loaded.Config,
					"sources": loaded.Sources,
				}
				return output.JSON(a.stdout, payload)
			}
			return output.ConfigHuman(a.stdout, loaded)
		},
	})
	return configCmd
}

func (a *App) newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print toolbox version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Fprintln(a.stdout, a.version)
		},
	}
}

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

	catalog := manifest.Load(manifest.LoadOptions{
		ProjectDir: config.ProjectTaskDir(cwd),
		UserDir:    config.UserTaskDir(home),
	})
	if global.Verbose {
		a.trace("manifest resolution", "project_dir", config.ProjectTaskDir(cwd), "user_dir", config.UserTaskDir(home), "task_count", fmt.Sprintf("%d", len(catalog.Tasks)))
	}
	return loaded, catalog, nil
}

func (a *App) trace(event string, kv ...string) {
	if len(kv)%2 != 0 {
		kv = append(kv, "")
	}
	attrs := make([]any, 0, len(kv))
	for i := 0; i < len(kv); i += 2 {
		attrs = append(attrs, kv[i], kv[i+1])
	}
	a.logger.Debug(event, attrs...)
}

func (a *App) configureLogger(level string) {
	resolvedLevel := parseLogLevel(level)
	a.logger = slog.New(slog.NewTextHandler(a.stderr, &slog.HandlerOptions{Level: resolvedLevel}))
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
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
