package world

import (
	"context"

	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(attachCmd)
}

var attachCmd = &cobra.Command{
	Use:   "attach <world-id>",
	Short: "Open interactive session into a running world",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		worldID := args[0]
		s := newStepper(cmd)

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return err
		}

		s.Blank()
		s.Done("Attaching to", worldID)
		s.Blank()

		return arc.Attach(ctx, worldID)
	},
}
