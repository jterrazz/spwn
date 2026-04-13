package world

import (
	"context"
	"fmt"
	"strings"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/universe"
	"github.com/spf13/cobra"
)

var (
	spawnConfig      string
	spawnName        string
	spawnAgents      []string
	spawnWorkspaces  []string
	spawnWorld       string
	spawnInteractive bool
)

func init() {
	// Both the bare `spwn world` parent and the explicit `spwn world up`
	// subcommand carry the spawn flag set. `spwn world` with at least one
	// spawn flag still spawns (the original ergonomic shortcut); `spwn world`
	// with no args drops to help.
	registerSpawnFlags(Cmd)
	registerSpawnFlags(upCmd)

	Cmd.AddCommand(upCmd)
	Cmd.SetHelpFunc(worldHelp)
}

// registerSpawnFlags attaches the spawn flag set to a cobra command. It's
// reused by `spwn world up`, the top-level `spwn up` alias, and any future
// shortcut that needs the same surface.
func registerSpawnFlags(c *cobra.Command) {
	c.Flags().StringVarP(&spawnConfig, "config", "c", "", "Named world config (default: default)")
	c.Flags().StringVarP(&spawnName, "name", "n", "", "Display name for the world")
	c.Flags().StringArrayVarP(&spawnAgents, "agent", "a", nil, "Agent name (repeatable; first agent becomes chief in multi-agent worlds)")
	c.Flags().StringArrayVarP(&spawnWorkspaces, "workspace", "w", nil, `Host directory to mount. Repeatable. Forms: "path", "name=path", "name=path:ro". Omit for ephemeral.`)
	c.Flags().StringVarP(&spawnWorld, "world", "u", "", "Explicit path to a YAML config file")
	c.Flags().BoolVarP(&spawnInteractive, "interactive", "i", false, "Drop into the agent's session after spawn")
}

func worldHelp(cmd *cobra.Command, args []string) {
	// Only override help for the parent "world" command itself
	if cmd.Name() != "world" {
		ui.MinimalHelp(cmd, args)
		return
	}

	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ world")+" "+ui.Faint("— ephemeral runtime instances"),
		[]ui.HelpGroup{
			{Title: "Lifecycle", Commands: []ui.HelpEntry{
				{Name: "up", Desc: "Spawn a world " + ui.Faint("(see Spawn Flags below)")},
				{Name: "ls", Desc: "List active worlds"},
				{Name: "inspect <id>", Desc: "Inspect a running world"},
				{Name: "down <id>", Desc: "Destroy a world " + ui.Faint("(agent survives)")},
				{Name: "rename <id> <name>", Desc: "Rename " + ui.Faint("(empty name clears)")},
			}},
			{Title: "Observe", Commands: []ui.HelpEntry{
				{Name: "logs <id>", Desc: "Show event log for a world"},
				{Name: "enter <id>", Desc: "Open an interactive shell inside a world"},
			}},
			{Title: "Knowledge", Commands: []ui.HelpEntry{
				{Name: "knowledge ls <id>", Desc: "List a world's knowledge files"},
				{Name: "knowledge show <id> <path>", Desc: "Read a knowledge file"},
			}},
			{Title: "Spawn Flags", Commands: []ui.HelpEntry{
				{Name: "-a, --agent <name>", Desc: "Agent " + ui.Faint("(repeatable; first is chief)")},
				{Name: "-w, --workspace <path>", Desc: "Host dir to mount " + ui.Faint("(repeatable)")},
				{Name: "-c, --config <name>", Desc: "Named world config"},
				{Name: "-n, --name <name>", Desc: "Display name"},
				{Name: "-i, --interactive", Desc: "Drop into the agent's session after spawn"},
			}},
			{Title: "Examples", Commands: []ui.HelpEntry{
				{Name: "spwn up --agent neo -w .", Desc: "Spawn neo in current dir"},
				{Name: "spwn up --agent morpheus --agent neo -w .", Desc: ""},
				{Name: "spwn ls", Desc: "See what's running"},
			}},
		},
		"spwn world [command]",
		"",
	)
}

