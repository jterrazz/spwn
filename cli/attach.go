package cli

import (
	"context"

	"github.com/jterrazz/universe/cli/ui"
	"github.com/jterrazz/universe/internal/architect"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(attachCmd)
}

var attachCmd = &cobra.Command{
	Use:   "attach <universe-id>",
	Short: "Open interactive session into a running universe",
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
		s.Done("Attaching to", universeID)
		s.Blank()

		return arc.Attach(ctx, universeID)
	},
}
