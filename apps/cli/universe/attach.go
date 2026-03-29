package universe

import (
	"context"

	"github.com/jterrazz/spwn/core/universe"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(attachCmd)
}

var attachCmd = &cobra.Command{
	Use:   "attach <universe-id>",
	Short: "Open interactive session into a running universe",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		universeID := args[0]
		s := newStepper(cmd)

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return err
		}

		s.Blank()
		s.Done("Attaching to", universeID)
		s.Blank()

		return arc.Attach(ctx, universeID)
	},
}
