package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"toolbox/internal/add"
	"toolbox/internal/output"
)

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
				Changed:     collectChangedFlags(cmd),
			})
			if err != nil {
				return &ExitError{Code: 1, Message: err.Error()}
			}

			if global.JSON {
				return output.JSON(a.stdout, result)
			}
			printAddPythonHumanResult(a, result)
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
	cmd.Flags().StringVar(&opts.Scope, "scope", "", "Target scope (project|user|bundled)")
	cmd.Flags().BoolVar(&opts.Overwrite, "overwrite", false, "Overwrite existing generated files")
	_ = cmd.RegisterFlagCompletionFunc("scope", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"project", "user", "bundled"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("input-mode", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"none", "file", "json"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("output-mode", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"text", "json"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func collectChangedFlags(cmd *cobra.Command) map[string]bool {
	flagNames := []string{
		"from-spec", "name", "script", "description", "arg", "env", "timeout",
		"cwd", "tag", "input-mode", "output-mode", "python-bin", "scope", "overwrite",
	}
	changed := make(map[string]bool, len(flagNames))
	for _, flagName := range flagNames {
		changed[flagName] = cmd.Flags().Changed(flagName)
	}
	return changed
}

func printAddPythonHumanResult(a *App, result add.PythonResult) {
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
}
