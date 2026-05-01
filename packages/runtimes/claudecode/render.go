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
// world/skills/*.md indirection.
//
// Skills (both user-authored from spwn/skills/ and tool-shipped from
// resolved deps) land under each agent's `.claude/skills/<n>/` so
// Claude Code's native skill walker picks them up without a spawn-time
// symlink — the renderer is the single source of truth for the skill
// tree, not the image build step.
//
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

// SupportedHookEvents returns every hook event Claude Code recognises.
// `spwn check` cross-references each declared hook against this list
// to surface typos and Codex-only events that would silently no-op
// in a Claude session. See events.go for the canonical set.
func (r *renderer) SupportedHookEvents() []string {
	return append([]string(nil), SupportedEvents...)
}

// Render lays out Claude-specific output for each agent. Paths:
//
//   - agents/<name>/CLAUDE.md                   self-contained system prompt
//   - agents/<name>/.claude/settings.json       hooks + model + permissions
//   - agents/<name>/.claude/skills/<n>/SKILL.md every resolved skill (+ sidecar)
//
// Nothing lands under world/ or worlds/ — world-shared context
// (physics, faculties, roster) and the per-deployment role line are
// all inlined into each CLAUDE.md, so the runtime boots with every
// required context block already in the prompt. No @-import side
// files, no `worlds/<id>/role.md` indirection.
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
		// Skills: every skill gets its own directory under the agent's
		// `.claude/skills/` so Claude Code's native walker picks them
		// up. File ordering is deterministic because transpile.Tree
		// sorts by path at render time.
		for _, skill := range input.Skills {
			for relPath, body := range skill.Files {
				t.Add(
					fmt.Sprintf("agents/%s/.claude/skills/%s/%s", a.Name, skill.Name, relPath),
					body,
				)
			}
		}
		// Settings: always emitted so the renderer owns every key
		// Claude Code reads at startup (permissions prompt, hooks,
		// model pin). The spawn adapter's DefaultConfigFiles
		// intentionally does NOT emit this file — two writers would
		// race on the same path via docker cp.
		t.Add(
			fmt.Sprintf("agents/%s/.claude/settings.json", a.Name),
			GenerateAgentSettingsJSON(SettingsInput{
				Hooks: a.Hooks,
				Model: a.Model,
			}),
		)
	}

	return t, nil
}
