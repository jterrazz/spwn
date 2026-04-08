package codex

import (
	"spwn.sh/core/agent"
	rt "spwn.sh/core/universe/internal/runtime"
)

// Codex implements the Runtime interface for OpenAI Codex CLI.
type Codex struct{}

func init() { rt.Register(&Codex{}) }

// Name returns the runtime identifier.
func (c *Codex) Name() string { return "codex" }

// BuildCommand constructs the codex CLI command.
func (c *Codex) BuildCommand(cfg rt.SpawnConfig) []string {
	// NPC mode: no Mind, one-shot
	if cfg.MindPath == "" {
		cmd := []string{"codex", "exec", "--dangerously-bypass-approvals-and-sandbox"}
		if cfg.Model != "" {
			cmd = append(cmd, "--model", cfg.Model)
		}
		if cfg.Prompt != "" {
			cmd = append(cmd, cfg.Prompt)
		}
		return cmd
	}

	// Worker/Manager/Chief: session management
	sessID := agent.DeterministicSessionID(cfg.AgentName, cfg.WorldID)

	cmd := []string{"codex", "exec", "--dangerously-bypass-approvals-and-sandbox"}
	if cfg.Model != "" {
		cmd = append(cmd, "--model", cfg.Model)
	}

	// Check for existing session to resume
	existing, err := agent.LoadSession(cfg.MindPath, cfg.WorldID)
	if err == nil && existing != nil {
		cmd = append(cmd, "resume", "--session-id", sessID)
	}

	if cfg.Prompt != "" {
		cmd = append(cmd, cfg.Prompt)
	}

	_ = sessID // used for session tracking
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
