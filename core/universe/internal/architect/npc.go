package architect

import (
	"context"
	"fmt"
	"os"

	"spwn.sh/core/universe/internal/backend"
	"spwn.sh/core/universe/internal/models"
	"spwn.sh/core/universe/internal/physics"
	"spwn.sh/core/universe/internal/runtime"
)

// SpawnNPC runs an NPC — an ephemeral agent with a single task, no Mind, no persistence.
// NPCs have no persistent identity, no journal, and no session state.
func (a *Architect) SpawnNPC(ctx context.Context, universeID string, task string) error {
	u, err := a.state.Get(universeID)
	if err != nil {
		return fmt.Errorf("world %s not found", universeID)
	}

	running, err := a.backend.IsRunning(ctx, u.ContainerID)
	if err != nil {
		return fmt.Errorf("check container: %w", err)
	}
	if !running {
		return fmt.Errorf("world %s is not running", universeID)
	}

	a.state.UpdateStatus(universeID, models.StatusRunning)

	// Generate AGENT.md for NPC (minimal context)
	agentCtx := physics.GenerateAgentContext(physics.AgentContextOpts{
		Tier:      "npc",
		WorldID:   universeID,
		NPCTask:   task,
		Workspace: u.Workspace,
		Elements:  u.Manifest.Elements,
	})
	if err := a.backend.CopyTo(ctx, u.ContainerID, "world/AGENT.md", []byte(agentCtx)); err != nil {
		// Non-fatal: log warning but continue
		fmt.Fprintf(os.Stderr, "warning: failed to write NPC AGENT.md: %v\n", err)
	}

	// Build a minimal claude command — no Mind, no session
	cmd := a.runtime.BuildCommand(runtime.SpawnConfig{
		Prompt: task,
	})

	var env []string
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		env = append(env, "ANTHROPIC_API_KEY="+apiKey)
	}

	exitCode, err := a.backend.Exec(ctx, u.ContainerID, backend.ExecConfig{
		Cmd: cmd,
		Env: env,
		TTY: true,
	})

	a.state.UpdateStatus(universeID, models.StatusIdle)

	if err != nil {
		return fmt.Errorf("exec NPC: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("NPC exited with code %d", exitCode)
	}
	return nil
}
