package worldbook

import (
	"strings"

	)

// GeneratePhysics returns the world-physics markdown block. The
// claude-code renderer inlines it into each agent's CLAUDE.md under
// a "## Physics" heading. Callers that need the raw string (e.g.
// NPC prompts) use it directly; no separate /world/physics.md file
// is emitted by any active renderer.
func GeneratePhysics(_ []string) string {
	var sb strings.Builder

	sb.WriteString("# Physics of This World\n\n")

	// Laws
	sb.WriteString("## Laws\n")
	sb.WriteString("- Network: bridge (outbound access enabled)\n")
	sb.WriteString("- Filesystem is ephemeral except /workspaces and /mind\n\n")

	// Tools
	sb.WriteString("## Tools\n")
	sb.WriteString("/workspaces - project files, mounted from Host (read-write)\n")
	sb.WriteString("/mind - agent identity and memory (read-write)\n")
	sb.WriteString("/tmp - ephemeral scratch space\n\n")

	// Communication
	sb.WriteString("## Communication\n")
	sb.WriteString("Agents communicate via the inbox at /world/inbox/.\n")
	sb.WriteString("To send a message: write a JSON file to /world/inbox/{recipient}/.\n")
	sb.WriteString("To check messages: read files from /world/inbox/{your-name}/.\n\n")

	// Topology
	sb.WriteString("## Topology\n")
	sb.WriteString("/workspaces - project files, mounted from Host (read-write)\n")
	sb.WriteString("/mind - agent identity and memory (read-write)\n")
	sb.WriteString("/tmp - ephemeral scratch space\n")

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
