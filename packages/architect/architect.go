package architect

import (
	"context"
	"fmt"
	"time"

	"spwn.sh/packages/activity"
	"spwn.sh/packages/compile/backend"
	"spwn.sh/packages/world/models"
	"spwn.sh/packages/world/runtime"
	"spwn.sh/packages/world/state"

	// Register the claude-code runtime adapter
	_ "spwn.sh/packages/runtimes/claude_code"
)

// Architect orchestrates world lifecycle.
type Architect struct {
	backend backend.Backend
	state   *state.Store
	runtime runtime.Runtime // injected runtime adapter - claude-code
}

// New creates an Architect with the given backend and state store.
func New(b backend.Backend, s *state.Store) *Architect {
	rt, _ := runtime.Get("claude-code")
	return &Architect{
		backend: b,
		state:   s,
		runtime: rt,
	}
}

// SetSessionID stores a runtime session ID for an agent in a world.
func (a *Architect) SetSessionID(worldID, agentName, sessionID string) error {
	return a.state.SetSessionID(worldID, agentName, sessionID)
}

// GetSessionID returns the runtime session ID for an agent in a world.
func (a *Architect) GetSessionID(worldID, agentName string) string {
	return a.state.GetSessionID(worldID, agentName)
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

// List returns all worlds.
func (a *Architect) List(ctx context.Context) ([]models.World, error) {
	return a.state.List()
}

// Inspect returns a world by ID.
func (a *Architect) Inspect(ctx context.Context, worldID string) (*models.World, error) {
	return a.state.Get(worldID)
}

// Rename updates a world's display name. Empty name clears the field (UIs fall back to the ID).
func (a *Architect) Rename(ctx context.Context, worldID, name string) error {
	if _, err := a.state.Get(worldID); err != nil {
		return err
	}
	return a.state.Rename(worldID, name)
}

// Snapshot commits the current state of a world's container as a Docker compile.
func (a *Architect) Snapshot(ctx context.Context, worldID, name string) (string, error) {
	u, err := a.state.Get(worldID)
	if err != nil {
		return "", fmt.Errorf("world %s not found.\nRun 'spwn list' to see active worlds", worldID)
	}

	tag := fmt.Sprintf("spwn-snapshot:%s--%s", worldID, name)
	if name == "" {
		tag = fmt.Sprintf("spwn-snapshot:%s--%s", worldID, time.Now().Format("2006-01-02T15-04"))
	}

	if err := a.backend.Commit(ctx, u.ContainerID, tag); err != nil {
		return "", fmt.Errorf("snapshot failed: %w", err)
	}

	activity.Log(activity.Event{
		Type:    activity.TypeWorldSnapshot,
		Actor:   "user",
		Verb:    "snapshotted",
		Target:  worldID,
		Phrase:  activity.PhraseWorldSnapshot(worldID, tag),
		WorldID: worldID,
	})

	return tag, nil
}

// ListSnapshots returns all snapshot images.
func (a *Architect) ListSnapshots(ctx context.Context) ([]backend.ImageInfo, error) {
	return a.backend.ImageList(ctx, "spwn-snapshot:")
}

// RestoreSnapshot creates a new world from a snapshot compile.
func (a *Architect) RestoreSnapshot(ctx context.Context, snapshotTag string, opts SpawnOpts) (*SpawnResult, error) {
	opts.Image = snapshotTag
	return a.Spawn(ctx, opts)
}

// DeleteSnapshot removes a snapshot compile.
func (a *Architect) DeleteSnapshot(ctx context.Context, snapshotTag string) error {
	return a.backend.ImageRemove(ctx, snapshotTag)
}

// Attach opens an interactive shell into a running world.
func (a *Architect) Attach(ctx context.Context, worldID string) error {
	u, err := a.state.Get(worldID)
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
