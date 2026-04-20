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

	// OneShotFlags returns base with runtime-specific flags appended
	// that tell the CLI to execute a single prompt in non-interactive
	// mode and emit its output in the requested format.
	//
	//   outputFormat == "stream-json" — runtime emits one JSONL event
	//     per turn as it happens; caller streams each line verbatim.
	//   anything else (including "") — runtime emits a single JSON
	//     envelope at end-of-run; caller parses it via ParseOneShotResult.
	//
	// Called by the `spwn agent talk <name> <message>` path after
	// BuildCommand, so `base` already includes the runtime binary,
	// session identifier, and prompt argv. Runtimes that have no
	// output-format flag return base unchanged; runtimes whose output
	// is always parseable JSON by default can ignore outputFormat.
	//
	// The interactive path (no message) does not call this — every
	// flag here is a one-shot concern.
	OneShotFlags(base []string, outputFormat string) []string

	// ParseOneShotResult extracts the assistant's response text and
	// the runtime's session/thread identifier from one complete
	// non-streaming one-shot invocation's stdout bytes.
	//
	// Returns a non-nil error when the bytes don't match any known
	// envelope shape for this runtime — callers then fall back to
	// printing the raw output + scanning for an embedded session-id
	// via extractSessionID so users never lose conversation
	// continuity on a parser miss. A blank sessionID is valid for
	// runtimes that don't surface one; a blank text is valid for a
	// prompt that produced no assistant reply.
	ParseOneShotResult(raw []byte) (text string, sessionID string, err error)
}
