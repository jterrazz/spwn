// Package world provides the public API for the world domain.
// It wraps architect, backend, manifest, state, and related operations.
package world

import (
	"spwn.sh/packages/agent"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"spwn.sh/packages/platform"
	"spwn.sh/packages/activity"
	"spwn.sh/packages/auth"
	"spwn.sh/packages/world/architect"
	"spwn.sh/packages/world/internal/backend"
	"spwn.sh/packages/world/internal/labels"
	"spwn.sh/packages/world/manifest"
	"spwn.sh/packages/world/models"
	"spwn.sh/packages/world/runtime"
	"spwn.sh/packages/world/state"

	containerTypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// Re-export model types so consumers don't need to reach into internal packages.
type World = models.World
type Workspace = models.Workspace
type Manifest = models.Manifest
type Status = models.Status

// Re-export status constants.
const (
	StatusCreating  = models.StatusCreating
	StatusRunning   = models.StatusRunning
	StatusIdle      = models.StatusIdle
	StatusStopped   = models.StatusStopped
	StatusDestroyed = models.StatusDestroyed
)

// Re-export architect types.
type Architect = architect.Architect
type SpawnResult = architect.SpawnResult
type SpawnOpts = architect.SpawnOpts
type AgentSpec = architect.AgentSpec
type AgentRecord = models.AgentRecord

// Re-export backend types.
type Backend = backend.Backend
type ImageInfo = backend.ImageInfo
type ContainerInfo = backend.ContainerInfo
type Store = state.Store


// --- Architect constructors ---

// NewArchitect returns an Architect that provisions and destroys worlds using
// the given container backend and persists state through the provided store.
func NewArchitect(b Backend, s *Store) *Architect {
	return architect.New(b, s)
}

// NewArchitectFromEnv returns an Architect configured from the host environment,
// using the default Docker backend and the standard state store at ~/.spwn/state.json.
func NewArchitectFromEnv() (*Architect, error) {
	return architect.NewFromEnv()
}

// --- Backend constructors ---

// NewDocker returns a Docker-based container backend initialised from the
// host's Docker daemon environment (DOCKER_HOST, etc.).
func NewDocker() (*backend.Docker, error) {
	return backend.NewDocker()
}

// --- State constructors ---

// NewStore returns a Store backed by ~/.spwn/state.json, creating the file
// if it does not exist.
func NewStore() (*Store, error) {
	return state.NewStore()
}

// NewStoreAt returns a Store backed by the file at the given path, creating it
// if it does not exist.
func NewStoreAt(path string) (*Store, error) {
	return state.NewStoreAt(path)
}

// --- Manifest operations ---

// LoadManifest reads and parses the world config named {name} from
// ~/.spwn/worlds/{name}.yaml.
func LoadManifest(name string) (Manifest, error) {
	return manifest.Load(name)
}

// LoadManifestPath reads a world config from an explicit file path.
func LoadManifestPath(path string) (Manifest, error) {
	return manifest.LoadPath(path)
}

// ListConfigs returns the names of all world configs found in ~/.spwn/worlds/.
func ListConfigs() ([]string, error) {
	return manifest.ListConfigs()
}

// CreateDefaultConfig writes a default.yaml world config to ~/.spwn/worlds/.
func CreateDefaultConfig() error {
	return manifest.CreateDefault()
}

// CreateConfig scaffolds a new named world config file in ~/.spwn/worlds/.
func CreateConfig(name string) error {
	return manifest.CreateConfig(name)
}

// ValidateManifest checks that a Manifest is well-formed, returning an error
// describing the first problem found.
func ValidateManifest(m Manifest) error {
	return manifest.Validate(m)
}

// ApplyDefaults fills zero-value fields in the given Manifest with built-in
// defaults (CPU, memory, timeout, base tools).
func ApplyDefaults(m *Manifest) {
	manifest.ApplyDefaults(m)
}

// LoadAgentManifest reads an agent.yaml file from the given agent
// directory. Returns (nil, nil) when agent.yaml is absent — callers
// treat the manifest as optional. External callers (notably the CLI
// project resolver) use this to compute the tool union across a
// world's referenced agents without reaching into internal packages.
func LoadAgentManifest(agentDir string) (*agent.Manifest, error) {
	return agent.LoadManifestPath(agentDir)
}

// --- Runtime ---

// RuntimeSpawnConfig holds the parameters for building a runtime command.
type RuntimeSpawnConfig = runtime.SpawnConfig

// Runtime is the public alias for the internal runtime adapter
// interface. Exposed so CLI callers (talk.go, the interactive
// session path) can reach SyncHostCredentials / PrelaunchShell
// without importing the internal package.
type Runtime = runtime.Runtime

// GetRuntime looks up a registered runtime adapter by backend name
// ("claude-code", "codex", …) and returns the canonical interface.
// Callers use the returned Runtime for credential sync, prelaunch
// shell, and command building.
func GetRuntime(name string) (Runtime, error) {
	return runtime.Get(name)
}

// BuildRuntimeCommand returns the CLI command for a given runtime and config.
// This is the single source of truth for how to invoke any runtime inside a container.
func BuildRuntimeCommand(runtimeName string, cfg RuntimeSpawnConfig) ([]string, error) {
	rt, err := runtime.Get(runtimeName)
	if err != nil {
		return nil, err
	}
	return rt.BuildCommand(cfg), nil
}

// --- Architect Daemon operations ---

// ArchitectDaemonInfo describes the state of the Architect daemon container.
type ArchitectDaemonInfo struct {
	ContainerID string
	Image       string
	Status      string
	Running     bool
	StartedAt   time.Time
	Uptime      time.Duration
	OrgName     string
}

// StartArchitectDaemonOpts configures architect daemon spawn. All
// fields are optional. The OnProgress callback is the canonical place
// to surface real-time spawn diagnostics - both the CLI stepper and
// the web UI's status endpoint feed off it.
type StartArchitectDaemonOpts struct {
	// ImageOverride lets the caller pin a specific architect image
	// (used by tests and SPWN_ARCHITECT_IMAGE). When empty, the
	// canonical image is built/refreshed by the image package.
	ImageOverride string
	// LogWriter receives raw output from the image build (npm install,
	// docker build steps, …). nil → io.Discard.
	LogWriter io.Writer
	// OnProgress is called at each step of the spawn pipeline with
	// (event, detail) pairs. Events are stable strings that the
	// the API polls; detail is a free-form human-readable note.
	OnProgress func(event, detail string)
}

func (o *StartArchitectDaemonOpts) progress(event, detail string) {
	if o != nil && o.OnProgress != nil {
		o.OnProgress(event, detail)
	}
}

func (o *StartArchitectDaemonOpts) writer() io.Writer {
	if o != nil && o.LogWriter != nil {
		return o.LogWriter
	}
	return io.Discard
}

// StartArchitectDaemon creates and starts the spwn-architect Docker
// container. It returns the container ID. If the container is already
// running, it returns an error indicating that.
//
// This is the back-compat shim. New code should call
// StartArchitectDaemonWithOpts so it can subscribe to progress events.
func StartArchitectDaemon(ctx context.Context, imageOverride string, logWriters ...io.Writer) (string, error) {
	var lw io.Writer
	if len(logWriters) > 0 {
		lw = logWriters[0]
	}
	return StartArchitectDaemonWithOpts(ctx, StartArchitectDaemonOpts{
		ImageOverride: imageOverride,
		LogWriter:     lw,
	})
}

// StartArchitectDaemonWithOpts is the rich entry point. It emits an
// OnProgress event at every step so callers can render real-time spawn
// diagnostics instead of guessing from elapsed time.
//
// Event vocabulary (in order):
//
//	docker_check         - opening the Docker client
//	already_running      - fast path, returned with error
//	cleanup              - removing a stopped container
//	image_resolve        - resolving image tag
//	image_building       - building the architect image (long step)
//	image_ready          - image present
//	credentials_sync     - credentials being written
//	host_files           - stack.md / knowledge dir bootstrapped
//	container_creating   - Docker create call
//	container_starting   - Docker start call
//	ready                - daemon up and labelled
func StartArchitectDaemonWithOpts(ctx context.Context, opts StartArchitectDaemonOpts) (string, error) {
	opts.progress("docker_check", "opening Docker client")
	docker, err := backend.NewDocker()
	if err != nil {
		return "", fmt.Errorf("docker is not reachable: %w", err)
	}

	// Check if already running
	info, err := docker.Inspect(ctx, platform.ArchitectContainerName())
	if err == nil && info.Running {
		opts.progress("already_running", info.ID)
		return info.ID, fmt.Errorf("architect is already running (container %s)", platform.ArchitectContainerName())
	}

	// If container exists but stopped, remove it first
	if err == nil && !info.Running {
		opts.progress("cleanup", "removing stopped architect container")
		_ = docker.Remove(ctx, platform.ArchitectContainerName())
	}

	// Resolve image
	image := platform.ArchitectImage
	if opts.ImageOverride != "" {
		image = opts.ImageOverride
	}
	opts.progress("image_resolve", image)

	// Ensure architect image exists and is up to date (auto-build if needed)
	if opts.ImageOverride == "" {
		opts.progress("image_building", "building "+image+" - first run takes minutes")
		if err := architect.BuildArchitectImage(ctx, docker, opts.writer()); err != nil {
			return "", fmt.Errorf("ensure architect image: %w", err)
		}
	} else {
		exists, err := docker.ImageExists(ctx, image)
		if err != nil {
			return "", fmt.Errorf("checking image: %w", err)
		}
		if !exists {
			return "", fmt.Errorf("image %s not found", image)
		}
	}
	opts.progress("image_ready", image)

	// Sync credentials to bind-mountable directory
	opts.progress("credentials_sync", "syncing host credentials")
	_ = auth.SyncCredentials()

	envVars := []string{
		"SPWN_HOME=/home/spwn/.spwn",
	}

	// Create container - entrypoint is "sleep infinity" (set in Dockerfile),
	// we docker exec claude into it when we want to talk. The architect
	// gets the spwn kind label too, so it shows up in tooling that
	// queries `docker ps --filter label=sh.spwn.kind=architect`.
	architectLabels := map[string]string{labels.KindKey: labels.KindArchitect}
	labels.ApplyTestRun(architectLabels)
	containerCfg := &containerTypes.Config{
		Image:  image,
		Env:    envVars,
		Labels: architectLabels,
	}
	// Ensure architect stack file exists on the host
	architectStackPath := platform.BaseDir() + "/architect/stack.md"
	if _, err := os.Stat(architectStackPath); os.IsNotExist(err) {
		_ = os.MkdirAll(platform.BaseDir()+"/architect", 0755)
		_ = os.WriteFile(architectStackPath, []byte("# Architect Stack\n\n## Focus\n\n## Queued\n- [ ] Review agent health and journal entries\n- [ ] Consolidate old agent memories\n\n## Done\n"), 0644)
	}

	opts.progress("host_files", "stack ready")

	hostCfg := &containerTypes.HostConfig{
		Binds: []string{
			platform.BaseDir() + ":/home/spwn/.spwn",
			architectStackPath + ":/me/stack.md",
			"/var/run/docker.sock:/var/run/docker.sock",
			platform.CredentialsDir() + ":/credentials:ro",
		},
		RestartPolicy: containerTypes.RestartPolicy{Name: "unless-stopped"},
	}

	opts.progress("container_creating", platform.ArchitectContainerName())
	id, err := docker.CreateNamedContainer(ctx, platform.ArchitectContainerName(), containerCfg, hostCfg)
	if err != nil {
		return "", fmt.Errorf("creating architect container: %w", err)
	}

	opts.progress("container_starting", id[:12])
	if err := docker.Start(ctx, id); err != nil {
		_ = docker.Remove(ctx, id)
		return "", fmt.Errorf("starting architect container: %w", err)
	}

	activity.Log(activity.Event{
		Type:   activity.TypeArchitectStarted,
		Actor:  "architect",
		Verb:   "started",
		Target: "architect",
		Phrase: activity.PhraseArchitectStarted(),
	})

	opts.progress("ready", id[:12])
	return id, nil
}

// StopArchitectDaemon stops and removes the spwn-architect container.
func StopArchitectDaemon(ctx context.Context) error {
	docker, err := backend.NewDocker()
	if err != nil {
		return fmt.Errorf("docker is not reachable: %w", err)
	}

	info, err := docker.Inspect(ctx, platform.ArchitectContainerName())
	if err != nil {
		if client.IsErrNotFound(err) {
			return fmt.Errorf("architect is not running")
		}
		return fmt.Errorf("inspecting architect container: %w", err)
	}

	if info.Running {
		if err := docker.Stop(ctx, platform.ArchitectContainerName()); err != nil {
			return fmt.Errorf("stopping architect container: %w", err)
		}
	}

	if err := docker.Remove(ctx, platform.ArchitectContainerName()); err != nil {
		return fmt.Errorf("removing architect container: %w", err)
	}

	activity.Log(activity.Event{
		Type:   activity.TypeArchitectStopped,
		Actor:  "architect",
		Verb:   "stopped",
		Target: "architect",
		Phrase: activity.PhraseArchitectStopped(),
	})

	return nil
}

// GetArchitectDaemonStatus queries Docker for the architect container status.
func GetArchitectDaemonStatus(ctx context.Context) (*ArchitectDaemonInfo, error) {
	docker, err := backend.NewDocker()
	if err != nil {
		return nil, fmt.Errorf("docker is not reachable: %w", err)
	}

	info, err := docker.Inspect(ctx, platform.ArchitectContainerName())
	if err != nil {
		if client.IsErrNotFound(err) {
			return &ArchitectDaemonInfo{Running: false, Status: "not running"}, nil
		}
		return nil, fmt.Errorf("inspecting architect container: %w", err)
	}

	result := &ArchitectDaemonInfo{
		ContainerID: info.ID[:12],
		Image:       info.Image,
		Status:      info.Status,
		Running:     info.Running,
		StartedAt:   info.StartedAt,
	}

	if info.Running {
		result.Uptime = time.Since(info.StartedAt)
	}

	return result, nil
}

// TalkToArchitectExecArgs returns the docker exec arguments needed to talk to
// the Architect. The caller is responsible for executing the command (so it can
// handle interactive vs one-shot modes and streaming).
//
// If message is non-empty a one-shot --print invocation is returned; otherwise
// an interactive Claude session is returned.
func TalkToArchitectExecArgs(message string) ([]string, error) {
	// Build docker exec args
	args := []string{"exec"}

	if message == "" {
		// interactive
		args = append(args, "-it")
	}

	// Run as 'architect' user (Claude Code refuses --dangerously-skip-permissions as root)
	args = append(args, "-u", "architect", "-w", "/me")
	// Pass SPWN_HOME so spwn CLI works inside the exec
	args = append(args, "-e", "SPWN_HOME=/home/spwn/.spwn")

	// Sync credentials before exec
	_ = auth.SyncCredentials()

	args = append(args, platform.ArchitectContainerName())

	// Claude Code invocation - wrapped to source credentials from bind mount
	claudeArgs := []string{"claude", "--dangerously-skip-permissions"}
	if message != "" {
		claudeArgs = append(claudeArgs, "-p", message, "--print",
			"--append-system-prompt",
		"You are the Architect. Read /me/ARCHITECT.md for your identity. "+
				"IMPORTANT: When a user asks you to do something, you MUST include a [STACK_PUSH] marker in your response. "+
				"Format: [STACK_PUSH] Short task title\\nPriority: blocking|queued\\nBrief description. "+
				"Also update /me/stack.md with the new task. "+
				"When completing a task use [STACK_POP] Short task title. "+
				"Read /me/skills/ for detailed guides.")
	}

	// Wrap command to source credentials from bind-mounted /credentials/.env
	escaped := make([]string, len(claudeArgs))
	for i, arg := range claudeArgs {
		escaped[i] = "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
	}
	setup := "source /credentials/.env 2>/dev/null"
	setup += "; [ -f /credentials/openai/auth.json ] && mkdir -p $HOME/.codex && ln -sf /credentials/openai/auth.json $HOME/.codex/auth.json 2>/dev/null"
	shellCmd := setup + "; exec " + strings.Join(escaped, " ")
	args = append(args, "bash", "-c", shellCmd)
	return args, nil
}
