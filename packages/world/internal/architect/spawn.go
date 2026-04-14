package architect

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"spwn.sh/packages/agent"
	ib "spwn.sh/packages/imagebuilder"
	"spwn.sh/packages/imagebuilder/base"
	"spwn.sh/packages/imagebuilder/catalog"
	"spwn.sh/packages/world/internal/backend"
	"spwn.sh/packages/world/internal/labels"
	"spwn.sh/packages/world/internal/manifest"
	"spwn.sh/packages/world/internal/models"
	"spwn.sh/packages/world/internal/physics"
	"spwn.sh/packages/foundation"
	"spwn.sh/packages/foundation/activity"
	"spwn.sh/packages/foundation/auth"
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
	Image         string                     // Override base image (used for testing). Defaults to foundation.WorldImage.
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
	id := foundation.GenerateWorldID(opts.ConfigName)

	// Parse memory
	memBytes, err := parseMemory(opts.Manifest.Physics.Constants.Memory)
	if err != nil {
		return nil, fmt.Errorf("invalid memory: %w", err)
	}

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
		binds = append(binds, foundation.BaseDir()+":/home/spwn/.spwn")
	}

	// SINGLE shared agents mount: ~/.spwn/agents/ → /agents (rw).
	// Every container sees every agent's persistent home. New agents
	// added via DeployAgent appear instantly through this bind without
	// any container restart, because the kernel sees the new directory
	// the moment it's created on the host.
	binds = append(binds, foundation.AgentsDir()+":/agents")

	// Validate each named agent's mind directory and profile. We
	// validate but no longer mount per-agent — all visibility goes
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
	// exact image, don't rebuild" — they're how tests inject a mock
	// runtime. Only when neither is set do we auto-build from the base
	// Dockerfile + tool catalog.
	image := foundation.WorldImage
	explicitImage := false
	if envImage := os.Getenv("SPWN_BASE_IMAGE"); envImage != "" {
		image = envImage
		explicitImage = true
	}
	if opts.Image != "" {
		image = opts.Image
		explicitImage = true
	}

	if !explicitImage {
		opts.progress("image_building", image)

		// Build image using imagebuilder with manifest tools
		reg := ib.NewRegistry()
		if err := catalog.RegisterDefaults(reg); err != nil {
			return nil, fmt.Errorf("register catalog: %w", err)
		}
		builder := ib.New(reg, a.backend)

		// Always include runtime essentials, then add user-specified tools on top.
		// The registry deduplicates and resolves dependencies.
		required := []string{"@spwn/unix", "@spwn/node", "@spwn/claude-code", "@spwn/cli"}
		tools := append(required, opts.Manifest.Tools...)

		// Deduplicate
		seen := make(map[string]bool)
		deduped := make([]string, 0, len(tools))
		for _, t := range tools {
			if !seen[t] {
				seen[t] = true
				deduped = append(deduped, t)
			}
		}
		tools = deduped

		_, err := builder.Build(ctx, ib.BuildRequest{
			BaseDockerfile: base.WorldDockerfile,
			Tools:          tools,
			Tag:            image,
			Version:        foundation.WorldImageVersion,
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

	// Sync credentials to bind-mountable directory (live — containers see updates)
	if err := auth.SyncCredentials(); err != nil {
		warnings = append(warnings, fmt.Sprintf("credential sync: %v", err))
	}
	binds = append(binds, foundation.CredentialsDir()+":/credentials:ro")

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
	// canonical store — see packages/world/internal/labels.
	worldRecord := models.World{
		ID:         id,
		Name:       opts.Name,
		Config:     opts.ConfigName,
		Agent:      opts.AgentName,
		Backend:    foundation.DefaultBackend,
		Workspaces: resolvedWorkspaces,
		CreatedAt:  time.Now(),
		Manifest:   opts.Manifest,
	}
	if len(opts.Agents) > 0 {
		worldRecord.Agent = opts.Agents[0].Name
		worldRecord.AgentID = foundation.GenerateAgentID(opts.Agents[0].Name)
		for _, spec := range opts.Agents {
			role := manifest.DefaultRole(spec.Role)
			worldRecord.Agents = append(worldRecord.Agents, models.AgentRecord{
				Name:    spec.Name,
				AgentID: foundation.GenerateAgentID(spec.Name),
				Role:    role,
				Status:  models.StatusIdle,
			})
		}
	} else if opts.AgentName != "" {
		worldRecord.AgentID = foundation.GenerateAgentID(opts.AgentName)
		worldRecord.Agents = []models.AgentRecord{{
			Name:    opts.AgentName,
			AgentID: worldRecord.AgentID,
			Role:    "worker",
			Status:  models.StatusIdle,
		}}
	}

	// Per-world state directory on the host. This is the canonical
	// /world/ inside the container — physics, faculties, roster,
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

	// Create container
	containerCfg := backend.ContainerConfig{
		Image:       image,
		Name:        id,
		CPU:         int64(opts.Manifest.Physics.Constants.CPU),
		Memory:      memBytes,
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
	// container are the canonical store — this struct is just what we
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

func parseMemory(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty memory string.\nSpecify memory like '512m', '1g', or '4g'")
	}

	var multiplier int64
	var numStr string

	if strings.HasSuffix(s, "gb") {
		multiplier = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(s, "gb")
	} else if strings.HasSuffix(s, "g") {
		multiplier = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(s, "g")
	} else if strings.HasSuffix(s, "mb") {
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(s, "mb")
	} else if strings.HasSuffix(s, "m") {
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(s, "m")
	} else if strings.HasSuffix(s, "kb") {
		multiplier = 1024
		numStr = strings.TrimSuffix(s, "kb")
	} else if strings.HasSuffix(s, "k") {
		multiplier = 1024
		numStr = strings.TrimSuffix(s, "k")
	} else {
		n, err := strconv.ParseInt(s, 10, 64)
		return n, err
	}

	n, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse memory %q: %w", s, err)
	}
	return n * multiplier, nil
}

// buildWorkspaceBinds generates Docker bind specs for the resolved
// workspaces. Layout is uniform:
//
//   - 0 workspaces: no binds. /work does not exist; the agent's only
//     writable space is its own home at /agents/<name>.
//   - 1+ workspaces: each mounted at /work/<name>. There is no
//     special-cased single-workspace path — `ls /work` always tells
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
	return filepath.Join(foundation.LocalStateDir(), "world-states", worldID)
}

// initAgentDeployment creates the per-agent per-world filesystem layout
// inside the agent's persistent home dir on the host:
//
//	~/.spwn/agents/<name>/worlds/<world-id>/
//	  inbox/    — messages received in this world
//	  outbox/   — messages I sent (audit trail)
//	  notes/    — my private notes for this world's project
//	  role.md   — what role I play here
//
// It also writes CLAUDE.md at the agent root so the Claude Code
// runtime loads the agent's identity on startup. The cwd is set to
// /agents/<name>/ so CLAUDE.md is the first thing it reads.
//
// The single /agents bind on the world container makes these dirs
// visible at /agents/<name>/worlds/<world-id>/ instantly. Hot-deploy
// uses the same helper.
func initAgentDeployment(rec models.AgentRecord, worldID string) error {
	agentDir := agent.AgentDir(rec.Name)
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

	// Write CLAUDE.md — the entry point Claude Code reads on startup.
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

Follow the voice, style, and purpose defined there. You are NOT a generic assistant — you are %s.

## Your world

- Read %s for your operating manual (how memory, skills, and communication work).
- Read %s for the physics and resource constraints of this world.
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

// rosterColony adapts an agent record list into the physics package's
// ColonyAgentSpec list (used by GenerateRoster).
func rosterColony(recs []models.AgentRecord) []physics.ColonyAgentSpec {
	out := make([]physics.ColonyAgentSpec, 0, len(recs))
	for _, r := range recs {
		out = append(out, physics.ColonyAgentSpec{Name: r.Name, Role: r.Role})
	}
	return out
}
