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
//   - imports the per-deployment role via `@worlds/<id>/role.md`
//   - emits a "Your playbooks" section iff at least one playbook
//     carries valid frontmatter (name + description)
//   - spells out the spwn conventions (memory, messaging, knowledge)
//     that used to live scattered across four "system skills" and a
//     standalone world/AGENTS.md
//
// Tool-shipped skills aren't listed here — Claude Code auto-
// discovers them from `.claude/skills/` (symlinked at spawn time to
// /world/skills/ where CollectSkills baked them into the image).
func GenerateAgentCLAUDEMD(in AgentClaudeMDInput) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# %s — %s in world %q\n\n", in.AgentName, in.Role, in.WorldID)

	sb.WriteString("## Identity\n\n")
	sb.WriteString("Your soul is the source of truth for who you are. Read it first, every session.\n\n")
	sb.WriteString("@SOUL.md\n\n")

	sb.WriteString("## Physics\n\n")
	sb.WriteString(strings.TrimSpace(demoteHeadings(stripLeadingH1(in.Physics))))
	sb.WriteString("\n\n")

	sb.WriteString("## Faculties\n\n")
	sb.WriteString(strings.TrimSpace(demoteHeadings(stripLeadingH1(in.Faculties))))
	sb.WriteString("\n\n")

	sb.WriteString("## Roster\n\n")
	sb.WriteString(strings.TrimSpace(demoteHeadings(stripLeadingH1(in.Roster))))
	sb.WriteString("\n\n")

	sb.WriteString("## Role here\n\n")
	fmt.Fprintf(&sb, "@worlds/%s/role.md\n\n", in.WorldID)

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

// stripLeadingH1 drops a `# …` heading if the body starts with one,
// so inlined blocks don't double up with the "## Physics" / "##
// Faculties" / "## Roster" headings we wrap them in above.
func stripLeadingH1(body string) string {
	body = strings.TrimLeft(body, "\n")
	if !strings.HasPrefix(body, "# ") {
		return body
	}
	if idx := strings.Index(body, "\n"); idx != -1 {
		return strings.TrimLeft(body[idx+1:], "\n")
	}
	return ""
}

// demoteHeadings prefixes every markdown heading line with one more
// "#" so an inlined block whose top-level sections were H2s nests
// cleanly under the H2 wrapper we emit for it. Code fences (```)
// are skipped so shell examples that happen to contain # comments
// aren't mangled.
func demoteHeadings(body string) string {
	var out strings.Builder
	inFence := false
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			out.WriteString(line)
			out.WriteByte('\n')
			continue
		}
		if !inFence && strings.HasPrefix(line, "#") {
			out.WriteByte('#')
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	// strings.Split adds a trailing empty element for bodies ending
	// in \n; strip the extra newline we appended for that element.
	return strings.TrimRight(out.String(), "\n") + "\n"
}
