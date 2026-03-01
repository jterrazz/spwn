package architect

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jterrazz/universe/internal/backend"
	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/id"
	"github.com/jterrazz/universe/internal/journal"
	"github.com/jterrazz/universe/internal/manifest"
	"github.com/jterrazz/universe/internal/mind"
	"github.com/jterrazz/universe/internal/physics"
	"github.com/jterrazz/universe/internal/session"
	"github.com/jterrazz/universe/internal/state"
)

// defaultProbeList is the set of binaries probed during element introspection.
var defaultProbeList = []string{
	"bash", "sh", "git", "node", "npm", "python3", "curl", "wget", "jq", "claude", "go", "rustc", "gcc", "make",
}

// Architect orchestrates universe lifecycle.
type Architect struct {
	backend backend.Backend
	state   *state.Store
}

// New creates an Architect with the given backend and state store.
func New(b backend.Backend, s *state.Store) *Architect {
	return &Architect{backend: b, state: s}
}

// parseMemory converts human-readable memory strings to bytes.
func parseMemory(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	multipliers := map[string]int64{
		"k": 1024, "kb": 1024,
		"m": 1024 * 1024, "mb": 1024 * 1024,
		"g": 1024 * 1024 * 1024, "gb": 1024 * 1024 * 1024,
	}

	for suffix, mult := range multipliers {
		if strings.HasSuffix(s, suffix) {
			numStr := strings.TrimSuffix(s, suffix)
			var val int64
			if _, err := fmt.Sscanf(numStr, "%d", &val); err != nil {
				return 0, fmt.Errorf("invalid memory value: %s", s)
			}
			return val * mult, nil
		}
	}

	// Try plain number (bytes).
	var val int64
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil {
		return 0, fmt.Errorf("invalid memory value: %s", s)
	}
	return val, nil
}

// Spawn creates a new universe from the given options.
func (a *Architect) Spawn(ctx context.Context, opts config.SpawnOptions) (*config.Universe, error) {
	universeID := id.Generate()

	// Parse memory.
	memBytes, err := parseMemory(opts.Manifest.Physics.Constants.Memory)
	if err != nil {
		return nil, fmt.Errorf("invalid memory: %w", err)
	}

	// Resolve workspace to absolute path.
	workspace := opts.Workspace
	if workspace != "" {
		workspace, err = filepath.Abs(workspace)
		if err != nil {
			return nil, fmt.Errorf("resolving workspace path: %w", err)
		}
		if _, err := os.Stat(workspace); err != nil {
			return nil, fmt.Errorf("workspace directory %s does not exist", workspace)
		}
	}

	// Build bind mounts.
	var binds []string
	if workspace != "" {
		binds = append(binds, fmt.Sprintf("%s:/workspace", workspace))
	}

	// Mount Mind if agent specified.
	var mindPath string
	if opts.AgentName != "" {
		if err := mind.Validate(opts.AgentName); err != nil {
			return nil, err
		}
		dir, err := mind.AgentDir(opts.AgentName)
		if err != nil {
			return nil, err
		}
		mindPath = dir
		binds = append(binds, fmt.Sprintf("%s:/mind", dir))
	}

	// Create container.
	containerCfg := backend.ContainerConfig{
		Image:       config.BaseImage,
		Name:        universeID,
		CPU:         int64(opts.Manifest.Physics.Constants.CPU),
		Memory:      memBytes,
		PidsLimit:   int64(opts.Manifest.Physics.Laws.MaxProcesses),
		NetworkMode: opts.Manifest.Physics.Laws.Network,
		Binds:       binds,
	}

	containerID, err := a.backend.Create(ctx, containerCfg)
	if err != nil {
		return nil, fmt.Errorf("provisioning container: %w", err)
	}

	// Start container.
	if err := a.backend.Start(ctx, containerID); err != nil {
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("starting container: %w", err)
	}

	// Element introspection: probe the container for available binaries.
	verifiedElements, err := a.probeElements(ctx, containerID, opts.Manifest.Physics.Elements)
	if err != nil {
		a.backend.Remove(ctx, containerID)
		return nil, err
	}

	// Generate and copy physics.md into the container.
	physicsMd := physics.Generate(opts.Manifest, verifiedElements)
	if err := a.backend.CopyTo(ctx, containerID, "/universe/", strings.NewReader(physicsMd)); err != nil {
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("copying physics.md: %w", err)
	}

	// Determine initial status.
	status := config.StatusIdle
	if opts.AgentName != "" {
		status = config.StatusCreating
	}

	u := &config.Universe{
		ID:          universeID,
		Origin:      opts.Manifest.Physics.Origin,
		Agent:       opts.AgentName,
		Backend:     config.DefaultBackend,
		Status:      status,
		ContainerID: containerID,
		Workspace:   workspace,
		MindPath:    mindPath,
		CreatedAt:   time.Now(),
		Manifest:    opts.Manifest,
	}

	if err := a.state.Save(*u); err != nil {
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("saving state: %w", err)
	}

	return u, nil
}

