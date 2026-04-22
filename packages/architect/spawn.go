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
	runtimes "spwn.sh/packages/runtimes"
	"spwn.sh/packages/dependency"


	"spwn.sh/packages/transpile"
	ib "spwn.sh/packages/compile"
	ibbase "spwn.sh/packages/compile/base"
	"spwn.sh/packages/container/backend"
	"spwn.sh/packages/dependency/resolver"
	"spwn.sh/packages/architect/internal/deploy"
	"spwn.sh/packages/world/labels"
	"spwn.sh/packages/world/models"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/activity"
	"spwn.sh/packages/auth"
)

// SpawnResult is returned by Spawn with the world and any non-fatal warnings.
type SpawnResult struct {
	World    *models.World
	Warnings []string
}

// SpawnOpts configures world creation.
type SpawnOpts struct {
	ConfigName    string
	Name          string // Optional user-facing display name.
	AgentName     string
	Workspaces    []models.Workspace
	Manifest      models.Manifest
	Image         string                     // Override base image (used for testing). Defaults to platform.WorldImage.
	OnProgress    func(event, detail string) // Optional callback at each milestone.
	LogWriter     io.Writer                  // Receives Docker build output. nil defaults to io.Discard.
	Agents        []AgentSpec                // Multi-agent list (alternative to single AgentName).
	IsArchitect   bool                       // When true, mounts Docker socket + SPWN_HOME for Architect mode.
	ForceRebuild  bool                       // When true, bypass the content-addressed image cache.
	// RuntimeName selects the runtime adapter that drives spawn-time
	// behavior (BuildCommand, credential sync, prelaunch shell) and
	// the transpile target. Short form: "claude-code", "codex".
	// Empty defaults to "claude-code" — the historical behavior and
	// the only runtime with a Renderer today.
	RuntimeName string
	// Knowledge is an absolute host path to bind into /world/knowledge/.
	// When empty, no bind mount is performed and the compile step is
	// told no knowledge base exists (so the agent's system prompt
	// never mentions /world/knowledge/). The CLI resolves any
	// project-relative path to absolute before calling Spawn.
	Knowledge string
}

// runtimeName returns opts.RuntimeName with the default-runtime
// fallback. Keeps callers and tests that don't populate the field
// working on the legacy default. Shares the same constant used by
// the per-world resolver (see runtime_route.go) so the "what does
// empty mean" question has exactly one answer in the package.
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

// firstWriteNotifier forwards writes to an inner writer and calls
// `once` exactly once on the first non-empty write. Used to
// lift "the build is actually running now" out of the Docker
// stream: the first byte arriving from `docker build` is the
// signal that the cache was missed and a real build started. No
// bytes ever arrive on a pure cache hit, so `once` never fires
// and the UI stays on "Resolving compile...".
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

