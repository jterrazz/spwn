package codex

import (
	"fmt"
	"strings"

	"spwn.sh/packages/transpile"
)

// AgentAgentsMDInput carries everything the codex renderer needs to
// build the per-agent AGENTS.md. Parallel to
// claudecode.AgentClaudeMDInput but with bodies where claude uses
// `@-imports` — codex doesn't resolve markdown imports at startup,
// so everything that must reach the runtime has to be inlined.
//
// Soul and AgentMD may be empty; the renderer handles each omission
// gracefully (just skips the corresponding section).
type AgentAgentsMDInput struct {
	AgentName        string
	Role             string
	WorldID          string
	Soul             []byte
	AgentMD          []byte
	Physics          string
	Faculties        string
	Roster           string
	Playbooks        []transpile.PlaybookEntry
	KnowledgeMounted bool
}

// GenerateAgentAgentsMD returns the per-agent AGENTS.md — the file
// codex reads at cwd on startup. It inlines:
//
//   - Identity: the body of SOUL.md (if any), under an "## Identity"
//     heading.
//   - Physics / Faculties / Roster: the worldbook blocks, demoted to
//     nest under this file's H2s.
//   - Role here: one line summarising the world deployment role.
//   - Your playbooks: auto-index of frontmatter-promoted playbooks,
//     omitted when empty.
//   - Conventions: the same numbered memory/messaging/evolution rules
//     the claudecode renderer emits, adapted to reference AGENTS.md /
//     playbooks/ paths.
//   - Task (if any): the user-authored AGENTS.md body, appended
//     verbatim so domain-specific instructions survive rendering.
//
// Codex's AGENTS.md convention collides with the USER-AUTHORED
// source file name: the user writes `spwn/agents/<name>/AGENTS.md`
// as the provider-neutral seed, and this renderer's output also lands
// at `/agents/<name>/AGENTS.md` inside the container. That's by
// design — codex reads from cwd and the source body shows up as
// "## Task" at the bottom of the final file.
func GenerateAgentAgentsMD(in AgentAgentsMDInput) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# %s — %s in world %q\n\n", in.AgentName, in.Role, in.WorldID)

	if len(bytesTrim(in.Soul)) > 0 {
		sb.WriteString("## Identity\n\n")
		sb.WriteString("Your soul — who you are. Re-read this when you feel lost.\n\n")
		sb.WriteString(strings.TrimSpace(transpile.DemoteHeadings(transpile.StripLeadingH1(string(in.Soul)))))
		sb.WriteString("\n\n")
	}

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

	if len(bytesTrim(in.AgentMD)) > 0 {
		sb.WriteString("## Task\n\n")
		sb.WriteString(strings.TrimSpace(string(in.AgentMD)))
		sb.WriteString("\n")
	}

	return sb.String()
}

// conventionsSection mirrors claudecode's conventionsSection but
// references codex-native paths and entry-file conventions. Kept
// line-for-line parallel with the claude version so content authors
// don't have to learn two sets of boilerplate; one edit in one place
// keeps the story consistent.
func conventionsSection(in AgentAgentsMDInput) string {
	var sb strings.Builder
	sb.WriteString("1. **Read your identity first** every session. It shapes your voice, values, and priorities.\n")
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

// bytesTrim returns b with leading/trailing whitespace trimmed.
// Used to treat a body of only whitespace as "effectively empty" so
// we don't emit an empty "## Identity" heading when SOUL.md is a
// blank file.
func bytesTrim(b []byte) []byte {
	s := strings.TrimSpace(string(b))
	if s == "" {
		return nil
	}
	return []byte(s)
}
