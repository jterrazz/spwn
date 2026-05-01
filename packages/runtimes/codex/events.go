package codex

// SupportedEvents enumerates the runtime hook events the Codex CLI
// understands today. Codex's hook system adopted Claude Code's
// event/matcher layering intentionally, so the names that overlap
// match byte-for-byte. The set is narrower than Claude's because
// codex hasn't shipped equivalents for compaction, sub-agents,
// notifications, or session-end yet.
//
// Authors targeting Codex who write a Claude-only event will get a
// `spwn check` warning rather than silent drift. Update this list
// when codex releases new events.
var SupportedEvents = []string{
	"PostToolUse",
	"PreToolUse",
	"SessionStart",
	"Stop",
	"UserPromptSubmit",
}
