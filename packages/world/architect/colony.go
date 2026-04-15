package architect

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/compile"
	"spwn.sh/packages/compile/runtimes/claudecode"
	"spwn.sh/packages/ids"
	"spwn.sh/packages/world/manifest"
	"spwn.sh/packages/world/models"
)

// AgentSpec describes an agent to spawn in a world.
type AgentSpec struct {
	Name      string
	Role      string // "chief", "manager", "worker", or "npc"
	Ephemeral bool   // true for NPC-style throwaway agents
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
	agentID := ids.GenerateAgentID(agentName)
	rec := models.AgentRecord{
		Name:    agentName,
		AgentID: agentID,
		Role:    resolvedRole,
		Status:  models.StatusRunning,
	}

	// 1. Create the per-agent per-world layout on the host. This
	// brings hot-deployed agents up to first-class parity with
	// spawn-time agents - inbox/outbox/notes/role.md all in place.
	if err := initAgentDeploymentDirs(rec, worldID); err != nil {
		return fmt.Errorf("init deployment: %w", err)
	}

	// Sync the agent's home tree into the container at /agents/<name>/
	// — the same copy-in step spawn uses. Without this the runtime
	// process started in step 4 would find no identity/profile.md,
	// no agent.yaml, nothing.
	agentHome := "/agents/" + agentName
	if err := syncAgentsInto(ctx, a.backend, u.ContainerID, map[string]string{agentName: agentHome}); err != nil {
		return fmt.Errorf("sync agent into container: %w", err)
	}

	// Render just this agent's content (CLAUDE.md + per-world
	// role.md) through the compiler and docker-cp it on top of the
	// copied-in home. We only handle agents/* entries — the world/*
	// files already exist from spawn time.
	hotTree, err := compile.Compile("claude-code", compile.Input{
		Manifest:      models.Manifest{},
		VerifiedTools: nil,
		WorldID:       worldID,
		Agents:        []compile.AgentInput{{Name: rec.Name, Role: resolvedRole}},
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

	// 2. Register in runtimestate so the next List() includes the
	// agent in u.Agents.
	if err := a.state.AddAgent(worldID, rec); err != nil {
		return fmt.Errorf("register agent: %w", err)
	}

	// 3. Regenerate /world/roster.md so existing agents in the
	// container can see the new member on their next read. The file
	// lives in ~/.spwn/world-states/<world-id>/, visible at /world/
	// in the container via the bind mount.
	if rosterErr := regenRoster(worldID, a); rosterErr != nil {
		// Non-fatal: the agent is registered, the host filesystem is
		// in place, the runtime can talk. Just log the warning.
		fmt.Printf("warning: failed to regenerate roster: %v\n", rosterErr)
	}

	// 4. Start the runtime process in the background.
	if err := a.SpawnAgentDetached(ctx, worldID, agentName); err != nil {
		_ = a.state.RemoveAgent(worldID, agentID)
		return fmt.Errorf("start agent: %w", err)
	}

	return nil
}

// regenRoster rebuilds /world/roster.md from the current set of agents
// in the world. Called whenever the roster changes (DeployAgent today;
// agent removal in future). The file is written to the host so the
// bind mount propagates it into the container.
func regenRoster(worldID string, a *Architect) error {
	worlds, err := a.state.List()
	if err != nil {
		return err
	}
	var current *models.World
	for i := range worlds {
		if worlds[i].ID == worldID {
			current = &worlds[i]
			break
		}
	}
	if current == nil {
		return fmt.Errorf("world %s not found", worldID)
	}
	worldStateDir := worldStateDirFor(worldID)
	roster := claudecode.GenerateRoster(worldID, rosterColony(current.Agents))
	return os.WriteFile(filepath.Join(worldStateDir, "roster.md"), []byte(roster), 0o644)
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
	// (agents are already registered by Spawn() - avoid duplicates)
	for _, spec := range agents {
		agentID := ids.GenerateAgentID(spec.Name)
		if err := a.state.UpdateAgentStatus(worldID, agentID, models.StatusCreating); err != nil {
			// Agent not yet registered (shouldn't happen in normal flow) - add it
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

	// 4. Spawn chief first (detached - chiefs run in background like others)
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
