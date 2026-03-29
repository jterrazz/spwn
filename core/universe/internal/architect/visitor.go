package architect

import (
	"context"
	"fmt"
	"os"

	"github.com/jterrazz/spwn/core/universe/internal/backend"
	"github.com/jterrazz/spwn/core/universe/internal/models"
	"github.com/jterrazz/spwn/core/universe/internal/runtime"
)

// SpawnVisitor runs an ephemeral agent with a single task, no Mind, no persistence.
// Visitors have no persistent identity, no journal, and no session state.
func (a *Architect) SpawnVisitor(ctx context.Context, universeID string, task string) error {
	u, err := a.state.Get(universeID)
	if err != nil {
		return fmt.Errorf("universe %s not found", universeID)
	}

	running, err := a.backend.IsRunning(ctx, u.ContainerID)
	if err != nil {
		return fmt.Errorf("check container: %w", err)
	}
	if !running {
		return fmt.Errorf("universe %s is not running", universeID)
	}

	a.state.UpdateStatus(universeID, models.StatusRunning)

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
		return fmt.Errorf("exec visitor: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("visitor exited with code %d", exitCode)
	}
	return nil
}
