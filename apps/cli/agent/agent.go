package agent

import (
	"context"
	"fmt"

	"spwn.sh/apps/cli/ui"
	agentDomain "spwn.sh/core/agent"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

var (
	spawnName  string
	spawnWorld string
	spawnImport string
	npcTask    string
)

func init() {
	Cmd.Flags().StringVarP(&spawnName, "name", "n", "", "Agent name (default: default)")
	Cmd.Flags().StringVarP(&spawnWorld, "world", "u", "", "Target world ID")
	Cmd.Flags().StringVar(&spawnImport, "import", "", "Import Mind from tar.gz before spawning")
	Cmd.Flags().StringVar(&npcTask, "npc", "", "Run as NPC — no Mind, no memory, just execute this task")
}

// Cmd is the agent command — spawns an agent when run directly,
// and groups subcommands (init, list, inspect, export).
var Cmd = &cobra.Command{
	Use:   "agent",
	Short: "Spawn an agent — a living identity that inhabits a world",
	Long: `Spawn an agent into an existing world.

An agent is backed by a Mind — a persistent directory of personas, skills,
knowledge, playbooks, journal entries, and session state. The agent survives
after the world is destroyed.`,
	Example: `  spwn agent -n neo -u w-abc123      Spawn named agent into world
  spwn agent --npc "run tests"       Fire-and-forget NPC task
  spwn agent --import backup.tar.gz  Import a Mind archive first`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no flags set at all, show help
		if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("world") &&
			!cmd.Flags().Changed("npc") && !cmd.Flags().Changed("import") {
			return cmd.Help()
		}

		ctx := context.Background()
		s := newStepper(cmd)

		// NPC mode — no Mind, no identity, just execute
		if npcTask != "" {
			worldID := spawnWorld
			if worldID == "" {
				return fmt.Errorf("error: --world is required for NPC mode.\nRun 'spwn world list' to see active worlds.")
			}
			arc, err := universe.NewArchitectFromEnv()
			if err != nil {
				return err
			}
			s.Blank()
			s.Done("NPC dispatched", fmt.Sprintf("%q → %s", npcTask, worldID))
			s.Blank()
			return arc.SpawnNPC(ctx, worldID, npcTask)
		}

		agentName := "default"
		if spawnName != "" {
			agentName = spawnName
		}

		// Import Mind archive if requested
		if spawnImport != "" {
			s.Blank()
			s.Start("Importing agent...")
			if err := agentDomain.ImportMind(agentName, spawnImport); err != nil {
				s.Fail("Import failed", err)
				return fmt.Errorf("error: import failed.\n%w", err)
			}
			s.Done("Imported agent", agentName)
		}

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return err
		}

		// Resolve world ID
		worldID := spawnWorld
		if worldID == "" {
			worlds, err := arc.List(ctx)
			if err != nil {
				return fmt.Errorf("error: cannot list worlds.\n%w", err)
			}
			if len(worlds) == 0 {
				return fmt.Errorf("error: no active worlds.\nRun 'spwn world --no-agent' first")
			}
			if len(worlds) > 1 {
				s.Blank()
				s.Fail("Multiple active worlds", fmt.Errorf("error: specify one with --world."))
				for _, u := range worlds {
					s.Info("", fmt.Sprintf("%-20s (%s)", u.ID, u.Status))
				}
				s.Blank()
				return fmt.Errorf("error: multiple active worlds.\nSpecify one with --world.")
			}
			worldID = worlds[0].ID
		}

		s.Blank()
		s.Done("Spawning agent into", worldID)
		s.Blank()

		if err := arc.SpawnAgent(ctx, worldID, agentName); err != nil {
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
