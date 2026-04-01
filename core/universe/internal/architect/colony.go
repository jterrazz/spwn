package architect

import (
	"context"
	"fmt"

	"spwn.sh/core/agent"
	"spwn.sh/core/foundation"
	"spwn.sh/core/universe/internal/manifest"
	"spwn.sh/core/universe/internal/models"
)

// AgentSpec describes an agent to spawn in a universe.
type AgentSpec struct {
	Name string
	Tier string // "governor" or "citizen"
}

// SpawnAgents spawns multiple agents in a world.
// Governors are spawned first (blocking), then citizens (detached).
func (a *Architect) SpawnAgents(ctx context.Context, worldID string, agents []AgentSpec) error {
	if len(agents) == 0 {
		return nil
	}

	// 1. Validate all agents exist and have valid Minds
	for _, spec := range agents {
		if err := agent.ValidateMind(spec.Name); err != nil {
			return fmt.Errorf("agent %q: %w", spec.Name, err)
		}
	}

	// 2. Separate governors and citizens
	var governors, citizens []AgentSpec
	for _, spec := range agents {
		tier := manifest.DefaultTier(spec.Tier)
		switch tier {
		case "governor":
			governors = append(governors, spec)
		case "citizen":
			citizens = append(citizens, spec)
		default:
			return fmt.Errorf("agent %q: invalid tier %q.\nUse \"governor\" or \"citizen\" in the colony spec", spec.Name, spec.Tier)
		}
	}

	if len(governors) > 1 {
		return fmt.Errorf("at most one governor allowed, got %d.\nRemove extra governors from the colony spec", len(governors))
	}

	// 3. Update existing agent records to "creating" status
	// (agents are already registered by Spawn() — avoid duplicates)
	for _, spec := range agents {
		agentID := foundation.GenerateAgentID(spec.Name)
		if err := a.state.UpdateAgentStatus(worldID, agentID, models.StatusCreating); err != nil {
			// Agent not yet registered (shouldn't happen in normal flow) — add it
			tier := manifest.DefaultTier(spec.Tier)
			rec := models.AgentRecord{
				Name:    spec.Name,
				AgentID: agentID,
				Tier:    tier,
				Status:  models.StatusCreating,
			}
			if addErr := a.state.AddAgent(worldID, rec); addErr != nil {
				return fmt.Errorf("register agent %q: %w", spec.Name, addErr)
			}
		}
	}

	// 4. Spawn governor first (blocking)
	for _, gov := range governors {
		if err := a.SpawnAgent(ctx, worldID, gov.Name); err != nil {
			return fmt.Errorf("spawn governor %q: %w", gov.Name, err)
		}
	}

	// 5. Spawn citizens detached
	for _, cit := range citizens {
		if err := a.SpawnAgentDetached(ctx, worldID, cit.Name); err != nil {
			return fmt.Errorf("spawn citizen %q: %w", cit.Name, err)
		}
	}

	return nil
}
