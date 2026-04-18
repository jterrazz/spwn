// Package worldbook holds spwn's opinionated world content — the
// default prose, skills, and identity material that spwn injects into
// every compiled world regardless of which runtime renders it.
//
// The split between worldbook and the runtime-specific renderer under
// packages/runtimes/<name>/ is the split between content and layout:
//
//   - worldbook owns the WHAT: physics.md prose, the agent operating
//     manual, the opinionated system skills (mind-management,
//     collaboration, world-awareness, self-evolution), the architect's
//     identity + skills, role-aware agent context, the roster format.
//     Runtime-neutral.
//
//   - runtimes/<name>/render.go owns the WHERE: which paths each
//     piece of content lands at, how the per-agent entrypoint file is
//     named (CLAUDE.md for claude-code, whatever codex wants, …), and
//     runtime-specific conventions like Claude's `@-import` syntax.
//
// Consumers that need the content itself — the architect for NPC
// prompts and image-build, the renderer for compilation — import from
// worldbook and never reach into a runtime package for text. This
// keeps the content authored once and shared across every runtime
// that ships a renderer.
//
// When a future "spwn build --bare" mode lands (user content only, no
// spwn opinions), it will skip this package's output while still
// running through the runtime's layout step.
package worldbook
