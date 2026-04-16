package architect

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"spwn.sh/packages/agent"
	runtimes "spwn.sh/catalog/runtimes"
	packs "spwn.sh/catalog/packs"
	"spwn.sh/packages/compile"
	"spwn.sh/packages/compile/runtimes/claudecode"
	ib "spwn.sh/packages/image"
	ibbase "spwn.sh/packages/image/base"
	"spwn.sh/packages/world/internal/backend"
	"spwn.sh/packages/world/internal/labels"
	"spwn.sh/packages/world/internal/runtime"
	"spwn.sh/packages/world/manifest"
	"spwn.sh/packages/world/models"
	"spwn.sh/packages/base"
	"spwn.sh/packages/activity"
	"spwn.sh/packages/auth"
	"spwn.sh/packages/paths"
	"spwn.sh/packages/ids"
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
	Image         string                     // Override base image (used for testing). Defaults to base.WorldImage.
	OnProgress    func(event, detail string) // Optional callback at each milestone.
	LogWriter     io.Writer                  // Receives Docker build output. nil defaults to io.Discard.
	Agents        []AgentSpec                // Multi-agent list (alternative to single AgentName).
	IsArchitect   bool                       // When true, mounts Docker socket + SPWN_HOME for Architect mode.
	ForceRebuild  bool                       // When true, bypass the content-addressed image cache.
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
// and the UI stays on "Resolving image...".
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
	var warnings []string

	// Generate ID
	id := ids.GenerateWorldID(opts.ConfigName)

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
		binds = append(binds, paths.BaseDir()+":/home/spwn/.spwn")
	}

	// No /agents bind mount under the new architecture. Each
	// agent's home directory is copied INTO the container at
	// /agents/<name>/ by syncAgentsInto() right after container
	// start. On graceful shutdown (Destroy), syncAgentsOutOf()
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
		if _, err := manifest.LoadAgent(agent.AgentDir(name)); err != nil {
			return nil, fmt.Errorf("load agent manifest for %s: %w", name, err)
		}
	}

	// Resolve image. SPWN_BASE_IMAGE and opts.Image both mean "use this
	// exact image, don't rebuild" - they're how tests inject a mock
	// runtime. Only when neither is set do we auto-build from the base
	// Dockerfile + tool catalog.
	image := base.WorldImage
	explicitImage := false
	if envImage := os.Getenv("SPWN_BASE_IMAGE"); envImage != "" {
		image = envImage
		explicitImage = true
	}
	if opts.Image != "" {
		image = opts.Image
		explicitImage = true
	}

	// Registry + resolved plugin list are computed unconditionally.
	// Even when the image is prebuilt (tests injecting SPWN_BASE_IMAGE),
	// plugin config still needs to be merged into the container's
	// runtime settings file after the container boots.
	reg := ib.NewRegistry()
	if err := packs.RegisterDefaults(reg); err != nil {
		return nil, fmt.Errorf("register tools: %w", err)
	}
	if err := runtimes.RegisterDefaults(reg); err != nil {
		return nil, fmt.Errorf("register runtimes: %w", err)
	}

	// Always include runtime essentials, then add user-specified tools
	// and plugins on top. The registry deduplicates and resolves
	// dependencies; plugins share the tool resolution pipeline.
	//
	// @spwn/cli is deliberately excluded here - it installs the
	// spwn binary itself and is only meaningful inside the
	// architect container, not inside the workers' world container.
	//
	// @spwn/node used to live here because the claude-code runtime
	// was installed via `npm install -g @anthropic-ai/claude-code`.
	// The native binary installer removed that dependency, so node
	// is no longer part of the baseline footprint. Users who want
	// node for their own tools still add @spwn/node to agent.yaml.
	required := []string{"@spwn/unix", "@spwn/claude-code"}
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

	// Hydrate local (bare-name) refs into synthetic image.Tool
	// instances before resolving. Without this, a ref like
	// `my-local-tool` would blow up reg.Resolve with "unknown tool".
	// Project root defaults to paths.ProjectRoot() — set by the CLI
	// PersistentPreRunE when a spwn.yaml is discovered.
	if projectRoot := paths.ProjectRoot(); projectRoot != "" {
		hydrated, hErr := hydrateLocalPacks(reg, projectRoot, toolList)
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
		// flips the spinner label from "Resolving image..." to
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
	binds = append(binds, paths.CredentialsDir()+":/credentials:ro")

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

	// Build the World record up-front so we can imprint it onto the
	// container as labels at create time. The container becomes the
	// canonical store - see packages/world/internal/labels.
	worldRecord := models.World{
		ID:         id,
		Name:       opts.Name,
		Config:     opts.ConfigName,
		Agent:      opts.AgentName,
		Backend:    base.DefaultBackend,
		Workspaces: resolvedWorkspaces,
		CreatedAt:  time.Now(),
		Manifest:   opts.Manifest,
	}
	if len(opts.Agents) > 0 {
		worldRecord.Agent = opts.Agents[0].Name
		worldRecord.AgentID = ids.GenerateAgentID(opts.Agents[0].Name)
		for _, spec := range opts.Agents {
			role := manifest.DefaultRole(spec.Role)
			worldRecord.Agents = append(worldRecord.Agents, models.AgentRecord{
				Name:    spec.Name,
				AgentID: ids.GenerateAgentID(spec.Name),
				Role:    role,
				Status:  models.StatusIdle,
			})
		}
	} else if opts.AgentName != "" {
		worldRecord.AgentID = ids.GenerateAgentID(opts.AgentName)
		worldRecord.Agents = []models.AgentRecord{{
			Name:    opts.AgentName,
			AgentID: worldRecord.AgentID,
			Role:    "worker",
			Status:  models.StatusIdle,
		}}
	}

	// Per-world state directory on the host. This is the canonical
	// /world/ inside the container - physics, faculties, roster,
	// AGENTS.md, system skills, and the shared whiteboard all live
	// here. Surviving container destroy is a deliberate choice: the
	// user's notes belong to the world, not the runtime.
	worldStateDir := worldStateDirFor(id)
	for _, sub := range []string{
		filepath.Join("shared", "notes"),
		filepath.Join("shared", "outputs"),
		"skills",
	} {
		if err := os.MkdirAll(filepath.Join(worldStateDir, sub), 0o755); err != nil {
			return nil, fmt.Errorf("create world-state dir %s: %w", sub, err)
		}
	}
	binds = append(binds, worldStateDir+":/world")

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
	if err := syncAgentsInto(ctx, a.backend, containerID, agentHomes); err != nil {
		a.backend.Stop(ctx, containerID)
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("sync agent homes into container: %w", err)
	}

	// Merge plugin runtime-config into the container's runtime settings
	// file. Currently only the claude-code backend has a known target
	// path; additional runtimes can grow their own branch as needed.
	if err := injectPluginRuntimeConfig(ctx, a.backend, containerID, resolvedTools); err != nil {
		a.backend.Stop(ctx, containerID)
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("inject plugin runtime config: %w", err)
	}

	// Write the runtime provider's default config files directly
	// into the running container at each agent's HOME. These
	// pre-dismiss first-run UI — Claude Code's onboarding banner,
	// trust dialogs, dangerous-mode prompt — so `spwn agent <name>`
	// drops straight into a clean session. docker cp'd per file,
	// overwrites any placeholder that came in via the host copy.
	if len(agentHomes) > 0 {
		if err := writeRuntimeDefaultConfig(ctx, a.backend, containerID, agentHomes); err != nil {
			warnings = append(warnings, fmt.Sprintf("runtime default config: %v", err))
		}
	}

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
	// claude-code Runtime produces a Tree with two kinds of entries:
	// world/* (shared per-world files -- physics, roster, skills)
	// and agents/<name>/* (per-agent entrypoint + per-deployment
	// role.md). materialiseWorldTree splits by prefix: world/* goes
	// to the host state dir (visible in the container via the /world
	// bind mount), agents/* is docker-cp'd directly into the running
	// container on top of the home tree seeded by syncAgentsInto.
	compileInput := compile.Input{
		Manifest:      opts.Manifest,
		VerifiedTools: verifiedTools,
		WorldID:       id,
		Agents:        rosterCompileAgents(rosterAgents),
	}
	tree, err := compile.Compile("claude-code", compileInput)
	if err != nil {
		a.backend.Stop(ctx, containerID)
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("compile world: %w", err)
	}
	if err := materialiseWorldTree(ctx, a.backend, containerID, tree, worldStateDir); err != nil {
		a.backend.Stop(ctx, containerID)
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("materialise world tree: %w", err)
	}
	opts.progress("world_state_written", "physics, faculties, roster, AGENTS.md")

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

// probeTools runs each resolved tool's Verify() commands inside the
// container and returns the names of the tools that passed. A
// failing Verify aborts the spawn: the only safe interpretation of
// "the tool I declared didn't verify" is that the image is broken,
// and silently proceeding would hand users a world without the
// capabilities they asked for.
//
// All checks run inside a SINGLE `sh -c` invocation - one container
// exec for the whole probe instead of one per tool. The old
// per-tool loop needed N sequential docker execs which added 2-4s
// of round-trip per tool to every spawn and blew the e2e hook
// timeout on CI runners with a modest tool set.
//
// This replaces an older implementation that maintained a parallel
// hardcoded @pack → binary map in packages/world/manifest and fell
// back to a static list of binaries. That map drifted as new tools
// were added and left unknown @pack refs erroring as "base image
// missing @spwn/foo". The catalog is now the single source of
// truth: each tool's Verify() method decides what must be present.
func (a *Architect) probeTools(ctx context.Context, containerID string, tools []ib.Tool) ([]string, error) {
	if len(tools) == 0 {
		return nil, nil
	}

	// Build a single shell script. Each block runs one tool's
	// checks; on any failure the block emits "FAIL <tool> :: <cmd>"
	// and the script exits 1. On success the block emits
	// "OK <tool>". The final exit 0 is only reached when every
	// tool passed.
	var b strings.Builder
	b.WriteString("set -e\n")
	for _, t := range tools {
		for _, check := range t.Verify() {
			fmt.Fprintf(&b,
				"if ! %s >/dev/null 2>&1; then echo 'FAIL %s :: %s'; exit 1; fi\n",
				check, t.Name(), check,
			)
		}
		fmt.Fprintf(&b, "echo 'OK %s'\n", t.Name())
	}

	output, err := a.backend.ExecOutput(ctx, containerID, []string{"sh", "-c", b.String()})
	if err != nil {
		// The script printed the offending "FAIL <tool> :: <cmd>"
		// line before exiting non-zero; surface it verbatim.
		failLine := strings.TrimSpace(output)
		for _, line := range strings.Split(failLine, "\n") {
			if strings.HasPrefix(line, "FAIL ") {
				failLine = line
				break
			}
		}
		return nil, fmt.Errorf(
			"world tool verification failed (%s).\n"+
				"Hint: rebuild with --force-rebuild, or remove the tool from the agent's tools list",
			failLine,
		)
	}

	verified := make([]string, 0, len(tools))
	for _, line := range strings.Split(output, "\n") {
		if name := strings.TrimPrefix(line, "OK "); name != line {
			verified = append(verified, strings.TrimSpace(name))
		}
	}
	return verified, nil
}

// buildWorkspaceBinds generates Docker bind specs for the resolved
// workspaces. Layout is uniform:
//
//   - 0 workspaces: no binds. /workspaces does not exist; the agent's
//     only writable space is its own home at /agents/<name>.
//   - 1+ workspaces: each mounted at /workspaces/<name>. There is no
//     special-cased single-workspace path — `ls /workspaces` always
//     tells the agent what projects it can touch.
func buildWorkspaceBinds(workspaces []models.Workspace) []string {
	if len(workspaces) == 0 {
		return nil
	}
	binds := make([]string, 0, len(workspaces))
	for _, ws := range workspaces {
		ro := ""
		if ws.ReadOnly {
			ro = ":ro"
		}
		binds = append(binds, fmt.Sprintf("%s:/workspaces/%s%s", ws.Path, ws.Name, ro))
	}
	return binds
}

// workspaceContainerPath returns the absolute path inside the container
// where a workspace named `name` is mounted. This is the single source
// of truth for the container-side workspace path scheme.
func workspaceContainerPath(name string, totalWorkspaces int) string {
	_ = totalWorkspaces // legacy parameter; layout is uniform now
	return "/workspaces/" + name
}

// worldStateDirFor returns the host-side directory where a given
// world's per-instance state is stored. Used by both spawn (initial
// write) and DeployAgent (roster regeneration).
func worldStateDirFor(worldID string) string {
	return filepath.Join(paths.LocalStateDir(), "world-states", worldID)
}

// initAgentDeploymentDirs creates the empty per-agent per-world
// filesystem skeleton inside the agent's persistent home:
//
//	~/.spwn/agents/<name>/worlds/<world-id>/
//	  inbox/    - messages received in this world
//	  outbox/   - messages I sent (audit trail)
//	  notes/    - my private notes for this world's project
//
// The content-bearing files (role.md, CLAUDE.md) are produced by the
// compiler and materialised alongside these dirs via
// materialiseWorldTree. Hot-deploy uses the same helper.
func initAgentDeploymentDirs(rec models.AgentRecord, worldID string) error {
	agentDir := agent.AgentDir(rec.Name)
	deploymentDir := filepath.Join(agentDir, "worlds", worldID)
	for _, sub := range []string{"inbox", "outbox", "notes"} {
		if err := os.MkdirAll(filepath.Join(deploymentDir, sub), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", sub, err)
		}
	}
	return nil
}

// materialiseWorldTree splits a compile.Tree into its two
// destination surfaces: every entry under world/* goes to
// worldStateDir on the host (bound as /world in the container),
// every entry under agents/* is docker-cp'd directly into the
// running container at /agents/<name>/ (no host side — under the
// new architecture the agent tree is not bind-mounted).
//
// Any other prefix is an error.
func materialiseWorldTree(ctx context.Context, be backend.Backend, containerID string, tree *compile.Tree, worldStateDir string) error {
	var worldErr error
	tree.Walk(func(path string, content []byte) {
		if worldErr != nil {
			return
		}
		switch {
		case strings.HasPrefix(path, "world/"):
			full := filepath.Join(worldStateDir, filepath.FromSlash(strings.TrimPrefix(path, "world/")))
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				worldErr = fmt.Errorf("mkdir %s: %w", filepath.Dir(full), err)
				return
			}
			if err := os.WriteFile(full, content, 0o644); err != nil {
				worldErr = fmt.Errorf("write %s: %w", full, err)
				return
			}
		case strings.HasPrefix(path, "agents/"):
			// Example path: "agents/neo/CLAUDE.md" →
			// container path "/agents/neo/CLAUDE.md". Deliver via
			// docker cp straight into the already-running container.
			containerPath := "/" + path
			if err := be.CopyTo(ctx, containerID, containerPath, content); err != nil {
				worldErr = fmt.Errorf("cp %s into container: %w", containerPath, err)
				return
			}
		default:
			worldErr = fmt.Errorf("unexpected tree path %q: claudecode runtime must namespace files under world/ or agents/", path)
			return
		}
	})
	return worldErr
}

