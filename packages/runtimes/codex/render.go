package codex

import (
	"fmt"

	"spwn.sh/packages/transpile"
	"spwn.sh/packages/transpile/worldbook"
)

// renderer is the transpile.Runtime implementation for codex.
//
// Codex reads `AGENTS.md` from the process cwd on startup and uses
// it as its system prompt. Unlike Claude Code, codex does not
// resolve `@path/to/file.md` imports — everything the runtime must
// see has to be inlined. This renderer emits one self-contained
// AGENTS.md per agent and a parallel worlds/<id>/role.md that exists
// for parity with claudecode (not referenced by codex today;
// future-proofing for when codex learns imports).
type renderer struct{}

// Renderer is the exported render adapter for codex. Bundled into
// the package-level Adapter (see adapter.go) which registers itself
// into both the runtimes registry and transpile's renderer registry
// at init time.
var Renderer = &renderer{}

// Name returns "codex", the identifier used by transpile.Compile to
// look up this runtime.
func (r *renderer) Name() string { return "codex" }

// Render lays out codex-specific output for each agent. Paths:
//
//   - agents/<name>/AGENTS.md              self-contained boot prompt
//   - agents/<name>/worlds/<id>/role.md    per-deployment role (parity)
//
// The AGENTS.md file inlines SOUL + physics + faculties + roster +
// playbooks + conventions + role + user's AGENTS.md body (if any).
// Everything that claude-code delivers via `@-imports` in CLAUDE.md
// arrives here as inlined markdown — codex's runtime contract
// doesn't resolve imports.
func (r *renderer) Render(input transpile.Input) (*transpile.Tree, error) {
	t := transpile.New()

	physics := worldbook.GeneratePhysics(input.Deps)
	faculties := worldbook.GenerateFaculties(input.VerifiedTools)

	roster := make([]worldbook.ColonyAgentSpec, 0, len(input.Agents))
	for _, a := range input.Agents {
		roster = append(roster, worldbook.ColonyAgentSpec{Name: a.Name, Role: a.Role})
	}
	rosterMD := worldbook.GenerateRoster(input.WorldID, roster, input.WorldKnowledgeMounted)

	for _, a := range input.Agents {
		role := a.Role
		if role == "" {
			role = "worker"
		}
		t.AddString(
			fmt.Sprintf("agents/%s/worlds/%s/role.md", a.Name, input.WorldID),
			fmt.Sprintf("# Role in %s\n\n%s\n", input.WorldID, role),
		)
		t.AddString(
			fmt.Sprintf("agents/%s/AGENTS.md", a.Name),
			GenerateAgentAgentsMD(AgentAgentsMDInput{
				AgentName:        a.Name,
				Role:             role,
				WorldID:          input.WorldID,
				Soul:             a.Soul,
				AgentMD:          a.AgentMD,
				Physics:          physics,
				Faculties:        faculties,
				Roster:           rosterMD,
				Playbooks:        a.Playbooks,
				KnowledgeMounted: input.WorldKnowledgeMounted,
			}),
		)
	}

	return t, nil
}
