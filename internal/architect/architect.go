package architect

import (
	"context"
	"fmt"

	"github.com/jterrazz/universe/internal/backend"
	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/gate"
	"github.com/jterrazz/universe/internal/state"
)

// Architect orchestrates universe lifecycle.
type Architect struct {
	backend backend.Backend
	state   *state.Store
	gates   map[string]*gate.Server // universeID → running gate server
}

// New creates an Architect with the given backend and state store.
func New(b backend.Backend, s *state.Store) *Architect {
	return &Architect{backend: b, state: s, gates: make(map[string]*gate.Server)}
}

// NewFromEnv creates an Architect using the default Docker backend and state store.
func NewFromEnv() (*Architect, error) {
	docker, err := backend.NewDocker()
	if err != nil {
		return nil, fmt.Errorf("cannot connect to Docker: %w", err)
	}

	store, err := state.NewStore()
	if err != nil {
		return nil, fmt.Errorf("cannot initialize state store: %w", err)
	}

	return New(docker, store), nil
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
