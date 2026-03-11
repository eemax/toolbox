package cli

import "github.com/spf13/cobra"

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
	_ = root.RegisterFlagCompletionFunc("log-level", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"debug", "info", "warn", "error"}, cobra.ShellCompDirectiveNoFileComp
	})

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
