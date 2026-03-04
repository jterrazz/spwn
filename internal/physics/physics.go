package physics

import (
	"fmt"
	"strings"

	"github.com/jterrazz/universe/internal/config"
)

// GeneratePhysics returns the contents of /universe/physics.md.
func GeneratePhysics(m config.UniverseManifest) string {
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
	switch m.Physics.Laws.Network {
	case "none":
		sb.WriteString("- No outbound network access\n")
	case "bridge":
		sb.WriteString("- Network: bridge mode (filtered outbound access)\n")
	case "host":
		sb.WriteString("- Network: host mode (full network access)\n")
	}
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
func GenerateFaculties(verifiedElements []string, gateBridges []config.GateBridge) string {
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
