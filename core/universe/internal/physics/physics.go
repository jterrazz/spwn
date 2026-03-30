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
	sb.WriteString(fmt.Sprintf("- Maximum process count: %d\n", m.Physics.Laws.MaxProcesses))
	sb.WriteString("- Filesystem is ephemeral except /workspace and /mind\n\n")

	// Elements
	sb.WriteString("## Elements\n")
	sb.WriteString("/workspace — project files, mounted from Host (read-write)\n")
	sb.WriteString("/mind — agent identity and memory (read-write)\n")
	sb.WriteString("/tmp — ephemeral scratch space\n\n")

	// Topology
	sb.WriteString("## Topology\n")
	sb.WriteString("/workspace — project files, mounted from Host (read-write)\n")
	sb.WriteString("/mind — agent identity and memory (read-write)\n")
	sb.WriteString("/tmp — ephemeral scratch space\n")

	return sb.String()
}

// GenerateFaculties returns the contents of /universe/faculties.md.
func GenerateFaculties(verifiedElements []string, gateBridges []gate.Bridge) string {
	var sb strings.Builder

	sb.WriteString("# Faculties\n\n")

	// Elements
	sb.WriteString("## Elements\n")
	if len(verifiedElements) > 0 {
		sb.WriteString(strings.Join(verifiedElements, ", "))
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
