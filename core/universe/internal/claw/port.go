package claw

// Claw defines the adapter interface for the God daemon.
type Claw interface {
	// Name returns the claw identifier.
	Name() string
	// StartCommand returns the command to start the claw daemon.
	StartCommand() []string
	// StopCommand returns the command to stop the daemon.
	StopCommand() []string
	// ConnectChannel returns the command to connect a messaging channel.
	ConnectChannel(channel string) []string
	// InstallCommands returns shell commands to install the claw.
	InstallCommands() []string
	// RequiredEnvVars returns env vars needed for the daemon.
	RequiredEnvVars() []string
	// SupportedChannels returns the list of messaging channels supported.
	SupportedChannels() []string
	// BaseImage returns the Docker base image.
	BaseImage() string
	// SystemPackages returns apt packages needed.
	SystemPackages() []string
}
