package world

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/core/agent"
	"spwn.sh/core/gate"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

var (
	spawnConfig      string
	spawnName        string
	spawnAgents      []string
	spawnWorkspaces  []string
	spawnWorld       string
	spawnInteractive bool
	spawnNoAgent     bool
	spawnGate        []string
	spawnRuntime     string
)

func init() {
	Cmd.Flags().StringVarP(&spawnConfig, "config", "c", "", "Named world config (default: default)")
	Cmd.Flags().StringVarP(&spawnName, "name", "n", "", "Display name for the world")
	Cmd.Flags().StringArrayVarP(&spawnAgents, "agent", "a", nil, "Agent name (repeatable; first agent becomes chief in multi-agent worlds)")
	Cmd.Flags().StringArrayVarP(&spawnWorkspaces, "workspace", "w", nil, `Host directory to mount. Repeatable. Forms: "path", "name=path", "name=path:ro". Omit for ephemeral.`)
	Cmd.Flags().StringVarP(&spawnWorld, "world", "u", "", "Explicit path to a YAML config file")
	Cmd.Flags().BoolVarP(&spawnInteractive, "interactive", "i", false, "Attach to agent interactively")
	Cmd.Flags().BoolVar(&spawnNoAgent, "no-agent", false, "Create the world without spawning an agent")
	Cmd.Flags().StringArrayVar(&spawnGate, "gate", nil, `Bridge tool from Host: "source:as:cap1,cap2"`)
	Cmd.Flags().StringVar(&spawnRuntime, "runtime", "claude-code", "Agent runtime (claude-code, pi, codex, opencode, gemini, aider)")

	defaultWorldHelp = Cmd.HelpFunc()
	Cmd.SetHelpFunc(worldHelp)
}

var defaultWorldHelp func(*cobra.Command, []string)

func worldHelp(cmd *cobra.Command, args []string) {
	// Only override help for the parent "world" command itself
	if cmd.Name() != "world" {
		if defaultWorldHelp != nil {
			defaultWorldHelp(cmd, args)
		}
		return
	}

	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ world")+" "+ui.Faint("— ephemeral runtime instances"),
		[]ui.HelpGroup{
			{Title: "Lifecycle", Commands: []ui.HelpEntry{
				{Name: "ls", Desc: "List active worlds"},
				{Name: "show <id>", Desc: "Inspect a running world"},
				{Name: "down <id>", Desc: "Destroy a world " + ui.Faint("(agent survives)")},
				{Name: "rename <id> <name>", Desc: "Rename " + ui.Faint("(empty name clears)")},
			}},
			{Title: "Observe", Commands: []ui.HelpEntry{
				{Name: "logs <id>", Desc: "Stream agent output"},
				{Name: "attach <id>", Desc: "Open interactive shell"},
			}},
			{Title: "Snapshots", Commands: []ui.HelpEntry{
				{Name: "snap <id>", Desc: "Save world state"},
				{Name: "snaps", Desc: "List snapshots"},
				{Name: "restore <snap>", Desc: "Restore from snapshot"},
			}},
			{Title: "Spawn Flags", Commands: []ui.HelpEntry{
				{Name: "-a, --agent <name>", Desc: "Agent " + ui.Faint("(repeatable; first is chief)")},
				{Name: "-w, --workspace <path>", Desc: "Host dir to mount " + ui.Faint("(repeatable)")},
				{Name: "-c, --config <name>", Desc: "Named world config"},
				{Name: "-n, --name <name>", Desc: "Display name"},
				{Name: "-i, --interactive", Desc: "Attach to agent after spawn"},
				{Name: "--no-agent", Desc: "Spawn empty world"},
				{Name: "--runtime <name>", Desc: "Agent runtime " + ui.Faint("(default: claude-code)")},
				{Name: "--gate <spec>", Desc: "Bridge tool from host"},
			}},
			{Title: "Examples", Commands: []ui.HelpEntry{
				{Name: "spwn up --agent neo -w .", Desc: "Spawn neo in current dir"},
				{Name: "spwn up --agent morpheus --agent neo -w .", Desc: ""},
				{Name: "spwn ls", Desc: "See what's running"},
			}},
		},
		"spwn world [flags]\n    spwn world [command]",
		"",
	)
}