// Spawn creates a new world.
func (a *Architect) Spawn(ctx context.Context, opts SpawnOpts) (*SpawnResult, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	var warnings []string

	// Lifecycle hooks (`hook:pre-spawn` and friends) were retired
	// alongside the old `/world/skills/` pipeline. Runtime hooks
	// (Claude Code / Codex PreToolUse + UserPromptSubmit + …) are
	// now declared in spwn/hooks.yaml and rendered into each agent's
	// `.claude/settings.json` / `.codex/hooks.json` by the transpile
	// layer — they run inside the container, not on the host.

	// Generate ID
	id := platform.GenerateWorldID(opts.ConfigName)

	// Resolve each workspace to absolute path and validate it exists.
	// Layout inside the container:
	//   - 0 workspaces (ephemeral): no mounts, container uses its image's /workspace dir.
	//   - 1+ workspaces: each mounted at /workspaces/<name>. The first is also
	//     mounted at /workspace for legacy tools that expect a single root.
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

	// Build mounts.
	binds := buildWorkspaceBinds(resolvedWorkspaces)

	// Architect mode: mount Docker socket + SPWN state directory
	if opts.IsArchitect {
		binds = append(binds, "/var/run/docker.sock:/var/run/docker.sock")
		binds = append(binds, platform.BaseDir()+":/home/spwn/.spwn")
	}

	// No /agents bind mount under the new architecture. Each
	// agent's home directory is copied INTO the container at
	// /agents/<name>/ by deploy.SyncIn() right after container
	// start. On graceful shutdown (Destroy), deploy.SyncOut()
	// copies the allowlisted memory directories back. Dotfiles,
	// npm cache, .claude/*, etc. stay inside the container and die
	// with it — the host project tree is never written to by a
	// container process.

	// Validate each named agent's mind directory and profile. We
	// still need to check that every agent's tree exists on the
	// host because we're about to copy it into the container.
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
		// Parse agent.yaml (optional). Used for future composition validation
		// against the world's available tools.
		if _, err := agent.LoadManifestPath(agent.AgentDir(name)); err != nil {
			return nil, fmt.Errorf("load agent manifest for %s: %w", name, err)
		}
	}

	// Resolve compile. SPWN_BASE_IMAGE and opts.Image both mean "use this
	// exact image, don't rebuild" - they're how tests inject a mock
	// runtime. Only when neither is set do we auto-build from the base
	// Dockerfile + tool catalog.
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

	// Registry + resolved dependency list are computed unconditionally.
	// Even when the image is prebuilt (tests injecting SPWN_BASE_IMAGE)
	// we still need the resolved tool list for tool-probe verification
	// and for rendering the Faculties block in every agent's CLAUDE.md.
	reg := resolver.NewRegistry()
	if err := dependency.RegisterBuiltins(reg); err != nil {
		return nil, fmt.Errorf("register tools: %w", err)
	}
	if err := runtimes.RegisterDefaults(reg); err != nil {
		return nil, fmt.Errorf("register runtimes: %w", err)
	}

	// Always include runtime essentials, then add user-specified tools
	// and dependencies on top. The registry deduplicates and resolves
	// dependencies; dependencies share the tool resolution pipeline.
	//
	// spwn:cli is deliberately excluded here - it installs the
	// spwn binary itself and is only meaningful inside the
	// architect container, not inside the workers' world container.
	//
	// The runtime tool is chosen from the declared backend:
	//   claude-code → spwn:claude-code (self-contained binary install)
	//   codex       → spwn:codex (npm install -g @openai/codex; pulls
	//                 spwn:node transitively)
	// Hardcoding spwn:claude-code here silently installed the wrong
	// runtime for codex agents, making their containers non-functional.
	runtimeTool := runtimeBackendTool(opts.runtimeName())
	required := []string{"spwn:unix", runtimeTool}
	toolList := append(required, opts.Manifest.Deps...)

	// Deduplicate
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

	// Hydrate local (tool:<name>) refs into synthetic tool.Tool
	// instances before resolving. Without this, a ref like
	// `tool:my-local-tool` would blow up reg.Resolve with "unknown tool".
	// Project root defaults to platform.ProjectRoot() — set by the CLI
	// PersistentPreRunE when a spwn.yaml is discovered.
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

	// Surface the resolved tool list so the CLI stepper can show
	// the user what's about to install before the (potentially
	// minutes-long) image build begins.
	resolvedNames := make([]string, 0, len(resolvedTools))
	for _, t := range resolvedTools {
		resolvedNames = append(resolvedNames, t.Name())
	}
	opts.progress("tools_resolved", strings.Join(resolvedNames, ", "))

	if !explicitImage {
		opts.progress("image_resolving", image)

		builder := ib.New(reg, a.backend)

		// Wrap the log writer so the first docker build line
		// flips the spinner label from "Resolving compile..." to
		// "Building image". Emits image_building exactly once,
		// the first time we see actual build output - which
		// means cache-hit spawns never raise the build label.
		// Simpler than pre-checking the cache in Go.
		buildLogWriter := &firstWriteNotifier{
			inner: opts.logWriter(),
			once: func() {
				opts.progress("image_building", image)
			},
		}

		buildResult, err := builder.Build(ctx, ib.BuildRequest{
			BaseDockerfile: ibbase.WorldDockerfile,
			Tools:          toolList,
			Tag:            image,
			ForceRebuild:   opts.ForceRebuild,
			SkipVerify:     true, // probeTools handles verification below
			LogWriter:      buildLogWriter,
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

	// Sync credentials to bind-mountable directory (live - containers see updates)
	if err := auth.SyncCredentials(); err != nil {
		warnings = append(warnings, fmt.Sprintf("credential sync: %v", err))
	}
	binds = append(binds, platform.CredentialsDir()+":/credentials:ro")

	// Determine credential source for progress reporting
	creds := auth.ResolveAll()
	credSource := "none"
	for _, cred := range creds {
		if cred.Type != auth.CredTypeNone {
			credSource = string(cred.Type)
			break
		}
	}
	opts.progress("credentials_resolved", credSource)

	// Non-credential env vars
	var env []string
	if opts.IsArchitect {
		env = append(env, "SPWN_ARCHITECT_MODE=1")
		env = append(env, "SPWN_HOME=/home/spwn/.spwn")
	}

	// Workspace discovery env vars
	if len(resolvedWorkspaces) > 0 {
		total := len(resolvedWorkspaces)
		pairs := make([]string, 0, total)
		for _, ws := range resolvedWorkspaces {
			pairs = append(pairs, fmt.Sprintf("%s:%s", ws.Name, workspaceContainerPath(ws.Name, total)))
		}
		env = append(env, "SPWN_WORKSPACES="+strings.Join(pairs, ","))
		env = append(env, "SPWN_WORKSPACE_DEFAULT="+workspaceContainerPath(resolvedWorkspaces[0].Name, total))
	}

	// Resolve the requested knowledge bind BEFORE constructing
	// worldRecord so the labels-as-truth store captures whether a
	// knowledge dir was actually mounted. The actual `binds` append
	// happens further below alongside the other world state mounts,
	// but the decision is made here so the worldRecord reflects
	// reality.
	knowledgeMounted := false
	if opts.Knowledge != "" {
		if info, err := os.Stat(opts.Knowledge); err == nil && info.IsDir() {
			knowledgeMounted = true
		} else {
			warnings = append(warnings, fmt.Sprintf("knowledge path %s not found; skipping bind", opts.Knowledge))
		}
	}

	// Build the World record up-front so we can imprint it onto the
	// container as labels at create time. The container becomes the
	// canonical store - see packages/world/internal/labels.
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

	// Per-world state directory on the host. Used as a stable target
	// for optional sub-path binds (`/world/knowledge`, and historically
	// `/world/shared` + `/world/skills`). We no longer bind-mount it at
	// `/world/` wholesale: that shadowed the image-baked `/world/skills/`
	// (CollectSkills bakes tool-shipped SKILL.md files there at build
	// time), which meant Claude Code's native skill discovery found an
	// empty directory and never surfaced any spwn-provided skill.
	//
	// Instead: leave `/world/` coming from the image (contains
	// `/world/skills/*` + anything a runtime's build step wrote) and
	// only bind the subpaths that need host-side persistence.
	worldStateDir := worldStateDirFor(id)
	if err := os.MkdirAll(worldStateDir, 0o755); err != nil {
		return nil, fmt.Errorf("create world-state dir %s: %w", worldStateDir, err)
	}

	// Bind the explicit knowledge path on top of /world/knowledge so
	// edits inside the container persist straight back into git. The
	// CLI resolves any project-relative path in spwn.yaml to an
	// absolute host path before calling Spawn. When the field is
	// empty (or stat failed earlier), there is no bind mount AND the
	// compile step below is told no knowledge base exists, so the
	// agent's system prompt never mentions /world/knowledge/.
	if knowledgeMounted {
		binds = append(binds, opts.Knowledge+":/world/knowledge")
	}

	// Per-agent per-world deployment dirs. Each agent gets a personal
	// inbox/outbox/notes scoped to this world id, all rooted in the
	// agent's persistent home so messages survive container destroy.
	// The rendered files (role.md, CLAUDE.md) come from the compile
	// Tree below; here we only create the empty inbox/outbox/notes
	// directories because they're runtime state, not generated
	// content.
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

	// Create container. CPU/memory limits intentionally omitted — the
	// Docker host defaults govern. Per-world hard limits may return as
	// a dedicated knob later but are not declared in spwn.yaml.
	containerCfg := backend.ContainerConfig{
		Image:       image,
		Name:        id,
		PidsLimit:   256,
		NetworkMode: "bridge",
		Binds:       binds,
		Env:         env,
		Labels:      labels.WorldLabels(worldRecord),
	}

	containerID, err := a.backend.Create(ctx, containerCfg)
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	if err := a.backend.Start(ctx, containerID); err != nil {
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("start container: %w", err)
	}
	opts.progress("container_created", id)

	// Copy every deployed agent's home directory from the host into
	// the container at /agents/<name>/. This is the replacement for
	// the former bind mount — it's a one-way snapshot, writes inside
	// the container never flow back to the host except on graceful
	// shutdown (see Destroy → syncAgentsOutOf).
	agentHomes := agentHomesForSpawn(opts)
	if err := deploy.SyncIn(ctx, a.backend, containerID, agentHomes); err != nil {
		a.backend.Stop(ctx, containerID)
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("sync agent homes into container: %w", err)
	}

	// Write the runtime provider's default config files directly
	// into the running container at each agent's HOME. These
	// pre-dismiss first-run UI — Claude Code's onboarding banner,
	// trust dialogs, dangerous-mode prompt — so `spwn agent <name>`
	// drops straight into a clean session. docker cp'd per file,
	// overwrites any placeholder that came in via the host copy.
	if len(agentHomes) > 0 {
		if err := writeRuntimeDefaultConfig(ctx, a.backend, containerID, opts.runtimeName(), agentHomes); err != nil {
			warnings = append(warnings, fmt.Sprintf("runtime default config: %v", err))
		}
	}

	// The chown used to sit here — but the transpile tree gets
	// docker-cp'd further down, which re-creates root-owned files
	// on top of our work. ChownAgentHomes now runs AFTER every cp
	// step (see the second call below) so spwn can actually write
	// to its own home (e.g. codex's PrelaunchShell appending the
	// trust table to $HOME/.codex/config.toml).

	// Probe tools by running each resolved tool's Verify() commands
	// inside the container. This is the canonical "is my image
	// actually healthy" check - the probe pulls its expectations
	// straight from the catalog, so the same install specs that
	// built the image decide what must be present at runtime.
	verifiedTools, err := a.probeTools(ctx, containerID, resolvedTools)
	if err != nil {
		a.backend.Stop(ctx, containerID)
		a.backend.Remove(ctx, containerID)
		return nil, err
	}
	opts.progress("tools_probed", fmt.Sprintf("%d verified", len(verifiedTools)))

	// Render every file this world needs through the compiler. The
	// claude-code Runtime produces agents/<name>/CLAUDE.md (fully
	// self-contained system prompt — world context inlined) and
	// agents/<name>/worlds/<id>/role.md per deployment. Nothing
	// lands under world/ any more: physics, faculties, and roster
	// are inlined into each CLAUDE.md. MaterialiseTree is still
	// prefix-aware in case a future runtime emits world/* files;
	// today every entry flows via docker-cp into the agent home.
	compileInput := transpile.Input{
		Deps:                  opts.Manifest.Deps,
		VerifiedTools:         verifiedTools,
		WorldID:               id,
		Agents:                rosterCompileAgents(rosterAgents),
		WorldKnowledgeMounted: knowledgeMounted,
		Skills:                collectRuntimeSkills(platform.ProjectRoot(), resolvedTools),
		Hooks:                 loadRuntimeHooks(platform.ProjectRoot()),
	}
	tree, err := transpile.Compile(opts.runtimeName(), compileInput)
	if err != nil {
		a.backend.Stop(ctx, containerID)
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("compile world: %w", err)
	}
	if err := deploy.MaterialiseTree(ctx, a.backend, containerID, tree, worldStateDir); err != nil {
		a.backend.Stop(ctx, containerID)
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("materialise world tree: %w", err)
	}
	opts.progress("world_state_written", "per-agent CLAUDE.md + role.md")

	// Chown the whole agent home tree AFTER every docker-cp step so
	// every file the agent might need to touch at runtime (auth
	// writes, codex config appends, journal scribbles) is spwn-owned
	// and writable. Tar extraction from docker cp always lands files
	// as root:root regardless of source; this pass repairs that.
	if len(agentHomes) > 0 {
		if err := deploy.ChownAgentHomes(ctx, a.backend, containerID, agentHomes); err != nil {
			a.backend.Stop(ctx, containerID)
			a.backend.Remove(ctx, containerID)
			return nil, fmt.Errorf("chown agent homes: %w", err)
		}
	}

	// Finalize the world record. The labels we already wrote to the
	// container are the canonical store - this struct is just what we
	// hand back to the caller. ContainerID and Status come from the
	// runtime side, not from labels. Future state.List() calls will
	// reconstruct identical Worlds straight from container labels.
	u := worldRecord
	u.ContainerID = containerID
	u.Status = models.StatusIdle

	// Emit activity events
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

