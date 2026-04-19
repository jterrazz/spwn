package agent

import (
	"context"
	"fmt"

	"spwn.sh/apps/cli/ui"
	worldcmd "spwn.sh/apps/cli/world"

	"github.com/spf13/cobra"
	"spwn.sh/packages/agent"
	"spwn.sh/packages/architect"
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
		ui.Strong("⬡ agent")+" "+ui.Faint("- composed minds: SOUL.md at root + 2 Mind layers (playbooks/journal); skills and knowledge are world-scoped"),
		[]ui.HelpGroup{
			{Title: "Lifecycle", Commands: []ui.HelpEntry{
				{Name: "create <name>", Desc: "Create a blank agent (auto-creates a single-agent world)"},
				{Name: "<name>", Desc: "Open an interactive session with <name> " + ui.Faint("(shortcut)")},
				{Name: "start <name>", Desc: "Run <name> as an autonomous daemon " + ui.Faint("[planned]")},
				{Name: "stop <name>", Desc: "Kill <name>'s daemon loop " + ui.Faint("[planned]")},
				{Name: "ls", Desc: "List agents"},
				{Name: "rm <name>", Desc: "Delete an agent"},
			}},
			// Composition lives on the root `spwn install` /
			// `spwn uninstall` verbs now; pass --agent <name> to
			// Scope a change to a single agent. The old `agent add`
			// / `agent remove` subcommands were retired.
			{Title: "Conversation", Commands: []ui.HelpEntry{
				{Name: "talk <name> [msg]", Desc: "Open a session with a running agent " + ui.Faint("(sync)")},
				{Name: "send <name> <msg>", Desc: "Send an async message to an agent's inbox " + ui.Faint("[planned]")},
				{Name: "inbox <name>", Desc: "Show an agent's inbox " + ui.Faint("[planned]")},
				{Name: "watch <name>", Desc: "Tail an agent's inbox in real time " + ui.Faint("[planned]")},
			}},
			{Title: "Observe", Commands: []ui.HelpEntry{
				{Name: "show <name>", Desc: "Show agent details " + ui.Faint("(alias of inspect)")},
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
				{Name: "spwn agent create neo", Desc: ""},
				{Name: "spwn install python --agent neo", Desc: ""},
				{Name: "spwn install skill:paper-reading --agent neo", Desc: ""},
				{Name: "spwn agent neo", Desc: ""},
			}},
		},
		"spwn agent [command]",
		"",
	)
}

// Cmd is the agent command. In the new grammar:
//   - `spwn agent` with no args and no flags -> help
//   - `spwn agent <name>` -> open an interactive session with <name>
//     (boots the world transparently if needed, then attaches a TTY
//     to the runtime inside the container)
//   - `spwn agent --flags ...` -> legacy imperative spawn
//
// Subcommands (new, ls, inspect, ...) resolve first.
var Cmd = &cobra.Command{
	Use:   "agent [name]",
	Short: "Spawn an agent - a living identity that inhabits a world",
	Long: `Spawn an agent into an existing world.

An agent is backed by a Mind - a persistent directory holding its SOUL.md
(purpose, voice, principles), playbooks/, and journal/. The agent survives
after the world is destroyed. Skills are build-time dependencies injected
into /world/skills/; knowledge lives at /world/knowledge/ inside each
world when the manifest opts in via worlds.<name>.knowledge — the path
resolves relative to the project root and is shared across every agent
in that world.`,
	Args: cobra.ArbitraryArgs, // subcommands still resolve first
	Example: `  spwn agent create neo              Create a blank agent
  spwn agent neo                     Open an interactive session with neo
  spwn agent -n neo -u w-abc123      Legacy: spawn named agent into world
  spwn agent --ephemeral "run tests"  Fire-and-forget ephemeral task`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Compose-style shortcut: `spwn agent <name>` -> open an
		// interactive session (boot world if needed, then attach).
		if len(args) == 1 && !cmd.Flags().Changed("name") && !cmd.Flags().Changed("world") &&
			!cmd.Flags().Changed("ephemeral") && !cmd.Flags().Changed("npc") && !cmd.Flags().Changed("import") {
			return runInteractiveSession(cmd, args[0])
		}
		// If no flags set at all, show help
		if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("world") &&
			!cmd.Flags().Changed("ephemeral") && !cmd.Flags().Changed("npc") && !cmd.Flags().Changed("import") {
			return cmd.Help()
		}

		ctx := context.Background()
		s := ui.New()

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
			arc, err := architect.NewFromEnv()
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
			if err := agent.ImportMind(agentName, spawnImport); err != nil {
				return s.FailHint("Import failed", err, "Check that the archive exists and is a valid tar.gz")
			}
			s.Done("Imported agent", agentName)
		}

		arc, err := architect.NewFromEnv()
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

// runInteractiveSession is the `spwn agent <name>` shortcut: open an
// interactive claude session with <name> as if running claude locally
// in your terminal. Boots the world containing <name> transparently
// if it isn't already running (idempotent - no-op when up), then
// attaches a TTY via the existing talk path. The container is left
// running on exit so the next invocation is instant and journal /
// knowledge state keeps accumulating across sessions.
func runInteractiveSession(cmd *cobra.Command, agentName string) error {
	if err := agent.ValidateMind(agentName); err != nil {
		return fmt.Errorf("agent %q not found\n\n  Create one with: spwn agent create %s", agentName, agentName)
	}
	worldName, err := findWorldForAgent(agentName)
	if err != nil {
		return err
	}
	// Bring the world up. composeUpRunE is idempotent: if a container
	// for this world config is already running it prints a notice and
	// returns nil, so we can call it unconditionally without paying
	// the Docker bootstrap cost twice.
	if err := worldcmd.UpCmd.RunE(cmd, []string{worldName}); err != nil {
		return err
	}
	// Attach an interactive session inside the running container.
	return talkCmd.RunE(cmd, []string{agentName})
}
