package architect

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"spwn.sh/core/agent"
	"spwn.sh/core/foundation"
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

	// Write journal entries for all agents in the world (best-effort).
	// Multi-agent worlds store agents in the Agents slice; single-agent
	// worlds only have MindPath set.
	duration := time.Since(u.CreatedAt)
	if len(u.Agents) > 0 {
		for _, rec := range u.Agents {
			agentPath := filepath.Join(foundation.AgentsDir(), rec.Name)
			if journalErr := agent.AppendJournal(agentPath, worldID, -1, duration); journalErr != nil {
				log.Printf("warning: failed to write journal for agent %s on destroy: %v", rec.Name, journalErr)
			}
		}
	} else if u.MindPath != "" {
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

// DestroyAll stops and removes all worlds sequentially.
// Returns the list of destroyed worlds and the first error encountered (if any).
func (a *Architect) DestroyAll(ctx context.Context) ([]*models.World, error) {
	worlds, err := a.state.List()
	if err != nil {
		return nil, err
	}

	var destroyed []*models.World
	for _, w := range worlds {
		u, err := a.Destroy(ctx, w.ID)
		if err != nil {
			log.Printf("warning: failed to destroy world %s: %v", w.ID, err)
			continue
		}
		destroyed = append(destroyed, u)
	}
	return destroyed, nil
}
