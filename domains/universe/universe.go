// Package universe provides the public API for the universe domain.
// It wraps architect, backend, manifest, state, and related operations.
package universe

import (
	"github.com/jterrazz/spwn/domains/universe/internal/architect"
	"github.com/jterrazz/spwn/domains/universe/internal/backend"
	"github.com/jterrazz/spwn/domains/universe/internal/manifest"
	"github.com/jterrazz/spwn/domains/universe/internal/models"
	"github.com/jterrazz/spwn/domains/universe/internal/state"
)

// Re-export model types so consumers don't need to reach into internal packages.
type Universe = models.Universe
type UniverseManifest = models.UniverseManifest
type PhysicsManifest = models.PhysicsManifest
type ConstantsManifest = models.ConstantsManifest
type LawsManifest = models.LawsManifest
type UniverseStatus = models.UniverseStatus

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

// Re-export backend types.
type Backend = backend.Backend
type Store = state.Store

// Re-export manifest types.
type LifeManifest = manifest.LifeManifest

// --- Architect constructors ---

// NewArchitect creates an Architect with the given backend and state store.
func NewArchitect(b Backend, s *Store) *Architect {
	return architect.New(b, s)
}

// NewArchitectFromEnv creates an Architect using the default Docker backend and state store.
func NewArchitectFromEnv() (*Architect, error) {
	return architect.NewFromEnv()
}

// --- Backend constructors ---

// NewDocker creates a Docker backend from the environment.
func NewDocker() (*backend.Docker, error) {
	return backend.NewDocker()
}

// --- State constructors ---

// NewStore creates a state Store at ~/.spwn/state.json.
func NewStore() (*Store, error) {
	return state.NewStore()
}

// NewStoreAt creates a state Store at an explicit path.
func NewStoreAt(path string) (*Store, error) {
	return state.NewStoreAt(path)
}

// --- Manifest operations ---

// LoadManifest reads a named universe config from ~/.spwn/universes/{name}.yaml.
func LoadManifest(name string) (UniverseManifest, error) {
	return manifest.Load(name)
}

// LoadManifestPath reads a universe config from an explicit file path.
func LoadManifestPath(path string) (UniverseManifest, error) {
	return manifest.LoadPath(path)
}

// ListConfigs returns the names of all universe configs.
func ListConfigs() ([]string, error) {
	return manifest.ListConfigs()
}

// CreateDefaultConfig creates a default.yaml in ~/.spwn/universes/.
func CreateDefaultConfig() error {
	return manifest.CreateDefault()
}

// CreateConfig scaffolds a new named config.
func CreateConfig(name string) error {
	return manifest.CreateConfig(name)
}

// ValidateManifest checks that a manifest is well-formed.
func ValidateManifest(m UniverseManifest) error {
	return manifest.Validate(m)
}

// ApplyDefaults fills zero-value fields with built-in defaults.
func ApplyDefaults(m *UniverseManifest) {
	manifest.ApplyDefaults(m)
}

// ExpandElements expands @packs into individual binaries and deduplicates.
func ExpandElements(elems []string) []string {
	return manifest.ExpandElements(elems)
}
