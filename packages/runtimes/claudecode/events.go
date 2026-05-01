package claudecode

// SupportedEvents enumerates the runtime hook events Claude Code
// understands today. Hooks declared with any other event name are
// silently dropped by the runtime at firing time, so spwn surfaces
// the mismatch as a `spwn check` warning instead of letting the user
// believe their hook is wired.
//
// The list mirrors the public Claude Code hook reference. Keep it
// in alphabetical order so the validator's "did you mean X?" hint
// renders predictably.
var SupportedEvents = []string{
	"Notification",
	"PostToolUse",
	"PreCompact",
	"PreToolUse",
	"SessionEnd",
	"SessionStart",
	"Stop",
	"SubagentStop",
	"UserPromptSubmit",
}
