package runtimes

// SpawnConfig holds the configuration for a single agent spawn.
//
// Note: MindPath used to live here in the legacy file-mounted layout.
// In the labels-as-truth + per-agent HOME architecture the runtime
// adapter does not need a host path - talk.go sets HOME and -w on the
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

// Spawner is the spawn-time adapter port for a runtime. It covers
// everything a runtime does on the host/container boundary at spawn
// time: command building, credential sync, prelaunch shell setup,
// default-config materialisation, session-id handling. The image-
// build recipe lives separately as a tool.Tool (Adapter.Tool) and the
// source-to-Tree renderer lives separately as a transpile.Runtime
// (Adapter.Render).
type Spawner interface {
	Name() string
	BuildCommand(cfg SpawnConfig) []string
	SupportsSession() bool
	Available() bool

	DefaultConfigFiles(agentHome string) map[string][]byte

	SyncHostCredentials(credsDir string) error

	PrelaunchShell() string

	OneShotFlags(base []string, outputFormat string) []string

	ParseOneShotResult(raw []byte) (text string, sessionID string, err error)
}
