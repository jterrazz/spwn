package architect

import (
	"context"
	"fmt"
	"time"

	"spwn.sh/core/foundation/activity"
	"spwn.sh/core/gate"
	"spwn.sh/core/universe/internal/backend"
	"spwn.sh/core/universe/internal/models"
	"spwn.sh/core/universe/internal/runtime"
	"spwn.sh/core/universe/internal/state"

	// Register all runtime adapters
	_ "spwn.sh/core/universe/internal/runtime/aider"
	_ "spwn.sh/core/universe/internal/runtime/claude"
	_ "spwn.sh/core/universe/internal/runtime/codex"
	_ "spwn.sh/core/universe/internal/runtime/gemini"
	_ "spwn.sh/core/universe/internal/runtime/opencode"
	_ "spwn.sh/core/universe/internal/runtime/pi"

	// Register all claw adapters
	_ "spwn.sh/core/universe/internal/claw/hermes"
	_ "spwn.sh/core/universe/internal/claw/zeroclaw"
)

// Architect orchestrates world lifecycle.
type Architect struct {
	backend backend.Backend
	state   *state.Store
	gates   map[string]*gate.Server // worldID → running gate server
	runtime runtime.Runtime          // injected runtime adapter
}

// New creates an Architect with the given backend and state store.
func New(b backend.Backend, s *state.Store) *Architect {
	rt, _ := runtime.Get("claude-code") // default runtime
	return &Architect{
		backend: b,
		state:   s,
		gates:   make(map[string]*gate.Server),
		runtime: rt,
	}
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

// Logs streams container output.
func (a *Architect) Logs(ctx context.Context, worldID string, follow bool, tail string) (interface{ Read([]byte) (int, error); Close() error }, error) {
	u, err := a.state.Get(worldID)
	if err != nil {
		return nil, err
	}
	return a.backend.Logs(ctx, u.ContainerID, backend.LogsConfig{
		Follow: follow,
		Tail:   tail,
	})
}

// Snapshot commits the current state of a world's container as a Docker image.
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

// RestoreSnapshot creates a new world from a snapshot image.
func (a *Architect) RestoreSnapshot(ctx context.Context, snapshotTag string, opts SpawnOpts) (*SpawnResult, error) {
	opts.Image = snapshotTag
	return a.Spawn(ctx, opts)
}

// DeleteSnapshot removes a snapshot image.
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
