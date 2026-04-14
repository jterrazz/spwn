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

	"spwn.sh/packages/mind"
	plugins "spwn.sh/catalog/plugins"
	runtimes "spwn.sh/catalog/runtimes"
	tools "spwn.sh/catalog/tools"
	ib "spwn.sh/packages/image"
	ibbase "spwn.sh/packages/image/base"
	"spwn.sh/packages/world/internal/backend"
	"spwn.sh/packages/world/internal/labels"
	"spwn.sh/packages/world/internal/manifest"
	"spwn.sh/packages/world/internal/models"
	"spwn.sh/packages/world/internal/physics"
	"spwn.sh/packages/base"
	"spwn.sh/packages/base/activity"
	"spwn.sh/packages/base/auth"
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

var defaultProbeList = []string{
	"bash", "sh", "git", "node", "npm", "python3", "curl", "wget", "jq", "claude", "go", "rustc", "gcc", "make",
}

// Spawn creates a new world.
func (a *Architect) Spawn(ctx context.Context, opts SpawnOpts) (*SpawnResult, error) {
	var warnings []string

	// Generate ID
	id := base.GenerateWorldID(opts.ConfigName)

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
		binds = append(binds, base.BaseDir()+":/home/spwn/.spwn")
	}

	// SINGLE shared agents mount: ~/.spwn/agents/ → /agents (rw).
	// Every container sees every agent's persistent home. New agents
	// added via DeployAgent appear instantly through this bind without
	// any container restart, because the kernel sees the new directory
	// the moment it's created on the host.
	binds = append(binds, base.AgentsDir()+":/agents")

	// Validate each named agent's mind directory and profile. We
	// validate but no longer mount per-agent - all visibility goes
	// through the single /agents bind above.
	agentNamesToValidate := []string{}
	if len(opts.Agents) > 0 {
		for _, spec := range opts.Agents {
			agentNamesToValidate = append(agentNamesToValidate, spec.Name)
		}
	} else if opts.AgentName != "" {
		agentNamesToValidate = append(agentNamesToValidate, opts.AgentName)
	}
	for _, name := range agentNamesToValidate {
		if err := mind.ValidateMind(name); err != nil {
			return nil, err
		}
		opts.progress("mind_validated", name)
		// Parse agent.yaml (optional). Used for future composition validation
		// against the world's available tools.
		if _, err := manifest.LoadAgent(mind.AgentDir(name)); err != nil {
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
	if err := tools.RegisterDefaults(reg); err != nil {
		return nil, fmt.Errorf("register tools: %w", err)
	}
	if err := runtimes.RegisterDefaults(reg); err != nil {
		return nil, fmt.Errorf("register runtimes: %w", err)
	}
	if err := plugins.RegisterDefaults(reg); err != nil {
		return nil, fmt.Errorf("register plugins: %w", err)
	}

	// Always include runtime essentials, then add user-specified tools
	// and plugins on top. The registry deduplicates and resolves
	// dependencies; plugins share the tool resolution pipeline.
	required := []string{"@spwn/unix", "@spwn/node", "@spwn/claude-code", "@spwn/cli"}
	toolList := append(required, opts.Manifest.Tools...)
	toolList = append(toolList, opts.Manifest.Plugins...)

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

	resolvedTools, resolveErr := reg.Resolve(toolList)
	if resolveErr != nil {
		return nil, fmt.Errorf("resolve tools: %w", resolveErr)
	}

	if !explicitImage {
		opts.progress("image_building", image)

		builder := ib.New(reg, a.backend)

		_, err := builder.Build(ctx, ib.BuildRequest{
			BaseDockerfile: ibbase.WorldDockerfile,
			Tools:          toolList,
			Tag:            image,
			Version:        base.WorldImageVersion,
			SkipVerify:     true, // probeTools handles verification below
			LogWriter:      opts.logWriter(),
		})
		if err != nil {
			return nil, fmt.Errorf("build world image: %w", err)
		}
		opts.progress("image_ready", image)
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
	binds = append(binds, base.CredentialsDir()+":/credentials:ro")

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
		worldRecord.AgentID = base.GenerateAgentID(opts.Agents[0].Name)
		for _, spec := range opts.Agents {
			role := manifest.DefaultRole(spec.Role)
			worldRecord.Agents = append(worldRecord.Agents, models.AgentRecord{
				Name:    spec.Name,
				AgentID: base.GenerateAgentID(spec.Name),
				Role:    role,
				Status:  models.StatusIdle,
			})
		}
	} else if opts.AgentName != "" {
		worldRecord.AgentID = base.GenerateAgentID(opts.AgentName)
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
		if err := initAgentDeployment(rec, id); err != nil {
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

	// Merge plugin runtime-config into the container's runtime settings
	// file. Currently only the claude-code backend has a known target
	// path; additional runtimes can grow their own branch as needed.
	if err := injectPluginRuntimeConfig(ctx, a.backend, containerID, resolvedTools); err != nil {
		a.backend.Stop(ctx, containerID)
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("inject plugin runtime config: %w", err)
	}

	// Probe tools
	verifiedTools, err := a.probeTools(ctx, containerID, opts.Manifest.Tools)
	if err != nil {
		a.backend.Stop(ctx, containerID)
		a.backend.Remove(ctx, containerID)
		return nil, err
	}
	opts.progress("tools_probed", fmt.Sprintf("%d verified", len(verifiedTools)))

	// Generate world-state files. They live on the HOST under
	// ~/.spwn/world-states/<id>/ and are visible inside the container
	// at /world/ via the bind mount. Writing to the host means the
	// user can also `vim ~/.spwn/world-states/<id>/shared/notes/...`
	// from their terminal.
	type worldFile struct {
		path    string
		content []byte
	}
	files := []worldFile{
		{"physics.md", []byte(physics.GeneratePhysics(opts.Manifest))},
		{"faculties.md", []byte(physics.GenerateFaculties(verifiedTools))},
		{"AGENTS.md", []byte(physics.AgentsBook)},
		{"roster.md", []byte(physics.GenerateRoster(id, rosterColony(rosterAgents)))},
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(worldStateDir, f.path), f.content, 0o644); err != nil {
			a.backend.Stop(ctx, containerID)
			a.backend.Remove(ctx, containerID)
			return nil, fmt.Errorf("write %s: %w", f.path, err)
		}
	}
	for skillName, skillContent := range physics.SystemSkills() {
		if err := os.WriteFile(filepath.Join(worldStateDir, "skills", skillName), []byte(skillContent), 0o644); err != nil {
			warnings = append(warnings, fmt.Sprintf("write skill %s: %v", skillName, err))
		}
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

// probeTools verifies which tools are available in the container.
func (a *Architect) probeTools(ctx context.Context, containerID string, declaredTools []string) ([]string, error) {
	// Expand @packs and merge with default probe list
	expanded := manifest.ExpandTools(declaredTools)
	probeList := mergeUnique(expanded, defaultProbeList)

	// Build probe command
	var checks []string
	for _, b := range probeList {
		checks = append(checks, fmt.Sprintf(`command -v "%s" >/dev/null 2>&1 && echo "%s"`, b, b))
	}
	cmd := []string{"sh", "-c", strings.Join(checks, "; ") + "; true"}

	output, err := a.backend.ExecOutput(ctx, containerID, cmd)
	if err != nil {
		return nil, fmt.Errorf("probe tools: %w", err)
	}

	verified := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			verified[line] = true
		}
	}

	// Verify all declared tools exist
	for _, e := range expanded {
		if !verified[e] {
			return nil, fmt.Errorf("world requires tool '%s' but the base image does not provide it.\nHint: Add %s to the container image, or remove it from the config's tools", e, e)
		}
	}

	// Return all verified tools
	var result []string
	for _, e := range probeList {
		if verified[e] {
			result = append(result, e)
		}
	}
	return result, nil
}

func mergeUnique(a, b []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// buildWorkspaceBinds generates Docker bind specs for the resolved
// workspaces. Layout is uniform:
//
//   - 0 workspaces: no binds. /work does not exist; the agent's only
//     writable space is its own home at /agents/<name>.
//   - 1+ workspaces: each mounted at /work/<name>. There is no
//     special-cased single-workspace path - `ls /work` always tells
//     the agent what projects it can touch.
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
		binds = append(binds, fmt.Sprintf("%s:/work/%s%s", ws.Path, ws.Name, ro))
	}
	return binds
}

// workspaceContainerPath returns the absolute path inside the container
// where a workspace named `name` is mounted. This is the single source
// of truth for the container-side workspace path scheme.
func workspaceContainerPath(name string, totalWorkspaces int) string {
	_ = totalWorkspaces // legacy parameter; layout is uniform now
	return "/work/" + name
}

// worldStateDirFor returns the host-side directory where a given
// world's per-instance state is stored. Used by both spawn (initial
// write) and DeployAgent (roster regeneration).
func worldStateDirFor(worldID string) string {
	return filepath.Join(base.LocalStateDir(), "world-states", worldID)
}

// initAgentDeployment creates the per-agent per-world filesystem layout
// inside the agent's persistent home dir on the host:
//
//	~/.spwn/agents/<name>/worlds/<world-id>/
//	  inbox/    - messages received in this world
//	  outbox/   - messages I sent (audit trail)
//	  notes/    - my private notes for this world's project
//	  role.md   - what role I play here
//
// It also writes CLAUDE.md at the agent root so the Claude Code
// runtime loads the agent's identity on startup. The cwd is set to
// /agents/<name>/ so CLAUDE.md is the first thing it reads.
//
// The single /agents bind on the world container makes these dirs
// visible at /agents/<name>/worlds/<world-id>/ instantly. Hot-deploy
// uses the same helper.
func initAgentDeployment(rec models.AgentRecord, worldID string) error {
	agentDir := mind.AgentDir(rec.Name)
	deploymentDir := filepath.Join(agentDir, "worlds", worldID)
	for _, sub := range []string{"inbox", "outbox", "notes"} {
		if err := os.MkdirAll(filepath.Join(deploymentDir, sub), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", sub, err)
		}
	}
	role := rec.Role
	if role == "" {
		role = "worker"
	}
	roleContent := fmt.Sprintf("# Role in %s\n\n%s\n", worldID, role)
	if err := os.WriteFile(filepath.Join(deploymentDir, "role.md"), []byte(roleContent), 0o644); err != nil {
		return fmt.Errorf("write role.md: %w", err)
	}

	// Write CLAUDE.md - the entry point Claude Code reads on startup.
	// It loads the agent's profile and tells the runtime where to find
	// the world manual and system skills. Without this, the agent runs
	// as a generic Claude instance with no identity.
	claudeMD := generateAgentCLAUDEMD(rec.Name, role)
	claudePath := filepath.Join(agentDir, "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte(claudeMD), 0o644); err != nil {
		return fmt.Errorf("write CLAUDE.md: %w", err)
	}

	return nil
}

// generateAgentCLAUDEMD creates the CLAUDE.md that Claude Code reads
// on startup. It includes the profile inline (so it's always loaded)
// and references the world files.
func generateAgentCLAUDEMD(agentName, role string) string {
	return fmt.Sprintf(`# %s

You are **%s**, a spwn agent with role: %s.

## Your identity

Read your full profile and behavioral instructions from:

@core/profile.md

Follow the voice, style, and purpose defined there. You are NOT a generic assistant - you are %s.

## Your world

- Read %s for your operating manual (how memory, skills, and communication work).
- Read %s for the rules of this world (network, filesystem, communication).
- Read %s to see what tools are physically available.
- Read %s for system skills (mind management, collaboration, evolution).

## Key rules

1. **Read your profile first** before doing anything else. Your identity shapes how you respond.
2. Save important discoveries to your knowledge (write to %s).
3. After significant work, check if a playbook should be created in %s.
4. **Messaging**: to send a message to another agent, write a .json or .md file to %s. To check YOUR inbox, read %s. Read %s for the full messaging protocol.
5. Never modify system files in /world/ (physics.md, faculties.md, AGENTS.md are read-only).
`, agentName, agentName, role, agentName,
		"`/world/AGENTS.md`",
		"`/world/physics.md`",
		"`/world/faculties.md`",
		"`/world/skills/`",
		"`./knowledge/`",
		"`./playbooks/`",
		"`/world/inbox/<their-name>/`",
		fmt.Sprintf("`/world/inbox/%s/`", agentName),
		"`/world/skills/collaboration.md`")
}

// injectPluginRuntimeConfig computes the merged plugin config for the
// world's runtime backend and writes it into the container's runtime
// settings file.
//
// Current scope: only @spwn/claude-code has a known settings path
// (/home/spwn/.claude/settings.json). The container's baseline
// settings file — written by the claude_code tool's UserCommands at
// image build time — is read back, shallow-merged with every plugin's
// Config(runtime) output (last write wins), and rewritten in place.
//
// When no plugin targets the runtime, this is a no-op: the baseline
// settings.json stays untouched.
//
// Additional runtimes can grow their own branch here as plugins for
// them materialize.
func injectPluginRuntimeConfig(ctx context.Context, be backend.Backend, containerID string, resolved []ib.Tool) error {
	// The plugin-facing runtime identifier is the same as the image
	// builder's runtime tool name. Spawn always installs
	// @spwn/claude-code, so hard-code it here until a second runtime
	// lands (codex is built but has no plugin target yet).
	const runtimeName = "@spwn/claude-code"
	const settingsPath = "/home/spwn/.claude/settings.json"

	configs := ib.CollectPluginConfigs(resolved, runtimeName)
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

// rosterColony adapts an agent record list into the physics package's
// ColonyAgentSpec list (used by GenerateRoster).
func rosterColony(recs []models.AgentRecord) []physics.ColonyAgentSpec {
	out := make([]physics.ColonyAgentSpec, 0, len(recs))
	for _, r := range recs {
		out = append(out, physics.ColonyAgentSpec{Name: r.Name, Role: r.Role})
	}
	return out
}
