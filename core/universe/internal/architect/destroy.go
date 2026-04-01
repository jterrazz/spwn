package architect

import (
	"context"
	"log"
	"os"
	"time"

	"spwn.sh/core/agent"
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

	// Write journal entry for this world session (best-effort)
	if u.MindPath != "" {
		duration := time.Since(u.CreatedAt)
		if journalErr := agent.AppendJournal(u.MindPath, worldID, -1, duration); journalErr != nil {
			log.Printf("warning: failed to write journal on destroy: %v", journalErr)
		}
	}

	// Clean up gate temp directory
	if u.GateDir != "" {
		os.RemoveAll(u.GateDir)
	}

	a.state.Delete(worldID)

	return u, nil
}
