package hermes

import claw "spwn.sh/packages/universe/internal/claw"

// Hermes implements the Claw interface for Hermes.
type Hermes struct{}

func init() { claw.Register(&Hermes{}) }

// Name returns the claw identifier.
func (h *Hermes) Name() string { return "hermes" }

// StartCommand returns the command to start the claw daemon.
func (h *Hermes) StartCommand() []string {
	return []string{"hermes", "gateway"}
}

// StopCommand returns the command to stop the daemon.
func (h *Hermes) StopCommand() []string {
	return []string{"hermes", "gateway", "stop"}
}

// InstallCommands returns shell commands to install Hermes.
func (h *Hermes) InstallCommands() []string {
	return []string{
		"curl -fsSL https://raw.githubusercontent.com/NousResearch/hermes-agent/main/scripts/install.sh | bash",
	}
}

// RequiredEnvVars returns env vars needed for the daemon.
func (h *Hermes) RequiredEnvVars() []string { return []string{} }

// BaseImage returns the Docker base image.
func (h *Hermes) BaseImage() string { return "debian:bookworm" }

// SystemPackages returns apt packages needed.
func (h *Hermes) SystemPackages() []string { return []string{"python3", "nodejs", "git", "curl"} }
func (h *Hermes) Available() bool          { return true }
