package agent

import (
	"context"
	"fmt"

	"spwn.sh/apps/cli/ui"

	agentDomain "spwn.sh/packages/mind"
	"spwn.sh/packages/world"
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
	Cmd.Flags().StringVar(&ephemeralTask, "ephemeral", "", "Run as ephemeral agent - no Mind, no memory, just execute this task")
	Cmd.Flags().StringVar(&npcTaskCompat, "npc", "", "Run as ephemeral agent (deprecated: use --ephemeral)")
	_ = Cmd.Flags().MarkHidden("npc")

	Cmd.SetHelpFunc(agentHelp)
}

func agentHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "agent" {
		ui.MinimalHelp(cmd, args)
		return
	}

	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ agent")+" "+ui.Faint("- composed minds that live in worlds"),
		[]ui.HelpGroup{
			{Title: "Lifecycle", Commands: []ui.HelpEntry{
				{Name: "new <name>", Desc: "Create a blank agent (auto-creates a single-agent world)"},
				{Name: "<name>", Desc: "Start the world that contains <name> " + ui.Faint("(shortcut)")},
				{Name: "start <name>", Desc: "Start the world containing this agent"},
				{Name: "stop <name>", Desc: "Stop the world containing this agent"},
				{Name: "ls", Desc: "List agents"},
				{Name: "rm <name>", Desc: "Delete an agent"},
			}},
			{Title: "Compose", Commands: []ui.HelpEntry{
				{Name: "add <name>", Desc: "Attach blocks " + ui.Faint("(--tool / --skill / --profile)")},
				{Name: "remove <name>", Desc: "Detach blocks " + ui.Faint("(--tool / --skill / --profile)")},
			}},
			{Title: "Conversation", Commands: []ui.HelpEntry{
				{Name: "talk <name> [msg]", Desc: "Open a session with a running agent " + ui.Faint("(sync)")},
				{Name: "send <name> <msg>", Desc: "Send an async message to an agent's inbox"},
				{Name: "inbox <name>", Desc: "Show an agent's inbox"},
				{Name: "watch <name>", Desc: "Tail an agent's inbox in real time"},
			}},
			{Title: "Observe", Commands: []ui.HelpEntry{
				{Name: "inspect <name>", Desc: "Inspect composition, memory, and history"},
				{Name: "logs <name>", Desc: "Show the event log for this agent"},
			}},
			{Title: "Evolution", Commands: []ui.HelpEntry{
				{Name: "dream <name>", Desc: "Analyze experience, promote playbooks"},
				{Name: "sleep <name>", Desc: "Consolidate memory, prune stale strategies"},
				{Name: "fork <src> <dst>", Desc: "Clone an agent with everything it knows"},
			}},
			{Title: "Portability", Commands: []ui.HelpEntry{
				{Name: "publish <name>", Desc: "Ship to registry " + ui.Faint("[planned]")},
				{Name: "get <ref>", Desc: "Install a shared agent " + ui.Faint("[planned]")},
				{Name: "export <name>", Desc: "Export as tar.gz"},
				{Name: "import <path>", Desc: "Import from tar.gz"},
			}},
			{Title: "Examples", Commands: []ui.HelpEntry{
				{Name: "spwn agent new neo", Desc: ""},
				{Name: "spwn agent add neo --tool @spwn/python --profile researcher", Desc: ""},
				{Name: "spwn up --agent neo -w .", Desc: ""},
			}},
		},
		"spwn agent [command]",
		"",
	)
}

// Cmd is the agent command. In the new grammar:
//   - `spwn agent` with no args and no flags -> help
//   - `spwn agent <name>` -> start the world that contains <name>
//   - `spwn agent --flags ...` -> legacy imperative spawn
// Subcommands (new, ls, start, stop, inspect, ...) resolve first.
var Cmd = &cobra.Command{
	Use:   "agent [name]",
	Short: "Spawn an agent - a living identity that inhabits a world",
	Long: `Spawn an agent into an existing world.

An agent is backed by a Mind - a persistent directory holding its profile,
skills, knowledge, playbooks, journal entries, and session state. The agent
survives after the world is destroyed.`,
	Args: cobra.ArbitraryArgs, // subcommands still resolve first
	Example: `  spwn agent new neo                 Create a blank agent
  spwn agent neo                     Start the world that contains neo
  spwn agent -n neo -u w-abc123      Legacy: spawn named agent into world
  spwn agent --ephemeral "run tests"  Fire-and-forget ephemeral task`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Compose-style shortcut: `spwn agent <name>` -> agent start <name>.
		if len(args) == 1 && !cmd.Flags().Changed("name") && !cmd.Flags().Changed("world") &&
			!cmd.Flags().Changed("ephemeral") && !cmd.Flags().Changed("npc") && !cmd.Flags().Changed("import") {
			return startCmd.RunE(cmd, args)
		}
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

		// Ephemeral mode - no Mind, no identity, just execute
		if task != "" {
			worldID := spawnWorld
			if worldID == "" {
				s.Blank()
				return s.FailHint("Ephemeral requires --world", fmt.Errorf("no world specified"),
					"Run \"spwn ls\" to see active worlds")
			}
			arc, err := world.NewArchitectFromEnv()
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

		arc, err := world.NewArchitectFromEnv()
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
				return s.FailHint("Cannot list worlds", err, "")
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

func newStepper(cmd *cobra.Command) *ui.Stepper {
	return ui.New()
}
