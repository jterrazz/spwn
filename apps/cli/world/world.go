package world

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/cliproject"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/architect"
	"spwn.sh/packages/authentication"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/project"
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
	spawnYes          bool
)

func init() {
	Cmd.SetHelpFunc(worldHelp)
}

// SetSpawnBackend overrides the runtime backend used by the next
// Spawn. Intended for callers in sibling packages (apps/cli/agent)
// That delegate to UpCmd.RunE without going through cobra flag parsing
// On the world command itself (see agent.runInteractiveSession where
// `spwn agent neo --backend codex` routes the backend through here
// before handing off to UpCmd).
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
	c.Flags().BoolVar(&spawnYes, "yes", false, "Allow, for this run only, the host paths outside the project that spwn.yaml mounts, printing each; never kept")
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
				{Name: "list", Desc: "List active worlds"},
				{Name: "rename <id> <name>", Desc: "Rename " + ui.Faint("(empty name clears)")},
			}},
			{Title: "Observe", Commands: []ui.HelpEntry{
				{Name: "inspect <id>", Desc: "Inspect a running world's composition and state"},
				{Name: "logs <id>", Desc: "Show event log for a world"},
				{Name: "enter <id>", Desc: "Open an interactive shell inside a world"},
			}},
			{Title: "Examples", Commands: []ui.HelpEntry{
				{Name: "spwn up --agent neo -w .", Desc: "Spawn neo in current dir"},
				{Name: "spwn up --agent morpheus --agent neo -w .", Desc: ""},
				{Name: "spwn world list", Desc: "See what's running"},
			}},
		},
		"spwn world [command]",
		"",
	)
}

var Cmd = &cobra.Command{
	Use:   "world",
	Short: "Manage worlds - ephemeral runtime instances for agents",
}

// UpCmd is `spwn up`, the one spelling that brings worlds up.
var UpCmd = &cobra.Command{
	Use:   "up [name]",
	Short: "Spawn a world - an isolated container for agents",
	Long: `Spawn a world: an isolated container and the agents declared in it.

Inside a Spwn project:
  spwn up             brings up every world declared in spwn.yaml
  spwn up <name>      brings up a specific world from spwn.yaml

Outside a project, the legacy global-mode flags still work and spawn
a one-off world from ~/.spwn/worlds/<config>.yaml.`,
	Args: cobra.MaximumNArgs(1),
	Example: `  spwn up                                          Bring up every world in spwn.yaml
  spwn up neo                                      Start the "neo" world
  spwn up --agent neo -w .                         Single agent in current dir
  spwn up --agent morpheus --agent neo -w .        Multi-agent (morpheus is chief)`,
	RunE: composeUpRunE,
}

type plannedWorld struct {
	name    string
	pw      *projectWorld
	running *world.World
	drifted *world.World
	reaches []cliproject.Reach
}

var (
	startWorld   = spawnRunE
	askTerminal  = cliproject.AskTerminal
	runningWorld = findRunningWorldByConfig
)

func composeUpRunE(cmd *cobra.Command, args []string) error {
	p, err := loadProject()
	if err != nil || p == nil || len(p.Manifest.Worlds) == 0 {
		return startWorld(cmd, args, nil)
	}
	names := args
	if len(names) == 0 {
		names = sortedWorldNames(p)
	}
	many := len(names) > 1
	plans, err := planWorlds(cmd, p, names, many)
	if err != nil {
		return err
	}
	var reaches []cliproject.Reach
	for _, pl := range plans {
		reaches = append(reaches, pl.reaches...)
	}
	allowed, err := cliproject.Consent(p.Root, reaches, spawnYes, askTerminal(), cmd.InOrStdin(), cmd.ErrOrStderr())
	if err != nil {
		return ui.New().FailHint("Outside the project", err,
			"Run `spwn up` in a terminal to be asked once (a yes is kept), or `spwn up --yes` to allow these paths for this run only")
	}
	for _, pl := range plans {
		if many {
			resetSpawnFlags(cmd)
		}
		if err := startWorld(cmd, []string{pl.name}, &pl); err != nil {
			return err
		}
		if err := allowed.Keep(pl.reaches); err != nil {
			ui.New().Warn("Grants", fmt.Sprintf("the yes was not kept in %s: %v", cliproject.GrantsPath(), err))
		}
	}
	return nil
}

