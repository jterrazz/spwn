package openclaw

import claw "spwn.sh/core/universe/internal/claw"

// OpenClaw implements the Claw interface for OpenClaw.
type OpenClaw struct{}

func init() { claw.Register(&OpenClaw{}) }

// Name returns the claw identifier.
func (o *OpenClaw) Name() string { return "openclaw" }

// StartCommand returns the command to start the claw daemon.
func (o *OpenClaw) StartCommand() []string {
	return []string{"openclaw", "start"}
}

// StopCommand returns the command to stop the daemon.
func (o *OpenClaw) StopCommand() []string {
	return []string{"openclaw", "stop"}
}

// ConnectChannel returns the command to connect a messaging channel.
func (o *OpenClaw) ConnectChannel(channel string) []string {
	return []string{"openclaw", "channel", "connect", channel}
}

// InstallCommands returns shell commands to install OpenClaw.
func (o *OpenClaw) InstallCommands() []string {
	return []string{"npm install -g openclaw"}
}

// RequiredEnvVars returns env vars needed for the daemon.
func (o *OpenClaw) RequiredEnvVars() []string { return []string{"OPENCLAW_GATEWAY_TOKEN"} }

// SupportedChannels returns the list of messaging channels supported.
func (o *OpenClaw) SupportedChannels() []string {
	return []string{"telegram", "slack", "discord", "whatsapp", "signal", "matrix", "email", "sms"}
}

// BaseImage returns the Docker base image.
func (o *OpenClaw) BaseImage() string { return "node:20" }

// SystemPackages returns apt packages needed.
func (o *OpenClaw) SystemPackages() []string { return []string{"git", "curl"} }
