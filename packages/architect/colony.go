package architect

import (
	"context"
	"fmt"
	"strings"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/architect/internal/deploy"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/transpile"
	"spwn.sh/packages/world/models"
)

// AgentSpec describes an agent to spawn in a world.
type AgentSpec struct {
	Name      string
	Role      string
	Ephemeral bool
}

// DeployAgent adds a single agent to a running world: validates the
// mind, creates the agent's per-world deployment dirs on the host,
// syncs the agent home into the container, regenerates roster.md,
// and starts the agent process in the background. Safe to call on a
// world that's already running with other agents.
//
// Hot-deploy uses the same docker-cp mechanism as cold spawn: the
// host-side spwn/agents/<name>/ tree is copied into the container at
// /agents/<name>/ once, and per-agent compile output (CLAUDE.md,
// role.md) is docker-cp'd on top. Subsequent writes inside the
// container are only flushed back on graceful world destroy.
func (a *Architect) DeployAgent(ctx context.Context, worldID, agentName, role string) error {
	if err := agent.ValidateMind(agentName); err != nil {
		return fmt.Errorf("agent %q: %w", agentName, err)
	}

	u, err := a.rstate.Get(worldID)
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

	resolvedRole := agent.DefaultRole(role)
	agentID := platform.GenerateAgentID(agentName)
	rec := models.AgentRecord{
		Name:    agentName,
		AgentID: agentID,
		Role:    resolvedRole,
		Status:  models.StatusRunning,
	}

	if err := initAgentDeploymentDirs(rec, worldID); err != nil {
		return fmt.Errorf("init deployment: %w", err)
	}

	agentHome := "/agents/" + agentName
	if err := deploy.SyncIn(ctx, a.backend, u.ContainerID, map[string]string{agentName: agentHome}); err != nil {
		return fmt.Errorf("sync agent into container: %w", err)
	}

	hotTree, err := transpile.Compile(resolveRuntimeName(u), transpile.Input{
		Deps:          nil,
		VerifiedTools: nil,
		WorldID:       worldID,
		Agents:        []transpile.AgentInput{{Name: rec.Name, Role: resolvedRole}},
	})
	if err != nil {
		return fmt.Errorf("compile agent deployment: %w", err)
	}
	var hotCpErr error
	hotTree.Walk(func(path string, content []byte) {
		if hotCpErr != nil {
			return
		}
		const prefix = "agents/"
		if !strings.HasPrefix(path, prefix) {
			return
		}
		containerPath := "/" + path
		if err := a.backend.CopyTo(ctx, u.ContainerID, containerPath, content); err != nil {
			hotCpErr = fmt.Errorf("cp %s into container: %w", containerPath, err)
		}
	})
	if hotCpErr != nil {
		return hotCpErr
	}

	if err := a.rstate.AddAgent(worldID, rec); err != nil {
		return fmt.Errorf("register agent: %w", err)
	}

	if err := a.SpawnAgentDetached(ctx, worldID, agentName); err != nil {
		_ = a.rstate.RemoveAgent(worldID, agentID)
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

	for _, spec := range agents {
		if err := agent.ValidateMind(spec.Name); err != nil {
			return fmt.Errorf("agent %q: %w", spec.Name, err)
		}
	}

	var chiefs, managers, workers []AgentSpec
	for _, spec := range agents {
		role := agent.DefaultRole(spec.Role)
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

	for _, spec := range agents {
		agentID := platform.GenerateAgentID(spec.Name)
		if err := a.rstate.UpdateAgentStatus(worldID, agentID, models.StatusCreating); err != nil {
			role := agent.DefaultRole(spec.Role)
			rec := models.AgentRecord{
				Name:    spec.Name,
				AgentID: agentID,
				Role:    role,
				Status:  models.StatusCreating,
			}
			if addErr := a.rstate.AddAgent(worldID, rec); addErr != nil {
				return fmt.Errorf("register agent %q: %w", spec.Name, addErr)
			}
		}
	}

	for _, ch := range chiefs {
		if err := a.SpawnAgentDetached(ctx, worldID, ch.Name); err != nil {
			return fmt.Errorf("spawn chief %q: %w", ch.Name, err)
		}
	}

	for _, mgr := range managers {
		if err := a.SpawnAgentDetached(ctx, worldID, mgr.Name); err != nil {
			return fmt.Errorf("spawn manager %q: %w", mgr.Name, err)
		}
	}

	for _, wkr := range workers {
		if err := a.SpawnAgentDetached(ctx, worldID, wkr.Name); err != nil {
			return fmt.Errorf("spawn worker %q: %w", wkr.Name, err)
		}
	}

	return nil
}