// Cmd is the world command — spawns a world when run directly,
// and groups subcommands (list, show, logs, attach, destroy).
var Cmd = &cobra.Command{
	Use:   "world",
	Short: "Spawn a world — an isolated reality for agents",
	Long: `Spawn a world — the Big Bang.

Creates an isolated Docker environment and brings an agent to life inside it.
Uses a named world config from ~/.spwn/worlds/ (default: default.yaml).`,
	Example: `  spwn world --agent neo -w .                  Single agent in current directory
  spwn world --agent morpheus --agent neo -w .  Multi-agent (morpheus is chief)
  spwn world --name "Big Refactor" --agent neo  Ephemeral (no host mount)
  spwn world --no-agent                        Empty world (no agent)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no flags set at all, show help instead of spawning with defaults
		if !cmd.Flags().Changed("config") && !cmd.Flags().Changed("agent") &&
			!cmd.Flags().Changed("workspace") && !cmd.Flags().Changed("world") &&
			!cmd.Flags().Changed("interactive") && !cmd.Flags().Changed("no-agent") &&
			!cmd.Flags().Changed("gate") {
			return cmd.Help()
		}

		ctx := context.Background()
		s := newStepper(cmd)

		s.Blank()
		s.Start("Loading config...")

		// Resolve config name
		configName := "default"
		if spawnConfig != "" {
			configName = spawnConfig
		}

		// Load manifest
		var (
			m   universe.Manifest
			err error
		)
		if spawnWorld != "" {
			m, err = universe.LoadManifestPath(spawnWorld)
		} else {
			m, err = universe.LoadManifest(configName)
		}
		if err != nil {
			return s.FailHint("Config failed", fmt.Errorf("cannot load %q: %w", configName, err),
				"Run \"spwn init\" to create default configs")
		}

		if err := universe.ValidateManifest(m); err != nil {
			return s.FailHint("Config invalid", err, "Check ~/.spwn/worlds/"+configName+".yaml")
		}

		// Parse --gate flags and merge with manifest gates
		for _, g := range spawnGate {
			bridge, err := parseGateFlag(g)
			if err != nil {
				return s.FailHint("Invalid gate", fmt.Errorf("%q: %w", g, err),
					`Expected format: "source:as:cap1,cap2"`)
			}
			m.Gate = append(m.Gate, bridge)
		}

		s.Done("Loaded config", configName)

		// Build spawn opts based on --agent flags
		agentName := ""
		var agents []universe.AgentSpec

		if spawnNoAgent {
			// No agents at all
		} else if len(spawnAgents) == 0 {
			// Auto-create the default agent on-demand when no --agent is given
			agentName = "default"
			agent.InitMind("default") // ignore error if already exists
		} else if len(spawnAgents) == 1 {
			// Single agent mode
			agentName = spawnAgents[0]
		} else {
			// Multi-agent mode: first is chief, rest are workers
			agents = append(agents, universe.AgentSpec{Name: spawnAgents[0], Role: "chief"})
			for _, name := range spawnAgents[1:] {
				agents = append(agents, universe.AgentSpec{Name: name, Role: "worker"})
			}
		}

		s.Start("Connecting to Docker...")
		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return s.FailHint("Docker", err,
				"Start Docker Desktop or OrbStack, then try again")
		}
		s.Done("Docker connected", "")
		s.Start("Validating agent...")

		workspaces, wsErr := parseWorkspaceFlags(spawnWorkspaces)
		if wsErr != nil {
			return s.FailHint("Invalid workspace", wsErr,
				`Expected: "path", "name=path", or "name=path:ro"`)
		}

		result, err := arc.Spawn(ctx, universe.SpawnOpts{
			ConfigName: configName,
			Name:       spawnName,
			AgentName:  agentName,
			Runtime:    spawnRuntime,
			Workspaces: workspaces,
			Manifest:   m,
			Agents:     agents,
			LogWriter:  s.Writer(),
			OnProgress: func(event, detail string) {
				switch event {
				case "mind_validated":
					s.Done("Validated agent", detail)
					s.Start("Mounting mind...")
				case "mind_mounted":
					s.Done("Mounted mind", detail)
					s.Start("Resolving image...")
				case "image_building":
					s.Done("Image not cached", detail)
					s.Start("Building image (first run — may take a few minutes)...")
				case "image_built":
					s.Done("Built image", detail)
					s.Start("Resolving credentials...")
				case "image_ready":
					s.Done("Image ready", detail)
					s.Start("Resolving credentials...")
				case "credentials_resolved":
					s.Done("Credentials", detail)
					s.Start("Creating container...")
				case "container_created":
					s.Done("Created container", detail)
					s.Start("Probing tools...")
				case "gates_bridged":
					s.Done("Bridged gates", detail)
				case "tools_probed":
					s.Done("Probed tools", detail)
					s.Start("Generating physics...")
				case "faculties_generated":
					s.Done("Generated physics", detail)
				}
			},
		})
		if err != nil {
			return s.FailHint("Spawn failed", err, spawnHint(err, agentName, agents))
		}

		u := result.Universe

		// Show non-fatal warnings
		for _, w := range result.Warnings {
			s.Warn("Warning", w)
		}

		// Spawn agents
		if len(agents) > 0 {
			// Multi-agent mode
			s.Start("Spawning colony...")
			if err := arc.SpawnAgents(ctx, u.ID, agents); err != nil {
				return s.FailHint("Colony failed", err, "Check agent logs with \"spwn logs "+u.ID+"\"")
			}
			s.Done("Colony spawned", fmt.Sprintf("%d agent(s)", len(agents)))
		} else if agentName != "" {
			// Single-agent mode
			if spawnInteractive {
				s.Blank()
				s.Success("Agent is alive.")
				s.Blank()
				if err := arc.SpawnAgent(ctx, u.ID, agentName); err != nil {
					return err
				}
			} else {
				s.Start("Spawning agent...")
				if err := arc.SpawnAgentDetached(ctx, u.ID, agentName); err != nil {
					return s.FailHint("Agent failed", err, "Check agent logs with \"spwn logs "+u.ID+"\"")
				}
				s.Done("Agent spawned", "detached")
			}
		}

		j, _ := cmd.Flags().GetBool("json")
		if j {
			data, _ := json.MarshalIndent(u, "", "  ")
			fmt.Println(string(data))
		} else if spawnNoAgent || !spawnInteractive {
			s.Blank()
			if spawnNoAgent {
				s.Success("World spawned.")
			} else {
				s.Success("Agent is alive.")
			}
			s.Blank()
			s.Info("World:", u.ID)
			if u.AgentID != "" {
				s.Info("Agent:", u.AgentID)
			}
			s.Info("Status:", string(u.Status))
			if agentName != "" {
				s.Blank()
				fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", ui.Faint(fmt.Sprintf("Talk: spwn talk %s", agentName)))
				fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", ui.Faint(fmt.Sprintf("Logs: spwn logs %s", u.ID)))
			}
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

// dockerHint wraps a NewArchitectFromEnv error with a user-friendly hint
// when Docker is not running.
func dockerHint(err error) error {
	if strings.Contains(err.Error(), "cannot connect to Docker") {
		return fmt.Errorf("Docker is not running")
	}
	return err
}

// parseGateFlag parses "source:as:cap1,cap2" into a GateBridge.
func parseGateFlag(s string) (gate.Bridge, error) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) < 2 {
		return gate.Bridge{}, fmt.Errorf("expected format \"source:as[:cap1,cap2]\", got %q", s)
	}

	bridge := gate.Bridge{
		Source: parts[0],
		As:     parts[1],
	}

	if len(parts) == 3 && parts[2] != "" {
		bridge.Capabilities = strings.Split(parts[2], ",")
	}

	return bridge, nil
}

// parseWorkspaceFlags parses a list of "-w" values into universe.Workspace.
// Accepted forms:
//   "/host/path"             → {Name: "default" or "wN", Path: "/host/path"}
//   "name=/host/path"        → {Name: "name", Path: "/host/path"}
//   "name=/host/path:ro"     → read-only
// Empty input returns a nil slice (ephemeral world — no mounts).
func parseWorkspaceFlags(flags []string) ([]universe.Workspace, error) {
	if len(flags) == 0 {
		return nil, nil
	}
	result := make([]universe.Workspace, 0, len(flags))
	for i, raw := range flags {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		// Strip optional :ro suffix. Be careful not to confuse with a colon inside the path
		// — we only accept :ro at the very end.
		readOnly := false
		if strings.HasSuffix(raw, ":ro") {
			readOnly = true
			raw = strings.TrimSuffix(raw, ":ro")
		}

		// name=path or bare path
		name := ""
		path := raw
		if eq := strings.Index(raw, "="); eq > 0 {
			name = strings.TrimSpace(raw[:eq])
			path = strings.TrimSpace(raw[eq+1:])
		}
		if path == "" {
			return nil, fmt.Errorf("workspace #%d has empty path", i+1)
		}
		if name == "" {
			if i == 0 && len(flags) == 1 {
				name = "default"
			} else {
				name = fmt.Sprintf("w%d", i)
			}
		}
		result = append(result, universe.Workspace{Name: name, Path: path, ReadOnly: readOnly})
	}
	return result, nil
}

// spawnHint returns an actionable hint for common spawn errors.
func spawnHint(err error, agentName string, agents []universe.AgentSpec) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not found") && strings.Contains(msg, "agent"):
		name := agentName
		if len(agents) > 0 {
			name = agents[0].Name
		}
		return fmt.Sprintf("Run \"spwn agent new %s\" to create the agent first", name)
	case strings.Contains(msg, "missing the core"):
		name := agentName
		if len(agents) > 0 {
			name = agents[0].Name
		}
		return fmt.Sprintf("Run \"spwn agent new %s\" to set up the agent layers", name)
	case strings.Contains(msg, "image") && strings.Contains(msg, "not found"):
		return "Run \"spwn doctor\" to check your Docker images"
	case strings.Contains(msg, "workspace") && strings.Contains(msg, "not found"):
		return "Check that the -w path exists"
	case strings.Contains(msg, "tool"):
		return "Remove the missing tool from your world config, or add it to the base image"
	case strings.Contains(msg, "docker") || strings.Contains(msg, "Docker"):
		return "Start Docker Desktop or OrbStack, then try again"
	default:
		return "Run \"spwn doctor\" to diagnose the issue"
	}
}
