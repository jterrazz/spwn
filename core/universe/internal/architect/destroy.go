package architect

import (
	"context"
	"os"

	"spwn.sh/core/universe/internal/models"
)

// Destroy stops and removes a world.
func (a *Architect) Destroy(ctx context.Context, worldID string) (*models.World, error) {
	u, err := a.state.Get(worldID)
	if err != nil {
		return nil, err
	}

	// Stop gate server if running
	if srv, ok := a.gates[worldID]; ok {
		srv.Stop()
		delete(a.gates, worldID)
	}

	a.backend.Stop(ctx, u.ContainerID)
	a.backend.Remove(ctx, u.ContainerID)

	// Clean up gate temp directory
	if u.GateDir != "" {
		os.RemoveAll(u.GateDir)
	}

	a.state.Delete(worldID)

	return u, nil
}
