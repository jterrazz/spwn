package world

import (
	"context"

	"spwn.sh/packages/architect"
	"github.com/spf13/cobra"
	"spwn.sh/apps/cli/ui"
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
		s := ui.New()

		arc, err := architect.NewFromEnv()
		if err != nil {
			return dockerHint(err)
		}

		// Validate the world exists before emitting the "Entering"
		// banner. Without this guard, `world enter nonexistent` prints
		// a success line and then errors, which reads incoherently
		// (Finding #15).
		if _, err := arc.Inspect(ctx, worldID); err != nil {
			return s.FailHint("Enter failed", err,
				"Check running worlds with \"spwn world ls\"")
		}

		s.Blank()
		s.Done("Entering", worldID)
		s.Blank()

		return arc.Attach(ctx, worldID)
	},
}