// probeElements checks which binaries are available inside the container.
// It merges manifest-declared elements (expanded from packs) with a default probe list,
// then verifies each one. If a manifest-declared element is missing, it returns an error.
func (a *Architect) probeElements(ctx context.Context, containerID string, declaredElements []string) ([]string, error) {
	// Expand packs from manifest declarations.
	expandedDeclared := manifest.ExpandElements(declaredElements)

	// Build the full probe list: declared + defaults, deduplicated.
	seen := make(map[string]bool)
	var probeList []string
	for _, e := range expandedDeclared {
		if !seen[e] {
			seen[e] = true
			probeList = append(probeList, e)
		}
	}
	for _, e := range defaultProbeList {
		if !seen[e] {
			seen[e] = true
			probeList = append(probeList, e)
		}
	}

	if len(probeList) == 0 {
		return nil, nil
	}

	// Probe all at once with a single exec.
	shellCmd := fmt.Sprintf("for b in %s; do command -v \"$b\" >/dev/null 2>&1 && echo \"$b\"; done",
		strings.Join(probeList, " "))

	output, err := a.backend.ExecOutput(ctx, containerID, []string{"sh", "-c", shellCmd})
	if err != nil {
		// Introspection failed — fall back to declared elements.
		return expandedDeclared, nil
	}

	// Parse output into a set of verified binaries.
	verified := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			verified[line] = true
		}
	}

	// Check that all manifest-declared elements were found.
	for _, e := range expandedDeclared {
		if !verified[e] {
			return nil, fmt.Errorf("error: universe requires element %q but origin does not provide it.\nEnsure the origin image includes it, or remove it from physics.elements", e)
		}
	}

	// Return all verified elements (declared + discovered).
	var result []string
	for _, e := range probeList {
		if verified[e] {
			result = append(result, e)
		}
	}

	return result, nil
}

// SpawnAgent brings an agent to life inside an existing universe.
func (a *Architect) SpawnAgent(ctx context.Context, universeID, agentName string) (int, error) {
	u, err := a.state.Get(universeID)
	if err != nil {
		return -1, err
	}

	if u.Agent != "" && u.Agent != agentName {
		return -1, fmt.Errorf("universe %s already has an agent (%s). A universe hosts exactly one agent. Destroy it first, or spawn a new universe", universeID, u.Agent)
	}

	// Update state.
	u.Agent = agentName
	u.Status = config.StatusRunning
	if err := a.state.Save(u); err != nil {
		return -1, err
	}

	// Session management.
	sessionID := session.GenerateID(agentName, universeID)
	existing, _ := session.Load(u.MindPath, universeID)

	// Build the Claude Code command.
	systemPrompt := `Read /universe/physics.md to understand the reality you exist in. Read /mind/personas/ to understand who you are. Your workspace is at /workspace.`

	var cmd []string
	if existing != nil {
		// Resume existing session.
		cmd = []string{"claude", "--session-id", sessionID, "--resume"}
	} else {
		// New session.
		cmd = []string{"claude", "--system-prompt", systemPrompt, "--session-id", sessionID}
	}

	// Record start time for journal.
	startTime := time.Now()

	// Exec Claude Code interactively.
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	exitCode, execErr := a.backend.Exec(ctx, u.ContainerID, backend.ExecConfig{
		Cmd: cmd,
		Env: []string{fmt.Sprintf("ANTHROPIC_API_KEY=%s", apiKey)},
		TTY: true,
	})

	endTime := time.Now()

	// After Claude exits, update status.
	u.Status = config.StatusIdle
	a.state.Save(u)

	// Save session (best-effort).
	now := time.Now()
	sess := &session.Session{
		SessionID:  sessionID,
		UniverseID: universeID,
		AgentName:  agentName,
		UpdatedAt:  now,
	}
	if existing != nil {
		sess.CreatedAt = existing.CreatedAt
	} else {
		sess.CreatedAt = now
	}
	if err := session.Save(u.MindPath, sess); err != nil {
		log.Printf("warning: failed to save session: %v", err)
	}

	// Append journal entry (best-effort).
	outcome := "completed"
	if exitCode != 0 {
		outcome = "failed"
	}
	if _, err := journal.Append(u.MindPath, journal.Entry{
		UniverseID: universeID,
		Origin:     u.Origin,
		Outcome:    outcome,
		ExitCode:   exitCode,
		Duration:   endTime.Sub(startTime),
		StartedAt:  startTime,
		EndedAt:    endTime,
	}); err != nil {
		log.Printf("warning: failed to write journal entry: %v", err)
	}

	if execErr != nil {
		return exitCode, fmt.Errorf("running Claude Code: %w", execErr)
	}

	return exitCode, nil
}

