package physics

import (
	"fmt"
	"strings"

	"spwn.sh/core/gate"
	"spwn.sh/core/universe/internal/models"
)

// GeneratePhysics returns the contents of /universe/physics.md.
func GeneratePhysics(m models.Manifest) string {
	var sb strings.Builder

	sb.WriteString("# Physics of This Universe\n\n")

	// Constants
	sb.WriteString("## Constants\n")
	sb.WriteString(fmt.Sprintf("CPU: %d core(s) | Memory: %s | Disk: %s | Timeout: %s\n\n",
		m.Physics.Constants.CPU,
		m.Physics.Constants.Memory,
		m.Physics.Constants.Disk,
		m.Physics.Constants.Timeout,
	))

	// Laws
	sb.WriteString("## Laws\n")
	sb.WriteString("- Network: bridge (outbound access enabled)\n")
	sb.WriteString("- Filesystem is ephemeral except /workspace and /mind\n\n")

	// Tools
	sb.WriteString("## Tools\n")
	sb.WriteString("/workspace — project files, mounted from Host (read-write)\n")
	sb.WriteString("/mind — agent identity and memory (read-write)\n")
	sb.WriteString("/tmp — ephemeral scratch space\n\n")

	// Communication
	sb.WriteString("## Communication\n")
	sb.WriteString("Agents communicate via the inbox at /world/inbox/.\n")
	sb.WriteString("To send a message: write a JSON file to /world/inbox/{recipient}/.\n")
	sb.WriteString("To check messages: read files from /world/inbox/{your-name}/.\n\n")

	// Topology
	sb.WriteString("## Topology\n")
	sb.WriteString("/workspace — project files, mounted from Host (read-write)\n")
	sb.WriteString("/mind — agent identity and memory (read-write)\n")
	sb.WriteString("/tmp — ephemeral scratch space\n")

	return sb.String()
}

// GenerateFaculties returns the contents of /universe/faculties.md.
func GenerateFaculties(verifiedTools []string, gateBridges []gate.Bridge) string {
	var sb strings.Builder

	sb.WriteString("# Faculties\n\n")

	// Tools
	sb.WriteString("## Tools\n")
	if len(verifiedTools) > 0 {
		sb.WriteString(strings.Join(verifiedTools, ", "))
		sb.WriteString("\n")
	} else {
		sb.WriteString("(none verified)\n")
	}

	// Gate Bridges
	if len(gateBridges) > 0 {
		sb.WriteString("\n## Gate Bridges\n")
		for _, gb := range gateBridges {
			caps := ""
			if len(gb.Capabilities) > 0 {
				caps = " [" + strings.Join(gb.Capabilities, ", ") + "]"
			}
			sb.WriteString(fmt.Sprintf("- `%s` — %s%s\n", gb.As, gb.Source, caps))
		}
	}

	return sb.String()
}
