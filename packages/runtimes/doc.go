// Package runtimes collects the built-in agent runtimes spwn ships
// with.
//
// A runtime is the thinking engine that executes an agent's prompts —
// Claude Code today, Codex for host-side setup, more to come. Unlike
// tools (which ship as YAML in the catalog), runtimes carry non-trivial
// host-side Go logic (credential sync, default-config materialisation,
// prelaunch shell, session-id parsing, container-side settings paths)
// so they're kept in Go.
//
// A runtime has up to three orthogonal facets, bundled in Adapter:
//
//   - Tool:   install recipe (tool.Tool — apt/curl/npm, user config)
//   - Render: source → Tree renderer (transpile.Runtime)
//   - Spawn:  host-side spawn-time adapter (Spawner)
//
// Each facet is optional. claudecode ships all three; codex ships
// Tool + Spawn. Future YAML-first runtimes could ship Tool only.
//
// This top-level package does NOT import subpackages — that would
// cycle back through each subpackage's own import of runtimes. Each
// runtime subpackage self-registers its Adapter via init(), so
// binaries opt in by blank-importing either an individual subpackage
// (apps/cli, tests) or the convenience aggregator runtimes/defaults
// (the production path).
//
// Spwn's opinionated world content (physics, faculties, roster,
// architect identity) lives in packages/transpile/worldbook — not
// here. Runtime renderers read from worldbook for the prose and
// decide how to inline or lay it out on disk.
package runtimes
