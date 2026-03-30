// Package universe provides the public API for the universe domain.
// It wraps architect, backend, manifest, state, and related operations.
package universe

import (
	"spwn.sh/core/universe/internal/architect"
	"spwn.sh/core/universe/internal/backend"
	"spwn.sh/core/universe/internal/manifest"
	"spwn.sh/core/universe/internal/models"
	"spwn.sh/core/universe/internal/observatory"
	"spwn.sh/core/universe/internal/state"
	"spwn.sh/core/universe/internal/sync"
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
type Store = state.Store

// Re-export manifest types.
type LifeManifest = manifest.LifeManifest
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

// --- Organization manifest operations ---

// LoadOrg reads and parses the organization manifest from ~/.spwn/org.yaml.
func LoadOrg() (*OrgManifest, error) { return manifest.LoadOrg() }

// LoadOrgPath reads and parses an organization manifest from the given path.
func LoadOrgPath(path string) (*OrgManifest, error) { return manifest.LoadOrgPath(path) }

// CreateOrg writes a default org.yaml for the given organization name to ~/.spwn/org.yaml.
func CreateOrg(name string) error { return manifest.CreateOrg(name) }

// --- Observatory ---

// ObservatoryServer is the Observatory HTTP API server type.
type ObservatoryServer = observatory.Server

// NewObservatoryServer returns an Observatory API server bound to addr that
// serves world and agent state from the provided Store.
func NewObservatoryServer(s *Store, addr string) *ObservatoryServer {
	return observatory.New(s, addr)
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
