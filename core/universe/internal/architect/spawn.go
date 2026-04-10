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

	// Mount knowledge read-only into agent worlds
	knowledgeDir := foundation.KnowledgeDir()
	if _, statErr := os.Stat(knowledgeDir); statErr == nil {
		binds = append(binds, knowledgeDir+":/world/knowledge:ro")
	}

	// Mount Mind(s) for agents
	mindPath := ""
	if len(opts.Agents) > 0 {
		// Multi-agent: mount each agent's mind at /mind/<name>
		for _, spec := range opts.Agents {
			if err := agent.ValidateMind(spec.Name); err != nil {
				return nil, err
			}
			opts.progress("mind_validated", spec.Name)
			agentDir := agent.AgentDir(spec.Name)
			binds = append(binds, agentDir+":/mind/"+spec.Name)

			// Validate profile manifest requirements
			profile, err := manifest.LoadProfile(agentDir)
			if err != nil {
				return nil, fmt.Errorf("load profile manifest for %s: %w", spec.Name, err)
			}
			if profile != nil {
				expandedTools := manifest.ExpandTools(opts.Manifest.Tools)
				if err := manifest.ValidateRequires(profile, expandedTools); err != nil {
					return nil, err
				}
			}
			opts.progress("mind_mounted", spec.Name+" → /mind/"+spec.Name)
		}
		// Use first agent's dir as primary mindPath for backward compat
		mindPath = agent.AgentDir(opts.Agents[0].Name)
	} else if opts.AgentName != "" {
		// Single-agent (backward compatible)
		if err := agent.ValidateMind(opts.AgentName); err != nil {
			return nil, err
		}
		opts.progress("mind_validated", opts.AgentName)
		mindPath = agent.AgentDir(opts.AgentName)
		binds = append(binds, mindPath+":/mind")

		// Validate profile manifest requirements
		profile, err := manifest.LoadProfile(mindPath)
		if err != nil {
			return nil, fmt.Errorf("load profile manifest: %w", err)
		}
		if profile != nil {
			expandedTools := manifest.ExpandTools(opts.Manifest.Tools)
			if err := manifest.ValidateRequires(profile, expandedTools); err != nil {
				return nil, err
			}
		}
		opts.progress("mind_mounted", opts.AgentName+" → /mind")
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
		MindPath:    mindPath,
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

	// Generate physics.md
	physicsContent := physics.GeneratePhysics(opts.Manifest)
	if err := a.backend.CopyTo(ctx, containerID, "universe/physics.md", []byte(physicsContent)); err != nil {
		a.backend.Stop(ctx, containerID)
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("copy physics.md: %w", err)
	}

	// Generate faculties.md
	facultiesContent := physics.GenerateFaculties(verifiedTools, opts.Manifest.Gate)
	if err := a.backend.CopyTo(ctx, containerID, "universe/faculties.md", []byte(facultiesContent)); err != nil {
		a.backend.Stop(ctx, containerID)
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("copy faculties.md: %w", err)
	}
	opts.progress("faculties_generated", "physics.md, faculties.md")

	// Generate AGENT.md (personalized agent context)
	if len(opts.Agents) > 0 {
		// Multi-agent: generate AGENT-{name}.md for each agent
		for i, spec := range opts.Agents {
			role := manifest.DefaultRole(spec.Role)

			// Build list of other agents (everyone except this one)
			var others []physics.AgentInfo
			var chiefName string
			for j, other := range opts.Agents {
				if i == j {
					continue
				}
				otherRole := manifest.DefaultRole(other.Role)
				others = append(others, physics.AgentInfo{Name: other.Name, Role: otherRole})
				if otherRole == "chief" {
					chiefName = other.Name
				}
			}

			agentCtx := physics.GenerateAgentContext(physics.AgentContextOpts{
				AgentName:   spec.Name,
				Role:        role,
				WorldID:     id,
				Workspaces:  resolvedWorkspaces,
				Tools:       verifiedTools,
				CPU:         opts.Manifest.Physics.Constants.CPU,
				Memory:      opts.Manifest.Physics.Constants.Memory,
				Timeout:     opts.Manifest.Physics.Constants.Timeout,
				OtherAgents: others,
				Chief:       chiefName,
			})
			if err := a.backend.CopyTo(ctx, containerID, "world/AGENT-"+spec.Name+".md", []byte(agentCtx)); err != nil {
				a.backend.Stop(ctx, containerID)
				a.backend.Remove(ctx, containerID)
				return nil, fmt.Errorf("copy AGENT-%s.md: %w", spec.Name, err)
			}
		}

		// Also generate a combined /world/AGENT.md listing all agents
		var colonySpecs []physics.ColonyAgentSpec
		for _, spec := range opts.Agents {
			colonySpecs = append(colonySpecs, physics.ColonyAgentSpec{
				Name: spec.Name,
				Role: manifest.DefaultRole(spec.Role),
			})
		}
		combinedCtx := physics.GenerateColonyContext(id, colonySpecs)
		if err := a.backend.CopyTo(ctx, containerID, "world/AGENT.md", []byte(combinedCtx)); err != nil {
			a.backend.Stop(ctx, containerID)
			a.backend.Remove(ctx, containerID)
			return nil, fmt.Errorf("copy colony AGENT.md: %w", err)
		}
	} else if opts.AgentName != "" {
		// Single agent: generate /world/AGENT.md
		agentCtx := physics.GenerateAgentContext(physics.AgentContextOpts{
			AgentName:  opts.AgentName,
			Role:       "worker",
			WorldID:    id,
			Workspaces: resolvedWorkspaces,
			Tools:      verifiedTools,
			CPU:       opts.Manifest.Physics.Constants.CPU,
			Memory:    opts.Manifest.Physics.Constants.Memory,
			Timeout:   opts.Manifest.Physics.Constants.Timeout,
		})
		if err := a.backend.CopyTo(ctx, containerID, "world/AGENT.md", []byte(agentCtx)); err != nil {
			a.backend.Stop(ctx, containerID)
			a.backend.Remove(ctx, containerID)
			return nil, fmt.Errorf("copy AGENT.md: %w", err)
		}
	}

	// Write system files: AGENTS.md (operating manual) + skill guides
	if err := a.backend.CopyTo(ctx, containerID, "world/AGENTS.md", []byte(physics.AgentsBook)); err != nil {
		a.backend.Stop(ctx, containerID)
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("copy AGENTS.md: %w", err)
	}
	for skillName, skillContent := range physics.SystemSkills() {
		if err := a.backend.CopyTo(ctx, containerID, "world/skills/"+skillName, []byte(skillContent)); err != nil {
			a.backend.Stop(ctx, containerID)
			a.backend.Remove(ctx, containerID)
			return nil, fmt.Errorf("copy skill %s: %w", skillName, err)
		}
	}
	opts.progress("system_files_written", "AGENTS.md, 4 skill guides")

	// Create inbox directories for agent communication
	a.backend.ExecOutput(ctx, containerID, []string{"mkdir", "-p", "/world/inbox"})
	if len(opts.Agents) > 0 {
		for _, spec := range opts.Agents {
			a.backend.ExecOutput(ctx, containerID, []string{"mkdir", "-p", "/world/inbox/" + spec.Name})
		}
	} else if opts.AgentName != "" {
		a.backend.ExecOutput(ctx, containerID, []string{"mkdir", "-p", "/world/inbox/" + opts.AgentName})
	}

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

// buildWorkspaceBinds generates Docker bind specs for the resolved workspaces.
// Layout (always rooted at /workspace so `ls /workspace` tells the agent what
// it can work with):
//   - 0 workspaces: no binds — container uses image-baked /workspace.
//   - 1 workspace:  mounted directly at /workspace (legacy flat layout).
//   - 2+:           each mounted at /workspace/<name>. Running `ls /workspace`
//                   shows the workspace names; agents cd into one to drill in.
func buildWorkspaceBinds(workspaces []models.Workspace) []string {
	if len(workspaces) == 0 {
		return nil
	}
	binds := make([]string, 0, len(workspaces))
	if len(workspaces) == 1 {
		ws := workspaces[0]
		ro := ""
		if ws.ReadOnly {
			ro = ":ro"
		}
		return append(binds, fmt.Sprintf("%s:/workspace%s", ws.Path, ro))
	}
	for _, ws := range workspaces {
		ro := ""
		if ws.ReadOnly {
			ro = ":ro"
		}
		binds = append(binds, fmt.Sprintf("%s:/workspace/%s%s", ws.Path, ws.Name, ro))
	}
	return binds
}

// workspaceContainerPath returns the absolute path inside the container where
// a workspace named `name` is mounted, given the total number of workspaces.
// This is the single source of truth for the container-side path scheme.
func workspaceContainerPath(name string, totalWorkspaces int) string {
	if totalWorkspaces == 1 {
		return "/workspace"
	}
	return "/workspace/" + name
}
