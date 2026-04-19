// Package worldbook holds spwn's opinionated world content — the
// runtime-neutral markdown blocks that every renderer inlines into
// the boot prompt it emits.
//
// The split between worldbook and the runtime-specific renderer
// under packages/runtimes/<name>/ is the split between content and
// layout:
//
//   - worldbook owns the WHAT: physics prose, the faculties briefing,
//     the roster format, the architect's identity + skills, and the
//     role-aware NPC context. Runtime-neutral strings.
//
//   - runtimes/<name>/render.go owns the WHERE (and HOW): whether
//     each string is inlined into a single CLAUDE.md or written to
//     separate files, which paths it lands at, and runtime-specific
//     conventions like Claude's `@-import` syntax.
//
// Consumers that need the content itself — the architect for NPC
// prompts and image-build, the renderer for compilation — import
// from worldbook and never reach into a runtime package for text.
// This keeps the content authored once and shared across every
// runtime that ships a renderer.
//
// When a future "spwn build --bare" mode lands (user content only,
// no spwn opinions), it will skip this package's output while still
// running through the runtime's layout step.
package worldbook