// injectPluginRuntimeConfig computes the merged plugin config for the
// world's runtime backend and writes it into the container's runtime
// settings file.
//
// Current scope: only @spwn/claude-code has a known settings path
// (/home/spwn/.claude/settings.json). The container's baseline
// settings file — written by the claude_code tool's UserCommands at
// image build time — is read back, shallow-merged with every pack's
// Config(runtime) output (last write wins), and rewritten in place.
//
// When no plugin targets the runtime, this is a no-op: the baseline
// settings.json stays untouched.
//
// Additional runtimes can grow their own branch here as plugins for
// them materialize.
func injectPluginRuntimeConfig(ctx context.Context, be backend.Backend, containerID string, resolved []ib.Tool) error {
	// The pack-facing runtime identifier is the same as the image
	// builder's runtime tool name. Spawn always installs
	// @spwn/claude-code, so hard-code it here until a second runtime
	// lands (codex is built but has no plugin target yet).
	const runtimeName = "@spwn/claude-code"
	const settingsPath = "/home/spwn/.claude/settings.json"

	configs := ib.CollectRuntimeConfigs(resolved, runtimeName)
	if len(configs) == 0 {
		return nil
	}

	// Read the container's baseline settings.json. Missing file is
	// fine — an empty base layer merges cleanly.
	baseStdout, _ := be.ExecOutput(ctx, containerID, []string{"sh", "-c", "cat " + settingsPath + " 2>/dev/null || true"})
	base := []byte(strings.TrimSpace(baseStdout))

	merged, err := ib.MergeRuntimeConfig(base, configs...)
	if err != nil {
		return fmt.Errorf("merge config: %w", err)
	}

	// Encode the merged JSON as base64 and pipe it through the shell
	// so we don't have to worry about escaping JSON inside sh -c.
	encoded := base64.StdEncoding.EncodeToString(merged)
	script := fmt.Sprintf(
		"mkdir -p %s && printf '%%s' '%s' | base64 -d > %s",
		filepath.Dir(settingsPath), encoded, settingsPath,
	)
	if _, err := be.ExecOutput(ctx, containerID, []string{"sh", "-c", script}); err != nil {
		return fmt.Errorf("write merged settings: %w", err)
	}
	return nil
}

