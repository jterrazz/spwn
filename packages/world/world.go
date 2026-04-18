// Package world provides the public API for the world domain.
// It wraps backend, manifest, runtimestate, and related operations.
package world

import (
	"spwn.sh/packages/agent"

	"spwn.sh/packages/container/backend"
	"spwn.sh/packages/world/manifest"
	"spwn.sh/packages/world/models"
	"spwn.sh/packages/world/runtimestate"
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

type AgentRecord = models.AgentRecord

// Re-export backend types.
type Backend = backend.Backend
type ImageInfo = backend.ImageInfo
type ContainerInfo = backend.ContainerInfo
// Store is the world-facing store: enumerates worlds from Docker
// labels and persists per-world mutable data (session ids, deployed
// agents, editable display name) under ~/.spwn/world-states/.
type Store = runtimestate.Store

// --- Backend constructors ---

// NewDocker returns a Docker-based container backend initialised from the
// host's Docker daemon environment (DOCKER_HOST, etc.).
func NewDocker() (*backend.Docker, error) {
	return backend.NewDocker()
}

// --- Store constructors ---

// NewStore returns a production Store wired to the host Docker
// daemon and the user's world-state directory.
func NewStore() (*Store, error) {
	return runtimestate.NewStore()
}

// NewStoreAt returns a Store rooted at dir with no Docker backend.
// Suitable for tests that only exercise the mutable-state methods
// (SetSessionID, AddAgent, SetDisplayName, …). List/Get error until
// a backend is wired in via NewStoreWith.
func NewStoreAt(dir string) (*Store, error) {
	return runtimestate.NewStoreAt(dir)
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

