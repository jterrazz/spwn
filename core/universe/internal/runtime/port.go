package runtime

// SpawnConfig holds the configuration for a single agent spawn.
type SpawnConfig struct {
	Prompt     string
	SessionID  string
	Resume     bool
	Model      string
	Provider   string
	MindPath   string
	AgentName  string
	UniverseID string
	ExtraFlags []string
}

// Runtime defines the adapter interface for any agent backend.
type Runtime interface {
	// Name returns the runtime identifier.
	Name() string
	// BuildCommand returns the CLI command + args to execute inside the container.
	BuildCommand(cfg SpawnConfig) []string
	// InstallCommands returns shell commands to install the runtime in a Dockerfile.
	InstallCommands() []string
	// RequiredEnvVars returns env var names needed for auth.
	RequiredEnvVars() []string
	// OptionalEnvVars returns useful but not required env vars.
	OptionalEnvVars() []string
	// BaseImage returns the Docker base image needed.
	BaseImage() string
	// SystemPackages returns apt packages needed beyond the base image.
	SystemPackages() []string
	// SupportsSession returns true if the runtime can resume sessions.
	SupportsSession() bool
}
