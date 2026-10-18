package architect

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/dependency"
	runtimes "spwn.sh/packages/runtimes"

	"spwn.sh/packages/activity"
	"spwn.sh/packages/architect/internal/deploy"
	"spwn.sh/packages/authentication"
	authgh "spwn.sh/packages/authentication/gh"
	ib "spwn.sh/packages/compile"
	ibbase "spwn.sh/packages/compile/baseimages"
	"spwn.sh/packages/container/backend"
	"spwn.sh/packages/dependency/resolver"
	"spwn.sh/packages/gate"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/transpile"
	"spwn.sh/packages/world/labels"
	"spwn.sh/packages/world/models"
)

func buildPolicyMap(in map[string]models.DepPolicy) map[string]ib.ToolPolicy {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]ib.ToolPolicy, len(in))
	for k, v := range in {
		out[k] = ib.ToolPolicy{Allow: v.Allow, Deny: v.Deny}
	}
	return out
}

// SpawnResult is returned by Spawn with the world and any non-fatal warnings.
type SpawnResult struct {
	World    *models.World
	Warnings []string
}

// SpawnOpts configures world creation.
type SpawnOpts struct {
	ConfigName   string
	Name         string
	AgentName    string
	Workspaces   []models.Workspace
	Manifest     models.Manifest
	Image        string
	OnProgress   func(event, detail string)
	LogWriter    io.Writer
	Agents       []AgentSpec
	IsArchitect  bool
	ForceRebuild bool
	// RuntimeName selects the runtime adapter that drives spawn-time
	// behavior (BuildCommand, credential sync, prelaunch shell) and
	// the transpile target. Short form: "claude-code", "codex".
	// Empty defaults to "claude-code", the historical behavior and
	// the only runtime with a Renderer today.
	RuntimeName string
	// Knowledge is an absolute host path to bind into /world/knowledge/.
	// When empty, no bind mount is performed and the compile step is
	// told no knowledge base exists (so the agent's system prompt
	// never mentions /world/knowledge/). The CLI resolves any
	// project-relative path to absolute before calling Spawn.
	Knowledge string
}

func (opts *SpawnOpts) runtimeName() string {
	if opts.RuntimeName != "" {
		return opts.RuntimeName
	}
	return defaultRuntimeName
}

// Validate returns a non-nil error when SpawnOpts is missing
// required fields or has an internally inconsistent combination.
// Called at the top of Spawn before any side-effectful work.
//
// Agent-less spawns are legitimate: a world can be created first
// and the agent attached later via SpawnAgent / SpawnAgentDetached.
// The e2e suite exercises this path via the NoAgent() builder.
func (opts *SpawnOpts) Validate() error {
	if opts.ConfigName == "" {
		return fmt.Errorf("SpawnOpts.ConfigName is required")
	}
	return nil
}

func (opts *SpawnOpts) progress(event, detail string) {
	if opts.OnProgress != nil {
		opts.OnProgress(event, detail)
	}
}

func (opts *SpawnOpts) logWriter() io.Writer {
	if opts.LogWriter != nil {
		return opts.LogWriter
	}
	return io.Discard
}

type firstWriteNotifier struct {
	inner io.Writer
	once  func()
	fired bool
}

func (w *firstWriteNotifier) Write(p []byte) (int, error) {
	if !w.fired && len(p) > 0 {
		w.fired = true
		if w.once != nil {
			w.once()
		}
	}
	return w.inner.Write(p)
}

func (a *Architect) rollback(ctx context.Context, containerID string) {
	_ = a.backend.Stop(ctx, containerID)
	_ = a.backend.Remove(ctx, containerID)
}

