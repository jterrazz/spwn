package architect

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jterrazz/universe/container"
	"github.com/jterrazz/universe/internal/backend"
	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/gate"
	"github.com/jterrazz/universe/internal/manifest"
	"github.com/jterrazz/universe/internal/mind"
	"github.com/jterrazz/universe/internal/physics"
)

// SpawnResult is returned by Spawn with the universe and any non-fatal warnings.
type SpawnResult struct {
	Universe *config.Universe
	Warnings []string
}

// SpawnOpts configures universe creation.
type SpawnOpts struct {
	ConfigName    string
	AgentName     string
	Workspace     string
	Manifest      config.UniverseManifest
	Image         string                     // Override base image (used for testing). Defaults to config.BaseImage.
	InvokeHandler gate.InvokeHandler         // Override gate handler (used for testing). Defaults to stub.
	OnProgress    func(event, detail string) // Optional callback at each milestone.
	LogWriter     io.Writer                  // Receives Docker build output. nil defaults to io.Discard.
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

// Spawn creates a new universe.
func (a *Architect) Spawn(ctx context.Context, opts SpawnOpts) (*SpawnResult, error) {
	var warnings []string
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

		gateDir, err = os.MkdirTemp("", "universe-gate-")
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

	// Resolve image
	image := config.BaseImage
	if opts.Image != "" {
		image = opts.Image
	}

	// Ensure image exists (auto-build for default image, fail for custom/test images)
	if opts.Image == "" {
		if err := a.backend.EnsureImage(ctx, image, container.Dockerfile, opts.logWriter()); err != nil {
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

	// Gate bridges require network access to reach the host-side server.
	if gateSrv != nil {
		if containerCfg.NetworkMode == "none" {
			containerCfg.NetworkMode = "bridge"
		}
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

	// Build universe record
	u := config.Universe{
		ID:          id,
		Config:      opts.ConfigName,
		Agent:       opts.AgentName,
		Backend:     config.DefaultBackend,
		ContainerID: containerID,
		Workspace:   workspace,
		MindPath:    mindPath,
		GateDir:     gateDir,
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
			return nil, fmt.Errorf("universe requires element '%s' but the base image does not provide it.\nHint: Add %s to the container image, or remove it from the config's elements", e, e)
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
