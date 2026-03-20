package cli

import (
	"github.com/spf13/cobra"

	"toolbox/internal/output"
)

func (a *App) newConfigCommand(global *globalFlags) *cobra.Command {
	configCmd := &cobra.Command{Use: "config", Short: "Inspect resolved configuration"}
	configCmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show resolved config and precedence",
		RunE: func(cmd *cobra.Command, _ []string) error {
			loaded, err := a.loadConfigOnly(cmd, global)
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
			output.ConfigHuman(a.stdout, loaded)
			return nil
		},
	})
	return configCmd
}
