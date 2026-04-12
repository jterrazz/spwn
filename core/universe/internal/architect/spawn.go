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

	"spwn.sh/core/agent"
	"spwn.sh/core/gate"
	ib "spwn.sh/core/imagebuilder"
	"spwn.sh/core/imagebuilder/base"
	"spwn.sh/core/imagebuilder/catalog"
	"spwn.sh/core/universe/internal/backend"
	"spwn.sh/core/universe/internal/labels"
	"spwn.sh/core/universe/internal/manifest"
	"spwn.sh/core/universe/internal/models"
	"spwn.sh/core/universe/internal/physics"
	"spwn.sh/core/foundation"
	"spwn.sh/core/foundation/activity"
	"spwn.sh/core/foundation/auth"
)

// SpawnResult is returned by Spawn with the universe and any non-fatal warnings.
type SpawnResult struct {
	Universe *models.World
	Warnings []string
}

// SpawnOpts configures world creation.
type SpawnOpts struct {
	ConfigName    string
	Name          string // Optional user-facing display name.
	AgentName     string
	Runtime       string                     // Agent runtime name (e.g., "claude-code", "codex"). Defaults to "claude-code".
	Workspaces    []models.Workspace
	Manifest      models.Manifest
	Image         string                     // Override base image (used for testing). Defaults to foundation.WorldImage.
	InvokeHandler gate.InvokeHandler         // Override gate handler (used for testing). Defaults to stub.
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

	// Set runtime adapter if specified
	if opts.Runtime != "" {
		if err := a.SetRuntime(opts.Runtime); err != nil {
			warnings = append(warnings, fmt.Sprintf("runtime %q not found, using default", opts.Runtime))
		}
	}

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
		profile, err := manifest.LoadProfile(agent.AgentDir(name))
		if err != nil {
			return nil, fmt.Errorf("load profile manifest for %s: %w", name, err)
		}
		if profile != nil {
			expandedTools := manifest.ExpandTools(opts.Manifest.Tools)
			if err := manifest.ValidateRequires(profile, expandedTools); err != nil {
				return nil, err
			}
		}
	}

	// Start gate TCP server and set up bridge scripts if bridges are configured
	gateDir := ""
	var gateSrv *gate.Server
	if len(opts.Manifest.Gate) > 0 {
		handler := opts.InvokeHandler
		if handler == nil {
			handler = gate.StubHandler()
		}

		gateSrv = gate.NewServer(handler)
		if err := gateSrv.Start(); err != nil {
			return nil, fmt.Errorf("start gate server: %w", err)
		}

		gateDir, err = os.MkdirTemp("", "spwn-gate-")
		if err != nil {
			gateSrv.Stop()
			return nil, fmt.Errorf("create gate dir: %w", err)
		}
		binds = append(binds, gateDir+":/gate")

		// Write wrapper scripts with the TCP port baked in
		if err := gate.SetupBridges(gateDir, opts.Manifest.Gate, gateSrv.Port()); err != nil {
			gateSrv.Stop()
			os.RemoveAll(gateDir)
			return nil, fmt.Errorf("setup gate bridges: %w", err)
		}
	}

	// Resolve image (env override for testing, then opts, then default with imagebuilder)
	image := foundation.WorldImage
	if envImage := os.Getenv("SPWN_BASE_IMAGE"); envImage != "" {
		image = envImage
	}
	if opts.Image != "" {
		image = opts.Image
	}

	// Ensure image exists
	if opts.Image == "" {
		opts.progress("image_building", image)

		// Build image using imagebuilder with manifest tools
		reg := ib.NewRegistry()
		catalog.RegisterDefaults(reg)
		builder := ib.New(reg, a.backend)

		// Always include runtime essentials, then add user-specified tools on top.
		// The registry deduplicates and resolves dependencies.
		runtimeTool := "@spwn/claude-code"
		if opts.Runtime == "codex" {
			runtimeTool = "@spwn/codex"
		} else if opts.Runtime != "" && opts.Runtime != "claude-code" {
			// For other runtimes, try @spwn/{runtime} if it exists
			candidate := "@spwn/" + opts.Runtime
			if reg.Get(candidate) != nil {
				runtimeTool = candidate
			}
		}
		required := []string{"@spwn/unix", "@spwn/node", runtimeTool, "@spwn/cli"}
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
	// canonical store — see core/universe/internal/labels.
	runtimeName := opts.Runtime
	if runtimeName == "" {
		runtimeName = "claude-code"
	}
	worldRecord := models.World{
		ID:          id,
		Name:        opts.Name,
		Config:      opts.ConfigName,
		Agent:       opts.AgentName,
		Backend:     foundation.DefaultBackend,
		Workspaces:  resolvedWorkspaces,
		GateDir:     gateDir,
		Runtime:     runtimeName,
		CreatedAt:   time.Now(),
		Manifest:    opts.Manifest,
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

	// Gate bridges require host access to reach the host-side server.
	if gateSrv != nil {
		containerCfg.ExtraHosts = []string{"host.docker.internal:host-gateway"}
	}

	containerID, err := a.backend.Create(ctx, containerCfg)
	if err != nil {
		if gateSrv != nil {
			gateSrv.Stop()
		}
		return nil, fmt.Errorf("create container: %w", err)
	}

	if err := a.backend.Start(ctx, containerID); err != nil {
		a.backend.Remove(ctx, containerID)
		if gateSrv != nil {
			gateSrv.Stop()
		}
		return nil, fmt.Errorf("start container: %w", err)
	}
	opts.progress("container_created", id)

	// Install gate bridges inside container
	if gateSrv != nil {
		// Symlink bridge wrappers to /usr/local/bin/ and extend PATH
		for _, bridge := range opts.Manifest.Gate {
			_, symErr := a.backend.ExecOutput(ctx, containerID, []string{
				"ln", "-sf", "/gate/bin/" + bridge.As, "/usr/local/bin/" + bridge.As,
			})
			if symErr != nil {
				warnings = append(warnings, fmt.Sprintf("failed to symlink bridge %s: %v", bridge.As, symErr))
			}
		}

		// Add /gate/bin to PATH for all shells
		pathScript := []byte("export PATH=/gate/bin:$PATH\n")
		if err := a.backend.CopyTo(ctx, containerID, "etc/profile.d/gate.sh", pathScript); err != nil {
			warnings = append(warnings, fmt.Sprintf("failed to write gate PATH extension: %v", err))
		}

		a.gates[id] = gateSrv
		opts.progress("gates_bridged", fmt.Sprintf("%d bridge(s)", len(opts.Manifest.Gate)))
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
		{"faculties.md", []byte(physics.GenerateFaculties(verifiedTools, opts.Manifest.Gate))},
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

	return &SpawnResult{Universe: &u, Warnings: warnings}, nil
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
	return filepath.Join(foundation.BaseDir(), "world-states", worldID)
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
	// It loads the agent's persona and tells the runtime where to find
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
// on startup. It includes the persona inline (so it's always loaded)
// and references the world files.
func generateAgentCLAUDEMD(agentName, role string) string {
	return fmt.Sprintf(`# %s

You are **%s**, a spwn agent with role: %s.

## Your identity

Read your full persona and behavioral instructions from:

@core/persona.md

Follow the voice, style, and purpose defined there. You are NOT a generic assistant — you are %s.

## Your world

- Read %s for your operating manual (how memory, skills, and communication work).
- Read %s for the physics and resource constraints of this world.
- Read %s to see what tools are physically available.
- Read %s for system skills (mind management, collaboration, evolution).

## Key rules

1. **Read your persona first** before doing anything else. Your identity shapes how you respond.
2. Save important discoveries to your knowledge (write to %s).
3. After significant work, check if a playbook should be created in %s.
4. Communicate with other agents via your inbox/outbox in your current world deployment.
5. Never modify files in /world/ (read-only system area).
`, agentName, agentName, role, agentName,
		"`/world/AGENTS.md`",
		"`/world/physics.md`",
		"`/world/faculties.md`",
		"`/world/skills/`",
		"`./knowledge/`",
		"`./playbooks/`")
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
