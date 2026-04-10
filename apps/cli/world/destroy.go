package world

import (
	"context"
	"fmt"

	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

var destroyAll bool

func init() {
	Cmd.AddCommand(destroyCmd)
	destroyCmd.Flags().BoolVar(&destroyAll, "all", false, "Destroy all running worlds")
}

var destroyCmd = &cobra.Command{
	Use:   "destroy [world-id]",
	Short: "Destroy a world",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		s := newStepper(cmd)

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return s.FailHint("Docker", err, "Start Docker Desktop or OrbStack, then try again")
		}

		// --all: destroy all running worlds sequentially
		if destroyAll {
			s.Blank()
			s.Start("Stopping all worlds...")

			destroyed, err := arc.DestroyAll(ctx)
			if err != nil {
				return s.FailHint("Destroy all failed", err, "Check Docker is running")
			}

			if len(destroyed) == 0 {
				s.Done("No worlds running", "")
			} else {
				for _, u := range destroyed {
					label := u.ID
					if u.Agent != "" {
						label += " (" + u.Agent + ")"
					}
					s.Done("Destroyed", label)
				}
			}

			s.Blank()
			s.Success(fmt.Sprintf("%d world(s) destroyed.", len(destroyed)))
			s.Blank()
			return nil
		}

		// Single world destroy
		if len(args) == 0 {
			return fmt.Errorf("requires a world-id argument or --all flag")
		}
		worldID := args[0]

		s.Blank()
		s.Start("Destroying world...")

		u, err := arc.Destroy(ctx, worldID)
		if err != nil {
			return s.FailHint("Destroy failed", err,
				fmt.Sprintf("Check that world %q exists with \"spwn ls\"", worldID))
		}

		s.Done("Stopped agent", "")
		s.Done("Removed container", "")
		if u.Agent != "" {
			s.Done("Mind persisted", "~/.spwn/agents/"+u.Agent)
		}

		s.Blank()
		s.Success("World destroyed. Agent survives.")
		s.Blank()

		return nil
	},
}
