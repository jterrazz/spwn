package world

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"spwn.sh/packages/world"
)

var destroyAll bool

func init() {
	Cmd.AddCommand(destroyCmd)
	destroyCmd.Flags().BoolVar(&destroyAll, "all", false, "Destroy all running worlds")
}

// composeDownRunE is the compose-aware entry point for `spwn down`.
// With no positional arg inside a project, it stops every running
// world whose Config name matches an entry in spwn.yaml. Otherwise it
// forwards to destroyCmd.RunE, which handles both <world-id> and
// --all.
func composeDownRunE(cmd *cobra.Command, args []string) error {
	if destroyAll || len(args) > 0 {
		return destroyCmd.RunE(cmd, args)
	}
	p, err := loadProject()
	if err != nil || p == nil || len(p.Manifest.Worlds) == 0 {
		return destroyCmd.RunE(cmd, args)
	}

	ctx := context.Background()
	s := newStepper(cmd)
	arc, err := world.NewArchitectFromEnv()
	if err != nil {
		return s.FailHint("Docker", err, "Start Docker Desktop or OrbStack, then try again")
	}

	running, err := arc.List(ctx)
	if err != nil {
		return fmt.Errorf("cannot list worlds: %w", err)
	}

	projectWorldNames := map[string]bool{}
	for name := range p.Manifest.Worlds {
		projectWorldNames[name] = true
	}

	s.Blank()
	s.Start("Stopping project worlds...")

	n := 0
	for _, u := range running {
		if !projectWorldNames[u.Config] {
			continue
		}
		if _, err := arc.Destroy(ctx, u.ID); err != nil {
			s.Warn("Destroy failed", fmt.Sprintf("%s: %v", u.ID, err))
			continue
		}
		label := u.ID
		if u.Agent != "" {
			label += " (" + u.Agent + ")"
		}
		s.Done("Destroyed", label)
		n++
	}

	s.Blank()
	if n == 0 {
		s.Success("No project worlds were running.")
	} else {
		s.Success(fmt.Sprintf("%d project world(s) destroyed.", n))
	}
	s.Blank()
	return nil
}

var destroyCmd = &cobra.Command{
	Use:   "destroy [world-id]",
	Short: "Destroy a world",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		s := newStepper(cmd)

		arc, err := world.NewArchitectFromEnv()
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
				fmt.Sprintf("Check that world %q exists with \"spwn world list\"", worldID))
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
