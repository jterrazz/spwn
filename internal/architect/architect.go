package architect

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"time"

	"github.com/jterrazz/universe/internal/agent"
	"github.com/jterrazz/universe/internal/backend"
	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/journal"
	"github.com/jterrazz/universe/internal/physics"
	"github.com/jterrazz/universe/internal/session"
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

	// Start container to write physics.md and probe elements.
	if err := a.backend.Start(ctx, id); err != nil {
		return nil, fmt.Errorf("starting container to write physics: %w", err)
	}

	// Probe installed binaries, fall back to defaults on error.
	elements, err := physics.DetectElements(ctx, a.backend, id)
	var physicsContent string
	if err != nil {
		slog.Warn("element detection failed, using defaults", "error", err)
		physicsContent = physics.Generate(cfg)
	} else {
		physicsContent = physics.GenerateWithElements(cfg, elements)
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

	// Prepare spawn options with session resume if mind is configured.
	var opts *agent.SpawnOptions
	if cfg.Mind != "" {
		opts = &agent.SpawnOptions{}
		existing, err := session.Load(cfg.Mind, u.ID)
		if err != nil {
			slog.Warn("failed to load session", "error", err)
		}
		if existing != nil {
			opts.SessionID = existing.SessionID
		} else {
			opts.SessionID = deterministicSessionID(cfg.Mind, u.ID)
		}
	}

	start := time.Now()
	result, spawnErr := agent.Spawn(ctx, a.backend, u.ID, opts)

	// Write journal entry (best-effort).
	if cfg.Mind != "" {
		outcome := "completed"
		exitCode := 0
		if result != nil {
			exitCode = result.ExitCode
		}
		if spawnErr != nil {
			outcome = "failed"
		}
		entry := journal.Entry{
			UniverseID: u.ID,
			Image:      cfg.Image,
			Outcome:    outcome,
			ExitCode:   exitCode,
			Duration:   time.Since(start),
			Timestamp:  time.Now(),
		}
		if err := journal.Append(cfg.Mind, entry); err != nil {
			slog.Warn("failed to write journal entry", "error", err)
		}

		// Save session.
		sessionID := deterministicSessionID(cfg.Mind, u.ID)
		if opts != nil && opts.SessionID != "" {
			sessionID = opts.SessionID
		}
		s := &session.Session{
			SessionID:  sessionID,
			UniverseID: u.ID,
			MindID:     cfg.Mind,
			CreatedAt:  start,
			UpdatedAt:  time.Now(),
		}
		if err := session.Save(s); err != nil {
			slog.Warn("failed to save session", "error", err)
		}
	}

	if spawnErr != nil {
		return nil, spawnErr
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

// deterministicSessionID generates a stable session ID from mind and universe IDs.
func deterministicSessionID(mindID, universeID string) string {
	h := sha256.Sum256([]byte(mindID + ":" + universeID))
	return fmt.Sprintf("%x", h[:8])
}
