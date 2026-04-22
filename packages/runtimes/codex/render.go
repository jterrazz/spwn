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
//   - agents/<name>/AGENTS.md                  self-contained boot prompt
//   - agents/<name>/.codex/config.toml         profile pin + hook feature flag
//   - agents/<name>/.codex/hooks.json          hook definitions (iff any)
//   - agents/<name>/.agents/skills/<n>/SKILL.md every resolved skill
//
// The AGENTS.md file inlines SOUL + physics + faculties + roster +
// playbooks + conventions + role + user's AGENTS.md body (if any).
// Nothing lands under worlds/ — codex doesn't resolve imports and
// the per-deployment role line is inlined into AGENTS.md directly.
func (r *renderer) Render(input transpile.Input) (*transpile.Tree, error) {
	t := transpile.New()

	physics := worldbook.GeneratePhysics(input.WorldKnowledgeMounted)
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
		// Skills land under `.agents/skills/<n>/SKILL.md` (not
		// `.codex/skills/` — codex follows the cross-vendor AGENTS.md
		// ecosystem convention where skills live at `.agents/skills/`).
		for _, skill := range input.Skills {
			for relPath, body := range skill.Files {
				t.Add(
					fmt.Sprintf("agents/%s/.agents/skills/%s/%s", a.Name, skill.Name, relPath),
					body,
				)
			}
		}
		// Config: always emit the project-local config.toml so codex
		// picks up spwn-owned flags without user intervention. Trust
		// for this directory is seeded by the spawn adapter's
		// PrelaunchShell via the user-level ~/.codex/config.toml.
		t.Add(
			fmt.Sprintf("agents/%s/.codex/config.toml", a.Name),
			GenerateAgentConfigTOML(ConfigInput{
				AgentName: a.Name,
				Model:     a.Model,
				HasHooks:  len(input.Hooks) > 0,
			}),
		)
		if body := GenerateAgentHooksJSON(input.Hooks); body != nil {
			t.Add(
				fmt.Sprintf("agents/%s/.codex/hooks.json", a.Name),
				body,
			)
		}
	}

	return t, nil
}
