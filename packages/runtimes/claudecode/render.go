package claudecode

import (
	"fmt"

	"spwn.sh/packages/transpile"
	"spwn.sh/packages/transpile/worldbook"
)

// renderer is the transpile.Runtime implementation for Claude Code.
//
// Claude Code reads a CLAUDE.md at the working directory on startup.
// This renderer is a thin LAYOUT adapter: the prose itself lives in
// packages/transpile/worldbook (physics, manual, skills, roster,
// architect identity — all runtime-neutral spwn content). Our job
// here is to place that content at Claude-specific paths and to emit
// the per-agent CLAUDE.md entrypoint with Claude's @-import syntax.
type renderer struct{}

// Renderer is the exported render adapter for Claude Code. It is
// bundled into the package-level Adapter (see adapter.go) which
// registers itself into both the runtimes registry and transpile's
// global renderer registry at init time.
var Renderer = &renderer{}

// Name returns "claude-code", the identifier used by
// transpile.Compile to look up this runtime.
func (r *renderer) Name() string { return "claude-code" }

// Render lays out worldbook content at Claude-specific paths. Paths
// it chooses:
//
//   - world/physics.md, world/faculties.md, world/AGENTS.md,
//     world/roster.md                          (runtime-neutral content)
//   - world/skills/*.md                        (runtime-neutral content)
//   - agents/<name>/CLAUDE.md                  (Claude-specific entrypoint)
//   - agents/<name>/worlds/<id>/role.md        (runtime-neutral content)
//
// The world/ paths happen to be the same ones codex would likely use
// if it grew a renderer — they're generic filesystem layout. The
// agents/<name>/CLAUDE.md filename is where Claude-specificity lives;
// another runtime would pick its own entrypoint name.
func (r *renderer) Render(input transpile.Input) (*transpile.Tree, error) {
	t := transpile.New()

	t.AddString("world/physics.md", worldbook.GeneratePhysics(input.Deps))
	t.AddString("world/faculties.md", worldbook.GenerateFaculties(input.VerifiedTools))
	t.AddString("world/AGENTS.md", worldbook.AgentsBook(input.WorldKnowledgeMounted))

	// Roster, if we have agents to put in it.
	roster := make([]worldbook.ColonyAgentSpec, 0, len(input.Agents))
	for _, a := range input.Agents {
		roster = append(roster, worldbook.ColonyAgentSpec{Name: a.Name, Role: a.Role})
	}
	t.AddString("world/roster.md", worldbook.GenerateRoster(input.WorldID, roster, input.WorldKnowledgeMounted))

	// System skills. The mind-management guide varies with the
	// knowledge-mount flag — when no knowledge dir is bound, the
	// "Saving Knowledge" section is dropped entirely.
	for name, body := range worldbook.SystemSkills(input.WorldKnowledgeMounted) {
		t.AddString("world/skills/"+name, body)
	}

	// Per-agent files. Source AGENTS.md -> target CLAUDE.md lives
	// here: this renderer is the single place that encodes "Claude
	// Code reads CLAUDE.md".
	for _, a := range input.Agents {
		role := a.Role
		if role == "" {
			role = "worker"
		}
		// role.md is per-deployment -- it describes what the agent
		// does in THIS world -- so it lives under worlds/<id>/. The
		// CLAUDE.md entrypoint lives at the agent root because
		// Claude Code loads the cwd's CLAUDE.md on startup and the
		// agent runs with cwd = /agents/<name>/.
		t.AddString(
			fmt.Sprintf("agents/%s/worlds/%s/role.md", a.Name, input.WorldID),
			fmt.Sprintf("# Role in %s\n\n%s\n", input.WorldID, role),
		)
		t.AddString(
			fmt.Sprintf("agents/%s/CLAUDE.md", a.Name),
			GenerateAgentCLAUDEMD(a.Name, role, input.WorldKnowledgeMounted),
		)
	}

	return t, nil
}
