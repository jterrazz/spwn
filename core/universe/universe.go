// Package universe provides the public API for the universe domain.
// It wraps architect, backend, manifest, state, and related operations.
package universe

import (
	"context"
	"fmt"
	"sort"
	"time"

	"spwn.sh/core/foundation"
	"spwn.sh/core/universe/internal/architect"
	"spwn.sh/core/universe/internal/backend"
	"spwn.sh/core/universe/internal/claw"
	"spwn.sh/core/universe/internal/manifest"
	"spwn.sh/core/universe/internal/models"
	"spwn.sh/core/universe/internal/observatory"
	"spwn.sh/core/universe/internal/runtime"
	"spwn.sh/core/universe/internal/state"
	"spwn.sh/core/universe/internal/sync"

	containerTypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// Re-export model types so consumers don't need to reach into internal packages.
type World = models.World
type Manifest = models.Manifest
type PhysicsManifest = models.PhysicsManifest
type ConstantsManifest = models.ConstantsManifest
type LawsManifest = models.LawsManifest
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

// Re-export manifest types.
type LifeManifest = manifest.LifeManifest
type ProfileManifest = manifest.ProfileManifest
type OrgManifest = manifest.OrgManifest

// Re-export state types.
type ClawState = state.ClawState

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
// defaults (CPU, memory, timeout, base elements).
func ApplyDefaults(m *Manifest) {
	manifest.ApplyDefaults(m)
}

// ExpandElements resolves @pack references into individual binary names and
// deduplicates the resulting list.
func ExpandElements(elems []string) []string {
	return manifest.ExpandElements(elems)
}

// --- Universe manifest operations ---

// LoadOrg reads and parses the universe manifest from ~/.spwn/org.yaml.
func LoadOrg() (*OrgManifest, error) { return manifest.LoadOrg() }

// LoadOrgPath reads and parses a universe manifest from the given path.
func LoadOrgPath(path string) (*OrgManifest, error) { return manifest.LoadOrgPath(path) }

// CreateOrg writes a default universe manifest to ~/.spwn/org.yaml.
func CreateOrg(name string) error { return manifest.CreateOrg(name) }

// --- Observatory ---

// ObservatoryServer is the Observatory HTTP API server type.
type ObservatoryServer = observatory.Server

// NewObservatoryServer returns an Observatory API server bound to addr that
// serves world and agent state from the provided Store. arch may be nil for
// read-only mode (no world spawn/destroy).
func NewObservatoryServer(s *Store, arch *Architect, addr string) *ObservatoryServer {
	return observatory.New(s, arch, addr)
}

// --- Git sync operations ---

// SyncToGit commits and pushes pending ~/.spwn/ changes to the given git
// repository and branch.
func SyncToGit(repo, branch string) error { return sync.SyncToGit(repo, branch) }

// PullFromGit fetches and applies the latest changes from the given git
// repository and branch into ~/.spwn/.
func PullFromGit(repo, branch string) error { return sync.PullFromGit(repo, branch) }

// --- Claw state operations ---

// LoadClawState reads the Claw daemon state from disk (~/.spwn/claw.json).
func LoadClawState() (*ClawState, error) { return state.LoadClawState() }

// SaveClawState persists the Claw daemon state to disk (~/.spwn/claw.json).
func SaveClawState(s *ClawState) error { return state.SaveClawState(s) }

// --- Runtime / Claw discovery ---

// GenerateRuntimeDockerfile creates a Dockerfile for the given runtime.
func GenerateRuntimeDockerfile(runtimeName string) (string, error) {
	rt, err := runtime.Get(runtimeName)
	if err != nil {
		return "", err
	}
	return runtime.GenerateDockerfile(rt), nil
}

// ListRuntimes returns all registered runtime names.
func ListRuntimes() []string {
	all := runtime.All()
	names := make([]string, 0, len(all))
	for name := range all {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ListClaws returns all registered claw names.
func ListClaws() []string {
	all := claw.All()
	names := make([]string, 0, len(all))
	for name := range all {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RuntimeAvailable returns true if the named runtime is production-ready.
func RuntimeAvailable(name string) bool {
	r, err := runtime.Get(name)
	if err != nil {
		return false
	}
	return r.Available()
}

// ClawAvailable returns true if the named claw adapter is production-ready.
func ClawAvailable(name string) bool {
	c, err := claw.Get(name)
	if err != nil {
		return false
	}
	return c.Available()
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

// StartArchitectDaemon creates and starts the spwn-architect Docker container.
// It returns the container ID. If the container is already running, it returns
// an error indicating that.
func StartArchitectDaemon(ctx context.Context, imageOverride string) (string, error) {
	docker, err := backend.NewDocker()
	if err != nil {
		return "", fmt.Errorf("docker is not reachable: %w", err)
	}

	// Check if already running
	info, err := docker.Inspect(ctx, foundation.ArchitectContainerName)
	if err == nil && info.Running {
		return info.ID, fmt.Errorf("architect is already running (container %s)", foundation.ArchitectContainerName)
	}

	// If container exists but stopped, remove it first
	if err == nil && !info.Running {
		_ = docker.Remove(ctx, foundation.ArchitectContainerName)
	}

	// Resolve image
	image := foundation.ArchitectImage
	if imageOverride != "" {
		image = imageOverride
	}

	// Check image exists
	exists, err := docker.ImageExists(ctx, image)
	if err != nil {
		return "", fmt.Errorf("checking image: %w", err)
	}
	if !exists {
		return "", fmt.Errorf("image %s not found. Build it with: make build-architect-image", image)
	}

	// Create container
	containerCfg := &containerTypes.Config{
		Image: image,
		Env: []string{
			"SPWN_HOME=/root/.spwn",
		},
	}
	hostCfg := &containerTypes.HostConfig{
		Binds: []string{
			"/var/run/docker.sock:/var/run/docker.sock",
			foundation.BaseDir() + ":/root/.spwn",
		},
		RestartPolicy: containerTypes.RestartPolicy{Name: "unless-stopped"},
	}

	id, err := docker.CreateNamedContainer(ctx, foundation.ArchitectContainerName, containerCfg, hostCfg)
	if err != nil {
		return "", fmt.Errorf("creating architect container: %w", err)
	}

	if err := docker.Start(ctx, id); err != nil {
		_ = docker.Remove(ctx, id)
		return "", fmt.Errorf("starting architect container: %w", err)
	}

	// Save claw state
	clawState := &state.ClawState{
		Active:    true,
		StartedAt: time.Now(),
	}
	_ = state.SaveClawState(clawState)

	return id, nil
}

// StopArchitectDaemon stops and removes the spwn-architect container.
func StopArchitectDaemon(ctx context.Context) error {
	docker, err := backend.NewDocker()
	if err != nil {
		return fmt.Errorf("docker is not reachable: %w", err)
	}

	info, err := docker.Inspect(ctx, foundation.ArchitectContainerName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return fmt.Errorf("architect is not running")
		}
		return fmt.Errorf("inspecting architect container: %w", err)
	}

	if info.Running {
		if err := docker.Stop(ctx, foundation.ArchitectContainerName); err != nil {
			return fmt.Errorf("stopping architect container: %w", err)
		}
	}

	if err := docker.Remove(ctx, foundation.ArchitectContainerName); err != nil {
		return fmt.Errorf("removing architect container: %w", err)
	}

	// Update claw state
	clawState := &state.ClawState{
		Active: false,
	}
	_ = state.SaveClawState(clawState)

	return nil
}

// GetArchitectDaemonStatus queries Docker for the architect container status.
func GetArchitectDaemonStatus(ctx context.Context) (*ArchitectDaemonInfo, error) {
	docker, err := backend.NewDocker()
	if err != nil {
		return nil, fmt.Errorf("docker is not reachable: %w", err)
	}

	info, err := docker.Inspect(ctx, foundation.ArchitectContainerName)
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

	// Load org info if available
	org, err := manifest.LoadOrg()
	if err == nil && org != nil {
		result.OrgName = org.Name
	}

	return result, nil
}