// Spawn creates a new world.
func (a *Architect) Spawn(ctx context.Context, opts SpawnOpts) (*SpawnResult, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	var warnings []string

	id := platform.GenerateWorldID(opts.ConfigName)

	resolvedWorkspaces := make([]models.Workspace, 0, len(opts.Workspaces))
	seenNames := map[string]bool{}
	for i, ws := range opts.Workspaces {
		abs, absErr := filepath.Abs(ws.Path)
		if absErr != nil {
			return nil, fmt.Errorf("resolve workspace %q: %w", ws.Path, absErr)
		}
		if _, statErr := os.Stat(abs); statErr != nil {
			return nil, fmt.Errorf("workspace %s not found.\nCheck the path exists and is accessible", abs)
		}
		name := strings.TrimSpace(ws.Name)
		if name == "" {
			name = fmt.Sprintf("w%d", i)
		}
		if seenNames[name] {
			return nil, fmt.Errorf("duplicate workspace name %q", name)
		}
		seenNames[name] = true
		resolvedWorkspaces = append(resolvedWorkspaces, models.Workspace{Name: name, Path: abs, ReadOnly: ws.ReadOnly})
	}

	binds := buildWorkspaceBinds(resolvedWorkspaces)
	var groupAdd []string

	if opts.IsArchitect {
		binds = append(binds, "/var/run/docker.sock:/var/run/docker.sock")
		binds = append(binds, platform.BaseDir()+":/home/spwn/.spwn")
	}

	agentNamesToValidate := []string{}
	if len(opts.Agents) > 0 {
		for _, spec := range opts.Agents {
			agentNamesToValidate = append(agentNamesToValidate, spec.Name)
		}
	} else if opts.AgentName != "" {
		agentNamesToValidate = append(agentNamesToValidate, opts.AgentName)
	}
	for _, name := range agentNamesToValidate {
		if err := agent.ValidateMind(name); err != nil {
			return nil, err
		}
		opts.progress("mind_validated", name)
		if _, err := agent.LoadManifestPath(agent.AgentDir(name)); err != nil {
			return nil, fmt.Errorf("load agent manifest for %s: %w", name, err)
		}
	}

	image := platform.WorldImage
	explicitImage := false
	if envImage := os.Getenv("SPWN_BASE_IMAGE"); envImage != "" {
		image = envImage
		explicitImage = true
	}
	if opts.Image != "" {
		image = opts.Image
		explicitImage = true
	}

	reg := resolver.NewRegistry()
	if err := dependency.RegisterBuiltins(reg); err != nil {
		return nil, fmt.Errorf("register tools: %w", err)
	}
	if err := runtimes.RegisterDefaults(reg); err != nil {
		return nil, fmt.Errorf("register runtimes: %w", err)
	}

	runtimeTool := runtimeBackendTool(opts.runtimeName())
	required := []string{"spwn:unix", runtimeTool}
	toolList := append(required, opts.Manifest.Deps...)

	{
		seen := make(map[string]bool)
		deduped := make([]string, 0, len(toolList))
		for _, t := range toolList {
			if !seen[t] {
				seen[t] = true
				deduped = append(deduped, t)
			}
		}
		toolList = deduped
	}

	if projectRoot := platform.ProjectRoot(); projectRoot != "" {
		hydrated, hErr := dependency.HydrateLocals(reg, projectRoot, toolList)
		if hErr != nil {
			return nil, fmt.Errorf("load local tools: %w", hErr)
		}
		toolList = hydrated
	}

	resolvedTools, resolveErr := reg.Resolve(toolList)
	if resolveErr != nil {
		return nil, fmt.Errorf("resolve tools: %w", resolveErr)
	}

	resolvedNames := make([]string, 0, len(resolvedTools))
	for _, t := range resolvedTools {
		resolvedNames = append(resolvedNames, t.Name())
	}
	opts.progress("tools_resolved", strings.Join(resolvedNames, ", "))

	releaseImage := func() {}
	defer func() { releaseImage() }()
	if !explicitImage {
		release, lockErr := lockWorldImage(ctx)
		if lockErr != nil {
			return nil, fmt.Errorf("wait for the world image: %w", lockErr)
		}
		releaseImage = release
		opts.progress("image_resolving", image)

		builder := ib.New(reg, a.backend)

		buildLogWriter := &firstWriteNotifier{
			inner: opts.logWriter(),
			once: func() {
				opts.progress("image_building", image)
			},
		}

		buildResult, err := builder.Build(ctx, ib.BuildRequest{
			BaseDockerfile: ibbase.WorldDockerfile,
			Tools:          toolList,
			Policies:       buildPolicyMap(opts.Manifest.DepPolicies),

			Tag:          image,
			ForceRebuild: opts.ForceRebuild,
			SkipVerify:   true,
			LogWriter:    buildLogWriter,
		})
		if err != nil {
			return nil, fmt.Errorf("build world image: %w", err)
		}
		if buildResult.Cached {
			opts.progress("image_cached", image)
		} else {
			opts.progress("image_built", image)
		}
	} else {
		exists, err := a.backend.ImageExists(ctx, image)
		if err != nil {
			return nil, fmt.Errorf("check image: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("image %s not found.\nBuild it first or use the default base image", image)
		}
		opts.progress("image_ready", image)
	}

	if err := authentication.SyncCredentials(); err != nil {
		warnings = append(warnings, fmt.Sprintf("credential sync: %v", err))
	}
	binds = append(binds, platform.CredentialsDir()+":/credentials:ro")

	if err := gate.EnsureRunning(ctx, opts.logWriter()); err != nil {
		warnings = append(warnings, fmt.Sprintf("gate ensure-running: %v", err))
	}

	if ghCache := authgh.CacheDir(); ghCache != "" {
		if err := os.MkdirAll(ghCache, 0o700); err == nil {
			binds = append(binds, ghCache+":/credentials/gh")
		}
	}

	creds := authentication.ResolveAll()
	credSource := "none"
	for _, cred := range creds {
		if cred.Type != authentication.CredTypeNone {
			credSource = string(cred.Type)
			break
		}
	}
	if credSource == "none" {
		if os.Getenv("SPWN_SKIP_AUTH_VALIDATION") != "" {
			credSource = "skipped"
		} else if providerName := runtimeProvider(opts.runtimeName()); providerName != "" {
			p := authentication.Provider(providerName)
			return nil, fmt.Errorf("no credentials configured for %s.\n%s", p, authentication.NotConfiguredHint(p))
		}
	}
	opts.progress("credentials_resolved", credSource)

	var env []string
	if opts.IsArchitect {
		env = append(env, "SPWN_ARCHITECT_MODE=1")
		env = append(env, "SPWN_HOME=/home/spwn/.spwn")
	}

	env = append(env, "GH_CONFIG_DIR=/credentials/gh")

	if len(resolvedWorkspaces) > 0 {
		total := len(resolvedWorkspaces)
		pairs := make([]string, 0, total)
		for _, ws := range resolvedWorkspaces {
			pairs = append(pairs, fmt.Sprintf("%s:%s", ws.Name, workspaceContainerPath(ws.Name, total)))
		}
		env = append(env, "SPWN_WORKSPACES="+strings.Join(pairs, ","))
		env = append(env, "SPWN_WORKSPACE_DEFAULT="+workspaceContainerPath(resolvedWorkspaces[0].Name, total))
	}

	knowledgeMounted := false
	if opts.Knowledge != "" {
		if info, err := os.Stat(opts.Knowledge); err == nil && info.IsDir() {
			knowledgeMounted = true
		} else {
			warnings = append(warnings, fmt.Sprintf("knowledge path %s not found; skipping bind", opts.Knowledge))
		}
	}

	worldRecord := models.World{
		ID:               id,
		Name:             opts.Name,
		Config:           opts.ConfigName,
		Agent:            opts.AgentName,
		Backend:          platform.DefaultBackend,
		Runtime:          opts.runtimeName(),
		Workspaces:       resolvedWorkspaces,
		CreatedAt:        time.Now(),
		Manifest:         opts.Manifest,
		KnowledgeMounted: knowledgeMounted,
	}
	if len(opts.Agents) > 0 {
		worldRecord.Agent = opts.Agents[0].Name
		worldRecord.AgentID = platform.GenerateAgentID(opts.Agents[0].Name)
		for _, spec := range opts.Agents {
			role := agent.DefaultRole(spec.Role)
			worldRecord.Agents = append(worldRecord.Agents, models.AgentRecord{
				Name:    spec.Name,
				AgentID: platform.GenerateAgentID(spec.Name),
				Role:    role,
				Status:  models.StatusIdle,
			})
		}
	} else if opts.AgentName != "" {
		worldRecord.AgentID = platform.GenerateAgentID(opts.AgentName)
		worldRecord.Agents = []models.AgentRecord{{
			Name:    opts.AgentName,
			AgentID: worldRecord.AgentID,
			Role:    "worker",
			Status:  models.StatusIdle,
		}}
	}

	worldStateDir := worldStateDirFor(id)
	if err := os.MkdirAll(worldStateDir, 0o755); err != nil {
		return nil, fmt.Errorf("create world-state dir %s: %w", worldStateDir, err)
	}

	if knowledgeMounted {
		binds = append(binds, opts.Knowledge+":/world/knowledge")
	}

	rosterAgents := worldRecord.Agents
	if len(rosterAgents) == 0 && worldRecord.Agent != "" {
		rosterAgents = []models.AgentRecord{{
			Name:    worldRecord.Agent,
			AgentID: worldRecord.AgentID,
			Role:    "worker",
			Status:  models.StatusIdle,
		}}
	}
	for _, rec := range rosterAgents {
		if err := initAgentDeploymentDirs(rec, id); err != nil {
			warnings = append(warnings, fmt.Sprintf("init deployment for %s: %v", rec.Name, err))
		}
	}

	pidsLimit := int64(256)
	containerCfg := backend.ContainerConfig{
		Image:       image,
		Name:        id,
		PidsLimit:   pidsLimit,
		NetworkMode: "bridge",
		Binds:       binds,
		Env:         env,
		Labels:      labels.WorldLabels(worldRecord),
		GroupAdd:    groupAdd,
	}

	containerID, err := a.backend.Create(ctx, containerCfg)
	releaseImage()
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	if err := a.backend.Start(ctx, containerID); err != nil {
		a.rollback(ctx, containerID)
		return nil, fmt.Errorf("start container: %w", err)
	}
	opts.progress("container_created", id)

	agentHomes := agentHomesForSpawn(opts)
	if err := deploy.SyncIn(ctx, a.backend, containerID, agentHomes); err != nil {
		a.rollback(ctx, containerID)
		return nil, fmt.Errorf("sync agent homes into container: %w", err)
	}

	if len(agentHomes) > 0 {
		if err := writeRuntimeDefaultConfig(ctx, a.backend, containerID, opts.runtimeName(), agentHomes); err != nil {
			warnings = append(warnings, fmt.Sprintf("runtime default config: %v", err))
		}
	}

	verifiedTools, err := a.probeTools(ctx, containerID, resolvedTools)
	if err != nil {
		a.rollback(ctx, containerID)
		return nil, err
	}
	opts.progress("tools_probed", fmt.Sprintf("%d verified", len(verifiedTools)))

	runtimeSkills := collectRuntimeSkills(platform.ProjectRoot(), resolvedTools)
	hookPool := loadRuntimeHooks(platform.ProjectRoot())
	commandPool := loadRuntimeCommands(platform.ProjectRoot())
	compileInput := transpile.Input{
		Deps:                  opts.Manifest.Deps,
		VerifiedTools:         facultiesForRuntime(verifiedTools, opts.Manifest.Deps),
		WorldID:               id,
		Agents:                rosterCompileAgents(rosterAgents, hookPool, commandPool),
		WorldKnowledgeMounted: knowledgeMounted,
		Skills:                runtimeSkills,
		Hooks:                 hookPool,
		Commands:              commandPool,
	}
	tree, err := transpile.Compile(opts.runtimeName(), compileInput)
	if err != nil {
		a.rollback(ctx, containerID)
		return nil, fmt.Errorf("compile world: %w", err)
	}
	if err := deploy.MaterialiseTree(ctx, a.backend, containerID, tree, worldStateDir); err != nil {
		a.rollback(ctx, containerID)
		return nil, fmt.Errorf("materialise world tree: %w", err)
	}
	opts.progress("world_state_written", "per-agent "+runtimePromptFile(opts.runtimeName()))

	if len(agentHomes) > 0 {
		if err := deploy.ChownAgentHomes(ctx, a.backend, containerID, agentHomes); err != nil {
			a.rollback(ctx, containerID)
			return nil, fmt.Errorf("chown agent homes: %w", err)
		}
	}

	u := worldRecord
	u.ContainerID = containerID
	u.Status = models.StatusIdle

	agentNames := []string{}
	for _, ag := range u.Agents {
		agentNames = append(agentNames, ag.Name)
	}
	if len(agentNames) == 0 && u.Agent != "" {
		agentNames = append(agentNames, u.Agent)
	}
	activity.Log(activity.Event{
		Type:    activity.TypeWorldSpawned,
		Actor:   "architect",
		Verb:    "spawned",
		Target:  u.ID,
		Phrase:  activity.PhraseWorldSpawned(u.ID, agentNames),
		WorldID: u.ID,
	})
	for _, name := range agentNames {
		activity.Log(activity.Event{
			Type:    activity.TypeAgentJoined,
			Actor:   "architect",
			Verb:    "joined",
			Target:  u.ID,
			Phrase:  activity.PhraseAgentJoined(name, u.ID),
			WorldID: u.ID,
			AgentID: name,
		})
	}

	return &SpawnResult{World: &u, Warnings: warnings}, nil
}
