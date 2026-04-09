package aider

import rt "spwn.sh/core/universe/internal/runtime"

// Aider implements the Runtime interface for aider-chat.
type Aider struct{}

func init() { rt.Register(&Aider{}) }

// Name returns the runtime identifier.
func (a *Aider) Name() string { return "aider" }

// BuildCommand constructs the aider CLI command.
func (a *Aider) BuildCommand(cfg rt.SpawnConfig) []string {
	cmd := []string{"aider", "--yes-always", "--no-auto-commits", "--no-stream", "--message", cfg.Prompt}
	if cfg.Model != "" {
		cmd = append(cmd, "--model", cfg.Model)
	}
	return cmd
}

// InstallCommands returns shell commands to install aider.
func (a *Aider) InstallCommands() []string {
	return []string{"pip install --no-cache-dir aider-chat"}
}

// RequiredEnvVars returns env var names needed for auth.
func (a *Aider) RequiredEnvVars() []string { return []string{} }

// OptionalEnvVars returns useful but not required env vars.
func (a *Aider) OptionalEnvVars() []string {
	return []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GEMINI_API_KEY", "DEEPSEEK_API_KEY", "OPENROUTER_API_KEY"}
}

// BaseImage returns the Docker base image needed.
func (a *Aider) BaseImage() string { return "python:3.12-slim" }

// SystemPackages returns apt packages needed beyond the base image.
func (a *Aider) SystemPackages() []string { return []string{"git"} }

// SupportsSession returns true if the runtime can resume sessions.
func (a *Aider) SupportsSession() bool { return false }
func (a *Aider) Available() bool       { return false }
func (a *Aider) CredentialFiles() map[string]string { return nil }
