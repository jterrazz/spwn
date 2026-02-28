package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jterrazz/universe/internal/backend"
)

// SpawnOptions configures agent spawning behavior.
type SpawnOptions struct {
	SessionID string // If set, passes --resume <sessionID> to claude CLI.
}

// SpawnResult holds the result of an agent spawn.
type SpawnResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Spawn starts Claude Code CLI inside a universe.
func Spawn(ctx context.Context, b backend.Backend, universeID string, opts *SpawnOptions) (*SpawnResult, error) {
	slog.Info("spawning Claude Code agent", "universe_id", universeID)

	cmd := []string{"claude", "--dangerously-skip-permissions"}
	if opts != nil && opts.SessionID != "" {
		cmd = append(cmd, "--resume", opts.SessionID)
		slog.Info("resuming session", "session_id", opts.SessionID)
	}

	result, err := b.Exec(ctx, universeID, cmd)
	if err != nil {
		return &SpawnResult{}, fmt.Errorf("spawning agent in universe %s: %w", universeID, err)
	}

	sr := &SpawnResult{
		ExitCode: result.ExitCode,
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
	}

	if result.ExitCode != 0 {
		return sr, fmt.Errorf("agent spawn failed for universe %s (exit %d): %s",
			universeID, result.ExitCode, strings.TrimSpace(result.Stderr))
	}

	return sr, nil
}
