package runtime

import (
	"spwn.sh/core/agent"
)

// SpawnConfig holds the parameters needed to build a runtime command.
type SpawnConfig struct {
	MindPath   string
	AgentName  string
	UniverseID string
	Prompt     string // If set, passed as the initial prompt (used for visitors).
}

// ClaudeCode implements the Runtime port for Claude Code CLI via ACP.
type ClaudeCode struct{}

// NewClaudeCode creates a new ClaudeCode runtime adapter.
func NewClaudeCode() *ClaudeCode {
	return &ClaudeCode{}
}

// Name returns the runtime identifier.
func (c *ClaudeCode) Name() string {
	return "claude-code"
}

// BuildCommand constructs the claude CLI command with all flags.
func (c *ClaudeCode) BuildCommand(cfg SpawnConfig) []string {
	cmd := []string{"claude", "--dangerously-skip-permissions"}

	if cfg.MindPath == "" {
		return cmd
	}

	sessID := agent.DeterministicSessionID(cfg.AgentName, cfg.UniverseID)
	cmd = append(cmd, "--session-id", sessID)

	// Check if session exists (resume vs new)
	existing, err := agent.LoadSession(cfg.MindPath, cfg.UniverseID)
	if err == nil && existing != nil {
		cmd = append(cmd, "--resume")
	}

	if cfg.Prompt != "" {
		cmd = append(cmd, "-p", cfg.Prompt)
	}

	return cmd
}
