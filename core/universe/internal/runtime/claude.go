package runtime

import (
	"spwn.sh/core/agent"
)

// SpawnConfig holds the parameters needed to build a runtime command.
type SpawnConfig struct {
	MindPath   string
	AgentName  string
	UniverseID string
	Prompt     string // If set, passed as the initial prompt (used for NPCs).
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

	// NPC mode: no Mind, no session — just run the prompt and exit
	if cfg.MindPath == "" {
		if cfg.Prompt != "" {
			cmd = append(cmd, "-p", cfg.Prompt, "--print")
		}
		return cmd
	}

	// Citizen/Governor mode: session management
	sessID := agent.DeterministicSessionID(cfg.AgentName, cfg.UniverseID)
	cmd = append(cmd, "--session-id", sessID)

	existing, err := agent.LoadSession(cfg.MindPath, cfg.UniverseID)
	if err == nil && existing != nil {
		cmd = append(cmd, "--resume")
	}

	if cfg.Prompt != "" {
		cmd = append(cmd, "-p", cfg.Prompt)
	}

	return cmd
}