// agentHomesForSpawn returns the (agentName -> containerHomePath)
// map for every agent attached to this spawn. Single-agent worlds
// return one entry; multi-agent worlds return one per agent. The
// home paths are the in-container view (/agents/<name>). Each has
// a mirror under <project>/spwn/agents/<name>/ on the host, but the
// two are NOT a live bind mount — spawn copies host→container and
// graceful shutdown copies an allowlisted subset back.
func agentHomesForSpawn(opts SpawnOpts) map[string]string {
	homes := map[string]string{}
	if opts.AgentName != "" {
		homes[opts.AgentName] = "/agents/" + opts.AgentName
	}
	for _, a := range opts.Agents {
		if a.Name == "" {
			continue
		}
		homes[a.Name] = "/agents/" + a.Name
	}
	return homes
}

// writeRuntimeDefaultConfig writes the runtime provider's default
// config files into each agent's HOME *inside the running container*
// via docker cp. Under the new architecture the /agents tree is
// copied into the container at start (not bind-mounted), so dotfiles
// must also be delivered via docker cp — there's no shared host path
// to write to.
//
// Unlike the old host-side write, there's no "preserve existing"
// guard: each spawn re-seeds the default config, which is always
// safe because the container is ephemeral. Any user customizations
// to .claude.json that land in durable memory layers
// (journal/knowledge/playbooks/skills) will be synced back on
// graceful shutdown and re-seeded fresh next spawn.
//
// The runtime lookup is hardcoded to claude-code because every
// world spawn today installs @spwn/claude-code as a required tool.
// When the runtime becomes a per-world choice this should resolve
// off the world manifest.
func writeRuntimeDefaultConfig(ctx context.Context, be backend.Backend, containerID string, agentHomes map[string]string) error {
	rt, err := runtime.Get("claude-code")
	if err != nil {
		return fmt.Errorf("unknown runtime: %w", err)
	}

	for _, agentHome := range agentHomes {
		files := rt.DefaultConfigFiles(agentHome)
		if len(files) == 0 {
			continue
		}
		for relPath, content := range files {
			absPath := agentHome + "/" + relPath
			if err := be.CopyTo(ctx, containerID, absPath, content); err != nil {
				return fmt.Errorf("cp %s to container: %w", absPath, err)
			}
		}
	}
	return nil
}