// SpawnAgentDetached starts an agent in the background without waiting for completion.
func (a *Architect) SpawnAgentDetached(ctx context.Context, universeID, agentName string) error {
	u, err := a.state.Get(universeID)
	if err != nil {
		return err
	}

	if u.Agent != "" && u.Agent != agentName {
		return fmt.Errorf("universe %s already has an agent (%s). A universe hosts exactly one agent. Destroy it first, or spawn a new universe", universeID, u.Agent)
	}

	// Update state.
	u.Agent = agentName
	u.Status = config.StatusRunning
	if err := a.state.Save(u); err != nil {
		return err
	}

	// Session management.
	sessionID := session.GenerateID(agentName, universeID)
	existing, _ := session.Load(u.MindPath, universeID)

	systemPrompt := `Read /universe/physics.md to understand the reality you exist in. Read /mind/personas/ to understand who you are. Your workspace is at /workspace.`

	var cmd []string
	if existing != nil {
		cmd = []string{"claude", "--session-id", sessionID, "--resume"}
	} else {
		cmd = []string{"claude", "--system-prompt", systemPrompt, "--session-id", sessionID}
	}

	// Save session upfront (we won't get a callback when it finishes).
	now := time.Now()
	sess := &session.Session{
		SessionID:  sessionID,
		UniverseID: universeID,
		AgentName:  agentName,
		UpdatedAt:  now,
	}
	if existing != nil {
		sess.CreatedAt = existing.CreatedAt
	} else {
		sess.CreatedAt = now
	}
	session.Save(u.MindPath, sess)

	// Exec in background.
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	return a.backend.ExecDetached(ctx, u.ContainerID, backend.ExecConfig{
		Cmd: cmd,
		Env: []string{fmt.Sprintf("ANTHROPIC_API_KEY=%s", apiKey)},
	})
}

// Destroy stops and removes a universe.
func (a *Architect) Destroy(ctx context.Context, universeID string) error {
	u, err := a.state.Get(universeID)
	if err != nil {
		return err
	}

	// Stop and remove the container.
	a.backend.Stop(ctx, u.ContainerID)
	if err := a.backend.Remove(ctx, u.ContainerID); err != nil {
		return fmt.Errorf("removing container: %w", err)
	}

	// Update state.
	u.Status = config.StatusDestroyed
	return a.state.Delete(universeID)
}

// List returns all universes.
func (a *Architect) List(ctx context.Context) ([]config.Universe, error) {
	return a.state.List()
}

// Inspect returns details of a specific universe.
func (a *Architect) Inspect(ctx context.Context, universeID string) (config.Universe, error) {
	return a.state.Get(universeID)
}

// Logs streams container output for a universe.
func (a *Architect) Logs(ctx context.Context, universeID string, follow bool, tail string) (io.ReadCloser, error) {
	u, err := a.state.Get(universeID)
	if err != nil {
		return nil, err
	}
	return a.backend.Logs(ctx, u.ContainerID, backend.LogsConfig{Follow: follow, Tail: tail})
}

// Attach opens an interactive shell inside a running universe.
func (a *Architect) Attach(ctx context.Context, universeID string) (int, error) {
	u, err := a.state.Get(universeID)
	if err != nil {
		return -1, err
	}

	running, err := a.backend.IsRunning(ctx, u.ContainerID)
	if err != nil || !running {
		return -1, fmt.Errorf("universe %s is not running", universeID)
	}

	return a.backend.Exec(ctx, u.ContainerID, backend.ExecConfig{
		Cmd: []string{"/bin/bash"},
		TTY: true,
	})
}
