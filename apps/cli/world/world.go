package world

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/architect"
	"spwn.sh/packages/authentication"
	"spwn.sh/packages/runtimes"
	"spwn.sh/packages/world"
)

var (
	spawnConfig       string
	spawnName         string
	spawnAgents       []string
	spawnWorkspaces   []string
	spawnWorld        string
	spawnBackend      string
	spawnInteractive  bool
	spawnForceRebuild bool
)

func init() {
	registerSpawnFlags(Cmd)
	registerSpawnFlags(upCmd)

	Cmd.AddCommand(upCmd)
	Cmd.SetHelpFunc(worldHelp)
}

// SetSpawnBackend overrides the runtime backend used by the next
// Spawn. Intended for callers in sibling packages (apps/cli/agent)
// That delegate to UpCmd.RunE without going through cobra flag parsing
// On the world command itself — see agent.runInteractiveSession where
// `spwn agent neo --backend codex` routes the backend through here
// Before handing off to UpCmd.
func SetSpawnBackend(backend string) {
	spawnBackend = backend
}

func registerSpawnFlags(c *cobra.Command) {
	c.Flags().StringVarP(&spawnConfig, "config", "c", "", "Named world config (default: default)")
	c.Flags().StringVarP(&spawnName, "name", "n", "", "Display name for the world")
	c.Flags().StringArrayVarP(&spawnAgents, "agent", "a", nil, "Agent name (repeatable; first agent becomes chief in multi-agent worlds)")
	c.Flags().StringArrayVarP(&spawnWorkspaces, "workspace", "w", nil, `Host directory to mount. Repeatable. Forms: "path", "name=path", "name=path:ro". Omit for ephemeral.`)
	c.Flags().StringVarP(&spawnWorld, "world", "u", "", "Explicit path to a YAML config file")
	c.Flags().StringVar(&spawnBackend, "backend", "", "Override the runtime backend for this spawn (e.g. claude-code, codex). Skips the auth-state auto-resolver.")
	c.Flags().BoolVarP(&spawnInteractive, "interactive", "i", false, "Drop into the agent's session after spawn")
	c.Flags().BoolVar(&spawnForceRebuild, "force-rebuild", false, "Ignore the image cache and rebuild the world image from scratch")
}

func worldHelp(cmd *cobra.Command, args []string) {
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
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
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
		if !anySet && len(args) == 1 {
			return composeUpRunE(cmd, args)
		}
		return spawnRunE(cmd, args)
	},
}

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

func composeUpRunE(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return spawnRunE(cmd, args)
	}
	p, err := loadProject()
	if err != nil || p == nil || len(p.Manifest.Worlds) == 0 {
		return spawnRunE(cmd, args)
	}
	names := sortedWorldNames(p)
	if len(names) == 1 {
		return spawnRunE(cmd, []string{names[0]})
	}
	for _, name := range names {
		resetSpawnFlags(cmd)
		if err := spawnRunE(cmd, []string{name}); err != nil {
			return err
		}
	}
	return nil
}

func resetSpawnFlags(cmd *cobra.Command) {
	spawnConfig = ""
	spawnName = ""
	spawnAgents = nil
	spawnWorkspaces = nil
	spawnBackend = ""
}

