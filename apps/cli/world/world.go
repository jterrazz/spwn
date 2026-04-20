package world

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/architect"
	"spwn.sh/packages/world"
)

var (
	spawnConfig       string
	spawnName         string
	spawnAgents       []string
	spawnWorkspaces   []string
	spawnWorld        string
	spawnInteractive  bool
	spawnForceRebuild bool
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
	c.Flags().BoolVar(&spawnForceRebuild, "force-rebuild", false, "Ignore the image cache and rebuild the world image from scratch")
}

func worldHelp(cmd *cobra.Command, args []string) {
	// Only override help for the parent "world" command itself
	if cmd.Name() != "world" {
		ui.MinimalHelp(cmd, args)
		return
	}

	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ world")+" "+ui.Faint("- ephemeral runtime instances"),
		[]ui.HelpGroup{
			{Title: "Lifecycle", Commands: []ui.HelpEntry{
				{Name: "up", Desc: "Spawn a world " + ui.Faint("(see Spawn Flags below)")},
				{Name: "ls", Desc: "List active worlds"},
				{Name: "down <id>", Desc: "Destroy a world " + ui.Faint("(agent survives)")},
				{Name: "rename <id> <name>", Desc: "Rename " + ui.Faint("(empty name clears)")},
			}},
			{Title: "Observe", Commands: []ui.HelpEntry{
				{Name: "inspect <id>", Desc: "Inspect a running world's composition and state"},
				{Name: "logs <id>", Desc: "Show event log for a world"},
				{Name: "enter <id>", Desc: "Open an interactive shell inside a world"},
			}},
			{Title: "Snapshots", Commands: []ui.HelpEntry{
				{Name: "snap save <id>", Desc: "Save world state"},
				{Name: "snap ls", Desc: "List snapshots"},
				{Name: "snap restore <snap-id>", Desc: "Rollback to a snapshot"},
				{Name: "snap rm <snap-id>", Desc: "Remove a snapshot"},
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
// `spwn world <name>` is a compose-style shortcut that starts the
// named world from spwn.yaml (equivalent to `spwn world start <name>`).
// `spwn world --agent neo` (with at least one spawn flag) also still
// acts as a shortcut for `spwn world up --agent neo`.
var Cmd = &cobra.Command{
	Use:   "world [name]",
	Short: "Manage worlds - ephemeral runtime instances for agents",
	Args:  cobra.ArbitraryArgs, // we inspect args manually so subcommands still resolve
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no spawn flags and no positional arg, just render help.
		spawnFlagNames := []string{"config", "name", "agent", "workspace", "world", "interactive"}
		anySet := false
		for _, n := range spawnFlagNames {
			if cmd.Flags().Changed(n) {
				anySet = true
				break
			}
		}
		if !anySet && len(args) == 0 {
			return cmd.Help()
		}
		// `spwn world <name>` with no flags -> start the named world.
		if !anySet && len(args) == 1 {
			return composeUpRunE(cmd, args)
		}
		return spawnRunE(cmd, args)
	},
}

// upCmd is `spwn world up` - the canonical spawn verb. The top-level
// `spwn up` alias in aliases.go just reuses upCmd.RunE. When invoked
// with no positional arg inside a spwn project, it brings up every
// world declared in spwn.yaml compose-style.
var upCmd = &cobra.Command{
	Use:   "up [name]",
	Short: "Spawn a world - an isolated reality for agents",
	Long: `Spawn a world - the Big Bang.

Inside a spwn project:
  spwn up             brings up every world declared in spwn.yaml
  spwn up <name>      brings up a specific world from spwn.yaml

Outside a project, the legacy global-mode flags still work and spawn
a one-off world from ~/.spwn/worlds/<config>.yaml.`,
	Args: cobra.MaximumNArgs(1),
	Example: `  spwn up                                          Bring up every world in spwn.yaml
  spwn up neo                                      Start the "neo" world
  spwn world up --agent neo -w .                  Single agent in current dir
  spwn world up --agent morpheus --agent neo -w .  Multi-agent (morpheus is chief)`,
	RunE: composeUpRunE,
}

// composeUpRunE is the top-level entry point for `spwn up`.
// With no args and a project active, it iterates every world in
// spwn.yaml. With one positional arg, it forwards to spawnRunE.
func composeUpRunE(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return spawnRunE(cmd, args)
	}
	// No positional arg: try compose-style iteration if a project is loaded.
	p, err := loadProject()
	if err != nil || p == nil || len(p.Manifest.Worlds) == 0 {
		return spawnRunE(cmd, args)
	}
	names := sortedWorldNames(p)
	if len(names) == 1 {
		// Single-world project: behave identically to `spwn up <name>`.
		return spawnRunE(cmd, []string{names[0]})
	}
	// Multi-world: iterate. Reset flag state between iterations so each
	// world gets its own resolved agents/workspaces from the inline map.
	for _, name := range names {
		resetSpawnFlags(cmd)
		if err := spawnRunE(cmd, []string{name}); err != nil {
			return err
		}
	}
	return nil
}

// resetSpawnFlags zeroes out the flag-backed globals between successive
// spawnRunE calls in compose-style iteration. Without this, the second
// world would inherit the first world's agents/workspaces.
func resetSpawnFlags(cmd *cobra.Command) {
	spawnConfig = ""
	spawnName = ""
	spawnAgents = nil
	spawnWorkspaces = nil
}

func spawnRunE(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	s := ui.New()

	s.Blank()

	// A positional name arg always refers to a world entry in spwn.yaml.
	// `spwn up foo` = start world `foo`. This is the compose-style path.
	positionalName := ""
	if len(args) > 0 {
		positionalName = args[0]
	}

	// Idempotency guard: if a world with the same config name is
	// already running, treat `spwn up` as a no-op. This matches docker
	// compose semantics and prevents the duplicate-container bug where
	// two invocations of `spwn up` would spawn two containers for the
	// same world. See finding #7.
	if positionalName != "" {
		if existing := findRunningWorldByConfig(ctx, positionalName); existing != nil {
			s.Blank()
			s.Success(fmt.Sprintf("world %q is already running (%s)", positionalName, existing.ID))
			s.Blank()
			s.Info("Enter:", fmt.Sprintf("spwn world enter %s", existing.ID))
			s.Blank()
			return nil
		}
	}

	// Per-world spawn lock: prevents two concurrent `spwn up` calls
	// from racing past the idempotency check above and both creating
	// containers. The lock lives under the project's local state dir
	// and is released in a defer. See finding #8.
	if positionalName != "" {
		unlock, lockErr := acquireUpLock(positionalName)
		if lockErr != nil {
			return s.FailHint("Up in progress",
				fmt.Errorf("another `spwn up` is spawning world %q", positionalName),
				"Wait for the other run to finish, or remove "+lockErr.Error()+" if it is stale")
		}
		defer unlock()
	}

	// If we're inside a spwn project, prefer the inline spwn.yaml world
	// over the legacy ~/.spwn/worlds/<name>.yaml file. When a project is
	// active we synthesize a world.Manifest straight from the inline
	// map, so no stub file needs to exist on disk.
	pw, projectErr := applyProjectDefaults(cmd, positionalName)
	if projectErr != nil {
		return s.FailHint("Project", projectErr, "Check spwn.yaml or pick an existing world name")
	}

	s.Start("Loading config...")

	configName := "default"
	if spawnConfig != "" {
		configName = spawnConfig
	}

	var (
		m   world.Manifest
		err error
	)
	switch {
	case pw != nil && spawnWorld == "":
		// Project mode: manifest is fully synthesized from spwn.yaml.
		m = pw.Manifest
		configName = pw.Name
	case spawnWorld != "":
		m, err = world.LoadManifestPath(spawnWorld)
	default:
		m, err = world.LoadManifest(configName)
	}
	if err != nil {
		return s.FailHint("Config failed", fmt.Errorf("cannot load %q: %w", configName, err),
			"Run \"spwn init\" to create default configs")
	}

	if err := world.ValidateManifest(m); err != nil {
		return s.FailHint("Config invalid", err, "Check ~/.spwn/worlds/"+configName+".yaml")
	}

	s.Done("Loaded config", configName)

	// Build spawn opts based on --agent flags. No --agent = empty world.
	agentName := ""
	var agents []architect.AgentSpec

	switch len(spawnAgents) {
	case 0:
		// empty world
	case 1:
		agentName = spawnAgents[0]
	default:
		// Multi-agent mode: first is chief, rest are workers
		agents = append(agents, architect.AgentSpec{Name: spawnAgents[0], Role: "chief"})
		for _, name := range spawnAgents[1:] {
			agents = append(agents, architect.AgentSpec{Name: name, Role: "worker"})
		}
	}

	s.Start("Connecting to Docker...")
	arc, err := architect.NewFromEnv()
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

	// BuildProgressWriter parses the Docker build stream for
	// `Step N/M :` lines and updates the spinner label in place.
	// Callers see "Building image [5/12] Installing packages"
	// instead of a mystery spinner during the long image build.
	buildProgress := s.BuildProgressWriter("Building image")

	// Knowledge path resolution: the CLI is the ONLY layer that turns
	// the manifest's project-relative `worlds.<name>.knowledge` value
	// into an absolute host path. When no inline world is active
	// (pw == nil, legacy global-mode spawn), knowledge stays empty and
	// the spawn pipeline drops the bind mount.
	knowledge := ""
	runtimeName := ""
	if pw != nil {
		knowledge = pw.Knowledge
		runtimeName = pw.Runtime
	}

	result, err := arc.Spawn(ctx, architect.SpawnOpts{
		ConfigName:   configName,
		Name:         spawnName,
		AgentName:    agentName,
		Workspaces:   workspaces,
		Manifest:     m,
		Agents:       agents,
		ForceRebuild: spawnForceRebuild,
		Knowledge:    knowledge,
		RuntimeName:  runtimeName,
		LogWriter:    buildProgress,
		OnProgress: func(event, detail string) {
			switch event {
			case "mind_validated":
				s.Done("Validated agent", detail)
			case "tools_resolved":
				// Surface the resolved tool list right after the
				// agent validation so users can see exactly what's
				// about to flow into the world image before the
				// spinner starts.
				if detail != "" {
					s.Info("Tools", detail)
				}
				s.Start("Resolving compile...")
			case "image_resolving":
				s.UpdateLabel("Resolving image (checking cache)...")
			case "image_building":
				// Flip the spinner label to "Building image" so
				// BuildProgressWriter's in-place step updates land
				// on the right base. Each docker `Step N/M :`
				// line rewrites this label with the current action.
				s.UpdateLabel("Building image")
			case "image_built":
				s.Done("Built image", detail)
				s.Start("Resolving credentials...")
			case "image_cached":
				s.Done("Image cached", detail)
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

	u := result.World

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
			fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", ui.Faint(fmt.Sprintf("Talk: spwn agent talk %s", agentName)))
			fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", ui.Faint(fmt.Sprintf("Logs: spwn world logs %s", u.ID)))
		}
	}

	return nil
}

// dockerHint wraps a NewArchitectFromEnv error with a user-friendly hint
// when Docker is not running.
func dockerHint(err error) error {
	if strings.Contains(err.Error(), "cannot connect to Docker") {
		return fmt.Errorf("Docker is not running")
	}
	return err
}

// parseWorkspaceFlags parses a list of "-w" values into world.Workspace.
// Accepted forms:
//
//	"path"           → auto-named workspace<N>
//	"name=path"      → explicit name
//	"name=path:ro"   → same, read-only
//
// Users never write container platform. The container-side layout is
// decided by the spawn pipeline (currently /workspaces/<name>/).
//
// Empty input returns a nil slice (ephemeral world — no mounts).
func parseWorkspaceFlags(flags []string) ([]world.Workspace, error) {
	if len(flags) == 0 {
		return nil, nil
	}
	result := make([]world.Workspace, 0, len(flags))
	for i, raw := range flags {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		// Strip optional :ro suffix. Only accept :ro at the very end —
		// anything else in the entry is path content.
		readOnly := false
		if strings.HasSuffix(raw, ":ro") {
			readOnly = true
			raw = strings.TrimSuffix(raw, ":ro")
		}

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
			name = fmt.Sprintf("workspace%d", i)
		}
		result = append(result, world.Workspace{Name: name, Path: path, ReadOnly: readOnly})
	}
	return result, nil
}

// spawnHint returns an actionable hint for common spawn errors.
func spawnHint(err error, agentName string, agents []architect.AgentSpec) string {
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

// applyProjectDefaults resolves the current spwn.yaml (if any) and
// overlays its inline world entry on the flag state. It returns the
// projectWorld used so callers can build from its synthesized manifest
// directly. Silently no-ops (returning nil, nil) when no project is
// active, so the legacy global-mode CLI still works unchanged.
//
// If requestedName is non-empty and the project has no such world,
// this returns a hard error - we refuse to silently fall through to
// the legacy config path because the user asked for a specific name.
func applyProjectDefaults(cmd *cobra.Command, requestedName string) (*projectWorld, error) {
	p, err := loadProject()
	if err != nil || p == nil {
		return nil, nil
	}
	pw, err := resolveProjectWorld(p, requestedName)
	if err != nil {
		// Named world explicitly requested but missing -> hard error.
		if requestedName != "" {
			return nil, err
		}
		return nil, nil
	}

	if !cmd.Flags().Changed("config") && !cmd.Flags().Changed("world") && spawnConfig == "" {
		spawnConfig = pw.Name
	}
	if !cmd.Flags().Changed("agent") && len(spawnAgents) == 0 {
		spawnAgents = append(spawnAgents, pw.Agents...)
	}
	if !cmd.Flags().Changed("workspace") && len(spawnWorkspaces) == 0 && len(pw.Workspaces) > 0 {
		spawnWorkspaces = append(spawnWorkspaces, pw.Workspaces...)
	}
	return pw, nil
}

