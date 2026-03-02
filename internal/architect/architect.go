package architect

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jterrazz/universe/internal/backend"
	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/manifest"
	"github.com/jterrazz/universe/internal/mind"
	"github.com/jterrazz/universe/internal/physics"
	"github.com/jterrazz/universe/internal/state"
)

// Architect orchestrates universe lifecycle.
type Architect struct {
	backend backend.Backend
	state   *state.Store
}

// SpawnOpts configures universe creation.
type SpawnOpts struct {
	ConfigName string
	AgentName  string
	Workspace  string
	Manifest   config.UniverseManifest
	Image      string // Override base image (used for testing). Defaults to config.BaseImage.
}

// New creates an Architect with the given backend and state store.
func New(b backend.Backend, s *state.Store) *Architect {
	return &Architect{backend: b, state: s}
}

var defaultProbeList = []string{
	"bash", "sh", "git", "node", "npm", "python3", "curl", "wget", "jq", "claude", "go", "rustc", "gcc", "make",
}

// Spawn creates a new universe.
func (a *Architect) Spawn(ctx context.Context, opts SpawnOpts) (*config.Universe, error) {
	// Generate ID
	id := config.GenerateUniverseID(opts.ConfigName)

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

	// Mount Mind if agent specified
	mindPath := ""
	if opts.AgentName != "" {
		if err := mind.Validate(opts.AgentName); err != nil {
			return nil, err
		}
		mindPath = mind.AgentDir(opts.AgentName)
		binds = append(binds, mindPath+":/mind")
	}

	// Resolve image
	image := config.BaseImage
	if opts.Image != "" {
		image = opts.Image
	}

	// Check image exists
	exists, err := a.backend.ImageExists(ctx, image)
	if err != nil {
		return nil, fmt.Errorf("check image: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("base image %s not found. Run 'make build-image' first", image)
	}

	// Create container
	containerCfg := backend.ContainerConfig{
		Image:       image,
		Name:        id,
		CPU:         int64(opts.Manifest.Physics.Constants.CPU),
		Memory:      memBytes,
		PidsLimit:   int64(opts.Manifest.Physics.Laws.MaxProcesses),
		NetworkMode: opts.Manifest.Physics.Laws.Network,
		Binds:       binds,
	}

	containerID, err := a.backend.Create(ctx, containerCfg)
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	if err := a.backend.Start(ctx, containerID); err != nil {
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("start container: %w", err)
	}

	// Probe technologies
	verifiedTechs, err := a.probeTechnologies(ctx, containerID, opts.Manifest.Technologies)
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
	facultiesContent := physics.GenerateFaculties(verifiedTechs, opts.Manifest.Gate)
	if err := a.backend.CopyTo(ctx, containerID, "universe/faculties.md", []byte(facultiesContent)); err != nil {
		a.backend.Stop(ctx, containerID)
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("copy faculties.md: %w", err)
	}

	// Build universe record
	u := config.Universe{
		ID:          id,
		Config:      opts.ConfigName,
		Agent:       opts.AgentName,
		Backend:     config.DefaultBackend,
		ContainerID: containerID,
		Workspace:   workspace,
		MindPath:    mindPath,
		Status:      config.StatusIdle,
		Manifest:    opts.Manifest,
	}

	if opts.AgentName != "" {
		u.AgentID = config.GenerateAgentID(opts.AgentName)
	}

	// Save state
	if err := a.state.Save(u); err != nil {
		a.backend.Stop(ctx, containerID)
		a.backend.Remove(ctx, containerID)
		return nil, fmt.Errorf("save state: %w", err)
	}

	return &u, nil
}

// SpawnAgent execs Claude Code interactively inside a universe.
func (a *Architect) SpawnAgent(ctx context.Context, universeID, agentName string) error {
	u, err := a.state.Get(universeID)
	if err != nil {
		return err
	}

	running, err := a.backend.IsRunning(ctx, u.ContainerID)
	if err != nil {
		return fmt.Errorf("check container: %w", err)
	}
	if !running {
		return fmt.Errorf("universe %s is not running", universeID)
	}

	// Update status
	a.state.UpdateStatus(universeID, config.StatusRunning)

	// Build Claude Code command
	cmd := []string{"claude", "--dangerously-skip-permissions"}

	// Pass ANTHROPIC_API_KEY
	var env []string
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		env = append(env, "ANTHROPIC_API_KEY="+apiKey)
	}

	exitCode, err := a.backend.Exec(ctx, u.ContainerID, backend.ExecConfig{
		Cmd: cmd,
		Env: env,
		TTY: true,
	})
	if err != nil {
		a.state.UpdateStatus(universeID, config.StatusIdle)
		return fmt.Errorf("exec claude: %w", err)
	}

	a.state.UpdateStatus(universeID, config.StatusIdle)

	if exitCode != 0 {
		return fmt.Errorf("agent exited with code %d", exitCode)
	}
	return nil
}

