package runtime

// SpawnConfig holds the configuration for a single agent spawn.
//
// Note: MindPath used to live here in the legacy file-mounted layout.
// In the labels-as-truth + per-agent HOME architecture the runtime
// adapter does not need a host path — talk.go sets HOME and -w on the
// docker exec. Adapters that need to distinguish "named agent" from
// "anonymous NPC" should check AgentName != "".
type SpawnConfig struct {
	Prompt     string
	SessionID  string
	Resume     bool
	Model      string
	Provider   string
	AgentName  string
	WorldID    string
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
	// CredentialFiles returns a map of paths to place in the credentials
	// directory. Key = relative path inside credentials dir, value = host
	// source path. Return nil if the runtime uses only env vars from .env.
	CredentialFiles() map[string]string
	// SupportsSession returns true if the runtime can resume sessions.
	SupportsSession() bool
	// Available returns true if the runtime is production-ready.
	Available() bool
}