func spawnRunE(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	s := ui.New()

	positionalName := ""
	if len(args) > 0 {
		positionalName = args[0]
	}

	if positionalName != "" {
		if existing := findRunningWorldByConfig(ctx, positionalName); existing != nil {
			wanted := resolveWantedRuntime(positionalName, spawnBackend)
			if wanted != "" && existing.Runtime != "" && existing.Runtime != wanted {
				s.Blank()
				s.Warn("Runtime drift",
					fmt.Sprintf("world %q was spawned with %q; now wants %q — rebuilding",
						positionalName, existing.Runtime, wanted))
				s.Blank()
				if arc, derr := architect.NewFromEnv(); derr == nil {
					_, _ = arc.Destroy(ctx, existing.ID)
				}
			} else {
				s.Blank()
				s.Success(fmt.Sprintf("world %q is already running (%s)", positionalName, existing.ID))
				s.Blank()
				s.Info("Enter:", fmt.Sprintf("spwn world enter %s", existing.ID))
				s.Blank()
				return nil
			}
		}
	}

	if positionalName != "" {
		unlock, lockErr := acquireUpLock(positionalName)
		if lockErr != nil {
			return s.FailHint("Up in progress",
				fmt.Errorf("another `spwn up` is spawning world %q", positionalName),
				"Wait for the other run to finish, or remove "+lockErr.Error()+" if it is stale")
		}
		defer unlock()
	}

	pw, projectErr := applyProjectDefaults(cmd, positionalName, spawnBackend)
	if projectErr != nil {
		return s.FailHint("Project", projectErr, "Check spwn.yaml or pick an existing world name")
	}

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

	agentName := ""
	var agents []architect.AgentSpec

	switch len(spawnAgents) {
	case 0:
	case 1:
		agentName = spawnAgents[0]
	default:
		agents = append(agents, architect.AgentSpec{Name: spawnAgents[0], Role: "chief"})
		for _, name := range spawnAgents[1:] {
			agents = append(agents, architect.AgentSpec{Name: name, Role: "worker"})
		}
	}

	workspaces, wsErr := parseWorkspaceFlags(spawnWorkspaces)
	if wsErr != nil {
		return s.FailHint("Invalid workspace", wsErr,
			`Expected: "path", "name=path", or "name=path:ro"`)
	}

	s.Blank()
	switch {
	case agentName != "":
		s.Hero("⬡", "Waking "+agentName)
	case len(agents) > 0:
		s.Hero("⬡", fmt.Sprintf("Waking %d agents", len(agents)))
	default:
		s.Hero("⬡", "Waking "+configName)
	}

	s.Phase("Environment")
	s.SubStart("Connecting to Docker…")
	arc, err := architect.NewFromEnv()
	if err != nil {
		return s.SubFailHint("Docker", err,
			"Start Docker Desktop or OrbStack, then try again")
	}
	s.SubDone("Docker", "connected")

	knowledge := ""
	runtimeName := ""
	if pw != nil {
		knowledge = pw.Knowledge
		runtimeName = pw.Runtime
	}

	if providerName := runtimeAuthProvider(runtimeName); providerName != "" {
		s.SubStart("Checking credentials…")
		ctxCred, cancelCred := context.WithTimeout(ctx, 4*time.Second)
		p := authentication.Provider(providerName)
		cred := authentication.Resolve(p)
		status := authentication.ValidateWithCache(ctxCred, cred, 5*time.Minute)
		cancelCred()
		if status == nil || !status.Connected {
			return s.SubFailHint("Credentials",
				authentication.CredentialError(p, cred, status),
				authentication.CredentialFixHint(p, cred))
		}
		s.SubDone("Credentials", credentialDetail(cred))
	}

	buildProgress := s.BuildProgressWriter()

	const (
		phaseEnv   = 0
		phaseImage = 1
		phaseSpawn = 2
	)
	phase := phaseEnv

	enterImage := func() {
		if phase >= phaseImage {
			return
		}
		s.PhaseBreak()
		s.Phase("Image")
		phase = phaseImage
	}
	enterSpawn := func() {
		enterImage()
		if phase >= phaseSpawn {
			return
		}
		s.PhaseBreak()
		s.Phase("Spawn")
		phase = phaseSpawn
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
				s.SubDone("Agent", detail)
			case "tools_resolved":
				enterImage()
				if detail != "" {
					s.TreeLine("tools: "+ui.Faint(detail), "")
				}
			case "image_resolving", "image_building":
			case "image_built":
				buildProgress.CompleteCurrent()
				s.SubDone("Built", detail)
			case "image_cached":
				buildProgress.CompleteCurrent()
				s.SubDone("Cached", detail)
			case "image_ready":
				buildProgress.CompleteCurrent()
				s.SubDone("Ready", detail)
			case "credentials_resolved":
				enterSpawn()
			case "container_created":
				enterSpawn()
				s.SubDone("Created container", detail)
			case "tools_probed":
				enterSpawn()
				s.SubDone("Tools verified", detail)
			case "world_state_written":
				enterSpawn()
				s.SubDone("Prompt", detail)
			}
		},
	})
	if err != nil {
		buildProgress.CompleteCurrent()
		return s.SubFailHint("Spawn failed", err, spawnHint(err, agentName, agents))
	}

	u := result.World

	for _, w := range result.Warnings {
		s.Warn("Warning", w)
	}

	if len(agents) > 0 {
		enterSpawn()
		s.SubStart("Spawning colony…")
		if err := arc.SpawnAgents(ctx, u.ID, agents); err != nil {
			return s.SubFailHint("Colony failed", err, "Check events with \"spwn world logs "+u.ID+"\"")
		}
		s.SubDone("Colony spawned", fmt.Sprintf("%d agent(s)", len(agents)))
	} else if agentName != "" {
		if spawnInteractive {
			if err := arc.SpawnAgent(ctx, u.ID, agentName); err != nil {
				return err
			}
			return nil
		}
		enterSpawn()
		s.SubStart("Spawning agent…")
		if err := arc.SpawnAgentDetached(ctx, u.ID, agentName); err != nil {
			return s.SubFailHint("Agent failed", err, "Check events with \"spwn world logs "+u.ID+"\"")
		}
		s.SubDone("Agent spawned", "detached")
	}

	s.Blank()
	switch {
	case agentName == "" && len(agents) == 0:
		s.Success("World spawned.")
	default:
		s.Success("Agent is alive.")
	}
	s.Blank()
	s.Info("World:", u.ID)
	if u.AgentID != "" {
		s.Info("Agent:", u.AgentID)
	}
	if agentName != "" {
		s.Blank()
		fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", ui.Faint("Talk: spwn agent talk "+agentName))
		fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", ui.Faint("Logs: spwn world logs "+u.ID))
	}
	s.Blank()

	return nil
}

