package agent

import (
	"context"
	"fmt"

	"github.com/jterrazz/spwn/cli/ui"
	"github.com/jterrazz/spwn/internal/architect"
	"github.com/jterrazz/spwn/internal/mind"
	"github.com/spf13/cobra"
)

var (
	spawnName     string
	spawnUniverse string
	spawnImport   string
)

func init() {
	Cmd.Flags().StringVarP(&spawnName, "name", "n", "", "Agent name (default: default)")
	Cmd.Flags().StringVarP(&spawnUniverse, "universe", "u", "", "Target universe ID")
	Cmd.Flags().StringVar(&spawnImport, "import", "", "Import Mind from tar.gz before spawning")
}

// Cmd is the agent command — spawns an agent when run directly,
// and groups subcommands (init, list, inspect, export).
var Cmd = &cobra.Command{
	Use:   "agent",
	Short: "Spawn an agent — a living identity that inhabits a universe",
	Long: `Spawn an agent into an existing universe.

An agent is backed by a Mind — a persistent directory of personas, skills,
knowledge, playbooks, journal entries, and session state. One agent per
universe. The agent survives after the universe is destroyed.

Subcommands: init, list, inspect, export.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		s := newStepper(cmd)

		agentName := "default"
		if spawnName != "" {
			agentName = spawnName
		}

		// Import Mind archive if requested
		if spawnImport != "" {
			s.Blank()
			s.Start("Importing agent...")
			if err := mind.Import(agentName, spawnImport); err != nil {
				s.Fail("Import failed", err)
				return fmt.Errorf("error: import failed.\n%w", err)
			}
			s.Done("Imported agent", agentName)
		}

		arc, err := architect.NewFromEnv()
		if err != nil {
			return err
		}

		// Resolve universe ID
		universeID := spawnUniverse
		if universeID == "" {
			universes, err := arc.List(ctx)
			if err != nil {
				return fmt.Errorf("error: cannot list universes.\n%w", err)
			}
			if len(universes) == 0 {
				return fmt.Errorf("error: no active universes.\nRun 'spwn universe --no-agent' first")
			}
			if len(universes) > 1 {
				s.Blank()
				s.Fail("Multiple active universes", fmt.Errorf("specify one with --universe"))
				for _, u := range universes {
					s.Info("", fmt.Sprintf("%-20s (%s)", u.ID, u.Status))
				}
				s.Blank()
				return fmt.Errorf("multiple active universes")
			}
			universeID = universes[0].ID
		}

		s.Blank()
		s.Done("Spawning agent into", universeID)
		s.Blank()

		if err := arc.SpawnAgent(ctx, universeID, agentName); err != nil {
			return fmt.Errorf("error: agent spawn failed.\n%w", err)
		}

		return nil
	},
}

// newStepper creates a Stepper using the persistent root flags.
func newStepper(cmd *cobra.Command) *ui.Stepper {
	q, _ := cmd.Flags().GetBool("quiet")
	v, _ := cmd.Flags().GetBool("verbose")
	j, _ := cmd.Flags().GetBool("json")
	return ui.New(q, v, j)
}
