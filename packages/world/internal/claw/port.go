package claw

// Claw defines the adapter interface for the Architect daemon.
type Claw interface {
	// Name returns the claw identifier.
	Name() string
	// StartCommand returns the command to start the claw daemon.
	StartCommand() []string
	// StopCommand returns the command to stop the daemon.
	StopCommand() []string
	// InstallCommands returns shell commands to install the claw.
	InstallCommands() []string
	// RequiredEnvVars returns env vars needed for the daemon.
	RequiredEnvVars() []string
	// BaseImage returns the Docker base image.
	BaseImage() string
	// SystemPackages returns apt packages needed.
	SystemPackages() []string
	// Available returns true if the claw is production-ready.
	Available() bool
}
