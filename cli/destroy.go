package cli

import (
	"context"

	"github.com/jterrazz/universe/cli/ui"
	"github.com/jterrazz/universe/internal/architect"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(destroyCmd)
}

var destroyCmd = &cobra.Command{
	Use:   "destroy <universe-id>",
	Short: "Destroy a universe",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		universeID := args[0]
		s := ui.New(quiet, verbose, jsonOutput)

		arc, err := architect.NewFromEnv()
		if err != nil {
			return err
		}

		s.Blank()
		s.Start("Destroying universe...")

		u, err := arc.Destroy(ctx, universeID)
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
		s.Success("Universe destroyed. Agent survives.")
		s.Blank()

		return nil
	},
}
