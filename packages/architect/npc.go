package architect

import (
	"context"
	"fmt"
	"os"

	"spwn.sh/packages/container/backend"
	"spwn.sh/packages/world/models"
	"spwn.sh/packages/transpile/worldbook"
	"spwn.sh/packages/runtimes"
)

// SpawnNPC runs an NPC - an ephemeral agent with a single task, no Mind, no persistence.
// NPCs have no persistent identity, no journal, and no session state.
func (a *Architect) SpawnNPC(ctx context.Context, worldID string, task string) error {
	u, err := a.state.Get(worldID)
	if err != nil {
		return fmt.Errorf("world %s not found.\nRun 'spwn list' to see active worlds", worldID)
	}

	running, err := a.backend.IsRunning(ctx, u.ContainerID)
	if err != nil {
		return fmt.Errorf("check container: %w", err)
	}
	if !running {
		return fmt.Errorf("world %s is not running.\nStart a world first with 'spwn world'", worldID)
	}

	a.state.UpdateStatus(worldID, models.StatusRunning)

	// Generate AGENT.md for NPC (minimal context)
	agentCtx := worldbook.GenerateAgentContext(worldbook.AgentContextOpts{
		Role:       "npc",
		Ephemeral:  true,
		WorldID:    worldID,
		NPCTask:    task,
		Workspaces: convertWorkspaces(u.Workspaces),
		Deps:    u.Manifest.Deps,
	})
	if err := a.backend.CopyTo(ctx, u.ContainerID, "world/AGENT.md", []byte(agentCtx)); err != nil {
		// Non-fatal: log warning but continue
		fmt.Fprintf(os.Stderr, "warning: failed to write NPC AGENT.md: %v\n", err)
	}

	// Build a minimal claude command - no Mind, no session
	cmd := a.runtime.BuildCommand(runtimes.SpawnConfig{
		Prompt: task,
	})

	env := agentEnv()

	exitCode, err := a.backend.Exec(ctx, u.ContainerID, backend.ExecConfig{
		Cmd: cmd,
		Env: env,
		TTY: false,
	})

	a.state.UpdateStatus(worldID, models.StatusIdle)

	if err != nil {
		return fmt.Errorf("exec NPC: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("npc exited with code %d.\nCheck container logs with 'spwn logs %s' for details", exitCode, worldID)
	}
	return nil
}
