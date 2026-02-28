package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jterrazz/universe/internal/architect"
)

func inspectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <id>",
		Short: "Inspect a universe",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := architect.New()
			if err != nil {
				return err
			}

			u, err := a.Inspect(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			fmt.Printf("Universe: %s\n", u.ID)
			fmt.Printf("  Image:      %s\n", u.Image)
			fmt.Printf("  Status:     %s\n", u.Status)
			mind := "-"
			if u.Mind != "" {
				mind = u.Mind
			}
			fmt.Printf("  Mind:       %s\n", mind)
			fmt.Printf("  Created at: %s\n", u.CreatedAt.Format("2006-01-02 15:04:05"))
			return nil
		},
	}
}
