package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"toolbox/internal/output"
	"toolbox/internal/runner"
)

func (a *App) newRunCommand(global *globalFlags) *cobra.Command {
	runOpts := &runFlags{}
	cmd := &cobra.Command{
		Use:   "run <task>",
		Short: "Run a task by name",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return a.completeTaskNames(cmd, global, args, toComplete)
		},
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
				return wrapExecError(execErr, global.JSON)
			}

			if global.JSON {
				if err := output.JSON(a.stdout, result.Envelope); err != nil {
					return err
				}
			} else {
				output.RunHuman(a.stdout, result.Envelope)
			}
			return wrapExecError(execErr, global.JSON)
		},
	}

	cmd.Flags().StringVar(&runOpts.InputFile, "input", "", "Input file path")
	cmd.Flags().BoolVar(&runOpts.DryRun, "dry-run", false, "Print resolved execution plan without running")
	cmd.Flags().BoolVar(&runOpts.DryRunFullEnv, "dry-run-full-env", false, "Include inherited environment in dry-run output (redacted)")
	cmd.Flags().StringVar(&runOpts.Timeout, "timeout", "", "Override timeout for this run (e.g. 30s)")
	return cmd
}

func wrapExecError(execErr error, jsonOutput bool) error {
	if execErr == nil {
		return nil
	}
	var runErr *runner.ExecError
	if errors.As(execErr, &runErr) {
		msg := runErr.Message
		if jsonOutput {
			msg = ""
		}
		return &ExitError{Code: runErr.ExitCode, Message: msg}
	}
	return execErr
}
