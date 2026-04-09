package claude

import (
	rt "spwn.sh/core/universe/internal/runtime"
)

// Claude implements the Runtime interface for Claude Code CLI.
type Claude struct{}

func init() { rt.Register(&Claude{}) }

// Name returns the runtime identifier.
func (c *Claude) Name() string { return "claude-code" }

// BuildCommand constructs the claude CLI command with all flags.
func (c *Claude) BuildCommand(cfg rt.SpawnConfig) []string {
	cmd := []string{"claude", "--dangerously-skip-permissions"}

	// NPC mode: no Mind, just print
	if cfg.MindPath == "" {
		if cfg.Prompt != "" {
			cmd = append(cmd, "-p", cfg.Prompt, "--print")
		}
		return cmd
	}

	// Worker/Manager/Chief: session management via --resume
	// If a SessionID is provided (from a previous response), resume that session.
	// Otherwise start a fresh session. The session_id is returned in the JSON
	// response and should be captured by the caller for subsequent messages.
	if cfg.SessionID != "" {
		cmd = append(cmd, "--resume", cfg.SessionID)
	}

	if cfg.Prompt != "" {
		cmd = append(cmd, "-p", cfg.Prompt)
	}

	return cmd
}

// InstallCommands returns shell commands to install Claude Code.
func (c *Claude) InstallCommands() []string {
	return []string{"npm install -g @anthropic-ai/claude-code@latest"}
}

// RequiredEnvVars returns env var names needed for auth.
func (c *Claude) RequiredEnvVars() []string {
	return []string{"ANTHROPIC_API_KEY"}
}

// OptionalEnvVars returns useful but not required env vars.
func (c *Claude) OptionalEnvVars() []string {
	return []string{"CLAUDE_CODE_OAUTH_TOKEN", "ANTHROPIC_AUTH_TOKEN"}
}

// BaseImage returns the Docker base image needed.
func (c *Claude) BaseImage() string { return "node:20" }

// SystemPackages returns apt packages needed beyond the base image.
func (c *Claude) SystemPackages() []string {
	return []string{"git", "jq", "curl", "wget"}
}

// SupportsSession returns true if the runtime can resume sessions.
func (c *Claude) SupportsSession() bool { return true }
func (c *Claude) Available() bool       { return true }
func (c *Claude) CredentialFiles() map[string]string { return nil }
