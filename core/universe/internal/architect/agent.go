package architect

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"spwn.sh/core/agent"
	"spwn.sh/core/foundation"
	"spwn.sh/core/universe/internal/backend"
	"spwn.sh/core/universe/internal/models"
	"spwn.sh/core/universe/internal/runtime"
)

// SpawnAgent execs Claude Code interactively inside a world.
func (a *Architect) SpawnAgent(ctx context.Context, worldID, agentName string) error {
	u, err := a.state.Get(worldID)
	if err != nil {
		return err
	}

	running, err := a.backend.IsRunning(ctx, u.ContainerID)
	if err != nil {
		return fmt.Errorf("check container: %w", err)
	}
	if !running {
		return fmt.Errorf("world %s is not running", worldID)
	}

	// Update status
	a.state.UpdateStatus(worldID, models.StatusRunning)

	// Session management
	mindPath := u.MindPath
	cmd := a.runtime.BuildCommand(runtime.SpawnConfig{
		MindPath:  mindPath,
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

	// Save session (best-effort)
	if mindPath != "" {
		sessID := agent.DeterministicSessionID(agentName, worldID)
		sess := &agent.Session{
			ID:        sessID,
			AgentName: agentName,
			WorldID:   worldID,
			Resumed:   true,
		}
		if saveErr := agent.SaveSession(mindPath, sess); saveErr != nil {
			log.Printf("warning: failed to save session: %v", saveErr)
		}
	}

	// Write journal entry (best-effort)
	if mindPath != "" {
		ec := exitCode
		if err != nil {
			ec = 1
		}
		if journalErr := agent.AppendJournal(mindPath, worldID, ec, duration); journalErr != nil {
			log.Printf("warning: failed to write journal: %v", journalErr)
		}
	}

	a.state.UpdateStatus(worldID, models.StatusIdle)

	if err != nil {
		return fmt.Errorf("exec claude: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("agent exited with code %d", exitCode)
	}
	return nil
}

// SpawnAgentDetached starts Claude Code in the background.
func (a *Architect) SpawnAgentDetached(ctx context.Context, worldID, agentName string) error {
	u, err := a.state.Get(worldID)
	if err != nil {
		return err
	}

	running, err := a.backend.IsRunning(ctx, u.ContainerID)
	if err != nil {
		return fmt.Errorf("check container: %w", err)
	}
	if !running {
		return fmt.Errorf("world %s is not running", worldID)
	}

	a.state.UpdateStatus(worldID, models.StatusRunning)

	mindPath := u.MindPath
	cmd := a.runtime.BuildCommand(runtime.SpawnConfig{
		MindPath:  mindPath,
		AgentName: agentName,
		WorldID:   worldID,
	})

	env := agentEnv()

	// Save session for detached mode (best-effort, no journal since exit unknown)
	if mindPath != "" {
		sessID := agent.DeterministicSessionID(agentName, worldID)
		sess := &agent.Session{
			ID:        sessID,
			AgentName: agentName,
			WorldID:   worldID,
			Resumed:   false,
		}
		if saveErr := agent.SaveSession(mindPath, sess); saveErr != nil {
			log.Printf("warning: failed to save session: %v", saveErr)
		}
	}

	return a.backend.ExecDetached(ctx, u.ContainerID, backend.ExecConfig{
		Cmd: cmd,
		Env: env,
		TTY: false,
	})
}

// agentEnv builds environment variables for agent execution inside containers.
func agentEnv() []string {
	var env []string
	for _, key := range []string{
		"ANTHROPIC_API_KEY",
		"CLAUDE_CODE_OAUTH_TOKEN",
		"ANTHROPIC_AUTH_TOKEN",
	} {
		if val := os.Getenv(key); val != "" {
			env = append(env, key+"="+val)
		}
	}

	// Read cached OAuth token if no explicit auth set
	if !hasAgentEnv(env, "CLAUDE_CODE_OAUTH_TOKEN") && !hasAgentEnv(env, "ANTHROPIC_API_KEY") {
		cachePath := foundation.BaseDir() + "/.auth-token"
		if data, err := os.ReadFile(cachePath); err == nil {
			token := strings.TrimSpace(string(data))
			if token != "" {
				env = append(env, "CLAUDE_CODE_OAUTH_TOKEN="+token)
			}
		}
	}
	return env
}

func hasAgentEnv(env []string, key string) bool {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return true
		}
	}
	return false
}

