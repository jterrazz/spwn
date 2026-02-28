package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jterrazz/universe/internal/architect"
)

func destroyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "destroy <id>",
		Short: "Destroy a universe (stop + remove)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := architect.New()
			if err != nil {
				return err
			}

			if err := a.Destroy(cmd.Context(), args[0]); err != nil {
				return err
			}

			fmt.Printf("Universe destroyed: %s\n", args[0])
			return nil
		},
	}
}
