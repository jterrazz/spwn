package pi

import rt "spwn.sh/core/universe/internal/runtime"

// Pi implements the Runtime interface for the Pi coding agent.
type Pi struct{}

func init() { rt.Register(&Pi{}) }

// Name returns the runtime identifier.
func (p *Pi) Name() string { return "pi" }

// BuildCommand constructs the pi CLI command.
func (p *Pi) BuildCommand(cfg rt.SpawnConfig) []string {
	if cfg.Prompt != "" {
		cmd := []string{"pi", cfg.Prompt, "--mode", "print"}
		if cfg.Model != "" {
			cmd = append(cmd, "--model", cfg.Model)
		}
		if cfg.MindPath == "" {
			cmd = append(cmd, "--no-session")
		}
		return cmd
	}
	// Interactive
	return []string{"pi"}
}

// InstallCommands returns shell commands to install Pi.
func (p *Pi) InstallCommands() []string {
	return []string{"npm install -g @mariozechner/pi-coding-agent"}
}

// RequiredEnvVars returns env var names needed for auth.
func (p *Pi) RequiredEnvVars() []string { return []string{} }

// OptionalEnvVars returns useful but not required env vars.
func (p *Pi) OptionalEnvVars() []string {
	return []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GEMINI_API_KEY", "OPENROUTER_API_KEY"}
}

// BaseImage returns the Docker base image needed.
func (p *Pi) BaseImage() string { return "node:20" }

// SystemPackages returns apt packages needed beyond the base image.
func (p *Pi) SystemPackages() []string { return []string{"git", "curl"} }

// SupportsSession returns true if the runtime can resume sessions.
func (p *Pi) SupportsSession() bool { return true }
func (p *Pi) Available() bool       { return false }
func (p *Pi) CredentialFiles() map[string]string { return nil }
