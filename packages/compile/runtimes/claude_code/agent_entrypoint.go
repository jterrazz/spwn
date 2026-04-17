package claudecode

import "fmt"

// GenerateAgentCLAUDEMD returns the per-agent CLAUDE.md file that
// Claude Code loads on startup. It inlines a reference to the
// agent's identity and the world's shared manuals so the runtime
// always boots with the agent's identity in context.
//
// This is Claude Code specific on purpose: the filename, the use of
// @-imports for identity loading, and the paths under /world/ are all
// conventions that belong to Claude Code, not to spwn's source
// format. A future Codex renderer will produce its own entrypoint
// file under its own name.
func GenerateAgentCLAUDEMD(agentName, role string) string {
	return fmt.Sprintf(`# %s

You are **%s**, a spwn agent with role: %s.

## Your identity

Read your full identity and behavioral instructions from:

@identity/profile.md

Follow the voice, style, and purpose defined there. You are NOT a generic assistant - you are %s.

## Your world

- Read %s for your operating manual (how memory, skills, and communication work).
- Read %s for the rules of this world (network, filesystem, communication).
- Read %s to see what tools are physically available.
- Read %s for system skills (mind management, collaboration, evolution).

## Key rules

1. **Read your identity first** before doing anything else. Your identity shapes how you respond.
2. Save important discoveries about the project or its domain to the world's knowledge base (write to %s). It's committed per-world and shared with every other agent in this world.
3. After significant work, check if a playbook should be created in %s.
4. **Messaging**: to send a message to another agent, write a .json or .md file to %s. To check YOUR inbox, read %s. Read %s for the full messaging protocol.
5. Never modify /world/physics.md, /world/faculties.md, or /world/AGENTS.md — they are read-only system context. /world/knowledge/ is writable.
`, agentName, agentName, role, agentName,
		"`/world/AGENTS.md`",
		"`/world/physics.md`",
		"`/world/faculties.md`",
		"`/world/skills/`",
		"`/world/knowledge/`",
		"`./playbooks/`",
		"`/world/inbox/<their-name>/`",
		fmt.Sprintf("`/world/inbox/%s/`", agentName),
		"`/world/skills/collaboration.md`")
}
