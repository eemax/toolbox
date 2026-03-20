package cli

import (
	"toolbox/internal/output"
	"toolbox/internal/shared"

	"github.com/spf13/cobra"
)

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

			tasks := shared.SortedTasks(catalog.Tasks)
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
