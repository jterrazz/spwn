package gemini

import rt "spwn.sh/core/universe/internal/runtime"

// Gemini implements the Runtime interface for Google Gemini CLI.
type Gemini struct{}

func init() { rt.Register(&Gemini{}) }

// Name returns the runtime identifier.
func (g *Gemini) Name() string { return "gemini" }

// BuildCommand constructs the gemini CLI command.
func (g *Gemini) BuildCommand(cfg rt.SpawnConfig) []string {
	cmd := []string{"gemini", "-p", cfg.Prompt, "--yolo"}
	if cfg.Model != "" {
		cmd = append(cmd, "-m", cfg.Model)
	}
	return cmd
}

// InstallCommands returns shell commands to install Gemini CLI.
func (g *Gemini) InstallCommands() []string {
	return []string{"npm install -g @google/gemini-cli"}
}

// RequiredEnvVars returns env var names needed for auth.
func (g *Gemini) RequiredEnvVars() []string { return []string{"GEMINI_API_KEY"} }

// OptionalEnvVars returns useful but not required env vars.
func (g *Gemini) OptionalEnvVars() []string { return []string{"GOOGLE_API_KEY"} }

// BaseImage returns the Docker base image needed.
func (g *Gemini) BaseImage() string { return "node:20" }

// SystemPackages returns apt packages needed beyond the base image.
func (g *Gemini) SystemPackages() []string { return []string{"git", "curl"} }

// SupportsSession returns true if the runtime can resume sessions.
func (g *Gemini) SupportsSession() bool { return false }
func (g *Gemini) Available() bool       { return false }
