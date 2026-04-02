package zeroclaw

import claw "spwn.sh/core/universe/internal/claw"

// ZeroClaw implements the Claw interface for ZeroClaw.
type ZeroClaw struct{}

func init() { claw.Register(&ZeroClaw{}) }

// Name returns the claw identifier.
func (z *ZeroClaw) Name() string { return "zeroclaw" }

// StartCommand returns the command to start the claw daemon.
func (z *ZeroClaw) StartCommand() []string {
	return []string{"zeroclaw", "daemon"}
}

// StopCommand returns the command to stop the daemon.
func (z *ZeroClaw) StopCommand() []string {
	return []string{"zeroclaw", "daemon", "stop"}
}

// InstallCommands returns shell commands to install ZeroClaw.
func (z *ZeroClaw) InstallCommands() []string {
	return []string{
		"curl -fsSL https://zeroclaws.io/install.sh | bash",
	}
}

// RequiredEnvVars returns env vars needed for the daemon.
func (z *ZeroClaw) RequiredEnvVars() []string { return []string{} }

// BaseImage returns the Docker base image.
func (z *ZeroClaw) BaseImage() string { return "debian:bookworm-slim" }

// SystemPackages returns apt packages needed.
func (z *ZeroClaw) SystemPackages() []string { return []string{"ca-certificates", "curl"} }
func (z *ZeroClaw) Available() bool          { return false }
