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
	// Name returns the runtime identifier (e.g. "claude-code").
	Name() string
	// BuildCommand returns the CLI command + args to execute inside the container.
	BuildCommand(cfg SpawnConfig) []string
	// SupportsSession returns true if the runtime can resume sessions.
	SupportsSession() bool
	// Available returns true if the runtime is production-ready.
	Available() bool

	// DefaultConfigFiles returns files that the runtime provider
	// wants written into the agent's HOME (/agents/<name>/) at
	// spawn time. Used to pre-dismiss first-run UI prompts -
	// onboarding banners, trust dialogs, terminal preferences - so
	// the user drops straight into a working session on
	// `spwn agent <name>`. Path keys are relative to HOME. Return
	// nil when the runtime has no such setup.
	DefaultConfigFiles(agentHome string) map[string][]byte

	// SyncHostCredentials copies host-side auth state into credsDir
	// (the directory bind-mounted at /credentials/ inside every
	// container). Called before every exec - the first call
	// bootstraps, subsequent calls refresh. Runtimes should fail
	// silently when no auth source is available (env vars may still
	// work as a fallback). Return an error only on true I/O or
	// command-execution failures.
	SyncHostCredentials(credsDir string) error

	// PrelaunchShell returns a shell snippet that runs immediately
	// before the runtime command inside the container. Typical
	// uses: source /credentials/.env, symlink credential files into
	// their expected paths on the agent home, set runtime-specific
	// env vars. An empty string means no wrapping is needed.
	PrelaunchShell() string
}
