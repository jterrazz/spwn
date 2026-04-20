package architect

import (
	"context"
	"fmt"

	"spwn.sh/packages/container/backend"
	"spwn.sh/packages/runtimes"
)

// SpawnNPC runs an NPC - an ephemeral agent with a single task, no Mind, no persistence.
// NPCs have no persistent identity, no journal, and no session state.
func (a *Architect) SpawnNPC(ctx context.Context, worldID string, task string) error {
	u, err := a.rstate.Get(worldID)
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

	rt, err := a.resolveSpawner(u)
	if err != nil {
		return err
	}

	// Build a minimal one-shot command — no Mind, no session. NPCs
	// receive their full context via the task prompt itself (see the
	// `Prompt:` field below); nothing needs to be written to disk.
	// The runtime adapter decides what "one-shot" means — claude
	// takes `-p --print`, codex takes `exec`, etc.
	cmd := rt.BuildCommand(runtimes.SpawnConfig{
		Prompt: task,
	})

	env := agentEnv()

	exitCode, err := a.backend.Exec(ctx, u.ContainerID, backend.ExecConfig{
		Cmd: cmd,
		Env: env,
		TTY: false,
	})

	if err != nil {
		return fmt.Errorf("exec NPC: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("npc exited with code %d.\nCheck container logs with 'spwn logs %s' for details", exitCode, worldID)
	}
	return nil
}
