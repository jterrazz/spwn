// Package runtimes collects the built-in agent runtimes spwn ships
// with.
//
// A runtime is the thinking engine that executes an agent's
// prompts — Claude Code today, Codex on the way, Aider planned.
// Unlike tools and skills (which ship as YAML in the catalog),
// runtimes carry non-trivial spawn-time Go logic (credential sync,
// default-config materialisation, prelaunch shell, authentication
// flows) so they're kept in Go.
//
// The package aggregator (runtimes.go) exposes All and
// RegisterDefaults. Each runtime lives in its own sub-package
// (claude_code, codex) with a Tool singleton (the compile.Tool side)
// and, when needed, a Runtime (the spawn-time adapter registered
// with world/runtime).
package runtimes
