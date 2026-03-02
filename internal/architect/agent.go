package architect

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jterrazz/universe/internal/backend"
	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/journal"
	"github.com/jterrazz/universe/internal/session"
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
	a.state.UpdateStatus(universeID, config.StatusRunning)

	// Session management
	mindPath := u.MindPath
	cmd := a.buildClaudeCmd(mindPath, agentName, universeID)

	// Pass ANTHROPIC_API_KEY
	var env []string
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		env = append(env, "ANTHROPIC_API_KEY="+apiKey)
	}

	startTime := time.Now()

	exitCode, err := a.backend.Exec(ctx, u.ContainerID, backend.ExecConfig{
		Cmd: cmd,
		Env: env,
		TTY: true,
	})

	duration := time.Since(startTime)

	// Save session (best-effort)
	if mindPath != "" {
		sessID := session.DeterministicID(agentName, universeID)
		sess := &session.Session{
			ID:         sessID,
			AgentName:  agentName,
			UniverseID: universeID,
			Resumed:    true,
		}
		if saveErr := session.Save(mindPath, sess); saveErr != nil {
			log.Printf("warning: failed to save session: %v", saveErr)
		}
	}

	// Write journal entry (best-effort)
	if mindPath != "" {
		ec := exitCode
		if err != nil {
			ec = 1
		}
		if journalErr := journal.Append(mindPath, universeID, ec, duration); journalErr != nil {
			log.Printf("warning: failed to write journal: %v", journalErr)
		}
	}

	a.state.UpdateStatus(universeID, config.StatusIdle)

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

	a.state.UpdateStatus(universeID, config.StatusRunning)

	mindPath := u.MindPath
	cmd := a.buildClaudeCmd(mindPath, agentName, universeID)

	var env []string
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		env = append(env, "ANTHROPIC_API_KEY="+apiKey)
	}

	// Save session for detached mode (best-effort, no journal since exit unknown)
	if mindPath != "" {
		sessID := session.DeterministicID(agentName, universeID)
		sess := &session.Session{
			ID:         sessID,
			AgentName:  agentName,
			UniverseID: universeID,
			Resumed:    false,
		}
		if saveErr := session.Save(mindPath, sess); saveErr != nil {
			log.Printf("warning: failed to save session: %v", saveErr)
		}
	}

	return a.backend.ExecDetached(ctx, u.ContainerID, backend.ExecConfig{
		Cmd: cmd,
		Env: env,
		TTY: false,
	})
}

// buildClaudeCmd constructs the Claude Code command with session flags.
func (a *Architect) buildClaudeCmd(mindPath, agentName, universeID string) []string {
	cmd := []string{"claude", "--dangerously-skip-permissions"}

	if mindPath == "" {
		return cmd
	}

	sessID := session.DeterministicID(agentName, universeID)
	cmd = append(cmd, "--session-id", sessID)

	// Check if session exists (resume vs new)
	existing, err := session.Load(mindPath, universeID)
	if err == nil && existing != nil {
		cmd = append(cmd, "--resume")
	}

	return cmd
}
