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
	spawnName     string
	spawnWorld    string
	spawnImport   string
	ephemeralTask string
	npcTaskCompat string // deprecated alias for ephemeralTask
)

func init() {
	Cmd.Flags().StringVarP(&spawnName, "name", "n", "", "Agent name (default: default)")
	Cmd.Flags().StringVarP(&spawnWorld, "world", "u", "", "Target world ID")
	Cmd.Flags().StringVar(&spawnImport, "import", "", "Import Mind from tar.gz before spawning")
	Cmd.Flags().StringVar(&ephemeralTask, "ephemeral", "", "Run as ephemeral agent — no Mind, no memory, just execute this task")
	Cmd.Flags().StringVar(&npcTaskCompat, "npc", "", "Run as ephemeral agent (deprecated: use --ephemeral)")
	_ = Cmd.Flags().MarkHidden("npc")

	defaultAgentHelp = Cmd.HelpFunc()
	Cmd.SetHelpFunc(agentHelp)
}

var defaultAgentHelp func(*cobra.Command, []string)

func agentHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "agent" {
		if defaultAgentHelp != nil {
			defaultAgentHelp(cmd, args)
		}
		return
	}

	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ agent")+" "+ui.Faint("— create and manage agents"),
		[]ui.HelpGroup{
			{Title: "Lifecycle", Commands: []ui.HelpEntry{
				{Name: "new <name>", Desc: "Create a new agent (interactive wizard)"},
				{Name: "ls", Desc: "List all agents"},
				{Name: "show <name>", Desc: "Show agent composition and history"},
				{Name: "rm <name>", Desc: "Remove an agent"},
				{Name: "talk <name> [msg]", Desc: "Talk to a running agent"},
			}},
			{Title: "Composition", Commands: []ui.HelpEntry{
				{Name: "add <name> --tool <pack>", Desc: "Add a tool pack to an agent"},
				{Name: "add <name> --skill <skill>", Desc: "Add a skill to an agent"},
				{Name: "add <name> --profile <name>", Desc: "Apply a profile to an agent"},
			}},
			{Title: "Evolution", Commands: []ui.HelpEntry{
				{Name: "dream <name>", Desc: "Analyze experience, discover patterns, promote playbooks"},
				{Name: "sleep <name>", Desc: "Shutdown — save state, consolidate, archive"},
			}},
			{Title: "Portability", Commands: []ui.HelpEntry{
				{Name: "fork <src> <dst>", Desc: "Clone an agent (memory included)"},
				{Name: "publish <name>", Desc: "Publish an agent to the registry (memory stripped)"},
				{Name: "pull <name>", Desc: "Install a shared agent from the registry"},
				{Name: "export <name>", Desc: "Export as tar.gz"},
				{Name: "import <path>", Desc: "Import from tar.gz"},
			}},
			{Title: "Spawn Flags", Commands: []ui.HelpEntry{
				{Name: "--ephemeral <task>", Desc: "Run as ephemeral agent (fire-and-forget)"},
				{Name: "-u, --world <id>", Desc: "Target world ID"},
			}},
		},
		"spwn agent [command]",
		"Agents are composed from tools, skills, and a profile.\n\n    Use \"spwn agent <command> --help\" for more information.",
	)
}

// Cmd is the agent command — spawns an agent when run directly,
// and groups subcommands (init, list, inspect, export).
var Cmd = &cobra.Command{
	Use:   "agent",
	Short: "Spawn an agent — a living identity that inhabits a world",
	Long: `Spawn an agent into an existing world.

An agent is backed by a Mind — a persistent directory holding its profile,
skills, knowledge, playbooks, journal entries, and session state. The agent
survives after the world is destroyed.`,
	Example: `  spwn agent -n neo -u w-abc123      Spawn named agent into world
  spwn agent --ephemeral "run tests"  Fire-and-forget ephemeral task
  spwn agent --import backup.tar.gz  Import a Mind archive first`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no flags set at all, show help
		if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("world") &&
			!cmd.Flags().Changed("ephemeral") && !cmd.Flags().Changed("npc") && !cmd.Flags().Changed("import") {
			return cmd.Help()
		}

		ctx := context.Background()
		s := newStepper(cmd)

		// Support deprecated --npc alias
		task := ephemeralTask
		if task == "" {
			task = npcTaskCompat
		}

		// Ephemeral mode — no Mind, no identity, just execute
		if task != "" {
			worldID := spawnWorld
			if worldID == "" {
				s.Blank()
				return s.FailHint("Ephemeral requires --world", fmt.Errorf("no world specified"),
					"Run \"spwn ls\" to see active worlds")
			}
			arc, err := universe.NewArchitectFromEnv()
			if err != nil {
				s.Blank()
				return s.FailHint("Docker", err, "Start Docker Desktop or OrbStack, then try again")
			}
			s.Blank()
			s.Done("Ephemeral dispatched", fmt.Sprintf("%q → %s", task, worldID))
			s.Blank()
			return arc.SpawnNPC(ctx, worldID, task)
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
				return s.FailHint("Import failed", err, "Check that the archive exists and is a valid tar.gz")
			}
			s.Done("Imported agent", agentName)
		}

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			s.Blank()
			return s.FailHint("Docker", err, "Start Docker Desktop or OrbStack, then try again")
		}

		// Resolve world ID
		worldID := spawnWorld
		if worldID == "" {
			worlds, err := arc.List(ctx)
			if err != nil {
				s.Blank()
				return s.FailHint("Cannot list worlds", err, "Run \"spwn doctor\" to diagnose")
			}
			if len(worlds) == 0 {
				s.Blank()
				return s.FailHint("No active worlds", fmt.Errorf("nothing to spawn into"),
					"Run \"spwn up -w .\" to create a world first")
			}
			if len(worlds) > 1 {
				s.Blank()
				s.Fail("Multiple worlds", fmt.Errorf("specify one with --world"))
				for _, u := range worlds {
					s.Info("", fmt.Sprintf("%-20s (%s)", u.ID, u.Status))
				}
				return &ui.DisplayedError{Err: fmt.Errorf("multiple worlds")}
			}
			worldID = worlds[0].ID
		}

		s.Blank()
		s.Done("Spawning agent into", worldID)
		s.Blank()

		if err := arc.SpawnAgent(ctx, worldID, agentName); err != nil {
			return fmt.Errorf("agent spawn failed: %w", err)
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
