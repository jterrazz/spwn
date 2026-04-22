package worldbook

import (
	"strings"

	)

// GeneratePhysics returns the world-physics markdown block. The
// claude-code renderer inlines it into each agent's CLAUDE.md under
// a "## Physics" heading. Callers that need the raw string (e.g.
// NPC prompts) use it directly; no separate /world/physics.md file
// is emitted by any active renderer.
//
// knowledgeMounted controls whether the Topology line advertises
// /world/knowledge/ as a readable path. When a world has no
// `knowledge:` key in spwn.yaml the directory is absent from the
// container, so mentioning it would mislead the agent.
func GeneratePhysics(knowledgeMounted bool) string {
	var sb strings.Builder

	sb.WriteString("# Physics of This World\n\n")

	// Laws
	sb.WriteString("## Laws\n")
	sb.WriteString("- Network: bridge (outbound access enabled)\n")
	sb.WriteString("- Filesystem is ephemeral except /workspaces and /agents\n\n")

	// Topology — where the agent can read and write.
	sb.WriteString("## Topology\n")
	sb.WriteString("/agents/<your-name>/ - your home: SOUL.md, playbooks/, journal/ (read-write, persists across worlds)\n")
	sb.WriteString("/workspaces/         - host project dirs mounted read-write, your actual work surface\n")
	if knowledgeMounted {
		sb.WriteString("/world/              - world-shared state: knowledge/, inbox/<name>/ (read-write)\n")
	} else {
		sb.WriteString("/world/              - world-shared state: inbox/<name>/ (read-write)\n")
	}
	sb.WriteString("/tmp                 - ephemeral scratch\n\n")

	// Communication
	sb.WriteString("## Communication\n")
	sb.WriteString("Agents communicate via the inbox at /world/inbox/.\n")
	sb.WriteString("To send a message: write a file to /world/inbox/{recipient}/.\n")
	sb.WriteString("To check your inbox: read files from /world/inbox/{your-name}/.\n")

	return sb.String()
}

// GenerateFaculties returns the world-faculties markdown block
// (installed tools). Inlined by the claude-code renderer under
// "## Faculties" in each agent's CLAUDE.md.
func GenerateFaculties(verifiedTools []string) string {
	var sb strings.Builder

	sb.WriteString("# Faculties\n\n")

	sb.WriteString("## Tools\n")
	if len(verifiedTools) > 0 {
		sb.WriteString(strings.Join(verifiedTools, ", "))
		sb.WriteString("\n")
	} else {
		sb.WriteString("(none verified)\n")
	}

	return sb.String()
}
