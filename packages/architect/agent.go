package architect

import (
	"context"
	"fmt"
	"log"
	"time"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/container/backend"
	"spwn.sh/packages/runtimes"
)

// SpawnAgent execs the world's runtime interactively inside its
// container. The runtime is whatever the world record declares
// (`spwn:claude-code`, `spwn:codex`, …) — resolved per-call so one
// Architect drives heterogeneous worlds without re-instantiation.
func (a *Architect) SpawnAgent(ctx context.Context, worldID, agentName string) error {
	u, err := a.rstate.Get(worldID)
	if err != nil {
		return err
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

	// Session management lives entirely in the runtime adapter —
	// BuildCommand knows whether to pass --resume (claude), --thread
	// (codex), etc. Status is derived from container state on every
	// List/Get — no explicit transition needed here.
	cmd := rt.BuildCommand(runtimes.SpawnConfig{
		AgentName: agentName,
		WorldID:   worldID,
	})

	// Forward auth credentials to the exec
	env := agentEnv()

	startTime := time.Now()

	exitCode, err := a.backend.Exec(ctx, u.ContainerID, backend.ExecConfig{
		Cmd: cmd,
		Env: env,
		TTY: true,
	})

	duration := time.Since(startTime)

	// Save session + journal (best-effort) - both live in the agent's
	// persistent home dir, addressed by name.
	agentPath := agent.AgentDir(agentName)
	sessID := agent.DeterministicSessionID(agentName, worldID)
	sess := &agent.Session{
		ID:        sessID,
		AgentName: agentName,
		WorldID:   worldID,
		Resumed:   true,
	}
	if saveErr := agent.SaveSession(agentPath, sess); saveErr != nil {
		log.Printf("warning: failed to save session: %v", saveErr)
	}
	ec := exitCode
	if err != nil {
		ec = 1
	}
	if journalErr := agent.AppendJournal(agentPath, worldID, ec, duration); journalErr != nil {
		log.Printf("warning: failed to write journal: %v", journalErr)
	}

	if err != nil {
		return fmt.Errorf("exec claude: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("agent exited with code %d.\nCheck container logs with 'spwn logs %s' for details", exitCode, worldID)
	}
	return nil
}

// SpawnAgentDetached starts the world's runtime in the background.
// Mirrors SpawnAgent for session continuity but returns as soon as
// the exec is accepted by the container — no TTY, no blocking on
// the runtime process.
func (a *Architect) SpawnAgentDetached(ctx context.Context, worldID, agentName string) error {
	u, err := a.rstate.Get(worldID)
	if err != nil {
		return err
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

	cmd := rt.BuildCommand(runtimes.SpawnConfig{
		AgentName: agentName,
		WorldID:   worldID,
	})

	env := agentEnv()

	// Save session for detached mode (best-effort, no journal since exit unknown)
	agentPath := agent.AgentDir(agentName)
	sessID := agent.DeterministicSessionID(agentName, worldID)
	sess := &agent.Session{
		ID:        sessID,
		AgentName: agentName,
		WorldID:   worldID,
		Resumed:   false,
	}
	if saveErr := agent.SaveSession(agentPath, sess); saveErr != nil {
		log.Printf("warning: failed to save session: %v", saveErr)
	}

	return a.backend.ExecDetached(ctx, u.ContainerID, backend.ExecConfig{
		Cmd: cmd,
		Env: env,
		TTY: false,
	})
}

// agentEnv builds environment variables for agent execution inside containers.
func agentEnv() []string {
	return DockerEnvVars()
}

