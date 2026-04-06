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
	Name      string
	Role      string // "chief", "manager", "worker", or "npc"
	Ephemeral bool   // true for NPC-style throwaway agents
}

// DeployAgent adds a single agent to a running world: validates the mind,
// registers it in state, and starts the agent process in the background.
// Safe to call on a world that's already running with other agents.
func (a *Architect) DeployAgent(ctx context.Context, worldID, agentName, role string) error {
	if err := agent.ValidateMind(agentName); err != nil {
		return fmt.Errorf("agent %q: %w", agentName, err)
	}

	u, err := a.state.Get(worldID)
	if err != nil {
		return err
	}
	if u.Status != models.StatusRunning && u.Status != models.StatusIdle {
		return fmt.Errorf("world %s is not running (status: %s)", worldID, u.Status)
	}

	for _, existing := range u.Agents {
		if existing.Name == agentName {
			return fmt.Errorf("agent %q is already deployed in world %s", agentName, worldID)
		}
	}

	resolvedRole := manifest.DefaultRole(role)
	agentID := foundation.GenerateAgentID(agentName)
	rec := models.AgentRecord{
		Name:    agentName,
		AgentID: agentID,
		Role:    resolvedRole,
		Status:  models.StatusRunning,
	}
	if err := a.state.AddAgent(worldID, rec); err != nil {
		return fmt.Errorf("register agent: %w", err)
	}

	if err := a.SpawnAgentDetached(ctx, worldID, agentName); err != nil {
		_ = a.state.RemoveAgent(worldID, agentID)
		return fmt.Errorf("start agent: %w", err)
	}

	return nil
}

// SpawnAgents spawns multiple agents in a world.
// Chiefs are spawned first (blocking), then managers and workers (detached).
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

	// 2. Separate chiefs, managers, and workers
	var chiefs, managers, workers []AgentSpec
	for _, spec := range agents {
		role := manifest.DefaultRole(spec.Role)
		switch role {
		case "chief":
			chiefs = append(chiefs, spec)
		case "manager":
			managers = append(managers, spec)
		case "worker":
			workers = append(workers, spec)
		default:
			return fmt.Errorf("agent %q: invalid role %q.\nUse a valid role in the colony spec", spec.Name, spec.Role)
		}
	}

	if len(chiefs) > 1 {
		return fmt.Errorf("at most one chief allowed, got %d.\nRemove extra chiefs from the colony spec", len(chiefs))
	}

	// 3. Update existing agent records to "creating" status
	// (agents are already registered by Spawn() — avoid duplicates)
	for _, spec := range agents {
		agentID := foundation.GenerateAgentID(spec.Name)
		if err := a.state.UpdateAgentStatus(worldID, agentID, models.StatusCreating); err != nil {
			// Agent not yet registered (shouldn't happen in normal flow) — add it
			role := manifest.DefaultRole(spec.Role)
			rec := models.AgentRecord{
				Name:    spec.Name,
				AgentID: agentID,
				Role:    role,
				Status:  models.StatusCreating,
			}
			if addErr := a.state.AddAgent(worldID, rec); addErr != nil {
				return fmt.Errorf("register agent %q: %w", spec.Name, addErr)
			}
		}
	}

	// 4. Spawn chief first (detached — chiefs run in background like others)
	for _, ch := range chiefs {
		if err := a.SpawnAgentDetached(ctx, worldID, ch.Name); err != nil {
			return fmt.Errorf("spawn chief %q: %w", ch.Name, err)
		}
	}

	// 5. Spawn managers detached
	for _, mgr := range managers {
		if err := a.SpawnAgentDetached(ctx, worldID, mgr.Name); err != nil {
			return fmt.Errorf("spawn manager %q: %w", mgr.Name, err)
		}
	}

	// 6. Spawn workers detached
	for _, wkr := range workers {
		if err := a.SpawnAgentDetached(ctx, worldID, wkr.Name); err != nil {
			return fmt.Errorf("spawn worker %q: %w", wkr.Name, err)
		}
	}

	return nil
}
