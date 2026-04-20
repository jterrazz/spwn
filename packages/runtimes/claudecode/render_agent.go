package claudecode

import (
	"fmt"
	"strings"

	"spwn.sh/packages/transpile"
)

// AgentClaudeMDInput carries everything the renderer needs to write
// the per-agent CLAUDE.md. World-shared bodies (physics, faculties,
// roster) are already rendered as markdown strings and get inlined
// verbatim into the output.
type AgentClaudeMDInput struct {
	AgentName        string
	Role             string
	WorldID          string
	Physics          string
	Faculties        string
	Roster           string
	Playbooks        []transpile.PlaybookEntry
	KnowledgeMounted bool
}

// GenerateAgentCLAUDEMD returns the per-agent CLAUDE.md — the file
// Claude Code loads on startup. It:
//
//   - imports the agent's identity via `@SOUL.md`
//   - inlines the world-shared physics, faculties, and roster bodies
//     so the agent never has to go read separate files
//   - inlines the one-line "Role here" directly (no separate role.md)
//   - emits a "Your playbooks" section iff at least one playbook
//     carries valid frontmatter (name + description)
//   - spells out the spwn conventions (memory, messaging, knowledge)
//     that used to live scattered across four "system skills" and a
//     standalone world/AGENTS.md
//
// Tool-shipped skills land in `.claude/skills/<n>/SKILL.md` at render
// time (see render.go) — Claude Code's native walker picks them up
// with no boot-time indirection.
func GenerateAgentCLAUDEMD(in AgentClaudeMDInput) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# %s — %s in world %q\n\n", in.AgentName, in.Role, in.WorldID)

	sb.WriteString("## Identity\n\n")
	sb.WriteString("Your soul is the source of truth for who you are. Read it first, every session.\n\n")
	sb.WriteString("@SOUL.md\n\n")

	sb.WriteString("## Physics\n\n")
	sb.WriteString(strings.TrimSpace(transpile.DemoteHeadings(transpile.StripLeadingH1(in.Physics))))
	sb.WriteString("\n\n")

	sb.WriteString("## Faculties\n\n")
	sb.WriteString(strings.TrimSpace(transpile.DemoteHeadings(transpile.StripLeadingH1(in.Faculties))))
	sb.WriteString("\n\n")

	sb.WriteString("## Roster\n\n")
	sb.WriteString(strings.TrimSpace(transpile.DemoteHeadings(transpile.StripLeadingH1(in.Roster))))
	sb.WriteString("\n\n")

	sb.WriteString("## Role here\n\n")
	fmt.Fprintf(&sb, "You are deployed as a %s in %s.\n\n", in.Role, in.WorldID)

	if len(in.Playbooks) > 0 {
		sb.WriteString("## Your playbooks\n\n")
		for _, p := range in.Playbooks {
			fmt.Fprintf(&sb, "- **%s** — %s\n", p.Name, p.Description)
		}
		sb.WriteString("\n_Read the full procedure at `./playbooks/<name>.md`. Promote a procedure by adding a `name:` + `description:` YAML header — that's how it appears in this list._\n\n")
	}

	sb.WriteString("## Conventions\n\n")
	sb.WriteString(conventionsSection(in))

	return sb.String()
}

// conventionsSection folds the content of the four retired
// /world/skills/*.md files into one block of rules. Content is
// preserved in spirit; wording is tightened.
func conventionsSection(in AgentClaudeMDInput) string {
	var sb strings.Builder

	sb.WriteString("1. **Read your soul first** every session. It shapes your voice, values, and priorities.\n")
	sb.WriteString("2. **Mind lives at `/agents/" + in.AgentName + "/`**:\n")
	sb.WriteString("   - `SOUL.md` — who you are (edit freely to grow).\n")
	sb.WriteString("   - `playbooks/` — reusable procedures. Add a `name:`/`description:` header to any playbook to have it indexed in this prompt as a shortcut.\n")
	sb.WriteString("   - `journal/` — session history; auto-appended by the system.\n")
	sb.WriteString("3. **Messaging** — send with `/world/inbox/<their-name>/<timestamp>-from-" + in.AgentName + ".md`; check yours at `/world/inbox/" + in.AgentName + "/`.\n")
	if in.KnowledgeMounted {
		sb.WriteString("4. **World knowledge** — save durable facts about this project or its domain to `/world/knowledge/`. Committed to the project; every agent in this world sees it.\n")
		sb.WriteString("5. **Evolve** — when asked to dream, analyze the journal and promote recurring patterns to playbooks.\n")
	} else {
		sb.WriteString("4. **Evolve** — when asked to dream, analyze the journal and promote recurring patterns to playbooks.\n")
	}
	sb.WriteString("\n")

	return sb.String()
}

// Markdown-shape helpers — `stripLeadingH1` and `demoteHeadings` —
// used to live here; they moved to `packages/transpile/mdfmt.go` when
// codex became a first-class renderer that needed the same inlining
// primitives. Refer to `transpile.StripLeadingH1` and
// `transpile.DemoteHeadings`.
