package world

import (
	"context"

	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(destroyCmd)
}

var destroyCmd = &cobra.Command{
	Use:   "destroy <world-id>",
	Short: "Destroy a world",
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
		s.Start("Destroying world...")

		u, err := arc.Destroy(ctx, worldID)
		if err != nil {
			s.Fail("Destroy failed", err)
			return err
		}

		s.Done("Stopped agent", "")
		s.Done("Removed container", "")
		if u.MindPath != "" {
			s.Done("Mind persisted", u.MindPath)
		}

		s.Blank()
		s.Success("World destroyed. Agent survives.")
		s.Blank()

		return nil
	},
}