func resolveWantedRuntime(worldName, backendOverride string) string {
	p, err := loadProject()
	if err != nil || p == nil {
		return ""
	}
	pw, err := resolveProjectWorld(p, worldName, backendOverride)
	if err != nil || pw == nil {
		return ""
	}
	return pw.Runtime
}

func runtimeAuthProvider(runtimeName string) string {
	if runtimeName == "" {
		runtimeName = "claude-code"
	}
	adapter, ok := runtimes.Get(runtimeName)
	if !ok {
		return ""
	}
	return adapter.DefaultProvider
}

func credentialDetail(cred *authentication.Credential) string {
	if cred == nil {
		return ""
	}
	method := string(cred.Method())
	if method == "" {
		method = string(cred.Type)
	}
	if cred.Source == "" {
		return method
	}
	return method + " · " + cred.Source
}

func dockerHint(err error) error {
	if strings.Contains(err.Error(), "cannot connect to Docker") {
		return fmt.Errorf("cannot reach Docker — is Docker Desktop or OrbStack running?")
	}
	return err
}

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
			name = world.AutoWorkspaceName(path, i)
		}
		result = append(result, world.Workspace{Name: name, Path: path, ReadOnly: readOnly})
	}
	return result, nil
}

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

func applyProjectDefaults(cmd *cobra.Command, requestedName, backendOverride string) (*projectWorld, error) {
	p, err := loadProject()
	if err != nil || p == nil {
		return nil, nil
	}
	pw, err := resolveProjectWorld(p, requestedName, backendOverride)
	if err != nil {
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