// SpawnAgentDetached starts Claude Code in the background.
func (a *Architect) SpawnAgentDetached(ctx context.Context, universeID, agentName string) error {
	u, err := a.state.Get(universeID)
	if err != nil {
		return err
	}

	running, err := a.backend.IsRunning(ctx, u.ContainerID)
	if err != nil {
		return fmt.Errorf("check container: %w", err)
	}
	if !running {
		return fmt.Errorf("universe %s is not running", universeID)
	}

	a.state.UpdateStatus(universeID, config.StatusRunning)

	cmd := []string{"claude", "--dangerously-skip-permissions"}

	var env []string
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		env = append(env, "ANTHROPIC_API_KEY="+apiKey)
	}

	return a.backend.ExecDetached(ctx, u.ContainerID, backend.ExecConfig{
		Cmd: cmd,
		Env: env,
		TTY: false,
	})
}

// Destroy stops and removes a universe.
func (a *Architect) Destroy(ctx context.Context, universeID string) (*config.Universe, error) {
	u, err := a.state.Get(universeID)
	if err != nil {
		return nil, err
	}

	a.backend.Stop(ctx, u.ContainerID)
	a.backend.Remove(ctx, u.ContainerID)

	a.state.Delete(universeID)

	return u, nil
}

// List returns all universes.
func (a *Architect) List(ctx context.Context) ([]config.Universe, error) {
	return a.state.List()
}

// Inspect returns a universe by ID.
func (a *Architect) Inspect(ctx context.Context, universeID string) (*config.Universe, error) {
	return a.state.Get(universeID)
}

// Logs streams container output.
func (a *Architect) Logs(ctx context.Context, universeID string, follow bool, tail string) (interface{ Read([]byte) (int, error); Close() error }, error) {
	u, err := a.state.Get(universeID)
	if err != nil {
		return nil, err
	}
	return a.backend.Logs(ctx, u.ContainerID, backend.LogsConfig{
		Follow: follow,
		Tail:   tail,
	})
}

// Attach opens an interactive shell into a running universe.
func (a *Architect) Attach(ctx context.Context, universeID string) error {
	u, err := a.state.Get(universeID)
	if err != nil {
		return err
	}

	running, err := a.backend.IsRunning(ctx, u.ContainerID)
	if err != nil {
		return fmt.Errorf("check container: %w", err)
	}
	if !running {
		return fmt.Errorf("universe %s is not running", universeID)
	}

	_, err = a.backend.Exec(ctx, u.ContainerID, backend.ExecConfig{
		Cmd: []string{"bash"},
		TTY: true,
	})
	return err
}

// probeTechnologies verifies which technologies are available in the container.
func (a *Architect) probeTechnologies(ctx context.Context, containerID string, declaredTechs []string) ([]string, error) {
	// Expand @packs and merge with default probe list
	expanded := manifest.ExpandTechnologies(declaredTechs)
	probeList := mergeUnique(expanded, defaultProbeList)

	// Build probe command
	var checks []string
	for _, b := range probeList {
		checks = append(checks, fmt.Sprintf(`command -v "%s" >/dev/null 2>&1 && echo "%s"`, b, b))
	}
	cmd := []string{"sh", "-c", strings.Join(checks, "; ")}

	output, err := a.backend.ExecOutput(ctx, containerID, cmd)
	if err != nil {
		return nil, fmt.Errorf("probe technologies: %w", err)
	}

	verified := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			verified[line] = true
		}
	}

	// Verify all declared technologies exist
	for _, t := range expanded {
		if !verified[t] {
			return nil, fmt.Errorf("universe requires technology '%s' but the base image does not provide it.\nHint: Add %s to the container image, or remove it from the config's technologies", t, t)
		}
	}

	// Return all verified technologies
	var result []string
	for _, t := range probeList {
		if verified[t] {
			result = append(result, t)
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