// invariant: planWorlds prints nothing unless it refuses and reads nothing outside the project, so the run's one ask, after it, precedes every byte a world prints.
func planWorlds(cmd *cobra.Command, p *project.Project, names []string, many bool) ([]plannedWorld, error) {
	ctx := context.Background()
	backend := spawnBackend
	if many {
		backend = ""
	}
	plans := make([]plannedWorld, 0, len(names))
	for _, name := range names {
		if err := platform.CheckName("world", name); err != nil {
			return nil, ui.New().FailHint("Project", err, "Name a world by one name of spwn.yaml's worlds:")
		}
		pw, err := resolveProjectWorld(p, name, backend)
		if err != nil {
			return nil, ui.New().FailHint("Project", err, "Check spwn.yaml or pick an existing world name")
		}
		pl := plannedWorld{name: name, pw: pw}
		if existing := runningWorld(ctx, name); existing != nil {
			if pw.Runtime != "" && existing.Runtime != "" && existing.Runtime != pw.Runtime {
				pl.drifted = existing
			} else {
				pl.running = existing
			}
		}
		if pl.running == nil {
			for _, r := range pw.Outside {
				if strings.Contains(r.Grants, " mount at /workspaces/") && cmd.Flags().Changed("workspace") {
					continue
				}
				pl.reaches = append(pl.reaches, r)
			}
		}
		plans = append(plans, pl)
	}
	return plans, nil
}

func (pw *projectWorld) recheck(withWorkspaces bool) func() error {
	declared := slices.Clone(pw.DeclaredKnowledge)
	if withWorkspaces {
		declared = append(declared, pw.DeclaredWorkspaces...)
	}
	root := pw.Project.Root
	return func() error { return cliproject.Recheck(root, declared) }
}

func resetSpawnFlags(cmd *cobra.Command) {
	spawnConfig = ""
	spawnName = ""
	spawnAgents = nil
	spawnWorkspaces = nil
	spawnBackend = ""
}

func spawnRunE(cmd *cobra.Command, args []string, pl *plannedWorld) error {
	ctx := context.Background()
	s := ui.New()

	positionalName := ""
	if len(args) > 0 {
		positionalName = args[0]
	}

	var running, drifted *world.World
	switch {
	case pl != nil:
		running, drifted = pl.running, pl.drifted
	case positionalName != "":
		if err := platform.CheckName("world", positionalName); err != nil {
			return s.FailHint("Project", err, "Name a world by one name of spwn.yaml's worlds:")
		}
		running = runningWorld(ctx, positionalName)
	}
	if running != nil {
		s.Blank()
		s.Success(fmt.Sprintf("world %q is already running (%s)", positionalName, running.ID))
		s.Blank()
		s.Info("Enter:", fmt.Sprintf("spwn world enter %s", running.ID))
		s.Blank()
		return nil
	}
	if drifted != nil {
		s.Blank()
		s.Warn("Runtime drift",
			fmt.Sprintf("world %q was spawned with %q and now wants %q, rebuilding",
				positionalName, drifted.Runtime, pl.pw.Runtime))
		s.Blank()
		if arc, derr := architect.NewFromEnv(); derr == nil {
			_, _ = arc.Destroy(ctx, drifted.ID)
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

	var pw *projectWorld
	if pl != nil {
		pw = pl.pw
		applyWorldDefaults(cmd, pw)
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
			"Run \"station project create <name>\" to make a project")
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
	var recheck func() error
	if pw != nil {
		ownWorkspaces := !cmd.Flags().Changed("workspace") && len(spawnWorkspaces) == 0
		if ownWorkspaces {
			workspaces = pw.Workspaces
		}
		recheck = pw.recheck(ownWorkspaces)
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
	s.SubStart("Connecting to Docker...")
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
		s.SubStart("Checking credentials...")
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
		Recheck:      recheck,
		LogWriter:    buildProgress,
		OnProgress: func(event, detail string) {
			switch event {
			case "mind_validated":
				s.SubDone("Agent", detail)
			case "tools_resolved":
				enterImage()
				if detail != "" {
					s.TreeLine("tools:", detail)
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
		s.SubStart("Spawning colony...")
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
		s.SubStart("Spawning agent...")
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
		return fmt.Errorf("cannot reach Docker: is Docker Desktop or OrbStack running?")
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
		return fmt.Sprintf("Run \"station agents create %s\" to create the agent first", name)
	case strings.Contains(msg, "missing the core"):
		name := agentName
		if len(agents) > 0 {
			name = agents[0].Name
		}
		return fmt.Sprintf("Run \"station agents create %s\" to set up the agent layers", name)
	case strings.Contains(msg, "passwordless sudo"):
		return "Give the image a spwn user and passwordless sudo, or derive it from a spwn/world image"
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

func applyWorldDefaults(cmd *cobra.Command, pw *projectWorld) {
	if !cmd.Flags().Changed("config") && !cmd.Flags().Changed("world") && spawnConfig == "" {
		spawnConfig = pw.Name
	}
	if !cmd.Flags().Changed("agent") && len(spawnAgents) == 0 {
		spawnAgents = append(spawnAgents, pw.Agents...)
	}
}
