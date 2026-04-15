package claudecode

import (
	"fmt"

	"spwn.sh/packages/compile"
)

// Runtime is the compile.Runtime implementation for Claude Code.
//
// Claude Code reads a CLAUDE.md at the working directory on startup.
// This runtime translates the provider-neutral spwn source format
// into that convention: each agent's source AGENT.md becomes an
// emitted CLAUDE.md inside the container's agent home.
type Runtime struct{}

func init() { compile.Register(&Runtime{}) }

// Name returns "claude-code", the identifier used by
// compile.Compile to look up this runtime.
func (r *Runtime) Name() string { return "claude-code" }

// Render translates the Input into a Tree laid out the way Claude
// Code expects. It produces:
//
//   - world/physics.md, world/faculties.md, world/AGENTS.md,
//     world/roster.md
//   - world/skills/*.md (system skills)
//   - agents/<name>/CLAUDE.md + agents/<name>/role.md for every agent
//
// This is the in-memory equivalent of what architect.Spawn used to
// write file-by-file.
func (r *Runtime) Render(input compile.Input) (*compile.Tree, error) {
	t := compile.New()

	// World-wide files. These happen to be rendered the same way for
	// every runtime today, but they still belong to a concrete
	// runtime until a second target forces the shared bits up into
	// a neutral sub-package.
	t.AddString("world/physics.md", GeneratePhysics(input.Manifest))
	t.AddString("world/faculties.md", GenerateFaculties(input.VerifiedTools))
	t.AddString("world/AGENTS.md", AgentsBook)

	// Roster, if we have agents to put in it.
	roster := make([]ColonyAgentSpec, 0, len(input.Agents))
	for _, a := range input.Agents {
		roster = append(roster, ColonyAgentSpec{Name: a.Name, Role: a.Role})
	}
	t.AddString("world/roster.md", GenerateRoster(input.WorldID, roster))

	// System skills.
	for name, body := range SystemSkills() {
		t.AddString("world/skills/"+name, body)
	}

	// Per-agent files. Source AGENT.md -> target CLAUDE.md lives
	// here: the claudecode renderer is the single place that
	// encodes "Claude Code reads CLAUDE.md".
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
			GenerateAgentCLAUDEMD(a.Name, role),
		)
	}

	return t, nil
}
