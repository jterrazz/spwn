package architect

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jterrazz/universe/internal/agent"
	"github.com/jterrazz/universe/internal/backend"
	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/physics"
)

// Architect orchestrates universe lifecycle.
type Architect struct {
	backend backend.Backend
}

// NewWithBackend creates a new Architect with the given backend.
func NewWithBackend(b backend.Backend) *Architect {
	return &Architect{backend: b}
}

// New creates a new Architect with a Docker backend.
func New() (*Architect, error) {
	b, err := backend.NewDockerBackend()
	if err != nil {
		return nil, err
	}
	return NewWithBackend(b), nil
}

// Create provisions a new universe without starting it.
func (a *Architect) Create(ctx context.Context, cfg *config.UniverseConfig) (*config.Universe, error) {
	slog.Info("creating universe", "image", cfg.Image)

	id, err := a.backend.Create(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// Write physics.md into the container
	physicsContent := physics.Generate(cfg)
	if err := a.backend.Start(ctx, id); err != nil {
		return nil, fmt.Errorf("starting container to write physics: %w", err)
	}

	writeCmd := []string{"sh", "-c", fmt.Sprintf("mkdir -p /universe && cat > /universe/physics.md << 'PHYSICS_EOF'\n%sPHYSICS_EOF", physicsContent)}
	if _, err := a.backend.Exec(ctx, id, writeCmd); err != nil {
		return nil, fmt.Errorf("writing physics.md: %w", err)
	}

	if err := a.backend.Stop(ctx, id); err != nil {
		return nil, fmt.Errorf("stopping container after physics write: %w", err)
	}

	return &config.Universe{
		ID:        id,
		Status:    config.StatusCreated,
		Image:     cfg.Image,
		Mind:      cfg.Mind,
		CreatedAt: time.Now(),
	}, nil
}

// Spawn creates, starts, and spawns an agent in a universe.
func (a *Architect) Spawn(ctx context.Context, cfg *config.UniverseConfig) (*config.Universe, error) {
	u, err := a.Create(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err := a.backend.Start(ctx, u.ID); err != nil {
		return nil, fmt.Errorf("starting universe: %w", err)
	}
	u.Status = config.StatusRunning

	if err := agent.Spawn(ctx, a.backend, u.ID); err != nil {
		return nil, err
	}

	return u, nil
}

// List returns all universes.
func (a *Architect) List(ctx context.Context) ([]config.Universe, error) {
	containers, err := a.backend.List(ctx)
	if err != nil {
		return nil, err
	}

	universes := make([]config.Universe, 0, len(containers))
	for _, c := range containers {
		status := config.StatusStopped
		switch c.Status {
		case "running":
			status = config.StatusRunning
		case "created":
			status = config.StatusCreated
		}
		universes = append(universes, config.Universe{
			ID:        c.ID,
			Image:     c.Image,
			Mind:      c.Mind,
			Status:    status,
			CreatedAt: time.Now(),
		})
	}

	return universes, nil
}

// Inspect returns details about a specific universe.
func (a *Architect) Inspect(ctx context.Context, id string) (*config.Universe, error) {
	info, err := a.backend.Inspect(ctx, id)
	if err != nil {
		return nil, err
	}

	status := config.StatusStopped
	switch info.Status {
	case "running":
		status = config.StatusRunning
	case "created":
		status = config.StatusCreated
	}

	return &config.Universe{
		ID:        info.ID,
		Image:     info.Image,
		Mind:      info.Mind,
		Status:    status,
		CreatedAt: time.Now(),
	}, nil
}

// Destroy stops and removes a universe.
func (a *Architect) Destroy(ctx context.Context, id string) error {
	slog.Info("destroying universe", "id", id)
	return a.backend.Remove(ctx, id)
}
