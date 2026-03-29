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

// SpawnAgent execs Claude Code interactively inside a universe.
func (a *Architect) SpawnAgent(ctx context.Context, universeID, agentName string) error {
	u, err := a.state.Get(universeID)
	if err != nil {
		return err
	}

	running, err := a.backend.IsRunning(ctx, u.ContainerID)
	if err != nil {
		return fmt.Errorf("check container: %w", err)
	}
	if !running {
		return fmt.Errorf("universe %s is not running", universeID)
	}

	// Update status
	a.state.UpdateStatus(universeID, models.StatusRunning)

	// Session management
	mindPath := u.MindPath
	cmd := a.runtime.BuildCommand(runtime.SpawnConfig{
		MindPath:   mindPath,
		AgentName:  agentName,
		UniverseID: universeID,
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
		sessID := agent.DeterministicSessionID(agentName, universeID)
		sess := &agent.Session{
			ID:         sessID,
			AgentName:  agentName,
			UniverseID: universeID,
			Resumed:    true,
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
		if journalErr := agent.AppendJournal(mindPath, universeID, ec, duration); journalErr != nil {
			log.Printf("warning: failed to write journal: %v", journalErr)
		}
	}

	a.state.UpdateStatus(universeID, models.StatusIdle)

	if err != nil {
		return fmt.Errorf("exec claude: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("agent exited with code %d", exitCode)
	}
	return nil
}

// SpawnAgentDetached starts Claude Code in the background.
func (a *Architect) SpawnAgentDetached(ctx context.Context, universeID, agentName string) error {
	u, err := a.state.Get(universeID)
	if err != nil {
		return err
	}

	running, err := a.backend.IsRunning(ctx, u.ContainerID)
	if err != nil {
		return fmt.Errorf("check container: %w", err)
	}
	if !running {
		return fmt.Errorf("universe %s is not running", universeID)
	}

	a.state.UpdateStatus(universeID, models.StatusRunning)

	mindPath := u.MindPath
	cmd := a.runtime.BuildCommand(runtime.SpawnConfig{
		MindPath:   mindPath,
		AgentName:  agentName,
		UniverseID: universeID,
	})

	env := agentEnv()

	// Save session for detached mode (best-effort, no journal since exit unknown)
	if mindPath != "" {
		sessID := agent.DeterministicSessionID(agentName, universeID)
		sess := &agent.Session{
			ID:         sessID,
			AgentName:  agentName,
			UniverseID: universeID,
			Resumed:    false,
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

