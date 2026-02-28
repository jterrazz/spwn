package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jterrazz/universe/internal/backend"
)

// Spawn starts Claude Code CLI inside a universe.
func Spawn(ctx context.Context, b backend.Backend, universeID string) error {
	slog.Info("spawning Claude Code agent", "universe_id", universeID)

	result, err := b.Exec(ctx, universeID, []string{"claude", "--dangerously-skip-permissions"})
	if err != nil {
		return fmt.Errorf("spawning agent in universe %s: %w", universeID, err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("agent spawn failed for universe %s (exit %d): %s",
			universeID, result.ExitCode, strings.TrimSpace(result.Stderr))
	}

	return nil
}
