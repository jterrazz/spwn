package world

import (
	"context"

	"spwn.sh/packages/universe"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(enterCmd)
}

var enterCmd = &cobra.Command{
	Use:   "enter <world-id>",
	Short: "Open an interactive shell inside a running world",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		worldID := args[0]
		s := newStepper(cmd)

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return dockerHint(err)
		}

		s.Blank()
		s.Done("Entering", worldID)
		s.Blank()

		return arc.Attach(ctx, worldID)
	},
}
