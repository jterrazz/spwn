package architect

import (
	"context"
	"fmt"

	"spwn.sh/packages/container/backend"
	"spwn.sh/packages/world/models"
	"spwn.sh/packages/world/runtimestate"

	_ "spwn.sh/packages/runtimes/defaults"
)

// Architect orchestrates world lifecycle. The runtime adapter is
// resolved per-world at operation time (see resolveSpawner) rather
// than held as state, so a single Architect drives claude-code and
// codex worlds side-by-side without re-instantiation.
type Architect struct {
	backend backend.Backend
	rstate  *runtimestate.Store
}

// New creates an Architect with the given backend and runtimestate
// store. Runtime adapters are looked up per-world at call time, so
// the constructor takes no runtime dependency: the blank-import of
// runtimes/defaults at the top of this file is the only wiring needed
// to make every built-in adapter discoverable.
func New(b backend.Backend, s *runtimestate.Store) *Architect {
	return &Architect{backend: b, rstate: s}
}

// SetSessionID stores a runtime session ID for an agent in a world.
func (a *Architect) SetSessionID(worldID, agentName, sessionID string) error {
	return a.rstate.SetSessionID(worldID, agentName, sessionID)
}

// GetSessionID returns the runtime session ID for an agent in a world.
func (a *Architect) GetSessionID(worldID, agentName string) string {
	return a.rstate.GetSessionID(worldID, agentName)
}

// NewFromEnv creates an Architect using the default Docker backend and runtimestate store.
func NewFromEnv() (*Architect, error) {
	docker, err := backend.NewDocker()
	if err != nil {
		return nil, fmt.Errorf("cannot connect to Docker: %w", err)
	}

	store, err := runtimestate.NewStore()
	if err != nil {
		return nil, fmt.Errorf("cannot initialize runtimestate store: %w", err)
	}

	return New(docker, store), nil
}

// List returns all worlds.
func (a *Architect) List(ctx context.Context) ([]models.World, error) {
	return a.rstate.List()
}

// Inspect returns a world by ID.
func (a *Architect) Inspect(ctx context.Context, worldID string) (*models.World, error) {
	return a.rstate.Get(worldID)
}

// Rename updates a world's display name. Persists via runtimestate
// so the label-derived name on the container stays unchanged while
// every subsequent List/Get surfaces the new value via hydrate(). An
// empty name clears the override and restores the label default.
func (a *Architect) Rename(ctx context.Context, worldID, name string) error {
	return a.rstate.SetDisplayName(worldID, name)
}

// Attach opens an interactive shell into a running world.
func (a *Architect) Attach(ctx context.Context, worldID string) error {
	u, err := a.rstate.Get(worldID)
	if err != nil {
		return err
	}

	running, err := a.backend.IsRunning(ctx, u.ContainerID)
	if err != nil {
		return fmt.Errorf("check container: %w", err)
	}
	if !running {
		return fmt.Errorf("world %s is not running.\nStart a world first with 'spwn world'", worldID)
	}

	_, err = a.backend.Exec(ctx, u.ContainerID, backend.ExecConfig{
		Cmd: []string{"bash"},
		TTY: true,
	})
	return err
}
