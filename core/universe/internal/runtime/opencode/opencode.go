package opencode

import rt "spwn.sh/core/universe/internal/runtime"

// OpenCode implements the Runtime interface for OpenCode.
type OpenCode struct{}

func init() { rt.Register(&OpenCode{}) }

// Name returns the runtime identifier.
func (o *OpenCode) Name() string { return "opencode" }

// BuildCommand constructs the opencode CLI command.
func (o *OpenCode) BuildCommand(cfg rt.SpawnConfig) []string {
	cmd := []string{"opencode", "run", cfg.Prompt}
	if cfg.Resume {
		cmd = append(cmd, "--continue")
	}
	if cfg.Model != "" {
		cmd = append(cmd, "--model", cfg.Model)
	}
	return cmd
}

// InstallCommands returns shell commands to install OpenCode.
func (o *OpenCode) InstallCommands() []string {
	return []string{
		"curl -fsSL https://raw.githubusercontent.com/opencode-ai/opencode/refs/heads/main/install | bash",
	}
}

// RequiredEnvVars returns env var names needed for auth.
func (o *OpenCode) RequiredEnvVars() []string { return []string{} }

// OptionalEnvVars returns useful but not required env vars.
func (o *OpenCode) OptionalEnvVars() []string {
	return []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GEMINI_API_KEY", "GROQ_API_KEY"}
}

// BaseImage returns the Docker base image needed.
func (o *OpenCode) BaseImage() string { return "debian:bookworm-slim" }

// SystemPackages returns apt packages needed beyond the base image.
func (o *OpenCode) SystemPackages() []string { return []string{"git", "curl", "ca-certificates"} }

// SupportsSession returns true if the runtime can resume sessions.
func (o *OpenCode) SupportsSession() bool { return true }
func (o *OpenCode) Available() bool       { return false }
func (o *OpenCode) CredentialFiles() map[string]string { return nil }
