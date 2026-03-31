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

// ConnectChannel returns the command to connect a messaging channel.
func (z *ZeroClaw) ConnectChannel(channel string) []string {
	return []string{"zeroclaw", "channel", "add", channel}
}

// InstallCommands returns shell commands to install ZeroClaw.
func (z *ZeroClaw) InstallCommands() []string {
	return []string{
		"curl -fsSL https://zeroclaws.io/install.sh | bash",
	}
}

// RequiredEnvVars returns env vars needed for the daemon.
func (z *ZeroClaw) RequiredEnvVars() []string { return []string{} }

// SupportedChannels returns the list of messaging channels supported.
func (z *ZeroClaw) SupportedChannels() []string {
	return []string{"telegram", "slack", "discord", "whatsapp", "signal", "matrix", "irc", "email"}
}

// BaseImage returns the Docker base image.
func (z *ZeroClaw) BaseImage() string { return "debian:bookworm-slim" }

// SystemPackages returns apt packages needed.
func (z *ZeroClaw) SystemPackages() []string { return []string{"ca-certificates", "curl"} }
