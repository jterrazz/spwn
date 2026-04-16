package world

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"spwn.sh/packages/platform"
	"spwn.sh/packages/architect"
	"spwn.sh/packages/world"
	"spwn.sh/packages/world/models"
)

// findRunningWorldByConfig returns a pointer to the most recently
// created running/idle world whose Config equals the given name, or
// nil if none is active. Errors are swallowed - a best-effort check;
// if Docker is unreachable the caller will hit a clearer error later
// in the spawn flow.
func findRunningWorldByConfig(ctx context.Context, name string) *models.World {
	arc, err := architect.NewFromEnv()
	if err != nil {
		return nil
	}
	worlds, err := arc.List(ctx)
	if err != nil {
		return nil
	}
	var match *models.World
	for i := range worlds {
		w := &worlds[i]
		if w.Config != name {
			continue
		}
		if w.Status != world.StatusRunning && w.Status != world.StatusIdle && w.Status != world.StatusCreating {
			continue
		}
		if w.ContainerID != "" && !containerExists(w.ContainerID) {
			continue
		}
		if match == nil || w.CreatedAt.After(match.CreatedAt) {
			match = w
		}
	}
	return match
}

// acquireUpLock creates an exclusive lock file for the given world
// name under the project's .spwn/ state dir. Returns an unlock func
// on success, or an error carrying the lock file path on failure
// (another `spwn up` is in progress, or the file is stale).
func acquireUpLock(worldName string) (func(), error) {
	dir := platform.LocalStateDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("%s: %w", dir, err)
	}
	lockPath := filepath.Join(dir, ".up."+worldName+".lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		// Surface the path via the returned error message so the
		// caller can show a meaningful hint.
		return nil, fmt.Errorf("%s", lockPath)
	}
	_, _ = fmt.Fprintf(f, "pid=%d\n", os.Getpid())
	f.Close()
	return func() { _ = os.Remove(lockPath) }, nil
}
