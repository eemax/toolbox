package cli

import (
	"github.com/spf13/cobra"

	"toolbox/internal/doctor"
	"toolbox/internal/output"
)

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
