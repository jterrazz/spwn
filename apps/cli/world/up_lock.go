package world

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"spwn.sh/packages/architect"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/world"
	"spwn.sh/packages/world/models"
)

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

func acquireUpLock(worldName string) (func(), error) {
	dir := platform.LocalStateDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("%s: %w", dir, err)
	}
	lockPath := filepath.Join(dir, ".up."+worldName+".lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("%s", lockPath)
	}
	_, _ = fmt.Fprintf(f, "pid=%d\n", os.Getpid())
	f.Close()
	return func() { _ = os.Remove(lockPath) }, nil
}