// Cmd is the parent for `spwn world …`. Bare `spwn world` shows help.
// `spwn world --agent neo` (with at least one spawn flag) acts as a
// shortcut for `spwn world up --agent neo`.
var Cmd = &cobra.Command{
	Use:   "world",
	Short: "Manage worlds — ephemeral runtime instances for agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no spawn flags are set, just render the grouped help.
		spawnFlagNames := []string{"config", "name", "agent", "workspace", "world", "interactive"}
		anySet := false
		for _, n := range spawnFlagNames {
			if cmd.Flags().Changed(n) {
				anySet = true
				break
			}
		}
		if !anySet {
			return cmd.Help()
		}
		return spawnRunE(cmd, args)
	},
}

// upCmd is `spwn world up` — the canonical spawn verb. The top-level
// `spwn up` alias in aliases.go just reuses upCmd.RunE.
var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Spawn a world — an isolated reality for agents",
	Long: `Spawn a world — the Big Bang.

Creates an isolated Docker environment. Pass --agent (repeatable) to bring
agents to life inside it. Without any --agent flag, the world spawns empty.`,
	Example: `  spwn world up --agent neo -w .                  Single agent in current dir
  spwn world up --agent morpheus --agent neo -w .  Multi-agent (morpheus is chief)
  spwn world up --name "Big Refactor" --agent neo  Ephemeral (no host mount)
  spwn world up                                    Empty world (no agent)`,
	RunE: spawnRunE,
}

func spawnRunE(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	s := newStepper(cmd)

	s.Blank()
	s.Start("Loading config...")

	configName := "default"
	if spawnConfig != "" {
		configName = spawnConfig
	}

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

	s.Done("Loaded config", configName)

	// Build spawn opts based on --agent flags. No --agent = empty world.
	agentName := ""
	var agents []universe.AgentSpec

	switch len(spawnAgents) {
	case 0:
		// empty world
	case 1:
		agentName = spawnAgents[0]
	default:
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

	workspaces, wsErr := parseWorkspaceFlags(spawnWorkspaces)
	if wsErr != nil {
		return s.FailHint("Invalid workspace", wsErr,
			`Expected: "path", "name=path", or "name=path:ro"`)
	}

	if agentName != "" || len(agents) > 0 {
		s.Start("Validating agent...")
	}

	result, err := arc.Spawn(ctx, universe.SpawnOpts{
		ConfigName: configName,
		Name:       spawnName,
		AgentName:  agentName,
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
			return s.FailHint("Colony failed", err, "Check events with \"spwn world logs "+u.ID+"\"")
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
				return s.FailHint("Agent failed", err, "Check events with \"spwn world logs "+u.ID+"\"")
			}
			s.Done("Agent spawned", "detached")
		}
	}

	if !spawnInteractive {
		s.Blank()
		if agentName == "" && len(agents) == 0 {
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
			fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", ui.Faint(fmt.Sprintf("Logs: spwn world logs %s", u.ID)))
		}
	}

	return nil
}

func newStepper(cmd *cobra.Command) *ui.Stepper {
	return ui.New()
}

// dockerHint wraps a NewArchitectFromEnv error with a user-friendly hint
// when Docker is not running.
func dockerHint(err error) error {
	if strings.Contains(err.Error(), "cannot connect to Docker") {
		return fmt.Errorf("Docker is not running")
	}
	return err
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
		return "Check that the image exists locally or pull it manually"
	case strings.Contains(msg, "workspace") && strings.Contains(msg, "not found"):
		return "Check that the -w path exists"
	case strings.Contains(msg, "tool"):
		return "Remove the missing tool from your world config, or add it to the base image"
	case strings.Contains(msg, "docker") || strings.Contains(msg, "Docker"):
		return "Start Docker Desktop or OrbStack, then try again"
	default:
		return ""
	}
}
