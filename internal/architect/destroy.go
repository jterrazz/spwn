package architect

import (
	"context"
	"os"

	"github.com/jterrazz/universe/internal/config"
)

// Destroy stops and removes a universe.
func (a *Architect) Destroy(ctx context.Context, universeID string) (*config.Universe, error) {
	u, err := a.state.Get(universeID)
	if err != nil {
		return nil, err
	}

	// Stop gate server if running
	if srv, ok := a.gates[universeID]; ok {
		srv.Stop()
		delete(a.gates, universeID)
	}

	a.backend.Stop(ctx, u.ContainerID)
	a.backend.Remove(ctx, u.ContainerID)

	// Clean up gate temp directory
	if u.GateDir != "" {
		os.RemoveAll(u.GateDir)
	}

	a.state.Delete(universeID)

	return u, nil
}
