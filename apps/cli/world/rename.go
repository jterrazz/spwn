package world

import (
	"context"
	"fmt"
	"strings"

	"spwn.sh/packages/universe"

	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(renameCmd)
}

var renameCmd = &cobra.Command{
	Use:   "rename <world-id> [name]",
	Short: "Rename a world (omit name to clear)",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		s := newStepper(cmd)

		worldID := args[0]
		name := ""
		if len(args) > 1 {
			name = strings.TrimSpace(args[1])
		}

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return s.FailHint("Docker", err, "Start Docker Desktop or OrbStack, then try again")
		}

		s.Blank()
		s.Start("Renaming world...")

		if err := arc.Rename(ctx, worldID, name); err != nil {
			return s.FailHint("Rename failed", err,
				fmt.Sprintf("Check that world %q exists with \"spwn ls\"", worldID))
		}

		if name == "" {
			s.Done("Name cleared", worldID)
		} else {
			s.Done("Renamed", fmt.Sprintf("%s → %s", worldID, name))
		}
		s.Blank()
		return nil
	},
}
