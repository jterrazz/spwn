package codex

import (
	rt "spwn.sh/core/universe/internal/runtime"
)

// Codex implements the Runtime interface for OpenAI Codex CLI.
type Codex struct{}

func init() { rt.Register(&Codex{}) }

// Name returns the runtime identifier.
func (c *Codex) Name() string { return "codex" }

// BuildCommand constructs the codex CLI command.
// Uses --json to emit JSONL (includes thread_id for session resumption).
// When SessionID is set, resumes that thread for conversation continuity.
func (c *Codex) BuildCommand(cfg rt.SpawnConfig) []string {
	cmd := []string{"codex", "exec", "--dangerously-bypass-approvals-and-sandbox", "--json"}

	if cfg.Model != "" {
		cmd = append(cmd, "--model", cfg.Model)
	}

	// Resume existing thread if we have a session ID
	if cfg.SessionID != "" {
		cmd = append(cmd, "resume", cfg.SessionID)
	}

	if cfg.Prompt != "" {
		cmd = append(cmd, cfg.Prompt)
	}

	return cmd
}

// InstallCommands returns shell commands to install Codex.
func (c *Codex) InstallCommands() []string {
	return []string{"npm install -g @openai/codex"}
}

// RequiredEnvVars returns env var names needed for auth.
func (c *Codex) RequiredEnvVars() []string { return []string{"OPENAI_API_KEY"} }

// OptionalEnvVars returns useful but not required env vars.
func (c *Codex) OptionalEnvVars() []string { return []string{"CODEX_API_KEY"} }

// BaseImage returns the Docker base image needed.
func (c *Codex) BaseImage() string { return "node:20" }

// SystemPackages returns apt packages needed beyond the base image.
func (c *Codex) SystemPackages() []string { return []string{"git", "curl"} }

// SupportsSession returns true if the runtime can resume sessions.
func (c *Codex) SupportsSession() bool { return true }
func (c *Codex) Available() bool       { return true }
