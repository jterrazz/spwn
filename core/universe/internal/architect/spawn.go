package architect

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"spwn.sh/core/agent"
	"spwn.sh/core/gate"
	"spwn.sh/core/universe/internal/backend"
	"spwn.sh/platform/images"
	"spwn.sh/core/universe/internal/manifest"
	"spwn.sh/core/universe/internal/models"
	"spwn.sh/core/universe/internal/physics"
	"spwn.sh/core/foundation"
)

// SpawnResult is returned by Spawn with the universe and any non-fatal warnings.
type SpawnResult struct {
	Universe *models.World
	Warnings []string
}

// SpawnOpts configures world creation.
type SpawnOpts struct {
	ConfigName    string
	AgentName     string
	Workspace     string
	Manifest      models.Manifest
	Image         string                     // Override base image (used for testing). Defaults to foundation.BaseImage.
	InvokeHandler gate.InvokeHandler         // Override gate handler (used for testing). Defaults to stub.
	OnProgress    func(event, detail string) // Optional callback at each milestone.
	LogWriter     io.Writer                  // Receives Docker build output. nil defaults to io.Discard.
	Agents        []AgentSpec                // Multi-agent list (alternative to single AgentName).
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

	// Resolve workspace to absolute path
	workspace := ""
	if opts.Workspace != "" {
		workspace, err = filepath.Abs(opts.Workspace)
		if err != nil {
			return nil, fmt.Errorf("resolve workspace: %w", err)
		}
		if _, err := os.Stat(workspace); err != nil {
			return nil, fmt.Errorf("workspace %s not found", workspace)
		}
	}

	// Build mounts
	var binds []string
	if workspace != "" {
		binds = append(binds, workspace+":/workspace")
	}

	// Mount Mind(s) for agents
	mindPath := ""
	if len(opts.Agents) > 0 {
		// Multi-agent: mount each agent's mind at /mind/<name>
		for _, spec := range opts.Agents {
			if err := agent.ValidateMind(spec.Name); err != nil {
				return nil, err
			}
			agentDir := agent.AgentDir(spec.Name)
			binds = append(binds, agentDir+":/mind/"+spec.Name)

			// Validate life manifest body requirements
			life, err := manifest.LoadLife(agentDir)
			if err != nil {
				return nil, fmt.Errorf("load life manifest for %s: %w", spec.Name, err)
			}
			if life != nil {
				expandedElements := manifest.ExpandElements(opts.Manifest.Elements)
				if err := manifest.ValidateBody(life, expandedElements); err != nil {
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
		mindPath = agent.AgentDir(opts.AgentName)
		binds = append(binds, mindPath+":/mind")

		// Validate life manifest body requirements
		life, err := manifest.LoadLife(mindPath)
		if err != nil {
			return nil, fmt.Errorf("load life manifest: %w", err)
		}
		if life != nil {
			expandedElements := manifest.ExpandElements(opts.Manifest.Elements)
			if err := manifest.ValidateBody(life, expandedElements); err != nil {
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

	// Resolve image (env override for testing, then opts, then default)
	image := foundation.BaseImage
	if envImage := os.Getenv("SPWN_BASE_IMAGE"); envImage != "" {
		image = envImage
	}
	if opts.Image != "" {
		image = opts.Image
	}

	// Ensure image exists (auto-build for default image, fail for custom/test images)
	if opts.Image == "" {
		if err := a.backend.EnsureImage(ctx, image, images.Dockerfile, opts.logWriter()); err != nil {
			return nil, fmt.Errorf("ensure base image: %w", err)
		}
	} else {
		exists, err := a.backend.ImageExists(ctx, image)
		if err != nil {
			return nil, fmt.Errorf("check image: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("image %s not found", image)
		}
	}
	opts.progress("image_ready", image)

	// Forward AI provider credentials to the container
	var env []string
	for _, key := range []string{
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"GOOGLE_API_KEY",
		"CLAUDE_CODE_OAUTH_TOKEN",
		"ANTHROPIC_AUTH_TOKEN",
	} {
		if val := os.Getenv(key); val != "" {
			env = append(env, key+"="+val)
		}
	}

	// Auto-extract OAuth token from macOS Keychain for subscription auth
	if !hasEnv(env, "CLAUDE_CODE_OAUTH_TOKEN") && !hasEnv(env, "ANTHROPIC_API_KEY") {
		if token := extractKeychainToken(); token != "" {
			env = append(env, "CLAUDE_CODE_OAUTH_TOKEN="+token)
		}
	}

	// Mount Claude auth directory (for config, not credentials — token via env var)
	home, _ := os.UserHomeDir()
	claudeAuthDir := filepath.Join(home, ".claude")
	if _, err := os.Stat(claudeAuthDir); err == nil {
		binds = append(binds, claudeAuthDir+":/home/spwn/.claude")
	}

	// Create container
	containerCfg := backend.ContainerConfig{
		Image:       image,
		Name:        id,
		CPU:         int64(opts.Manifest.Physics.Constants.CPU),
		Memory:      memBytes,
		PidsLimit:   int64(opts.Manifest.Physics.Laws.MaxProcesses),
		NetworkMode: "bridge",
		Binds:       binds,
		Env:         env,
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
		opts.progress("gates_bridged", fmt.Sprintf("%d element(s)", len(opts.Manifest.Gate)))
	}

	// Probe elements
	verifiedElements, err := a.probeElements(ctx, containerID, opts.Manifest.Elements)
	if err != nil {
		a.backend.Stop(ctx, containerID)
		a.backend.Remove(ctx, containerID)
		return nil, err
	}

	// Generate physics.md
	physicsContent := physics.GeneratePhysics(opts.Manifest)
	if err := a.backend.CopyTo(ctx, containerID, "universe/physics.md", []byte(physicsContent)); err != nil {
		a.backend.Stop(ctx, containerID)
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("copy physics.md: %w", err)
	}

	// Generate faculties.md
	facultiesContent := physics.GenerateFaculties(verifiedElements, opts.Manifest.Gate)
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
			tier := manifest.DefaultTier(spec.Tier)

			// Build list of other agents (everyone except this one)
			var others []physics.AgentInfo
			var governorName string
			for j, other := range opts.Agents {
				if i == j {
					continue
				}
				otherTier := manifest.DefaultTier(other.Tier)
				others = append(others, physics.AgentInfo{Name: other.Name, Tier: otherTier})
				if otherTier == "governor" {
					governorName = other.Name
				}
			}

			agentCtx := physics.GenerateAgentContext(physics.AgentContextOpts{
				AgentName:   spec.Name,
				Tier:        tier,
				WorldID:     id,
				Workspace:   workspace,
				Elements:    verifiedElements,
				CPU:         opts.Manifest.Physics.Constants.CPU,
				Memory:      opts.Manifest.Physics.Constants.Memory,
				Timeout:     opts.Manifest.Physics.Constants.Timeout,
				OtherAgents: others,
				Governor:    governorName,
			})
			if err := a.backend.CopyTo(ctx, containerID, "world/AGENT-"+spec.Name+".md", []byte(agentCtx)); err != nil {
				a.backend.Stop(ctx, containerID)
				a.backend.Remove(ctx, containerID)
				return nil, fmt.Errorf("copy AGENT-%s.md: %w", spec.Name, err)
			}
		}
	} else if opts.AgentName != "" {
		// Single agent: generate /world/AGENT.md
		agentCtx := physics.GenerateAgentContext(physics.AgentContextOpts{
			AgentName: opts.AgentName,
			Tier:      "citizen",
			WorldID:   id,
			Workspace: workspace,
			Elements:  verifiedElements,
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

	// Create inbox directories for agent communication
	a.backend.ExecOutput(ctx, containerID, []string{"mkdir", "-p", "/world/inbox"})
	if len(opts.Agents) > 0 {
		for _, spec := range opts.Agents {
			a.backend.ExecOutput(ctx, containerID, []string{"mkdir", "-p", "/world/inbox/" + spec.Name})
		}
	} else if opts.AgentName != "" {
		a.backend.ExecOutput(ctx, containerID, []string{"mkdir", "-p", "/world/inbox/" + opts.AgentName})
	}

	// Build universe record
	u := models.World{
		ID:          id,
		Config:      opts.ConfigName,
		Agent:       opts.AgentName,
		Backend:     foundation.DefaultBackend,
		ContainerID: containerID,
		Workspace:   workspace,
		MindPath:    mindPath,
		GateDir:     gateDir,
		Status:      models.StatusIdle,
		CreatedAt:   time.Now(),
		Manifest:    opts.Manifest,
	}

	if len(opts.Agents) > 0 {
		// Multi-agent: populate Agents slice and set primary Agent/AgentID for backward compat
		u.Agent = opts.Agents[0].Name
		u.AgentID = foundation.GenerateAgentID(opts.Agents[0].Name)
		for _, spec := range opts.Agents {
			tier := manifest.DefaultTier(spec.Tier)
			u.Agents = append(u.Agents, models.AgentRecord{
				Name:    spec.Name,
				AgentID: foundation.GenerateAgentID(spec.Name),
				Tier:    tier,
				Status:  models.StatusIdle,
			})
		}
	} else if opts.AgentName != "" {
		u.AgentID = foundation.GenerateAgentID(opts.AgentName)
	}

	// Save state
	if err := a.state.Save(u); err != nil {
		a.backend.Stop(ctx, containerID)
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("save state: %w", err)
	}

	return &SpawnResult{Universe: &u, Warnings: warnings}, nil
}

// probeElements verifies which elements are available in the container.
func (a *Architect) probeElements(ctx context.Context, containerID string, declaredElements []string) ([]string, error) {
	// Expand @packs and merge with default probe list
	expanded := manifest.ExpandElements(declaredElements)
	probeList := mergeUnique(expanded, defaultProbeList)

	// Build probe command
	var checks []string
	for _, b := range probeList {
		checks = append(checks, fmt.Sprintf(`command -v "%s" >/dev/null 2>&1 && echo "%s"`, b, b))
	}
	cmd := []string{"sh", "-c", strings.Join(checks, "; ") + "; true"}

	output, err := a.backend.ExecOutput(ctx, containerID, cmd)
	if err != nil {
		return nil, fmt.Errorf("probe elements: %w", err)
	}

	verified := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			verified[line] = true
		}
	}

	// Verify all declared elements exist
	for _, e := range expanded {
		if !verified[e] {
			return nil, fmt.Errorf("world requires element '%s' but the base image does not provide it.\nHint: Add %s to the container image, or remove it from the config's elements", e, e)
		}
	}

	// Return all verified elements
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

func hasEnv(env []string, key string) bool {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return true
		}
	}
	return false
}

// extractKeychainToken reads the cached Claude OAuth token from ~/.spwn/.auth-token.
// If not cached, attempts to extract from macOS Keychain and caches for future use.
func extractKeychainToken() string {
	// Try cached token first (no keychain popup)
	cachePath := filepath.Join(foundation.BaseDir(), ".auth-token")
	if data, err := os.ReadFile(cachePath); err == nil {
		token := strings.TrimSpace(string(data))
		if token != "" {
			return token
		}
	}

	// Fall back to macOS Keychain (will prompt user once)
	out, err := exec.Command("security", "find-generic-password", "-s", "Claude Code-credentials", "-w").Output()
	if err != nil {
		return ""
	}

	var creds struct {
		ClaudeAiOauth struct {
			AccessToken string `json:"accessToken"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(out, &creds); err != nil {
		return ""
	}

	// Cache for future use
	if creds.ClaudeAiOauth.AccessToken != "" {
		os.MkdirAll(foundation.BaseDir(), 0755)
		os.WriteFile(cachePath, []byte(creds.ClaudeAiOauth.AccessToken), 0600)
	}

	return creds.ClaudeAiOauth.AccessToken
}

func parseMemory(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty memory string")
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