// syncAgentsInto copies every agent's host-side home tree into the
// running container at /agents/<name>/. Called right after container
// start, before any agent command runs.
//
// Directories that don't exist on the host (e.g. a freshly scaffolded
// agent whose memory dirs are still empty) are silently skipped.
func syncAgentsInto(ctx context.Context, be backend.Backend, containerID string, agentHomes map[string]string) error {
	hostRoot := paths.AgentsDir()
	for agentName, containerHome := range agentHomes {
		hostDir := filepath.Join(hostRoot, agentName)
		if info, err := os.Stat(hostDir); err != nil || !info.IsDir() {
			// Nothing to copy — a no-op agent (uncommon in practice).
			continue
		}
		if err := be.CopyDirTo(ctx, containerID, containerHome, hostDir); err != nil {
			return fmt.Errorf("copy %s → %s: %w", hostDir, containerHome, err)
		}
	}
	return nil
}

// syncAgentsOutOf is the sync-back step run at graceful shutdown
// (Destroy). For each agent in the world, it copies the allowlisted
// memory directories (journal, knowledge, playbooks, skills) out of
// the container and into the host-side agent tree. Anything else
// inside /agents/<name>/ — identity files that didn't change, dotfiles,
// runtime caches — stays in the container and is discarded.
//
// Failures per-agent per-dir are logged as warnings but don't abort
// the destroy: a best-effort snapshot is better than nothing, and
// the container is about to be removed anyway.
func syncAgentsOutOf(ctx context.Context, be backend.Backend, containerID string, agentHomes map[string]string) []string {
	// Allowlist: only these subdirs of /agents/<name>/ sync back.
	// Everything else (dotfiles, .claude/, .npm/, .cache/, rebuilt
	// CLAUDE.md, etc.) stays inside the container.
	syncDirs := []string{"journal", "knowledge", "playbooks", "skills"}
	hostRoot := paths.AgentsDir()

	var warnings []string
	for agentName, containerHome := range agentHomes {
		hostDir := filepath.Join(hostRoot, agentName)
		for _, sub := range syncDirs {
			src := containerHome + "/" + sub
			dst := filepath.Join(hostDir, sub)
			if err := be.CopyDirFrom(ctx, containerID, src, dst); err != nil {
				warnings = append(warnings, fmt.Sprintf("sync %s/%s: %v", agentName, sub, err))
			}
		}
	}
	return warnings
}

// rosterColony adapts an agent record list into the claudecode
// ColonyAgentSpec list (used by colony.go's GenerateRoster call for
// roster regeneration on hot-deploy). New code should build a
// compile.Input instead.
func rosterColony(recs []models.AgentRecord) []claudecode.ColonyAgentSpec {
	out := make([]claudecode.ColonyAgentSpec, 0, len(recs))
	for _, r := range recs {
		out = append(out, claudecode.ColonyAgentSpec{Name: r.Name, Role: r.Role})
	}
	return out
}

// rosterCompileAgents projects the world record's agent list onto
// the compile.Input shape.
func rosterCompileAgents(recs []models.AgentRecord) []compile.AgentInput {
	out := make([]compile.AgentInput, 0, len(recs))
	for _, r := range recs {
		out = append(out, compile.AgentInput{Name: r.Name, Role: r.Role})
	}
	return out
}
