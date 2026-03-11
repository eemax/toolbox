package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (a *App) newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print toolbox version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Fprintln(a.stdout, a.version)
		},
	}
}
