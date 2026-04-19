package claudecode

import (
	"fmt"

	"spwn.sh/packages/transpile"
	"spwn.sh/packages/transpile/worldbook"
)

// renderer is the transpile.Runtime implementation for Claude Code.
//
// Claude Code reads a CLAUDE.md at the working directory on startup.
// This renderer emits one per agent, inlining every world-shared
// context block (physics, faculties, roster, conventions) directly
// into that file so the agent's system prompt is self-contained —
// no separate world/AGENTS.md, no world/physics.md, no
// world/skills/*.md indirection. Tool-shipped skills keep living at
// /world/skills/<tool>/SKILL.md (baked into the image via the
// dependency resolver's CollectSkills) and surface to Claude Code
// through a spawn-time symlink at /agents/<name>/.claude/skills.
// Playbook-index entries come from AgentInput.Playbooks —
// frontmatter-promoted playbooks the agent has authored.
type renderer struct{}

// Renderer is the exported render adapter for Claude Code. It is
// bundled into the package-level Adapter (see adapter.go) which
// registers itself into both the runtimes registry and transpile's
// global renderer registry at init time.
var Renderer = &renderer{}

// Name returns "claude-code", the identifier used by
// transpile.Compile to look up this runtime.
func (r *renderer) Name() string { return "claude-code" }

// Render lays out Claude-specific output for each agent. Paths:
//
//   - agents/<name>/CLAUDE.md              self-contained system prompt
//   - agents/<name>/worlds/<id>/role.md    per-deployment role
//
// Nothing lands under world/ — the world-shared context (physics,
// faculties, roster) is inlined into every agent's CLAUDE.md so the
// runtime boots with all of it already in the prompt.
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
			fmt.Sprintf("agents/%s/CLAUDE.md", a.Name),
			GenerateAgentCLAUDEMD(AgentClaudeMDInput{
				AgentName:        a.Name,
				Role:             role,
				WorldID:          input.WorldID,
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
